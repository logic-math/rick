# 调研报告 — 跨领域调研 → 多视角候选（subagent 触发确定性） — E 视角生成

日期：2026-08-13
阶段：E（视角生成）· research subagent 输出

---

## 信源配置

| 信源 | 权重 | 验证方式 | 本次使用 |
|---|---|---|---|
| 代码原文 | 0.4 | Read/Grep 直接读取 | ✅（pi-subagents 源码 tool-description.ts / agents.ts / docs；rick 模板与 human_loop_prompt.go） |
| 运行时行为 | 0.3 | Bash 跑命令 | ✅（grep 验证 rick 模板 123 处自然语言 subagent 术语 / 0 处触发语法 / 0 处内置 agent 名；`.pi/subagents/` 运行产物存在） |
| 文档 | 0.2 | Read 官方文档 | ✅（pi-subagents SKILL.md / prompting-and-roles.md / agents.md / models.md / configuration.md） |
| 反事实 | 0.1 | 修改后还原 | ❌ 未使用（E 阶段只读，不改代码） |

**信源限制声明**：本次运行环境**无 WebSearch/WebFetch 工具**（派发单提到"可 WebSearch"，但实际不可用）。因此「来源理论」的出处采用各领域**公认的经典文献**（自知识引用，标注"经典文献"），**未经本次网络检索核验**；如需网络核验具体出处，可后续补派 research。

置信度 = Σ(信源验证结果 × 权重)。高 ≥ 0.8 | 中 0.5-0.8 | 低 < 0.5（R7）。「文件包含/不包含某文字」类事实由直接 Grep 判定为确定（高）。

---

## 尽调树（快照）

```
根：subagent 触发确定性的多视角候选（跨领域）
├─ V1 可靠性工程/控制论视角 ✅高
├─ V2 人机交互/指令遵循视角 ✅高
├─ V3 组织管理/委托代理视角 ✅高
├─ V4 编程语言/编译器（名字解析/确定性派发）视角 ✅高
├─ V5 协议设计（协议对齐/握手）视角 ✅高
└─ V6 认知科学（外部化认知/分布式认知）视角 ✅高
```

树规模：深度 2 | 总节点 7 ≤ 30 ✅。各视角均为「理论视角候选」而非可证伪事实，故按"理论可引用性 + 与已证实事实的一致性"判定为高置信候选（见各候选"融贯性"）。

---

## 事实基础（引用 S 阶段已证实事实 + 本次复核）

S 阶段已证实（沿用）：
- **N3.1**：rick 全部模板中 `workflowScript` / `runs.run` / `runs.all` **零出现**。
- **N3.2**：rick 全部模板中内置 agent 名（scout/worker/reviewer/researcher/delegate/oracle）**零出现**。
- **D1–D7**：触发入口 / agent 名 / 任务形态 / async-context / 编排 recipe / 触发权归属 / 模型认知来源 七项差距。
- **BP-1~BP-9**：pi 最佳实践九条（触发入口唯一且显式、agent 名显式、compact contract、async 默认、context 明确、单写者、编排 recipe、编排权在 parent、模型认知来源=tool description）。
- **B1~B4**：自定义 agent = frontmatter markdown；注册作用域 builtin/package/user/project；触发用 `runs.run(agent:'name')`；rick 现状零自定义 agent。

本次复核补充：
- **N3.1′**：本次 grep 复核 rick 模板自然语言 subagent 术语 **123 处**、pi 触发语法 **0 处**、内置 agent 名 **0 处**（与 S 阶段"150 处"略有出入，系统计口径不同：本次仅 `internal/prompt/templates/`，不含 skills 子目录全部；结论一致）。
- **N3.2′**：`.pi/subagents/` 目录存在，但内容为 **运行产物**（`artifacts/`、`missions/`），**非 agent 定义目录**；agent 定义的项目级作用域为 `.pi/agents/**/*.md`（标准 Pi，见 `docs/agents.md`），该目录**不存在** → 零自定义 agent 的结论仍成立。
- **N3.3′（新增事实）**：pi `docs/agents.md` 明确「Custom agents start with a clean system prompt and only the context you intentionally give them. They do not automatically inherit Pi's whole base prompt, project instruction files, or discovered skills catalog.」→ 自定义 agent 是**空系统提示词起步**，需显式注入上下文。

