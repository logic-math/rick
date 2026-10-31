# 生产级 LLM 导师：自由文本回答 → 掌握度分档（离线知识整理）
（环境无外网，未调用任何搜索；全部内容出自内部记忆，所有 URL 未在线核验）

## 1. LearnLM 五原则（arXiv:2412.16429，Google 技术报告，2024）【B：官方一手文档】
五原则名称：培养好奇心与求知欲（Foster curiosity and wonder）；促进主动学习（Encourage active learning）；激发高阶思维（Drive higher-level thinking）；关怀学习者福祉（Support learner well-being）；个性化学习体验（Personalize the learning experience）。
与「评估学生回答」相关的操作化规则：主动学习节规定「不直接给答案，而是给提示/脚手架」「一次只问一个问题」「让学生先尝试再介入」；反馈规则为「具体而非笼统、正负反馈平衡、聚焦可改进点」；高阶思维节要求追问建立在学习者先前回答之上，引导其自己重构结论；个性化节要求先探测学习者水平再调问题难度。报告附 bad/good/better 示例并由专家按 rubric 评分验证。URL 未在线核验。

## 2. Khanmigo 泄露/报道提示词（2023 年华盛顿邮报等报道）【C：泄露文本+媒体转述】
规则要点：一次只问一个问题；绝不直接给答案，学生索要答案时反问引导；答对才明确肯定，答错不说「你错了」而以提问引导；优先提示（hint）并先询问是否需要；回复保持简短；掌握判断依赖连续追问链（追问-回答-再追问）而非一次性打分。另有教师端「Exercise Explorer」分步批改（逐步判对错并引导）的报道，属转述非一手提示词。不确定：泄露版本是否含隐藏置信度输出字段——记忆中存在但无法核验，明写不确定。URL 未在线核验。

## 3. Khan Academy 掌握度分档（官方帮助中心/博客）【B：官方文档】
四档：Attempted（尝试过该技能）→ Familiar（基本熟悉概念）→ Proficient（熟练）→ Mastered（精通）。
升级：技能内答对题目累积「掌握分」，达到阈值逐级升档；Course Challenge / Unit Test 按正确率可一次升到 Proficient 甚至 Mastered。
降级：后续错误作答会使等级回落（练习与测验会重估已得等级）。不确定：各档具体分数线（记忆有 70/100/130 之说，无法确证）；已 Mastered 技能是否会被降级亦不确定。URL 未在线核验。

## 4. AutoTutor expectation 匹配机制（Graesser 等，系列同行评审论文）【A：同行评审】
机制：每题预设约 2-5 条 expectation（理想答案要素）及常见 misconception；学生每轮回答与各 expectation 计算 LSA 余弦相似度，超过阈值即视为「已覆盖该 expectation」；命中 misconception 或整体低相似则触发纠错性即时反馈。
五步梯度（five-step frame）：①主问题 → ②学生回答 → ③简短即时反馈（正/中/负）→ ④脚手架序列（hint → prompt → 让学习者用自己的话拼出 expectation）→ ⑤导师拼接完整 expectation 收尾。
阈值：LSA 余弦约 0.4-0.5，不同论文/语料取值不一；具体数值不确定，方向可信：过低误判、过高漏检。URL 未在线核验。

## 5. GenTutor 与 Socratic by Google【C：线索】
GenTutor：偶见于文献题名（疑为基于遗传算法或生成式模型的导师系统），未见公开 rubric 或掌握度分档文档；其系统细节与是否存在均不确定，不编造。
Socratic by Google（2018 上线，约 2020 起停止维护）：定位为解题与步骤讲解工具，判定依赖 OCR+内置求解器对答案对错，未公开自由文本 rubric 或分档标准。URL 未在线核验。

## 结论
1. LearnLM 与 Khanmigo 规则高度一致：不给答案、一次一问、追问链判级。
2. 两类可落地映射器：离散四档+累积分可降级；连续 LSA 相似度阈值（~0.4-0.5）。
3. 公开 rubric 仅 LearnLM 最完整；GenTutor/Socratic 无公开 rubric，均为线索级。
