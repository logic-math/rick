# research_1 — human-loop 改进·领域全景（事实还原）— 2026-10-21

元信息：任务模式 facts（为教学大纲提供领域全景）。信源等级：A=一手研究/同行评审/经典教科书/工作区协议原文；B=官方文档/权威综述/作者一手未评审；C=线索。核验状态分两类：「本轮在线核验」（工作区直读 / 叶子 DOI 核验）与「知识库经典·本轮未复验」（网络通道本轮限流，见 Gaps）。

## 事实性结论

**Q1 human-loop 现状机制（信源 A=工作区原文，本轮直读核验）**

- **当前协议 v6 为三角色教师会话**：parent（编排+逐字呈现）+ research（SENSE 第1步事实还原，fresh 上下文，独写 research/ 与 values.md 追加）+ teach（第2-5步+全部呈现内容，fresh 上下文）；显式 pi 触发语法，60 分钟超时上限 — 前提：loop_12/prompts/sense_loop.md 原文 —（A）
- **「输出长」的结构性来源（协议明文+实测）**：① teach 正文必须全文返回、parent 逐字呈现（「不改写、不摘要」）；② lesson 三硬要求（引注+2-4 启发式追问+荐 primary source）+「还原原创作者完整 SENSE 过程」写作要求；③ loop_11 实测 syllabus 22.7KB + 9 课 131KB + values 5.3KB ≈ 160KB/会话、单课 15-20KB；④ 简报上限宽（research ≤3000 字/轮、叶子 ≤1000 字/个） —（A，bash wc 实测）
- **「高延迟/高 token 成本」的结构性来源**：① 每次 dispatch 均 fresh 上下文，research/teach 各重读约 6 份 canon/材料文件（skill_sense+skill_research/teach+mission+values+research_N+judgment）；② research→teach 强制串行（禁止并行）；③ 每教学回合固定四段（认知强制提问→派发→呈现→记 judgment）；④ 协议自记上游单轮 TTFB 可达 8 分钟 —（A，协议原文）
- **协议演化**：loop_4（2026-08）旧协议 = S/E/N/S-R/EC 阶段序列 + think/research/exporter 三代理 + 批判门禁重试（≤5 次）+ 逐派发文件 + RFC 产出；v6（loop_11/12）= 教师-学生会话，删 think/exporter/RFC/阶段标签/门禁重试，改为 mission→research→价值裁决（阻塞）→syllabus 确认→逐课教学→考核演练状态机 —（A，loop_4 dispatch 文件 vs sense_loop.md 对读）
- **四份 RFC 核心主张（human 既定判断「不够有原创性」）**：R1(08-02) 四文件架构+R7 规则+三层复核；R2(08-10) rick=引导程序、G′ 问题需迭代、LLM=有损非确定压缩（已 human 确认 YE1/YE3）；R3(08-14) 三层金字塔+spec 信息内核、243 处自然语言触发词→0 处 pi 语法对齐；R4(08-28) pi 裁决+人机双循环度量体系 —（A，RFC 原文；判断见 mission.md）

**Q2 学习科学已证机制（部分经叶子核验，其余知识库经典·未复验）**

- **检索练习/间隔重复**：测试式提取比重复学习更利长期保持（Roediger & Karpicke 2006, Psych Science 17(3):249-255）；间隔优于集中、最优间隔随目标保持期增长（Cepeda et al. 2006, Psych Bull 132(3):354-380，317 项实验元分析）— 交互含义：让学习者产出/提取而非接收再认 —（A·知识库经典，本轮未复验）
- **认知负荷理论**：工作记忆有限、须管理外在负荷（Sweller 1988；2019 扩展版）；worked example 效应=新手学样例优于自由解题（Sweller & Cooper 1985, DOI 10.1207/s1532690xci0201_3，本轮叶子交叉核验）；**expertise reversal**=对新手有效的技术对有经验者可失效负效（Kalyuga et al. 2003, DOI 10.1207/S15326985EP3801_4，本轮叶子核验）— 交互含义：引导量须随专长动态调整 —（A）
- **desirable difficulties**（Bjork 1994）：间隔/交错/生成式等适度困难损害短期表现但提升长期保持与迁移 — 交互含义：即时流畅感与长期习得可反向 —（A·章节经典，本轮未复验）

**Q3 AI 辅导业界（本轮在线核验，详见 research/leaf-q3.md）**

