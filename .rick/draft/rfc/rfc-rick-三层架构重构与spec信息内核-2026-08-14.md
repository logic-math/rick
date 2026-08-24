# RFC — rick 三层金字塔架构重构 + spec 信息内核 + 功能下沉 pi

- 日期：2026-08-14
- 主题：subagent 在 pi runtime 下触发概率低，优化提示词（使触发确定性提升到上限内最高）
- 产出：sense_loop loop_6（S/E/N/S-R/EC 全流程完成，跃迁 = 维持）
- 方法：本 RFC 严格基于 `loops/loop_6/judgment.md` 中 human 判断原话 + `loops/loop_6/briefs/` 事实陈述；R7 上报项标注「待 human 决策」。

---

## 1. 主题

把 rick 重构为「三层金字塔架构（cli / handler / env+builder+runtime）」，将 doing 的 dag 调度与门禁检查下沉到 pi，并把「spec（结构化自然语言工程实现契约）」确立为 rick 的信息内核——使 subagent 触发确定性提升到上限内最高，且方法/实现隔离（丢弃源码即可 AI coding 出功能等价的 rick）。

---

## 2. 背景与哲学基础

### 2.1 现状 / 期望 / 差距（S 阶段 human 判断）

- **现状**：human-loop 命令内部的三个 subagent（think / research / exporter）触发概率低；plan / doing / easy / learning / dream / ctrl 等命令在提示词中已明确要求用 subagent 的情况下，触发概率也较低（human 原话）。
- **期望**：通过改动 rick 现有框架、优化提示词和某些配置，使 subagent 触发确定性**提升到上限内最高**（原为「最大化」，后修订），让模型更能遵守提示词约束（human 原话）。
- **差距**：pi 触发 subagent 的最佳实践 与 rick 现状 之间的差距（human 原话）。

**事实支撑（现状的量化面）**：rick 模板中自然语言 subagent 触发词 **243 处**（root 模板 134 处 + skills/ 109 处），而 pi 触发语法（`workflowScript` / `runs.run` / `runs.all`）**0 处**、pi 内置 agent 名 **0 处**（来源：`briefs/research-report-N1.md` F1；`research-report-S.md` N3.1/N3.2）。

### 2.2 因果（S 阶段 human 确认）

提示词未对齐 pi 的触发机制（无触发语法、无内置 agent 名）是触发概率低的**主要原因**（human 确认；缺口 D1/D2/D6/D7 已证实，来源：`research-report-S-bestpractice.md` 差距对照表）。

### 2.3 视角与类比（E 阶段 human 判断）

- 视角 = **协议对齐/兼容**（对应候选 V5）。
- 类比 = 两个各自自洽、但合作时暴露问题的**语言体系**（pi 语言体系 vs rick 语言体系）（human 原话）。
- 动作 = 在 pi 框架长期发展的既定前提下，**高度定制化改造 rick 提示词**，基于 pi 的 subagent 最佳触发实战对齐。

### 2.4 核心假设（human 显式自标）

- **方法/实现隔离**：rick = 方法（自然语言描述），pi = 实现（编程语言描述）；方法描述经模型可转化为预期行为完全一致的开发计划。
- **核心假设**：只要自然语言无歧义地描述正确的验收标准，即可实现「方法与实现隔离、任意切换实现」。
- **「等价」的定义**：功能等价 = 通过所有功能验收即一致（human 原话：「近似测试通过了所有的功能验收，就算是一致的…只要功能等价，就认为是效果等价的」）。
- **自然语言的无歧义性**：自然语言有表达力、可无歧义表达；信息压缩可接受，只要关键信息描述正确即可刻画功能（human 原话）。

### 2.5 模型角色与进化迭代（辩证逆转，human 判断）

- rick 不在乎模型的能力上限；模型 = 信息压缩工具 / 信息模糊提取的存储器；所有解决老问题的信息被压缩进模型，rick 专注解决新问题；基于**进化算法的持续迭代**提升智能表现（human 原话）。

---

