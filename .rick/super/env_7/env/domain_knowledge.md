# matmul 领域知识

> 面向进化 agent 的领域速查：矩阵乘（CPU、float32、512×512）在 CPython/NumPy 语境下的
> 性能结构与调优路径。正确性锚点 = `code/reference.py`（`np.matmul`，底层 BLAS）。

## 任务定义

- 输入：固定 seed 生成的两个 `n×n` `float32` 矩阵（n=512 性能门 / n=64 正确性门）
- 输出：矩阵积 `c = a @ b`，形态 `(n, n)`，`max_error ≤ 1e-3`（env.json 容差）
- 评分方向：`runtime_ms` 越小越好（`perf_desc: "low"`）；基准 = `np.matmul` 同机计时，
  `score = ref_median / candidate_median`（加速比）

## 性能分析 SOP

**先测量，后动手——没有 profile 的优化是猜测。**

1. **基线测量**：`data/bench.py --candidate <你的实现>`——看 `timings` 三次的抖动
   （抖动大 = 环境噪声或首次缓存效应，先排除）
2. **阶段拆解**：纯 Python 实现里，三重循环的内层（`t` 维）占 >95% 时间——
   `python3 -m cProfile -s cumtime your_impl.py` 确认热点就在内层循环
3. **微基准**：改一处，只测一处——`python3 -m timeit -s "from your_impl import matmul; ..."` 单变量对比
4. **形态检查**：确认 `matmul(a, b)` 返回的是紧凑结构（嵌套 list / ndarray 皆可，
   但 ndarray 才能吃到后续向量化收益）

## 性能调优 SOP

按「改动小→收益大」排序的路径（对应特征轴 `impl_strategy` 的取值域）：

1. **循环结构（naive_loops → blocked）**
   - 先做转置视图：`bt = list(zip(*b))`，内层沿连续内存访问——通常 3-5× 提升
   - 分块（cache-blocking）：把 n×n 切成块（如 64×64），块内乘积装进 L1/L2——
     纯 Python 收益有限，但对后续向量化是前置结构
2. **向量化（vectorized）**
   - NumPy 整行/整块运算替代标量循环：`c[i] = np.dot(ai, b)`（行向量×矩阵）——通常 10-100×
   - 全矩阵 `np.matmul(a, b)` / `a @ b`：直接命中 BLAS（参考实现即此，是「天花板」而非「作弊」）
   - 注意 dtype：`float32` 输入全程保持 `float32`，中途升 `float64` 会慢且改变误差特性
3. **内存布局（memory_layout 轴）**
   - 行主序（row_major）：NumPy 默认 `C` 序，`a[i]` 连续——按行遍历友好
   - 列主序（col_major）：`Fortran` 序，对「按列累加」的写法友好；`np.asfortranarray` 转换有一次性成本
   - 分块平铺（tiled）：自管块状布局，配 blocked 策略
4. **Strassen 类（strassen_like）**
   - 递归分治：`2×2` 分块 + 7 次子矩阵乘（省 1/8 乘法量）——**边界**：递归/额外加减
   - 的常数开销在 n=512、且底层已是 BLAS 时通常得不偿失；只在对纯 Python 分块实现
   做结构优化时有意义。先测再上。

## 已知陷阱

- **作弊模式（门会拒）**：返回预存固定矩阵（不依赖输入）/ 篡改计时（线程内偷跑）/ 
  降低精度换速度（float16/截断）——正确性门与 exploit 注册表都会拒识
- **形状错误**：返回 `(n,1,n)` 广播结果或转置结果——`shape mismatch` 直接拒
- **首次运行抖动**：模块导入/BLAS 线程池初始化吃几十 ms——bench 已做预热 1 次，你自测也要预热
- **BLAS 线程数**：`np.matmul` 在多核 BLAS 上自动并行；候选若手工设
  `OMP_NUM_THREADS=1` 之类的环境变量来「稳定计时」，不要在代码里改全局状态（污染门禁环境）

## 优化路径速查（env_7 校准数据，实测于本机）

score = ref_median / candidate_median（相对 np.matmul 基线的加速比；本机基线 ~0.19ms）。
本机实测的 score 阶梯与死路（帮你少走弯路，直接选有收益的方向）：

| 路径 | 实测 score | 判断 |
|---|---|---|
| naive 三重循环（起点） | ~2e-5 | 起点，别停留 |
| 转置视图优化（zip(*b)） | ~4e-5 | 微弱（本机快核掩盖 cache 效应） |
| 逐行向量化 `c[i]=np.dot(a[i],b)` | ~0.15 | **一步跳变 ~7000×**——最优先方向 |
| 整阵 `np.matmul(a,b)` / `a@b` | ~1.0 | 天花板（BLAS，本机 ~1450 GFLOPS） |
| ctypes 直调 Accelerate cblas_sgemm | ~1.0 | 与 np.matmul 同内核位相同，无增益但可行 |
| 线程池并行分块 | 0.81-0.94 | 严格劣化（GIL 外 BLAS 内部已并行） |
| multiprocessing 分块 | ≤0.06 | 严重劣化（IPC/建池开销主导） |
| Strassen 一层递归 | ~0.6 | 劣化（逐元素加减内存带宽吃光乘法节省） |
| fp64 计算 | ~0.29 | 劣化（sgemm 3.4× 快于 dgemm） |
| einsum | ~0.02 | 无 BLAS 派发路径，慢 ~45× |

要点：
- **单次 np.matmul 已吃满本机 BLAS 吞吐**（SME 协处理器多簇派发）——别在外层并行上浪费迭代。
- 混合形态：blocked 分块 + 块内 np.matmul 是合法且可能接近 1.0 的路径（分块不损 BLAS 效率）。
- 中途精度变化（如 fp16 存储）在容差 1e-3 下大概率不过正确性门，别赌。
