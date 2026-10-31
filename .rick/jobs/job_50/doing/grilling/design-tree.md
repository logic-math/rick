# human-loop v7 设计树（grilling）

> 源头：ADR-human-loop-v7（`.rick/draft/loops/loop_12/ADR-human-loop-v7.md`，D1–D10 已裁决）+ job_50 requirement（优化 human-loop 实现启发 human 原创性思考）。
> 本文件是活文档，逐层追加（层号 + 模块 + pipeline + 判断节点）。

## 第 0 层（根层）——O + KR 集

### O（全局目标）

将 human-loop 从 v6（三角色教师会话：parent 编排 + research 事实还原 + teach 逐字呈现）升级为 v7：以「启发 human 原创性思考」为目标函数（D2：差异源注入 + 想法差异性），按 ADR-human-loop-v7 已裁决决策（D1–D10）落地为 rick 可运行的 loop+skill 架构（D7：自研组件、rick 格式内化）；消除 v6「啰嗦/低效」结构性症状（输出长/慢贵/原创无实证支撑）；ADR §7 七个未决问题与实现级判断节点全部由 human 裁决闭环。

### KR 集（演绎：KR1 ∧ KR2 ∧ KR3 ∧ KR4 ∧ KR5 ⟹ O）

| KR | 内容 | 验证方式 |
|---|---|---|
| KR1 架构落地 | 五步 human-learning-loop（①目标对齐 ②ZPD 初始化 ③大纲 ④逐课推进 ⑤总结复习）+ 全部 skills 按 rick 格式落地，human 可启动一个完整学习会话 | loop/skill 文件存在 + 启动命令可跑 |
| KR2 症状消除 | v6 啰嗦/低效三来源（teach 正文全文返回+逐字呈现、lesson 三硬要求、fresh 上下文重读 6 份 canon）被硬约束处置；lesson 篇幅上限**先于内容质量**成为硬约束（ADR 附：元证据） | 新协议文本中三条来源各有对应处置条款 + 篇幅硬约束数值 |
| KR3 机制闭环 | 三大机制有可执行操作定义：D4 复习机制（锚载体+冷启动）、D6 掌握度判定（强/中/弱边界+误差吸收）、D8 原创启发（真实异质性来源+更优替代+判定） | 每个机制在 skill 中可操作（输入/输出/判据），无「待定」字样 |
| KR4 判断不吞 | ADR §7 七问 + 实现级判断节点（架构整合、篇幅数值、teach agent 去留、验收方式）全部经 human 裁决 | grilling 提问记录 + human 答复落盘 |
| KR5 实跑验收 | human 用 v7 实跑一个真实学习会话并给出主动判断（D9：连续三次不行即删） | 一次完整 v7 会话 + human 判断原话记录 |

### OKR 充分性自检（根层）

- KR1 给出「可运行的架构」，KR3 保证「机制可操作」，KR2 保证「v6 症状不复现」，KR4 保证「判断权归 human」（不吞节点 = 不引入未裁决假设），KR5 保证「最终目标被实测检验」。
- 推导链：架构可运行 + 机制可操作 + 症状消除 + 无未裁决假设 + 实跑有效 ⟹ human-loop v7 达成「启发原创性思考」且不复现啰嗦/低效。充分。
- 风险：KR5 是 D9 人的主动判断（非量化 eval，已知偏向「读着顺」，ADR §6 负面后果已记录）——已由 human 裁决接受（D9），不构成分解缺口。

---

## 第 1 层（模块层）——六模块 pipeline

### 模块清单

| 模块 | 职责 | 承载（ADR §4 映射） |
|---|---|---|
| M1 Loop 骨架 | 五步编排逻辑 + 文件状态机 + 回合节奏（认知负荷管理 D1） | `.rick/loops/human-learning-loop.md`（或 Go 模板，见 J-arch） |
| M2 Skill 集 | 六个 skill：mission-builder / zpd-assessor / syllabus-builder / lesson-writer / originality-sparker / recall-review | `.rick/skills/<name>_skill/skill.md`（或 Go 模板 skills，见 J-arch） |
| M3 research agent | 事实还原（递归调研至常识性置信源 + 价值性信息上报裁决） | 复用 v6 `agent:'research'`（`~/.rick/pi/agent/agents/research.md`） |
| M4 存储与锚 | loop 目录结构、可重访锚（D4/D5 并存的物理条件）、跨 loop 历史索引 | loop_N 目录内文件（锚载体待裁决：ADR §7 Q1） |
| M5 启动与接线 | human 发起方式（rick 命令 / loop trigger）、prompt 生成、session 管理 | `internal/handler/human_loop.go` 等 Go wiring（改造范围待裁决） |
| M6 验收与迭代 | D9 判断记录、三次规则淘汰、跨会话效果追踪 | loop 目录内 judgment 文件 + 判断记录 |

