# Job 44 设计树：rick 迭代升级（web UI 交互体验 + human-loop 效果优化）

> 活文档，逐层追加（层号 + 模块 + pipeline + 判断节点）。
> Grilling 协议：L1 调研消解 → L2 提炼追问 → L3 批量追问 human → L4 事实回流 → L5 终止判定。

---

## 根层：全局 OKR（O + KR 集；草案经 L3 human 确认）

### 现状事实（L1 轻量自查已消解）

- rick v5.0.13，自进化架构 5.0.11（生产/dev worktree 隔离，`rick tools dev-web`/`release` 通道，RSI 制度 job_43 最近迭代收口）
- **Web UI**：React+Vite+TS。路由 Sessions/Settings/Jobs/Dreams/SessionPage；组件 chat（ChatView/MessageList/ToolCallCard/ThinkingBlock/StreamText/Markdown）、monitor（EventStream/GateResult/TaskBoard）、knowledge（FileTree/MarkdownView）、files（FileReaderPanel）、recovery（RecoveryBanner）、starfield 装饰组件群；stores（ui/sessions/events/jobs/workspaces）
- **human-loop**：`rick human-loop <topic>`，SENSE 5 阶段（S 问题确认→E 视角生成→N1 矛盾生成→N2 主要矛盾→S-R 辩证逆转→EC 良知批判，允许反向回流上限 3）；四文件架构（sense_loop.md 414 行 parent 编排协议 / think.md 171 行 / research.md 172 行 / exporter.md 93 行）；产物 RFC 存 `.rick/draft/rfc/`；v4.4.9 会话持久化 loop_N/session_id 可恢复；web NewSessionModal 可选 cmdType "human-loop" + topic 启动
- **产品战略（human 裁决 2026-09-22）**：Web 优先——新能力以 web 为第一入口；CLI 与 web 同能力
- 最近 job_43 迭代已做：跨工作区文件阅读、工具箱图标语义化、会话信息原文 toggle 修复；PWA 暂缓留遗留

### O（全局目标，草案）

rick 迭代升级：①web UI 交互体验达到「human 实际使用的高频痛点清零」的可验证状态；②human-loop 模式经架构与流程升级后，单次会话的执行成果（RFC/决策质量）获得可验证提升。

### KR 集（草案，待 human 裁决）

- **KR1（web UI）**：human 实际使用痛点清单（P0/P1）逐项修复，每项有可演示的验收标准
- **KR2（human-loop）**：升级后的流程产出物（RFC/judgment）质量优于当前基线，判据明确（human 评审 + 结构化指标对比）
- **KR3（工程）**：全部改动通过既有门禁（go test + npm build + rsi_check 六项机器契约）并按 RSI 流程发布到生产

### OKR 充分性自检（草案）

KR1（交互痛点清零）∧ KR2（产出质量提升有判据）∧ KR3（门禁+发布通道）⟹ O（两条升级线均达成可验证状态）。**缺口**：O 中「可验证」的判据本身依赖 human 裁决（见判断节点 J0-1/J0-2）。

### 根层判断节点（待 L3 追问 human）

- **J0-1「web UI 交互体验」的具体痛点**：human 实际使用中不爽的点是什么？（代码审计只能找客观缺陷，体感痛点只有 human 知道）
- **J0-2「更好的执行成果」的度量**：RFC 质量？思考深度？human 判断负担？如何对比基线？
- **J0-3 两条线的关系与优先级**：human-loop 是否要在 web UI 中有专属交互形态（当前只是 cmdType+topic 启动一个 pi 会话）？两线是独立交付还是互相依赖？
- **J0-4 范围与节奏**：一个 job 内交付两线，还是分期（本 job 只做一条线）？

---

## 根层 L1 调研回填（research-L1，web UI 线）

简报：`grilling/research-L1.md`（12 条结论，A/B/C 信源等级，4 叶子文件）。与根层判断节点直接相关的关键事实：

- **无 ETA 是业界共识**：LangGraph/n8n/Dify 均无预计剩余时间——进度 = 节点状态 + 流式日志 + 耗时统计。rick 若做进度感知应对齐此模式。
- **逐轮审批弹窗已被证伪**（Anthropic 遥测 ~93% 盲批准）→ HITL 应转向「阶段级判断点」而非「工具级弹窗」——这与 human-loop 的阶段判断结构天然契合。
- **中断/恢复/可纠偏是一等公民**：LangGraph interrupt()、Claude Code checkpoint//rewind//branch、Cursor 插话——rick 已有挂起/恢复语义，但 web 呈现层可能不足。
- **编排可视化 = DAG + 节点状态 + 时间旅行**（LangGraph Studio、Dify）——SENSE 五阶段在 web 上结构化呈现有业界成熟范式。
- **web agent 界面须内嵌可验证性**（Copilot Workspace 下线教训）——rick 的门禁/GateResult 已符合，需保持一等公民地位。
- 开发者要素证据强度序：可验证准确性 > 可打断纠偏 > 可回滚 > 后台并行低打扰 > 用量/成本透明。


## 根层 L1 调研回填（research-L1b，human-loop 线）

简报：`grilling/research-L1b.md`（12 结论 + 7 R7 + 3 叶子，arXiv/会议级信源）。关键事实（⟹ 后为对 rick 的启示）：

