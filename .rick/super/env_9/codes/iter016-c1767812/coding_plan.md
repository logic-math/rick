# 编码执行计划 · 迭代 16 · dispatch_target=np_matmul|impl_strategy=m_halves

## 0. 前置确认（热点）

iter16 profile 已录（实验记录 §1）：`matmul` self 100%，无 Python 层热点——
父本结构干净，改动点唯一且明确：两次半阵 gemm 的**执行结构**（顺序→并发）。
无重复跑 profile 的必要（builder 探针 + 同窗官方 benchmark 复测父本 607.9 双向印证）。

## 1. 改动点（父本结构 → 目标结构，逐处说明）

父本 = `codes/iter004-e77e66ca/solution.py`（本格精英 484.43ms），候选落
`codes/iter016-c1767812/solution.py`。**单变量原则：只改执行结构，不动其它任何东西。**

| # | 位置 | 父本 | 候选 |
|---|---|---|---|
| 1 | import 区 | `import numpy as np` | 增加 `import threading`（标准库，无新依赖） |
| 2 | `matmul()` 体内 | `c = np.empty((m,n))` 后**顺序**两次 `np.matmul(a32[:mid], b32, out=c[:mid])` / `np.matmul(a32[mid:], b32, out=c[mid:])` | 同一 `c` 预分配不变；两次半阵 gemm 改为**两个 `threading.Thread` 并发执行**，worker 函数第一行 `np.seterr(divide/over/invalid="ignore")`（线程局部——探针实证 vecLib 虚假 FP 警告只在 worker 线程冒出），随后各自 `np.matmul(..., out=c[...])`；`try/finally` 保证两线程 join 后才 return `c` |
| 3 | docstring | 顺序两半描述 | 更新为并发两半描述（并发重叠依据：np.matmul BLAS 段释放 GIL + Accelerate 允许并发 sgemm 重叠，探针 F -104ms 实证） |

**逐字保留的契约**：`matmul(a, b)` 签名 / `np.asarray(..., dtype=np.float32)` /
模块级 `np.seterr` 三项抑制 / `_self_check` 用例表（方阵/奇数行/非方阵/嵌套 list/M=0/M=1）/
`mid = m // 2` 奇数行语义 / C 连续行切片 out= 直写（numpy PR #29179 非连续 out 坑不触碰）。

**静态检查红线**（verify ⑩ 全文正则扫描，含注释）：源码不得出现
`sys.modules`、`__main__.`（属性访问形态）、`time.xxx =`、`signal.alarm =`、`__builtins__`。

## 2. 风险卡点（已知的 API 限制 / 边界条件 / 性能陷阱，附来源）

1. **np.seterr 是线程局部状态**（builder 探针 2026-10-26 实证：F 臂 worker 线程冒出
   vecLib 虚假 FP 警告，主线程模块级 seterr 不覆盖 worker）——worker 函数**第一行**必须
   `np.seterr(divide="ignore", over="ignore", invalid="ignore")`。数值本身 bitwise 正确，
   但警告噪声 + 警告机制短暂持 GIL 会拖慢并发收益。
2. **两线程写同一 `c` 的不相交行切片**：`c[:mid]` / `c[mid:]` 无别名无交叠
   （numpy#8440 的别名坑不触发）；行切片是 C 连续视图，out= 直写合法（父本 docstring
   已录 numpy 官方文档 + PR #29179 + SO「views vs copy」依据，沿用）。
3. **过订阅反噬**（ThreadPinning.jl 文档警示，调研报告 §2）：安静窗 2×并发 gemm
   各自拉满 BLAS 线程队可能过订阅——这是本实验**主要证伪面**，§3 判据已备
   （中位 ≥ 484.43 且 ref ∈ 380-460 稳态带 → 机制证伪，如实 reject）。
4. **线程生命周期**：join 必须在任何返回路径前完成（try/finally），否则 benchmark
   计时窗口内可能残留活跃线程 → 读数污染 + verify ⑦ 双跑一致性风险。
   M=0 边界：mid=0 两半皆空，np.matmul 空维合法，线程照常跑空 gemm 后 join。
5. **每调用建 2 线程的开销**：~0.1ms 级（探针实证），对 500ms 量级读数无感；
   不引入线程池（池 = 变量数增加，违反单变量原则，且 iter13 线程管理类变体恒劣）。

## 3. 测试清单（每个测试断言什么；对应实验假设的哪一部分）

测试文件：`codes/iter016-c1767812/test_solution.py`（纯 Python runner，无 pytest 依赖）。

| 测试 | 断言 | 对应假设 |
|---|---|---|
| `test_bitwise_vs_np_matmul_anchor` | n∈{64,512,8192} × seed∈{42,59}，候选输出 `np.array_equal` 逐位 == `np.matmul(a32,b32)` 锚，dtype/ndarray 类型正确 | 正确性契约 + verify ⑤⑥（双规模双 seed） |
| `test_shape_robustness` | 方阵/奇数行(129×64)/奇列(64×127)/M=1/M=0/嵌套 list 全部 bitwise == 锚且形状正确 | 契约逐字保留（§4 步骤 1） |
| `test_double_call_bitwise_consistent` | 同输入两次调用逐位一致 | verify ⑦（抓缓存/特判） |
| `test_halves_run_concurrently_off_main_thread` | **判别测试（先红）**：以 spy 包裹 `np.matmul` 记录每次派发的调用线程 ident——两次半阵 gemm 必须发生在主线程之外的 2 个 worker 线程 | 实验假设的机制本体（顺序→并发结构变更） |

红→绿协议：先把父本原样拷入 `solution.py` 跑测试——前三个绿（契约已在父本成立），
`test_halves_run_concurrently_off_main_thread` **必红**（父本两次 gemm 都在主线程，
caller_idents == {main_ident}）→ 确认测试真的在测本次改动；随后实现并发变更使其转绿。

## 4. 验证顺序（verify → benchmark → gate_check）

1. TDD 绿后自检：`env/.venv/bin/python codes/iter016-c1767812/solution.py`（`_self_check` 全 bitwise）。
2. `env/.venv/bin/python env/skills/verify.py --candidate codes/iter016-c1767812/solution.py`
3. `env/.venv/bin/python env/skills/benchmark.py --candidate codes/iter016-c1767812/solution.py`
4. `env/.venv/bin/python env/skills/gate_check.py --candidate codes/iter016-c1767812/solution.py --baseline 484.4295`

**门禁纪律**：各至多一次、读数即记录、不复跑不追噪声（iter12-15 教训）。
**判据**（§3）：admit = 中位 < 484.43；reject 附 ref 判窗（ref ∈ 380-460 → 机制证伪；
ref ≥ 550 负载窗 → 窗口所限，机制证据见探针）。
