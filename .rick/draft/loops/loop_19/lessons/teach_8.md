# 第 8 课｜FlashInfer-Bench：kernel 生成-评测-部署的生产级闭环基准

> 输入：syllabus 模块 3 第 8 课｜research_4（下称 R4；信源 A/B 内联标注——「周评」＝B 级独立技术评述，对论文 Fig.7/8 逐图重算）｜zpd.md（第 7 课 consolidate＋加速偏好：快收口、大步推进）｜judgment.md 第 8 课认知强制两答（切入锚）｜values.md 待裁决 10/11 已裁决接受（立场层引用，不作事实教）

## 0｜从你的两句话切入（认知强制裁决位）

你 a 答（kernel 离线验证过 → 跑进生产栈，中间还差哪几步）：「大规模的正确性测评，生产框架下的算子替换与部署，灰度放量观察，效果验证，推全。」——一条完整的五步生产放量管线。你 b 答（三段哪段最成熟、哪段最缺）：「生成最成熟，测评和部署最缺。」

**a 的裁决位**：FlashInfer-Bench（arXiv 2601.00227v1，MLSys 2026 正式收录；UW/CMU/NVIDIA/Berkeley 12 作者；repo Apache-2.0）正好是三段闭环：生成 → 评测 → 部署。对回你的五步——「大规模正确性测评」↔ 评测段；「生产框架下的算子替换与部署」↔ 部署段 apply()；「灰度放量观察／效果验证／推全」**整段缺失**（v1 的端到端验收只有 4 个请求的均值，§5 炸弹④拆）。你的管线与 apply() 的落差＝美团版骨架的增量位——第 21 课整合的原材料，本课 §7 展开。

**b 的裁决位**：与论文自己的论题对上了。论文 2.3（A）：KernelBench 类离线基准只评「生成能力」（编译过＋selected inputs 匹配 reference＋合理加速），生产缺的是「missing operational scaffolding」——评测与部署脚手架，本文补的正是这两段。但先钉一个实证扭转：**脚手架建成 ≠ 生成侧能赢**——论文 4.5 端到端实验里两个 AI kernel 全部慢于 native（§5 炸弹①）。所以 b 答要说精确：成熟的是生成 loop 的**基础设施**（反馈管道／数据集／部署路径），最缺的是生成**质量**。本课任务：把三段机制拆到能判真，再判 AI 到底赢了没。

## 1｜闭环全景：三段一图与交接物

结构关系，先图后文：

```
serving 引擎（SGLang × DeepSeek-V3/Llama-3.1-8B/Qwen3-30B-A3B，喂 ShareGPT 流量）
   ↓ FlashInfer Level 10 logging 捕获 → sanitize → 按性能敏感轴去重（每 Definition ~50 workloads）
① 生成段｜feedback-loop agent（GPT-5/o3/Gemini 2.5 Pro/Claude × CUDA/Triton）
   ·   每轮：生成 kernel → 送评测 → 拿反馈改下一轮（10 轮顺序或 beam-3）；取最优 passing kernel
   ·   〔verifier 性质：无独立 verifier——输出对错由评测段裁决〕
   ↓
② 评测段｜正确性三分支 gate ＋ CUPTI 锁定设备计时 → Evaluation 记录
   ·   〔verifier 性质：唯一 verifier——客观、可自动判定〕
   ↓
leaderboard（rolling ＋ frozen versioned snapshot 双轨）
   ↓
③ 部署段｜apply()：error threshold 过滤 → shapes 构 key → 每 key 最快 solution → O(1) 查表
   ·   〔verifier 性质：不重验——复用评测结果作 gate，原实现 fallback 兜底〕
   ↓
serving 引擎（FIB_ENABLE_APPLY=1 路由替换；AI 全慢时 fallback 透传原实现）
```

交接物＝FlashInfer Trace JSON 四件套（自包含可移植，A）：**Definition**（I/O tensors/dtypes/axes const-var 划分＋PyTorch reference＝语义唯一源）｜**Workload**（var 轴赋值；输入以 safetensors dump／seeded random／literal 三式物化）｜**Solution**（源码＋entry point＋目标硬件/软件兼容元数据）｜**Evaluation**（不可变记录：精确绑定 Definition×Solution×Workload，含 run state、correctness/performance 汇总、执行环境快照）。axes 支持 var/const 双型与 ragged 输入（page table＋indptr）——paged/ragged attention 这类 serving 特有形态原生进契约。