- **无外部反馈的内在自我修正会降质**（Huang ICLR'24，强证据）⟹ S-R/EC 阶段纯 LLM 自我批判须以 research 事实或 human 反馈兜底。
- **Multi-agent debate 被等算力复现反驳**（Smit ICML'24 等 4 篇，强证据；投票≥辩论有鞅论证）⟹ N1/N2 矛盾判断升级须对照「多采样+投票」基线，不能默认 debate 有效。
- **采样+投票（self-consistency）是最可靠增益**（Wang ICLR'23，多次复现）⟹ E 阶段视角多样性+聚合是低风险升级点。
- **单模型多角色有条件有效**：需对抗/说服压力与独立性设计防视角趋同（中证据）。
- **LLM-as-judge 三大偏差有实证**（位置/长度/自我偏好）⟹ think 打分需换序+多 judge+结构化 rubric。
- **Cognitive forcing functions 可靠降低 over-reliance**（Buçinca CSCW'21，强证据）：「先自判再看 AI/延迟展示」显著降低盲从，仅附加解释无效甚至加重 ⟹ human 判断点应先独立表态再呈现 AI 简报——**这是 human-loop 流程+web 呈现的共同升级点**。
- **对比/排序反馈优于绝对评分**（Christiano'17+InstructGPT 工业实践，强证据）⟹ 裁决点并排对比+差异标注呈现，而非逐一打分。
- **automation complacency 高复现**（系统越可靠人越松懈）；专家自定节奏场景疲劳衰减有反驳 ⟹ 定期注入「应否决」判断点防松懈。
- **RFC 生成：outline→分节填充→验证式修订**（中证据，故事/维基域）+ **claim 级引用+先证据后结论**（ALCE 等，强证据）+ **rubric 化评估是工程标准**（OpenAI evals 指南）⟹ exporter 升级方向 + KR2 判据实现方式。

**两线交汇的实证支撑**：结论 7/8/9（cognitive forcing、对比呈现、complacency 防护）既是 human-loop 流程设计原则，也是 web 判断点交互的设计原则——印证根层 J0-3 交汇假设。

---

## 根层 L3 human 裁决记录（2026-10-05）

**J0-1（web UI 痛点，human 原话）**：
1. 工作区只有 session/jobs/dreams 三页，human-loop 内容无展示 → **应加 humans 页面**，展示 human-loop 过程中产生的各种文件，便于 human 复习。human-loop 本质是学习过程，展示的文件本质是**学习材料与学习进度**。
2. **书写体验痛点**：session 长期无对话被回收/中断 → 对话框中正在输入的内容丢失；页面刷新 → 未发送的草稿丢失。如何提高书写体验？

**J0-2（human-loop 效果，human 原话）**：最大问题是**教学能力不足**，引导过程机械、不流畅。应将 SENSE 流程**内化在教学内容的呈现中**——为 human 呈现一个 SENSE 思考过程，不需要刻意标明每个阶段，而是**隐含在一段话之中**：教学内容本身就包含事实还原、视角选择、主要矛盾界定等内容；引导提问也要**启发**用户进行 SENSE 思考。sense 重要的不是怎么思考——系统论、矛盾论等理论依据**不应作为思考方法限定**模型，可以去除。human 要求：按此愿景提出改造方案。

**J0-3（两线关系，human 裁决）**：必须要 web 呈现，但**不需要**阶段进展、判断点卡片——只需一个 humans 工作区页面，把内容文档呈现出来，human 直接查看即可。human-loop 本身还是一个对话框，与 easy/job 会话无异。

**J0-4（范围，human 裁决）**：本 job 要**讨论出具体设计方案并落地执行**，完成 rick 的自改进升级。

### 根层 OKR 修订（按裁决收敛）

- **O**：rick 自改进升级——①web 新增 humans 学习页面 + 书写体验（草稿持久化）修复；②human-loop 从「机械五阶段引导」重构为「教学内容内化 SENSE 流程」的教学式会话，教学能力可验证提升。
- **KR1（web UI）**：humans 页面上线（human-loop 产物=学习材料+学习进度可浏览复习）；草稿不因 session 回收/页面刷新丢失。
- **KR2（human-loop）**：重构后的 human-loop 产出的教学简报/RFC 质量有可验证提升（rubric 评审 + human 评审，参照 research-L1b 结论 11）。
- **KR3（工程）**：门禁（go test + npm build + rsi_check）全绿并按 RSI 发布。


---

# 第 1 层：模块划分与 pipeline（L1 事实自查已消解）

## 关键代码事实（自查）

