# research-L1-r2 — 外部「激发原创性学习/思考/表达」skill 先例与方法论 — 2026-11-25

> 元信息：时间基准 2026-11-25。三条主线：①AI agent 生态写作/思考类 skill 先例；②writing-to-think 方法论；③学习科学生成性学习方法。方法：web_search（多 provider）+ fetch_content + curl OpenAlex/arXiv API 交叉验证。信源等级 A=一手（原始论文/官方原文/代码原文）、B=权威二手、C=弱来源/线索。前置：research-L1.md 已覆盖认知机制/AI 先例/CPS 映射，本文只做增量。

### 事实性结论

1. 【官方共同创作工作流】anthropics/skills `doc-coauthoring`（官方，代码原文，A）：三阶段——Context Gathering（Claude 只提澄清性问题收集上下文）→ Refinement & Structure（逐节头脑风暴+编辑迭代）→ Reader Testing（用一个无上下文的 fresh Claude 读文档、抓盲点后再给人看）。核心机制一句话：人供上下文与判断，AI 管结构与迭代，以「无知读者」收尾检验。为什么有效：把人绑定在「知道什么/要什么」的位置上，表达检验外包给机器。适合极简会话：✓（三阶段可折叠为三个回合）。来源：github.com/anthropics/skills/blob/main/skills/doc-coauthoring/SKILL.md（A，直接读取）。
2. 【官方反思钩子】anthropics/skills `discernment-nudge`（官方，A）：在实质性回答/草稿后追加 2-3 个**绑定到产出中具体内容**的短问题（帮用户核查事实/审视推理/注意缺失上下文），每会话最多一次，有显式跳过边界（琐碎问答/创意写作不触发）。核心机制：模型化核查习惯而非说教。为什么有效：轻量制造「行动前的反思时刻」，直击「AI 说什么信什么」（对应 research-L1 主线 2 信心效应）。适合极简：✓（极轻）。来源：同仓库 skills/discernment-nudge/SKILL.md（A）。
3. 【实现前对话】obra/superpowers `brainstorming`（社区，代码原文，A）：原句 "People often haven't finished working out what they want. Good questions help them think it through." 设计规则：未知 why 之前只问开放式问题、不给选项（"Their own words carry intent that your options can't"）；一次只问一个；只问「答案会改变产出」的问题；问 anti-goals（什么会让它即使技术上成功也算失败）；提供随时退出（"if you'd rather I just dive in, say so"）。适合极简：✓（规则即约束、无重工具）。来源：github.com/obra/superpowers tree/main/skills/brainstorming（A，直接读取）。
4. 【搏击式陪练】mattnowdev/thinking-partner（社区 234★，代码原文，A）："Not a lecture — a sparring session"——挑战假设（把「当事实用的隐藏信念」翻出来）、6 级思维状态诊断+定向干预、7 对认知操作（Hold/Resolve、Decouple/Re-couple）、"maintains productive tension — holds complexity open long enough to find real insight"（保持张力、不急于收束）、显式 anti-patterns 清单。适合极简：△（150+ 模型库偏重，但「状态诊断+保持张力」模式可移植）。来源：github.com/mattnowdev/thinking-partner（A）。
5. 【追问止损规则】m4vic/socratic（社区 140★，代码原文，A）：停止规则="question until the answer is solid"——只问最小必要问题集；物质性判据：下一问若不改变 design/risk/cost/verification 就不再问。给「无限追问」这一失效模式提供显式止损（与脚手架坍塌互为镜像）。适合极简：✓（一条规则）。来源：github.com/m4vic/socratic（A）。
6. 【读者模拟】haowjy/creative-writing-skills `reader-sim`（社区，代码原文，A）：AI 扮演指定画像的「第一次读者」，逐时刻报告阅读体验（哪里被吸住/走神/ held 什么问题），只锚定具体段落、明确不做工艺批评。核心机制：把「面向读者的表达迫使澄清」机器化。反面参考：同仓库 `creative-writing-muse` 是「AI 以你的声音代写」路线（代写而非激发）。来源：github.com/haowjy/creative-writing-skills（A）。
7. 【生态位证据】blader/humanizer（73k★）：抹除 26 种 AI 写作痕迹（Not X but Y/一行收束句等），盲测 16/16 优于原文——写作生态最大众的需求是「去 AI 腔」，侧证「AI 代写→人修整」已是主流痛点。Master-cai/Research-Paper-Writing-Skills（8.5k★）：reverse outlining+以怀疑审稿人视角对抗式自审——激发的是 AI 的批判而非人的原创（对照组）。来源：两仓库 README/SKILL.md（A）。

