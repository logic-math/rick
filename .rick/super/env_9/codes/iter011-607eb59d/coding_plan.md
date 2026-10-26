# 编码执行计划 · 迭代 11 · dispatch_target=einsum_path|impl_strategy=m_quarters

## 0. 输入与事实基础

- 父本：`naive_start.py`（rowwise np.dot listcomp，@8192 实测 11516-11657ms，profile 热点
  `<listcomp>` 99.9%——本迭代已重跑 profile 确认）。
- 实验计划 idea（builder turn-budget-exceeded，plan 截断但方向明确）：手动复制 einsum 内部
  序列（tensordot 462.6 + identity copy 16.6 = 479ms）绕开 `np.einsum(..., out=dst,
  optimize=True)` 的自身调度开销（实测 832ms）。
- 本格定义：dispatch_target=einsum_path（einsum optimize 路径派发 BLAS）×
  impl_strategy=m_quarters（M 维四分：8192→4×2048 行块，余数并入末块）。

## 1. numpy 2.0.2 源码级依据（本次落地调研，来源=本机 venv numpy 源码 + 官方文档语义）

`np.einsum('ij,jk->ik', a, b, out=dst, optimize=True)` 的内部序列
（`numpy/_core/einsumfunc.py` L1418-1503，einsum 主执行循环）：

1. `einsum_path(..., einsum_call=True)` 生成收缩列表——2 操作数单步收缩，`blas=True`；
2. `handle_out = specified_out and (num+1)==len(contraction_list)` → True（带 out= 的单步收缩）；
3. `new_view = tensordot(a, b, axes=((1,), (0,)))`（BLAS 收缩，L1477）；
4. **`new_view = c_einsum('ik->ik', new_view, out=out)`（L1482-1491）——恒等收缩把结果拷入
   out，走的是 C einsum 通用迭代器的逐元素拷贝，不是 memcpy**；
5. 不带 out= 时（`tensor_result == results_index`）跳过第 4 步，直接返回 tensordot 结果。

`tensordot` 内部（`numpy/_core/numeric.py` L1108-1163）：2-D C 连续输入 + `axes=((1,),(0,))`
时 transpose/reshape 均为恒等视图（零拷贝）→ `np.dot(at, bt)` → BLAS sgemm——与 np.matmul
同一 K 求和序。

**本机实测（env_9 venv，8192²f32，预热1+K3 取中位）**：

| 形态 | runtime_ms | 说明 |
|---|---|---|
| np.matmul（锚） | 452.4 | |
| np.tensordot 整阵 | 500.0 | 噪声带内（iter006 debug 实测两者 med 均 ~442） |
| 手动 tensordot+切片拷 | 458.2 | builder「手动序列」复现 |
| einsum optimize=True（无 out） | 450.6 | 无恒等拷贝 → ≈锚 |
| **einsum optimize=True + out=** | **735.0** | c_einsum 恒等拷贝惩罚（builder 量到 832） |
| **m_quarters 原型（本计划）** | **475.5** | 落 BLAS 平价带（同族 np_dot m_halves 462.8 / np_matmul m_quarters 481.8） |

正确性：tensordot / m_quarters / einsum(opt) 与 np.matmul 锚 @8192 **逐位一致（max_error=0.0）**。

## 2. 改动点（父本结构 → 目标结构）

唯一改动 = `matmul(a, b)` 内核（`naive_start.py`）：

```
父本：a32/b32 归一 → np.stack([np.dot(a32[i], b32) for i in range(m)])   # 逐行 gemv + 拼装
目标：a32/b32 归一 → C 预分配 np.empty((m, n), f32)
      → M 四分：q = ceil(m/4)，4 个行块（min 钳位边界，空块跳过，余数并入末块）
      → 每块：C[lo:hi] = np.tensordot(a32[lo:hi], b32, axes=((1,), (0,)))
        ——即 einsum optimize 路径第 3+4 步的手动复制，恒等拷贝换成 memcpy 速度的切片赋值
```

- 逐块调用形态与 einsum 源码一字不差（`tensordot(*ops, axes=(tuple(left_pos), tuple(right_pos)))`
  = `axes=((1,), (0,))`），K 求和序不动（只动 M 维）。