流转全自动，环上无人工 gate；唯一例外：博客披露 speedup>1 的 kernel 发布前人工复核正确性（A）。

**挂图定位**（谱系图锚，第 5 课）：路线 3（serving infra 自动优化，target≠harness）**第二个 exemplar**——第一个是 z.ai（第 7 课）。对照：z.ai 优化部署工作流/运行时层，FlashInfer-Bench 下沉到**算子层**（你 #11 infra 深度梯的最底格）；verifier 从「人类工程师设计的门禁＋监控」变成「生产级基准本身」——路线 3 定义里 verifier＝吞吐/延迟/正确性，这一篇是目前最完整的实现。判语课尾你自己下（费曼⑤）。

## 2｜生成段：feedback-loop agent（略讲——agent loop 你已熟）

机制（论文 Algorithm 1，A）：每轮 agent 生成 kernel → FlashInfer-Bench 评测 → 按反馈改进 → 至改进上限，取最优 passing kernel 为最终解。代码实态（repo examples/kernel_generator，A）：默认 10 轮顺序生成，或 beam search（width=3，按 speedup 排序保留/扩展）；反馈 prompt 注入 Definition 全文（axes/inputs/outputs/constraints/PyTorch reference）＋评测 status/log/正确性 max error/延迟。模型＝GPT-5/o3/Gemini 2.5 Pro/Claude Opus 4.1；语言＝CUDA＋Triton（另支持 CUTLASS/CuTe DSL，博客列 Python/TVM 规划中）。MLSys 2026 配套竞赛（starter-kit，A）：人机同场，三赛道 fused_moe/sparse_attention/gated_delta_net，Blackwell GPU，数据集 2026-02-05 上 HF。

能力判读（论文 4.3；Fig.7 数值经周评转录，B）：32 个正确性错误中 **30 个是编译失败**（API 误用/host-device 混淆/dtype-shape 错）；语言正确率 GPT-5 CUDA 83%/Triton 96%、o3 58/96、Gemini 29/79、Claude 25/83；多数 workload 上 LLM kernel 不到 SOTA 性能的 50%——例外：RMSNorm（memory-bound，容易追平）与 GEMM CUDA（亮眼，但靠调 cuBLAS——待裁决 11「训练限库」建议的实证来由，你已裁决接受作立场层）。GQA 显式 prompt 优化策略 10 次尝试仍失败。一句话判读：**正确性层及格（Triton 最高 96%）、性能层普遍不过半——生成质量的缺口主要在「快」不在「对」。**

## 3｜评测段：正确性三分支＋测量协议（详讲——你判真协议的工业实现版）

数据集（A）：8 算子族（GEMM、Ragged/Paged GQA、Ragged/Paged MLA、Fused MoE、RMSNorm、Sampling）→ 41 Definitions × 每 Definition ~50 workloads ＝ **1,600 workloads**；240 solutions → 9,600 evaluation records（非笛卡尔积：每 solution 只评所属 Definition 的 workloads，40/解）。主评测硬件 NVIDIA B200。

**Definition 窄契约哲学**（A）：同一 I/O spec＋同一 axes const/var 角色＋同一 const 值才并为一 Definition——理想下窄到具体模型某一层的 kernel call；禁止 optional 输入/行为开关，默认行为编进 reference；行为分叉→新 Definition 而非加开关。（对照第 6 课攻击面：转嫁/开关类 reward hacking 在契约层被预先拿掉——没有可拨的开关。）

**正确性三分支 gate**（代码＋论文一致，A）——三类数值语义，三类专用 validator：

```
确定性算子 → 逐元素 |sol−ref| ≤ atol＋rtol·|ref|（默认 atol=rtol=1e-2）＋NaN/Inf 即拒＋全 trial 全元素须过（顺序：shape→dtype→非有限→误差统计）
FP8 MoE（moe_fp8_block_scale 类）→ matched ratio ≥ ρ=0.95（专用 LowBitEvaluator）
随机采样 → 100 次采样经验分布 vs mask（top-k/top-p）后归一化期望分布：TVD ≤ 0.2，任一 mask 违例即败
```

**测量协议**（代码默认，A）——第 2 课判真协议三要素，逐项认旧朋友：

