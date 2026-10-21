# ab_notes.md — 迭代 11 A/B 归因（M 维行四分 np.matmul 派发 vs 父本行二分 / iter010 np.dot 行四分）

- 候选：`solution.py` = iter008 父本骨架（np.matmul 派发 + out= 切片直写预分配缓冲）
  + 单变量变更：行二分（h=m//2 两次乘）→ 行四分（edges=[(i*m//4,(i+1)*m//4) for i in range(4)]，
  四块非空即整阵 np.matmul 直写；m==0 退化守卫取自 iter010）
  （target_cell: dispatch_target=np_matmul|impl_strategy=m_quarters）
- 父本：codes/iter008-1446e1c9/solution.py = np_matmul|m_halves 精英（同机同负载配对 7.32ms）
- 对照形态：codes/iter010-bf6c26c0/solution.py = np_dot|m_quarters（行四分 + np.dot 派发）
- A/B 脚本：`ab_test.py`（同进程 k=9 中位 × 5 轮，轮内固定次序 候选→父本→iter010 形态→full 锚；
  内联正确性 sanity：奇数行 257×130@130×129、tall/wide 非方阵、1-D b、嵌套 list、m<4、m==0、
  双跑一致）
- 环境：env_8 venv numpy 2.0.2，Apple M5 Max / macOS Accelerate(vecLib)，@2048×2048 f32，
  **负载态 load avg 5.09-5.47**（与 iter008 记录的 5.0-5.3 负载态延续，绝对水平可横向对照）

## 实现差异（diff vs 父本 iter008，单变量）

行分裂机制 2→4（边界公式与 m==0 守卫取自 iter010），其余逐字不动：
```diff
     m = a32.shape[0]
+    if m == 0:
+        return np.zeros((0,) + b32.shape[1:], dtype=np.float32)
     c = np.empty((m,) + b32.shape[1:], dtype=np.float32)
-    h = m // 2
-    np.matmul(a32[:h], b32, out=c[:h])
-    np.matmul(a32[h:], b32, out=c[h:])
+    edges = [(i * m // 4, (i + 1) * m // 4) for i in range(4)]
+    for s, e in edges:
+        if e > s:
+            np.matmul(a32[s:e], b32, out=c[s:e])
```
asarray float32 守卫、c=np.empty 预分配、模块级 np.seterr、契约全部逐字不动。
恰好四次 np.matmul 派发（m%4 余行归靠后块，无缝覆盖 [0,m)）、无转置、无 concatenate、
不动 K 维、无模块级缓存或状态。

## 正确性（sanity + 自检）

- ab_test.py 内联 sanity（9 案例 × 双跑一致）：**全部通过**（判据 = env 容差 1e-3 + 双跑逐位一致）。
  - 2-D sgemm 主路径（奇数行 257×130@130×129、tall/wide 非方阵、square 2048、嵌套 list、m==0）：
    **全部 max_err=0.0 位精确**——四块边界无缝覆盖（形状/数值逐位对齐 np.matmul 锚），
    out= 行切片仍走 cblas_sgemm 快路径（@2048 实测 8.4-8.7ms 带，无 >50ms 量级 gufunc 慢回退）。
  - **两处 ε 级非位精确（如实记录，远小于容差 1e-3，双跑一致）**：
    1-D b m=128（四块各 32 行）max_err≈1.9e-06~3.8e-06；m=3 退化（三块各 1 行）max_err≈6e-08~1.2e-07。
- **ε 级差异归因（对照实验定位）**：同输入下 quarters_matmul 与 quarters_dot 的 ε 完全相同，
  halves（64 行块）与 full 均为 0.0——即差异源是 **vecLib 对 32 行块 gemv / 单行块选用了
  与整阵不同的累加内核（K 求和序微差）**，与派发入口（matmul/dot）无关；m=257/2048 的 1-D b
  亦为 0.0（块行数≥64 时内核一致）。父本「M 分解不改 K 求和序」的论断在 2-D sgemm 路径严格成立，
  在 1-D gemv 的小 M 块与 m<4 单行块路径应弱化为「ε 级差异 ≤4e-06，容差内」。
- 自检门（self_check.py → test_correctness.py）×4 轮：passed，**max_error=0.0**，
  sizes [64, 2048] × 双 seed（42 + urandom），双跑逐位一致，静态检查干净
  （框架门只测 2-D 方阵，位精确构造性成立）。

## 自检单发（self_check.py --k 1，跨进程 ×4 轮）

| 轮 | runtime_ms | score | ref_runtime_ms |
|---|---|---|---|
| 1 | 8.533 | 0.800 | 6.829 |
| 2 | 8.703 | 0.832 | 7.237 |
| 3 | 8.553 | 0.814 | 6.959 |
| 4 | 8.440 | 0.873 | 7.366 |

bench.py k=3 协议单跑：runtime 8.548ms（timings 8.619/8.455/8.548），ref 7.052ms，
score 0.825。ref 基线全部落在 [5,10]ms 合理带内。

## 配对 A/B（同进程 k=9 中位 × 5 轮，env/.venv 同解释器）

