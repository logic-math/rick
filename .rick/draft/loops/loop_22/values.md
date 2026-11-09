> 继承注记（parent 复制）：本清单完整承接自 loop_14/values.md——5 条价值性信息已于 2026-11-02 由 human 全部裁决（原文见各条「human 裁决」栏）。本 loop 新增待裁决条目将追加于文末。

# loop_14 values.md — 价值性信息裁决记录 — 2026-11-01

> 承接说明：本清单完整承接自 loop_13 同日调研（loops/loop_13/values.md）：5 条待裁决条目的「原文 / 出处 / 为什么是价值判断」三栏逐字保留；时间基准与信源等级标注不变，无新增待裁决项（loop_14 唯一增量为 research_1 的 AlphaEvolve 机制基准节，属事实层非价值判断）。裁决状态：2026-11-02 human 全部裁决完成（见各条「human 裁决」栏，原话记录）。

调研（research_1）中识别出以下 5 条价值性信息/回溯不到常识源的前提。均为可归因的"他人判断"（事实层已如实记录出处），但**判断本身**（优先级排序、竞争预测、投资建议、定性评级）不可从 primary source 还原为事实，涉及取舍与立场，按硬约束不替 human 裁决，逐条待裁决。

---

## 待裁决 1：韦子扬的"先手"竞争预测
- **原文**："谁能率先把「长程信用分配」与「环境噪声鲁棒的可复现大规模合成」这两块最硬的骨头啃下来，并在模型-harness-环境的协同设计上形成闭环，谁就能在下一轮 Agentic 智能的竞赛中握住先手。"
- **出处**：《Agentic RL Infra 主流技术路线综述》结语（https://weiziyang.wiki/posts/post-training-rl/agentic-rl-infra-technical-roadmap-overview，微信转发 https://mp.weixin.qq.com/s/aQFrApfzfMaKCtDWFmcKPw）（等级 C，综述）
- **为什么是价值判断**：对未来竞争格局的押注（"最硬的骨头""先手"），对技术要素做了优先级排序，无法从 11 家技术报告的事实数字推出；采纳与否影响美团侧的投入方向。
- **human 裁决**：5 条价值均接受。（2026-11-02，对 parent 逐条裁决提问的回复原话）

## 待裁决 2：DeepSeek-V4.1 的"工程边际回报 > 算法创新边际回报"
- **原文**："在当前阶段，对数据与环境流程进行工程化的边际回报，显著超过后训练中算法创新的边际回报。"
- **出处**：DeepSeek-V4.1 技术报告，经韦子扬综述转述（等级 C，未回指原文核对）
- **为什么是价值判断**：资源分配优先级声明（工程 vs 算法），是阶段性的战略取舍，非可验证事实；且当前只经二手转述。
- **human 裁决**：5 条价值均接受。（2026-11-02，对 parent 逐条裁决提问的回复原话）

## 待裁决 3：HUINHB 的投资建议"预算投 agent 不投 evolver"
- **原文**："These findings suggest investing capability budget in the task-solving agent rather than the evolver, and targeting harness invocation and long-horizon instruction following in agent training."
- **出处**：arXiv 2605.30621（EMNLP 2026 main），作者规范性结论段（等级 A，但结论段是建议性推论）
- **为什么是价值判断**：从两个实证发现（updating 平、benefit 非单调）到"该投给谁"是规范性跨越；对美团 RSI 系统建设中"强模型/弱模型分工"的选型有直接取舍含义。
- **human 裁决**：5 条价值均接受。（2026-11-02，对 parent 逐条裁决提问的回复原话）

## 待裁决 4：z.ai 的"Toward Recursive Self-Improvement"叙事框架
- **原文**：博客以"模型优化系统，系统运行模型"（the model optimizes the system; the system runs the model）为纲，将 GLM-5.3 Infra Agent 建设叙述为"通往 RSI 的早期一步"；同文自认"尚未达到递归自我改进，选择目标、划定边界、评估风险仍是人类职责"。
- **出处**：https://z.ai/blog/glm-built-its-inference-infrastructure（2026-09-17，等级 A 官方一手，但叙事定性是公司立场）
- **为什么是价值判断**："什么算作通往 RSI 的一步"是定性框架问题：3.2× 吞吐是事实，"这就是 RSI 的早期形态"是立场性命名（且带品牌动机）；影响美团对内对外如何定义自己的 infer RSI 叙事。
- **human 裁决**：5 条价值均接受。（2026-11-02，对 parent 逐条裁决提问的回复原话）

