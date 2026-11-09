# research_2 — 第 2 课「判真统计协议」机制级事实材料 — 2026-11-03

> SENSE 第 1 步问题还原：期望 = 第 2 课「判真统计协议」教学需机制级事实（三要素：重复测量/方差与显著性/环境控制，rubric 要求能机制化说清）；现状 = research_1 §② 只有名词级骨架（allclose + do_bench 锁频 + Welch p<0.05/d>0.2 的名词清单，无任何机制细节），human 现有协议 = 阈值 3% + review agent（零统计要素，zpd 判不会）。差距 = 本材料填充的 8 块。
> 时间基准 2026-11-03（triton main 当日 HEAD 已克隆核对；PyTorch/NVIDIA 博客原文 2026-10-19）。信源等级：A = 源码原文/论文原文/官方一手；B = 官方文档/权威教材；C = 弱来源仅线索。本材料只下钻 research_1 §② 骨架的机制细节（其已有结论与数字不重复：2-7% 量级规律、ACF 机制、Caveats、联合搜索论证均见 research_1 §②）。第 2 课教学判据 = 判真协议三要素（重复测量/方差与显著性/环境控制）机制化说清。

## ① do_bench 计时协议（triton 源码，A）

- 源码：python/triton/testing.py + third_party/nvidia/backend/driver.py（main 分支克隆逐行核对）。签名：`do_bench(fn, warmup=25, rep=100, grad_to_none=None, quantiles=None, return_mode="mean")`。
- **warmup/rep 是毫秒时间预算，不是次数**：先跑 1 次 fn() 并同步 → 用一对 CUDA events 给「5 次（每次前清 L2）」计时估出单次耗时 estimate_ms → n_warmup=max(1,int(25/estimate_ms))、n_repeat=max(1,int(100/estimate_ms))。0.097ms 的 kernel ≈ 257 次预热 + 1030 次计时/臂（由公式推导，标注为推导）。
- **L2 清除怎么清**：NVIDIA 后端 `get_empty_cache_for_benchmark()` = `torch.empty(256MB//4, dtype=torch.int, device='cuda')`（256 MiB 缓冲；AMD 后端同款实现），`clear_cache(cache)` = `cache.zero_()`——每次计时前把整块 256MB 写零。源码注释原话："We maintain a buffer of 256 MB that we clear before each kernel call to make sure that the L2 cache doesn't contain any input data before the run"。
- **L2 清除为何**：不清则第 i 轮计时依赖第 i-1 轮残留在 L2 的数据（输入恰好命中缓存→虚快），轮间不独立、跨 kernel 不可比。256MB > 数据中心卡 L2 容量（H100=50MB，B：NVIDIA 规格），保证覆写全缓存。
- **CUDA events 计时**：每轮 [清梯度(grad_to_none) → 清 L2 → start_event.record() → fn() → end_event.record()]，n_repeat 轮全部入队后一次 di.synchronize()，times = 各对事件 elapsed_time(ms)。事件时间戳打在 GPU 侧 stream 上，不含 host 启动开销；轮间无 host 同步间隙（只有清 L2 的写零 kernel 占据 stream，恰好维持 GPU 连续负载）。
- **短 kernel 的 host 开销变体**：do_bench_cudagraph 把 n_repeat 次展开捕获成 CUDA graph 后整体 replay 10 次（n_retries=10）再除回 n_repeat——消除 CPU 侧 launch 开销。源码注释佐证噪声面：建图本身 ~300ms（A100）须预热排除；Python GC 会卸载 CUDA module 破坏捕获，捕获窗内禁 GC。
- **返回什么统计量**：`_summarize_statistics`：return_mode ∈ {min/max/mean/median/all}（**当前 main 默认 "mean"**——网络旧资料常称默认 median，系旧版本行为，源码为准）；quantiles 参数（如 [0.2,0.5,0.8]）返回分位数列表；"all" 返回原始逐轮列表。
- **为何报分布而非单值**：逐轮时间本身有分布（频率漂移/温度/缓存态，见⑤）；单值丢失方差信息→下游无法做显著性检验；分布（原始列表或分位数）才供 Welch t/Cohen d 消费。Helion×CompileIQ 调 `do_bench(run, warmup=25, rep=100, return_mode="mean")`，逐轮样本交给外层统计过滤（见⑥）。

