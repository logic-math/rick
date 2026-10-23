# matmul 领域知识

> 面向进化 agent 的领域速查：矩阵乘（CPU、float32、512×512）在 CPython/NumPy 语境下的
> 性能结构与调优路径。正确性锚点 = `code/reference.py`（`np.matmul`，底层 BLAS）。

## 任务定义

- 输入：固定 seed 生成的两个 `n×n` `float32` 矩阵（n=512 性能门 / n=64 正确性门）
- 输出：矩阵积 `c = a @ b`，形态 `(n, n)`，`max_error ≤ 1e-3`（env.json 容差）
- 评分方向：`runtime_ms` 越小越好（`perf_desc: "low"`）；基准 = `np.matmul` 同机计时，
  `score = ref_median / candidate_median`（加速比）

## 三要素（env 基石契约，agent 与框架共用同一份资产）

| 要素 | 语义（泛化） | 脚本 | agent 怎么用 | 框架怎么用 |
|---|---|---|---|---|
| **现状** | 事实信息获取 | `skills/profile.py` | 设计实验前剖析父本：热点在哪、环境事实、代码结构 | 不做门禁判定（事实非裁决） |
| **目标** | 目标优化的评估方法 | `skills/benchmark.py` | 拿 K 次中位耗时与 score，判断「变好了没有」 | stage3（经 gate_check） |
| **约束** | 优化过程中的约束 | `skills/verify.py` | 判断候选是否合法（正确性容差） | stage2 + stage1 参考实现锚定 |

**融合门禁**：`skills/gate_check.py` = verify → benchmark(K 次中位) → 软门禁（与
`--baseline` 按 `env.json` 的 `tolerance.soft_gate.relative_band` 比较）→ 一次给出
`{passed, failure_stage, metrics, details}`。**框架与 agent 共用它**——你在 coding/debug
里自行调用得到的结论与框架终验完全同源（不会出现「自检过、框架拒」）。

常用命令（venv 解释器）：

```bash
PY=.venv/bin/python
$PY skills/profile.py    --candidate <path>            # 现状：hotspots/env_facts/parent_facts
$PY skills/verify.py     --candidate <path>            # 约束：{passed, max_error}
$PY skills/benchmark.py  --candidate <path>            # 目标：{runtime_ms, score, timings}
$PY skills/gate_check.py --candidate <path> [--baseline <perf>]   # 融合门禁（权威）
```

（`data/bench.py`、`data/test_correctness.py` 保留为**薄转发**，仅兼容旧调用方；
三要素唯一实现位于 `skills/`。`self_check.py` 已退役——按需调用上表三个脚本即可。）

## 特征格实验指引（6 格精选空间，3 策略 × 2 布局）

> 格手术依据（env_7 实测 score 阶梯）：naive 三重循环 2.1e-5 → 逐行 np.dot **0.153**
> （LLM 一步可达的跳变，~7000×）→ 整阵 BLAS **0.98-1.00**（天花板族）。
> strassen / 多进程 / 线程池 / einsum 系在本机 512×512 实测**劣化**（死格族已裁除）；
> >1.0 需绕开单调用 SME 路径，本机无实测可达方案，不作默认目标。

### impl_strategy 轴（算法结构）

- **blocked_loops（分块循环）**：把 n×n 切成块（如 64×64），块内乘积装进 L1/L2。
  纯 Python 收益有限，但它是向量化/BLAS 路径的**前置结构**——分块边界预计算、
  行引用提升（`ai = a[i]`）、循环序调优在这里做。实测：blocked+tiled 比 naive 快 ~3.2×
  （demo 数据）。适合作为「结构性改进」的实验格。
- **rowwise_vectorized（逐行向量化）**：`c[i] = np.dot(ai, b)` 行向量×矩阵——
  score 0.153 的已验证跳板。注意 sgemv（np.dot）不触发 vecLib FP 噪声路径。
  本格的深挖方向：批量行处理（`np.dot(a[i:i+8], b)`）、避免逐行调度开销。
