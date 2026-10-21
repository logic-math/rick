# ab_notes.md — 迭代 8 A/B 归因（m_halves 骨架 np.matmul 派发 vs 父本 np.dot 派发）

- 候选：`solution.py` = iter001 m_halves 骨架（h=m//2 上下半 + out= 切片直写预分配缓冲）
  + 派发入口单变量替换 np.dot → np.matmul（target_cell: dispatch_target=np_matmul|impl_strategy=m_halves）
- 父本：codes/iter001-4fe95963/solution.py = 同骨架 + np.dot 派发（档 7.42ms）
- 派发入口证据源：codes/iter003-93ddaebc/solution.py = 整阵 np.matmul 直接派发（6.86ms，
  仅作轴值证据，不复制其代码形态）
- A/B 脚本：`ab_test.py`（同进程 k=9 中位 × 多轮；内联正确性 sanity：
  奇数行 257×130@130×129、非方阵、1-D b、嵌套 list、双跑一致）
- 环境：env_8 venv numpy 2.0.2，Apple M5 Max / macOS Accelerate(vecLib)，@2048×2048 f32，
  **负载态 load avg 5.0-5.3**（高于 iter003 记录的 3.8-5.3 带上半段——绝对水平整体抬升）

## 实现差异（diff vs 父本 iter001，单变量）

仅两处派发调用 + docstring 标注：
```diff
-    np.dot(a32[:h], b32, out=c[:h])
-    np.dot(a32[h:], b32, out=c[h:])
+    np.matmul(a32[:h], b32, out=c[:h])
+    np.matmul(a32[h:], b32, out=c[h:])
```
asarray float32 守卫、c=np.empty 预分配、模块级 np.seterr、契约全部逐字不动。
恰好两次 np.matmul 派发、无转置、无 concatenate、不动 K 维、无模块级缓存或状态。

## 正确性（sanity + 自检全过）

- ab_test.py 内联 sanity（7 案例 × 双跑一致）：**全部 max_err=0.0**——奇数行 257×130@130×129、
  tall/wide 非方阵、square 2048、1-D b（128×64@64 → (128,) 与 257×64@64 → (257,)）、
  嵌套 list 输入。**out= 切片直写未落入 gufunc 慢回退**（若回退，@2048 runtime 会飙至
  >50ms 量级，实测全部在 7-8ms 带）。
- 自检门（self_check.py → test_correctness.py）×4 轮：passed，**max_error=0.0**，
  sizes [64, 2048] × 双 seed（42 + urandom），双跑逐位一致，静态检查干净。
  M 分解不改 K 求和序 + np.matmul 2-D × C 连续 out 切片走与锚相同的 cblas_sgemm
  路径 → 逐位一致是构造性成立的。

## 自检单发（self_check.py --k 1，跨进程 ×4 轮）

| 轮 | runtime_ms | score | ref_runtime_ms |
|---|---|---|---|
| 1 | 7.468 | 1.003 | 7.486 |
| 2 | 7.711 | 0.938 | 7.233 |
| 3 | 8.213 | 0.909 | 7.468 |
| 4 | 8.134 | 0.834 | 6.779 |

score 分布 [0.834, 0.909, 0.938, 1.003]，中位 ~0.92——落 plan 预期带 ~0.90-0.95
（轮 4 的 0.834 为候选先计时相位吃负载 burst 的单发噪声，与 iter003 轮 1 的 0.844 同型；
k=1 单发对负载态极敏感，框架终验 K 次中位更稳）。

## 配对 A/B（同进程 k=9 中位 × 5 轮，env/.venv 同解释器）

| 轮 | halves_matmul_out（候选） | halves_dot_out（父本） | full_matmul | concat_golden | 候选−concat | 候选−dot |
|---|---|---|---|---|---|---|
| 1 | 7.426 | 7.487 | 6.792 | 7.805 | −0.379 | −0.061 |
| 2 | 7.946* | 8.035* | 6.760 | 8.252* | −0.306 | −0.089 |
| 3 | 7.286 | 7.429 | 6.697 | 8.064 | −0.778 | −0.143 |
| 4 | 7.302 | 7.374 | 7.743* | 7.698 | −0.396 | −0.072 |
| 5 | 7.262 | 7.349 | 7.650* | 7.643 | −0.381 | −0.087 |