## ② Welch t 检验（A：Welch 1947 原文题录经 OUP 核对；公式经多教材一致）

- **原始文献**：B. L. Welch, "THE GENERALIZATION OF 'STUDENT'S' PROBLEM WHEN SEVERAL DIFFERENT POPULATION VARIANCES ARE INVOLVED", Biometrika 34(1–2): 28–35, 1947-01, DOI 10.1093/biomet/34.1-2.28（OUP 期刊页核对题录；全文付费墙未逐字读）。
- **与 Student t 的区别**：Student 两样本 t 假设两总体方差相等，用合并方差 s_p 估标准误、自由度 = n₁+n₂−2；Welch 去掉等方差假设，标准误直接 √(s₁²/n₁ + s₂²/n₂)，自由度用 Welch–Satterthwaite 近似（Satterthwaite 1946，Biometrics Bulletin 2(6):110–114，题录未本次核对、常识置信 B）——df 小于 Student 版。不等样本量天然支持，无需配对。
- **为何适配 kernel 计时场景**：baseline 与 optimized 是两臂独立采样，方差通常不等（PyTorch 博客实测：baseline std=0.00218 vs optimized std=0.00133，差约 65%），样本量也不等（n=1744 vs n=2018）——等方差假设在此场景为假，用 Student t 引入系统偏差；Welch 恰是为此场景造的（论文标题即"several different population variances"）。
- **p 值含义**：若原假设（两臂均值相等）为真，观察到当前或更极端均值差的概率。p<0.05 = 5% 显著性水平下拒绝原假设（每次比较在原假设下期望 ≤5% 假阳性率）；它不是「原假设为真的概率」，也不携带效应大小信息（见③）。博客报 p=0.0 属浮点下溢（n≈1800、d≈2.5 下真实 p 极小——此句为解读标注）。
- **两样本应用方式**：每臂收集逐轮计时样本 → 各算 mean/std → Welch t + Satterthwaite df → 与 α=0.05 比较。边界：逐轮样本可能自相关（频率漂移有周期性），独立样本假设是近似，官方协议未处理（见 Gaps）。

## ③ Cohen 效应量 d（A：Cohen 1988；解释性文献 B）

- **含义**：d = (均值差)/合并标准差——以标准差为尺度的均值差（度量无关的标准化差异）。与 p 正交：d 说差异多大，p 说差异多不像偶然。
- **统计显著 ≠ 效应量大**：p 随 n 缩放（n 越大、越小差异也显著），d 不随 n 缩放（只依赖两分布本身）。n≈1800 时 0.1% 级真差异也 p<0.05，但 d≈0.05 的 kernel 改进无工程意义——这就是需要双闸（显著 ∧ 有量）的原因。
- **0.2 小效应阈值出处**：Cohen, *Statistical Power Analysis for the Behavioral Sciences* (2nd ed., 1988)：d=0.2 小 / 0.5 中 / 0.8 大（p.25 起）。Cohen 自述约定 "somewhat arbitrary"（引文中带 "recommended for use only when no better basis for estimating the index is available"）——0.2 是领域约定基线而非数学推导值（Funder & Ozer 2019 *Advances in Methods and Practices in Psychological Science* 及 Sawilowsky 系解释文献均确认此性质，B）。
- **在判真协议中的角色（Helion×CompileIQ 实测）**：4.7% 加速 → d≈2.5（12.5× 过 0.2 闸）；1.75–2.0% → d=0.78；双 shape 联合 1.05×→d=67.71 / 1.01×→d=4.54；多目标 6.9% → d=87.024（L1TEX 4.5pp→d=3.686）——均为官方博客报数。d>0.2 的机制作用：把「虽真实但太小」与「真且有量」分开；进化搜索每代多次比较，仅 p 闸下 5%/比较的假阳性随比较数累积——这正是 zpd 会区锚「0.5% 逐轮接受→噪声累积→系统性假阳性」的统计对应（对照句为综合标注）。

