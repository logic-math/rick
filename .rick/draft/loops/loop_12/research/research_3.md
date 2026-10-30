# research_3 — 五步教学 loop 的技术盘点与 skill 沉淀映射（事实还原）— 2026-10-21

元信息：任务模式 `facts`，承接 judgment.md「[V8 + loop/skill 架构提案]」原话（含 human 逐字的五步 loop 与 skill/agent 划分）。等级 A=一手规范/同行评审一手/仓库原文；B=官方文档/综述；C=线索。不重复 research_1/2 结论；引注在 leaf-r3-1…4。

## Q1 五环节技术盘点（业界已有技术与方法；详见 leaf-r3-1 / leaf-r3-2）

**① 目标对齐**：业界**无统一的「goal elicitation」规范术语与评测基准**（leaf-r3-1 Gaps）；可用的是目标形式化——Mager/ABCD 要求「可观测行为+条件+达标标准」，Bloom 修订版 6 认知层×4 知识类只定层级、不产生阈值（B）；差距分析无统一量化标准（B）。唯一 LLM 实现线索：GenMentor 以「目标+简历」由 LLM 推技能集 S′ 与自评 S₀ 求差 ΔS，人工确认、**无初始测验**；目标→技能 Recall .67 / Precision .63（GPT-4o，人工研究 n=20）—（B/C 小样本）

**② ZPD 初始化（诊断测评）**
- **CAT/IRT 的收益有前提**：一维 IRT + 已标定题库 + SE 终止；实测 CAT 均 5 题（SD=1.9）vs 7 题固定表，r=.90、T 分差<0.5，**中等能力+固定短表时收益≈0**；终止规则常取 SE≤3.0（A=PLUS-M、JCAT）
- **知识追踪精度**：良好调参 BKT AUC .73（加遗忘 .83、按题目索引 .90 ≥ DKT .86）；DKT 原报 ASSISTments .86 被复现研究指高估；AKT 最多 +6% AUC 但需海量交互、解释性弱；LLM-KT（用题面文本缓解冷启动）ASSIST09 ≈.83–.84、较 DLKT 增益约 2–5% —（A/B=EDM16、KDD20、arXiv 2406.02893）
- **前置知识探测**：pretest 非中性——先测后学对 ≥24h 保持有 d=.44–>2.0 增益，但限于被测项、概念性 pretest 可失败，故诊断性前测会**抬高基线** —（B=EPR）
- **口头回忆式测评**：learning-by-teaching g=.27（39 研究）、准备教 .35、教+测 .56（28 研究）——这些是**学习收益**证据；作为**测量工具**的效度（信度、与 IRT 估计的一致性、区分度）**无系统证据**，teach-back 16 项 RCT 中仅 8 项（N=721）测知识获得且方法学受限 —（A 收益 / B 效度缺位）

**③ 大纲生成（课程序列化/学习路径）**：知识结构建模用 **KST**（知识状态=题目子集，外缘=可学、内缘=刚会易失；Markov 型自适应 25–30 题定状态，需先建知识结构）—（A/B=ALEKS）；后续以「学习序列」表达任意外延知识空间（A）。**概念级先修自动推断**：MIT OCW 目录上 CGL.Rank AUC 0.96/MAP 0.46（vs 1NN .76、SVM .78），公开基准 654 门课/861 条边/1008 概念对（A=JAIR、AAAI17）。**路径规划**通常拆为「追踪定状态 → 图搜索/MDP+RL/层次 RL → 多维评测」三组件（B）。**LLM 生成课纲**的评测=与现有大纲相似度+专家 Likert，产出结构完整但趋同缺校本特色（B=IEEE TLT 2025）；EduPlanner 多 agent「评价+优化」闭环，但为 LLM 自评、**未测学习增益**（B）。产品侧：Khan Academy 以 skill 达 proficient/mastery 度量；Duolingo 2022 起改线性 path 并刻意混排概念（B）。