### pipeline（模块间调用关系）

```
human 发起学习主题
   │
   ▼
[M5 启动] ──render──▶ [M1 Loop 骨架]（五步编排 prompt）
   │
   ▼
① 学习目标对齐：[M2.mission-builder] → mission.md（human 确认）
   │
   ▼
② ZPD 初始化：[M4 历史锚/领域相关性] → [M2.zpd-assessor 复习探针]
      → human 回忆写出 → 强/中/弱 + 无能力标记 → ZPD 基线
   │
   ▼
③ 构建大纲：[M3 research agent → facts] + mission + 已裁决 values
      → [M2.syllabus-builder] → syllabus.md（human 确认）
   │
   ▼
④ 推进教学（逐课子循环）：
      research(facts) → [M2.lesson-writer 生成 lesson（事实层给满/推导层留白/篇幅硬约束）]
      → human 讲出来（阐述）→ [M2.zpd-assessor 判掌握档]
      → [M2.originality-sparker 原创性启发（真实异质性+更优替代）]
      → 推进下一课 / 倒退上一课（D1 节奏开关）
   │
   ▼
⑤ 总结与复习：[M2.recall-review] → 抽关键词 + 留可重访锚（写回 M4）→ 收口到学习目标
   │
   ▼
[M6 验收] human 主动判断（D9）；跨 loop 重访时 ② 从 M4 锚冷启动
```

### 第 1 层判断节点（提炼中，待 L1 research 后合并成批量追问）

- **J-arch（架构整合）**：ADR D7 说「按 rick 的 loop+skill 格式内化」到 `.rick/loops/` + `.rick/skills/`，但 v6 现实现 = Go 模板（`internal/prompt/templates/sense_loop.md`）→ `rick human-loop` 命令渲染 + pi agents（teach/research）。v7 落在哪种形态？（A: 纯 .rick 数据文件，无 Go 改动；B: 改写 Go 模板 sense_loop.md 保留命令；C: 混合）
- **J-teach-agent（teach 去留）**：§4.3 说「skill＝注入当前上下文、agent＝独立上下文只回摘要」——lesson-writer 由谁执行？parent 会话内（in-context skill）还是子 agent（fresh context）？成本结构完全不同。
- **J-anchor（ADR §7 Q1）**：可重访锚的载体——lesson 卡点条目 / judgment 原话 / research 缺口清单？
- **J-three-times（§7 Q2）**：「三次什么」——同类（信度）还是异类（效度）？
- **J-error-absorb（§7 Q3）**：误差吸收规则——先讲 vs 先给答案；单探针 vs 三探针交叉；多轮取样？
- **J-v9-boundary（§7 Q4）**：「想不起来＝无能力」只用于跨 loop 探针，还是也用于即时达标判决？
- **J-speak-out（§7 Q5）**：「讲出来」的验收位置——独立验收动作（非 skill）还是只判理解？
- **J-mastery-boundary（§7 Q6）**：强/中/弱的操作边界（关键概念覆盖率？错误断言数？）
- **J-coldstart（§7 Q7）**：首次 ZPD 初始化冷启动——领域相关性启发具体怎么问？
- **J-length（篇幅数值）**：lesson 篇幅硬约束的具体数值（ADR 附录：必须先于内容质量；v6 实测 15–20KB 被判啰嗦）
- **J-accept（验收方式）**：KR5 实跑验收用什么主题、何时算过？

---

## L1 调研结论（research-L1.md 终版=主线在线自验：curl 直连 arXiv 读 2412.16429v3/2501.15749v1 全文 + 11 篇 ID 核验；B/C 级残留项见 R7，doing 开工前网验）