## ④ 环境控制：锁频 / 预热 / 隔离（A：triton 源码 + NVIDIA 官方博客；B：TensorRT 性能文档）

- **浮频机制（要压的噪声）**：默认 GPU 时钟浮动——空闲降频、负载来时 boost（TensorRT 文档 "GPU Clock Locking and Floating Clock" 节）。NVIDIA SetStablePowerState 官方博客原话："Most modern processors, including GPUs, change processor core and memory clock rates during application execution. These changes can vary performance, introducing errors in measurements and rendering comparisons between runs difficult"——每个 kernel 可能跑在不同频率。
- **锁频机制**：`nvidia-smi -lgc <min>,<max>`（--lock-gpu-clocks）锁 SM 时钟、`-lmc`（--lock-memory-clocks）锁显存、`-rgc` 恢复浮频、`-pl <cap>` 功率上限、`-pm 1` persistence mode（Linux、需 root、重启失效；保持驱动常驻避免首次初始化惩罚）——均为 nvidia-smi 官方 man（B）。锁后频率固定，除非 power/thermal throttling 触发（届时表现同浮频）。
- **triton 源码级佐证（A）**：triton 自带测试工具 set_gpu_clock 顺序执行 `-pm 1` → `--lock-gpu-clocks=1350,1350` → `--lock-memory-clocks=1215,1215`，再回读 clocks.current.sm/memory 断言 |实测−目标| < 10MHz——锁频+回读校验的完整闭环在源码里。
- **锁频代价（NVIDIA 官方立场，如实记录）**：锁频下平均性能低于浮频/最高频锁定；TensorRT 文档明言 "There is no definite recommendation on whether the clock should be locked... It depends on whether the deterministic and stable performance or the best average performance is desired"——确定性 vs 平均峰值是官方承认的显式取舍、判断留给用户。
- **预热消除的三类瞬态**：① DVFS 爬升（空闲低频→负载 boost 的过渡期频率不稳）；② 一次性编译/加载成本：CUDA module 首次加载、Triton 首调编译、CUDA graph 捕获（~300ms A100）、allocator 首分配——必须挡在计时区外（do_bench 预热环为此存在）；③ cache 冷热：L2 残留上一轮数据（①节 256MB 清法）。
- **功率/温度节流（锁频也压不住的噪声）**：平均功耗达 -pl 上限→驱动降频（TensorRT：短窗测量「within 20ms」尤其不稳）；温度 ~85°C 阈值→热节流；高温增漏电流→同频更耗电→更易功率节流（恶性耦合）。
- **结构陷阱（连在线/离线差异）**：推理间留空隙（每步同步）→平均功耗被拉低→频率被抬→测出虚高吞吐；满载连续时实际频率更低、真吞吐达不到——trtexec 为此设计成 kernel 间几乎无间隙（TensorRT 文档原文）。
- **隔离**：`nvidia-smi -c EXCLUSIVE_PROCESS` 独占进程模式（每 GPU 仅一个进程上下文，B）；邻居干扰实证：同数据中心同型 GPU 性能有差异、风冷热点节点严重功率节流（arXiv 2208.11035《Analyzing Performance and Power-Efficiency Variations among NVIDIA GPUs》，A 级论文但数字未逐页核——见 Gaps）；跨 NUMA 的 PCIe 带宽显著受限，host 内存与拷贝线程须钉在 GPU 所在 NUMA 节点（TensorRT 文档，B）；CompileIQ 用 WorkerTypes.ISOLATED——崩溃候选跑在隔离子进程，源码注释 "crashing ACF candidates shouldn't kill the search"（A：博客代码）。
- **host 侧其他变量（TensorRT 文档，B）**：sync 模式（spin-wait 测量稳但耗 CPU / blocking-sync 省电但 overhead 不稳）；每次 kernel launch 占 CPU 5–15μs（短 kernel 成瓶颈→CUDA graph）；输入值分布影响功耗（全 0/NaN 位翻转少→功耗低→测得偏快——须用真实分布输入）。