## 3. 主要矛盾与辩证逆转

### 3.1 矛盾判断的演化

- **N1**：human 起初认为本质矛盾 = K4「深度定制 vs 框架独立性」，立场「完全偏向深度改造，不考虑独立性」；并给出「3 年后失去先进性」「rick 是核心节点（消失则退化为平庸 pi agent）」两个判断。
- **N2**：human 选定主要矛盾 = K4，控制手段 = 理解 pi 触发语言（K1）→ 改造 prompt。
- **S-R 辩证逆转**：human **否定 K4 是主要矛盾**——因 coding 让「实现」不再值钱、「方法」才值钱；无需在设计上保留独立，只需深刻改进 rick 持续提升效果（human 原话）。

### 3.2 逆转后的核心

核心 = **工程化的方法描述**（方法与实现隔离）；实现「深度定制 与 独立迭代」的辩证逆转。rick 做薄 = 引导程序（对 sense 方法的落地），dag 调度与门禁不再由 rick 维护，而是利用 pi 能力直接实现（human 原话）。

---

## 4. 目标架构设计（三层金字塔）

> 以下为 human 最终架构设计原话要点（来源：judgment.md「RFC 架构设计（最终）」）。

### 4.1 分层

- **路由层 cli**：路由命令、解析参数等 cli 功能层。
- **处理层 handler**：接受 cli 路由与解析后的命令参数，调用 env / builder / runtime 三个模块完成 rick cmd 功能实现，是调度聚合层。
- **执行层 env + builder + runtime**：
  - **env**：对 rick 在当前机器的环境配置、关键依赖检查/下载/安装/配置；pi 及 pi 开源扩展与插件的维护与更新。
  - **builder**：按不同 cli 功能拼接、在 cmd 触发时创建一组符合 runtime（pi/dsh 等）要求规范的一组产物。
  - **runtime**：对 pi 或 dsh 等调用逻辑的封装（参数解析 + 调用，目前支持 pi runtime）。

### 4.2 builder 三件

- **templates**：提示词模板，通过 go `embed` 内嵌二进制，按 cmd 功能拼接成某任务真正需要执行的提示词。
- **pibuilder**：为 pi 这一 runtime build 具体提示词文件目录的统一入口，内部组合 plan / doing / easy / human-loop 等多个子 builder。
- **xxxxbuilder**：可扩展位，后续新增 runtime 只需扩展这一 builder，其他组件不改动；与新 runtime 更好适配的信息封装在此层。

### 4.3 runtime 设计

- 封装具体 agent runtime 的调用逻辑 = 参数解析 + 调用；目前支持 pi runtime。

### 4.4 设计原则（作为信息内核）

- 暂不考虑其他 builder/runtime；目标是让 rick 在执行层拥有更强控制力，深入 agent runtime 内部、更直接控制 LLM 输入信息，实现**上下文熵减**。
- 保持**单一 runtime（pi）**：投入全部精力将方法付诸实践，强化一种实现。
- **切换 runtime 的前提** = 新 runtime 带来更强大的生态与可定制性。

---

## 5. 信息内核 = spec

- **spec** = 结构化自然语言描述的工程实现契约，是 rick 的信息内核。
- 后续 easy job 时，将信息内核以某种形式写入 `.rick/domain`；目标 = **丢弃一切源码，即可完全 AI coding 出一个功能等价的 rick**（human 原话）。
- **信息内核的范围**（human 明确纠正）：信息内核 = rick 项目内**源码 md 文件**（`internal/prompt/templates/` 顶层 loop + skills/），**不是** `.rick/` 目录的运行时文件（那是项目上下文信息）；rick（Go 代码）依然负责提示词拼接逻辑，拼接产物放每 job 的 prompts 目录。

