# loop_13 values.md — 价值性信息待裁决 — 2026-11-01

调研（research_1）中识别出以下 5 条价值性信息/回溯不到常识源的前提。均为可归因的"他人判断"（事实层已如实记录出处），但**判断本身**（优先级排序、竞争预测、投资建议、定性评级）不可从 primary source 还原为事实，涉及取舍与立场，按硬约束不替 human 裁决，逐条待裁决。

---

## 待裁决 1：韦子扬的"先手"竞争预测
- **原文**："谁能率先把「长程信用分配」与「环境噪声鲁棒的可复现大规模合成」这两块最硬的骨头啃下来，并在模型-harness-环境的协同设计上形成闭环，谁就能在下一轮 Agentic 智能的竞赛中握住先手。"
- **出处**：《Agentic RL Infra 主流技术路线综述》结语（https://weiziyang.wiki/posts/post-training-rl/agentic-rl-infra-technical-roadmap-overview，微信转发 https://mp.weixin.qq.com/s/aQFrApfzfMaKCtDWFmcKPw）（等级 C，综述）
- **为什么是价值判断**：对未来竞争格局的押注（"最硬的骨头""先手"），对技术要素做了优先级排序，无法从 11 家技术报告的事实数字推出；采纳与否影响美团侧的投入方向。
- **human 裁决**：

## 待裁决 2：DeepSeek-V4.1 的"工程边际回报 > 算法创新边际回报"
- **原文**："在当前阶段，对数据与环境流程进行工程化的边际回报，显著超过后训练中算法创新的边际回报。"
- **出处**：DeepSeek-V4.1 技术报告，经韦子扬综述转述（等级 C，未回指原文核对）
- **为什么是价值判断**：资源分配优先级声明（工程 vs 算法），是阶段性的战略取舍，非可验证事实；且当前只经二手转述。
- **human 裁决**：

## 待裁决 3：HUINHB 的投资建议"预算投 agent 不投 evolver"
- **原文**："These findings suggest investing capability budget in the task-solving agent rather than the evolver, and targeting harness invocation and long-horizon instruction following in agent training."
- **出处**：arXiv 2605.30621（EMNLP 2026 main），作者规范性结论段（等级 A，但结论段是建议性推论）
- **为什么是价值判断**：从两个实证发现（updating 平、benefit 非单调）到"该投给谁"是规范性跨越；对美团 RSI 系统建设中"强模型/弱模型分工"的选型有直接取舍含义。
- **human 裁决**：

## 待裁决 4：z.ai 的"Toward Recursive Self-Improvement"叙事框架
- **原文**：博客以"模型优化系统，系统运行模型"（the model optimizes the system; the system runs the model）为纲，将 GLM-5.3 Infra Agent 建设叙述为"通往 RSI 的早期一步"；同文自认"尚未达到递归自我改进，选择目标、划定边界、评估风险仍是人类职责"。
- **出处**：https://z.ai/blog/glm-built-its-inference-infrastructure（2026-09-17，等级 A 官方一手，但叙事定性是公司立场）
- **为什么是价值判断**："什么算作通往 RSI 的一步"是定性框架问题：3.2× 吞吐是事实，"这就是 RSI 的早期形态"是立场性命名（且带品牌动机）；影响美团对内对外如何定义自己的 infer RSI 叙事。
- **human 裁决**：

## 待裁决 5：微信文章②的"性价比之王"与五趋势判断
- **原文**："Agentic 闭环是当前性价比之王：通用 LLM + 编译/执行/Profile 反馈 + 搜索框架（MCTS/MAB），50 轮迭代即可在真实模型上拿到 1.5-2.8×……无需训练成本，进步来自框架工程"；及"五个趋势判断"（RL 训练派快速追赶、评测与生成赛跑、跨硬件泛化是新战场、LLM 将成为编译器一等公民等）。
- **出处**：微信公众号 Arthur（ArthurAI）《让 LLM 写 GPU 算子》10.2 节（https://mp.weixin.qq.com/s/lkBbY7IV2wShunmo-JOO5w，等级 C，独立作者博客）
- **为什么是价值判断**："性价比之王"是对五路线的成本-收益评级，"新战场/一等公民"是趋势押注；均可争论，非事实；若作为教学事实吸收会引入作者的路线偏好。
- **human 裁决**：