- **重复测量**：warmup 10 ＋ 每次计时 50 iterations × 3 trials（每 trial 独立重新生成输入）；每 trial 内报 **median**、3 个 trial 再取 **mean**（speedup＝ref 均值/sol 均值）；每 solution 超时 300s
- **环境控制**：CUPTI activity tracing 硬件级 kernel 计时（不可用退 CUDA events）；**冷 L2**（cold_l2_cache=True，防缓存驻留虚高）；per-GPU multiprocess 锁（同卡任务串行计时）；**双 runner**——persistent（默认：每 GPU 长驻 worker＋预热备用池，反复失败降级）与 isolated（每 solution 独立 subprocess＋终止后拆 CUDA context——防 cross-run 状态残留、防读残留内存反推 reference＝第 6 课缓存/预计算攻击的正面防御）；评测记录附环境快照（硬件/torch/triton/cuda 版本＋时间戳）
- **方差与显著性**：**缺席**——trial 间只报 mean 不报 spread；多候选选最小延迟另有 winner's curse（乐观偏差，无独立复核；周评 B）。三要素补了两件半，第三件正是炸弹④的伏笔

指标与基线（A）：fast_p 承自 KernelBench（你第 6 课已会：正确∧加速比>p 的样本占比，一个指标两端惩罚）；基线＝**FlashInfer 库 kernel（SOTA 库而非 naive）**，无对应实现才退 PyTorch（如 bf16 GEMM）——你第 2 课「naive 基线无引用权」的裁决在这里被工业基准采纳。fast_0.95＝近平价判据；AUC＝∫fast_p dp（正确率加权 speedup 均值）。leaderboard 双轨：rolling（最新评测）＋定期 frozen versioned snapshot（可引用可复现）。口径纪律：站点现值（2026-11-04 查询，B）是 660-workloads 口径（gemini-2.5-pro 0.628×/73.1% resolved 等），与论文快照 1,600-workloads 口径（gpt-5 正确率 83.9% 等）**不互通、不可混算**（周评 B 明确提醒）。

## 4｜部署段：apply() 与 AOT 索引（详讲——你的 serving 世界）

两 API（A）：**decorator**（原函数作 fallback；接受固定 Definition 名，或按运行时参数解析的 resolver）与 **imperative**。与 FlashInfer 一等集成：引擎 import flashinfer_bench ＋ 环境变量 FIB_ENABLE_APPLY=1 即路由，业务代码零改动；全局关闭时透传原实现。SGLang 已集成；vLLM：论文摘要并列、博客（2025-10）称「进展迅速、预计近期完成」，repo 未见 vLLM 采集路径（完成度 B）。

**AOT 索引机制**（A）——day-0 部署路径的全部机关：

```
服务启动前：本地 dataset 按 error threshold 过滤 traces
  → 按 workload 特征（shapes）构 key
  → 每 key 选最快 solution 为值
  → 高频选中者 AOT 编译、其余 JIT
运行时：按输入参数构 key → O(1) 查表 → 命中即替换；未命中/关闭 → fallback 原实现
```

开销（4.5 实测，A）：CUDA graph＋warmup 下每次调用 1-2µs；fallback 替换端到端 **<0.8%**。这就是「替换机制可用」的量化证据：dispatcher 存在感足够小，kernel 快慢忠实传导到端到端（batch 64：native 933ms vs fallback 934ms——那 1ms 就是开销传导的实感）。

**留白三（你来设计）**：你的五步管线里「灰度放量观察」若插进这个环，插在哪一拍？key 级放量（按 shapes 子集灰度）？shadow 并行跑 native＋candidate 对比？env var 按流量分片？只要求答三件：**gate 信号取什么、放量单位是什么、回滚边界在哪**。这正是第 2 课交棒点「在线灰度判真协议形式化无先例」的落点，也是你 gen env 提议的落点。

## 5｜判真拆：五枚炸弹＋一条 Amdahl 边界（本课高潮）

第 2/6/7 课的武器全部用上，逐枚：

**① 4.5 端到端实证：两个 AI kernel 全部慢于 native**（论文 A＋周评 B 重算）。设置：fused_add_rmsnorm_h4096，SGLang 服务 Llama-3.1-8B，batch 1/16/64。batch 64：端到端 native 933ms vs fallback 934／Gemini 939／GPT-5 1055ms；kernel 级 11.2 vs 16.0/24.7µs。batch 1：native 461ms vs 替换路径 483/594ms。实证证明的是三件事——**替换机制可用＋开销小＋kernel 快慢忠实传导端到端**；未证明「AI kernel 赢」，且证据恰好反向。一句话：**机制已通、AI 尚未赢**。博客口径一致（A）：「most kernels have p<1」，少数 speedup>1 待人工复核后发布。R4 Gaps 明载：v1 无 AI kernel 端到端胜过 native 的正面案例——不臆造。

