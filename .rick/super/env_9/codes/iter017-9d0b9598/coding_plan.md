# 编码执行计划 · 迭代 17 · dispatch_target=np_dot|impl_strategy=full_blas_dispatch

> 本轮是 iter007 同一设计的**流程修复重跑**（visit=2：上轮设计正确、预检全绿，但死于
> turn-budget → no_candidate）。纪律：每个工具至多跑一次、读数即记录、钉死最小动作链。

## 1. 改动点（父本结构 → 目标结构，逐处说明）

- **热点已锁定（不重复跑 profile）**：ctx 注入的 iter17 profile @8192——父本
  `naive_start.py` 11192.5ms，`<listcomp>` self 10400.1ms（99.8%）、`stack` 10.1ms、
  `dot` 1.23ms。瓶颈在 Python 层装配（8192 次逐行循环 + B 全量重读 + stack），非 BLAS
  （内核本身值 ~440ms）。按实验计划 plan_steps 最小动作链，profile 不复跑。
- **唯一 kernel 改动**：
  - 父本：`np.stack([np.dot(a32[i], b32) for i in range(a32.shape[0])])`
    （8192 次逐行 1-D gemv + stack 装配）
  - 目标：`return np.dot(a32, b32)`（**单次整阵 2-D cblas_sgemm 直派**，本格定义形态；
    继承 iter007 已写好的同格一行内核）
- **契约零改动**（照 iter003/iter007 模板）：`matmul(a,b)` 接口 / 嵌套 list 兼容 /
  `asarray→f32` / `np.seterr` 三项抑制 / main 的 warmup+计时+JSON 协议；
  impl 标签 `full_blas_dispatch_np_dot`。
- **增补（照 plan_steps 第 1 条）**：移植 iter010 的 `_self_check` 用例表
  （方阵/奇数行/非方阵/M=1/M=0/嵌套 list；K≥16 逐位、退化形 ≤1 ULP）进 solution 的
  `__main__` 自检。
- **明确不做**（调研结论 + 死族清单）：不用 `out=`（numpy#7124：out 须类型/维数/C 连续
  精确匹配；numpy#18669：matmul+out 未 fault-in 页更慢——plain 调用两入口差异在噪声带）、
  不预转置 B（改求和序，@8192 误差 ~1.4e-2 超容差）、不切 K/N、不做 1-D 化、无中间拷贝。

## 2. 风险卡点（已知的 API 限制 / 边界条件 / 性能陷阱，附来源）

1. **逐位一致是硬门**：2-D f32 下 np.dot 与 np.matmul 须映射同一 cblas_sgemm
   （numpy.dot 官方文档：both 2-D → matrix multiplication；iter007/iter010 预检
   @8192 array_equal=True、maxdiff=0.0）。若 verify max_error > 1e-3 → 假设被否定；
   严禁用转置/out= 变体换速度。
2. **M=0 空行**：父本 `np.stack([])` 抛 ValueError；np.dot 空维 2-D 合法——须测试覆盖
   （TDD 改前红点之一）。
3. **小矩阵内核域效应**：K<16 可能 1-ULP 差异（iter005/iter010 实测）——测试容
   2.4e-7（≪ 1e-3 门容差）。
4. **verify 静态检查红线**（加固⑩，正则全文扫描含 docstring）：候选源不得出现
   `sys.modules`、`__main__.`（带点）、time 属性赋值、`signal.alarm =`、`__builtins__`
   ——注释措辞也避开。
5. **测量纪律**：verify / benchmark / gate_check 各只跑一次，读数立即写进记录 §6；
   不复跑门禁（iter007 根因之一就是流程断裂烧预算）。
6. **gemm/gemv 派发由 numpy 内部决定**（numpy#13732）——整阵 2-D dot 走 gemm；
   父本逐行 1-D gemv 正是 ~25× 慢的结构根源之一。

## 3. 测试清单（test_solution.py，unittest；每个测试断言什么、对应假设哪部分）

| 测试 | 断言 | 对应假设 |
|---|---|---|
| `test_single_2d_dot_call` | spy `numpy.dot`：恰 **1** 次调用、双 2-D 参数、形状=整阵 (64,64)×(64,64) | 「单次整阵派发」结构判据（改前红：父本 64 次 1-D 调用） |
| `test_no_stack_assembly` | `numpy.stack` 调用数 = 0 | 「装配热点消失」（改前红：父本 1 次） |
| `test_zero_rows` | M=0 → 形状 (0,8)、dtype f32、不抛错 | 空维合法性（改前红：`np.stack([])` ValueError） |
| `test_bitwise_square_64` / `_square_128` / `_odd_rows_nonsquare` / `_single_row` / `_odd_square` | K≥16 时与 `np.matmul(f32,f32)` 锚**逐位**相等（err=0.0） | 「K 求和序不变 ⇒ max_error=0.0」（假设核心） |
| `test_small_k_1ulp` | K=8 误差 ≤ 1-ULP（2.4e-7） | 退化形内核域界 |
| `test_nested_lists` | 嵌套 list 输入合法、f32 输出、K=16 逐位 | 契约（list 兼容）不回归 |
| `test_f64_input_converted` | f64 输入 → f32 路径、与转换后 matmul 逐位 | 契约（asarray→f32）不回归 |
| `test_deterministic_double_run` | 同输入两次调用 `array_equal` | verify 加固⑦（双跑一致）预演 |

## 4. 验证顺序（固定：TDD → 自检 → verify → benchmark → gate_check）

1. **TDD**：`cp naive_start.py solution.py`（父本基线就位）→ 跑
   `env/.venv/bin/python test_solution.py -v`（预期红：≥3 项，见上表）→
   换 kernel 为单次整阵 `np.dot` → 重跑全绿（重构仅限注释/结构，测试保持绿）。
2. **自检**：`solution.py --n 64`（绿）→ `--n 8192`（预期 runtime ~434-488ms、
   max_error_vs_matmul=0.0）。
3. **正确性门（一次）**：`env/skills/verify.py --candidate solution.py`
   → 预期 passed=true、max_error=0.0。
4. **性能门（一次）**：`env/skills/benchmark.py --candidate solution.py`
   → 预期 @8192 k=3 中位 ∈ [434,488]ms、score ∈ [0.90,1.05]、ref ∈ [380,700]ms。
5. **融合判定（一次）**：`env/skills/gate_check.py --candidate solution.py
   --baseline 430.8255829999998` → 落 ±5% 软带内即「np_dot 与 np_matmul 入口等价」
   轴级证据（本格无精英，框架终验按豁免；此跑仅为轴证据与同源结论）。
6. **读数即记录**：每个工具读数立即追加 experiment_record.md §6；输出约定 JSON。