## ⑤ 重复测量协议：测几次 / 方差怎么估 / 单次为何不可信（A/B 混合，逐条标注）

- **测几次（Helion×CompileIQ 两级协议，A：博客）**：搜索环内 `do_bench(warmup=25, rep=100)`——按时间预算自动定次数（0.097ms kernel ≈ 1030 次/臂，推导）；独立验证环用 "increased warmup and repetition iterations"→ 每臂 n=1744–1757（baseline）/ 2018–2028（optimized）逐轮样本，且跑两轮独立复测（run1/run2）。
- **方差怎么估**：逐轮样本的样本标准差 s（无参数假设）；博客报 mean±std：baseline 0.09752±0.00218ms、optimized 0.09301±0.00133ms；SE(均值差)=√(s₁²/n₁+s₂²/n₂)（即 Welch 标准误，②）。
- **单次为何不可信（噪声源清单，机制见④/①）**：① 频率漂移/DVFS（驱动逐时刻选频）；② 功率节流（-pl 触发降频）与热节流（~85°C）；③ L2 残留态（轮间不独立）；④ JIT/加载/建图一次性成本；⑤ host 侧：launch 5–15μs、sync 模式差异、allocator、Python GC；⑥ 邻居：同节点他进程、同机房他 GPU（功耗/温度共享）；⑦ NUMA 错位（跨节点 PCIe 受限）；⑧ 输入值分布（全 0 vs 随机功耗不同）。
- **量级算术（自推导，标注为推导非引用）**：baseline CV ≈ 2.2%（0.00218/0.09752）→ 单次测量 ±2σ ≈ ±4.4%——比待判的 +3% 还宽，单次测量无法把 3% 改进与噪声分开；n≈1800/臂时 SE(均值) ≈ CV/√n ≈ 0.05%，故 4.7% 差异 p→0。√n 收缩是重复测量的全部统计收益。
- **三要素的因果链（综合标注）**：重复测量→可估方差→可算显著性（②）与效应量（③）；环境控制（④）→压低方差本身（锁频去①②、清 L2 去③、预热去④、隔离去⑥⑦）。方差越小，同 n 下检验越灵敏、同幅度改进的 d 越大——三要素不是并列清单，是「先降噪声、再测、再判」的递进结构。

## ⑥ Helion×CompileIQ 统计过滤流水线（组合机制，A：PyTorch 博客 2026-10-19 原文）

（research_1 §② 已录骨架与收益数字；本节只补要素怎么组合成流水线 + 官方原话细节。）

- **顺序结构（objective 函数，博客代码原文）**：候选 ACF → 编译 → 跑一次 → 对 PyTorch 参考实现做正确性检查（不过→invalid score，"dropped from the search before it ever contributes a timing measurement"——**正确性 gate 严格在计时之前**，不浪费计时预算）→ 过 gate 者 `do_bench(run, warmup=25, rep=100, return_mode="mean")` 计时返回目标值。
- **搜索外层（A：博客代码）**：`SearchConfiguration(problem_type="min", generations=8, pool_size=15)`——进化育种 8 代×15 候选池；`WorkerTypes.ISOLATED`（崩溃候选不杀搜索）；`gpu_benchmark_mode(clock_mhz=...)` 上下文管理器在整场搜索期锁频；`task_timeout=60`。
- **独立验证环（与搜索环分离的判真环）**：胜出 ACF "re-measured independently using increased warmup and repetition iterations under fixed GPU clock frequencies"；**三条件合取才算 significant**（博客代码注释原文 "(requiring p < 0.05, cohens_d > 0.2, and optimized.mean < baseline.mean)"）：Welch p<0.05 ∧ d>0.2 ∧ 方向闸（optimized 均值更低）。
- **复现性检查内嵌**：run1 与 run2 两轮独立复测 "virtually identical within experimental variance"（d=2.501 vs 2.489、均值差 <0.01%）——独立复测本身是协议一环。
- **同一过滤在所有目标形态上套用（A：博客）**：occupancy 目标（21.60→21.76%，p=9.2e-23、d=13.17）→ 独立转测 runtime 1.75–2.0%（p=3.6e-116、d=0.78）；多目标 Pareto 6.9% runtime + 4.5pp L1TEX；联合双 shape 1.05×（p=1.1e-224, d=67.71）/1.01×（p=6.2e-73, d=4.54）。
- **机制要义（综合标注）**：统计判真不在搜索环内逐代执行（搜索环只需相对排序）；在「独立、加预算、锁频」的验证环执行——搜索与判真分离，是「训练-验证拆分」的测量版对应。