- **draft 目录**：`<ws>/.rick/draft/{loops/loop_N/{briefs/,prompts/,judgment.md,session_id}, rfc/rfc-*.md, concepts/, human-learning/, rsi/}`；loops.md/progress.md 为 v2 遗留
- **humans 页面可镜像**：KnowledgeBrowser 用 `GET /api/workspaces/{ws}/knowledge/{tree,file}` 读 domain/loops/skills——humans 页面同模式读 `draft/` 即可（通用文件树天然兼容新旧 loop 布局）
- **草稿丢失根因**：ChatInput.tsx 的 text 是纯 `useState`——刷新/重挂载/路由切换即丢；无任何持久化。localStorage 模式已有先例（sse.ts 游标 rick-web-sse-cursor、token、starfield mode）
- **session 回收**：Supervisor `IdleTimeout` 默认 30min（`IdleDisabled=-1s` 禁用语义已存在，job_43），reapLoop 杀 worker → 会话 suspended → RecoveryBanner 提示一键恢复（绝不自动续跑）
- **human-loop 模板链**：`internal/prompt/templates/{sense_loop,think,research,exporter}.md`（embed）→ builder.SaveHumanLoopPrompt 写 `loop_N/prompts/` → agent 定义由 `internal/env/agents.go deployRickAgents` 幂等装到 `AgentDir()/agents/*.md`（system prompt=skill 源码正文）。think agent 的唯一消费者是 sense_loop.md
- **执行位置**：当前会话在生产树（main 分支）——RSI 硬约束：代码改动必须在 dev 工作区（/workdir/sunquan20/rick-dev，dev/self-evolve）完成，`rick tools release --merge-source` 是 main 唯一写入口。job_43 先例：dev 树持 job 数据，merge 后 main 获副本

## 模块划分

**模块 A（web UI 线）**
- A1 humans 页面：`/ws/:wsId/humans` 路由 + 后端 humans/tree|file API（读 draft/）+ 前端浏览组件
- A2 书写体验：ChatInput 草稿持久化（localStorage 键 rick-web-draft-{sessionId}）+ idle 回收配置化

**模块 B（human-loop 线）**
- B1 sense_loop.md → 教师协议重构（去机械性，SENSE 内化于教学）
- B2 agent 架构调整（think 去留 / exporter 职责）
- B3 学习材料落盘布局（teach 节拍 + judgment.md + rfc）

**模块 C（验证与执行）**
- C1 KR2 评估方案（试运行 + rubric）
- C2 RSI 执行流程（dev 树实现 → 层门禁 → 人类确认 → release --merge-source）

## pipeline

```
B1+B2+B3（协议/agent/布局重构）──┐
                                ├→ A1（humans 页面，通用树对 B 布局弱依赖）→ C1（试运行评估）→ C2（发布）
A2（草稿持久化，独立）──────────┘
```

## B1 教师协议重构方案（human 要求提出的改造设计，待确认）

**核心转变**：从「阶段机器」（parent 按固定 S→E→N→S-R→EC 派发+门禁+回流）→「教师会话」（parent 是一个 SENSE 教师，教学内容内化流程）。

1. **去阶段化**：删除 S/E/N1/N2/S-R/EC 阶段序列、阶段标签、每阶段固定追问集、N1/N2 硬性跳转约束、回流计数器。改为「教学职责清单」：还原事实（S）、呈现视角（E）、洞察矛盾（N）、启发逆转（S-R）、良知批判（EC）——作为教师应自然覆盖的关注面，**不规定顺序不规定话术**。
2. **去方法论强制**：系统论描述符/三维打分从协议中删除（理论依据可在教学内容中作为领域知识自然出现，不作流程约束）。
3. **教学叙事**：教师的每一段重要讲述=一段流畅教学（含事实还原+视角+张力的隐含编织），教学主张 claim 级标注来源信源。
4. **启发式提问**：提问跟随对话自然生成，激发 human 自己的 SENSE 思考（如「你看到哪两股力量在拉扯？」），不宣布阶段。
5. **认知强制模式**（research-L1b 结论7实证）：呈现 AI 分析前先请 human 表态直觉，抑制盲从。
6. **事实锚定**（research-L1b 结论1实证）：关键主张派 research 落盘简报，教学不凭空推理；批判以事实/human 反馈为据。
7. **跟随而非牵引**：human 跳跃则跟随，错过的关注面在适当时机补回。
8. **产物**：teach/NN-<slug>.md（学习材料）+ judgment.md（human 判断原话=学习进度）+ rfc（成果，outline→确认→fill→claim 级引用）。
9. **保留**：research agent（尽调树+信源加权+自落盘契约）、session_id 恢复、judgment.md 禁写 AI 推理、RFC 大纲先确认、exporter 作为 RFC 固化单写者（可选保留）。

## 判断节点（第 1 层，待 L3 human 裁决）

- J1-1（B2）：think agent 移除？exporter 去留？（推荐：think 移除——打分方法论已删、门禁职责并入教师认知强制；exporter 保留为 RFC 固化单写者——上下文经济+大纲两阶段已验证）
- J1-2（B3）：teach 节拍落盘粒度（推荐：重要教学节拍落盘，非逐消息全存）
- J1-3（A2）：idle 配置化方案（推荐：config.json web.session_idle_timeout，默认 30min 保持，负值禁用）
- J1-4（C1）：评估方案（推荐：真实主题试运行一次，human rubric 评审 5 维 + 结构指标）
- J1-5（C2）：执行位置（推荐：实现 worker 以 dev 树为 cwd，job 数据同步 dev 树，RSI 全流程收口）

---

## 裁决记录（第 1 层，L3 round 2，2026-10-05）

