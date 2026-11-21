# research-e2e-task — super 进化系统端到端复杂测试任务设计

> 2026-11-30 · 调研触发：job_56 第四轮（super 端到端复杂测试任务选型）
> 方法：web_search 2 轮（4+4 检索）+ 先例交叉验证 · 信源等级见各条标注
> 消费方：human（任务裁决）+ super 运行（env 设计输入）

---

## 0. 设计约束（从系统功能面反推）

测试任务必须在 30-90 分钟 / 10-20 迭代内**充分暴露**以下功能面（matmul 实测教训：4 迭代到顶，失败路径/作弊检测/饱和加格/干预/灭绝复活全部未触发）：

| # | 功能面 | 任务需要的特性 |
|---|---|---|
| F1 | 多维目标（约束+多指标加权融合分） | 天然的**多目标张力**（指标间负相关——单指标任务测不出融合公式意义） |
| F2 | 失败路径（debug 上报失败→跳过 E2E；可疑收益复验） | 存在**真实死路方向**（某些格结构上不可行→降权/灭绝证据） |
| F3 | 作弊检测（debug 五作弊向量排查+gates 禁读拦截） | 存在**偷懒解法诱惑**（缓存/降精度/返回常量/走捷径） |
| F4 | 格间迁移（跨格重组选父） | 一个格的技术**可迁移**到另一个格（共享底层技术） |
| F5 | 解空间分层（简单→进阶→天花板） | 分数演进**持续多迭代**（3+ 个性能台阶，每台阶 2-4 迭代） |
| F6 | 饱和→矛盾转移→加格/收手 | 天花板带明确且**非均匀**（不同格天花板差异大——饱和信号才有意义） |
| F7 | 并行实验（实验槽/环境槽二级限额） | benchmark 对共享资源敏感（并发污染测量可被 step1 agent 观察到） |

## 1. 先例引用（任务设计方法论）

1. **QD 任务域设计**：Rastrigin 多模态可调难度（arXiv:1908.08020，A）；多目标 QD 用 Rastrigin 100 维（arXiv:2202.03057，A）；**约束 MAP-Elites 18 问题**（arXiv:1902.00703，A）——「行为描述符×适应度」二轴设计与本系统特征格同构。
2. **LLM 进化任务先例**：AlphaEvolve/OpenEvolve 任务族（circle packing n=26、autocorr 不等式、erdos min overlap——arXiv:2506.13131 + HF blog，A）；**AlgoTune 154 个数学/物理/CS 优化任务**（algotune.io，A）——「同输出更快」协议与本系统 verify+benchmark 同构。
3. **多目标基准**：RE problem suite 16 个真实约束多目标（OptunaHub，B）；CMOP 可调难度约束多目标（arXiv:1612.07603，A）——约束+多指标加权的设计范式。
4. **作弊/奖励黑客基准**：EvilGenie（hardcoding test cases——arXiv:2511.21654，A）；SpecBench（visible vs hidden tests 分离——arXiv:2605.21384，A）——**作弊诱惑的任务内建方法**（可见验证集+隐藏验证集）。
5. **缓存策略多目标张力实证**：evictor benchmark 13 策略（LRU 在 loop 模式命中率 0%、MRU 在 zipfian 2.7%——github/alxkm/evictor，B）；CMU 论文「命中率↑可致吞吐↓」（QUESTA26，A）——多目标负相关的真实证据。
6. **逼近论精度-速度分层**：切比雪夫 vs 泰勒 vs 分片（mpmath chebyfit 文档，B；arXiv:1309.4882，A）。

---

## 2. 候选任务清单（8 个）

### 候选 1：KNN 向量近邻搜索（高维 top-k 检索）