---

## 多视角候选列表

> 说明：以下每个视角候选均含「来源理论 / 事实支撑 / 融贯性(自洽·他洽·续洽)」三要素。**不替 human 选择视角**，按 S 派发单列示的六个跨领域方向逐一呈现，供 human 综合出原创视角。

---

### 候选 V1：可靠性工程 / 控制论视角

- **来源理论**：可靠性工程中的「确定性响应 / 强制约束（interlock·failsafe）」；控制论的「必要多样性定律（Ashby's Law of Requisite Variety）」与反馈控制——系统对指令的响应确定性取决于控制器能否**强制**被控对象进入期望状态，而非依赖其自发服从。（领域：控制论 / 可靠性工程。经典文献：W. R. Ashby, *An Introduction to Cybernetics*, 1956。）
- **事实支撑**：rick 提示词用软自然语言「派发 subagent / 每个 subagent 独立启动 / SPAWN Sub Agent」（123 处），无任何强制触发约束（N3.1′）；而 pi 的唯一执行面是显式 `subagent({workflowScript:"..."})`（BP-1），官方要求「Run subagents only through { workflowScript }」（C1）。
- **融贯性**：
  - 自洽：把「触发」从「模型自发选择」重述为「控制器的确定性响应」，视角内部逻辑一致。
  - 他洽：与 D1（无触发语法）、C1（官方要求显式执行面）、C2（软指令不可靠）一致。
  - 续洽：预测「把软触发词替换为显式强制语法后，触发确定性显著提升」——该预测与 S 阶段 R7（改正后实测）方向一致，待实测验证。

---

### 候选 V2：人机交互 / 指令遵循视角

- **来源理论**：指令遵循（instruction following）与提示工程——模型对指令的遵循度是「指令显式程度 × 指令形态」的函数；结构化规范（schema/语法）比模糊自然语言遵循度高。（领域：人机交互 / LLM 提示工程。经典文献：LLM instruction-following 评估系列，如 IFEval 等。）
- **事实支撑**：pi 官方旁证「models don't always do this; use prompting or /skill:name to force it」（C2）；模型对 subagent 工具的认知来源是 tool description 文本（BP-9）；rick 提示词为长 procedural 脚本（sense_loop 五阶段 / plan 六维评审）而非 compact contract（D3）。
- **融贯性**：
  - 自洽：以「遵循度」为因变量、以「指令显式性/形态」为自变量，内部一致。
  - 他洽：与 C2、BP-9、D3 一致。
  - 续洽：预测「compact contract + 显式触发词 比 长脚本 + 软触发词 遵循度高」，可被后续实测触发率验证。

---

### 候选 V3：组织管理 / 委托代理（principal-agent）视角

- **来源理论**：委托代理理论（agency theory）——委托人（parent）通过**契约（contract）+ 权威边界（authority boundary）+ 监督** 约束代理人（child）行为，解决目标不一致与信息不对称；契约应写清目标、授权、验收、停止条件。（领域：组织经济学。经典文献：Jensen & Meckling, "Theory of the Firm: Managerial Behavior, Agency Costs and Ownership Structure", 1976；Holmström & Milgrom, 1991。）
- **事实支撑**：pi 官方定义「强 subagent prompt = compact contract」，含 Goal / Target / Authority boundary / Success criteria / Hard constraints / Validation / Output / Stop rules（BP-3）；编排权在 parent、子 agent 默认不持 subagent 工具（BP-8）；单写者约束（BP-6）；rick 提示词未区分「谁触发」（D6）。
- **融贯性**：
  - 自洽：把「触发子 agent」重述为「委托」，把「提示词」重述为「契约」，内部一致。
  - 他洽：与 BP-3、BP-8、D6 一致。
  - 续洽：预测「明确 authority boundary + stop rules 后，子 agent 执行更可控、越权/遗漏减少」，可由后续 RFC 落地验证。

---

### 候选 V4：编程语言 / 编译器（名字解析 / 确定性派发）视角