## 待裁决 5：微信文章②的"性价比之王"与五趋势判断
- **原文**："Agentic 闭环是当前性价比之王：通用 LLM + 编译/执行/Profile 反馈 + 搜索框架（MCTS/MAB），50 轮迭代即可在真实模型上拿到 1.5-2.8×……无需训练成本，进步来自框架工程"；及"五个趋势判断"（RL 训练派快速追赶、评测与生成赛跑、跨硬件泛化是新战场、LLM 将成为编译器一等公民等）。
- **出处**：微信公众号 Arthur（ArthurAI）《让 LLM 写 GPU 算子》10.2 节（https://mp.weixin.qq.com/s/lkBbY7IV2wShunmo-JOO5w，等级 C，独立作者博客）
- **为什么是价值判断**："性价比之王"是对五路线的成本-收益评级，"新战场/一等公民"是趋势押注；均可争论，非事实；若作为教学事实吸收会引入作者的路线偏好。
- **human 裁决**：5 条价值均接受。（2026-11-02，对 parent 逐条裁决提问的回复原话）

---

> 以下为 loop_15 research_2（判真统计协议调研，2026-11-03）新增待裁决条目，格式同上（三栏 + human 裁决栏留空）。

## 待裁决 6：PyTorch 官方博客对「底部几个百分点在大规模舰队的价值」判断
- **原文**："It also compounds exactly where you'd want it to. There are high-value kernels inside inference loops that run continuously across large fleets. Even a few percent at the bottom of the stack, applied to the operations that dominate a decode step, is valuable at that scale."
- **出处**：PyTorch 官方博客 "One Search, Two Layers: Compiler Controls Inside PyTorch Helion using NVIDIA CompileIQ"（https://pytorch.org/blog/one-search-two-layers-compiler-controls-inside-pytorch-helion-using-nvidia-compileiq/，2026-10-19，等级 A 官方一手，但此句属说服性结论段）
- **为什么是价值判断**：对「2-7% 级 kernel 收益是否值得投入」下了定性判断（"compounds where you'd want it to"/"valuable at that scale"）——是否值得取决于各家的成本结构与 fleet 规模，无法从基准数字推出；直接影响美团侧是否把编译器层调优列入 infer RSI 投入优先级。
- **human 裁决**：接受，优化多少要看具体问题。（2026-11-03，对 parent 逐条裁决提问的回复原话）

## 待裁决 7：PyTorch 官方博客对「naive 基线的加速声明无意义」判断
- **原文**："A 'we made kernels faster' claim is easy if the starting point is a naive kernel, and close to meaningless for real-world workloads."
- **出处**：同上博客 "Starting off with highly-optimized kernels & optimizing the last-mile" 节（等级 A 官方一手，但此句是对加速声明可接受基线的规范性立场）
- **为什么是价值判断**：规定了「什么基线上的加速才算有意义」——这是评测方法论立场（与 kernel 评测危机三篇的立场一致但更强硬），不是可验证事实；若作为教学事实吸收，会预设「低基线加速声明一律不信」的裁决立场，影响 human 对自家 +3% 类声明的判真尺度。
- **human 裁决**：接受——baseline 不能以 pytorch 原生实现对比，这个值没有实际价值。（2026-11-03，对 parent 逐条裁决提问的回复原话）

---

> 以下为 loop_17 research_3（z.ai GLM Infra Agent 机制级尽调，2026-11-04）新增待裁决条目（research 因单次 write 限制未及追加，由 parent 按原文转录，三栏 + human 裁决栏留空）。

