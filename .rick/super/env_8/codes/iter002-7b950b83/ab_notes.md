# ab_notes.md — 迭代 2 A/B 归因（einsum canonical vs swapped vs np.matmul）

- 候选：`solution.py` = `np.einsum('ij,jk->ik', a32, b32, optimize=True)`（canonical 拼写）
- A/B 脚本：`ab_test.py`（临时脚本，进程内配对、每轮随机次序 10 对、每对 K=3 取中位；不改变提交实现）
- 环境：env_8 venv numpy 2.0.2，macOS Accelerate/vecLib BLAS，@2048×2048 f32

## 正确性 sanity（两轮均过）

- `canonical == np.matmul` 逐位一致：True
- `swapped == np.matmul` 逐位一致：True
- 自检门：max_error=0.0，sizes [64, 2048] × 双 seed，双跑逐位一致 → 通过

## A/B 结果（两轮独立进程，负载态 load avg 4.4-4.9）

| 轮次 | canonical（中位之中位） | swapped | matmul | canonical 比 swapped 快（均值） | canonical 更快的对数 |
|---|---|---|---|---|---|
| 1 | 7.167 ms | 7.214 ms | 7.082 ms | +0.040 ms | 7/10 |
| 2 | 7.064 ms | 7.106 ms | 6.911 ms | +0.069 ms | 7/10 |

逐对差（swapped − canonical，ms）：
- 轮 1：[0.153, 0.134, 0.002, 0.123, −0.195, 0.036, 0.074, 0.138, −0.045, −0.023]
- 轮 2：[0.012, 0.147, 0.147, 0.007, −0.083, −0.110, −0.083, 0.028, 0.347, 0.277]

## 归因（如实记录）

1. **方向成立、幅度未达 0.2ms**：两轮 canonical 均值快 0.04-0.07ms、7/10 对更快，
   但未复现预研的 0.24-0.64ms（6/6 独立进程）。→ 预研机制主张『T-标志 gemm 在
   Accelerate 负载态比 C-标志稳定 ≥0.2ms』的**幅度量化在本次机器状态下未获证实**
   （可证伪点①按字面触发）；但『canonical 不劣于且多数时候优于 swapped』的方向性
   在全部 20 个配对样本中成立（均值 +0.04~0.07ms，无系统性反向）。
2. **canonical 仍是正确选择**：两种拼写都逐位一致于 np.matmul（max_error=0.0），
   canonical 从未出现系统性变慢；叠加预研观察——bench 门『候选先计时』时序对
   C-标志先计时函数（swapped）惩罚更重（预研 swapped 门协议退化到 7.5-8.6ms）——
   canonical 拼写在门协议下的期望更优。不构成改变提交实现的证据。
3. **einsum 包装开销可见**：canonical 比 np.matmul 慢 ~0.1-0.15ms（parse+path+wrapper
   + tensordot/转置视图派发），与 expected_behavior 中『@64 约 12μs』的量级一致
   （@2048 下相对开销 <2%）。
4. **结论**：提交实现保持 canonical 拼写不变；『≥0.2ms 幅度』作为未证实项记录在案，
   供谱系归因参考（负载状态、thermal、BLAS 内核选择的日内漂移均可能压缩 T/C 差）。

## 终验复跑（plan step 5）

`env/data/bench.py --candidate solution.py`（K=3，门协议：warmup1 + 候选先计时 + ref 后计时）：

| 轮 | runtime_ms | score | ref_runtime_ms | ref ∈ [5,10] |
|---|---|---|---|---|
| 1 | 7.144 | 0.994 | 7.103 | ✓ |
| 2 | 7.202 | 1.074 | 7.736 | ✓ |
| 3 | 7.826 | 0.870 | 6.809 | ✓ |
| 4 | 7.240 | 0.968 | 7.005 | ✓ |
| 5 | 7.053 | 0.999 | 7.049 | ✓ |

- 计划内 ×3：7.14 / 7.20 / 7.83 ms，score 0.99 / 1.07 / 0.87，ref 全落 [5,10] 带；
  为归因追加 ×2：7.24 / 7.05 ms，score 0.97 / 1.00。
- 5 轮分布：runtime_ms 中位 7.20（区间 7.05-7.83），score 中位 0.99（区间 0.87-1.07）。
- 轮 3 的 score 0.870 低于预期带 0.93-1.00（可证伪点③）：其 timings 首次计时
  9.12ms 为干扰离群（中位 7.83），而同轮 ref 相位 6.81ms 为全部 5 轮最快——
  「候选先计时 + 机器状态在候选相位与 ref 相位间漂移」的时序不对称特征，
  非包装开销系统性变化（轮 4/5 立即回到 0.97/1.00）。归因：机器状态漂移。
- 全部 5 轮 ref 均落 [5,10] 合理带，无 plausibility 拒收；max_error=0.0（自检门
  [64,2048]×双 seed、双跑逐位一致），静态检查无命中。