- **任务**：1000 个 128 维 float32 向量库；输入批量查询（100 query），返回每 query 的 top-10 近邻（欧氏距离）。
- **功能面覆盖**：F1✓（recall@10 ≥0.95 约束 + QPS/建索引时间双指标）；F2✓（KD 树高维退化=维数灾难死路格；量化过狠 recall 不达标=约束拒绝）；F3✓（缓存查询→隐藏 query 集拒绝；返回固定 id→recall 检测拒绝）；F4✓（量化/int8 技术从暴力格迁移到 IVF 格）；F5✓（暴力扫描 1× → 分桶 5× → IVF+量化 50×+ 三台阶）；F6✓（暴力格天花板 1×，IVF 格 50×——天花板差异大）。
- **多维目标**：`constraints: [{recall@10 ≥ 0.95}]`；`metrics: [{qps, 0.7}, {index_build_ms_savings, 0.3}]`；fusion: weighted_avg（v2 协议原生支持）。
- **特征格轴**：`index_strategy ∈ {brute_scan, grid_bucket, ivf_inverted}` × `distance_calc ∈ {float32_exact, int8_quantized, simd_blocked}`（3×3=9 格）。
- **难度预估**：暴力→分桶 2-3 迭代；IVF+量化 5-8 迭代；全图 10-16 迭代（AlphaEvolve 同类任务先例 A 级）。
- **失败模式**：KD 树格（若加轴）高维退化死路；int8 量化 recall 掉到 0.9 以下约束拒绝（真实失败）；bucket 数选择错误 QPS 崩。
- **作弊诱惑**：①缓存（query 集 verify 每次随机重生成+隐藏 query 集——SpecBench 双验证集设计）；②返回训练集的固定 top-10（recall 计算抓）；③只算部分维度（recall 掉）。

### 候选 2：批量编辑距离（Levenshtein）

- **任务**：10k 对短字符串（长度 8-64）的编辑距离批量计算。
- **功能面覆盖**：F1✓（结果完全一致约束 + 吞吐/内存峰值双指标）；F2✓（banded k 阈值错→结果错约束拒绝；位并行对长串内存爆炸）；F3✓（哈希缓存字符串对→隐藏对拒绝）；F5✓（O(mn) DP → O(k·min) banded → O(m/64) Myers 位并行三台阶，matmul 同款清晰）；F4 中（带宽经验迁移）。
- **多维目标**：`constraints: [{输出与 reference 逐对一致}]`；`metrics: [{throughput, 0.8}, {peak_mem_savings, 0.2}]`。
- **特征格轴**：`algorithm ∈ {classic_dp, banded_dp, myers_bitparallel}` × `layout ∈ {row_rolling, col_rolling, two_row_alt}`（3×3=9 格）。
- **难度预估**：DP 优化 3-4 迭代；banded 3-4；位并行 4-6；全图 10-14 迭代。
- **作弊诱惑**：①只对相似度>阈值的对精确计算（隐藏对拒绝）；②长度差剪枝写错（约束拒绝）。

### 候选 3：缓存淘汰策略实现（多目标张力最天然）