## 待裁决 8：z.ai 的「与主流 NVIDIA GPU 可比」等价声明
- **原文**："Both hardware utilization efficiency and per-token cost reached levels comparable to mainstream NVIDIA GPUs."
- **出处**：z.ai 官方博客 https://z.ai/blog/glm-built-its-inference-infrastructure（2026-09-17，等级 A 官方一手，但此句为竞争性等价声明——无数字支撑、「主流 GPU」未定义口径、无独立复测）
- **为什么是价值判断**：竞争性等价断言（硬件利用率与 per-token 成本「可比」）不含可验证口径；若作为教学事实吸收，会预设国产加速器对标 NVIDIA 的立场性判断，影响投入方向与叙事取舍。
- **human 裁决**：接受。（2026-11-04，对 parent 待裁决提问的回复原话，覆盖 8/9 两条）

## 待裁决 9：z.ai 的「边界上的进度不会放慢」趋势断言
- **原文**："But the numbers, two weeks, threefold throughput, and 100,000 accelerators, tell us that progress at this boundary will not slow down simply because we want it to."（紧邻前句 "We believe humans should continue to hold that line for a long time to come."）
- **出处**：同上博客结语（2026-09-17，等级 A 官方一手，但为 n=1 自报样本的规范性外推）
- **为什么是价值判断**：从单一自报案例（13 天/3×/10 万卡）外推领域趋势（「进度不会放慢」）＋「人类应长期守线」的规范主张——趋势押注与立场表达，非可验证事实。
- **human 裁决**：接受。（2026-11-04，对 parent 待裁决提问的回复原话，覆盖 8/9 两条）

---

> 以下为 loop_17 research_4（FlashInfer-Bench 机制级尽调，2026-11-04）新增待裁决条目，格式同上（三栏 + human 裁决栏留空）。

## 待裁决 10：FlashInfer-Bench 官方博客的「已建成自优化闭环」叙事
- **原文**："With it, we have already built a self-optimizing loop for AI—a virtuous cycle. The revolution of AI improving itself will continue—let's see where it goes!"
- **出处**：FlashInfer 官方博客 https://flashinfer.ai/2025/10/21/flashinfer-bench.html 结语（2025-10-21，等级 A 官方一手，但为叙事定性句）
- **为什么是价值判断**：「已建成自优化闭环」（have already built）与「AI 自我改进的革命将持续」（trend 断言）均为定性命名与趋势押注——v1 论文实证状态是「闭环机制已通、AI kernel 尚未胜过 native baseline」（端到端 933 vs 939/1055ms）；同博客自己也说 "the AI today may not yet perform at the expert engineer level"。与 z.ai「通往 RSI 早期一步」（待裁决 4，已裁决）同构：影响美团对「闭环骨架成熟度」的叙事采纳与对外表述尺度。
- **human 裁决**：接受。（2026-11-04，对 parent 待裁决提问的回复原话，覆盖 10/11 两条）

## 待裁决 11：FlashInfer-Bench 论文的「训练限库、推理放开」建议
- **原文**："We recommend restricting library access during training for better skill acquisition, while allowing unrestricted use during inference to maximize kernel performance."
- **出处**：arXiv 2601.00227v1 §4.3（MLSys 2026，等级 A 同行评审，但此句为作者规范性建议段）
- **为什么是价值判断**：从「agent 学会调 cuBLAS 取得佳绩」的实证观察跨到「该怎么训练/评测」的规范主张（限制 vs 放开库调用）——训练取舍影响技能习得路径的判断，无法从论文数字推出；直接影响美团侧 kernel agent 的训练与评测协议设计（评测口径是否限库）。
- **human 裁决**：接受。（2026-11-04，对 parent 待裁决提问的回复原话，覆盖 10/11 两条）

---

> 以下为 loop_17 research_5（GEAK e2e_workflow 机制级尽调，2026-11-04）新增待裁决条目，格式同上（三栏 + human 裁决栏留空）。