**② 论文图文矛盾**：正文把 Gemini kernel 写作「faster than the baseline」，与图值（933<939<1055ms，全慢）直接冲突——周评逐图重算发现，论文未勘误。这是第 7 课「判真逐主张」的第二个活样本：**读图不读文，数字对数字**。

**③ 计时口径张力**：论文正文只写「mean latency over m runs」，未提 trial 内 median 与冷 L2；代码口径更细（median→mean＋cold_l2_cache=True）。论文未给 warmup/timed 具体数，repo 默认 10/50/3 代（w/m 数值＝R7 缺口）；论文实验对应哪个代码版本，外部无法裁定。

**④ 端到端验收仅 4 请求均值**：无方差/CI/尾部分位（周评 B：observed 差 1-5ms 时，无 run-to-run 变异度即无法判精度）——你判真协议「方差与显著性」要素的直接靶子。**留白二（你来当 reviewer）**：4.5 若由你验收，要求作者补什么？n 取多少、报什么统计量、控什么时段效应——第 2 课 §6 在线灰度那节你已有答案骨架。

**⑤ matched-ratio 漏洞（gate 的边界）**：ρ=0.95 只数「对多少」不数「错得多坏」——950 全对＋50 个任意坏值可以通过；且 objective 会被优化：agent 重复优化会把误差集中打进允许的 5% 比例（第 6 课「objective 被优化」的 gate 版）。同族边界（周评 B）：empty context 的 −inf log-sum-exp 是 non-finite 规则未记载的例外；hidden workloads 论文提及但无完整 split 与搜索预算核算。**留白一（你来推）**：不改 ρ，怎么堵「950＋50 任意坏」？validator 加什么约束（幅值？误差分布形状？）——提示在三分支设计哲学里：确定性分支是「逐元素＋幅值」联合约束，matched-ratio 只继承了一半。

**Amdahl 边界（周评 B）**：替换有益条件 **f(1−1/s)＞δ**——节省（kernel 时间占比 f × 加速比 s 决定）必须超过 dispatcher 开销 δ。推论：**dispatcher 应把 native 保留在候选集内**——AI 全慢时选回 native（v1 实验即此情形，不是假想敌）。这条直接进 §7 骨架设计件 3。

**agent 能力判读收口**（连接①）：30/32 是编译失败＋多数 workload <50% SOTA——「AI 尚未赢」不是 gate 太严卡出来的，是生成质量的真实水位（§2 判读）。

## 6｜「生产级」三实差 vs KernelBench（对照表）

| 维度 | KernelBench（类离线基准） | FlashInfer-Bench（论文 2.3 口径，A） |
|---|---|---|
| 负载来源 | uniform/random 合成 | ① 真实 serving trace 采集：SGLang × DeepSeek-V3（原生 FP8，TP=8/EP=8）/Llama-3.1-8B/Qwen3-30B-A3B × ShareGPT prompts；Level 10 logging 捕获＋sanitize＋按性能敏感轴（batch size）与 tensor 统计（平均序列长）去重；值敏感输入（采样概率分布/极端 edge case）dump 全 tensor，否则 seeded random 省存储 |
| verifier | 编译＋selected inputs 匹配 reference＋「合理加速」 | ② 反 reward-hacking：runtime isolation（isolated runner 拆 context）／hidden workloads／三分支专用 validator |
| 部署路径 | 无 | ③ apply() day-0：env var 开关＋原实现 fallback＋AOT 索引 |

配套件（A）：Definition 窄契约＝精确任务规格（API 语义/形状 dtype/布局约束）；评测服务 FastAPI 编排（repo serve/；含多设备调度——cost matrix＋匈牙利算法＋EWMA，论文有述、生产规模未载，R4 Gap）。

「生产级」的边界（周评 B＋论文 A）：trace 只是所选模型/配置/流量样本——量化方式、上下文长、batching 策略、MoE 路由变化都会改变主导 kernel 与最优实现，**dispatch table 有有效期与适用域，非普适**；论文自认多 GPU/通信 kernel 未覆盖（R4 Gap）。

## 7｜美团骨架落差位：你的五步管线 vs apply() 的两步