\* 轮 2 全体 halves/concat 变体抬升至 ~7.9-8.3（负载 burst）；轮 4/5 的 full_matmul
抬升至 ~7.65-7.74（同 burst 相位不对称——full 单发大阵列对瞬时争用更敏感）。
**轮内配对次序固定（候选先于父本同相位），轮间绝对水平随机器负载态漂移**
（iter003 已记录的同型 bimodality：干净态 ~6.8-7.3 / 负载态 ~7.9-8.6）。

### 判据判读（plan step 5 成功判据）

| 判据 | 要求 | 实测 | 结论 |
|---|---|---|---|
| ① 正确性 | max_err=0.0 | 0.0（全部轮 + sanity 7 案例） | **PASS** |
| ② runtime 带 | 7.2-7.6ms | 5 轮中位 [7.26, 7.29, 7.30, 7.43, 7.95]，中位之中位 **7.30ms**；4/5 轮在带内，轮 2 为负载 burst（同轮 concat 8.25 同步抬升） | **PASS**（中位口径；单轮 burst 如实记录） |
| ③ vs concat | 快 ≥0.25ms | 5/5 轮全部满足：Δ ∈ [−0.78, −0.31]ms，中位 Δ ≈ −0.38ms | **PASS** |
| ④ vs halves_dot | 不慢 >0.2ms | 5/5 轮候选反而**快** 0.06-0.14ms | **PASS** |

**OVERALL: PASS**——「np.matmul 可平价接管 m_halves 派发且 out= 直写是格内最优写法」
的假设在配对口径下成立。

## 归因（如实记录）

1. **派发入口收益方向正确但幅度小**：候选比父本 halves_dot_out 稳定快 0.06-0.14ms
   （5/5 轮同号）——np.matmul 对 2-D × 2-D C 连续输入直连 cblas_sgemm，省去 np.dot
   的 dispatch 包装差异。该幅度 < 跨进程噪声 ±5-10%（~0.4-0.7ms），与 plan 预期
   「持平至 −0.1ms」一致。
2. **out= 直写收益确认**：候选 vs concat 金样例稳定快 0.31-0.78ms（5/5 轮）——
   消除 2×8MB 中间分配 + 16MB concatenate 拷贝，与 iter001 档记录的 ~0.3ms 一致；
   干净轮（3/4/5）幅度更大（0.38-0.78ms），负载轮分配器争用下拷贝代价更高。
3. **绝对水平负载敏感（非实现因素）**：机器当日 load 5.0-5.3（负载态），轮 2 全体
   变体同步抬升 ~0.5ms、轮 4/5 full_matmul 反超 halves 变体——多线程 sgemm 的线程
   调度争用使绝对 runtime bimodal（iter003 ab_notes 已系统记录该现象与交叉点）。
   配对口径（轮内同相位对照）不受此污染，判据③④的 5/5 同号是可靠信号。
4. **格内定位**：m_halves 格（dispatch_target=np_matmul × impl_strategy=m_halves）
   的格内最优写法 = np.matmul 派发 + out= 直写（本候选）。full_matmul（6.7-7.7ms）
   属另一格（impl_strategy=full），不在本格比较域内；其负载态波动（6.70→7.74）
   与 halves 形态的相对优势随机器状态翻转，值得后续迭代在 full 格内复查。
5. **远离死族确认**：无 K 分块/Strassen/转置/降精度/多进程/多线程——全部为谱系
   已证死路，本候选未引入任何新机制。

## 机器状态记录

当日 21:13-21:16 全程负载态 load avg 4.97-5.30（1m/5m/15m），无重 burst 但持续
高于 iter003 记录的干净态；所有轮次数字未做剔除地全量列于上表。