**J1-1（agent 架构，human 裁决）**：
- **think 保留**——它是进行 SENSE 思考的主体。但大幅简化：只告诉它 SENSE 思考流程 = **还原事实、呈现视角、洞察矛盾、启发逆转、良质批判**，解释每个思考概念的**概念本质**，**不再阐述方法论**（怎么还原事实/怎么呈现视角/怎么洞察矛盾/怎么逆转/怎么良质批判——全部变简单）。
- **subagent 依旧是 think + research**。
- **exporter 整个删除**，删除 exporter 逻辑。human-loop **彻底仅关注 human 的学习、思考、表达**——表达本身就是每一次 human 做出的判断（judgment.md）。不再需要 exporter；若想要总结，单独启动一个 easy job 对 human-loop 学习过程进行总结——**exporter 的作用被降级为 easy job**。
- **下游消费**：easy job 或 plan 直接消费 human-loop 的整个工作文件夹，自由获取有效信息指导接下来的工作。
- **human-loop 仅关注激发 human 的高质量原创性判断**。

**J1-2（学习材料组织，human 裁决）**：按节组织。一节知识的信息量**调研并参考** https://github.com/mattpocock/skills/blob/main/skills/productivity/teach/SKILL.md （尽可能调研）：该 skill 会 research，把 research 的实际事实信息**整理到学习材料的目录**中，教学过程中引用找到的事实性资源；**教学节奏设计参考该 skill**，每一次对话的信息量需要被把控。**唯一区别**：教学内容要引导学生进行 SENSE 思考——最重要，只有从 sense 出发才能让学生彻底理解事物本质。

**J1-3（idle 回收）**：按推荐——config.json `web.session_idle_timeout`（默认 30min，负值禁用）+ ChatInput 草稿 localStorage 持久化。

**J1-4（评估）**：按推荐——真实主题试运行 + 5 维 rubric（教学流畅度/SENSE 内化/事实锚定/提问启发性/RFC→改为学习成果质量）+ 结构指标。

**J1-5（执行位置）**：是——dev 工作区跑，通过后再合并。

## 事实回流：新调研需求（L4）

human 指定 teach skill 为教学节奏设计基准 → 派 research-L1-r2 调研（进行中）：①该 skill 教学流程组织 ②每次对话信息量控制规则 ③学习材料目录组织 ④事实资源引用方式 ⑤练习/检验环节；+认知负荷/chunking 在 LLM 教学的应用。

**注意**：J1-1 human 原话有一处截断（「exporter 要整个删除，而是直接叫做 ，删除 exporter 逻辑」）——待确认解释：exporter 删除后，教学节拍（teach 材料）由 think（SENSE 思考主体）产出，research 整理事实到学习材料目录。下一轮与叶子层问题一并确认。

---

## 调研回填：teach skill（research-L1-r2）

简报：`grilling/research-L1-r2.md`（12 结论，仓库 main@49dd158 一手精读）。关键设计输入：

