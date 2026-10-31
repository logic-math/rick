# Research L1-leaf-2：误差吸收 + 节奏触发操作化（grilling L1）

## Summary
可复用的操作化主要来自经典 ITS 与医学教育，而非 LLM 导师：AutoTutor 四级探针阶梯 + teach-back「换措辞重教再测」循环可区分「提取失败」vs「不会」；LearnLM 用误概念剧本 + pedagogy rubric（「恰好够用的帮助」）编排/评估进退。Khanmigo/MathGPT/EduChat 的 hint 层级均无公开操作细节。

## 证据状态（必读）
本机 web_search（auto/Exa/brave/openai/searxng）与 fetch_content 全部失败（代理 10.229.18.27:8412 宕机、无备用 key），本子代理无 bash 工具，curl 方案不可用。故：parent 已核实锚点标 A/B；其余为模型内部知识，一律标 C 并注明待核对/无法确认。已 contact_supervisor 请求决策（附默认方案），超时无回复，按默认方案执行。

## Findings
1. **AutoTutor 四级阶梯**：学生答不出时按 pump（「还有呢？」）→ hint（内容片段）→ prompt（填词）→ assertion/bottom-out（给全句）逐级升；IDK 不作终止信号，升一级后重新 elicitation。判据：**prompt 级能补全=提取失败；到 bottom-out 才通过=不会**。源：Graesser et al. 1999 AutoTutor；VanLehn 2006 "The Behavior of Tutoring Systems"（URL 离线无法确认，C，待联网核对）
2. **Teach-back 操作化（AHRQ）**：只做产出式复述，禁止「懂了吗」是非问；失败→换措辞重教（不许原句重复）→再测，循环至通过。判据：**换措辞一次即过=措辞/提取问题；两轮仍失败=理解缺口**。源：AHRQ Health Literacy Universal Precautions Toolkit, Tool 5 "Use the Teach-Back Method"，https://www.ahrq.gov/health-literacy/improve/precautions/tool5.html（URL 未核对，C）
3. **LearnLM pedagogy rubric**（arXiv:2412.16429v3，HTML 已由 parent 核实可访问）：rubric 项（凭记忆，待核对 §3.3）含「give an appropriate amount of help—not too much and not too little」「不直接给答案，先引导或给 hint」，即把 pacing 操作化为帮助量的最小充分性。https://arxiv.org/html/2412.16429v3（源 A 级；内容凭记忆，按 B 级待核对）
4. **LearnLM §3.1 场景编排**（parent 已核实）：49 个 scenario 含 persona、学习目标、误概念剧本——「不会/半会」由剧本驱动：学生产出正确步骤=推进信号；出现脚本化误概念=倒退/加脚手架信号。https://arxiv.org/html/2412.16429v3（B）
5. **Khan Academy 传统练习 hint 阶梯**：每题多级 hint，末级=完整解答（bottom-out）；「hint 未到底即答出=半会」。源：help.khanacademy.org 练习 hint 帮助文档（URL 未核对，C）
6. **Khanmigo**：parent 已核实营销表述 "guides learners to find the answer themselves" / "doesn't just give answers"，https://khanmigo.ai/learners（B）；媒体评测称拒绝直接给答案并反问（C）。hint 层级数、每级内容未公开——**无法确认**。
7. **LearnLM 五原则公开教学策略文档**：离线无法访问 edu.google / storage.googleapis PDF，具体五条**无法确认**；论文 HTML §2/附录是替代核对点（见锚点 4 的 URL）。
8. **EduChat（arXiv:2308.02773）/ TAL MathGPT**：论文与官网均未见可复用的 pacing/hint 阶梯规则——**无法确认**（C）。
9. **延迟复测**：生产系统（Khanmigo/Khan 练习、AutoTutor 家族）均不内置延迟复测触发；该项属 L2 间隔复习层，此处不重复学习科学实证。

## Sources
- Kept: arxiv.org/html/2412.16429v3（parent 核实锚点）；khanmigo.ai/learners（parent 核实锚点）；AHRQ tool5（高置信记忆 URL）
- Dropped: 各媒体评测（无操作细节）；Bing/arXiv/PMC 直连检索（工具宕机，未执行成功）

## Gaps
AutoTutor/teach-back 原文 URL、LearnLM 五原则、Khanmigo hint 层级、EduChat/MathGPT 规则均需联网复核（建议由有 bash 的 worker 执行任务文件中的 curl 方案）。

## 结论（3 条）
1. 判据：prompt 级补全=提取失败，bottom-out 才会=真不会（AutoTutor 四级阶梯）。
2. teach-back：禁是非问，失败即换措辞重教再测，两轮不过=理解缺口。
3. LearnLM 节奏=「恰好够用的帮助」+ 误概念剧本驱动进退，rubric 可评。