### 主线 1 top-3 可借鉴设计模式

1. **「实现前对话」+ 退出选项**（brainstorming）：未知 why 前只问开放问题、一次一问、只问改变产出的问题、anti-goals、随时可说「直接做」退出——保护自主性（与 SDT 内在动机证据直接兼容）。
2. **针对性核查钩子**（discernment-nudge）：2-3 个绑定到刚产出内容的具体问题、每会话一次封顶——最轻的「防照单全收」设计。
3. **无知读者检验**（doc-coauthoring 阶段 3 / reader-sim）：fresh-context AI 扮演目标读者测表达——把「表达迫使澄清」变成可执行回合。

（以下为 2026-11-25 追加批次：主线 2 → 主线 3 → 三线汇总 → R7 → Gaps）

## 主线 2：writing-to-think 方法论（Zettelkasten / freewriting / Craft of Research / They Say, I Say / Bereiter & Scardamalia）

8. 【理论内核：知识转化】Bereiter & Scardamalia 1987《The Psychology of Written Composition》区分 knowledge telling（把记忆中与主题相关内容顺序倒出填任务空间）与 knowledge transforming（内容空间↔修辞空间双向作用：「怎么说」的修辞问题反过来改造「知道什么」）；Klein 1999（EPR，498 引）认定知识转化是写作促进学习的主解释机制。writing-to-think 的内核=转化回路，不是「写下来」本身。信源：专著一手（A）+权威综述（B）。
9. 【Zettelkasten】Ahrens 2017《How to Take Smart Notes》：机制三件套——①永久笔记必须用自己的话写（结构上≈自解释）；②卡片靠显式链接生长，原创想法来自「意外连接」（组合式涌现），Luhmann 视卡片盒为「交流伙伴」；③无预设结构，链接即结构。Luhmann 产出（Luhmann 档案方 zettelkasten.de 口径，B）：50 本书+600+ 篇文章；「约 9 万张卡片」通行说法未在档案一手页核到（C，R7-4）。整书为方法论自述无 RCT（诚实分层：机制①有主线 3 实证呼应，②③仅方法论主张）。
10. 【无评判生成】Elbow 1973 freewriting：限时（10 分钟）不停笔、不修改、不评判——把「生成」与「评判」分离，让被自我编辑压制的想法浮出；配套 believing game（先彻底代入「相信」一个观点再质疑）用于读他人观点。证据层级：写作学科理论+案例（B）；独立 RCT 未检索到（R7-6）；方向与 Bangert-Drowns 表达性写作干预一致（见条目 15）。
11. 【对话式研究观】The Craft of Research（Booth/Colomb/Williams 1995-2016 四版）：研究=与读者的对话；装置链=topic→question→problem（双重「So what?」：谁在乎/错在哪）→claim-reason-evidence-warrant-承认并回应异议；写作不是报告已完成思考，而是思考本身——初稿暴露思维漏洞（面向读者迫使澄清）。教科书级方法论（B）。
12. 【对话定位+模板】They Say / I Say（Graff & Birkenstein 2005/2018）：原创观点=进入对话后的「I say」回应位；操作化=准确概括 they say（用 believing game）→标定分歧点→模板句式回应（"Although X, I argue Y because Z"）。模板悖论（作者主张）：句式约束反而把注意力释放到观点本身；无量化 RCT（C）。
13. 【社区知识建构】Scardamalia & Bereiter 2006（Cambridge Handbook of the Learning Sciences）：knowledge building=社区持续改进「可改进观点」（improvable ideas / idea diversity / epistemic agency）；原创=观点改进过程的产物，而非个体灵光。教育理论框架（B）。
14. 【环节实证强度分层】五套方法论中环节分层：强实证=「自己的话写/解释」（主线 3 自解释 g≈0.55、writing-to-learn 元分析）；中=生成/评判分离、面向读者表达（理论+间接实证）；弱（仅方法论主张）=卡片意外连接、模板赋能原创。弱项不冒充实证结论引用。

### 主线 2 top-3 可借鉴设计模式

