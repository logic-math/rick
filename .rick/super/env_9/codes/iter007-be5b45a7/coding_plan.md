# 编码执行计划 · 迭代 7 · dispatch_target=np_dot|impl_strategy=full_blas_dispatch

## 0. 前置事实确认（profile 实测，@8192，本机）

`profile.py --candidate naive_start.py`（本格父本）：
- runtime **11611.8ms**；热点 `<listcomp>` self **11496.3ms（99.9%）**、`stack` 9.6ms、`dot` 0.62ms——与 ctx 注入读数一致。
- env 事实：python 3.9.6 / numpy 2.0.2 / **BLAS=accelerate（vecLib）** / 18 核 / memcpy 34.3GB/s。
- 结论：瓶颈全在 Python 层 8192 次逐行循环 + B 全量重读 + stack 装配；BLAS 内核只值 ~440ms。改动点确认为「kernel 入口」。

## 1. 改动点（父本结构 → 目标结构，逐处说明）

工作目录 `solution.py` 当前为 iter003 模板（`full_blas_dispatch_np_matmul`，契约已照抄父本）。本迭代**唯一实质改动 = kernel 入口换成 np.dot**：

| # | 位置 | 现状（iter003 模板） | 目标（本格定义形态） |
|---|---|---|---|
| 1 | `matmul()` 函数体 | `return np.matmul(a32, b32)` | `return np.dot(a32, b32)`（单次整阵 2-D BLAS gemm 直派） |
| 2 | 模块 docstring / `matmul` docstring | 描述 np.matmul 派发 | 描述 np.dot 派发（本格 cell：dispatch_target=np_dot） |
| 3 | main JSON `impl` 标签 | `full_blas_dispatch_np_matmul` | `full_blas_dispatch_np_dot` |

**保持不动（契约）**：`matmul(a,b)` 接口、接受嵌套 list/array、`asarray→f32`、`np.seterr` 三项抑制、main 的 warmup+计时+自检 JSON 协议、不加 `out=`、不预转置 B、不切 K/N、无中间拷贝。

## 2. 风险卡点（API 限制 / 边界条件 / 性能陷阱，附来源）

- **`np.dot` 的 `out=` 别名坑**：`out` 与输入同阵会静默返回全零（numpy/numpy#8440）——本实现**不使用 out=**，直接返回新分配数组，天然规避。
- **dot 的 1-D gemv / 2-D gemm 分发差异**（numpy#13732）：gemv 与 gemm 走不同 BLAS 入口且求和序可能不同——本实现**只做 2-D×2-D 整阵调用**（`np.dot(a32, b32)` 两参均为 2-D），与锚同走 cblas_sgemm，K 求和序不变（预检 array_equal=True、maxdiff 0.0）。
- **文档口径**：numpy.dot 官方文档明示「2-D×2-D = matrix multiplication」且 dispatch 到 BLAS（numpy.org/doc/stable numpy.dot）；「matmul 更被推荐」仅是风格口径，非正确性/性能约束——本格定义形态要求 np.dot 入口，属合法族。
- **内存布局**：`np.asarray(x, dtype=f32)` 产出 C 连续阵；matmul.c.src `is_blasable2d` 与 cblasfuncs 的直派条件（第二轴连续）自动满足——无布局陷阱。
- **风险极低**：改动是把已验证等价的入口从 np.matmul 换成 np.dot；若 benchmark 显著偏离 [434,488]ms 带或 max_error ≠ 0，即假设被否定，如实上报。

## 3. 测试清单（test_solution.py；每个断言对应假设哪部分）

| 测试 | 断言 | 对应假设 | 预期（改前/改后） |
|---|---|---|---|
| T1 契约：嵌套 list 输入 | matmul 接受嵌套 list，返回 f32 ndarray、形状 (n,m) | 契约保持 | 绿/绿 |
| T2 契约：dtype 归一 | f64/int 输入被 asarray→f32（结果 dtype float32） | 契约保持 | 绿/绿 |
| T3 数值：逐位一致锚 | n∈{64,129,256}×seed{42,7}：max abs diff vs np.matmul == 0.0 | max_error=0.0（支持线） | 绿/绿 |
| T4 非方阵形状 | (7,5)@(5,11) 形状正确且逐位一致 | kernel 通用性 | 绿/绿 |
| T5 同输入双跑一致 | 两次调用 array_equal=True | verify ⑦ 双跑一致 | 绿/绿 |
| **T6 调用路径：np.dot 入口** | spy 记录：kernel 恰好 1 次 `np.dot`（2-D f32 参数）、0 次 `np.matmul` | 本格定义形态（dispatch_target=np_dot） | **红→绿** |
| **T7 main 协议：impl 标签** | `--n 64` 子进程 JSON：impl=="full_blas_dispatch_np_dot"、max_error_vs_matmul==0.0、runtime_ms>0 | 自检协议 + 入口标签 | **红→绿** |

T6/T7 是「能区分改前/改后」的判别测试（当前模板走 np.matmul 入口，必先红）；T1-T5 守契约防回归。

## 4. 验证顺序（verify → benchmark → gate_check）

1. 自检：`solution.py --n 64`（快速绿）→ `--n 8192`（预期 runtime ~434-488ms、max_error 0.0，对照父本 11612ms ≈ 26× 塌缩）。
2. `env/skills/verify.py --candidate solution.py`：预期 passed=true、max_error=0.0（[64,8192]×双 seed×双跑）。
3. `env/skills/benchmark.py --candidate solution.py`：预期 @8192 k=3 中位 ∈ [434,488]ms、ref ∈ [380,700]ms、score ∈ [0.90,1.05]、无拒收。
4. `env/skills/gate_check.py --candidate solution.py --baseline 430.8255829999998`（邻格精英对照；±5% 带内 = 入口等价的轴级证据）。
5. （佐证）`profile.py --candidate solution.py`：确认 `<listcomp>`/`stack` 热点消失、kernel 只剩单次 BLAS 调用。

## 5. 检索记录（role=coding，≤3 来源，落在实现细节）

- numpy.dot 官方文档（numpy.org/doc/stable/reference/generated/numpy.dot.html）：2-D×2-D 即矩阵乘、dispatch BLAS；`out=` 为性能特性且不查别名 → 不用。
- numpy#13732（github.com/numpy/numpy/issues/13732）：dot 派发 gemv/gemm 分层，2-D 路径与 matmul 同 gemm 内核。
- numpy#8440：np.dot `out=` 与输入同阵静默全零 → 本实现直接返回新数组。

## 6. TDD 执行顺序（五步：先测试→看红→最小实现→绿→重构）

1. 写 `test_solution.py`（上表 T1-T7）。
2. 跑测试 → **必须亲眼看 T6/T7 红**（当前 kernel 走 np.matmul）。
3. 最小实现：kernel 单行改 `return np.dot(a32, b32)` + impl 标签/docstring 同步。
4. 跑测试 → 全绿；重构仅限注释/docstring 清理（行为零改动）。
5. 进入第 4 节验证序列。
