# 调研报告 · 迭代 10 · dispatch_target=np_dot|impl_strategy=m_halves

## 1. 现有代码调研（父本与相邻格精英的结构、瓶颈、可复用点）

**父本（起点）`env/code/naive_start.py`（rowwise np.dot，@8192 ≈ 11575ms）**
- profile（iter10 注入，@8192）：`<listcomp>` self 11510ms 占 **99.9%**，`stack` 9.5ms，`dot` 仅 1.1ms——瓶颈完全在 Python 逐行循环（8192 次 1-D gemv，B 全量重读 8192 遍）+ np.stack 装配，BLAS 内核本身只值 ~440ms。
- 契约（必须原样保持）：`matmul(a,b)` 接受嵌套 list 或 ndarray → asarray f32；`np.seterr(divide/over/invalid="ignore")` 三项抑制；`main()` 的 `--n` 自检协议（预热 1 次 + 计时 + `max_error_vs_matmul` JSON 输出）。

**相邻格精英 `codes/iter004-e77e66ca/solution.py`（np_matmul|m_halves，484.43ms，admit）——本实验父本**
- 结构：`mid = m//2`（奇数行安全）+ `c = np.empty((m,n), f32)` 预分配 + 两个行切片 `out=` 直写：`np.matmul(a32[:mid], b32, out=c[:mid])` / `np.matmul(a32[mid:], b32, out=c[mid:])`——规避 concatenate 的 256MB 整阵拷贝。
- 自带 `_self_check()`：方阵/奇数行/非方阵/单行/零行/嵌套 list 全部断言逐位一致——可直接复用为测试模板（TDD 红绿）。
- 同 session 复测（本次调研实测，benchmark.py 协议）：**480.5ms，score 0.9455**（ref 454.3ms）——m_halves + matmul+out= 形态稳定带 ~480ms。

**本 session 关键预检（新事实，决定本实验可行性）**
- @8192 同进程交错计时（每形态 3 轮×3 次取中位）：`m_halves_np_dot` 中位 **454.4ms** ≈ `m_halves_matmul` 454.3 ≈ `full_matmul` 453.7 ≈ `full_dot` 480.7（均在 ±6% 噪声带）。
- 复刻 benchmark.py 协议（candidate 先、ref 后，同进程）跑 3 次：m_halves_dot 中位 **449.6 / 466.5 / 459.8ms**，ref 434.3 / 446.8 / 475.5ms，score **0.966 / 0.958 / 1.034**。
- **逐位一致性**：`np.dot(a32[:mid], b32, out=c[:mid])` 两半拼装 vs `np.matmul(a32,b32)` 锚——`array_equal=True, maxdiff=0.0`（M 分解不动 K 求和序）。
- 结论：iter007 遗留的「np.dot 578ms」读数是负载噪声，非入口开销；np_dot×m_halves 形态落在 BLAS 平价带内，且**比父本 iter004 的同 session 读数（480.5）低 ~20-30ms**。

## 2. 业界落地方法（同类思路实际怎么做：机制 / 参数 / 工程细节）

- **numpy GitHub issue #18669「np.matmul with `out` parameter slower than np.dot in some cases」**：2-D 输入下 matmul 与 dot 语义相同，但 `matmul` 的 `out=` 路径在输出页未 fault-in 时显著慢于 `np.dot`；页预 fault 后差异反转。工程含义：**同为 out= 直写，np.dot 入口比 np.matmul 入口更稳**——本格（np_dot 派发）恰好处在有利一侧，预检的 449.6 vs 480.5（同 session、同协议）与此吻合。
- **numpy GitHub issue #23588**：matmul 对可 gemm 化的形状/连续性检查更挑剔，不满足时落到无 cache 意识的 noblas 路径（~100× 慢）；`dot` 的派发更宽容。工程含义：行切片 `c[:mid]`（C 连续视图）两侧都合法，但 dot 入口风险更低。
- 2-D f32 下 `np.dot` 与 `np.matmul` 最终映射同一 `cblas_sgemm` 调用（iter007 调研引 1911.09421 + 本 session 预检 maxdiff 0.0 佐证）——**换入口不换内核、不改 K 求和序**，这是逐位一致的理论根据。

## 3. 论文的实验落地细节（实验怎么配、规模与硬件差异、可复现要点）

- **GEMMbench（arXiv:1511.03742）**：可复现 GEMM 评测框架——固定 seed 生成输入、warmup 后多次计时取中位、同进程/同机对照消负载噪声。与本 env benchmark.py 协议（seed 42、warmup 1 + k=3 中位、ref 同进程）同构；本报告 §1 的预检即按该口径执行。
- **Revealing Floating-Point Accumulation Orders（arXiv:2411.00442）**：累加型运算（含 GEMM）的浮点累加序是未文档化的实现细节，改变调用路径/分块方式即改变结果；逐位复现要求保持同一底层调用形态。落地含义：**每半必须保持单次 2-D np.dot 整块派发**——任何切 K/N、转置、象限拼接都会改变累加序而撞 1e-3 容差墙（与 domain 死族清单互证：blocked-K 1.4-1.6e-2、B 预转置 1.39e-2 均超 14×）。
- 规模差异要点（本 env 实测）：@8192 工作集 ~768MB 落内存墙区间，全合法族坍缩 434-488ms 窄带（±6%）；@2048 误差 1.6-2.9e-3、@8192 放大到 1.4e-2+——死族在 8192 死得更狠，正确性判据必须以 @8192 逐位一致为准。

## 4. 迁移结论（本实验具体要借哪一条，落到 plan_steps 的哪一步）

1. **借 iter004 精英的 m_halves 骨架与全部契约**（mid=m//2 奇数行安全 / C 预分配 + 行切片 out= 直写 / seterr / main JSON 协议 / `_self_check` 用例表）→ **plan_steps 第 1-2 步**：以此为父本，保证「单变量改动」的可归因性。
2. **借 numpy#18669 的入口不对称事实**：把两个半块的 `np.matmul(..., out=...)` 换成 `np.dot(..., out=...)`（2-D 整块派发），预期避开 matmul+out= 的慢路径 → **plan_steps 第 2 步（唯一 kernel 改动）**。
3. **借 2411.00442 的累加序结论 + 本 session 预检（maxdiff=0.0）**：不切 K/N、不转置、每半单次整块 dot → **plan_steps 第 3-4 步**（自检 + verify 的逐位判据）。
4. **借 GEMMbench 口径**：expected_behavior 与 benchmark.py 同尺度（@8192、k=3 中位、score=ref/cand），预检读数 449.6-466.5ms / score 0.96-1.03 作为预期带 → **plan_steps 第 5 步 + 记录第 5 段**。
5. **过程纪律（本 env 特有教训）**：iter006/007/008/009 连续 4 轮 no_candidate 均因 coding/debug 阶段 turn-budget-exceeded（过度复跑门禁、追噪声）——plan_steps 明令每工具跑一次即记录，不再复跑。