- **来源理论**：编译原理中的符号表（symbol table）、名字绑定（name binding）、名字解析（name resolution）、静态/动态派发（dispatch）——符号引用必须能解析到**唯一确定的实现**，否则编译期报"未定义符号"错误。（领域：编程语言 / 编译器。经典文献：Aho, Lam, Sethi, Ullman, *Compilers: Principles, Techniques, and Tools*（龙书）, 2nd ed., 2006。）
- **事实支撑**：rick 提示词用自造角色名 `think/research/exporter`、`subagent_1~6`、`父/子 Agent`，与 pi 内置 agent 名无映射（N3.2）；pi 触发必须写真实 `agent:'worker'` 等名字，名字不在名单被 launch 拒绝（"Unknown agent"）（BP-2）；自定义 agent 通过 frontmatter `name` 字段注册（B1/B2），触发用 `runs.run(agent:'name')`（B3）。
- **融贯性**：
  - 自洽：「自造角色名 = 未绑定符号；注册 agent = 建立符号表项；显式 agent 名 = 名字解析」的类比内部一致。
  - 他洽：与 N3.2、BP-2、B1/B2/B3 一致。
  - 续洽：预测「把自造名注册为真实 agent 名（符号表项）后，引用可解析、触发确定」，与 human 已确认方向（内置为系统级 agent）语义一致。

---

### 候选 V5：协议设计（协议对齐 / 握手）视角

- **来源理论**：通信协议设计中的协议对齐（protocol alignment）与握手（handshake）——双方（模型↔工具）通信的前提是共享同一份协议规范（消息格式、触发原语、可接受动作集）；未对齐即握手失败。（领域：计算机网络 / 分布式系统。经典文献：A. S. Tanenbaum, *Computer Networks*。）
- **事实支撑**：模型对 subagent 工具的认知来源 = tool description 文本（BP-9），其中明确触发语法 + 内置 agent 名 + async/context 语义（即"协议规范"）；rick 提示词与 tool description 语法/agent 名**不对齐**（D7、D1、D2）；官方唯一触发路径 = workflowScript（C1）。
- **融贯性**：
  - 自洽：把「触发概率低」重述为「模型侧协议实现 与 工具侧协议规范 未对齐」，内部一致。
  - 他洽：与 BP-9、C1、D7/D1/D2 一致。
  - 续洽：预测「提示词与 tool description 对齐后，握手（触发）成功率提升」，可由实测触发率验证。

---

### 候选 V6：认知科学（外部化认知 / 分布式认知）视角

- **来源理论**：分布式认知（distributed cognition）——认知不只发生在个体脑内，也分布在外部表征（工具、提示词、规范）中；把规则**外部化**为环境中的显式约束，可降低个体认知负荷、提升行为一致性。辅以双过程理论（dual-process）区分快速直觉式 vs 慢速分析式处理。（领域：认知科学。经典文献：E. Hutchins, *Cognition in the Wild*, 1995；D. Kahneman, *Thinking, Fast and Slow*, 2011。）
- **事实支撑**：pi 官方旁证「模型不总会遵守软指令，需显式提示强制」（C2）；rick 软触发词属"内隐语境"、无外部化触发规则（A3、N3.1′）；pi 把触发规则外部化为 tool description 语法（BP-9）；自定义 agent 空系统提示词起步、需显式注入上下文（N3.3′）。
- **融贯性**：
  - 自洽：把「提升触发确定性」重述为「把触发规则从内隐语境外部化到提示词/工具描述」，内部一致。
  - 他洽：与 C2、A3、BP-9、N3.3′ 一致。
  - 续洽：预测「把触发规则外部化为显式语法后，认知负荷降低、遵循度提升」，可由实测验证。

---

## R7 上报项（无法达高置信的叶节点）

1. **各视角「来源理论」的网络出处核验**——本次无 WebSearch 工具，理论出处为经典文献自知识引用，未经网络核验。
2. **「按任一视角改进后，触发确定性能提升多少」的量化证据**——需真实 pi 会话实测（延续 S 阶段 R7）。
3. **各视角之间的优劣/综合关系**——属 human 综合判断（原创视角），research 不替 human 选择。

---

## 整合摘要

多视角候选 6 个（V1~V6，均高置信候选）| 事实基础沿用 S 阶段 + 本次复核 3 项补充 | R7 上报 3 项。

**关键提示（事实性，非建议）**：六个视角共享同一组已证实事实（N3.1/N3.2、D1~D7、BP-1~BP-9、B1~B4、C1/C2），只是用不同领域的理论**重新表述**同一问题；human 可在其中综合、或提出候选未覆盖的新视角。