| 你的五步 | FlashInfer-Bench 现状 | 状态 |
|---|---|---|
| 大规模正确性测评 | 评测段：三分支 gate＋1,600 workloads＋SOTA 库基线 | 有（最厚的一段） |
| 生产框架下的算子替换与部署 | 部署段：apply()＋AOT 索引＋fallback | 有（机制验证到端到端） |
| 灰度放量观察 | —— | **缺** |
| 效果验证 | v1 验收＝4 请求均值（无方差/CI/分位） | **缺**（仅有弱替位） |
| 推全 | —— | **缺** |

客观对应物（R4 面 5，只罗列不替你设计）：生成段→面向在线算子的 agent 生成循环（模型×语言×目标 GPU 为配置轴；竞赛 solution.json 打包-提交-评测流同构）｜评测段→引擎日志捕获＋sanitize、op_type/Definition 两层窄契约、三类数值语义三类 validator、per-GPU 锁＋双 runner、fast_p＋SOTA 基线、rolling/frozen 双层排行榜＋HTTP 评测服务｜部署段→挂在 kernel 库调用边界的动态替换（env var＋fallback）、AOT＋O(1) dispatch、端到端以 serving 请求延迟验收（4 请求均值，v1 口径）。

原创视角——美团版骨架三件（第 21 课整合原材料，判断权在 human）：

1. **五步全长度**：apply() 的两步 → 补灰度/验证/推全三步。增量位＝你的判真协议在线化（第 2 课交棒点：在线灰度判真协议形式化无先例——你的主场空白）
2. **判真协议在线化**：4 请求均值 → n/方差/分位/时段效应/流量一致性的完整在线验收口径（留白二的答案直接迁移）
3. **Amdahl/native-保留设计**：dispatcher 候选集常驻 native；f(1−1/s)＞δ 作切换决策式——v1 已实证「AI 全慢」是真实情形而非假想敌

立场层引用（已裁决接受，不作事实教）：博客结语「已建成 self-optimizing loop，AI 自我改进的革命将持续」（待裁决 10）——叙事定性句；v1 实证是「机制通、AI 未赢」。论文 4.3「训练限库、推理放开」（待裁决 11）——规范性建议，影响美团 kernel agent 训练/评测口径设计。

## 8｜本课关键概念清单＋费曼引导

1. **闭环三段与各段 verifier 性质**：生成段＝feedback-loop agent（无独立 verifier——输出对错由评测段裁决）→ 评测段＝唯一 verifier（三分支正确性＋CUPTI 锁定设备计时，客观可自动判定）→ 部署段＝apply() 不重验（复用评测结果作 gate，原实现 fallback 兜底）
2. **三分支 gate**：确定性算子逐元素 atol=rtol=1e-2／FP8 MoE matched ratio ρ=0.95／采样 TVD≤0.2×100 trials——三类数值语义三类专用 validator
3. **测量协议**：warmup 10＋50 iterations×3 trials＋冷 L2＋trial 内 median 再 trial 间 mean＋per-GPU 锁＋双 runner——判真协议三要素的工业实现版，「方差与显著性」要素缺席
4. **4.5 实证状态**：机制已通（替换可用＋开销<0.8%＋快慢忠实传导端到端）、AI 尚未赢（933 vs 939/1055ms）；图文矛盾＝正文「faster than the baseline」vs 图值全慢（周评 B 重算，论文未勘误）
5. **生产级三实差**：真实 trace 采集／反 reward-hacking verifier／apply() day-0 部署路径——vs KernelBench 只评生成能力
6. **美团落差位**：你的五步管线 vs apply() 两步——缺灰度放量观察、效果验证、推全；在线灰度判真＝你的主场空白＋gen env 落点；Amdahl：native 常驻候选集，f(1−1/s)＞δ 才替换

**费曼检验（五件）——现在你当老师，把这一课讲给我听：**

① 画出闭环三段图，并说出各段 verifier 性质（提示：生成段为什么没有自己的 verifier？部署段为什么不再验一次？）
② 判真拆两说各一句：4.5 实验证明了什么、没证明什么；图文矛盾出在论文哪一句
③ 「生产级」三实差 vs KernelBench——一条一句
④ 你的五步管线 vs apply() 的两步：缺的三步是哪三步？其中哪一步直接接你第 2 课的交棒点？
⑤ 挂图定位：路线 3 第二个 exemplar——在你 #11 infra 深度梯上落在哪一格？判语一句（对照第一个 exemplar z.ai）