- **文件状态机驱动**（无 prompt 内循环）：MISSION 缺→面试；RESOURCES 缺→调研；每会话 1 课。课程状态在文件夹不在对话，新会话冷启动续课=验收标准。
- **信息量=范围型规则**：一课一 tightly-scoped 事 + 一次 tangible win + working memory 内；无数字规则（CLT 文献同：chunking+ZPD 校准+负荷分型）。
- **双难度原则**（Bjork）：知识阶段难度是敌人，技能阶段难度是工具。
- **learning records 高门槛**：证明理解才写；「Coverage is not learning」。
- **材料 7 件套**：MISSION.md / RESOURCES.md（信源库五律：high-trust only、逐条注释用途、显式 Gaps、prune ruthless、每课引注+荐 primary source）/ lessons/0001-*.html / reference/*.html / learning-records/ / assets/ / NOTES.md。
- **检验**：quiz 紧反馈回路（不引入 rick——human-loop 的检验=判断本身）。
- **避坑四项**：无入口评估（rick 以 mission 面试+SENSE 事实还原天然替代）、无退出准则（rick 需设计）、quiz 位置偏差（N/A）、幻觉核查成本（research 信源加权已覆盖）。
- **R7-2**：think+research 双 subagent 教学结构无先例——分工边界 rick 自行设计（见叶子层 J2-2）。

---

# 第 2 层（叶子层）：实现设计

## 模块 B 叶子：human-loop 教师会话 v4

### B1 parent 协议重写（internal/prompt/templates/sense_loop.md，文件名保留内容全重写）

**角色**：教师会话编排者。无阶段序列、无门禁、无回流计数器、无方法论强制。
**流程（文件状态机，参照 teach skill）**：
1. `mission.md` 缺失 → 面试 human：为什么学这个 + 期望状态 + 起点（入口评估，替代 S 阶段）
2. `resources.md` 未充实 → 派 research 尽调（落盘 briefs/ + 整理 resources.md）
3. 每个教学回合：按需派 research（事实缺口）→ 派 think 产教学节（lessons/NN-slug.md，一课一事）→ 呈现 human → human 判断 → 记录 judgment.md
4. 认知强制：呈现分析前先请 human 表态直觉（「在你看来……？」）
5. 跟随 human 跳跃，SENSE 关注面（还原事实/呈现视角/洞察矛盾/启发逆转/良质批判）作为教师覆盖清单在 mission/教学推进中自然覆盖，不标阶段
6. 退出：mission 学习目标达成信号 + human 明示完成（「我认为想清楚了/学习完成」）→ 收尾（不再产 RFC）
**保留**：session_id 恢复、judgment.md 只写 human 原话禁写 AI 推理、派发显式语法（workflowScript+timeoutMs）、交付门禁（回执+文件校验：wc -c + grep '^## '）
**删除**：S/E/N1/N2/S-R/EC 全部阶段内容、系统论描述符、三维打分、固定追问集、批判门禁 think-gate 流程、exporter 派发、RFC 产出、「禁止倾向性」改为「教学有据、不替 human 判断」

### B2 think 重写（internal/prompt/templates/think.md + skills/think skill 同步）

**角色**：SENSE 思考主体 + 教师。读 resources.md/briefs/，产教学节。
**SENSE 五关注面（仅概念本质，无方法论）**：
- 还原事实：把事情本来面目从噪音中剥离出来（什么确凿发生了、什么只是说法）
- 呈现视角：同一事实在不同立场/学科/尺度下的不同含义
- 洞察矛盾：系统中相互拉扯、不可兼得的力量；找到决定全局的那一对
- 启发逆转：若 X 是必然前提，实现 Y 应当如何——把阻碍转化为条件
- 良质批判：回到内心判断——这个理解经得起「如果错了最可能错在哪」的自问吗
**教学纪律（teach skill 移植）**：一课一事 + 一次有形收益 + working memory 内；每课引注（resources.md 信源）+ 荐一个 primary source；知识阶段难度是敌人（讲清优先于考倒）；提问激发 SENSE 思考（「你看到哪两股力量在拉扯？」式，不宣布在练什么）
**产出**：lessons/NN-slug.md（NN 四位递增编号；一节=完整教学叙事：发生了什么→这个领域知识长什么样→启发性追问）；落盘契约沿用（≤2500 字一次 write 或分批；最终回复=回执）

### B3 research 调整（internal/prompt/templates/research.md + skills 同步）

**角色**：现有尽调树+信源加权职责全部保留；**新增**：每次尽调后将事实信源整理进 `resources.md`（RESOURCES 五律：high-trust only、逐条注释用途、显式 ## Gaps 段、prune ruthlessly、按 Knowledge/Wisdom 分组），briefs/ 保留原始尽调简报。
**exporter 删除**：templates/exporter.md 删除、skills/exporter 删除、internal/env/agents.go 移除 exporter spec、internal/builder pibuilder.go SaveHumanLoopPrompt 不再生成 exporter prompt（签名调整）、rfc_dir 变量从模板注入中移除、PrepareHumanLoopDirsIn 仍建 rfc/ 目录（历史兼容， humans 页展示旧 RFC）

### 学习材料目录（新 loop 布局）

```
.rick/draft/loops/loop_N/
  mission.md        # 学习使命（why+期望+起点面试结论）
  resources.md      # 信源库（research 维护，五律）
  lessons/NN-slug.md# 教学节（think 产出，学习材料主体）
  judgment.md       # human 判断原话时间线（学习进度/表达记录）
  briefs/           # research 原始尽调简报
  prompts/          # 会话 prompt 文件（机制不变）
  session_id        # 恢复（机制不变）
```
humans 页面通用文件树天然兼容新旧布局（旧 loop 的 briefs/teach-*/judgment.md 照常展示）。

## 模块 A 叶子

### A1 humans 页面
- 后端：internal/web/humans.go（新）——handleHumansTree/handleHumansFile 读 `<ws>/.rick/draft/`，镜像 knowledge.go 模式（路径校验/白名单语义与 job_43 裁决一致：仅文件+大小上限）；routes.go 注册 `GET /api/workspaces/{ws}/humans/tree` + `/humans/file`
- 前端：web/src/routes/Humans.tsx（镜像 Dreams.tsx 骨架）+ components/humans/HumansBrowser.tsx（左树右内容双栏，镜像 KnowledgeBrowser 交互）；App.tsx WorkspaceRoutePage kinds 加 "humans" + 导航项
- 纯文件树+内容查看（human 裁决：不做阶段进展/判断卡片）

### A2 草稿持久化 + idle 配置化
- ChatInput.tsx：挂载恢复 localStorage `rick-web-draft-{sessionId}` → 输入防抖 500ms 写入 → 发送成功 remove；try/catch 静默降级（隐私模式）；sessionId 经 props 从 ChatView 传入
- config.json `web.session_idle_timeout`（分钟，默认 30，负值禁用）→ Supervisor IdleTimeout 接线（构造点在 web 命令组合根）；config schema + 文档同步

## 模块 C 叶子

### C1 评估（KR2 验证）
- 实现完成后在 dev 实例跑一次真实主题 human-loop（human 选主题参与判断）
- 5 维 rubric：①教学流畅度（非机械感）②SENSE 内化质量（五关注面隐含覆盖）③事实锚定（resources.md 五律+每课引注）④提问启发性（是否激发 human 原创判断）⑤学习成果质量（judgment.md 判断密度与质量 + lessons 材料价值）
- 通过线：human 评审 4/5 达标 + 结构指标（lessons 节数 ≥3、judgment ≥5 条、每课有引注）