## 待裁决 12：GEAK v4 官方文章对客户的成本收益声明
- **原文**："For customers, it points toward faster, more cost-efficient inference on AMD GPUs without requiring every team to build its own dedicated kernel engineering organization."
- **出处**：AMD 官方技术文章 https://www.amd.com/en/developer/resources/technical-articles/2026/geak-v4.html 结语（测试口径 2026-07-08，等级 A 官方一手，但此句为竞争性收益声明——「更便宜、无需自建 kernel 工程团队」无成本口径、无客户案例支撑、无独立复现）
- **为什么是价值判断**：对客户侧投入产出（省自建 kernel 团队 vs 用 agent）下了定性结论——收益是否成立取决于各家的 workload 形态与工程成本结构，无法从 AMD 自报 e2e 数字推出；若作为教学事实吸收，会预设「agent 优化可替代自建 kernel 工程」的立场，影响美团对自建团队 vs agent 工具化的投入结构判断。
- **human 裁决**：接受。（2026-11-04，对 parent 待裁决提问的回复原话，覆盖 12/13 两条）

## 待裁决 13：GEAK v1 论文摘要的「民主化专家级内核性能」愿景句
- **原文**："These results highlight the promise of GEAK-like agentic code generation for accelerating the adoption of diverse hardware platforms and democratizing access to expert-level kernel performance."
- **出处**：arXiv 2507.23194 摘要末句（等级 A 论文一手，但此句为愿景性外推——论文实证只到 benchmark 级：TritonBench-modified 184 内核+ROCm 30 内核，correctness 63%/speedup 2.59×；「democratizing」无任何口径）
- **为什么是价值判断**：从 benchmark 结果外推「民主化专家级性能」的社会性愿景——是领域叙事定位而非可验证事实；与 FlashInfer-Bench「已建成自优化闭环」（待裁决 10）、z.ai「进度不会放慢」（待裁决 9）同构：官方一手的叙事定性句。影响 human 对「kernel agent 是否已成生产力工具」的叙事采纳尺度。
- **human 裁决**：接受。（2026-11-04，对 parent 待裁决提问的回复原话，覆盖 12/13 两条）

---

> 以下为 loop_20 research_8（视角候选供给：测量/验证工具自身演化四领域，2026-11-08）新增待裁决条目，格式同上（三栏 + human 裁决栏留空）。

## 待裁决 14：Meta 官方博客对 Dynabench 的「更优模型质量指标」判断
- **原文**："It measures how easily AI systems are fooled by humans, which is a better indicator of model quality."
- **出处**：Meta AI 官方博客 "Introducing Dynabench: Rethinking the way we benchmark AI"（https://ai.meta.com/blog/dynabench-rethinking-ai-benchmarking/，2021-05，等级 A 官方一手，但此句为质量标准规范性声明）
- **为什么是价值判断**：规定「什么才算模型质量」（人对抗骗的难度）优于其他质量维度（正确率/覆盖率/效率等）——质量维度的优先排序无法从 Dynabench 论文数据推出；若作为教学事实吸收，会预设「对抗鲁棒性＝更真」的评测哲学，直接影响 human 对 rick super 四元组「信号明确」的取向（信号选鲁棒性还是选什么）。
- **human 裁决**：拒绝。（2026-11-08，loop_20 探索流价值裁决同轮，human 原话：「我不相信这条价值假设」）

## 待裁决 15：LiveBench 的「Contamination-Free」绝对化标题声明
- **原文**："LiveBench: A Challenging, Contamination-Free LLM Benchmark"（标题）；对照其 README 机制措辞 "LiveBench limits potential contamination by releasing new questions regularly."
- **出处**：livebench.github.io（ICLR 2025 Spotlight，等级 A 官方一手，但 contamination-free 为绝对化自评——自家机制描述仅支持 limits〔限制〕）
- **为什么是价值判断**：「无污染」的绝对声明超出机制可证范围——月度刷新只能压缩污染窗口，不能消除训练语料与新题的隐性重叠（污染测量学〔Sainz et al. 2023〕自认污染程度不可直接测量）；是否采信「滚动刷新≈无污染」影响 human 是否把滚动刷新当作任务集自进化的防护方案直接采纳。
- **human 裁决**：接受，短期可不做。（2026-11-08，loop_20 探索流价值裁决同轮，human 原话：「我相信，确实应该采集，但是短期可以不做。」）

