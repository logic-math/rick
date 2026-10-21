# ab_notes.md — 迭代 6 A/B 归因（np.matmul rowwise vectorized vs 父本整阵 np.matmul）

- 候选：`solution.py` = 预分配 out + 逐行 `np.matmul(a32[i], b32)`（1-D 行 × 2-D B，不预转置）
- 父本：iter003-93ddaebc = `np.matmul(a32, b32)`（整阵单发，记录基线 6.859ms）
- A/B 脚本：`ab_test.py`（进程内配对、每轮随机次序 10 对、每对 K=3 取中位；不改变提交实现）
- 变体（2×2 因子格 + 骨架对照）：
  - `candidate`：本迭代（np_matmul|rowwise_vectorized）
  - `naive_dot`：naive_start.py 原生形态（np.stack + 列表推导逐行 np.dot）
  - `naive_dot_pre`：骨架对照（预分配 out + 逐行 np.dot——与 candidate 仅差派发调用，隔离纯入口税）
  - `parent`：iter003（np_matmul|full_blas_dispatch）
  - `einsum_rowwise`：iter004（einsum_path|rowwise_vectorized，包装税对照）
- 环境：env_8 venv numpy 2.0.2，Apple M5 Max / macOS Accelerate(vecLib)，@2048×2048 f32，
  本机当日持续负载态（load 1m 4.26-4.91，与历次迭代同时段同型负载）

## 正确性（自检 + sanity 全过）

- A/B sanity（@2048）：candidate / naive_dot / naive_dot_pre / parent / einsum_rowwise
  全部与 `np.matmul` 锚**逐位一致**（True，3 轮均验）——1-D×2-D 不转置派发与整阵 gemm
  同 K 求和序，构造性成立。
- `solution.py --n 64`：max_error = **7.629e-06**（vecLib 小规模路径差异，与 naive rowwise
  np.dot 同级，远低于 1e-3 容差）；`--n 2048`：max_error = **0.0**（双跑均 0.0）。
- 嵌套 list 输入路径：@2048 逐位一致、@64 同级小误差（~7.6e-06）；输出 C-布局 float32，
  形状 (n, m)；双跑逐位一致（64/2048 双规模实测 True）。

## 自检单发（solution.py 直跑）

| n | runtime_ms | max_error |
|---|---|---|
| 64 | 0.056 | 7.629e-06 |
| 2048 | 143.045 / 147.336（两跑） | 0.0 |

## A/B 结果（3 轮独立进程，逐轮内配对随机次序 10 对 × K=3 中位）

| 轮 | 时刻 | load(1m) | candidate | naive_dot | naive_dot_pre | parent | einsum_rowwise |
|---|---|---|---|---|---|---|---|
| 1 | 20:56 | 4.26 | 158.458 | 157.497 | 157.908 | 6.729 | 181.287 |
| 2 | 20:56 | 4.64 | 145.010 | 143.054 | 144.609 | 7.140 | 168.201 |
| 3 | 20:57 | 4.91 | 142.639 | 141.001 | 141.857 | 6.825 | 166.379 |

归因量（逐对配对差均值，ms）：

| 量 | 轮 1 | 轮 2 | 轮 3 | 预期 | 判读 |
|---|---|---|---|---|---|
| rowwise 结构税（candidate−parent） | +151.5 | +138.6 | +136.0 | ≈ +135 | ✓（轮 1 负载相位抬升，带内） |
| matmul 入口税（candidate−naive_dot_pre，纯单变量对照） | +0.40 | +0.82 | +1.11 | 0±2 | ✓ 不可见 |
| matmul 入口税（candidate−naive_dot 原生形态） | +0.74 | +2.85 | +1.89 | 0±2 | ✓ 带内（骨架噪声叠加） |
| einsum 包装税（einsum_rowwise−candidate） | +23.2 | +22.7 | +23.4 | ≈ +23.5 | ✓ 高度复现 |
| 骨架税（naive_dot−naive_dot_pre，np.stack vs 预分配） | −0.34 | −2.03 | −0.78 | ~0 | 骨架等价（≤2ms 噪声） |

- **可证伪点检验**：candidate−naive 全部 < +5ms（最大逐轮均值 +2.85ms，纯对照
  naive_dot_pre 最大 +1.11ms）→ 「matmul gufunc 每行派发税 @2048 不可见」主张**成立**
  （可见上限 ~1ms 量级，处于负载态噪声带；非 einsum 的 23ms 级）。
- 入口税方向三轮一致微正（+0.4~+1.1ms vs 纯对照）——与预研 @64 微基准
  （np.matmul 0.73μs vs np.dot 0.62μs/行，折 @2048 ~0.2ms）量级相符，
  未触发放弃线。

## 终验口径复跑（bench.py 门协议，K=3，候选先计时）×3 轮

见下表（与 self_check 同协议）：

| 轮 | 时刻 | load(1m) | runtime_ms | score | ref_runtime_ms | ref∈[5,10] |
|---|---|---|---|---|---|---|
| 1 | 20:57 | 4.01 | 142.255 | 0.0493 | 7.009 | ✓ |
| 2 | 20:57 | 4.01 | 144.309 | 0.0494 | 7.128 | ✓ |
| 3 | 20:58 | 4.01 | 146.528 | 0.0463 | 6.785 | ✓ |

- 3 轮 runtime 142.3-146.5ms（预期带 135-170ms ✓）；score 0.046-0.049
  （预期 0.042-0.052 ✓）；ref_runtime 6.79-7.13ms ∈ [5,10] 合理带 ✓。
- 与父本 6.859ms 相比慢 ~21×——rowwise 内存墙结构性下界（每输出行完整流过 B，
  2048×16MB=32GB 流量），本格目标不是竞逐全局最优，而是建立
  dispatch_target=np_matmul × impl_strategy=rowwise_vectorized 的首条可归因实测。

## 归因（如实记录）

1. **正确性主张成立**：1-D×2-D 不转置的 np.matmul 派发与整阵 gemm 同 K 求和序
   → @2048 逐位一致（max_error=0.0，双 seed、嵌套 list、双跑均验）；
   @64 ~7.6e-06 与 naive rowwise np.dot 同级（vecLib 小规模路径差异）。
2. **「gufunc 派发税不可见」主张成立**：纯单变量对照（同骨架仅差
   np.matmul↔np.dot 派发调用）配对差 +0.4~+1.1ms，全部处于 0±2ms 带内、
   远低于 +5ms 放弃线；对比 einsum 同结构 +22.7~+23.4ms 包装税——
   dispatch_target 维度在 rowwise 下的差异比 impl_strategy 包装层小一个量级以上。
3. **rowwise 结构税实测 +136~151ms**（负载相位敏感，干净相位 ~136ms）：
   与内存墙量级模型（~137ms 下界）一致；candidate 绝对 runtime 142.6-158.5ms
   （A/B）与 142.3-146.5ms（门协议）均在预期带。
4. **对谱系的意义**：rowwise 格族内 np_matmul 与 np_dot 入口证实等价
   （差 ≤1-3ms 负载噪声内），einsum 入口被证实独带 ~23ms 包装税——
   impl_strategy=rowwise_vectorized 下 dispatch_target 的剩余可榨空间
   不在 np_matmul/np_dot 之间；rowwise 家族的绝对水平由内存墙决定，
   结构性慢于整阵族 ~20×，不存在接近 7ms 族的路径（除非改结构本身）。
5. **机器状态记录**：实验全程负载态（load 4.01-4.91），轮 1 A/B 绝对值整体
  抬升 ~13ms（同轮 parent 6.73ms 亦同步波动）；所有轮次数字未做剔除地列于上表。