### C2 RSI 执行流程
- job_44 数据同步 dev 树（.rick/jobs/job_44/）；实现 worker cwd=/workdir/sunquan20/rick-dev
- 层门禁（plan/gates/，human 确认后不可改）：
  - gate1：模板/agent/builder 链——go build + go test ./internal/prompt/... ./internal/env/... ./internal/builder/... + `rick human-loop --dry-run` 渲染断言（新协议关键词在、exporter 消失、think 五关注面在、无系统论描述符表）
  - gate2：web 后端——go test ./internal/web/...（humans API 树/文件/越权/大小上限测试）+ idle timeout 配置接线测试
  - gate3：前端——npm run build + tsc + 草稿持久化逻辑（组件单测或构建断言）
- 试运行评估（C1）→ human 确认（approval.md）→ `rick tools release --merge-source` → rsi_check 全绿

## 叶子层判断节点（待 human 最终确认）

- J2-1：截断句解释——think 吸收教学职能（教师=think），exporter 全删（含 agent 定义/模板/skill/builder 链）
- J2-2：resources.md 维护者=research（信源加权专家天然对齐 RESOURCES 五律）；think 只读不写 resources.md
- J2-3：退出准则=mission 达成信号 + human 明示完成（无硬性阶段门、无自动退出）
- J2-4：旧 loop 兼容——sense_loop.md 文件名保留（内容全重写）；旧 loop 目录原样展示；新 loop 不产 RFC
- J2-5：quiz 不引入（判断即检验；认知强制=检验机制）
- J2-6：gates 三层方案（gate1 模板链 / gate2 web 后端 / gate3 前端）+ 试运行评估后发布

---

## OKR 充分性自检（第 1→2 层，L5）

- KR1（humans 页面+草稿体验）⟸ A1（humans 页面：通用树+内容查看）+ A2（localStorage 草稿+idle 配置）——充分 ✓
- KR2（教学能力可验证提升）⟸ B1（去机械性教师协议）+ B2（think=SENSE 思考主体，五关注面概念本质）+ B3（research 信源库+exporter 删除）+ teach skill 节奏移植（一课一事/引注/文件状态机）+ C1（试运行 rubric 验证）——充分 ✓
- KR3（门禁+RSI 发布）⟸ C2（三层 gate + dev 树执行 + release --merge-source + rsi_check）——充分 ✓
- MECE 检查：A（web 呈现/体验）、B（协议/agent/材料）、C（验证/执行）互斥不重叠；A1 依赖 B3 的目录布局但以通用树解耦 ✓

---

## 叶子层 L3 human 最终裁决（round 3，2026-10-05）——设计冻结

**J2-1（架构，human 裁决）**：重构后**只剩一个 human-loop 主体提示词**（sense_loop.md 文件名保留，调度全局节奏）+ **两个 subagent**：
- **research**：输入=历史上下文 + human 需求；输出=调研事实，沉淀 `research_{N}.md`
- **teach**（think.md 删除改名）：输入=`research_{N}.md` + 历史对话；做 SENSE 思考并产出教学课程 `lessons/teach_{N}.md`；**教学内容须引导 human 进行 SENSE 思考**；所有呈现在会话中的内容都由 teach 产出
- **exporter 删除**（全链）；**think.md 删除 → teach.md**

**J2-2**：是（research 沉淀事实材料；材料形态= research_{N}.md 文件，不再单列 resources.md——五律语义由 research 的信源加权+R7 gaps 承担）

**J2-3**：按推荐 + **mission.md 设定学习目标与考核标准**

**J2-4**：think.md→teach.md、其余文件不变、builder 及测试链暴露问题则修复

**J2-5**：是（暂不引入 quiz；判断本身即表达与检验；**teach 隐式检查 human 判断作为隐式 quiz**）

**J2-6**：按推荐（三层 gate + 试运行评估 + release --merge-source）

### 冻结后的完整产物设计

**模板/技能文件变更**（写域 A）：
| 文件 | 动作 |
|---|---|
| `internal/prompt/templates/sense_loop.md` | 重写（教师会话主编排：mission 面试→research 派发→teach 派发→呈现→judgment 记录→退出准则；无阶段/门禁/方法论） |
| `internal/prompt/templates/think.md` | **删除** |
| `internal/prompt/templates/teach.md` | **新建**（SENSE 思考主体+教师；五关注面仅概念本质；一课一事；引注；隐式 quiz；引导 human SENSE 思考；产出 lessons/teach_{N}.md） |
| `internal/prompt/templates/exporter.md` | **删除** |
| `internal/prompt/templates/research.md` | 调整（输出 research_{N}.md；信源等级+A/B/C+## Gaps） |
| `internal/prompt/templates/skills/think.md` | **删除** → `skills/teach.md` **新建** |
| `internal/prompt/templates/skills/exporter.md` | **删除** |
| `internal/prompt/templates/skills/research.md` | 微调（输出路径约定 research_{N}.md） |