- **判级**：LearnLM 不判学生（29 题 rubric 评模型教学质量，[A]）；Khanmigo 判级=追问链 [C]；量化形态在 ITS 传统（AutoTutor expectation+LSA 阈值 0.4–0.5、Khan 四档累积可升降 [B]）——v7 判级用自然语言 rubric 佐证充分
- **吸收**：部分线索能救回=提取失败，换措辞+多线索两轮仍败=真缺口（Tulving & Pearlstone）；teach-back 三判据+「二不过=缺口」；AutoTutor 提示梯度 pump→hint→prompt→assertion
- **节奏**：生产系统无公开阈值；可操作规则=困惑 2 轮未解→换措辞/降难度；挫折信号→立即回退；advance=答对+能 teach-back
- **锚**：ALEKS=离散知识状态+fringe+定期重评估建模遗忘；LLM 侧=结构化画像块（自编辑）+可检索事件库（MemGPT/Mem0）
- **异质性**：同基座辩论有谄媚/镜像收敛风险；跨基座（ReConcile）与人类分歧数据（Delphi）误差更去相关；教育场景零实证
- **ASCII 图**：CJK 全角 2 列+UAX #11 East-Asian Ambiguous 宽度→LLM 自绘 ASCII 几乎必错位；code-fence 内简单结构可靠；Mermaid 更稳健但属 D8 否决的图文生成

## L3 批量提问（第 1 层 + 根层——2026-10-31 呈现，14 问，待 human 裁决）

Q0 根层 OKR（O+KR1–5）｜Q1 J-arch 架构整合｜Q2 J-teach-agent teach 去留｜Q3 J-anchor 锚载体｜Q4 J-three-times 三次什么｜Q5 J-error-absorb 误差吸收｜Q6 J-v9-boundary 无能力标记边界｜Q7 J-speak-out 讲出来位置｜Q8 J-mastery-boundary 三档边界｜Q9 J-coldstart 冷启动｜Q10 J-length 篇幅数值｜Q11 J-ascii 文本绘图约束｜Q12 J-accept 验收方式｜Q13 J-v6-fate v6 处置
（提问原文与推荐答案见会话 L3 轮；human 答复后回填本树并进入下一层）

---

## L3 裁决回填（human 2026-10-31 原话）

- Q0 根层 OKR：**确认**
- Q1 J-arch：**B——延续 Go 内嵌提示词方法，internal/prompt/templates 保持架构不变**（.rick 数据真身方案否决；v7 = 模板重写 + 新 skill 模板，命令/渲染管线不动）
- Q2 J-teach-agent：**每个 skill 都升级为一种子代理，保持上下文的独立**（推翻 ADR §4.3 in-context 读法；六 skill → 六 pi agent，research 保留，共 7 agent 由 parent 编排）
- Q3 J-anchor：**可以**——每课收尾写 anchors.md（关键词+卡点原话+未解矛盾），跨 loop 探针 grep 全部 loop
- Q4 J-three-times：**推荐**——三次同类（信度）
- Q5 J-error-absorb：**是的**——二次探针链（部分线索→换措辞；两轮不过=弱/缺口；仍败才直给）
- Q6 J-v9-boundary：**是的**——「想不起来=无能力」只用于跨 loop 探针
- Q7 J-speak-out：**可以**——讲出来=验收动作（zpd-assessor 内 teach-back 式），非独立 skill
- Q8 J-mastery-boundary：**是的**——自然语言 rubric+最低操作判据（强/中/弱按错误断言+关键概念覆盖）
- Q9 J-coldstart：**是的**——mission 起点自报 + 领域核心概念 2–3 探针校准
- Q10 J-length：**篇幅基于 ZPD 启发**——不设固定 KB 上限，详略由 zpd 档位驱动（弱=少而精，强=可跳过；技术输出上限仍受单响应限制约束）
- Q11 J-ascii：**是的，且方法上先判断画图还是文字描述，选择正确的描述**——呈现前先判「需要图还是文」，再选表示形式
- Q12 J-accept：**是的**——human 自选主题单会话走五步，主动判断
- Q13 J-v6-fate：**是的（替换）+ SENSE 升级**——第一步 事实还原→**问题还原**（还原事实信息与价值判断，分清期望与现状）；后四步（构建视角→洞察矛盾→辩证逆转→良质批判）不动

