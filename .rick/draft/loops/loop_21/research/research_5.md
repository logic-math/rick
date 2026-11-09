# research_5 — GEAK e2e_workflow 机制级尽调（loop_17 第 10 课地基） — 2026-11-04

> 时间基准 2026-11-04。信源：A = AMD 官方技术文章（amd.com/en/developer/resources/technical-articles/2026/geak-v4.html，2026 测试口径 2026-07-08）+ ROCm 官方文档站（rocm.docs.amd.com/projects/geak：what-is-geak / geak-pipeline / release-notes）+ GitHub AMD-AGI/GEAK 代码原文（本地 clone，main HEAD 2026-11，tags v4.0.0/4.1.0/4.2.0）+ arXiv 2507.23194 v1 摘要页 + GitHub Releases 全文 + GLM-5.2 AgentX 官方博客（2026-11-04）+ Hyperloom 官方文档/博客（B）；C = 微信文章⑨转述（仅线索）。**谱系关键发现：arXiv 2507.23194 = GEAK v1（2025-07，Triton kernel 生成 agent + 评测基准）；e2e_workflow 与进化知识库 = GEAK v4（2026-07-17 发版，无 arXiv 论文，一手来源为官方文档+代码）。research_1 行 13/15 的「GEAK 2507.23194 生产级 + e2e_workflow」是两代混标，第 10 课引用须拆开。**

## 事实性结论（五个尽调面）