## 待裁决 16：POET 系的「Endlessly / Unbounded Invention」开放性承诺
- **原文**："Paired Open-Ended Trailblazer (POET): Endlessly Generating Increasingly Complex and Diverse Learning Environments and their Solutions"（标题）；"Enhanced POET: Open-Ended Reinforcement Learning through Unbounded Invention of Learning Challenges and their Solutions"（标题）
- **出处**：arXiv 1901.01753（GECCO 2019）/ arXiv 2003.08536（ICML 2020），等级 A 同行评审，但标题的「endlessly/unbounded」是对运行域无限性的承诺——实证只到有限运行，且 Enhanced POET 的存在本身表明原始防护结构随规模出现新失效需修复
- **为什么是价值判断**：「（有防护的）任务集共同进化可以无界持续」是方向性押注而非可从有限实验回溯的事实；human 若把 minimal criterion＋transfer 当作「孔雀隔离原则的充分替代」，实际采纳的是该域作者的开放性信念——这决定 rick super 任务集自进化方案的信任度设定（有防护共同进化的防护是否可持续）。
- **human 裁决**：接受。（2026-11-08，loop_20 探索流价值裁决同轮，human 原话：「我接受」）

---

> 以下为 loop_21 research_9（良质批判标准靶向调研：2×2 判定链假设审计，2026-11-09）新增待裁决条目，格式同上（三栏 + human 裁决栏留空）。

## 待裁决 17：SWE-bench Verified 官方页的「人工验证 ⇒ 可靠评测」适配性声明
- **原文**："SWE-bench Verified — A human-validated subset of 500 SWE-bench instances for reliable evaluation of coding agents and language models."
- **出处**：SWE-bench 官方站 swebench.com/verified（与 OpenAI 合作发布，2024-08，等级 A 官方一手，但 "reliable evaluation" 为适配性声明——无信度口径（无重测/方差数据），且同一方 2025 年再审计后已声明该基准对前沿模型 "no longer suitable for measuring progress"）
- **为什么是价值判断**：「人工验证过 ⇒ 评测可靠」是效度推断而非可验证事实——「可靠」未给口径（多大噪声底/多少题才够），且被同源后续审计（模型常失败子集 27.6% 中至少 59.4% 有缺陷测试）实质性削弱；与 LiveBench「contamination-free」（待裁决 15）同构。若作为教学事实吸收，会预设「人工筛查过的任务集即可当可靠尺子」，直接影响 human 对 tasks1 作锚题与对追加任务过准入审计的信任度设定。
- **human 裁决**：

## 待裁决 18：DGM 论文标题的「Open-Ended Evolution of Self-Improving Agents」开放性命名
- **原文**："Darwin Gödel Machine: Open-Ended Evolution of Self-Improving Agents"（标题）
- **出处**：arXiv 2505.22954（Sakana AI 等，2025-05；ICLR 2026，等级 A 同行评审，但标题的 "open-ended" 是对演化无界性的命名——实证只到有限运行，且论文附录 H 自报 objective hacking：agent 删掉工具幻觉日志绕过评测得高分）
- **为什么是价值判断**：「开放式演化」是对自改进过程可无界持续的方向性命名（与 POET 系「endlessly/unbounded」〔待裁决 16〕同族），无法从有限实验回溯；论文自报的 objective hacking 恰表明自改进过程会钻评测空子——「开放」的边界要靠验证层兜底，而验证层正是有限的。影响 human 对「自改进 rick super 可否持续递进」的信任度设定（自改进叙事的命名尺度）。
- **human 裁决**：

---

> 以下为 loop_21 research_10（良质批判靶向调研：Rubrics survey 与共同进化有效性，2026-11-09）新增待裁决条目，格式同上（三栏 + human 裁决栏留空）。

