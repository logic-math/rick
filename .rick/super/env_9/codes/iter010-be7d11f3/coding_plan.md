# 编码执行计划 · 迭代 10

目标格：`dispatch_target=np_dot|impl_strategy=m_halves`（本格首访实作，无精英；对照基线 = 父格精英 iter004 484.43ms）。

## 1. 改动点（父本结构 → 目标结构，逐处说明）

**父本落位（plan 第 1 步「落候选」）**：
- `solution.py` = `codes/iter004-e77e66ca/solution.py`（np_matmul|m_halves 父格精英）逐字复制为起点；
- 按实验计划补齐契约缺口：iter004 无 `main()`，从 `env/code/naive_start.py`（= iter003/iter007 模板）移植 `--n` 自检 JSON 协议（预热 1 次 + 计时 1 次 + `max_error_vs_matmul`），impl 标签 `"m_halves_np_dot"`；
- `__main__` 顺序：`_self_check()`（形状鲁棒用例表，含逐位断言）→ `main()`。

**唯一 kernel 改动（单变量，plan 第 2 步）**——两处 `np.matmul` 换 `np.dot`：
| 位置 | 父本（iter004） | 目标（iter010） |
|---|---|---|
| 下半块 | `np.matmul(a32[:mid], b32, out=c[:mid])` | `np.dot(a32[:mid], b32, out=c[:mid])` |
| 上半块 | `np.matmul(a32[mid:], b32, out=c[mid:])` | `np.dot(a32[mid:], b32, out=c[mid:])` |

- `mid = m // 2`、`c = np.empty((m, n), f32)` 骨架不动；不加转置、不切 K/N、无中间拷贝（累加序约束）。
- docstring 更新为 np_dot 派发口径（引 numpy#18669 与本 session 预检 449.6-466.5ms / maxdiff=0.0）。

## 2. 风险卡点（已知的 API 限制 / 边界条件 / 性能陷阱，附来源）

1. **`np.dot` 的 `out=` 是严格契约**（NumPy 官方文档 v2.x，`numpy.dot` 词条）：out 必须类型正确、**C 连续**、dtype 与 `dot(a,b)` 返回一致——不满足直接 `ValueError` 而非弹性处理（GitHub #7124 实证：非 C-array out 报 "output array is not acceptable"）。→ 本实现只用 C 连续行切片 `c[:mid]` / `c[mid:]`（C 序数组的行切片是连续视图），形状精确匹配 (块高, N)，dtype f32——合法。
2. **out 与输入同阵别名 → 静默全零**（numpy#8440，iter007 调研记录）：本实现 `c` 是全新 `np.empty`，与 `a32/b32` 无别名——安全。
3. **K 求和序不可动**（2411.00442 + domain 死族清单）：M 分解不改 K 累加序（预检 `array_equal=True, maxdiff=0.0` @8192）；任何切 K/N、B 预转置都会撞 1e-3 容差墙（死族 1.4e-2+）——kernel 只做 2 次 2-D 整块 dot，别无其他。
4. **入口不对称**（numpy#18669）：matmul 的 out= 慢路径（输出页未 fault-in 时）正是本实验要绕开的；dot 入口在未 fault-in 输出上有利（预检 449.6-466.5 vs iter004 同 session 480.5ms）。#23588：matmul 对可 gemm 化形状更挑剔（noblas 100× 慢路径），dot 派发更宽容——行切片两侧合法但 dot 侧风险更低。
5. **退化小形状的 1-ULP 内核域效应**（iter005 实测记录）：M 分解 vs 整阵在极小/退化形状（如 M=13,K=5）存在 ~1 ULP（1.19e-07）差异；门规模（K≥16 常规块）逐位一致。→ `_self_check` 与测试采用分级断言：常规块形状逐位（err=0.0）、退化形状 ≤ 2.4e-07（1 ULP 界），全部远在 1e-3 门容差内。
6. **verify.py 静态检查（加固⑩）红线**：候选源码不得出现 `sys.modules`、`__main__.` 属性访问、`time.<attr> =` 赋值、`signal.alarm =`、`__builtins__`——本实现只用 `if __name__ == "__main__":` 样板（明确合法）与 `t0 = time.perf_counter()` 读取（不匹配赋值模式）。
7. **环境事实**（profile.py 本次实测）：numpy 2.0.2 / Accelerate BLAS / 18 核 / memcpy 35.9 GB/s；@8192 工作集 ~768MB 落内存墙区间，合法族坍缩 434-488ms 窄带——预期落点 430-490ms。