### 面 1｜e2e_workflow 机制：优化对象、循环形态、verifier
- **优化对象 = 别人的开源引擎栈的 serving 配置＋后端选择＋内核**，不含引擎编排/调度器本身改写。输入 = 磁盘上的模型 + serving backend（sglang|vllm|atom）+ 工作负载（ISL/OSL/concurrency）；profiling 与 benchmark 用同一负载保证可比。三档杠杆 cheapest-first：①launch flags（--attention-backend、--quantization 等）②env vars（SGLANG_USE_AITER、VLLM_ROCM_USE_AITER、AITER_CONFIG_GEMM_* 等）③source（Triton/FlyDSL/HIP/CK/asm/TileLang 重写或从头 author + rebind 调用点）。（官方文档+代码 A）
- **agent 循环形态 = 确定性 JS 控制面 × LLM 判断面分离**：Claude Code dynamic Workflows——JS 拥有预算循环、并行 fan-out、验证、递归、停止条件；LLM agent 只做结构化技术判断（分析/策略/内核 authoring/集成）。角色分工（每阶段一个专职 agent）：e2e Director（隔离评测目录+记录真基线含噪声带+终判独立复测仲裁）、System Architect（Amdahl 路由+里程碑计划+知识策展）、Profiler（warm-server trace→标准化 Top-N 表）、Config Tuner、Kernel Extractor（真实 shape+参考 I/O→不可变 standalone unittest）、Op Benchmarker（后端 bake-off）、e2e Integrator（可逆 overlay 回接+e2e gate）。（官方文档+代码 A）
- **流水线**：Setup/Preflight→Profile→Strategize→ConfigSweep→HeadKernel→Milestone loop→Finalize/Report/Validate；每接受一个改动即复合进 carried-forward overlay+config 并 re-profile（瓶颈移位即重排）。三模式：fast（仅 HeadKernel，并行 extract-and-bake）／default（全流程，每接受改动一次 e2e gate）／deep（跨内核×后端 lane 池+状态持久化+共享知识+reseeding+收敛停机）。**分形两层**：e2e 层把热内核递归交给原封不动的 kernel_workflow（传不可变 oracle+目标语言+shape/dtype+regime+预算，收 verified patch），kernel 层质量不回归、向后兼容单内核直通。（官方文档+README A）
- **verifier（端到端吞吐怎么测）＝backend 无关基准 harness**（bench_e2e.sh，Director/Profiler/Config Tuner/Integrator 共用同一测量）：拥有 server 生命周期、serving-GPU 互斥锁、untimed warmup、N timed rounds、median+spread 报告。三种测量生命周期不可混比：warm_server（默认，1 boot/leg，全量 untimed 轮暖 prefix cache 后 REPEATS 次 timed 取 median——即 Hyperloom warmup_round/measure_round 两遍协议）／isolated_server（每 timed replica 全新冷启，跨 replica median，spread 界 boot-to-boot 方差）／legacy。文档明确：warm vs isolated 同配置差值方向不可预测（实测 DeepSeek-V4-Pro TP8 warm 826.9 vs isolated median 833.3 tok/s），禁止跨生命周期取比值。（代码+文档 A）
- **接受 gate 三条件**（每个被接受改动）：engagement proof（改动真在 live server 跑）＋超噪声带（固定 shape 0.5%、AgentX trace replay 4% floor，代码 gateNoiseBand）＋输出 parity（greedy byte parity；量化/lossy 改动改走 gsm8k accuracy gate——5-shot greedy exact_match，cand ≥ baseline − tol）。代码内另有方法论卡（learned/method-e2e-ab-harness.md）：interleaved A/B（REF,CAND,REF,CAND…单 GPU+固定 port，≥5-7 repeats/leg）＋非重叠分布 gate（cand_min>ref_max，sub-band 且重叠=NULL）＋drift-control legs（防 warming 伪影：最后一条腿须为 REF 且属最低）＋同 session 重测分母（22h 旧基线曾把 +15.72% 报成 +0.86%；腿内 spread>~5% 视为无效重跑）＋parity 每腿探两次（4 hashes 先证基线自确定性）。Director 终判独立重测并仲裁官方数字。（代码+方法卡 A）
- **数字（AMD 性能实验室 2026-07-08，MI300X 8 卡+MI355X 8 卡，Opus 4.8 单流）**：e2e +60% Qwen3.5-27B-FP8（SGLang，gemm_a8w8_blockscale Triton→CK）、+96% Qwen3-14B-FP8（Triton→FlyDSL）、+42.2% Minimax-M3-MXFP8（MI355X vLLM TP8 sparse attn）、+25.4% Kimi K2.6 INT4、+47%/+108% DSV4 Pro FP4（MLA TileLang→Triton）；head-kernel geomean 1.33×（Claude Code 裸跑 1.09×、v3.2 1.09×）；单内核 Triton 2.21×/HIP 2.72×（v3.2 为 1.64×/1.99×）；预算-性能扩展律 47min 1.300×→70min 1.328×→104min 1.511×。MiniMax-M3 案例：吞吐 1183.8→1683.3 tok/s，TTFT 中位 1896→937ms（−50.6%），TPOT 51.4→36.2ms，GSM8K 在置信区间内不变；两个具体优化＝CDNA 子块化 sparse GQA prefill（128-token 块拆 SUB_K 匹配 MFMA 形状）＋单 query decode index-score 走向量化 FP32 GEMV 快路径（BLOCK_SIZE_HQ==1）。（官方技术文章 A）