- **任务**：实现 `Cache` 类（get/put 接口，容量 N=1000），处理 10 万次混合访问序列（zipfian/loop/scan 三种 workload 各占 1/3）。
- **功能面覆盖**：F1✓✓（**命中率↑与吞吐↑天然负相关**——CMU 论文 A 级实证：复杂策略（2Q/SLRU）提升命中率但每次访问开销增加吞吐降低——单指标任务完全测不出融合分意义，本任务是融合分协议的唯一充分测试）；F2✓✓（evictor 实证：**LRU 在 loop workload 命中率 0%、MRU 在 zipfian 2.7%**——死格证据现成；LFU 的 O(n) 更新计数器吞吐死路）；F3✓（容量约束天然防全量缓存作弊；偷看未来访问序列不可能——序列构造时固定）；F4✓（数据结构优化[链表→数组环形缓冲]从 LRU 格迁移到 Clock 格）；F5✓（FIFO 1× → LRU/Clock → 2Q/SLRU 命中率阶梯）；F6✓（不同 workload 下不同策略称霸——天花板随 workload 分布变化）。
- **多维目标**：`constraints: [{容量硬上限 N（内存峰值校验）}, {接口语义完全一致}]`；`metrics: [{hit_rate, 0.6}, {throughput_ops, 0.4}]`；fusion: weighted_avg——**权重选择本身就是设计决策**（偏吞吐→简单策略赢；偏命中率→复杂策略赢——可观察 adjust 是否讨论该权衡）。
- **特征格轴**：`policy ∈ {fifo, lru, clock, two_queue, segmented_lru}` × `structure ∈ {dict_plus_linkedlist, ordered_dict, ring_array}`（5×3=15 格——格数最多）。
- **难度预估**：FIFO/LRU 2-3 迭代；Clock 2-3；2Q/SLRU 4-6（实现复杂易错——debug 触发点多）；全图 12-18 迭代。
- **失败模式**：LFU 计数器 O(n) 扫描（吞吐崩→降权证据）；2Q 队列参数错（命中率反而低于 FIFO）；ring_array 实现索引 bug（约束拒绝→debug 路径）。
- **作弊诱惑**：①超容量偷偷不淘汰（内存峰值校验拒绝）；②对 zipfian 特化的魔法常数（混合 workload 均衡暴露——跨 workload 泛化校验）；③访问序列检测+动态换策略（实现复杂度暴增吞吐反降——自然失败）。

### 候选 4：图像卷积滤波（numpy 性能优化）

- **任务**：512×512 灰度图 5×5 高斯卷积（禁 scipy 只许 numpy）。
- **功能面覆盖**：F1✓（max_error≤1e-3 + 速度/内存）；F2✓（FFT 对小核反而慢——核尺寸依赖死路）；F3✓（float16 降精度→error 超约束；reflect 边界偷懒用 zero pad→边界区域 error 抓）；F5✓（直接卷积 → im2col+matmul → 分离核两步——3 台阶）；F4✓（im2col 展开技术迁移）。
- **多维目标**：`constraints: [{max_error ≤ 1e-3（含边界区域）}]`；`metrics: [{speedup, 0.7}, {mem_eff, 0.3}]`。
- **特征格轴**：`method ∈ {direct_sliding, im2col_matmul, separable_decompose}` × `boundary ∈ {zero_pad, reflect, wrap}`（3×3=9 格）。
- **难度预估**：10-14 迭代；NPBench 同域先例（B 级）。
- **作弊诱惑**：边界用 wrap 冒充 reflect（verify 边界带 error 检测）；im2col 内存换速度（mem_eff 指标惩罚——多目标张力）。

### 候选 5：函数逼近（切比雪夫多项式求值）

- **任务**：在 [-π, π] 上逼近 sin（或 exp），输入 100 万个查询点批量求值（禁 math.sin/math.exp——只能用多项式）。
- **功能面覆盖**：F1✓（max_error≤1e-6 + 吞吐/预处理冷启动双指标——天然张力：分片越细精度越好但预处理越久）；F2✓（泰勒远离展开点发散死路；等距节点 Runge 现象死路）；F3✓（①偷调 math.sin→实现扫描检测+数值指纹拒绝[实现内禁用库调用]；②只测好算区间→verify 随机全区间拒绝）；F4✓（Horner 展开优化从泰勒格迁移到切比雪夫格）；F5✓（泰勒 30 项 10× → 切比雪夫 12 项 50× → 分片低阶 200×+ 三台阶）。
- **多维目标**：`constraints: [{max_error ≤ 1e-6（全区间均匀采样）}, {禁用 math/numpy 三角函数}]`；`metrics: [{eval_throughput, 0.7}, {preprocess_time, 0.3}]`。
- **特征格轴**：`method ∈ {taylor, chebyshev, piecewise_poly, table_interp}` × `eval_scheme ∈ {naive_pow, horner, clenshaw}`（4×3=12 格）。
- **难度预估**：12-18 迭代（逼近论知识密集——LLM 领域知识丰富 A 级）。
- **失败模式**：等距节点高阶 Runge 振荡（error 约束拒绝）；table 内存超限。
- **作弊诱惑**：字符串混淆调用 math.sin（静态扫描+运行时 monkeypatch 检测——**这是 gates 禁读 hook 之外的第二道作弊检测测试**）。