## ⑦ 在线灰度判真 vs 离线判真（A：Kohavi/Tang/Xu 2020 + MSR 文章；B：负载模型文档）

- **离线（kernel 计时）结构**：实验者全控——负载自己造（固定 batch / 闭式并发 / do_bench 背靠背入队）、环境自己锁（④）、同机对照；唯一随机源是测量噪声。判真 = 对抗测量噪声（②③⑤）。
- **在线 A/B 结构与统计基础（A：Kohavi/Tang/Xu, *Trustworthy Online Controlled Experiments*, Cambridge 2020, Ch.17）**：按单元（用户/请求）随机分流到变体；每单元业务指标 → CLT → 两均值 t 检验/两比例 z 检验；显著性水平 α（0.05）；样本量由功效分析定（MDE/α/1-β）；置信区间报效应。统计内核与离线同源，差别在噪声结构与混杂控制。
- **在线特有噪声/混杂（离线没有的，A：同书）**：流量按小时/星期周期波动、用户构成漂移、外部事件；**新奇效应/启动效应**（novelty/primacy：新体验初期效应大后衰减——须跑满周期、勿信首日）；**变体间干扰/泄漏**（共享缓存、双边市场破坏 SUTVA）；**A/A 实验**校准假阳性率与管线。
- **分流一致性（SRM 护栏，A：同书 Ch.21 + MSR《Diagnosing Sample Ratio Mismatch in A/B Testing》）**：对实际分配比例做卡方检验——若与配置比例（如 50/50）显著不符，实验内部效度已坏（hash 偏置/重定向丢失/机器人），后续任何显著性都不可信；SRM 检查应进每个实验。
- **结构差异（排队论面，B：k6/Grafana/Gatling 官方负载模型文档；A：Little 1961）**：在线是**开式模型**（控制到达率：请求独立到达、不看系统当前状态）而离线基准多为**闭式**（控制并发数：下一迭代等上一迭代结束、满载时新请求进不来）。闭式下吞吐被服务速度封顶、迭代率由延迟决定；开式下队列可增长、延迟分布由**到达过程的方差**（不只均值）决定。两模型由 Little 定律 L=λW（Little 1961, *Operations Research* 9(3):383–387，题录常识置信）连接。human 会区锚「到达分布不一致→压测-灰度差异」即此机制；④的「推理空隙抬高频率→虚高吞吐」是同一差异的功耗面例证。
- **灰度特有形态**：小流量渐进放量（ramp）+ 护栏指标自动回滚——在线版的「显著 ∧ 有量 ∧ 方向对才接受」；判真时间尺度从分钟级（离线）变天/周级（覆盖周期）；样本量由流量决定、不可像离线那样加测随意扩。
- **三要素的在线对应（综合标注，供 lesson-writer 直引）**：重复测量→样本量功效分析+跑满周期；方差与显著性→同套检验+SRM 护栏+A/A 校准；环境控制→分流一致性+干扰控制+时段分层。

## ⑧ 延伸面：K8s 探针语义与反模式（供 lesson-writer 选材，非第 2 课核心；A：kubernetes.io 官方 docs）

（loop_15 主题=探活测试，此块供主题延伸选材。）

