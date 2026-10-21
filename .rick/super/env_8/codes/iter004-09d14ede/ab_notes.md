# ab_notes.md — 迭代 4 A/B 归因（rowwise einsum vs naive rowwise np.dot vs 父本整阵 einsum）

- 候选：`solution.py` = 预分配 out + 逐行 `np.einsum('j,jk->k', a32[i], b32, optimize=True)`
  （canonical 拼写；对父本 iter002 的单变量改动——只动 impl_strategy，派发入口不变）
- A/B 脚本：`ab_test.py`（临时脚本，进程内配对、每轮随机次序 8 对、每对 K=3 取中位；
  不改变提交实现）
- 环境：env_8 venv numpy 2.0.2，macOS Accelerate/vecLib BLAS，@2048×2048 f32，
  负载态 load avg 4.7-5.3

## 正确性 sanity

- `candidate == np.matmul` 逐位一致：True（@2048）
- `naive == np.matmul` 逐位一致：True
- `parent == np.matmul` 逐位一致：True
- 自检门（self_check.py）：passed=true，max_error=7.63e-06（@64；@2048 为逐位 0.0），
  sizes [64, 2048] × 双 seed（42 + urandom），双跑逐位一致 → 通过（容差 1e-3）。
  直接运行：`--n 64` 两跑均 7.63e-06 且逐位一致；`--n 2048` 两跑均 0.0。

## A/B 结果（单轮进程内，8 对 × K=3 中位；load 5.11/5.29/5.06）

| 实现 | 中位之中位 | min | max |
|---|---|---|---|
| candidate（rowwise einsum） | 162.119 ms | 161.459 | 163.056 |
| naive（rowwise np.dot） | 136.846 ms | 135.202 | 152.655 |
| parent（整阵 einsum） | 7.070 ms | 7.044 | 7.182 |
| np.matmul | 6.665 ms | 6.616 | 7.684 |

配对差：

- 包装税（candidate − naive，逐对 ms）：
  [24.343, 26.033, 27.250, 8.822, 25.357, 25.501, 26.484, 24.595]
  → 均值 **+23.5 ms**，8/8 对候选更慢，7/8 落 +24..+27ms。
- 结构差（candidate − parent，逐对 ms）：
  [154.388, 155.553, 155.270, 154.350, 156.012, 154.607, 155.961, 154.718]
  → 均值 **+155.1 ms**，极稳（±1.6ms 内）。

## 归因（两因素机制分解，如实记录）

1. **rowwise 内存墙（结构因素，~137ms）**：每输出行完整流过 B（2048×16MB=32GB
   流量）。naive rowwise np.dot 本轮 136.8ms，与预研 144.4ms 同级（负载态带宽内）。
   这是 rowwise 结构性下界——不管派发入口是 np.dot 还是 einsum 都绕不开。
2. **einsum 包装税（每行因素，~23.5ms）**：candidate − naive 配对差均值 +23.5ms，
   与量级模型一致（@64 微基准 11.3μs/调用 vs np.dot 0.55μs，@2048 折 ~13μs/行
   ×2048≈27ms；本轮配对差落在 +24..+27ms，略低于预算 27ms 上沿——量级模型成立）。
   8 对中 1 对为 +8.8ms（该对 naive 侧 152.7ms 离群，机器状态噪声，方向不变）。
3. **可证伪点核验**：
   - ① @2048 max_error=0.0（逐位一致，双 seed）——求和序不变主张**未被证伪**，实测确认；
   - ② 配对 A/B 候选−naive 均值 +23.5ms，落 +20..+35ms 带内——包装税量级主张**成立**；
   - ③ runtime 150-210ms 带内（见下终验 3 轮 + A/B 162.1）——内存墙量级主张**成立**。
4. **结论**：candidate = naive（内存墙 ~137ms）+ 包装税（~23.5ms）≈ 160ms，
   与实测 162.1ms 拟合；比父本慢 ~23×（rowwise 结构性不可能接近 7ms 族）。
   本格（dispatch_target=einsum_path|impl_strategy=rowwise_vectorized）首条实测证据成立。

## 终验复跑（plan step 5）

`env/data/bench.py --candidate solution.py`（K=3 门协议：warmup1 + 候选先计时 + ref 后计时）×3：

| 轮 | load avg | runtime_ms | score | ref_runtime_ms | ref ∈ [5,10] |
|---|---|---|---|---|---|
| 1 | 4.77/5.21/5.03 | 180.315 | 0.0392 | 7.064 | ✓ |
| 2 | 4.71/5.19/5.03 | 167.481 | 0.0460 | 7.709 | ✓ |
| 3 | 4.71/5.19/5.03 | 169.722 | 0.0458 | 7.781 | ✓ |

- 3 轮分布：runtime_ms 中位 169.7（区间 167.5-180.3，全部落预期带 165-185；
  轮 1 偏高为候选相位偶发干扰——timings 首次 177.4/181.1，与轮 2/3 的 167-170 稳态
  相比是负载噪声，方向不系统性）；score 中位 0.046（区间 0.039-0.046，预期带
  0.036-0.05 内）；ref 全落 [5,10] 合理带，无 plausibility 拒收（score_max=2.0、
  runtime_ms_min=0.01 均远不构成约束）。
- 自检门：passed=true（max_error 7.63e-06 << 1e-3，[64,2048]×双 seed、双跑逐位一致），
  静态检查无命中。
- 与预研门协议两轮（168.0/169.2 @load 5.1-5.4）一致：本机负载态下本实现稳定在
  ~165-180ms 带内。
