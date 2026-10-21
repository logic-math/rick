# ab_notes.md — 迭代 19 A/B 归因（einsum canonical-TT + concat-T 装配 vs 父本 swapped + 趁热拷贝）

- 候选：`solution.py` = canonical 拼写 `np.einsum('ij,jk->ik', a32[:h]/a32[h:], b32, optimize=True)`
  （TT 标志 gemm，输出 (h,N) F 连续视图）+ 装配 `np.concatenate((r1.T, r2.T), axis=1).T`
  （.T 零拷贝基视图沿 axis=1 拼成 (N,M) C 连续，返回 .T 视图即 (M,N) F 连续数组）
- 父本对照：`codes/iter013-0a966897/solution.py`（swapped 拼写 C-标志 gemm + 预分配逐块趁热
  memcpy，门中位 7.83875ms，score 0.857）——逐字节读取核对，本候选唯一改动即 matmul 内核
  （拼写 + 装配联动），模块级 seterr / main() 自检骨架原样保留
- 环境与预研：env_8 venv numpy 2.0.2，macOS Accelerate/vecLib，@2048×2048 f32；
  全部预研依据见 `pre_research.md`（全设计空间分解）与 `pre_bench6.py`（W8 终验）

## 正确性（全部通过，与预研一致）

- 与 `pre_bench6.py` 的 `w8_candidate` 逐位一致（np.array_equal=True）：64/2048 × 双 seed
  （42 + 914037288）全部 True
- 与 np.matmul 逐位一致：同上全部 True；max_error=0.0
- 自检门（self_check.py --candidate solution.py）：**passed=true，max_error=0.0**，
  sizes [64, 2048] × 双 seed（42 + urandom 派生），tolerance 0.001 → 通过
- 直接运行自检：@64 0.037ms / max_error=0.0；@2048 8.381ms / max_error=0.0
- 输出为 F 连续数组（值语义合法，契约不要求 C 连续）

## 热点剖析（perf_profile.py @2048）

| 项 | 候选 | 父本 |
|---|---|---|
| tensordot（BLAS 派发） | 7.554ms / 90.2%（2 calls） | 7.693ms / 94.9%（2 calls） |
| c_einsum | 0.006ms / 0.1%（无操作级噪声，**非** 0.41ms/8MB 病态恒等拷贝） | 不在 top-8 |
| 单次端到端 | 8.402ms | 8.246ms |

tensordot 主导、BLAS 派发路径未变（与父本同链 einsum→tensordot→gemm）；c_einsum 6μs
为输出规整的尾噪声，预研确认的死路形态（out= 触发 8MB 恒等拷贝）未出现。无异常。

## 端到端耗时与父本对比

| 测量 | 候选（W8） | 父本（V0/门中位） | Δ |
|---|---|---|---|
| 自检单次（K=1） | 8.465ms（ref 6.951，score 0.821） | ~8.25ms（同日 perf_profile 8.246） | +0.16~+0.22 |
| 进程内新鲜配对 ×3 轮（本日 load≈5.0，每轮 11 次中位） | 8.270 / 8.198 / 8.205 | 8.087~8.137（轮均值） | **+0.107 / +0.133 / +0.117（劣化一致）** |
| 预研 6 轮新鲜配对（pre_bench6，builder 日） | — | — | +0.14~+0.22 劣化一致 |
| 父本门中位（K=3，终验口径） | 待框架 gates.py 裁决 | **7.83875** | 预期 ≥ 7.83875 |

## 对照 expected_behavior（如实核对）

| 预期 | 实际 | 判定 |
|---|---|---|
| 正确性全程不变（max_error=0.0） | 0.0（双规模双 seed、嵌套 list、奇数形状、双跑逐位一致） | ✓ |
| gemm 段 −0.15~−0.25ms | tensordot 7.554 vs 父本 7.693（当日 −0.14，同量级） | ✓（T/C 漂移态） |
| 装配段 +0.4~0.5ms（F 序 4KB 段拷贝税） | 端到端净差 +0.11~+0.13（今日），覆盖后仍劣化 | ✓ |
| 端到端 +0.14~+0.22 劣化 | +0.107~+0.133（今日负载略压缩幅度，方向一致） | ✓ |
| 门中位大概率 ≥ 7.83875 → 证伪 | 待框架 K=3 中位裁决（自检/配对均指向劣化） | 待裁决 |
| 翻盘条件：门日 T/C 差 > +0.47ms（~3%） | 今日 gemm 级 T/C 差 −0.14，远低于翻盘阈值 | 未见 |

## 结论（门禁裁决前的实现侧归档）

实现侧全部证据（自检、逐位核验、热点剖析、当日新鲜配对 ×3 + 预研 ×6）一致指向：
**canonical-TT 的 gemm 级 T/C 漂移收益（当日 −0.14ms，量级与历史 ±0.25 摆动带一致）
不足以覆盖 F 序装配税（+0.47ms，负载无关）**，端到端净劣化 +0.11~+0.22。

- 若框架门中位 ≥ 7.83875（预期，~97%）：**拼写轴闭合**——按 pre_research.md 第四节归档
  「T/C 漂移收益上限不敌 F 序装配税；格地板确认」：父本（swapped + 趁热 memcpy）为
  einsum_path|m_halves 全构造空间最优点，格地板 ≈ 7.75-7.95（gemm 7.46 + 装配 0.24 +
  交互 0.19，全部结构项），scheduler 可降权本格。
- 若门中位 < 7.83875（尾部机会，~3%）：门日 T/C 差突破 +0.47（历史幅度 2.2 倍，未见
  先例）→ 新精英 + 装配范式改写为 concat-T。

自检 runtime 8.465ms 慢于父本 ~0.2ms 是实验预期而非实现错误（单变量对照即为此度量）；
门禁（K=3 中位、独立负载态）是唯一裁决。