### 面 2｜进化知识库机制：存什么、怎么进化、怎么注入（对照 AlphaEvolve 程序数据库）
- **GEAK 的「知识库」是三个不同物件，不是一个库**：①`perf_knowledge/` 静态参考库（~586 docs、54 算子×后端笛卡尔矩阵、SOTA registry 由卡片 frontmatter 生成、expert_skills 人工配方 opt-in 默认 off；消费契约＝「只 seed/定位候选，永不降实测基线、永不跳验证」，README 原文「may be stale, incomplete, or wrong…not decisions」）；②`knowledge/learned/` 蒸馏经验卡（kernel 层与 e2e 层各一份，按「产出证据的 gate」分工）；③`kb/` 程序库面（Python 包：identity/store_local/store_remote/retract/attest/curate）＋`e2e_store.py` warm-start 回写。（代码 A）
- **蒸馏卡（learned/）机制**：两层记忆——persistent 小集合蒸馏卡（≤40 张上限、一卡一原则 ~12-20 行、reuse key＝kernel_class·gfx、置信度 ★noise/★★单次非重叠或 ≥2 一致/★★★≥2 非重叠或 verified e2e、confirms 计数、last_seen）＋episodic（eval-dir 内 per-run 全记录，含 NULL）。**写入**：run 结束后 curate（E2E_LEARNED_KB=on 才开）——读 INDEX→merge/insert ≥★★/archive 被证伪卡。**注入**：run 中读路径＝先自成 profile 计划、后读卡交叉——**ADD-only 三铁律**（卡只能加候选、不能删候选/剪 bake-off/跳测量；测量永远裁决，卡与测量冲突→信机器、修卡；无卡可封路，「caution 是『也验证 X』，永不是『别做 Y』」）。检索是「语义的、由 reader agent 做」（索引小、行带 description；已知 INDEX 手工维护漂移问题，向生成式迁移中）。（代码 A）
- **kb/ 库面（写入/淘汰/回浮）**：store 无 DELETE——**retraction＝墓碑重写**（retained:false+retired_reason＋排序标量清零沉底＋champion 指针重指；「假旗若只打标不清标量仍霸榜」）。**attestations 台账**＝真机复跑记录（recalls 被真放上盒子跑的次数/validations 复现/failures/not_reproduced/inapplicable+history）——「validated 标志只是写者自评，三个月后读者问的是有没有别人拉出来跑成过」；单次失败是证据不是裁决，accumulated 台账才触发回撤。**curate**＝按 direction 折叠（一 idea 一 rank，runner-up 作 alternates 免占槽）。**e2e_store.py**：resolve（「这个部署有没有人调过、落在了什么配置」）+write 三梯级地址（全 benchmark 点/TP 级/precision 级——只有最粗梯级能横向比 TP4 vs TP8）；e2e 记录形态＝CONFIG（env flags/args/launch script）+值得 overlay 的 kernel 清单，patch 可选（config-only win 是真 win）；「warm-start REPLAYS：store 有可绑 overlay artifact 时重放 +85.9%」。（代码 A）
- **「越用越聪明」的实证边界**（判真锚）：learned/README 自述「cold runs（KB 出现前）也能到 ~20% e2e；KB 唯一职责是帮收敛更快/走得更远，绝不能把能干的 run 关进笼子」——KB 是加速器不是能力源；v4 文章口径「learned cards, workflow memory, and cross-run evidence for faster convergence」「cards can add candidates…cannot prune the search or override measurement」。（代码+官方文章 A）
- **对照 AlphaEvolve 程序数据库（部件语言）**：AlphaEvolve DB＝**搜索种群本身**（程序+多维分数+执行输出入库，岛模型/MAP-Elites，最优回浮 resurface 到 prompt，分数驱动保留淘汰）；GEAK KB＝**搜索旁的 advisory 先验库**（种群是 run 内 worktree 候选+round winner；KB 只 seed 候选不参与淘汰，测量裁决）。共同件：跨 run 保留/回浮（warm-start/KB replay≈resurface）、证据附着条目、多样性维护（collapse-by-direction 保 alternates≈岛多样性）。GEAK 独有件：真机 attestation 台账、无删除+墓碑回撤、两层记忆（蒸馏/episodic）、反笼子 add-only 契约；AlphaEvolve 独有件：元 prompt 进化、评估级联、多维分数共同优化论证。谱系连接：GEAK v2 曾有 GEAK-OpenEvolve（Triton-to-Triton QD/MAP-Elites，AlphaEvolve 谱系直接分支），v4 把进化搜索换成确定性 workflow+蒸馏 KB。（代码+releases A；对照为机制比对非单源事实）

