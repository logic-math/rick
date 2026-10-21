# ab_notes.md — 迭代 7 A/B 归因（m_quarters einsum+concat vs 父本 full einsum）

- 候选：`solution.py` = M 维四分 einsum 派发 + concatenate 拼接
  （`edges = [(i*m//4, (i+1)*m//4)]`，逐块 canonical einsum，concat 组装）
- 父本：codes/iter002-7b950b83/solution.py = 整阵 canonical einsum（full）
- A/B 脚本：`ab_test.py`（进程内配对、每轮随机次序 10 对、每对 K=3 取中位；
  另含三档微基准分解与正确性 sanity；不改变提交实现）
- 环境：env_8 venv numpy 2.0.2，macOS Accelerate/vecLib BLAS，@2048×2048 f32，
  负载态 load avg 3.8-4.7（与预研 4.6-5.0 相近的门态）

## 正确性 sanity（ab_test.py 内联）

- m_quarters == np.matmul **逐位一致 @2048**（max_error=0.0），双跑逐位一致
- 嵌套 list 输入正常（float32 输出）
- 非方阵 (100,37)@(37,50)（M 不整除 4）：max_error=0.0
- m=1/2/3 退化块：max_error ≤ 2.4e-7（小 M 块走不同 BLAS 内核的舍入差，
  远低于 1e-3 容差；@2048/@64 规模为 0.0）
- 自检门（env self_check.py）：passed，max_error=0.0，sizes [64,2048] × 双 seed，
  双跑逐位一致

## 配对 A/B（10 对 × K=3 中位，随机次序）

| 指标 | 值 |
|---|---|
| candidate m_quarters（中位之中位） | **9.429 ms**（区间 9.31-9.49） |
| parent full_einsum（中位之中位） | **7.211 ms**（区间 7.15-7.26） |
| 逐对差（candidate − parent） | +2.10 ~ +2.30 ms |
| 均值 / 中位 | **+2.21 / +2.23 ms** |
| candidate 更慢的对数 | **10/10** |

逐对差（ms）：[2.143, 2.228, 2.097, 2.296, 2.259, 2.086, 2.172, 2.238, 2.277, 2.293]
（ab_test.py 输出的 `parent_minus_candidate_per_pair` 取负即上表）

→ 落在预期带 **+1.9~2.3 ms** 内，且全部 10 对同向：可证伪点①（配对差 <1.5ms，
分裂损失未复现）**未触发**；可证伪点②（>3.0ms / F→C 直写陷阱）**未触发**；
可证伪点③（runtime <7.3ms 意外之喜）**未触发**——纯结构性劣化，符合本格
「结构定位数据点」预期。

## 三因子微基准分解（每档 warmup2 + K=3 × 10 组随机次序）

| 档位 | 中位之中位 | 含义 |
|---|---|---|
| full_matmul | 7.09 ms | 整阵 np.matmul（基线锚） |
| full_einsum | 7.27 ms | 父本路径（整阵 canonical einsum，T-标志 gemm） |
| gemm4_noassemble | 8.17 ms | 纯 4×512 sgemm（dot T-标志，无组装无 einsum 包装） |
| matmul_q_concat | 8.86 ms | np.matmul 分块 + concatenate |
| einsum_q_concat | 9.42 ms | einsum 分块 + concatenate（= 提交实现路径） |

三因子归因（vs 父本 full_einsum 7.27，总差距 +2.15ms）：

1. **sgemm 分裂损失 ≈ +0.9~1.1 ms**（gemm4_noassemble 8.17 − full 7.09~7.27）：
   4×512 行小 M 内核效率低于 1×2048 整阵（vecLib/AMX 结构性）。预研估 +1.57ms，
   本次复跑略低——方向一致、幅度在负载态间漂移。
2. **einsum 包装 ×4 ≈ +0.56 ms**（einsum_q_concat 9.42 − matmul_q_concat 8.86）：
   parse+path+tensordot 派发每块一次。预研估 ~+0.45ms，同量级。
3. **concatenate 拼接拷贝 ≈ +0.69 ms**（matmul_q_concat 8.86 − gemm4_noassemble 8.17）：
   16MB F-布局块 → 连续输出。预研估 ~+0.35ms；本档差含 dot/matmul 标志差，
   上界估计。合计 0.9~1.1 + 0.56 + 0.69 ≈ +2.2~2.3ms，与配对实测 +2.21ms 吻合。

组装陷阱复验：本次未走 out[s:e]=res 直写路径（预研实测该路径 F→C 转置拷贝
12.93ms 病态）；candidate 稳定 9.3-9.5ms，证明 concatenate 组装路径干净。

## 终验复跑（plan step 5）

`env/data/bench.py --candidate solution.py`（K=3，门协议：warmup1 + 候选先计时
+ ref 后计时），5 轮独立进程：

| 轮 | runtime_ms | score | ref_runtime_ms | ref ∈ [5,10] |
|---|---|---|---|---|
| 1 | 9.260 | 0.796 | 7.370 | ✓ |
| 2 | 9.330 | 0.738 | 6.887 | ✓ |
| 3 | 9.286 | 0.750 | 6.966 | ✓ |
| 4 | 9.327 | 0.770 | 7.179 | ✓ |
| 5 | 9.326 | 0.769 | 7.176 | ✓ |

- 5 轮分布：runtime 中位 **9.33 ms**（区间 9.26-9.33，极差 0.07ms 无离群轮），
  score 中位 **0.77**（区间 0.74-0.80）。
- 全部落在预期带 runtime 9.0-9.6ms / score 0.72-0.80；5 轮 ref 全落 [5,10]
  合理带，无 plausibility 拒收。
- 与 iter002 轮 3 的时序不对称离群（9.12ms 首计时干扰）不同，本次 15 个计时
  样本全部 9.20-9.40ms——m_quarters 的 +2.2ms 结构性劣化远大于负载噪声，
  分布稳定是本格数据点的特征。

## 结论

1. **谱系数据点落位**：M 粒度阶梯（einsum full 7.2 → quarters 9.3-9.4）确认
   「M 分四份在 einsum 派发下每份损失 ~0.55ms」的粒度-代价曲线，为 M 方向
   进一步细分（halves 8.23 预研 / quarters 9.19 预研）提供本机复测锚点。
2. **三因子分解复现**：分裂损失 + 包装 + 拼接 ≈ +2.2ms，与预研 +2.37ms 同量级
   （各因子幅度有 ±0.3ms 负载态漂移，总量稳定）。
3. **死族远离声明**：未用 einsum out= / K 分块 / Strassen / 转置后 gemv /
   降精度 / 多进程 / 多线程；未走 out[s:e]=res 直写陷阱；契约（matmul(a,b)、
   asarray f32、seterr、嵌套 list）与父本一致。
