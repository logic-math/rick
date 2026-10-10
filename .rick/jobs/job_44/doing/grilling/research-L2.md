# research-L2 简报（第 2 层/叶子层：human-loop 教师会话实现设计）

> 本简报 = teach skill 一手调研（原以 L4 事实回流名义派发，落盘 research-L1-r2.md；因它直接支撑叶子层教学设计，同步为第 2 层调研简报）。内容与 research-L1-r2.md 一致。


**主题**：human-loop 教师会话的教学节奏设计一手调研 | **时间基准**：仓库 main@49dd158 快照（2026-02） | **方法**：git clone mattpocock/skills 全仓库精读（teach/SKILL.md 140 行 + 4 个 FORMAT.md + docs 页 + grilling/research/handoff 配套 skills），web 交叉补充认知负荷文献

## 事实性结论

1. **teach 无显式流程循环，靠文件状态机驱动**：SKILL.md 不写「research→整理→教学」循环，而是声明工作区状态驱动——MISSION.md 缺失→第一件事是面试用户（"your first job should be to question the user on why"）；RESOURCES.md 未充实→先找高质量资源（"Never trust your parametric knowledge"）；然后每会话产出 1 课。〔仓库原文直读，一手，0.9〕

2. **单次信息量是范围型规则，无数字规则**：lesson 定义 = "one tightly-scoped thing tied to the mission" + "a single tangible win you can build on"，"short, completable very quickly"，"Learners' working memory is very small, and we need to stay within it"；知识只教获取目标技能所需的最小集（"Lessons should be designed around a skill... knowledge only what's required"）。〔一手 0.9〕

3. **双难度原则（Bjork 框架）**：区分 fluency（当下流畅=虚幻掌握感）与 storage strength（长期留存=真目标）；知识获取阶段 "difficulty is the enemy"（吃掉理解所需工作记忆），技能习得阶段 "difficulty is the tool"（effortful retrieval builds storage strength）；desirable difficulty 三工具：retrieval practice、spacing、interleaving（interleaving 限技能练习）。〔一手 0.9〕

4. **ZPD 由 learning-records 计算，写入门槛极高**：每课须落在用户 ZPD（"challenged 'just enough'"）；learning record = 教学版 ADR，仅在用户**证明**理解时写（4 触发：真理解非平凡概念 / 披露先验知识 / 纠正误解 / mission 变化），明确排除：「Coverage is not learning」、术语表已有内容、会话日志。〔一手 0.9〕