### 面 3｜生产级落地路径与实证状态
- **v1 论文（2507.23194）无 production 声明**：abstract 只谈 benchmark 结果（correctness 63%、speedup 2.59×）与「promise/democratizing」愿景句；评测集＝TritonBench-modified 184 内核+ROCm 仓库 30 内核。「AMD 生产级」标签源自二手综述（微信文章⑨）转述，非论文口径。（arXiv A）
- **v4 的生产叙述分四层**：①官方产品化——AMD-AGI org 开源、ROCm 官方文档站收录（rocm.docs.amd.com/projects/geak）、v4.0/4.1/4.2 版本化发布；②AMD 内部使用声明——「For AMD, GEAK v4 accelerates model bring-up and reduces dependence on scarce manual kernel expertise」（model bring-up 场景，无规模数字）；③**持续运行实证＝两 tier CI**（v4.1 起：GPU-free 静态 tier+每 enrolled model 在 ROCm 真机端到端跑、按 exit code/执行状态/baseline integrity 判定、每模型独立 verdict）——是持续自动化 e2e 优化，但形态是 CI 不是生产流量；④**上游生产路径**＝GLM-5.2 AgentX 博客（2026-11-04）：AMD 团队用 HyperLoom 辅助内核优化（11 候选阶段/16h 有效内核工时，fused indexer 91.24→35.56µs 即 2.57×、内含工程 roofline 33.1%→85.0%）→17+ 个 SGLang 合入 PR（DSA top-k/MLA fusion/fused indexer/sparse MLA/HiCache host pointers 等）→SemiAnalysis InferenceX AgentX 公开基准验证（393 个真实 Claude Code session、median 142K input tokens、3600s 窗口）→公开 versioned recipe。峰值 per-GPU 吞吐 1.72×、P90 ITV 2.46×（两个 deployed configuration 对比，作者自标「非受控 A/B」）；fused indexer 单改动受控 A/B P90 ITL −6.7%（PR #38583）。（官方博客+代码 A）
- **判真结论**：「生产先例」实证形态＝AMD 真实 serving 工程管线在运行（持续 CI 优化＋上游合入主干＋公开基准部署配置＋versioned recipe），**未见「GEAK 持续跑在客户生产流量」的一手声明**；GEAK e2e_workflow 本体数字全部 AMD 自报（性能实验室口径），无第三方复现。GLM-5.2 数字是 HyperLoom 辅助管线的结果（相关系统，非 GEAK e2e_workflow 直接端到端）。（A 综合判读，已标注为判读）

### 面 4｜与 vLLM/SGLang 的集成方式（优化物形态）
- **引擎不可知适配器层**：serving 栈不内置，`args.backend`（sglang|vllm|atom，默认 sglang）选 `scripts/adapters/<backend>.sh`，契约＝adapter_launch/adapter_health/adapter_bench（+可选 adapter_profile_window：在已运行、warm、负载中的 server 上开 profiler 窗口，取真实稳态 prefill+decode 混合而非冷启 ramp；adapter_prepare_measurement：JIT/编译态不稳则计时 fail closed）。加新栈＝加一个 adapter，角色与编排零改动。基准工具 bench_e2e.sh 为 backend 无关 dispatcher。（代码 A）
- **优化物三形态**（按成本递增）：①config-only（launch flags/env vars/launch script，Config Tuner 层，无源码改动）②**可逆 overlay 源码替换**——PYTHONPATH overlay+sitecustomize.py rebind dispatch attr（如重绑 `aiter:gemm_a8w8_blockscale`「attr vLLM re-imports per call; site-packages untouched」），**never editing installed packages**，可整体回退；③author from scratch（热 op 无现成可编辑实现时，kernel 层 author mode 写全新基线再 rebind；FlyDSL 为 dense/量化 GEMM 首选 author 目标，Triton 恒可行）。（代码 A）
- **交付物**（run 落盘 `exp/e2e_<model>_<ts>/`）：final bundle＝overlay+`final_patch.diff`+`final_launch.sh`（自包含启动脚本，替代原启动命令即用优化配置）+final_report.md+architect_report.md+各阶段工件；kernel 层交付＝verified `final_patch.diff`+geomean。模型权重与已装包全程只读。（文档+代码 A）
- **上游化持久路径**：focused PR 进 SGLang/vLLM 主干（带测速证据/ablation/CI review/硬件 guard，探索变体不进 final patch）；AgentX 博客实证该路径全链走通（17+ 合入 PR+InferenceX 公开 recipe 审查「每 sweep 用公开发布镜像+全参数文档化」）。（官方博客 A）