| 轮 | quarters_matmul_out（候选） | halves_matmul_out（父本） | quarters_dot_out（iter010 形态） | full_matmul（锚） |
|---|---|---|---|---|
| 1 | 8.397 | 7.274 | 8.462 | 6.690 |
| 2 | 8.400 | 7.421 | 8.535 | 6.701 |
| 3 | 8.403 | 7.336 | 8.496 | 6.603 |
| 4 | 8.440 | 7.306 | 8.528 | 6.700 |
| 5 | 8.452 | 7.315 | 8.541 | 6.831 |
| **中位之中位** | **8.403** | **7.315** | **8.528** | **6.700** |

轮间绝对水平漂移极小（候选 8.397-8.452，±0.03%），四变体 5/5 轮次序完全稳定
（full < halves < quarters_dot ≈ quarters_matmul），全部变体 max_err=0.0。

### 判据判读（plan step 4 两个量级判断 + step 5 预期带）

| 判据 | 要求 | 实测 | 结论 |
|---|---|---|---|
| ① 正确性 | max_err=0.0（2-D 主路径） | 0.0（全部轮 + sanity 主路径；两处退化路径 ε≤3.8e-06 在容差内） | **PASS** |
| ② runtime 预期带 | 8.2~8.9ms | 配对中位 **8.403ms**；自检单发 8.44-8.70；bench k=3 中位 8.548 | **PASS** |
| ③ 2→4 分裂结构性损失 | +0.8~+1.1ms | quarters_matmul − halves_matmul = 8.403 − 7.315 = **+1.088ms** | **PASS**（带上缘） |
| ④ np.dot→np.matmul 派发层差异 | −0.05~−0.15ms | quarters_matmul − quarters_dot = 8.403 − 8.528 = **−0.125ms** | **PASS** |
| ⑤ vs np_dot\|quarters 精英 | 持平或略优 0.05~0.15ms | 快 0.125ms（5/5 轮同号，逐轮 Δ ∈ [−0.09, −0.14]） | **PASS** |
| ⑥ 阶梯单调性 | full < halves < quarters | 6.700 < 7.315 < 8.403 | **成立** |

**OVERALL: PASS**——「分裂代价随 M 粒度单调上升」在 np_matmul 轴被独立复证
（+1.088ms vs np_dot 轴的 +0.93ms，同量级），np.matmul 直派发的 −0.05~−0.15ms
优势在 quarters 粒度下与 halves 粒度下（iter008 记录的 −0.06~−0.14ms）一致。

## 归因（如实记录）

1. **2→4 行分裂结构性损失 +1.088ms（判据③带上缘）**：与 np_dot 轴阶梯数据点
   （iter010：halves 7.51 → quarters 8.45，+0.93ms）同量级、略偏高；也与 builder 预研探针
   （quarters_matmul 8.36 vs halves_matmul 7.39，+0.97ms）一致。候选原因：4×512 行块下
   vecLib/AMX 的小 M 内核选择差异使每块 sgemm 效率低于 2×1024 行块——结构性损失
   （派发次数×2 + 每块 M 减半的内核效率损失），非负载噪声（轮间漂移 ±0.03%）。
2. **派发层差异 −0.125ms（判据④）**：np.matmul 对 2-D × 2-D C 连续输入直连 cblas_sgemm，
   省 np.dot 的 dispatch 包装层；该优势在 halves（iter008：−0.06~−0.14ms）与 quarters
   （本次：−0.09~−0.14ms）两种粒度下幅度一致——派发层差异与分裂粒度正交，量级判断成立。
3. **格内定位**：m_quarters 格（dispatch_target=np_matmul × impl_strategy=m_quarters）的
   格内最优写法 = np.matmul 派发 + out= 直写（本候选 8.40ms < iter010 的 np_dot 形态 8.53ms）。
   full_matmul（6.70ms）属另一格（impl_strategy=full），慢于锚约 +1.7ms（~25%），
   与 plan 预期「慢 15~25%」一致——本实验是格子填充与 M 粒度阶梯测量，非最优冲刺。
4. **可证伪点均未触发**：实测 8.40ms 未 <7.8ms（未证伪单调性）、未 >9.0ms（快路径与无缝
   覆盖假设成立）、正确性 2-D 主路径 0.0（M 分解不改 sgemm K 求和序）。
5. **新信息（如实记录）**：1-D gemv 32 行块与 m<4 单行块存在 ε 级（≤4e-06）累加序差异，
   quarters 特有（halves/full 均位精确），源是 vecLib 小 M 块内核选择，不影响任何门
   （框架正确性门只测 2-D 方阵且容差 1e-3），但修正了父本「M 分解不改 K 求和序」的
   全路径普适表述。
6. **远离死族确认**：无 K 分块/Strassen/转置/降精度/多进程/多线程——全部为谱系已证死路，
   本候选未引入任何新机制（相对父本仅分裂粒度单变量）。

## 机器状态记录

当日 21:41-21:43 负载态 load avg 5.09/5.47/5.30（1m/5m/15m），与 iter008 记录的负载态
（5.0-5.3）延续——本迭代绝对水平可与 iter008 横向对照（父本 halves 配对复测 7.32ms vs
iter008 记录的 7.30ms，一致性良好）。所有轮次数字未做剔除地全量列于上表。