**profile 热点确认（本次实测 @8192，与注入记录一致）**：`<listcomp>` self 10918.7ms 占 **99.9%**、`stack` 8.5ms、`dot` 0.29ms——瓶颈确认在 Python 逐行装配层，kernel 派发改动正对瓶颈（改的是「怎么调 BLAS」，不是 BLAS 内部）。

## 3. 测试清单（每个测试断言什么；对应实验假设的哪一部分）

测试文件 `test_solution.py`（stdlib unittest，venv 无 pytest；用 importlib 按门禁同源方式加载 solution.py）：

| 测试 | 断言 | 对应假设 | 预期（父本上） |
|---|---|---|---|
| `test_kernel_dispatches_two_np_dot_calls` | matmul(a,b) 恰好 2 次 `np.dot` 调用，入参 2-D 半块形状 (mid,K)/(m-mid,K)，`out=` 为预分配行切片 (mid,N)/(m-mid,N) | 假设主句「每半单次 2-D np.dot 整阵派发（C 预分配 + 行切片 out= 直写）」 | **红**（父本走 np.matmul，dot 调用数=0） |
| `test_kernel_independent_of_matmul` | 全局替换 np.matmul 为炸弹后 `matmul(a,b)` 仍正确完成（kernel 不依赖 matmul 入口） | 假设「np_dot 派发」格定义 | **红**（父本触发炸弹） |
| `test_main_json_protocol` | `python solution.py --n 64` 末行 JSON：ref=False/n=64/impl="m_halves_np_dot"/runtime_ms>0/max_error_vs_matmul==0.0 | 契约（main --n JSON 协议）+ impl 标签 | **红**（父本无 main()，无 JSON 行） |
| `test_bitwise_identical_to_matmul_anchor` | 512×512（RandomState 42）输出与 np.matmul 锚 `array_equal` 逐位一致 | 假设「max_error=0.0，M 分解不动 K 求和序」 | 绿（父本亦逐位——回归护栏） |
| `test_shape_robustness` | 方阵/奇数行/非方阵/单行/零行/嵌套 list 全部形状正确；常规块（K≥16）逐位、退化形状 ≤1 ULP | 假设边界（mid=m//2 奇数行安全 / 空半合法） | 绿（父本同构——回归护栏） |
| `test_contract_dtype_and_input` | 嵌套 list 输入 → f32 ndarray 输出；dtype=float32 | 契约（asarray f32 归一） | 绿（护栏） |

红→绿判据：前 3 个测试在父本（np.matmul kernel、无 main）必须失败且失败信息指向「缺 np_dot 派发/缺 main 协议」；实现后全部转绿，回归护栏始终绿。

## 4. 验证顺序（verify → benchmark → gate_check）

1. 单测：`env/.venv/bin/python test_solution.py`（红 → 实现 → 绿，全程同命令）；
2. 自检：`env/.venv/bin/python solution.py --n 64`（快速绿）→ `--n 8192`（预期 runtime ~430-490ms、max_error_vs_matmul 0.0）；
3. 正确性门（1 次）：`env/.venv/bin/python env/skills/verify.py --candidate solution.py`——预期 passed=true、max_error=0.0（[64,8192] 双 seed 双跑）；
4. 性能门（1 次）：`env/.venv/bin/python env/skills/benchmark.py --candidate solution.py`——预期 n=8192/k=3 中位 ∈ [430,490]ms、ref ∈ [380,700]ms、score ∈ [0.90,1.05]；
5. 融合判定（1 次）：`env/.venv/bin/python env/skills/gate_check.py --candidate solution.py --baseline 484.42954199999997`——软门禁 ±5% 带内或优于即成立。

**过程纪律：门禁工具每个至多跑一次，读数即记录，不复跑、不追噪声**（近 4 轮 no_candidate 全是 turn-budget 耗尽型过程失败）。

## 检索记录（role=coding，≤3 来源，落地为上述风险卡点 1/2/4）

- NumPy 官方文档 `numpy.dot`（v2.x Manual）：out= 严格契约（类型/C 连续/dtype 精确匹配，不满足抛异常）。
- numpy GitHub #7124：dot out= 非 C-array 报 ValueError 的实证（memmap 案例）。
- numpy GitHub #18669 / #23588：matmul out= 慢路径与 noblas 严苛检查 vs dot 入口——入口不对称依据。