- `main()` 自检协议保留（impl 标签改 `einsum_path_m_quarters`）。
- 不用 out= 传给 np.einsum（正是要绕开的慢路径）、不转置 B、不低精度、不动计时。

## 3. 风险卡点（附来源）

| # | 风险 | 依据与对策 |
|---|---|---|
| R1 | np.einsum(out=, optimize=True) 恒等拷贝惩罚（735-832ms） | einsumfunc.py L1482-1491：c_einsum('ik->ik',…,out=) 通用迭代器拷贝；对策=切片赋值走 memcpy（256MB ≈ 16.6ms @35.6GB/s，本机 profile env_facts） |
| R2 | tensordot 隐藏拷贝（非连续转置/reshape 触发 copy） | numeric.py L1140-1156：2-D C 连续 + axes=((1,),(0,)) 时全为恒等视图；实测整阵 tensordot 与 dot 同带 |
| R3 | K 求和序漂移 → 容差 1e-3 门死 | tensordot→np.dot→sgemm 与 np.matmul 同路径；实测 @8192 逐位一致 0.0；只动 M 维（domain_knowledge 合法族纪律） |
| R4 | 退化小形状（K<16 / m<4 / 余数行） | iter010 实测记录：K>=16 逐位，退化小形状 ≤1 ULP（2.4e-7）；min 钳位块边界保证 m∈{0,1,2,3} 安全 |
| R5 | 静态检查命中 | 不写 sys.modules / __main__. / time 赋值等模式（verify.py _STATIC_PATTERNS） |

## 4. 测试清单（TDD，`test_solution.py`）

**先红（区分改前/改后——父本 rowwise 无 tensordot 派发）**：
- `KernelDispatchTests.test_kernel_dispatches_four_tensordot_calls`：spy np.tensordot 计数
  恰好 4 次、每块入参 (q,K)×(K,N) 二维整块、K 求和轴 axes=((1,),(0,))——断言「M 四分 ×
  einsum 内部序列」的调用路径（假设主句）。父本=0 次 → 红。
- `KernelDispatchTests.test_result_written_to_preallocated_buffer`：4 次拷贝目标均为返回值
  C 的行切片（out 预分配 + 切片直写，非 np.stack 拼装）。父本 np.stack → 红。
- `MainProtocolTests.test_main_json_protocol`：`--n 64` JSON 输出 impl 标签
  `einsum_path_m_quarters` 且 max_error=0.0。父本标签 rowwise_np_dot → 红。

**回归护栏（父本已绿，实现后仍须绿）**：
- `KernelDispatchTests.test_kernel_bypasses_einsum_dispatch`：炸弹替换 np.einsum 后 matmul
  仍正确完成——钉住「绕开 einsum 自身调度、手动复制其内部序列」的假设。
- `KernelDispatchTests.test_kernel_independent_of_matmul`：炸弹替换 np.matmul 仍正确——
  派发入口是 einsum 路径内部序列（tensordot），不是 np.matmul。
- `BitwiseIdentityTests`：@512 双 seed 与锚逐位一致（array_equal）+ 双跑一致性（同输入两次
  逐位一致——verify.py ⑦ 门的预演）。
- `ShapeRobustnessTests`：m=129 奇数行 / 64×127 / 1 行 / 0 行 / 33×33 / 非方阵余数块
  （q=ceil → 块 17,17,17,16）；分级容差：K>=16 逐位，退化小形状 ≤ 2.4e-7（1 ULP，
  iter010 实测界；门容差 1e-3 远宽于此）。
- `ContractTests`：嵌套 list 输入归一 / 输出 f32 ndarray / f64 输入归一 f32。

## 5. 验证顺序（固定）

1. `verify.py --candidate solution.py`（双规模 [64,8192] + 双 seed + 双跑一致性 + 静态检查）
2. `benchmark.py --candidate solution.py`（8192 预热1+K3 中位 → 预期 ~460-490ms，score ~0.93-1.0）
3. `gate_check.py --candidate solution.py`（本格无精英 → baseline 空，软门禁 exempt；与框架终验同源）