**事实支撑（源码层现状）**：rick 方法层源码 md = `internal/prompt/templates/` 顶层 9 个 loop 文件（ctrl/doing/dream/exporter/learning/plan/research/sense_loop/think）+ `skills/` 约 20 个 skill 文件；拼接逻辑在 `internal/prompt/builder.go`（PromptBuilder·BuildAndSave/BuildAndSaveToDir）、`manager.go`（EnsurePromptsDir/WriteFile）、`human_loop_prompt.go` 等（来源：`research-report-SR-architecture.md` A1/A2 + 源码 `internal/prompt/`）。

---

## 6. 三个目标（O）与关键结果（KR）

### O1：升级 rick 项目的 domain 描述方法，增加 spec 概念，并改造 rick 项目使其拥有这份 spec

- **KR1.1 定义「spec」规范**：产出结构化自然语言工程实现契约的模板与结构（模块边界 / 职责 / 接口契约 / 验收标准），作为 domain 描述方法升级的规范文档。
- **KR1.2 产出 rick 第一份 spec**：覆盖三层金字塔架构 + 5 模块职责（cli/handler/env/builder/runtime）+ builder/runtime 契约 + 验收标准，写入 `.rick/domain/`，使 rick 项目拥有这份 spec。
- **KR1.3 定义 spec 的验收标准**：「spec → 开发计划 → 功能等价实现」；功能等价 = 通过所有功能验收（human 定义）。

### O2：重构当前 rick 设计，使其符合三层金字塔架构

- **KR2.1 落地三层金字塔包/目录结构**：cli（路由+参数解析）/ handler（调度聚合层，编排 env+builder+runtime）/ 执行层三模块。
- **KR2.2 抽出 env 模块**：pi 及扩展依赖的检查/下载/安装/配置/维护更新，从现有 `internal/cmd/tools_init_pi.go`（当前 `requiredExtensions=["pi-subagents","pi-web-access"]`、`ensureNpmExtension`）相关逻辑迁移（来源：`research-report-S-reasons-agent.md` A2）。
- **KR2.3 重构 builder 三件**：templates（go `embed` 内嵌现有 `internal/prompt/templates/` 模板）+ pibuilder（pi 统一入口，组合 plan/doing/easy/human-loop 等子 builder）+ xxxxbuilder（扩展位）。
- **KR2.4 重构 runtime 模块**：把 pi 调用逻辑（参数解析 + 调用，现有 `internal/agent/piagent/`）收口到 runtime 层。
- **KR2.5 下沉 doing 的 dag 调度与门禁检查**：现有 `internal/executor/` + `internal/cmd/tools_doing_check.go` 等中的 dag 调度 agent 逻辑与门禁检查逻辑，改为利用 pi 能力直接实现 dag 调度与门禁，rick 中不再维护（human 原话）。

### O3：优化 pibuilder，将非三层金字塔架构部分的功能下沉到 pi

- **KR3.1 pibuilder 产出 pi 定制化规范产物**：pi agent 运行时所需的所有提示词**内聚在单文件内**，被 pi 以标准规范化定制开发语言引用使用（human 原话）。
- **KR3.2 非三层架构功能下沉到 pi**：基于 pi 最佳定制化开发实践（扩展 / agent / 插件 / skill 等开放组件），如 think/research/exporter 注册为 pi 自定义 agent（frontmatter 落盘 `~/.rick/pi/agent/agents/` 或 `.pi/agents/`，system prompt = 对应源码 skill 的 wiki 内容，来源：`research-report-S-reasons-agent.md` B1~B4）、门禁/dag 用 pi 编排能力实现。
- **KR3.3 触发语言等价迁移**：把各命令模板中 243 处自然语言 subagent 触发词改写为显式 pi 触发语法（`workflowScript` + `runs.run` + 真实 agent 名，来源：`research-report-S-bestpractice.md` BP-1/BP-2），并显式化触发权归属（编排权 parent、单写者，BP-8/BP-6）与 SENSE 特有语义（批判门禁、反向回流、判断记录）。

---

## 7. 闭环逻辑说明

