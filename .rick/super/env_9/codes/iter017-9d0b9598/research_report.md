# 调研报告 · 迭代 17 · dispatch_target=np_dot|impl_strategy=full_blas_dispatch

> 预算语境：本迭代为预算末轮（17/17），本格 visit=2。上一轮（iter007，本格 visit=1）设计正确、预检全绿，但三段 agent 全部死于 turn-budget（verdict=no_candidate，**失败在过程非性能**——archive description 明文记载）。故本报告以「已验证事实整合 + 流程修复」为主线：profile 事实直接取 ctx 注入的本迭代（iter17，ts 21:04:38）剖析，不重复跑；检索按 research_brief 收敛（3 来源封顶）。

## 1. 现有代码调研（父本与相邻格精英的结构、瓶颈、可复用点）

- **父本 `env/code/naive_start.py`（rowwise np.dot，无转置）**——ctx 注入的本迭代 profile @8192：runtime **11192.5ms**；热点 `<listcomp>` self **10400.1ms（99.8%）**、`stack` 10.1ms（0.1%）、`dot` 仅 1.23ms——瓶颈全在 Python 层 8192 次逐行循环 + 每行 gemv 全量重读 B（~768MB 工作集反复过内存墙）+ stack 装配；BLAS 内核本身只值 ~440ms。env_facts：Python 3.9.6 / numpy 2.0.2 / accelerate BLAS / 18 核 / memcpy 28GB/s。
- **邻格精英 iter003（np_matmul|full_blas_dispatch，430.83ms 全场纪录）**：kernel 即单次整阵 `np.matmul(a32, b32)`，profile 中 `matmul` self 100%（440-454ms 稳态）。其契约结构（`matmul(a,b)` 接口 / asarray→f32 / seterr 三项抑制 / main 的 warmup+计时+JSON 协议）是本格候选的直接模板（research_report §4 第 2 条）。
- **邻格精英 iter010（np_dot|m_halves，462.82ms）**：关键预检事实——**2-D float32 下 np.dot 与 np.matmul 映射同一 cblas_sgemm，@8192 `array_equal=True`、maxdiff=0.0（逐位一致）**；同 session 交错复测 m_halves_np_dot 中位 449.6-466.5ms vs np.matmul 480.5ms。其 `_self_check` 用例表（方阵/奇数行/非方阵/M=1/M=0/嵌套 list）可整体移植。
- **同格 iter007（visit=1，verdict=no_candidate）**：`codes/iter007-be5b45a7/solution.py` **已写好本格定义形态的一行内核**（`return np.dot(a32, b32)`，无 out=/无转置/无 K 切分），docstring 完整、契约与 iter003 一致；实验记录 §6 为空（coding/debug 未及回填）。预检数据（其记录 §1）：np.dot 中位 **437.4ms** vs np.matmul 433.1ms（同进程交错，±1% 噪声带内）。
- **可复用点结论**：候选代码本体已存在且从未过门——本轮工作重心是**流程修复**（复制/微调 → 每工具一次走完 verify→benchmark→gate），不是重新发明。

## 2. 业界落地方法（同类思路实际怎么做：机制 / 参数 / 工程细节）

- **numpy#18669（GitHub issue，本次检索命中）**：`matmul` 带 `out=` 在未 fault-in 的输出页上显著慢于 `np.dot`，预故障后反转。对本格的工程含义：**plain 调用（不带 out=）两入口都在入口处新分配输出，入口差异在噪声带内**——iter007 预检 437.4 vs 433.1ms 印证。故本实验选 plain `np.dot(a32,b32)`（单变量、与 iter003 模板同构），out= 变体留给后续迭代。
- **Stack Overflow 52062496 / 77718293（本次检索命中）**：2-D 输入下 `a.dot(b)` / `a@b` / `np.matmul` 语义等价、大矩阵下都坍缩到 cblas_sgemm；差异只在入口检查与包装开销，规模够大（n≥512）时被内核时间淹没——@8192（1.1 TFLOP）正是这种规模。
- **Ash Vardanian《NumPy vs BLAS: Losing 90% of Throughput》（本次检索命中）**：量化 NumPy 入口包装成本，薄入口（dot 类 2-D 路径）接近裸 cblas 调用——支撑「单次整阵直派 ≈ 裸 sgemm」预期，也与 rowwise（8192 次入口 + 全量重读）vs 单次整阵（1 次入口）相差 25× 的结构解释互洽。
- **测量方法（Intel MKL/BLIS GEMM benchmarking 指南 + GEMMbench 框架，本次检索命中）**：warm-up 丢弃、重复计时取中位、round-robin 交错 A/B——与 env benchmark 口径（k=3 取中位）及 iter007 同进程交错预检一致。

## 3. 论文的实验落地细节（实验怎么配、规模与硬件差异、可复现要点）

- **arXiv:2411.00442《Revealing Floating-Point Accumulation Orders in Software/Hardware Implementations》**：累积型运算（summation/matmul）的**累积顺序普遍未文档化且直接决定逐位结果**；跨实现一致性只能靠「同一累积路径」保证，不能靠算法等价。落地到本任务：正确性 ≡ 与 f32 BLAS 锚同 K 求和序（逐位）——**单次整阵 np.dot 与 np.matmul 同走 cblas_sgemm 即满足**；而一切改 K 序结构（blocked-K/象限和/Strassen/预转置 gemv）都在死族清单里，与该论文结论互证。实验可复现要点：固定 seed、固定规模、双跑逐位比对。
- **GEMMbench（Collective Knowledge 框架，可复现 GEMM 评测）**：指出 GEMM 数十年无统一评测方法学，给出「warm-up + 重复计时 + 多实现交错对比 + 环境快照」协议。落地：本 env 的 k=3 中位 + ref 带校验（380-700ms）+ env_facts 快照（numpy/BLAS/核数/memcpy GB/s）正是该协议实例；本实验照此口径报告，不发明新指标。
- **规模与硬件差异**：小矩阵域入口开销占比大（SO n=500 例子可见入口差 ~2×），@8192 把入口成本压到 <1%——这也是「每工具至多跑一次、中位口径」在末轮预算下仍可信的量纲依据。

## 4. 迁移结论（本实验具体要借哪一条，落到 plan_steps 的哪一步）

1. **内核形态钉死**（2411.00442 累积序结论 + iter010/iter007 逐位预检）：`return np.dot(a32, b32)`——不用 out=（18669 只影响 matmul+out= 场景，plain 调用更稳且保单变量）、不预转置、不切 K/N。→ plan_steps 第 2 步。
2. **候选直接以 iter007 的 solution.py 为父本**（同格上轮已写好的同一行内核，从未过门）：复制到本迭代目录，移植 iter010 `_self_check` 用例表，契约照 iter003 模板。→ plan_steps 第 1 步。
3. **测量口径照 GEMMbench/benchmark.py**：--n 64 快速绿 → verify 一次 → benchmark 一次（k=3 中位）→（可选）gate_check 对照邻格精英 430.83ms 的 ±5% 软带。→ plan_steps 第 3-5 步。
4. **过程纪律是本轮真正的实验变量**（archive description 明文）：每个工具至多跑一次、读数即记录、不复跑门禁——修复 iter007 no_candidate 的根因。→ 贯穿全部 plan_steps。