1. **「先写后评判」两段式**（Elbow 生成/评判分离 × 自解释实证）：会话先开无评判生成回合（human 限时写自己的话，AI 只记录不纠正、不评价质量），后开反馈回合——生成期与评判期在时间上硬分离。
2. **「they say → I say」定位**（Graff & Birkenstein）：AI 摆出领域现有说法（they say）并只做「标定分歧」追问，human 的回复位天然是原创位（I say）——原创=对话中的回应位，不要求凭空产出。
3. **连接追问**（Zettelkasten 意外连接）：AI 只问「你刚产出的想法，与你会话内/记忆里已有的哪个想法相连、冲突或意外？」——把「意外连接」机器化为固定追问回合，不替 human 连接。

## 主线 3：学习科学中的生成性学习方法（效应量实证）

15. 【writing-to-learn 总效应】Bangert-Drowns 等 2004（RER，48 个项目，1091 引；A 级摘要已核）：写作→学业成就小正效应（通行转引均值 d≈0.30，B）；放大器=元认知提示、更长疗程；缩小器=6-8 年级实施、更长写作任务（「写更多」≠「学更多」）。常转引的元认知提示分组数值（0.55 vs 0.23）未核到一手表格（R7-2）。
16. 【自解释】Bisra 等 2018（EPR，369 引，A）：诱导自解释元分析总体 g≈0.55（通行转引值，一手正文未能抓取，R7-1）——「用自己的话解释」是生成策略族中证据最厚的单一机制，直接对应 Zettelkasten 永久笔记与 freewriting 的内核。
17. 【策略效用评级】Dunlosky 等 2013（PSPI，4834 引，A）：summarization=低效用、self-explanation=中、practice testing=高效用——关键区分不在「写不写」而在「生成 vs 复述」；概括若沦为 knowledge telling 即失效（与条目 8 呼应）。
18. 【生成策略族】Fiorella & Mayer 2015/2016（CUP 专著，692 引，A/B）：8 策略（summarizing / mapping / drawing / imagining / self-testing / self-explanation / teaching / enacting），共同内核=生成「超出给定信息」的产品。drawing 一级证据为评论/综述级（Ainsworth 等 2011《Science》评论，1011 引），无独立元分析数值（R7-5）。
19. 【教学与检索】learning-by-teaching 元分析（28 研究，A）：准备教学 g=0.35、真正互动式教学 g=0.56，「期待互动教学」本身放大效应；Chase 等 2009（460 引，A）：protégé effect——教「可教代理」使学生投入更多努力、更愿暴露自身知识漏洞（机制核心=投入量）。testing effect：Rowland 2014（Psych Bull，1341 引，A）测试 vs 重学 g≈0.50（通行转引值，R7-3）；Adesope 等 2017（RER，730 引，A）方向一致。
20. 【先挣扎后指导（已录增量）】Sinha & Kapur 2021 元分析（g=0.36）与 Bastani 等 PNAS 2025（护栏版 tutor 消除学习损害）已在 values v2/v4 与 L1 调研记录——不重复；增量=与 16/19 同向支持「human 先生成、AI 后反馈」。
21. 【刻意练习】Macnamara 等 2014（Psych Science，939 引，A；2018 有勘误）：刻意练习解释的绩效方差=游戏 26%、音乐 21%、体育 18%、教育仅 4%、职业 <1%（通行引用值，B）——「结构化反馈+专注重复」在教育场景解释力有限，不宜把「练习式重复」当 v8 激发原创的主机制。
22. 【AI 家教 RCT（正面）】Kestin 等 2025（Sci Rep，N=194，真实哈佛物理课，A）：教学法提示词定制 GPT-4 家教（主动参与+认知负荷管理+成长心态；答案预置防幻觉）0.63-1.3 SD 超课堂主动学习，用时更少（49 vs 60 分钟）——「提示词级教学法」可复制到 v8；但属解题型 tutoring，非写作驱动学习（R7-7）。
23. 【认知债（负面）】Kosmyna 等 2025（arXiv 2506.08872 预印本，C）：54 人 EEG 三组（LLM / 搜索引擎 / 纯脑）×3 场写作：LLM 组脑连接最弱（纯脑最强），「认知活动随外部工具使用缩减」；LLM→纯脑切换组仍 under-engaged——表达外包=神经参与下降（「认知债」）。未同行评审、N 小，仅方向线索。
24. 【证据生态警告】ChatGPT 学习效果被引最多的元分析（Wang & Fan，HSSC）2026 年被撤稿（撤稿说明 10.1057/s41599-026-07310-z：元分析差异+作者未回应；A 级一手已核）——「AI 提升学习」当前无可靠元分析级证据；v8 不可建立在「AI 助学高效应量」叙事上。可用 A 级证据=单点 RCT（Kestin、Bastani）+无 AI 的生成学习元分析。