**④ 教学内容生成（图解+文字）**：**Mayer 原则 2025 元分析**（92 篇/181 研究/591 效应）总体 g=0.37；最强=去除 seductive details 1.00、modality 0.82、personalization 0.70、multimedia（语词+图）0.68、句级 coherence 0.63；**segmenting、contiguity、cueing、voice、redundancy 不显著或很小**；主动生成类（self-explanation 0.46、testing 0.41）强于版式类；作者早期自报值更高（multimedia 1.67），**引旧值会高估**（A=Educ Res Rev 49:100730、JCAL 2017）。静态图形对阅读 g=0.39（39 实验/2103 人，图示/流程图无差异）；graphic organizer 学生自建 .59/.63、教师提供 .70/.53，可能有发表偏倚（A）。**生成技术现状**：TikZ 最好模型仅 73.9% 成功率（A=ACM L@S）；LLM 直出 SVG 语义 .80/句法 .76，DALL·E 3 像素图基本不可用，失败模式=紧贴题面、**丢失教学意图**（A=arXiv 2503.07429）；chart-to-code 在 ChartMimic 上 82.2/61.6（A=ICLR25）；NL→Mermaid 已有 MermaidSeqBench（B）。

**⑤ 掌握度判断与推进/倒退**：**阈值来源薄弱**——Bloom 1968 只主张「90% 以上学生可掌握」未定数值阈值，Guskey 述 80% 是常见约定、百分比切点**本质任意**；70/80/90% 三组对照中 90% 仅改善部分测验、**期末成绩无差异**（A/B）。产品实现：Khan Academy <70% = Attempted、70–99% = Familiar、满分 = Proficient/Mastered，课程掌握度只计后两级（B）。**推进**：model tracing 做步级诊断+知识追踪概率以即时提示/跳练（A，前提是领域生产式规则库）；KST 外缘/内缘界定可学与易失（A）。**遗忘调度**：SM-2 用 EF≥1.3 区间公式、不输出概率（A）；HLR 在 Duolingo 使回忆预测误差降 45%+（A）；FSRS-7 在 349.9M 复习上 logloss .3363/AUC .7243，对 SM-2 在 99.6% 用户更优（B；SM-2 非概率模型，比较有保留）；**「FSRS 比 SM-2 少 20–30% 复习」溯源失败，不可引用**（leaf-r3-1 Gaps）。
## Q2 skill vs agent 的业界划分原则（详见 leaf-r3-3）