- **full_blas_dispatch（整阵 BLAS 派发）**：`np.matmul(a, b)` / `a @ b` / ctypes 直调
  cblas_sgemm——**score ~1.0 的天花板族**（唯一能出平价解的族）。深挖方向：
  out= 预分配、dtype 全程 float32、ctypes 走 Accelerate 的调用开销消减。
  ⚠️ 与参考全同的候选 score 在 0.977-1.002 波动（±3% 噪声）——软门禁带 5% 匹配。

### memory_layout 轴（数据排布）

- **standard（标准 C 序）**：NumPy 默认行主序直用，`a[i]` 连续。基线布局——
  实验方向是与算法结构的最优配对（哪些结构根本不需要数据预处理）。
- **fused_transpose（预转置融合）**：`bt = list(zip(*b))` 或 `np.ascontiguousarray(b.T)`——
  内积沿连续内存访问，通常 3-5× 提升（纯 Python 路径实测最大单变量收益）。
  转置成本一次性摊销在 n 次内积里。进阶形态是按访问序打包（连续段重排），
  与分块/BLAS 路径的数据准备天然衔接。

### 格间迁移建议（builder 上下文）

- 跨格重组最有价值的方向：blocked_loops 的**结构发现**（块尺寸、循环序）可迁移到
  full_blas_dispatch 的分块派发；fused_transpose 的**数据洞察**（连续性/打包）对所有策略普适。
- 同格深挖（full_blas_dispatch 族）是通往 score~1.0 的主路径——首访后值得连续投入。

## 性能分析 SOP

**先测量，后动手——没有 profile 的优化是猜测。**

1. **基线测量**：`skills/benchmark.py --candidate <你的实现>`——看 `timings` 三次的抖动
   （抖动大 = 环境噪声或首次缓存效应，先排除）
2. **阶段拆解**：纯 Python 实现里，三重循环的内层（`t` 维）占 >95% 时间——
   或用 `skills/profile.py --candidate <你的实现>` 直接拿 hotspots 表确认热点在内层循环
3. **微基准**：改一处，只测一处——`python3 -m timeit -s "from your_impl import matmul; ..."` 单变量对比
4. **形态检查**：确认 `matmul(a, b)` 返回的是紧凑结构（嵌套 list / ndarray 皆可，
   但 ndarray 才能吃到后续向量化收益）

## 性能调优 SOP

按「改动小→收益大」排序的路径（对应特征轴取值域）：

1. **布局先行（memory_layout）**：内积沿连续内存——先做转置视图（fused_transpose），
   纯 Python 路径通常 3-5×；2. **结构改进（blocked_loops）**：分块边界预计算、行引用提升、循环序（i-k-j）调优
3. **向量化跳变（rowwise_vectorized）**：`np.dot(ai, b)` 替代内层标量循环——~7000× 跳变
4. **BLAS 平价（full_blas_dispatch）**：`np.matmul` / ctypes 直调——score ~1.0 天花板；
   注意 dtype 全程 float32，中途升 float64 会慢且改变误差特性

**已实测劣化路径（不要重复探索）**：Strassen 递归（1.6× 慢）、多进程 fork/spawn/持久池
（score≤0.063）、线程池（0.81-0.94）、fp64 累加（3.4× 慢）——env_7 实测数据，
死格族已从特征空间裁除。

## 已知陷阱

- **作弊模式（门会拒）**：返回预存固定矩阵（不依赖输入）/ 篡改计时（线程内偷跑）/
  降低精度换速度（float16/截断）——正确性门与 exploit 注册表都会拒识
- **形状错误**：返回 `(n,1,n)` 广播结果或转置结果——`shape mismatch` 直接拒
- **首次运行抖动**：模块导入/BLAS 线程池初始化吃几十 ms——bench 已做预热 1 次，你自测也要预热
- **BLAS 线程数**：`np.matmul` 在多核 BLAS 上自动并行；候选若手工设
  `OMP_NUM_THREADS=1` 之类的环境变量来「稳定计时」，不要在代码里改全局状态（污染门禁环境）
- **score=1.0 非安全线**：与参考全同的候选有 ±3% 噪声波动（软门禁带 5% 兜底）；
  >1.0 在本机 512×512 CPython+numpy 约束下无实测可达路径
