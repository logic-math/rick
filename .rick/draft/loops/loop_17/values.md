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
