# matmul 领域知识（2048×2048）

> 面向进化 agent 的领域速查：矩阵乘（CPU、float32、**2048×2048**）在 CPython/NumPy 语境下的
> 性能结构与调优路径。正确性锚点 = `code/reference.py`（`np.matmul`，底层 Accelerate BLAS）。

## 任务定义

- 输入：固定 seed 生成的两个 `n×n` `float32` 矩阵（n=2048 性能门 / n∈{64,2048} 正确性门）
- 输出：矩阵积 `c = a @ b`，形态 `(n, n)`，`max_error ≤ 1e-3`（env.json 容差，vs np.matmul f32 锚）
- 评分方向：`runtime_ms` 越小越好；基准 = `np.matmul` 同机计时（@2048 ≈ 6.9ms），
  `score = ref_median / candidate_median`（加速比；score 1.0 = BLAS 平价）

## 任务级数值事实（2048 规模，必须知道——决定哪些实现是「正确」的）

- @2048 uniform 输入下 C 元素量级 ~561（mean 512）；f32 累加的舍入误差量级 ~1.7e-3——**f32 BLAS 锚自身离 fp64 真值就有 1.73e-3**。
- **1e-3 容差 @2048 是紧约束：正确性 = 「与 np.matmul 的 BLAS 求和序一致」**：
  - ✅ 保持求和序的实现逐位一致（max_error = 0.0）：**M 维分解**（行/行块/上下半分治，
    每块各自整阵乘再拼接）、整阵直调（matmul/dot/einsum optimize=True）
  - ❌ 改变 K 求和序的实现误差 1.6-2.9e-3 **会被拒**：K 维分块累加、2×2 象限和、
    Strassen、**B 预转置后 gemv**（转置就换序）
- 推论：优化只能在「怎么分组输出行」与「走哪个库入口」上做文章，**不能动 K 维的累加结构**。

## 性能分析 SOP

**先测量，后动手——没有 profile 的优化是猜测。**

0. **热点剖析（首选工具）**：`skills/perf_profile.py --candidate <你的实现>`
    ——单行 JSON 输出：候选/参考耗时比、`subcalls`（子调用次数）、top-N 热点函数与占比。
    读法：`subcalls` 大 + 热点集中在 Python 循环/包装层 → 优化方向是「减少子调用次数」
    （见下方阶梯）；热点已在单次 BLAS 调用内 → 已到天花板附近，别再折腾。
    示例（本机实测 @2048）：rowwise 起点 subcalls=4111、99.2% 时间在逐行循环；
    整阵派发解 subcalls=4、100% 在单次 matmul。
1. **基线测量**：`skills/self_check.py --candidate <你的实现>`（快速反馈）或
   `data/bench.py --candidate <你的实现>`（终验口径）——看 `timings` 三次的抖动
2. **阶段拆解**：perf_profile 的 top 热点就是阶段拆解——若 listcomp/循环占比 >90%，
   不必再跑 cProfile 手工分析
3. **微基准**：改一处只测一处——`python3 -m timeit` 单变量对比

## 性能调优 SOP（按「改动小→收益大」排序，@2048 实测阶梯）

| 路径 | 实现 | @2048 实测 | score |
|---|---|---|---|
| 起点 | rowwise np.dot（无转置） | ~143ms | 0.048 |
| M 维二分 | 上下半各一次整阵乘再拼 | ~7.6ms | 0.90 |
| M 维四分 | 行四块各乘再拼 | ~7-7.6ms | ~0.90 |
| einsum 优化路径 | np.einsum(..., optimize=True) | ~7.9ms | 0.87 |
| 整阵 BLAS | np.matmul / np.dot / a @ b | ~6.9ms | ~1.0（天花板） |

关键洞察：**@2048 从 rowwise 到 BLAS 平价的本质 = 减少 Python 循环调度次数**——
rowwise 是 2048 次子调用，M 二分是 2 次，整阵是 1 次。每次子调用走 BLAS 内核的
绝对性能几乎不变（SME 协处理器单线程即饱和），变化的是 Python/NumPy 调度开销。

## 已知陷阱（@2048 实测，别浪费时间）

- **B 预转置后逐行 gemv**：换 K 求和序 → 1.71e-3 超容差被拒（且并不更快：188ms）
- **K 维分块累加**（A[:,k0:k1] @ B[k0:k1,:] 循环累加）：换求和序被拒（1.6-1.9e-3）；
  即便放宽也只 0.28-0.65（10.6-24.7ms），不通向更好性能
- **Strassen 分治**：换求和序被拒（2.87e-3）；性能也劣（11.3ms，0.61）
- **多进程**：fork 纯开销 ~6.8ms ≈ 整个 matmul；端到端 ≤0.65，无收益
- **多线程分片**：GCD 防超订导致 ≤0.92（7.5-8.4ms），无收益
- **float64 中途升精度**：慢 4×（27ms）且误差 1.73e-3 被拒——全程保持 float32
- **float16/量化存储**：量化误差远超容差，直接拒
- **ctypes 直调 cblas_sgemm**：@2048 ~10ms（0.69）劣于 np.matmul——不碰
- **作弊模式**（返回预存/篡改计时/降精度）：正确性门与 exploit 注册表都会拒识

## 未覆盖方向（adjust 决策素材——2026-10-21 12 迭代后补，供 expand_cell / 权重提案参考）

12 格已建 11（np_dot|rowwise_vectorized 未采）。当前梯形：rowwise 0.042-0.048 →
m_quarters 0.75-0.82 → m_halves 0.86-0.96 → full_dispatch 0.96-0.99。候选新方向
（按潜力排序，均有研究实测依据，未见任何格承载）：

1. **threaded 行分片**：ThreadPoolExecutor 2/4 线程按 M 维分片各自 np.matmul
   ——研究实测 7.5-8.0ms（score 0.86-0.92，**合法**：M 分解不改变 K 求和序，逐位一致）。
   当前特征空间无轴值承载线程形态（parallelism 未入轴）——若值得探索需 expand_cell。
   ⚠️ 已知风险：vecLib GCD 防超订在并发调用下可能劣化（实测 ≤0.92 未超单调用），
   但与 out= 预分配组合或负载态变化下可能不同。
2. **out= 预分配**：np.matmul(a, b, out=预分配 C)——env_7 实测 1.018（噪声内持平）；
   与 m_halves 组合（每块写各自 out 切片）未实测，理论上省 concatenate 拷贝。
3. **m_eighths**（16 块行分组）：粒度谱系 1/2/4 已有，1/8 未测——谱系两端（full=1 次
   vs eighths=8 次）之间的调度开销曲线未探明，但预期介于 quarters 与 full 之间，
   信息价值低。
4. **铺空格**：np_dot|rowwise_vectorized（visit=0）——naive_start 原型即此形态
   （~140ms 已知），预期 score ~0.049，仅完形价值。

死族（研究实测，勿重新提案）：mp 多进程（fork 开销 6.8ms）、blocked-K（容差拒）、
Strassen（容差拒+慢）、ctypes 直调（0.69）、fp16/fp64（容差拒）。

## 优化路径速查（一句话）

从 rowwise 起步的正确方向只有一个族：**把「多少次子调用」从 2048 降到 1**——
M 维分组粒度递减（halves → quarters → full），派发入口在 np_dot/np_matmul/einsum(optimize=True)
之间选择。天花板 = 整阵单次 BLAS 调用（score ~0.98-1.0）。