- **三探针语义（官方 concepts 页 "Liveness, Readiness and Startup Probes"）**：liveness 失败超过 failureThreshold → kubelet 杀容器、按 restartPolicy 重启（检测死锁类不可恢复态）；readiness 失败 → Pod IP 从匹配 Service 的 EndpointSlice 摘除、不再收流量，**容器继续跑**（临时不能服务：加载数据、依赖暂不可用、过载恢复）；startup 只在启动期跑，未成功前 liveness/readiness 被抑制（慢启动保护；failureThreshold×periodSeconds 覆盖最坏启动时间，官方例 30×10s=300s，之后 liveness 接管快速响应死锁）。
- **官方反模式警告（Caution 原文级，A）**：① "Incorrect implementation of liveness probes can lead to cascading failures... restarting of container under high load; failed client requests as your application became less scalable; and increased workload on remaining pods"——liveness 探测在高负载下失败（如探针端点本身变慢）会把过载转成**正反馈重启风暴**；② 依赖检查应放 readiness 而非 liveness（官方模式：liveness 查应用自身健康、readiness 额外查各后端依赖——放 liveness 则重启修不了依赖反而放大其负载）；③ exec 探针每次执行 fork 多进程，高 pod 密度+短 period 下 CPU 开销可观（官方 Caution）；④ readiness 实现不当可致容器内进程数持续增长+资源饥饿（官方 Caution）；⑤ HTTP 探针成功=2xx/3xx；kubelet 读完 10KiB 即断连（大 body 会在应用日志制造 connection reset 噪声、难与真网络故障区分）；≥11 次重定向按成功处理并记 ProbeWarning。
- **假阳/假阴结构（参数面，A：官方字段语义）**：periodSeconds（默认 10s，最小 1）/timeoutSeconds（默认 1s）/failureThreshold（默认 3）共同决定「故障检出延迟 = failureThreshold×periodSeconds」与探针自身开销的权衡——过密给系统加负载（③⑤）、过疏拉长检出窗；timeoutSeconds=1s 下 GC 暂停/慢盘可造成非死锁的超时（假阳）→触发重启；successThreshold 对 liveness/startup 必须为 1。假阴性结构（应用死锁但探针端点由另一线程存活）无官方表述——推演不入判据（见 Gaps）。
- **与判真的同构（综合标注，非官方原文）**：探针=二值化的重复测量——period=采样间隔、timeout=测量窗、failureThreshold=容忍度（工程形态的显著性水平：默认 3 连败才判死，即容忍 2 次噪声假阳）；级联重启=噪声假阳在分布式层的放大形态。可作第 2 课「重复测量+容忍度」的跨域对照素材。

## R7 上报项
- [do_bench 默认统计量]：当前 main 源码 return_mode 默认 "mean"，旧版与二手资料多称 "median"——缺：钉住具体 triton 版本（教学若引 median 须指定版本号）。
- [Welch 1947 全文]：原文付费墙，仅核题录；Satterthwaite 1946 题录未本次核对——缺：全文/原文核对。
- [arXiv 2208.11035]：同数据中心 GPU 间性能差异具体数字只达搜索摘要级——缺：逐页核对。
- [Kohavi 书]：novelty/SRM 的具体章节页码未精核（章级已核：Ch.17 统计、Ch.21 SRM）。

## Gaps
- 逐轮计时的自相关（频率漂移有周期性）未被 do_bench/Welch 协议显式处理——独立样本假设在锁频后近似成立、未锁频时存疑；无官方来源讨论此点，教学可如实作为协议边界呈现。
- 在线灰度的美团侧具体实践（分层实验平台、流量一致性 hash、护栏指标清单）无公开一手材料——human 实操为主场，教学从 human 经验接入（zpd 会区锚直引）。
- CompileIQ `gpu_benchmark_mode` 的锁频实现（内部是否调 nvidia-smi -lgc）未见源码级披露——本材料将 triton set_gpu_clock（源码级）与博客「GPU clock locked」表述并置、未合并为同一机制。
- K8s 探针假阴性结构（死锁但探针线程仍活）无官方文档表述，正文已标注为推演、不入判分锚点。
