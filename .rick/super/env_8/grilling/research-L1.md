# env_8 research-L1 — 2048×2048 f32 matmul 基线族实测 + 容差 + 门禁成本 + 外部校准 — 2026-10-21

## §0 方法与信源分级

- 环境：env_7 venv（py3.9.6 + numpy 2.0.2，BLAS=accelerate 实证 `np.show_config`），Apple M5 Max 64GB（6P+12E，sysctl 实证）。脚本 /tmp/mm8/*.py，串行执行，每次测量=预热1+计时K(5-20)取中位数。**[A]**
- 机器背景噪声：load ~4.2（winedevice×2 ~260% CPU + Chrome），未杀用户进程；跨进程 ref 中位数散布 6.7–8.0ms（±10%），进程内 IQR 仅 0.06ms。**[A]**
- 分级：A=本机实测（本轮 /tmp/mm8 全部可复现）；B=外部文档/基准；C=推断外推。

## §1 基线族表（@2048 f32，ref=np.matmul 中位 6.88ms；score预测=6.88/候选中位）

| # | 实现 | 中位 ms (min–max) | score 预测 | 容差@2048(uniform,1e-3) | 级 |
|---|---|---|---|---|---|
| 1 | np.matmul f32（K=20：med 6.88, IQR 0.06, min 6.75, max 7.01；跨进程 med 6.7–8.0） | 6.88 | 1.0（协议内实测 0.98） | — 锚 | A |
| 1b | np.matmul f64 | 27.2 | 0.25 | **不过**（vs f32 锚 1.73e-3） | A |
| 2 | rowwise np.dot 无转置 np.dot(a[i], b) | 142.8 | 0.048 | 过（逐位一致 0.0） | A |
| 2b | rowwise B 预转置 np.dot(bt, a[i]) | 188.2 | 0.037 | **不过**（1.71e-3） | A |
| 3 | blocked K 块 128 | 24.7 | 0.28 | **不过**（1.77e-3） | A |
| 3b | blocked 256 | 12.0 | 0.57 | **不过**（1.65e-3，实门复现拒） | A |
| 3c | blocked 512 | 10.6 | 0.65 | **不过**（1.71e-3） | A |
| 5 | Strassen-1（7×1024 gemm+18 加） | 11.3 | 0.61 | **不过**（2.87e-3） | A |
| 5b | quad_sum（2×2 象限 8×1024 gemm） | 10.2 | 0.67 | **不过**（1.77e-3） | A |
| 6 | mp fork 2/3/4/6 进程（端到端含 fork+join） | 10.6 / 11.3 / 12.7 / 15.1 | ≤0.65 | （n/a） | A |
| 6b | fork-only 6 空进程（纯开销） | 6.8 | — | fork 开销≈整个 matmul | A |
| 7 | threaded 2/3/4/6 线程行分片 | 7.5 / 7.5 / 8.0 / 8.4 | ≤0.92 | （n/a） | A |
| 5c | Strassen-1 + 7 线程并行叶 | 11.0 | 0.63 | 不过 | A |
| 8 | np.einsum 默认 | 502.3 | 0.014 | 未测（无意义） | A |
| 8b | einsum optimize=True | 7.9 | 0.87 | 过（0.0 逐位一致） | A |
| 9 | ctypes cblas_sgemm RowMajor 直调 | 10.0 | 0.69 | 过（同 BLAS） | A |
| — | row_halves（M 对半 2 次 gemm） | 7.6 | 0.90 | 过（0.0 逐位一致） | A |
| 4 | 纯 Python naive（list 输入）512/1024 实测 4.14s / 34.2s | 2048 外推 ≈**274s**（numpy 输入 512=8.44s→外推 **550s**） | ~0.0000 | 超时必死 | A/C |

补充实测：T(512)=0.185ms（cpu_ratio 1.0，与 env_7 0.187 吻合）；T(1024)=0.867ms（ratio 2.8）；VECLIB_MAXIMUM_THREADS 扫描（进程外设）：1线程=10.25ms，3=6.71（最优），4/6/12/18≈6.8–7.2 持平，cpu_ratio 恒 ~2.7——**单调用约 3 线程即饱和，6P 核吃不满也无增益**；进程内后设 env var 无效（库 init 时读取，候选模块无法改线程数）。matmul out= 预分配 / np.dot ≈ 无差别（<2%）。**[A]**

冷启动（item10）：新进程首次 2048 matmul 8.7–10.0ms vs 稳态 6.9ms（1.24–1.44×，5 次新进程）；512 时代 7× 冷惩罚在 2048 被计算量稀释，且 bench 有 warmup1，不影响计时。**[A]**

## §2 容差余量（@2048，max_error vs np.matmul f32 锚，双方转 fp64 相减；输入=门协议 uniform rand seed42 + 1 urandom + 任务指定 randn seed42）

关键事实（uniform=门协议实际用 `RandomState(seed).rand`）：**[A]**

| 对比 | uniform seed42 | uniform urandom | randn seed42 |
|---|---|---|---|
| 锚 vs fp64 真值（锚自身噪声） | **1.734e-3** | 1.714e-3 | 4.82e-4 |
| rowwise 无转置 / row_halves / einsum_opt | **0.0**（逐位一致） | 0.0 | 0.0 |
| rowwise B 预转置 | 1.71e-3 | 1.77e-3 | 4.88e-4 |
| blocked 128 / 256 / 512 | 1.77 / 1.65 / 1.71e-3 | 1.71 / 1.71 / 1.83e-3 | ~5.0e-4 |
| Strassen-1（修正版） | **2.87e-3** | 2.90e-3 | 6.79e-4 |
| quad_sum | 1.77e-3 | 1.71e-3 | 5.19e-4 |

结论（item 11/12）：**[A]**
1. @2048 uniform 输入下 C 元素量级 ~561（mean 512），f32 锚自身离 fp64 真值 1.73e-3 **已超 1e-3 容差**——"正确"只能定义为"与 f32 BLAS 锚一致"，f64 matmul 候选也会被拒（对称误差）。
2. 1e-3 容差 @2048 是**紧约束**：任何改变 K 求和序的实现（K 分块/象限和/Strassen/转置后 gemv）误差 1.6–2.9e-3 全被拒；**只有保持 BLAS 求和序（沿 M 维分解：rowwise 无转置、row_halves、einsum_opt）逐位一致 0.0 能过**。实测 env_7 门脚本复现：blocked256 @2048 被拒（max_error 1.65e-3）。
3. randn 输入下全员 ≤6.8e-4 全过——紧度由输入分布决定，门协议用 uniform，故以 uniform 为准。
4. 环境公式参考：max_error ≈ ε·|C|max·k，ε=1.19e-7，|C|max≈561，k≈20–30（BLAS 序）；512 时 |C|max≈128 → ~4e-4，容差舒适；2048 临界。

## §3 门禁成本（用 env_7 真门脚本实测）**[A]**

- correctness 门 sizes=[64,2048]（2 seeds + double-run）：rowwise 0.73+0.06≈**0.79s**；BLAS 0.16+0.06≈0.22s；blocked(256)（若放容差）~0.27s——全部 << 30s alarm。纯 Python naive：单次 2048 ≈274–550s → **首次 run 即 30s alarm 超时死**（item 13 实测列表见 §1#4）。
- bench 全协议 @2048（warmup1+K3 候选+K3 ref）：BLAS 0.15s / rowwise 0.70s / blocked 0.16s 端到端（含 py+numpy 启动）——<< 60s alarm（item 14）。
- **env_7 参数在 2048 全部失配（KR2 硬证据）**：ref_runtime_band_ms=[0.10,0.50] → @2048 ref 6.87–7.07ms，**所有候选（含 BLAS 平价）一律被拒**（实测 3 例全拒，stage=plausibility）；bench size/correctness sizes 需 512→2048；score_max=2.0 仍安全（BLAS 平价协议内实测 score=0.979，候选先计时有 -2~-3% 顺序效应）。
- 建议新 ref band：**[5.0, 10.0]ms**（中位 6.88，跨进程散布 6.7–8.0，冷首调 8.7–10.0，±35% 宽容带）。

## §4 外部校准（B 级）

15. **Apple Silicon 大矩阵 sgemm 多簇饱和**：AppleAccelerate.jl 官方基准（M2 Max/macOS 26，2026-07 复测）明确："Accelerate offloads large GEMM to a shared SME co-processor that **a single thread already saturates**"——SME 协处理器串行吐出吞吐，多线程无增益。与本机实测一致（cpu_ratio 2.7 恒定、VECLIB≥3 无增益、threaded/mp 全劣化）。Apple 开发者论坛（thread/666635）：Accelerate BLAS 线程数不可控（仅 VECLIB_MAXIMUM_THREADS 全局环境变量）；threadpoolctl#136（ogrisel）：vecLib GEMM 在并发调用下有 GCD 防超订机制——解释本机 threaded 劣化。**结论：numpy/CPython 栈内无已知 >1.0 路径（M2–M4 世代证据 + 本机 M5 Max 实测）。[B+A]**
16. **Strassen×BLAS 叶交叉规模**：文献（Bailey/Lee/Simon 1991；BLIS Strassen 论文；Huang et al. adaptive Strassen）：交叉点强烈依赖机器，经典结论 n>1000–2000 才可能占优，且需叶 ≥512 的 BLAS 叶 + BLAS 自身非 Strassen。本机 A 级实测：Strassen-1@2048（叶 1024 BLAS）= 11.3ms vs 单调用 6.88ms（-64%），因 Accelerate 单调用已近 SME 上限，7/8 FLOPs 收益 < 调用效率损失+加法开销；叠加容差 2.87e-3 被拒 → **Strassen 在 2048 本机双杀（性能+数值）。[B+A]**
17. **手工 cache-blocking vs BLAS 内建 tiling**：共识（scicomp.SE、Goto & van de Geijn 2008）：商用 BLAS GEMM 用 Goto 三级 cache blocking + 手写汇编 kernel，手工 tiling 最好到 BLAS 的 1.5–2× 慢，从未反超；Apple Silicon 上实测（lkforge.com，n=1024，同 cc -O2）：循环序优化 ikj 6.3×、tiling 增益有限、BLAS 再快一个量级（naive→BLAS 差 155×）。本机实测 blocked 128/256/512 = 24.7/12.0/10.6ms 全劣于单调用 6.88ms——**16MB/操作数规模手工 blocking 无真实余量（负余量）。[B+A]**

## §5 对判断节点的输入（供 J1–Jn 裁决，不替 human 决策）

1. **naive 起点**：纯 Python 三重循环 @2048 ≈274s（list）/550s（numpy 输入）→ correctness 30s alarm 必超时死，env_7 起点"慢是特性"前提在 2048 破产（实测确证）。可行起点：rowwise 无转置 np.dot（143ms、score 0.048、**逐位一致必过门**、门成本 0.79s）；或 row_halves（7.6ms，score 0.90）。[A]
2. **性能判据（J1 核心输入）**：本机 A 级全族扫描 **无任何 >1.0 路径**（BLAS 平价=天花板，协议内 0.98–1.0；所有 mp/threaded/blocked/Strassen/ctypes 变体 ≤0.92 且多数死在容差）。合理判据候选：score≥0.5（=blocked-256 水平）或"迭代内 best score 收敛至 ≥0.9"+ 失败归因要求。[A]
3. **特征空间取舍**：能出 score>0.5 的格只剩 BLAS 族（np.matmul 直调、einsum_opt、M 维分解）；mp/multiprocess/pure_python/Strassen/K-blocking 格全部结构性死（实测）；BLAS 族在 env_7 从未被采样——本轮建议 weight 收缩到 BLAS 族+其 M 分解变体。[A]
4. **ref band**：[5.0, 10.0]ms 建议（见 §3）；K=3+warmup1 足够（进程内 IQR 0.06ms 极稳）；测量噪声地板：跨进程 score ±5–10%。[A]
5. **timeout**：correctness 30s / bench 60s alarm 对所有非 naive 候选充裕（≤0.8s）；naive 级中间态（若进化产出）会死于 30s——可接受（自然淘汰）。[A]
6. **容差裁决点（新）**：1e-3 vs f32 锚 @2048 = 只放行 BLAS 求和序实现（搜索空间大幅收窄但判定干净、零假阳性）；若放宽到 5e-3，K-blocking 族（1.6–1.9e-3）与 Strassen（2.9e-3）数值上复活——但性能上仍全劣于 BLAS（§1），放宽只增加"合法但慢"的格子。两案均无作弊通道风险，属 human 偏好。注意锚自身噪声 1.73e-3>1e-3 意味着"绝对正确"已不可判定。[A]
7. **token/预算**：本轮未涉及；score>1.0 不可达 ⟹ KR4 判据若设 score≥1.0 必然失败，需按 2 重设。

## §6 Gaps

- ctypes ColMajor 直调未测（RowMajor 10.0ms 已慢 45%；ColMajor 大概率≈6.9 持平而非反超）[C]
- 持久进程池 steady-state 未测（门⑨必拒，无实测价值）
- M5 Max 专属 SME 行为无公开文档，SME 单协处理器结论由 M2–M4 世代文档 + 本机 cpu_ratio/VECLIB 实测推断 [B→A]
- Strassen-2+（49×512 叶）未测（Strassen-1 已双杀，外推更差）[C]
- bf16/f16 第一性原理死于容差（相对 eps 7.8e-3/4.9e-4 >> 1e-3/|C|max 量级要求）[C]
- 背景负载（~4 核忙）对多线程候选的影响未做有/无对照（不可杀用户进程）；24h 运行窗口内噪声漂移未测
- scipy 是否在 venv 未查（即便有也走同一 Accelerate 后端，不改变结论）