- **skill 规范形态**=目录 + `SKILL.md`；frontmatter 必填 `name`（≤64，须与父目录同名）、`description`（≤1024，须写「做什么+何时用」）；**三级渐进加载**=元数据 ~100 tokens 常驻 → 正文（建议 <5000 tokens）激活时加载 → 资源按需；`allowed-tools` 标实验性。agentskills.io 为开放标准，46 个客户端中 11 家（含 Claude Code、Codex、Cursor、Copilot、Gemini CLI）官方文档明示读取 SKILL.md —（A=spec；B=clients）
- **agent（subagent）**=独立上下文 + 自定义 system prompt + 独立工具白名单，只回传摘要；官方原话「Skills add knowledge to your current conversation…Subagents run in a separate context」「技能是培训材料，子代理是独立员工，子代理可用 skill」；skill=程序性知识 vs MCP=工具连通性 —（B=code.claude.com、academy.claude.com、skills-explained）
- **他厂**：OpenAI 以 Agent=LLM+instructions+tools(+handoffs) 界定 agent、Skills 为「知识/过程包」（B）；Google ADK 的 Skill 按该规范实现、经 `SkillToolset` 三级加载（实验性），skill 作 toolset 挂载于 LlmAgent（B）
- **pi**：实现该规范（只注入 name/description/path，`/skill:name` 强制加载，`disable-model-invocation` 仅命令触发）；pi agent=markdown+frontmatter（tools/defaultContext/thinking/timeoutMs/systemPromptMode）—（B=pi.dev/docs/skills；A=~/.rick/pi/agent/agents/*.md）
- **本工作区实证**：`skill_teach.md` 与 `agents/teach.md` 同文两用，差别只在「是否新开会话 + 工具白名单」—（A）
- **业界判据 5 维**：触发方式／上下文（注入 vs 独立）／工具权限／组合方式／状态（见 leaf-r3-3 表，A/B）
- **与 human 划分一致**：调研=agent（独立上下文+自主检索循环）、ZPD/大纲/内容生成=skill（声明式可复用逻辑），均落在业界判据内（对照判定，非新事实）
- **不一致（只列事实）**：① 业界 skill 形态特征=按需加载+可携带脚本资源，human 划分未涉形态维度；② **评估被业界拆为独立对象**——GenMentor 的 Skill Gap Identifier / Adaptive Learner Profiler 为独立路由与独立 agent（A），ALEKS 诊断在 KST 引擎内（B），human 并入 ZPD skill；③ human 判「讲出来不需要」，证据侧把学生生成当机制（learning-by-teaching g=.27、self-explanation g=.40–.70，A），GenMentor 用 Quiz+Learner Feedback Simulator 代替真人生成（A）；④ pi 层 skill/agent 是同一文本的两种挂载（A），故该划分在本工作区是**挂载选择**而非内容差异。

## Q3 sparkling SKILL.md 分析（详见 leaf-r3-4）

- **仓库事实**：7 个文件（SKILL.md 6419B），无脚本、无 references；AGPL-3.0；26 stars；创建 2026-10-07、末次推送 2026-10-14 —（A=GitHub API）
- **定位是「业务研讨」非「原创性启发」**：description 原文「携带材料与用户进行结构化业务研讨：对齐问题和目标，记录事实与假设，用证据对撞和异质解释突破旧判断…用于业务卡点、复杂资料讨论、深层洞察和需要认知突破的研讨任务」；全仓无「原创性」字样 —（A）
- **流程**：张力预处理（首轮出三张张力牌：反常点/互斥点/默认公理，明令「严禁输出常规公文综述」）→ 定锚（卡点须写成「为什么事实A与事实B同时发生」）→ 短兵相接（每轮约 250 字，必含「接住直觉」+「抛出单点抉择」，中间环节在**挑衅/PO** 与**概念混成**中二选一）→ 剃刀收敛或 `/research` 转接（严禁脑补假设）→ 两出口（Aha! 顿悟纪要／取证清单）—（A）
- **prompt 结构**：角色（「推导伙伴，不是会议记录员，更不是谄媚的顺从者」，价值=「认知摩擦力」）+ 白板契约（持续维护 `sparkling-whiteboard.md`，固定 5 项：卡点与目标／对撞硬事实／已击碎假设／领先解释／证据缺口）+ 红线 4 条（禁谄媚／禁 `[事实]/[假设]` 标签／禁大段公文／**防倒退**）—（A）
- **机制—证据对应**：挑衅/PO←de Bono 横向思维；混成←Fauconnier & Turner 概念整合；反常/互斥←anomaly-driven（B）；剃刀与 anti-regression 为自述、无外部引用（A）
- **可升级点（只列事实差异）**：① 无学习者角色、无掌握度闭环，白板存研讨状态非学习者模型；② 无 eval／案例／效果数据，README 自述「不承诺直接生成确定答案」（A）；③「防倒退」防的是重启已证伪假设，与创造力证据的**去固着**方向不同且无该手段（Einstellung 81% vs 0%，「别盲从」提示降至约 55%，research_2 结论 9）；④ 无**并置对比/变式**（d=0.99，结论 3）与**跨时间重访**（d=0.29，结论 1），只在单次会话内工作；⑤ PO/混成的原创产出效应无直接实证（结论 11）。

## Q4 GitHub / 业界类似项目与产品对标

- **GenMentor（GeminiLight/gen-mentor）= 覆盖环节最全的开源实现**：WWW 2025 Industry Track oral，arXiv 2501.15749；路由即环节——Goal Refiner／Skill Mapper+Skill Gap Identifier（目标→技能并推断水平）／Adaptive Learner Profiler（每课按 quiz 证据重建）／Learning Path Scheduler（建·改·重排序列）／Knowledge Explorer+Drafter+Integrator（选点→起草→整合）／Quiz Generator／Learner Feedback Simulator；论文只给定性结论、无效应量 —（A=仓库原文；B=摘要）；★135、CC0-1.0、推送 2026-10-08（A=API）
- **现成库**：pykt-toolkit（KT 基准库，MIT，★451，2026-10-18）／pyBKT（MIT，★288，活跃）／EduKTM（Apache-2.0，★267）／tutor-mcp（把任意 LLM 变 ITS 的 MCP server，MIT，★53）—（A=GitHub API）
- **skill 生态**：anthropics/skills（官方集合，A）；obra/superpowers（组合式 skills 框架，MIT）；awesome-claude-skills 系列（C）
- **其他教学 agent 开源项目（C，未深验，名单见 leaf-r3-3）**：WeSmartFlow、OpenTutor、forge（LangGraph+SM-2）、acesat-multiagent（+BKT+先修路由）、agora-agentic-tutor 等
- **产品侧实现**：Khanmigo=苏格拉底提示、不给答案、引导自解释（B）；LearnLM=提问+分步拆解+配图/测验（B；效果证据见 research_1 Q3）；Birdbrain=按作答更新知识点熟练度定难度（B）；ALEKS=KST 显式状态+周期复测（B）；Anki/RemNote=SM-2/FSRS 排程，只覆盖复习（A=官方文档）；NotebookLM=仅基于上传来源作答（B）；ChatGPT Study Mode=system instructions 做苏格拉底提问+自省提示（B）

## Q5 待沉淀 skill 清单（事实性映射，按 human 划分；只列映射，不下设计结论）

1. **ZPD skill（human 把掌握度判断并入其中）**：输入=历史 loop 关键词与知识点／human 回忆原话（V6/V9 已裁决形态）／本模块目标；输出=已掌握·未掌握·无能力标记、下一课定位（推进/倒退）、回忆引导关键词。依赖=自由回忆与自我解释类测查（A 学习收益，B 测量效度缺位）、CAT/IRT（A，收益有前提）、BKT/DKT/AKT 追踪（A）、KST 内外缘（A）、mastery 阈值（B，来源薄弱）。**现成可承担**=pyBKT/pyKT/EduKTM、FSRS 排程库、free-recall 自动评分开源管道（B）。**须自研**=自然语言回忆→结构化能力判定与推/倒决策、与 loop 历史文件/关键词索引对接（业界无同构开源实现，C）；LLM 判分与人类一致性仅中等至良好（65 项研究合成，B）。
2. **调研 agent**：已存在（research agent，A=工作区原文），输入=派发任务与材料路径，输出=`research_N.md`+values 待裁决；无新自研需求。
3. **大纲生成 skill**：输入=research 事实+已裁决价值+mission；输出=syllabus（SENSE 留痕+模块序+每块位置理由）。依赖=概念级先修推断（AUC 0.96，A）、KST 学习序列（A）；LLM 生成课纲**无学习增益 RCT**（B）。**现成**=GenMentor 的 Skill Mapper/Learning Path Scheduler 代码（A）；**须自研**=SENSE 五步留痕的 syllabus 形态（无现成）。
4. **教学内容生成 skill**：输入=模块+当回合事实；输出=lesson（图解+文字）。依赖=Mayer 2025 元分析（总体 g=0.37；最强=去 seductive details 1.00／modality 0.82／multimedia 0.68；segmenting·contiguity·cueing 不显著，A）、graphic organizer .53–.70（A）。**现成**=Mermaid（有 MermaidSeqBench，B）、TikZ（成功率 73.9%，A）、SVG 直出（语义 .80/句法 .76，A）；**须自研**=图文配比与教学意图约束、渲染失败回退（失败模式=丢失教学意图，A）。
5. **原创性启发（human 未指定 skill 或 agent）**：候选骨架=sparkling（Q3）；已知缺口=无学习者模型、无去固着手段、无并置对比/跨时间重访、无效果数据。
6. **总结与复习 skill**：输入=历史 loop 关键词；输出=回忆引导关键词+再创造任务。依赖=SRS 调度（FSRS-7 对 SM-2 在 99.6% 用户更优，B）+检索练习（research_1）。**现成**=FSRS 库；**须自研**=跨 loop 关键词抽取与再创造任务生成（形式受 V5–V9 限定）。
7. **「让学生讲出来」=不需独立 skill（human 裁决）**：与 learning-by-teaching g=.27／self-explanation g=.40–.70 的学习收益证据存在张力，而测量效度证据缺位；已入 V11。

## R7 上报项

1. 「80–90% 掌握阈值」无一手实验推导来源：Bloom/Guskey 视之为约定，唯一直接对照实验显示 90% 对期末无增益 → 阈值属工程选择而非实证结论。
2. 「FSRS 比 SM-2 少 20–30% 复习」溯源失败（官方基准当前无 SM-2 行，SM-2 不输出概率），该数字不应引用。
3. free recall / teach-back / protégé 作为**测量工具**的效度（信度、与 IRT 估计一致性、区分度）无系统证据；现有文献测的是学习收益。
4. 「goal elicitation」缺同行评审规范术语与评测基准；LLM 侧研究样本普遍 <100，指标多为 LLM 判分。
5. CAT 在课堂场景「何时优于固定测验」的头对头证据不足；Coursera 先修图、Khan 知识图谱无一手文档；Manim/Graphviz/Excalidraw 无专项评测。
6. 「EDUPLAN」未检索到同名 benchmark（最近者 EduPlanner arXiv:2504.05370），疑名称有误。
7. sparkling 全仓无「原创性启发」表述且无 eval/案例；OpenAI Agents SDK 是否有独立 Skills 文档页未确认；生态 star 数来自聚合页，许可与末次提交未逐仓核验。
8. GenMentor 论文 PDF 抓取失败（aborted），效应量未取到；Mayer 新元分析 segmenting/contiguity 具体 g 被付费墙遮挡。

## Gaps

- 「原创性启发」是五环节中唯一**无直接实证**支撑者：PO/概念混成有概念出处，「该机制提升原创产出」无 RCT。
- 缺「LLM 生成课纲→学习增益」RCT；缺「LLM 图解→理解增益」课堂级实验（现有仅生成成功率/质量评测）。
- 「口头回忆式测评」与标准测验的**效标关联效度**未收敛：只拿到自动评分管道与学习收益类间接证据。
- 无任何 loop 级系统（含 GenMentor）报告「原创性产出」指标；产品效果证据止于成绩/知识掌握。
- 本工作区无 token/延迟日志，无法量化 skill 化后的上下文开销变化（同 research_1 Gaps）。

## 价值性信息

本轮识别 5 条，已逐条追加 values.md 待裁决段 [V10]–[V14]：V10 掌握度阈值与推进条件（阈值来源经本轮核验为「约定」而非实证）；V11「讲出来」是否作为达标判据；V12 skill/agent 划分判据（业界形式规范 vs human 语义）；V13 原创性启发的交互形态（对抗式认知摩擦力）；V14 现成组件 vs 自研的边界（含 AGPL 等许可约束）。裁决全部留空，待 human。

## 叶子文件

- research/leaf-r3-1.md（环节①②⑤：目标对齐、ZPD 初始化测评、掌握度与推进/倒退；含逐条来源链接）
- research/leaf-r3-2.md（环节③④：课程序列化/学习路径 + Mayer 与图解生成技术）
- research/leaf-r3-3.md（skill vs agent 规范判据 5 维表 + 开源生态清单）
- research/leaf-r3-4.md（sparkling 流程还原 + 产品侧环节实现）