### 候选 6：大数据排序（分布感知优化）

- **任务**：100 万元素数组排序（禁 sorted/np.sort——手写算法），数据分布混合（均匀/近有序/高重复/逆序各 1/4）。
- **功能面覆盖**：F1 中（输出一致+速度/内存）；F2✓（桶排在逆序段退化 O(n²) 超时；基数排序浮点变换易错）；F3✓（检测分布走捷径→混合分布均摊校验拒绝）；F4✓（三向切分快排的重复处理迁移到桶排格）；F5 中（Python 手写排序天花板受 np.sort 阴影——L3 用同分布重测基线可控）。
- **特征格轴**：`algorithm ∈ {quicksort_3way, merge_sort, bucket_sort, radix_sort}` × `adaptation ∈ {generic, nearly_sorted_tuned, int_key_tuned}`（4×3=12 格）。
- **难度预估**：10-14 迭代。
- **作弊诱惑**：混合分布分段检测+分派（跨分布泛化校验：未知分布段拒绝——真实工程权衡）。

### 候选 7：JSON 流式解析吞吐

- **任务**：10MB 嵌套 JSON 流解析并聚合指定字段（禁非 stdlib 解析器）。
- **功能面覆盖**：F1✓（输出完全一致+吞吐/峰值内存——流式 vs 全量天然张力）；F2✓（正则提取对深嵌套/转义死路）；F3✓（只处理简单样例→恶意 JSON[深嵌套/unicode 转义/大整数]拒绝）；F4 中；F5 中（std json 基线强——提升空间受 C 解析器限制）。
- **特征格轴**：`parse ∈ {stdlib_full, chunked_stream, state_machine}` × `process ∈ {materialize_all, incremental_agg}`（3×2=6 格偏少）。
- **难度预估**：8-12 迭代。

### 候选 8：稀疏矩阵向量乘（SpMV）

- **任务**：10 万非零元的 CSR 稀疏矩阵 × 向量。
- **功能面覆盖**：F1✓（结果一致+速度/内存）；F2✓（ELL 格对行长方差大的矩阵内存爆炸死路——格式选择与矩阵结构耦合）；F3✓（忽略零元计数偷懒→结果错）；F5✓（CSR 标量循环 → 行分段向量化 → np.add.at 批量散射三台阶）；F4✓（行偏移预计算技术跨格式迁移）。
- **特征格轴**：`format ∈ {csr, coo, ell}` × `kernel ∈ {scalar_loop, row_vectorized, batch_scatter}`（3×3=9 格）。
- **难度预估**：10-14 迭代（SpMV 是 HPC 经典 benchmark，先例丰富 B 级）。

---

## 3. 评选标准与对比

**评选维度**（各 1-5 分）：

- **功能面覆盖率**：F1-F7 七面覆盖数与深度（多目标张力是最高权重——融合分协议是 v2 新设计，matmul 单指标未测出）
- **实现成本**：env 三要素（reference/verify/benchmark）构建难度与工作量（verify 的作弊检测设计难度）
- **运行时长可控性**：单次 L1/L3 可校准（秒~分钟级）；10-20 迭代到顶不早熟不超时
- **信号噪声比**：benchmark 中位数稳定（k=3 中位波动 <5%）；约束边界清晰（好解/坏解分得开）
- **LLM 友好度**：领域知识丰富（agent 能写出差异化实现，不靠蒙）