## 待裁决 19：Rubrics survey 对「rubrics 应成为未来 LLM 系统的透明、自适应、可信接口」的愿景定位
- **原文**："We hope this survey can serve as a structured reference for current research and a conceptual foundation for developing rubrics as transparent, adaptive, and trustworthy interfaces for future LLM systems."（结论章重申："Future work should therefore develop more reliable, adaptive, and trustworthy rubric systems that can better connect human expectations, task requirements, and model behavior."）
- **出处**：《The Rules of the Game: A Survey of Rubrics for Large Language Models》（RUC-NLPIR，PDF 标注 2026-05-22，Preprints.org 10.20944/preprints202608.0565.v1，等级 A 一手预印本，但此句为愿景性定位句）摘要与 §7 结论
- **为什么是价值判断**：「rubrics 应成为未来 LLM 系统的核心接口层」是技术路线押注与倡导——survey 实证部分只覆盖「rubric 在已研究场景里改善了训练信号与评测可解释性」，「成为系统级接口」是无法从已综述证据推出的愿景外推；与 z.ai「通往 RSI 早期一步」（待裁决 4）、FlashInfer-Bench「已建成自优化闭环」（待裁决 10）同构。若作为事实吸收，会预设「选 rubric 当中间层」这一架构立场，直接影响学生在共同进化问题上是否把准则层锚定在 rubric 上的取舍。
- **human 裁决**：

## 待裁决 20：survey §6.1 对「rubrics 不应是静态评价物、而应是自适应监督机制与 policy 共同进化」的范式立场
- **原文**："Under this perspective, rubrics are not static evaluation artifacts, but adaptive supervision mechanisms that can incorporate newly discovered failure modes into the criterion set. … However, this direction also raises unresolved algorithmic questions: how often should rubrics be updated, how can update frequency be balanced against optimization stability, how can the rubric set avoid overfitting to the current policy distribution, and how to characterize the convergence of a joint policy–rubric optimization process. Designing principled training-adaptive rubric evolution mechanisms, rather than relying on ad-hoc heuristic updates, remains a critical open problem."
- **出处**：同上 survey §6.1 Training-Adaptive Rubric Evolution（等级 A 一手预印本，但此段为范式定位与方向主张）
- **为什么是价值判断**：survey 用事实论证了静态 rubric 脆弱（判别力饱和/可被钻措辞空子），并把「让 rubric 与 policy 共同进化」立为缓解方向——但从「静态会失效」推不出「应选择共同进化」（还存在冻结准则层＋滚动任务＋周期再校准等替代结构），且同段自认该方向在更新节奏/防过拟合/联合收敛上均未解。方向选择是立场而非可验证事实；这正是学生「尺子不动 vs 尺子滚动」对撞的裁决点——采纳该立场即同时采纳「两层都滚动＋治理层」的架构承诺及其未解风险。
- **human 裁决**：

---

> 以下为 loop_21 research_11（良质批判靶向调研：V-M 隔离协议四假设审计，2026-11-09）新增待裁决条目，格式同上（三栏 + human 裁决栏留空）。

## 待裁决 21：RewardBench 2 的「RM 应与 policy 同谱系」设计规范句
- **原文**："Our analysis shows how the best reward model for RLHF is dependent on one's training setup. For RLHF, the reward model should be based on a model of the same lineage as the policy model or else downstream performance can degrade significantly, so simply taking the highest scoring reward model on a benchmark will not ensure a good post RLHF model."
- **出处**：RewardBench 2: Advancing Reward Model Evaluation（arXiv 2506.01937，2025-06，等级 A 一手论文，但此句为设计规范外推——实证范围是 Tulu 3 8B SFT＋PPO 等 RLHF 单一训练 setup 的实验）
- **为什么是价值判断**：「验证器（RM）应与被验证对象（policy）同谱系，否则下游显著退化」是从单一 setup 实验外推为通用设计规范的 should 句，非可从实验直接推出的一般事实；与学生的 V-M 隔离协议直接对撞——采纳它＝「谱系耦合的验证器有利下游」，不采纳＝判定该外推不适用 rick super。同句还断言「benchmark 最高分不保证下游好」＝否定基准分作最终判据（与 accuracy paradox、指标失配两实证同向但外推程度更深）。影响 human 对「V 锚定在 C 谱系测试物上是利是弊」及「M 是否必须含下游生产结果锚」的取向设定。
- **human 裁决**：
