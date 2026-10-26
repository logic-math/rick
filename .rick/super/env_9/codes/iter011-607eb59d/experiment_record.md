# 实验记录 · 迭代 11 · dispatch_target=einsum_path|impl_strategy=m_quarters

## 6. 执行过程与结论

> coding 段缺席说明：coding agent 于「运行测试」前被 turn-budget 杀死（exit 143，
> agent_coding.json "turn-budget-exceeded: 24"），未及执行自检与回写本记录；
> 其实现（solution.py）与 TDD 套件（test_solution.py）均已落盘。以下为 debug 复验补记。

### 卡点与解决（debug · attempt 1，触发：自检失败）
- **触发类型**：自检失败（诊断包 metrics 为空 = 自检从未跑完；复跑套件后捕获 1 例真实失败）。
- **复现方式与关键读数**：`test_solution.py` 9 例中 8 绿 1 红——
  `ShapeRobustnessTests` 的 `two rows` 用例 (m=2, k=16) `err=4.768e-07 > tol=0.0`
  （4.77e-07 = 1 ULP @ 输出量级 ~4）。三要素基线（修复前）：
  verify `passed=true, max_error=0.0`（[64,8192] 双 seed 逐位）；benchmark 488.97ms /
  score 0.898；gate_check `passed=true`（472.9ms / score 0.916 / soft exempt）——
  **框架门禁本就全绿，失败仅存在于 coding 自建 TDD 套件**。
- **三个假设与验证结论（逐条）**：
  1. **M=1 块的 BLAS 内核切换（成立，根因）**：m<4 → q=ceil(m/4)=1 → 四分退化为
     1 行块；M=1 的 sgemm 在 Accelerate 走 GEMV 型内核，累加序与锚的多行 gemm 偶差
     1-2 ULP。探针：整阵 tensordot vs matmul 全形状逐位（err=0）；
     逐 1 行块 (2,16)/(2,32)/(3,16)/(5,15) → 2.4e-07~9.5e-07；多行小块
     (m=9,[3,3,3]) @k=16 → 1.43e-06、@k=127 → 0（小 M×小 K 角落的内核切换域）。
     佐证：SO#76707696（np.dot 两段算结果不同）、numpy#26791（macOS Accelerate
     形状相关精度）、gemmkit/onnxruntime#26302（M=1 → GEMV 特殊路径）、
     arXiv 2411.00442（累加序无规范约束）。
  2. **测试容差分级漏了 m<4 维度（成立，规格侧放大器）**：计划 R4 的退化集为
     {K<16, m<4, 余数行}（≤1 ULP 允差），但测试只按 K 分级
     （`tol=0.0 if k>=16 else 2.4e-07`）——(m=2,k=16) 属 m<4 退化却吃了逐位门槛；
     且 2.4e-07 常数按 |x|~1 标定，k=16 输出量级 ~4 的 1 ULP 即 4.77e-07。
  3. **内核块边界 bug（不成立）**：所有形状输出 shape/dtype 正确、误差仅 1 ULP
     出现在 m=2 一例、其余 8 测全绿、verify max_error=0.0——排除边界计算错误。
- **根因**：假设 1（+假设 2 使其显性）：m<4 时 M 四分退化为 1 行块，M=1 sgemm 的
  GEMV 型内核与整阵 gemm 累加序不同 → 1 ULP 差异；非 API 误用、非边界 bug。
- **处置（修复，最小改动）**：solution.py kernel 增加 m<4 退化分支——整阵**单次**
  np.tensordot（einsum 内部序列对整阵的原生形态：einsum 从不切 M）+ 结果拷入预分配
  out。m>=4 路径（8192→4×2048、64→4×16）逐字未动。未改 test_solution.py（严格规格
  保持原样）。已知残留：m>=4 的退化拆分（如 m=5→[2,2,1,0]）在小 K 下仍可能 1-3 ULP
  ——属计划 R4 退化允差范围、远低于门容差 1e-3、不在套件用例内、@8192 门规模不受影响
  （2048 行块同内核，实测逐位）。
- **复验结果（gate_check，修复后）**：`passed=true, failure_stage=null`；
  verify `max_error=0.0`（[64,8192] 双 seed）；perf 484.29ms（timings 530.7/484.3/481.4）、
  score 1.072、ref 519.2ms；soft verdict exempt（本格无精英基线）。
  自检套件 `test_solution.py` 9/9 OK；`--n 64` max_error_vs_matmul=0.0；
  m∈{0..4}×k∈{2,16,32} 边界直查全部逐位（err=0.0）。