### 主线 3 top-3 可借鉴设计模式

1. **生成优先 + 元认知提示包裹**（16+15+17）：任何「AI 给信息/答案」回合前先让 human 生成（自解释/预测/初稿）；生成后必接 1-2 个绑定内容的元认知追问（「你怎么知道 X？哪里最不确定？」）——两步各有效应量，组合即 0.30→0.55 的放大结构。
2. **「教 AI」回合**（19，LbT g=0.35/0.56 + protégé effect）：AI 扮无知可教代理，human 讲解，AI 追问「为什么 / 我这样理解对吗」——互动期待本身放大效应，机制=投入量；与「AI 不代写」天然兼容。
3. **复述检测 → 知识转化追问**（17+8，低效用概括 × knowledge telling）：AI 检测 human 产出与给定材料同构（复述味）→ 不纠正，只追问「用你自己的话，这对你的问题意味着什么」——把 knowledge telling 拒转为 knowledge transforming。

## 三条主线的 top-3 可借鉴设计模式（跨线汇总）

- ① **生成优先 + 元认知追问**（主线 3 实证底座 × 主线 1 brainstorming「先问后做」）：human 先产出（初稿/自解释/预测），AI 后反馈，反馈必含 1-2 个绑定内容的元认知追问——三条线证据交汇最强的单一设计。
- ② **原创位设计**（主线 2 they say/I say + 教 AI 回合）：AI 摆 they say 或扮无知教者，human 的输出位=原创回应位/教者位——原创性由「位」的结构保证，不依赖灵感。
- ③ **表达检验收尾**（主线 1 无知读者 + 主线 2 面向读者迫使澄清 + 主线 3 反认知债）：fresh-context AI 扮目标读者检验 human 的最终表达；表达必须由 human 自己做（外包有认知代价），检验可以外包给 AI。

## R7 上报（无法澄清项，主线 2/3 新增）

1. Bisra 2018 总效应量 g≈0.55 与效应量数（k）未核到一手数值（Springer 正文 / Crossref / S2 / OpenAlex 摘要均不可得）——需人工下载 PDF 核对
2. Bangert-Drowns 2004 常转引的元认知提示分组数值（0.55 vs 0.23）未核到一手表格（A 级摘要仅确认方向；OA 不可得；Klein & Boscolo 2016 JoWR OA PDF 两次抓取失败）
3. Rowland 2014 testing effect 总效应量（g≈0.50 通行转引值）未核到一手数值；Adesope 2017 分组 g 仅有摘要方向
4. Luhmann 卡片总数（通行说法约 9 万张）未在 Luhmann 档案一手页核到；已核口径为 zettelkasten.de「50 本书+600+ 篇文章」（与部分转引「70 本书/400 篇」不一致）
5. drawing / imagining 生成策略无独立元分析效应量（仅《Science》评论与综述级证据）
6. freewriting 独立 RCT 证据未检索到（Elbow 为理论+案例）
7. 「AI 家教 × 写作驱动学习」直接结合的一手 RCT 缺失（Kestin 为解题型；Kosmyna 为预印本小样本）
8. ChatGPT 学习元分析生态撤稿后未重建（无未撤稿的高引替代元分析）

## Gaps（主线 2/3 增量）

- **组合设计无整体 RCT**：「先写后评判 + 元认知追问 + 教 AI / 连接追问」是机制层证据的拼接，不存在已验证的整体干预——v8 借鉴是模式移植，不是复制实证干预
- **模板化 scaffold 净效应无量化实证**（They Say/I Say 模板悖论止于方法论主张）；**卡片「意外连接」激发原创无 RCT**（Zettelkasten 同止于方法论主张）
- **AI 时代学习-思考-表达融合缺直接一手实证**：A 级证据分居两端（无 AI 的生成学习元分析 / 解题型 AI 家教 RCT），「AI 会话中让 human 先写/先教」的中间地带为空白——恰是 v8 所在位置