## O 改述（human 2026-10-31）

**核心目标 = 启发人类进行原创思考。**

核心流程（human 原话结构化）：

```
1. 调研与呈现
   (a) 调研：research agent 尽力调查事实信息，呈现调研结果
   (b) 呈现：按 ZPD 理论选择与当前用户认知程度最符合的内容，
       返回详略得当的信息 → 触发学习、启发思考
2. 费曼学习与记录（过程核心=原创性思考）
   - 内容讲出来后，启发 human 进行费曼学习：human 反过来当老师讲给 agent 听
   - agent 判断对错（通过讲解概念理解讲得对不对）
   - 将学习过程记录下来 = 学习知识记录（知道他差哪/不差哪 → 下次讲课详略得当）
   - 针对 human 费曼讲出的内容，原创性启发 Agent（ADR 的 originality sparker）
     返回与其观点部分违背、非完全否定的差异化知识
     → 不断启发跨领域知识融合 → 原创性启发
```

## 第 1 层 L5 终止判定

- 模块齐备 ✓（M1 loop 模板重写 / M2 六 agent + research / M4 存储（mission/syllabus/lessons/zpd 记录/anchors）/ M5 Go 接线不变 / M6 验收）
- pipeline 已按裁决修正：六 skill 全部 agent 化（上下文独立），parent 编排
- OKR 充分性：架构（Q1 B）+ 机制（Q3–Q9 裁决闭环）+ 症状消除（Q10 ZPD 详略 + agent 最小输入集）+ 判断不吞（本层 14 问全裁决）+ 实跑（Q12）⟹ O（启发原创思考）——成立
- **下钻第 2 层：模块内部结构（文件形态/接口契约/agent 输入输出）**

---

## 第 2 层（模块内部层）——六 agent 契约 + 存储 + 接线

### L1 自查事实（代码机制，已核实）

- agent 部署：`internal/env/agents.go` 的 `rickAgents` 硬编码清单 → `deployRickAgents()` 幂等写 `~/.rick/pi/agent/agents/{name}.md`（rick-managed 标记覆盖语义）；system prompt 正文 = `prompt.LoadCoreSkills([name])` = `internal/prompt/templates/skills/{name}.md` 嵌入内容。**加新 agent = 扩 rickAgents 清单 + 新建 skills/{name}.md 模板；删 teach = 从清单移除**
- loop prompt：`internal/prompt/templates/sense_loop.md` → `GenerateHumanLoopPromptFile` 渲染到 `loop_N/prompts/`（含 `WriteSkillFile` 复制 skill 文件）；目录分配/session_id/resume 由 `internal/handler/human_loop.go` 承担（不动）
- 调度：pi subagent 按已部署 agent 名派发（runs.run + task 文本动态组装），v6 的 per-agent task prompt 模板文件（teach.md/research.md）非必需

### 六 agent 契约（输入→输出→落盘）

| agent | 输入 | 输出 | 落盘（独写） |
|---|---|---|---|
| mission-builder（归属待裁 J2-1） | 面试原话（或无） | mission（为什么学/考核标准/起点） | mission.md |
| zpd-assessor | 历史 anchors/zpd + mission + human 回忆/费曼原话 + 本课关键概念清单 | 强/中/弱档位 + 对错条目 + 缺口清单 + 详略建议 + 推进/回退建议 | zpd.md（逐次追加） |
| syllabus-builder | research facts + mission + 已裁决 values | 模块序 + 每块判据 + 位置理由（SENSE 问题还原留痕） | syllabus.md |
| lesson-writer | syllabus 当前模块 + research_N + zpd 档位（详略依据） + SENSE canon | 课程正文（ZPD 详略；先判图/文再选表示；事实层给满/推导层留白） | lessons/teach_N.md |
| originality-sparker | human 费曼原话 + 当前课概念 + research_N（基础事实）+ 自检索真实分歧/异域类比 | 差异化启发内容（部分违背非否定 + 更优替代 + 差异维度） | lessons/spark_N.md |
| recall-review | 本 loop 全部产物（zpd.md/judgment/lessons） | 复习关键词 + anchors 蒸馏（关键词+卡点+未解矛盾） | anchors.md |
| research（复用） | 调研目标 | research_N.md + 待裁决 values 追加 | research/research_N.md |

