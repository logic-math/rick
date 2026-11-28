# 设计树：human-loop v8 彻底重构（激发原创性的学习、思考、表达）

> 需求原文：「彻底重构 human-loop，使其能够启发原创思考，创造性的解决问题。」
> 落盘目录：`/Users/logic/workdir/AI_CODING/rick/.rick/jobs/job_57/doing/grilling/`
> 调研依据：research-L1.md（24 条结论：认知机制/AI 先例/CPS 映射）、research-L1-r2.md（24 条结论：skill 生态先例/writing-to-think/生成性学习）、values.md（v1-v7 价值裁决记录）、research-L1-leaf-1.md

---

## 第 0 层（根层 OKR）

O（全局目标）：human-loop 彻底重写为极简的「原创性学习-思考-表达会话」——现有全部逻辑（教学机制/SENSE 内嵌/文件状态机/历史兼容）删除；human 通过学习提升自己；重构后实跑验证：thought_path.md 中呈现 human 自己的思考轨迹（非复述 AI 内容）。

KR1（调研与新范式定义）：外部「激发原创性学习/思考/表达」的 skill 先例与 writing-to-think 方法论调研清楚（research-L1/L1-r2 共 48 条结论）；新范式经 human 两轮裁决确认（不参考历史逻辑，完全重新思考）。✅ 已达成

KR2（删除与重写落地）：现有 human-loop 全部逻辑删除（sense_loop.md 旧内容、教学六 skills、四旧 agent、状态机、resume 历史兼容负担）→ 新极简逻辑落地（五阶段 skill + orig agent + thought_path 产出）→ `rick human-loop <topic>` 一键启动新会话。

KR3（实跑验证）：以真实主题实跑新会话（dev 分支，human 亲自测试），thought_path.md 呈现 human 的原创性思考轨迹，human 认可后合并。

---

## 第 1 层（范式层）

**调研简报**：research-L1.md + research-L1-r2.md（本层判断节点的依据）

**v8 范式（human 两轮裁决定型）**：

- **目标**：启发 human 进行思考
- **推进方法**：基于 SENSE 五阶段（问题还原 → 视角生产 → 判断矛盾 → 辩证逆转 → 良质批判）的**思考门禁制 loop**——每阶段 = 一个 skill（双重目标：①启发用户深入思考的内容 ②验证用户的思考是否足以通过思考门禁），通过门禁才进入下一阶段
- **门禁执行**：main agent 内联判定；门禁标准 = sense.md 每步可操作定义提取的可判定条件
- **agent 架构**：一个 orig agent——基于用户当前思考进度搜索调研信息，返回启发用户思考的内容；唯一提示词：「给用户返回怎样的内容才能引导他完成本阶段的思考目标？」
- **未过门禁**：AI 一句话指出缺口 + 重调 orig 针对性再启发，无硬性重试上限，推进权在 human（自主性最重要）
- **唯一产出物**：思考轨迹 `loop_N/thought_path.md`（human 回答 + AI 启发的时间线追加式）

**本层判断节点裁决记录**（human 逐条裁决，2026-11-25）：

- 第一轮（9 问 + 3 修正）：全部教学机制删除；SENSE 后续集成（集成方式已定型：五步即骨架）；状态机不要；流程全删要简单有效；本质=通过学习提升自己；历史逻辑不参考；完全重新思考「如何激发人类进行原创性的学习、思考、表达」
- 第二轮（7 问收敛）：门禁 skill 由 main agent 判断；未过门禁 A（自主性）；orig 调用 A（parent 自由裁量）；thought_path.md 时间线；删除清单确认；五阶段 skill 独立文件；命令 `rick human-loop` 不变，loop_N/ 目录保留，orig 中间结构落盘

---

## 第 2 层（叶子层）

**本层判断节点已全部消解**（P1 orig 中间结构格式已由 human 于 2026-11-25 裁决：`stage{N}_{seq}.md`；其余维度为事实性落实，无判断节点）。

**叶子 1：五个阶段 skill 文件**（新建 `internal/prompt/templates/skills/`）：

- `sense_reconstruct.md`（阶段 1 问题还原）/ `sense_perspective.md`（阶段 2 视角生产）/ `sense_contradiction.md`（阶段 3 判断矛盾）/ `sense_reversal.md`（阶段 4 辩证逆转）/ `sense_critique.md`（阶段 5 良质批判）
- 统一结构：①本阶段思考目标（sense.md 该步权威文本）②orig 派发契约（阶段 + 思考进度 + 缺口 → 唯一提示词）③启发呈现纪律（供张力/引导问题，不供答案——research 结论）④门禁标准（sense.md 可操作定义提取）⑤未过门禁行为（一句话指出缺口 + 重调 orig）

**叶子 2：orig agent**：

- `internal/prompt/templates/skills/orig.md`（新建）：唯一提示词（「给用户返回怎样的内容才能引导他完成本阶段的思考目标？」）+ 调研方法（web_search 多 provider / fetch_content / OpenAlex API）+ 返回启发内容（非答案）+ 中间结构落盘 `{loop_dir}/orig/stage{N}_{seq}.md`
- `internal/env/agents.go`：rickAgents 四旧删除 → 仅 [orig]（fullTools，fresh，thinking high，timeoutMs 3600000）

**叶子 3：主模板重写**（`internal/prompt/templates/sense_loop.md` 旧内容全删重写）：

- 极薄编排层：五阶段循环（读阶段 skill → 调 orig → 呈现启发 → human 思考回答 → 门禁判定 → 推进/继续）
- thought_path.md 写入契约（时间线追加：阶段标记 + human 原话 + AI 启发引用；工具层写入不播报）
- 极简输出纪律