- **O1 ⇒** rick 拥有 spec（结构化工程实现契约），成为可被 AI 转化为等价开发计划的信息内核（丢弃源码即可重构）。
- **O2 ⇒** rick 变薄为引导程序（三层金字塔），dag 调度与门禁下沉 pi，执行层控制力增强。
- **O3 ⇒** pibuilder 深度对齐 pi 定制化开发规范 → 消除 E2/E3 边协议不对齐 → subagent 触发确定性提升。
- **O1+O2+O3 共同 ⇒** 触发确定性提升到上限内最高 + rick 做薄（sense 方法落地）+ 方法/实现隔离（spec 信息内核）。

**三缺口收敛（human 已辩证回答）**：
1. **量化评估**——暂缺（当前靠直觉优化），假设暂时保留；后续专门思考如何评估 rick 优化效果。
2. **模型能力上限**——已辩证逆转：基于进化算法持续迭代提升智能表现，不构成「上限内最高」的阻塞约束（rick 不在乎模型上限）。
3. **无歧义自然语言 ⇒ 等价开发计划**——可接受假设；关键 = spec 对验收标准的无歧义表达（靠功能验收实测验证）。

---

## 8. 派生修订需求（SENSE 过程 human 修订轨迹）

- S：期望由「最大化」修订为「提升到上限内最高」。
- N2 → S-R：K4「深度定制 vs 独立性」由主要矛盾修订为「非主要矛盾」。
- S-R：转义层架构由「skill+loop + 多 runtime 转义层」收敛为「只保留单 runtime（pi）」，dsh 深定制时才架构升级。
- S-R：否定 research 缺口 1（tools 显式化）——tools 隐含在 skill 中，可忽略。
- EC 后：信息内核范围纠正为 rick 源码 md（`internal/prompt/templates/`），非 `.rick/` 运行时文件。
- 最终：信息内核上升为 spec（结构化自然语言工程实现契约，写入 `.rick/domain`）。

---

## 9. 遗留逻辑漏洞（R7 上报项）

> 均标注「待 human 决策 / 已收敛」，不替 human 决策。

- 「触发概率低」的量化/复现证据（N3.3）与「提示词缺口 ⇒ 触发概率低」的因果归属（N3.4）——**待 human 决策**（human 已表态：量化评估暂时保留、靠直觉优化，后续专门思考评估方法）。
- 「spec 是否真正支撑丢弃源码、AI 重构出功能等价的 rick」——**待 human 决策**（需实测）。
- 「无歧义自然语言 ⇒ 等价开发计划」——可接受假设（human 已确认），关键 = spec 对验收标准的无歧义表达，靠功能验收实测验证。
- 模型能力上限——已辩证逆转（进化算法持续迭代），不再作为阻塞项。
- pi vs deepseek-harness 的长期载体选择——human 判断：pi 为当前实现载体，dsh 在带来更强生态与可定制性时才切换（来源：`research-report-debate-dsh-vs-pi.md` 核心观点 + human 设计原则）。

---

## 10. SENSE 流程记录

| 阶段 | 结果 |
|---|---|
| S 问题确认 | ✅ 通过（批判门禁第 2 轮） |
| E 视角生成 | ✅ 通过（视角 = 协议对齐 V5） |
| N1 矛盾生成 | ✅ 通过 |
| N2 主要矛盾判断 | ✅ 通过（选定 K4，后于 S-R 修订为非主要矛盾） |
| S-R 辩证逆转 | ✅ 通过（批判门禁第 2 轮） |
| EC 良知批判 | ✅ 跃迁 = 维持 |

---

## 附：本 RFC 的核心来源

- human 判断原话：`loops/loop_6/judgment.md`
- 事实陈述：`loops/loop_6/briefs/research-report-S.md`、`research-report-S-bestpractice.md`（BP-1~BP-9、D1~D7）、`research-report-S-reasons-agent.md`（B1~B4、A2/C1/C2）、`research-report-N1.md`（F1 系统描述符）、`research-report-SR-architecture.md`（skill+loop 映射）、`research-report-debate-dsh-vs-pi.md`（pi vs dsh 辩论）
- 协议：`loops/loop_6/prompts/sense_loop.md` 及子 agent 协议