### 存储布局（loop_N/）

mission.md（parent 或 mission-builder）｜values.md（research 追加+human 裁决回填）｜syllabus.md（syllabus-builder）｜zpd.md（zpd-assessor，学习知识记录-细档）｜anchors.md（recall-review，跨 loop 探针源）｜research/research_N.md｜lessons/teach_N.md + spark_N.md｜judgment.md（parent，human 判断原话）｜session_id

### parent（sense_loop v7 主 prompt）职责

状态机（mission→ZPD 探针→research+syllabus→教学循环→收口）+ 派发（runs.run 动态 task）+ 交付门禁（回执/正文校验+resume 降级链）+ 呈现（lesson/spark 正文原样呈现）+ 记 judgment + 价值裁决回合 + 节奏规则执行（强→推进；中→同层复测；弱→二次探针→回退；困惑 2 轮→换措辞；挫折→立即回退）

### 第 2 层判断节点（待 human 裁决）

- **J2-1 mission-builder 归属**：面试必须 parent 与 human 交互（agent 无法直接对话）；mission=面试原话结构化。选项：A parent 面试+直接落盘（无 mission-builder agent，纪律写进 loop prompt）；B parent 面试→mission-builder agent 转写格式化（多一次派发往返）。推荐 A（面试上下文已在 parent 手里，转写无重计算价值）——但与「每个 skill 都子代理化」字面冲突，需 human 确认。
- **J2-2 学习知识记录形态**：human 已定功能（知道差哪/不差哪→下次详略得当）。推荐形态=zpd.md（zpd-assessor 每次费曼判断追加：档位+对错条目+缺口+详略建议）+ anchors.md（⑤ 时蒸馏为跨 loop 探针源）。确认即可。
- **J2-3 sparker 调研方式**：选项：A 串行 research→sparker（深但慢，v6 成本结构）；B sparker 自带 web 检索做小型定向调研（独立上下文，灵活）；C 只消费已有 research_N（快但异质性来源弱）。推荐 B（fullTools 本就含 web_search/fetch_content；真实分歧需定向检索，深度尽调 fallback 请 parent 派 research）。

---

## L2 裁决回填（human 2026-10-31「全按推荐」）

- Q14 J2-1：**A——parent 面试并直接落盘 mission.md**（全 skill 子代理化的唯一例外；面试纪律写入 loop prompt）
- Q15 J2-2：**zpd.md（zpd-assessor 逐次追加：档位+对错条目+缺口+详略建议）+ anchors.md（⑤ 蒸馏：关键词+卡点原话+未解矛盾）**
- Q16 J2-3：**B——sparker 自带 web 检索**（task 带费曼原话+当前课概念+research_N 路径；定向检索真实分歧+异域类比；深度尽调 fallback 请 parent 派 research）

第 2 层 L5 终止判定：六 agent 契约齐备（mission 除外）、存储布局确定、parent 状态机与节奏规则确定、sparker/assessor/reviewer 输入输出闭环 ⟹ 支撑第 1 层模块层。**下钻第 3 层（叶子层）**。

---

## 第 3 层（叶子层）——四维度落实

### M1 Loop 骨架

- **关键代码**：`internal/prompt/templates/sense_loop.md` **原地重写**为 v7 协议（模板名/渲染管线/命令全不动）。结构：角色（parent=教学会话编排者）→ agent 清单（research + zpd-assessor/syllabus-builder/lesson-writer/originality-sparker/recall-review）→ 文件状态机（mission 缺失→parent 面试；zpd.md 缺失→② ZPD 初始化探针；syllabus 缺失→research+syllabus-builder；教学循环；收口 recall-review）→ 派发语法（runs.run + 动态 task，timeoutMs 3600000）→ 交付门禁（回执/正文校验+resume 降级链，承 v6）→ 教学节奏（核心循环=调研与呈现→费曼学习与记录→原创性启发；节奏规则=强推进/中同层复测/弱二次探针回退/困惑 2 轮换措辞/挫折立即回退）→ 价值裁决回合（承 v6）→ 退出准则 → 开始
- **工具调用**：`rick human-loop <topic>`（启动）；`rick human-loop --resume loop_N`；`rick human-loop --dry-run <topic>`（冒烟）
- **环境依赖**：无新增（D7 自研）；pi agents 机制复用