- **Google LearnLM**：教学法指令遵循+微调 Gemini（对齐主动学习/认知负荷管理/元认知等五原则）；回复刻意更短（均长 297.6 vs 423.0 tokens）；专家偏好强度 +31% vs GPT-4o（Gemini 1.5 Pro 姊妹报告）；Eedi RCT（N=165）新题解决率 66.2% vs 纯人类导师 60.7% —（arXiv:2407.12687 / 2412.16429，B）
- **哈佛物理 RCT**：GPT-4+教学法提示的 AI 导师 vs 课堂主动学习（N=194 交叉设计），中位学习增益 >2 倍（后测 4.5 vs 3.5）、效应量 0.63-1.3 SD、用时 49 vs 60 min；作者限定仅测初学新内容（Bloom 理解/应用层）不外推高阶 —（Kestin et al. 2025, Sci Rep 15, DOI 10.1038/s41598-025-97652-6，A）
- **引导之争收敛边界**：Kirschner/Sweller/Clark 2006（极简引导对新手低效）vs Hmelo-Silver et al. 2007（真实 PBL/探究高度支架化、证据有效）——边界：引导量随先验知识调整，与 expertise reversal 一致；Khanmigo 官方约束「不给答案、引导自答」，无同行评审 RCT（仅评级+定性反馈）—（A/A + B）

**Q4 原创性如何发生 + TFT 传统（本轮在线核验，详见 research/leaf-q4.md）**

- **创造力实证结论**：Boden 三分类（组合/探索/变革型），变革型（改规则使不可能变可能）被论证罕见、多数创造属前两类（概念框架非实验结论）；类比迁移的是关系结构而非表面属性（Gentner 1983）；Dunbar 原位研究：真实科研类比高频但绝大多数为局部类比，远程类比很少直接生成假说 —（A）
- **incubation 元分析**（Sio & Ormerod 2009, Psych Bull 135(1):94-120）：酝酿效应总体为正、条件敏感（准备期长/填充低负荷/发散任务时最大）；adjacent possible（Kauffman 2000；Johnson 2010）：创新=逐级解锁一步可达的新可能，平台使他人层叠建造 —（A/B）
- **tools-for-thought**：Engelbart 1962「放大而非替代」+bootstrapping；Licklider 1960 人机分工（其自析 85% 思考时间耗于准备性文书）；Nielsen/Matuschak mnemonic medium=SRS 嵌入阅读，自报 29% 读者完成 1 月复习层、代价 35-50% 阅读时间、跨领域泛化无证据 —（A/A/B）

**Q5 LLM 对原创性/学习的影响 + 2025-2026 业界（本轮核验受阻，知识库引用，详见 Gaps）**

- **LLM 影响实证（知识库引用·本轮未复验）**：Doshi & Hauser 2024（Science Advances）——AI 辅助提升个体创造力但降低集体产出多样性（同质化）；Bastani et al. 2024（土耳其高中数学 RCT）——GPT-4 辅导提升练习成绩但显著降低无辅助考试表现，加护栏（不给答案的辅导版）无此损害；Gerlich 2025（Societies，问卷相关非因果）——AI 工具频繁使用与批判性思维负相关 —（A 期刊·本轮未复验）
- **2025-2026 业界形态（产品发布层面线索，效果证据未核验）**：Claude for Education learning mode（2025-04）、OpenAI ChatGPT study mode（2025）、Google AI co-scientist（2025-02，AI 协同科研共创）、NotebookLM——共同形态=教学模式开关/护栏式引导（Socratic 化），效果证据多为官方或待独立验证 —（C/B 线索）

## R7 上报项
- Q5 全部条目：本轮 Exa/Crossref/science.org 全通道限流（429/403），无法在线复验 Doshi & Hauser、Bastani、Gerlich 的具体数值与 2025-2026 产品效果证据——缺一手链接复核，下轮网络恢复后优先补验
- Q2 的 Roediger & Karpicke 2006 / Cepeda et al. 2006 / Bjork 1994 / Sweller 2019 扩展版：以知识库经典引用（DOI/卷期已知），本轮未能在线复验原文数值

## Gaps
- Q5「2025-2026 最新工作」的效果证据（study mode / co-scientist / NotebookLM 学习效果的独立评测）完全未核验——需下轮专项调研
- 网络通道本轮不可用（Exa MCP 限流、Crossref 429、science.org 403、DDG 失败）：所有「知识库经典·本轮未复验」条目的事实置信以原发表期刊等级背书，非本轮直接核验
- LearnLM 逐维度专家偏好数值存于论文图表，叶子未取到逐项精确数（leaf-q3 Gaps）
- Khanmigo 效果证据缺同行评审 RCT（leaf-q3，检索范围内结论）
- human-loop 实际 token 成本/延迟的量化数据（每次派发 token 数、每会话总成本）工作区无日志可考——只有结构性事实（fresh 重读、串行、四段回合），无实测数值
- Nielsen「Thought as a Technology」原文链接失效未能核验（leaf-q4）

## 叶子文件
- research/leaf-q3.md（AI 辅导业界，在线核验）
- research/leaf-q4.md（原创性 + TFT，在线核验，含 DOI）
- leaf-q2 / leaf-q5 两次派发均超时未落盘，内容已由主代理以知识库经典补入（核验状态如上标注），不再重派