**Go 代码链变更**（写域 B）：
| 文件 | 动作 |
|---|---|
| `internal/env/agents.go` | rickAgents：think→teach（name/description 重写）、删除 exporter spec |
| `internal/prompt/human_loop_prompt.go` | 变量 think_agent_path→teach_agent_path、删 exporter_agent_path；WriteSkillFile skill_teach.md；LoadTemplate("teach")；返回值 4 文件→3 文件（main/teach/research） |
| `internal/builder/pibuilder.go` | SaveHumanLoopPrompt 签名与调用适配（返回值数量） |
| `internal/handler/human_loop.go` | 适配 SaveHumanLoopPrompt 返回值 |
| `internal/prompt/human_loop_prompt_test.go` | 适配 think→teach/exporter 消失 |
| `internal/env/env_test.go` / `update_test.go` | 适配 agent 清单（teach 替代 think+exporter） |
| config（ThinkTopN/ThinkMinAssumptions/SenseMaxBackflows） | **保留不删**（无其他消费者、保留向后兼容；新模板不再注入）——死字段清理留后续 job |

**新 loop 产物布局（学习材料）**：
```
.rick/draft/loops/loop_N/
  mission.md              # 学习目标 + 考核标准 + 起点（面试产出）
  research/research_{N}.md# 事实材料（research 产出，含信源等级+Gaps）
  lessons/teach_{N}.md    # 教学节（teach 产出，学习材料主体）
  judgment.md             # human 判断原话时间线（表达=学习进度）
  prompts/                # sense_loop + teach + research（3 文件）
  session_id              # 恢复机制不变
```
旧 loop（briefs/teach-*/judgment.md）由 humans 页面通用树原样展示。

**退出准则**：mission.md 学习目标达成 + human 明示完成 → 收尾（无 RFC 产出）。parent 以良质批判的自然形态询问收敛（不自动判定）。

### 任务分解与写域（执行阶段 DAG 草案）

| 层 | task | 写域 | 门禁 |
|---|---|---|---|
| L1 | task1：human-loop v4 协议重构（上表「模板/技能文件变更」+「Go 代码链变更」全部） | `internal/prompt/templates/**`, `internal/prompt/*.go`, `internal/builder/*.go`, `internal/handler/human_loop.go`, `internal/env/*.go` | gate1：go build + go test 上述包 + `rick human-loop --dry-run` 渲染断言（teach/research 路径在、exporter 痕迹消失、五关注面在、无系统论描述符/三维打分/阶段序列、research_{N}.md 与 lessons/teach_{N}.md 路径在） |
| L2 | task2：humans 页面后端 API | `internal/web/humans.go`, `internal/web/routes.go`, `internal/web/humans_test.go` | gate2：go test ./internal/web/...（tree/file/越权/大小上限） |
| L2 | task3：idle 配置化后端接线 | `internal/config/*.go`(session_idle_timeout 字段), `internal/handler/web.go` 或 `internal/cmd/web.go`, 相关测试 | gate2 同上（含 config 解析断言） |
| L2 | task4：humans 页面前端 + 草稿持久化 | `web/src/routes/Humans.tsx`, `web/src/components/humans/**`, `web/src/App.tsx`, `web/src/components/chat/ChatInput.tsx`, `web/src/components/chat/ChatView.tsx` | gate3：npm run build + tsc + 草稿持久化断言 |
| L3 | task5：dev 实例试运行评估（C1）+ RSI 证据（C2） | `doing/rsi/**`, `doing/evaluation*.md` | gate4：rsi_check --json pass:true（G1-G4） |

（gate 为最终形态草案，human 确认后落 plan/gates/gate{N}.py，不得改动。）

---

## 层-调研映射（L5 终止判定证据）

- **第 1 层（模块划分与 pipeline）**：research-L1.md（web UI 交互业界基准：进度/中断恢复/HITL/可视化）+ research-L1b.md（human-loop 深度思考与 HITL 方法论基准：自我修正需外部feedback、MAD 被反驳、cognitive forcing、对比呈现、complacency、RFC 生成）——两线模块与 pipeline 的业界依据齐备。模块级轻量事实（目录布局、KnowledgeBrowser API 模式、ChatInput 状态、Supervisor idle reap、模板/agent/builder 链）由本人 grep/read 自查消解，另见「关键代码事实」节。
- **第 2 层（叶子层实现设计）**：research-L2.md（= teach skill 一手精读：文件状态机、一课一事信息量规则、RESOURCES 五律、引注要求、避坑清单、grilling sub-agent 事实回流模式、R7 指出「think+research 双 subagent 无先例」）——叶子层的教学节奏/材料组织/检验机制设计依据齐备。
- 层内判断节点 J1-1..J1-5、J2-1..J2-6 **全部上呈 human 并获裁决**（见两节裁决记录），无残留未消解判断节点。


---

## 执行阶段事实修正（L4 回流，2026-10-05）——A2 范围调整

**发现**：空闲回收（30min reap）**已在此前 job_43 中禁用**——`internal/cmd/web.go resolveSupervisorConfig` → `IdleTimeout: runtime.IdleDisabled`（注释：「pi 会话持久持有：只要 server 存续、用户不主动断开，worker 就不被回收」）；`internal/config/config.go` 注释同步确认。故「web.session_idle_timeout 配置化」**无必要，该子项删除**（原推荐基于过时假设）。