### M2 Agent 集（7 个）

- **文件结构**：
  - 修改：`internal/prompt/templates/skills/sense.md`（五步第一步 事实还原→**问题还原**：还原事实信息与价值判断，分清期望与现状；后四步不动）
  - 新建：`internal/prompt/templates/skills/{zpd_assessor,syllabus_builder,lesson_writer,originality_sparker,recall_review}.md`（各自=pi agent system prompt：触发/职责/输入输出/操作规则（含 Q3–Q11 裁决细则）/交付契约（落盘+回执或正文））
  - 删除：`internal/prompt/templates/teach.md`、`internal/prompt/templates/skills/teach.md`、`internal/prompt/templates/research.md`（task prompt 模板——v7 parent 动态组装 task，agent 契约在 skills/*.md；research 的 skills/research.md 保留）
- **关键代码**：`internal/env/agents.go`——rickAgents 删 teach，加 5 条 spec（name=zpd-assessor/syllabus-builder/lesson-writer/originality-sparker/recall-review；description 按职责；tools=fullTools；defaultContext=fresh；thinking=high；timeoutMs=3600000）。`internal/env/check.go`——requiredRickAgents 更新为 7 个。`internal/prompt/manager.go`——删 teachTemplate/researchTemplate embed 与 case
- **工具调用**：`rick tools update-pi`（customizations→deployRickAgents 幂等部署 agents 到 ~/.rick/pi/agent/agents/；需先构建新二进制）
- **环境依赖**：无第三方；agent 命名沿 pi 约定（小写连字符）

### M4 存储与锚

- **文件结构**：loop_N/ = mission.md（parent 独写）｜values.md（research 追加+parent 回填裁决）｜syllabus.md（syllabus-builder）｜zpd.md（zpd-assessor 独写，学习知识记录细档）｜anchors.md（recall-review 独写，跨 loop 探针源）｜research/research_N.md｜lessons/teach_N.md（lesson-writer）+ lessons/spark_N.md（originality-sparker）｜judgment.md（parent 独写 human 原话）｜session_id｜prompts/（渲染副本：sense_loop + skill_sense + 5 新 + skill_research）
- **工具调用**：parent 预建目录（mkdir -p research lessons）；跨 loop 探针 = grep 历史 loop anchors.md/zpd.md

### M5 Go 接线

- **关键代码**：`internal/prompt/human_loop_prompt.go`——GenerateHumanLoopPromptFile 重写：渲染 sense_loop 主 prompt（新占位符：agent 清单节、zpd/anchors/spark 路径）+ WriteSkillFile 复制 7 个 skill 文件；删除 teach/research task prompt 渲染段。`internal/builder/pibuilder.go`——SaveHumanLoopPrompt 签名收缩（teachFile/researchFile 返回值移除）。`internal/handler/human_loop.go`——不变（目录分配/session_id/resume/dry-run 原样）
- **测试**：human_loop_prompt_test.go（断言新文件集）、env_test.go/update_test.go（agent 清单）、pibuilder_test.go、web/humans_test.go 更新
- **验证**：`go build ./...` && `go test ./internal/...`；`rick human-loop --dry-run "冒烟"` 渲染 v7 prompt；部署后 `ls ~/.rick/pi/agent/agents/` 见 7 个

### M6 验收

- **工具调用**：human 自选主题 `rick human-loop <topic>` 实跑一次完整五步会话
- **判据**：human 主动判断「值得继续用」（D9）；judgment.md 留痕；三次同类失败规则记入 loop prompt

### 叶子层判断节点：**全部消解，无判断节点**

依据：本层全部决策（文件路径/函数签名/工具命令/配置）均由已裁决项（Q1 B、Q2 全 agent 化、Q3–Q16）+ 代码事实（agents.go 部署机制、manager.go embed、human_loop_prompt.go 渲染管线）机械推导；无权衡/阈值/取舍残留。SENSE 五步仅第一步改动已由 human 原话定死（问题还原=还原事实信息与价值判断，分清期望与现状）。

## 全局终止声明

**Grilling 完成——设计树已遍历完毕（第 0 层 OKR → 第 1 层模块 → 第 2 层契约 → 第 3 层叶子四维度），所有模块已落实到代码实现/文件结构/工具调用/环境配置。**
