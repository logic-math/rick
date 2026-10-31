# research-L1 简报（grilling L1：教学操作化层）

主题：生产级 LLM 导师系统的教学操作化（human-loop v7 设计输入）｜阶段：grilling-L1｜时间基准：2026-10-31｜方法：4 叶子（web 工具全断→文献综合+诚实分级）+ 主线 bash 直连自验（代理 10.229.18.27:8412 宕机，绕行 curl；researcher 叶子无 bash）

## 事实性结论（12 条，按优先级）

1. **判级映射器两种公开形态**：①AutoTutor expectation 匹配——每题预设 2-5 条期望要素+常见误概念，学生自由文本与各期望算 LSA 余弦（阈值~0.4-0.5）逐条判「已覆盖」，五步框架（主问→答→即时反馈→hint/prompt 脚手架→导师收尾）；②Khan Academy 四档累积制 Attempted→Familiar→Proficient→Mastered，掌握分累积升级、后续错答可回落。前提：Graesser 系列/Khan 帮助中心，本轮未在线复核。[B]
2. **LearnLM 不判学生**：Google 路线=pedagogical instruction following——教学属性写进 system instructions 训练+RLHF；29 题 pedagogy rubric（7 点量表）评**模型**教学质量，无学生答案判级 rubric。v7 借鉴点=可配置教学行为而非判级器。前提：arXiv:2412.16429v3 全文（主线读）。[A]
3. **Khanmigo 判级=追问链**：泄露 prompt 规则（一次只问一问、绝不直接给答案、先问要不要 hint、答对才明确肯定、答错用提问引导）；教师端 Exercise Explorer 分步批改（逐步判对错）；掌握判断靠连续追问链而非一次性打分。前提：2023 泄露文本+媒体转述，非一手。[C]
4. **误差吸收判据（AutoTutor 四级阶梯）**：pump（泛问重述）→hint（提示相关概念）→prompt（词根/首字母级）→bottom-out（给解）；判别规则：prompt 级补全成功=提取失败，需 bottom-out=不会；IDK 非终止，升一级后重新引出。前提：文献共识，本轮未复核。[B-]
5. **teach-back 操作化**：生产性复述（「用自己的话说」），禁「懂了吗」是非问；失败→换措辞重教→复测；一轮换措辞通过=措辞/提取问题，两轮不过=理解缺口。前提：AHRQ Health Literacy Tool 5（页面 403）。[C+]
6. **节奏=最小足够帮助+剧本进退**：LearnLM rubric 项「appropriate amount of help — not too much and not too little」「不直接给答案、引出或提示」；§3.1 的 49 个 scenario 用误概念剧本编排进退信号——正确步骤产出=推进，剧本误概念出现=处理（非倒退）。前提：§3.1 主线读，rubric 措辞为回忆。[A/B]
7. **跨会话锚三层**：①离散掌握状态（ALEKS/KST：知识状态=可解题集合+inner/outer fringe 可学前沿，定期知识检查重评估、状态可收缩建模遗忘）；②结构化自编辑 profile 块（MemGPT core memory，LLM 自调 append/replace 算子）；③向量 episodic 库（archival top-k；Mem0 抽取-比对-ADD/UPDATE/DELETE/NOOP）。前提：1511.06757/2310.08560/2504.19413 主线核 ID；ALEKS 机制文献共识。[A-ID/B]
8. **GenMentor 动态画像**：U_t=LLM_profiler(U_{t-1},I_t)——初像来自简历+技能差，每会话由表现/用时/反馈递归更新；三维=知识状态/学习偏好/行为模式；脱离信号（低频登录/单会话超长）→干预（动机 prompt 或调难度）；另设 learner simulator 用 U_t 预演反馈。前提：arXiv:2501.15749 全文（主线读）。[A]
9. **真实异质性证据排序**：跨基座>同基座>role-play——ReConcile（异构模型圆桌共识优于单模型）；同基座多实例辩论有效（Du et al.：数学/策略推理+事实性显著升、幻觉降）但谄媚倾向致过早收敛（Sharma）；人类分歧注入=Delphi 众包道德共识聚合。role-play 反方与辩论的直接对照实验未见。前提：2309.13007/2305.14325/2310.13548/2110.07574 主线核 ID。[A-ID]
10. **LLM 不可自评结构**：LLM 违反人类知识先修依赖（有前置线索也不利用）、模型间知识结构不共享（分布低重叠）——v7 概念图/判级不能直接采信 LLM 自评，且异构 ensemble 有去相关基础。前提：arXiv:2609.05245 摘要（主线核）。[A]
11. **ASCII 教学图可行但脆弱**：LLM 文本模态读 ASCII 视觉语义弱（ASCIIEval 3K+ 样本基准）；生成侧可经指令微调引出（SVE-ASCII/ASCIIArt-7K）；工程陷阱=等宽依赖、UAX#11 box-drawing 东亚宽度 Ambiguous、CJK 全角 2 列 vs LLM 码点计数必错位；稳健替代=Mermaid/Graphviz（纯文本仅适合 code-fence/RFC 场景）。前提：2410.01733/2603.14505 主线核；宽度标准为 Unicode 规范事实。[A/B]
12. **延迟复测缺位**：全部调研对象（Khanmigo/LearnLM/GenMentor）均无延迟复测触发机制——属间隔复习层（L2 已覆盖），L1 操作层无现成实现。前提：本轮检索缺席证据。[B-]

