# 设计树：human-loop 教师会话改造（v4 → v5，dev job_45）

> 需求来源：human 2026-10-10 真实验证 dev 实例（8414）教师会话后的两个问题 + 口述改造方向。
> 承载：rick-rsi-loop（修改 rick 自身：prompt 模板 + skills + code），开发在 dev 工作区 `/workdir/sunquan20/rick-dev`（分支 dev/self-evolve）。
> 基线：job_44 的 human-loop v4（`sense_loop.md` parent + `teach`/`research` 子代理 + humans 只读页面）。

## 第 1 层（根层）：OKR —— O（目标）+ KR（关键结果）集

**O（目标，可验证表述）**：
human-loop 教师会话改造为**总分结构 + 会话内直读**：先由 research 调研「本领域最权威学习资料 + 该领域的学习方法」，再由 teach 做一次 SENSE 还原思考产出**教学大纲**（先学什么再学什么，本质是「学习该领域的学习方法」的呈现），human 确认大纲后逐层下钻；**每节教学正文直接作为会话内容呈现**（human 无需打开文件）；每节叙事力图还原原创思考者的思考过程，并绑定「核心练习方法」主轴。

**KR（关键结果）集**：
| # | KR | 现状 → 目标 |
|---|---|---|
| KR1 | 会话内直读：teach 正文作为 assistant 消息出现在会话（不再只有文件/回执） | v4：teach 返回 ≤300 字回执，正文仅落盘 → v5：正文全文入会话 |
| KR2 | 「总」阶段：research（领域全景 + 学习方法）→ teach（SENSE 还原思考 → 教学大纲）→ human 确认 → 落盘 `syllabus.md` | v4：无总层，直接开课 |
| KR3 | 「分」阶段：每节=大纲一个模块的下钻；叙事=还原原创思考者的思考过程（自由叙事 + 3 硬要求） | v4：固定 6 段模板 |
| KR4 | 核心练习方法主轴：首次调研即含领域学习方法；每节绑定核心练习方法 | v4：无此概念 |
| KR5 | 不新增 agent 类型：research/teach 以「特殊任务」承载总层工作 | v4：机制复用 |
| KR6 | 模板 + code + 测试 + 构建收敛：dev 实例可验证（新会话即生效） | — |

**OKR 充分性自检**：KR1∧…∧KR6 ⟹ O？会话直读✓ 总层✓ 分层下钻✓ 练习方法主轴✓ 机制复用✓ 可验证✓。

## 第 2 层：改造点（文件级）+ human 裁决

### 根因（代码级实证）
- 问题1（看不到正文）：`skills/teach.md` 交付契约明文「最终回复=回执 ≤300 字，**正文在文件里**，parent 读文件展示给 human」；`web/src/components/chat/ToolCallCard.tsx` 工具卡片**默认折叠**（`open=false`）、结果 `<pre>` + `MAX_OUTPUT` 截断 → 正文不可读。
- 问题2（结构化死板）：`sense_loop.md` 文件状态机只有 mission→research→teach 回合，**无「总」层**；`skills/teach.md` 课程节结构为固定 6 段。

### human 裁决（2026-10-10，全部确认）
| # | 节点 | 裁决 |
|---|---|---|
| T1 | 会话内展示 | teach 正文作为回复返回；parent **原文呈现为会话消息**；文件仍落盘留痕 |
| T2 | 「总」层产出者 | **不新增 agent**：research（任务=调研本领域最权威学习资料+学习方法）→ teach（SENSE 思考→教学大纲）；先全貌再下钻本身即教学方法；先有教学计划再实施教学 |
| T3 | 大纲确认 | **先确认再开课**（大纲本身就是教学的一部分） |
| T4 | 课程节结构 | **自由叙事**（还原原创思考者的思考过程）+ 保留 3 硬要求（引注事实 / 启示性追问 2-4 / primary source） |
| T5 | 核心练习方法 | 首次调研即调研领域学习方法；**大纲本质=学习该领域的学习方法**；每节绑定 |

### 改造点清单（文件级）
1. `internal/prompt/templates/sense_loop.md`（parent）：
   - 新增材料：`{{syllabus_path}}`（`{{loop_dir}}/syllabus.md`）
   - 文件状态机新增「教学大纲缺失」分支：派 research（领域全景+学习方法）→ 派 teach（syllabus 任务）→ **原文呈现大纲 + 请 human 确认/修订** → 落盘 → 开课
   - 呈现层改造：**把 teach 的正文原文作为你的回复**（不改写/不摘要）
   - 教学节奏改造：大纲主轴 + 核心练习方法绑定 + 还原原创思考过程
   - 派发模板：research/teach 各两种任务类型（domain-survey / module-facts；syllabus / lesson）
2. `internal/prompt/templates/teach.md` + `internal/prompt/templates/skills/teach.md`（teach）：
   - 交付契约：最终回复 = **正文全文**（文件仅留痕）；去掉「只回执」
   - syllabus 任务模式：产出大纲（SENSE 还原思考：还原事实→视角→主要矛盾→辩证逆转→核心练习方法→展开序）
   - 课程节：自由叙事（还原原创思考者思考过程）+ 3 硬要求 + 绑定核心练习方法
3. `internal/prompt/templates/research.md` + `internal/prompt/templates/skills/research.md`（research）：
   - domain-survey 任务模式：调研本领域最权威学习资料 + 该领域学习方法（广度高）
4. `internal/prompt/human_loop_prompt.go`：新增 `syllabus_path` 变量注入
5. `internal/prompt/human_loop_prompt_test.go`：断言新变量

### 测试方法（自测指引）
- `go build ./... && go test ./internal/prompt/... -timeout 60s`
- 渲染冒烟：`rick human-loop --dry-run <topic>` → 含 `syllabus.md` 路径与「总」阶段说明
- dev 实例：`rick tools dev-web restart` → build_id 变化 → 新教师会话验证（human 真实验证）