### 面 5｜美团映射素材（客观对应物罗列，不替 human 设计）
- **端到端优化对象选层对应物**：Amdahl 优先分公式 `pct_gpu_time × achievable_speedup`（含 roofline 估算 achievable）；便宜杠杆先行阶梯 flag→env→源码→author；三预算模式 fast/default/deep；roofline advisory 路由（saturated 内核改 routed toward fewer bytes 而非 drop，且「roofline 预测 attainable=1.0 但实测 +26.9%」的反例已入卡——launch-overhead 型收益对 roofline 不可见）；preflight 环境自检+fail-closed 身份校验。
- **验证器家族对应物**：warm-server interleaved A/B+噪声带 gate（0.5%/4%）+非重叠分布 gate+engagement proof+greedy byte parity／gsm8k accuracy gate（lossy 改动）；Director 独立终判仲裁；三生命周期（warm/isolated/legacy）禁混比；真实 trace replay（AgentX 393 sessions）当生产负载。
- **知识库家族对应物**：两层记忆（蒸馏卡 keyed kernel_class·gfx+episodic 全记录含 NULL）；三梯级 warm-start 地址（benchmark 点/TP/precision）；attestation 真机台账+墓碑回撤+champion 重指；Hyperloom 远端 KB 集成（检索/warm-start/持续学习）；add-only 反笼子契约（卡只加候选不剪枝）。
- **组织形态对应物**：确定性控制面（JS workflow）×LLM 判断面分离；分形两层递归（system 层调 unchanged kernel 层，防 kernel 层质量回归）；两 tier CI（静态+真机 e2e）；「unit of value is the model, not the kernel」的总目标设计。
- 以上仅为事实对应；美团版取舍（选层深度/验证器强度/KB 形态）判断权在 human。

## R7 上报项
- [谱系混标]：research_1 行 13/15 把「2507.23194＋e2e_workflow＋生产级」混为一代——缺：第 10 课引用口径的拆正裁决（v1 论文≠v4 系统，teach 引用时须分开标注）。
- [生产级措辞]：无「GEAK 持续运行于客户生产流量」的一手声明，实证形态为 CI+评测+上游 PR——缺：human 对「生产先例」措辞强度的裁决。
- [v4 无同行评审]：e2e_workflow 无 arXiv/会议版本（截至 2026-11-04），全部数字 AMD 自报。
- [Hyperloom×GEAK 边界]：两独立 repo/文档站；v4.2 KB 集成+bench 协议互引，但相互依赖与组织边界细节未全载。
- [统计协议规格]：0.5% 噪声带来源与 interleaved repeats 数（≥5-7）只见于代码默认+方法卡，无正式规格文档。
- [learned/INDEX 漂移]：repo 自认手工 INDEX 曾漏卡（读卡前须 ls 兜底），迁移生成式中。
- [ATOM backend]：AMD 内部 serving 栈，公开资料少，无法核验其形态。

## Gaps
- GEAK e2e_workflow 数字无第三方复现；GLM-5.2 AgentX 是相关系统（HyperLoom）管线的部署结果，不能当作 GEAK 本体的独立复现。
- expert_skills 卡内容未逐卡尽调（默认 off 的 opt-in 功能）；Hyperloom repo 未代码级核验（仅文档/博客级）。
- v4 文章 Fig.1-4 原图未读（fetch 文本已含全部引用数值）。
- v1 论文正文（除摘要外）未逐节核对——本材料对 v1 的结论限于摘要与 Releases 口径。

## 立场层（新断言候选——已按三栏格式追加 values.md 待裁决 12/13，human 裁决栏留空）
- **待裁决 12**：v4 文章「For customers, it points toward faster, more cost-efficient inference on AMD GPUs without requiring every team to build its own dedicated kernel engineering organization.」——对客户价值与成本结构的竞争性收益声明（无口径无复现）。
- **待裁决 13**：v1 论文摘要「…democratizing access to expert-level kernel performance.」——愿景性定位句（论文证据只到 benchmark 级）。