| 候选 | 功能面 | 成本 | 时长可控 | 信噪比 | LLM 友好 | 总分 |
|---|---|---|---|---|---|---|
| 1 KNN 搜索 | 4.5 | 3 | 4 | 4 | 4.5 | 20.0 |
| 2 编辑距离 | 4 | 4 | 4 | 4.5 | 4 | 20.5 |
| **3 缓存淘汰策略** | **5** | **4.5** | **4.5** | **4.5** | **4** | **22.5** |
| 4 图像卷积 | 4 | 4 | 4 | 4 | 4 | 20.0 |
| **5 函数逼近** | **4.5** | **4** | **4** | **4.5** | **5** | **22.0** |
| 6 排序 | 3.5 | 4 | 3.5 | 4 | 4.5 | 19.5 |
| 7 JSON 解析 | 3.5 | 3.5 | 3.5 | 3.5 | 4 | 18.0 |
| 8 SpMV | 4 | 3.5 | 4 | 4 | 3.5 | 19.0 |

## 4. 推荐

**首选：候选 3（缓存淘汰策略实现）**——理由：
1. **唯一充分测试融合分协议**：命中率与吞吐天然负相关（CMU QUESTA26 A 级实证）——「speedup 权重 0.7+hit_rate 权重 0.3」vs「0.3/0.7」产生不同最优解，单指标任务（matmul）测不出加权融合的任何行为；adjust 讨论权重权衡本身是 step3 影子价格推理的真实素材。
2. **死格证据现成**（evictor 实证）：LRU 在 loop workload 命中率 0%（vs MRU 83.2%）——策略×workload 的死格是「格灭绝/降权」的直接触发器（matmul 无死格）。
3. **15 格最大解空间**：5 策略×3 数据结构——饱和→加格（expand_cell）与权重生态有充足施展面。
4. **作弊检测天然**：容量硬约束（内存峰值校验）+ 固定访问序列（不可预知未来）——无需复杂 verify 设计。
5. 接口小（get/put 两方法）——LLM 实现差异集中在策略与数据结构，可比性强。

**次选：候选 5（函数逼近）**——理由：作弊检测维度最丰富（禁库调用的静态+运行时双检测——测 gates 禁读 hook 之外的第二道防线）；逼近论 LLM 知识最密集；泰勒/切比雪夫/分片三台阶与 matmul 的 R 曲线演进同样清晰。**风险**：max_error 1e-6 约束的数值稳定性调试可能烧迭代（debug 路径反而会被充分测试——也算功能）。

**双任务备选**：若时长允许（90 分钟上限），候选 3 主跑 + 候选 5 附跑（验证 env 切换与跨任务观测基建）——单任务优先。

## R7 上报项（无法澄清的事实缺口）

1. 「30-90 分钟、10-20 迭代」的 LLM 实际耗时方差大（matmul 实测单实验 12-15 分钟）——**每实验实际耗时**需首跑 2 迭代后校准（本估算基于 matmul 实测外推，迭代数上限建议 max_iterations=16+adjust_interval=3）。
2. 缓存任务的「内存峰值」测量在 Python 内置 tracemalloc 开销大——建议约束改为「内部存储对象数 ≤ N」（计数校验替代内存测量）。
3. 函数逼近的「禁用三角函数」检测：静态扫描（AST/正则）可绕过（getattr/eval）——需运行时数值指纹（对已知点比对与多项式值的偏差模式）——verify 设计有工程量。

## Gaps

- 各候选的「迭代数到天花板」均为外推（基于 matmul 实测节奏+任务台阶数），非文献直接证据——首轮跑后需校准。
- 候选 3 的 workload 混合比例（zipfian/loop/scan 各 1/3）对最优策略的影响无文献定论——设计决策留 human（混合比例改变最优策略=任务本身的设计变量）。
- 候选 5 的 mpmath chebyfit 只支持双精度基准——reference 用 numpy.polynomial.chebyshev 更工程化（信源 B）。