**草稿真实丢失路径**（代码核实）：①手动页面刷新 ②`frontend_reload` SSE 事件 → `web/src/api/sse.ts:427` `window.location.reload()`（前端产物每次变更自动触发）③路由切换/组件重挂载 ④会话挂起后重载。根因：`ChatInput.tsx:73` 草稿是纯 React `useState`，零持久化。

**修正后方案**：仅前端 localStorage 草稿持久化（挂载恢复 / 防抖保存 / 发送清除 / 静默降级），**无后端配置变更**。task3 承担；原 task3（idle 配置化）取消。

**对 DAG 的影响**：4 task / 3 层（task1 → task2+task3 → task4），gate1/gate2/gate3 就位，pipeline_gate 通过。

---

# 阶段 2：SENSE 范式重构（human-loop v6，同一 job_44 内完成）

> 触发：human 2026-10-11 对现有 human-loop 的实证诊断（`loop_10` 会话）——「sense 的效果不足」+ 给出 `sense.md` 重构指令。
> 承载：job_44 后续层（新 gate4）。job_45（v5 代码已在 dev 分支）为基线不回退；job_45/46 已停止待命。

## 根层 OKR（本阶段）

- **O**：human-loop 的 SENSE 从「用 SENSE 讲透材料」升级为「**以 SENSE 为思考范式**：AI 用 SENSE 完成一次思考（产出教学大纲），教学单元还原原创作者的 SENSE 过程，并把学生自己的事实/价值/视角/矛盾/逆转逼出来」——`sense.md` 成为**全局统一**的思考范式 canon（human-loop 的 research/teach + dream 共用）。
- **KR1**：`skills/sense.md` 整体重写为五步范式（human 原话为权威文本）：①事实还原=递归调研至常识性置信信息源+**价值性信息必须上报 human 裁决**；②构建视角=第一性原理定知识领域（强相关=收敛 / 弱相关=发散）；③洞察矛盾=用领域术语解构成因+判定全局性/根本性/决定性（多备选供 human）；④辩证逆转=X 阻碍·Y 期望，当 X 是 Y 必然前提时如何实现 Y（把阻碍变手段，多方案供 human）；⑤良质批判=各假设是否足够置信+能否必然得出目标可达成+差在哪/漏了什么。**是范式而非流程**。
- **KR2**：research 接第 1 步（递归事实调研 + 价值清单落盘 `values.md`，批量呈报 human 裁决后阻塞推进）；teach 接第 2-5 步（`syllabus` 任务→产出 `syllabus.md`=一次完整 SENSE 思考留痕+教学序列）。
- **KR3**：教学单元 = **写作要求**（非固定模板）：必须回答①作者当时的原始问题/事实调研/价值判断/知识领域/视角/主要矛盾/辩证逆转 ②教学良质自检（是否结合学生自身知识体系/有无无法理解的信息/是否遵循教学良质；依据仅 mission 起点 + judgment 已表达判断）③追问回到学生自己的问题（构建学生自己的事实/价值/视角/矛盾/逆转）。
- **KR4**：大纲动态变更（teach 改、parent 识别与呈现、human 确认、changelog 留原始提问）；终止 = **考核演练**（human 当老师讲出自己的 SENSE 思考，teach 当学生+教练指出问题环节）收敛为单一形式，通过后终止。
- **KR5**：`method.md` 旧五阶段 SOP 指令冲突清理；dream 与 human-loop 共用新 canon 不破坏（渲染/测试冒烟）。

**OKR 充分性自检**：KR1（范式 canon）∧KR2（分工执行）∧KR3（教学单元）∧KR4（动态+终止）∧KR5（无冲突+兼容）⟹ O ✓

## 判断节点裁决（JN1-JN7，human 2026-10-11）

| # | 裁决 |
|---|---|
| JN1 | **保持全局统一**——`sense.md` 为全局 canon（human-loop 的 research/teach + dream 共用） |
| JN2 | 按推荐：价值信息**阶段末批量呈报 + 阻塞**（`values.md` 写回 human 裁决后才进入第 2 步） |
| JN3 | 按推荐：递归终止三层定义（①来源收敛到 primary/经典教科书·规范·权威综述 ②每步可答"我从哪知道？再往上？"直到不再需要引用 ③回溯不到常识层的**一律不是事实**→转价值/假设上报）；**不设深度硬上限**，收益递减即停，无法收敛入 Gaps |
| JN4 | **不要固定模板**——只要遵循 SENSE 思考流程；大纲随内容变化；本质是**写作要求**（教学内容必须能回答这些问题），不必固定形式 |
| JN5 | **teach 改大纲**（parent 识别触发+呈现+记录；变更需 human 确认；允许插入/重排；changelog 留 human 提问原话） |
| JN6 | 考核演练**收敛为单一形式**：human 当老师、teach 当学生，human 讲出自己的 SENSE 思考，teach 作为**教练**指出其中的问题环节 |
| JN7 | **就在当前 job_44 完成**（本阶段=新层 gate4）；job_45/46 已停止，待 job_44 完成后推进 |

## 层与门禁

- 层：**task5（单写者，单一层）** → `plan/gates/gate4.py`（human 确认后冻结）
- 之后：dev 换构建 → 你在 8414 用**同一篇论文**做对照试运行（v5 vs 新范式）→ 批准 → release