5. **学习材料目录 7 件套**：MISSION.md（why；"runs past a screen 就从 compass 变成 plan"）、RESOURCES.md（信源库，Knowledge/Wisdom 分组）、lessons/*.html（0001- 递增编号，一课一自包含 HTML，"primary unit of teaching"）、reference/*.html（压缩精华；"lessons 很少被复访，reference 会"）、learning-records/*.md、assets/*（可复用组件+共享样式表，"Reuse is the default"）、NOTES.md（用户教学偏好）。GLOSSARY-FORMAT.md 在仓库但 SKILL.md 已不链接（issue #559，glossary 需用户主动要才有）。〔一手 0.9〕

6. **事实引用规则（RESOURCES.md 五律）**：high-trust only（primary sources / 公认专家 / 同行评审，营销内容剔除）；每条必须注释用途（"a bare link is useless in three months"）；显式 ## Gaps 段驱动后续搜索；prune ruthlessly（"better five sharp sources than thirty mediocre"）；记录用户社区偏好。lesson 内要求 "littered with citations"，每课推荐 1 个 primary source 让用户自己去读。〔一手 0.9〕

7. **练习/检验环节**：quiz + 轻量 in-browser 任务，"feedback loop should be as tight as possible"（即时反馈、理想情况自动）；quiz 公平性规则：每选项字数一致、正确答案位置轮换、格式/顺序不泄题。skill 立场强硬：不允许跳过 quiz（用户道谢被拒——"the drill was not finished"）。〔一手 0.85〕

8. **已知缺陷四项（mattpocock docs 自述，对 rick 是直接避坑清单）**：①无入口评估——首会话零 records 时盲猜水平，wayfinder 用户抱怨 "It never did grilling to establish my starting point"（知识评估步骤是 #725 待做特性）；②无间隔重复调度、无退出准则（"good at making the next lesson, but not as good at knowing when to stop and switch to review"）；③quiz 答案位置偏差在 Sonnet/Opus/GLM 上 33/33 复现，prompt 层修不动，真修法是 assets 组件层运行时洗牌（#335）；④幻觉风险：Rubik's cube 用户遇到编造解法——docs 明说 grounding 降低核查成本但不消除核查必要。〔一手（docs 自述+issue 编号）0.9〕

9. **grilling 的 sub-agent 事实回流模式（rick L4 直接参照）**：grilling SKILL.md 明文——"Finding facts is your job, never the user's"：frontier 问题需要环境事实（文件系统/工具）时派 sub-agent 查找，**不阻塞**：运行中的探索视为未决前置，只挂起其下游问题，本轮其余 frontier 照常问；决策永远归用户、逐条等用户回答。配套 handoff：grilling 中途想学新话题→/handoff 到教学工作区用 /teach 学完再回来，不脱轨主线。〔一手 0.9〕

10. **teach 是单 agent 内联架构，无多 subagent 教学先例**：teach 的 research→材料→教学全流程单 agent 完成，"The course state lives in the folder, not in the conversation"；research skill 则是独立后台 agent（读 primary sources、产出单份带引用 md、按仓库存放惯例落盘）。**mattpocock 生态不存在 think+research 双 subagent 的教学结构先例**——分工边界无法从一手来源直接迁移。〔一手 0.9〕

11. **多会话节奏**：docs FAQ 明确三种方式等价——同一会话连上 / 新会话重跑 /teach / 同目录开新会话说 "next lesson for <topic>"；每课是一次独立 invocation，课程状态全在文件夹不在对话；新会话冷启动即续课是验收标准之一（"It's working if" 清单）。〔一手 0.85〕

12. **信息量的教学设计成熟原则（补充调研）**：CLT（Sweller）：工作记忆容量有限→最小化 extraneous load、用分段/chunking 管理 intrinsic load、促 germane load（schema 构建）；worked example 效应对新手显著（aptitude-treatment interaction：低先验知识者受益更大），stepwise worked example（每步先让学习者检索再揭示+反馈）2025 实证提升学习；adaptive lesson chunking 控制实验实证改善词汇学习。**无「每课 N 个知识点」的公认数字**——成熟做法=范围型控制（一课一事）+ 难度校准（ZPD）+ 负荷分型管理。〔同行评审文献多源交叉，二级信源 0.7〕

## R7 上报项

1. 「一节课含多少信息量」无权威数字答案——mattpocock 与 CLT 文献均只用范围型原则（一课一事 + working memory 上限 + ZPD 校准）；若 rick 设计需要硬性数字（如每节 ≤N 概念），需 human 自行定标。
2. think+research 双 subagent 教师会话结构在 mattpocock 生态无先例（teach 单 agent 内联、research 独立后台 agent），「谁做 SENSE 五面思考 / 谁做事实调研」的分工边界无法从一手来源迁移，需 rick 自行设计。
3. teach 的用户反馈证据（#335/#559/#725 等）以仓库 docs 页自述为准，未逐条打开 issue 原文核验（网络受限+浅克隆）；docs 页属一手来源，偏差风险低。
4. 未横向对比其他生态的 LLM 教学 skill（Anthropic 官方 skills、Khanmigo 等）——时间预算内未展开；若需横向对比需追加调研。

## 附：对 rick 教师会话设计的直接映射（供 grilling 提问引用）

- 教学流程 → 文件状态机而非 prompt 内循环：缺什么补什么（mission 缺先面试、resources 缺先调研）
- 信息量 → 「一课一事 + 一次有形收益 + working memory 内」，不设数字
- 五关注面的事实基础 → RESOURCES.md 模式：高信源 + 逐条注释 + 显式 Gaps + 无情修剪；每课引注 + 荐一个 primary source
- 检验 → 紧反馈回路 quiz/任务 + learning record 高门槛写入（证明才算学会）
- 避坑 → 入口评估（rick 可用 SENSE 事实还原替代）、退出准则、quiz 公平性组件化、幻觉核查成本