**叶子 4：删除清单**：

- skills：删 zpd.md / feynman.md / exploration_stream.md / syllabus_builder.md / lesson_writer.md / recall_review.md / research.md；保留 sense.md（dream 共用 + v8 骨架 canon）
- agents.go：删 research / syllabus-builder / lesson-writer / recall-review
- 历史 loop_2..24 目录不删（数据）

**叶子 5：代码改动**（internal/）：

- `handler/human_loop.go`：入口/dry-run/resume 保留；loop_N/ 目录结构 = thought_path.md + orig/ + session_id
- `builder/pibuilder.go`：SaveHumanLoopPrompt / BuildHumanLoop 渲染新模板（占位符：topic/draft_dir/loop_dir/orig_dir/thought_path_path/五阶段 skill 路径/orig skill 路径/sense skill 路径）
- 测试：`handler/human_loop_test.go` + `cmd/human_loop_test.go` 断言更新

**叶子 6：工具调用与环境**：

- 构建验证：`go build ./... && go test ./internal/handler/ ./internal/cmd/ ./internal/env/ ./internal/builder/`
- agent 部署：deployRickAgents 幂等部署 orig（rick tools update-pi / 启动时）
- 实跑验证：dev 分支 `rick human-loop <真实主题>`，human 亲自测试 thought_path.md；通过后合并

---

## 附录（过程归档，非层）

**grilling 过程**：根层 OKR 建立 → L1 调研（research 派发，多次故障后三跑收齐 48 条结论）→ L3 第一轮批量追问（9 问）→ human 裁决彻底推翻（8 条：全删教学/SENSE 后续集成/无状态机/极简/调研新方向/学习提升自己/不参考历史/重新思考三维度）→ L4 回流调研（research-L1-r2：外部 skill 先例 + writing-to-think + 生成性学习）→ L3 收敛轮（7 问 M1-M7）→ human 裁决 v8 范式（SENSE 五阶段门禁制 + orig agent + thought_path 唯一产出）→ 叶子层落实（P1 裁决 stage{N}_{seq}.md）→ 开工指示（dev 测试后合并）。

**v7 现状事实（L1 自查，供删除清单参照）**：教学会话结构（mission→values→zpd→syllabus→课程回合→探索流收口）；四 agent；parent 内联三方法论；SENSE canon 与 dream 共用 sense.md；实跑 loop_24 证明真实用法是创造性解决问题而非学课程——结构错配是重构动机。

---

## v8.1 迭代（2026-11-26，orig 首次实测后的 human 反馈修正）

**实测背景**：orig-test（原创性思考/科研品味检索）成功返回启发内容，human 反馈「信息的呈现不够优秀」+ 三层修正裁决：

**修正 1（呈现层）——新增教学润色 skill**：orig 返回的启发信息经教学 skill 润色后呈现，科普视角写作；句子规范参考 ASD-STE100：短句、一句一义、主动语态、主语明确、少嵌套（不套用受控词典和长句）；内容准确、通俗、有逻辑、有趣味，并说明已知结论、争议和知识边界。

**修正 2（检索层）——orig 检索规则重构为「概念跳跃法」**：寻找主概念领域 → 找到该领域最具启发力的信息源 → 基于该信息源寻找其他更具启发性的信息源；检索关键词核心原则 = **差异性**（与核心概念存在差异但有相似性）；不区分收敛/发散等结构化内容，只搜索 + 在概念间跳跃；跳跃关键 = 同一概念的差异性（同中求异）+ 差异概念的同一性（异中求同）；大量跳跃收集概念与解释，以唯一标准筛选：**哪些概念最能启发 human 推进思考通过当前门禁目标**。

**修正 3（门禁层）——五阶段验证规则重写（human 原文）**：
1. 问题还原：期望、现状、差距三个结构化表述说清楚，能清晰明确价值判断和事实判断
2. 视角选择：明确描述选择这一视角带来的应用价值，能描述其内在的因果运动
3. 矛盾分析：基于这一视角对问题进行解释，定义决定性/根本性/全局性的矛盾来描述其因果运动
4. 辩证逆转：符合逻辑地回答：当阻碍 X 是目标 Y 达成的必要条件，那么应当如何？
5. 良质批判：回顾思考过程所依赖的假设是否可证伪，以及推导过程的逻辑性

**呈现链（v8.1）**：orig 检索（概念跳跃）→ 教学润色（科普呈现）→ human 思考回答 → 门禁判定（五条验证规则）

---

## v8.3 迭代（2026-11-27，loop_21 实测后 human 升级裁决）

**orig agent 本质重述（理论化）**：orig = 概念空间中的**随机游走**——通过跳跃性检索将多个领域内「同中有异、异中有同」的概念联系起来，以此启发 human 思考、帮助 human 突破思考门禁。**唯一目的 = 帮助 human 突破当前思考门禁**。

**呈现原则（关键原则：递归解释）**：呈现以**概念**为主题，科普风格讲清楚一个概念；讲解**不存在领域专用概念词汇**——全部**递归地以通识内容解释清楚**（对概念的递归解释）。

**收尾引导（新增环节）**：呈现完所有启发概念后，留下一个**引导**——帮助 human 将这些概念串联起来、**绘制概念地图**，引导 human **用自己的话来表达思想**。main agent 按思考门禁 skill 判断 human 表达的思想能否突破门禁、进入下一思考阶段。

**理论总结（human 原话）**：概念空间随机游走 + 递归解释原则 = **发散思考**（启发 human 建立脑海中的概念地图）；最后的引导联系 = 启发 human **突破思考门禁**（收敛）。
