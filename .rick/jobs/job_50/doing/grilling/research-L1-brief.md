# research-L1 调研任务书（grilling L1 层）

调研主题：生产级 LLM 导师系统的教学操作化设计（human-loop v7 设计输入）。这是 rick job_50 的 grilling L1 层调研。
简报落盘：/Users/logic/workdir/AI_CODING/rick/.rick/jobs/job_50/doing/grilling/research-L1.md（write 首块 + bash 分批追加，每批 ≤60 行）。

背景（一行）：rick human-loop v7 要按 ADR（ZPD 认知负荷管理 + 检索式复习探针 + 真实异质性原创启发 + 更优替代反馈）自研 skill 集；已有调研覆盖了学习科学实证（检索练习/间隔/专长反转/Nemeth 真实异议/Bastani 护栏），但缺**操作化层**：业界系统怎么把判级/吸收/节奏/锚做成具体规则。

## 调研清单（逐条澄清到高置信度，附来源/信源等级）

1. **掌握度判级操作化**：Khanmigo / LearnLM / GenTutor / Socratic by Google 等 LLM 导师如何从学生自由文本回答映射到掌握分档——具体 rubric、判据（如错误断言数/关键概念覆盖）、判级 prompt 的公开形态（LearnLM pedagogy spec、Khanmigo 公开文档）
2. **误差吸收（提取失败 vs 不会）**：生产或学术系统怎么区分「一时想不起来」和「真不会」——二次探针/换措辞/延迟复测/部分线索，具体规则与出处（Khanmigo hint 系统、teach-back 文献操作化）
3. **节奏触发（advance/regress）**：LearnLM pedagogy 指南五原则中「follow the student / pace」的操作化——什么信号推进、什么信号倒退；其他 LLM 导师系统的 pacing 规则
4. **跨会话锚存储**：tutoring/learning 系统在会话间持久化什么供下次 retrieval/复习用（misconception 日志、concept index、掌握度状态）——ALEKS BKT 状态、GenMentor 论文里的跨会话机制
5. **真实异质性工程**：LLM 系统里如何构造「真实异议」（非角色扮演）——多模型辩论/异构信源 ensemble/真实人类分歧记录，有无实现或论文（区别于 devil's-advocate prompting）
6. **文本绘图（ASCII/text diagram）教学实践**：有实操案例/最佳实践/失败模式吗

## 约束

- 只报操作化事实（规则/rubric/机制形态），不重复学习科学实证（已覆盖）
- 每条结论附来源与信源等级（A=同行评审或一手文档，B=权威文档/综述，C=线索）
- 简报 ≤3000 字，事实性结论 ≤12 条按优先级排列，R7 上报 ≤8 条
- 最终回复 = 回执（路径 + 要点 ≤300 字）