## R7 上报（8 条）

1. GenTutor 出处无法澄清——arXiv 查无此文（两个叶子独立确认）；任务书/上游调研中的名称需正确引文
2. Khanmigo 判级 prompt 与 hint ladder 未公开（khanmigo.ai 反爬、帮助中心 403）——结论 3 依赖泄露转述
3. LearnLM 五原则一手文档不可达，且存在两个互相冲突的回忆版本（叶子：curiosity/active learning/higher-order/well-being/personalize vs 主线记忆：learner-centered/growth mindset/responsible citizens 等）——措辞与「follow the student」操作化待一手核实
4. AutoTutor 阶梯/LSA 阈值、AHRQ teach-back、Khan 四档分数线：本轮均未在线复核（PMC 相关性弱/AHRQ 403）——B/C 级待升级
5. 「多模型辩论 vs 单模型反方 role-play」直接对照实验未找到——item-5 区别性证据缺口
6. 异构 RAG 立场 ensemble、RLHF 标注分歧进入导师系统：查无实现案例（item-5 子问题）
7. ASCII 教学最佳实践专文少（仅能力基准 ASCIIEval/SVE-ASCII 与工程共识）——item-6 操作层证据薄
8. 环境故障残留：代理宕机期间 B/C 级项（AutoTutor/AHRQ/五原则/Khan 帮助文档）需网络恢复后由 bash 能力节点重验

## 叶子文件清单

- doing/grilling/research-L1-leaf-1.md（判级，leaf-1 超时但输出已镜像落盘）
- doing/grilling/research-L1-leaf-2.md + briefs/research-L1-leaf-2.md（误差吸收+节奏，双版本）
- doing/grilling/research-L1-leaf-3.md（跨会话锚，=leaf-3-receipt.md 镜像）
- doing/grilling/research-L1-leaf-4.md（异质性+ASCII）
- briefs/leaf-{1,2,3,4}-task.md（派发任务书，含环境绕行指引）；briefs/leaf-2-receipt.md、leaf-3-receipt.md

（主线自验源：arxiv.org/html/2412.16429v3、2501.15749v1 全文；arXiv API 核验 2305.14325/2309.13007/2310.13548/2110.07574/2310.08560/2504.19413/2501.13956/1511.06757/2609.05245/2410.01733/2603.14505；khanmigo.ai/learners、socratic.com 实抓）
