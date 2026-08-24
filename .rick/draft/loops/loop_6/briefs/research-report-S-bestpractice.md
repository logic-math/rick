# 调研报告 — pi 触发 subagent 搭建工作流的最佳实践 + 与 rick 现状的差距 — S 追加调研

日期：2026-08-13
阶段：S（问题确认）· research subagent 追加调研（human 提问：pi 最佳实践是什么）

---

## 信源配置

| 信源 | 权重 | 验证方式 | 本次使用 |
|---|---|---|---|
| 代码原文 | 0.4 | Read/Grep 直接读取 | ✅（pi-subagents 扩展源码 tool-description.ts / agents/*.ts / agents/*.md） |
| 运行时行为 | 0.3 | Bash 跑命令 | ⚠️ 部分（未做 live spawn，S 阶段只读） |
| 文档 | 0.2 | Read 官方文档/规范 | ✅（SKILL.md + references/*.md + docs/*.md） |
| 反事实 | 0.1 | 修改后还原 | ❌ 未使用 |

置信度 = Σ(信源验证结果 × 权重)。高 ≥ 0.8 | 中 0.5-0.8 | 低 < 0.5（R7）。

**计分说明**：「文件包含/不包含某文字」由直接 Read/Grep 判定即为确定（高）；「模型实际是否按最佳实践触发」属运行时行为，本次未做 live 复现，相关项标注。

---

## 一、最佳实践清单（pi 官方/推荐写法）

来源：`pi-subagents/skills/pi-subagents/SKILL.md` + `references/prompting-and-roles.md`、`references/execution-controls.md`、`references/constraints-and-recipes.md` + `src/extension/tool-description.ts` + `agents/*.md` + `prompts/*.md`。

### BP-1：触发入口唯一且显式 —— subagent 工具 + workflowScript
- **事实**：pi 下触发 subagent 的唯一公开执行面是 `subagent` 工具的 `{ workflowScript: "..." }` 字段；脚本内用全局 `runs.run(key, {agent, task})`（单子）/ `runs.all([...])`（并行）启动子 agent。省略 `action` 表示执行，`action` 用于管理/控制。
- **证据**：`tool-description.ts` 的 `FULL_SUBAGENT_TOOL_DESCRIPTION` 首句「Run subagents only through { workflowScript }; omit action.」；`execution-controls.md`「workflowScript is the sole public execution surface」。
- **置信度**：高（代码原文 + 文档双信源）。
- **官方示例**：`subagent({ workflowScript: "return runs.run('main', {agent:'worker', task:'...'})" })`。

### BP-2：agent 名必须显式引用内置/自定义 agent 名
- **事实**：内置 agent 为 `scout` / `worker` / `reviewer` / `researcher` / `delegate` / `oracle`（别名 `advisor`）。触发时必须写 `agent: 'worker'` 等真实名字；名字不在名单会被 launch 拒绝（"Unknown agent"）。
- **证据**：`prompting-and-roles.md`「Builtin Agents」表 + `agents/*.md` 各角色 frontmatter（name 字段）；`agents.ts` 枚举。
- **内置角色用途（事实）**：scout=代码侦察；worker=实现/单写者；reviewer=评审（默认只读）；researcher=**网页**研究（web_search）；delegate=轻量通用委托；oracle=高上下文决策一致性顾问（默认 fork）。
- **置信度**：高。

### BP-3：任务文本 = compact contract（目标/目标物/权限边界/上下文/成功标准/硬约束/验证/输出/停止规则）
- **事实**：官方明确「write the task prompt as a compact contract, not a long procedural script」，应包含 Goal / Target（repo、cwd、branch、source seam）/ Authority boundary（能否 edit/commit/push…）/ Context & evidence / Success criteria / Hard constraints（真正不变式）/ Validation / Output（形状+路径）/ Stop rules（何时 intercom/contact_supervisor、何时停）。
- **证据**：`prompting-and-roles.md`「A strong subagent prompt usually includes…」整段。
- **置信度**：高。

### BP-4：async 默认
- **事实**：每个 subagent 默认异步（`async: true`），`async:false` 仅用于需要阻塞前台结果的小任务。脚本式 workflow 默认异步启动。
- **证据**：`SKILL.md`「Scripted workflows start asynchronously by default」；`constraints-and-recipes.md`「Prefer async orchestration」。
- **置信度**：高。

### BP-5：context fresh/fork 明确
- **事实**：子 agent 上下文二选一：`context: "fresh"`（最小新上下文）或 `context: "fork"`（分支继承父会话历史）。对抗式评审用 fresh，顾问/实现默认 fork（worker/oracle/advisor 默认 fork）。
- **证据**：`execution-controls.md`「Forked context」「context: "fork" creates a branched child session…」；`constraints-and-recipes.md`「Use fork for branched advisory or execution threads」。
- **置信度**：高。

### BP-6：单写者（one writer per cwd/worktree）
- **事实**：同一 cwd/worktree 只允许一个写者；并行只用于读/评审/验证/研究，写要么串行、要么用 `worktree:true` 隔离。
- **证据**：`SKILL.md`「Use one writer per cwd/worktree」；`constraints-and-recipes.md`「Keep writes single-threaded by default」。
- **置信度**：高。

### BP-7：可复用编排 recipe（packaged prompt workflows）
- **事实**：官方随包提供可复用编排模式 `prompts/*.md`：`parallel-review`（fresh 多角度评审）、`gather-context-and-clarify`（scout/research → interview）、`parallel-research`（researcher+scout）、`review-loop`（worker→reviewer→fix worker 循环）、`parallel-cleanup`（deslop+verbosity 评审）。提示词可直接「apply the same pattern directly with subagent(...)」。
- **证据**：`prompting-and-roles.md`「Applying Prompt Techniques Without Slash Commands」整节 + `prompts/` 目录 5 个文件。
- **置信度**：高。

### BP-8：子 agent 默认不持 subagent 工具，编排权在 parent
- **事实**：普通子 agent **不接收** `subagent` 扩展工具、不接收 `pi-subagents` skill；只有显式配置 `tools: subagent` 的 fanout 子 agent 才能再派发（且受深度限制，默认 2 层）。因此「谁触发 subagent」只能是 parent orchestrator。
- **证据**：`constraints-and-recipes.md`「Ordinary children also do not receive the subagent extension tool…Every child receives a boundary instruction: ordinary children are told the parent owns orchestration…」。
- **置信度**：高。**此项对「触发确定性」关键**：提示词必须让 parent（main agent）触发，而非让子 agent 递归触发。

### BP-9：模型对 subagent 工具的认知来源 = tool description
- **事实**：模型能调用 subagent 工具，靠的是工具描述文本 `FULL_SUBAGENT_TOOL_DESCRIPTION`（或 compact/custom）。描述里给出触发语法、内置 agent 名、async/context 语义、安全护栏。`toolDescriptionMode` 可配 full/compact/custom。
- **证据**：`tool-description.ts` 全文；`prompting-and-roles.md`「Tool description modes live in … config.json」。
- **置信度**：高。
- **含义（事实性陈述，非建议）**：提示词若想让模型确定触发 subagent，需与工具描述中的语法/agent 名对齐——这是「提示词措辞」能影响触发确定性的机制入口。

---

## 二、rick 现状（引用前序 research-report-S.md 已证实事实）

- **N3.1**：rick 全部模板（`internal/prompt/templates/` 含 skills/）中 `workflowScript` / `runs.run` / `runs.all` 均**零出现**。
- **N3.2**：rick 全部模板中内置 agent 名（scout/worker/reviewer/researcher/delegate/oracle）**零出现**。
- **G2 层**：human-loop（think/research/exporter）、plan（subagent_1~6）、dream（subagent_1~4）、doing/easy（Main/Sub Agent）、learning（父/子 Agent）均用**自然语言**要求「用 subagent」，全文共 **150 处**自然语言 subagent 术语，但零触发语法、零内置 agent 名。

---

## 三、差距对照表（最佳实践 vs rick 现状）

| # | 维度 | pi 最佳实践 | rick 现状 | 差距 |
|---|---|---|---|---|
| D1 | 触发入口 | 显式 `subagent({workflowScript:"..."})` + `runs.run/runs.all` | 自然语言「派发 subagent / SPAWN Sub Agent / 子 Agent」 | 提示词中无任何一行 pi 触发语法（N3.1） |
| D2 | agent 名 | 显式 `agent:'scout/worker/reviewer/researcher/delegate/oracle'` | 自造角色名 think/research/exporter、subagent_1~6、父/子 Agent | 自造名与内置名无映射（N3.2） |
| D3 | 任务形态 | compact contract（goal/target/authority/success criteria/validation/stop rules） | 长 procedural 流程脚本（如 sense_loop 五阶段、plan 六维评审步骤） | 形态偏「过程脚本」而非「契约」 |
| D4 | async/context | 显式 `async:true` + `context:"fresh"/"fork"` | 无 | 提示词未声明运行模式 |
| D5 | 编排 recipe | 引用 packaged recipes（parallel-review/gather-context-and-clarify 等）与 SKILL.md | 未引用 pi-subagents SKILL 或 recipes | 未接入官方编排范式 |
| D6 | 触发权归属 | 编排权在 parent，子 agent 默认不持 subagent 工具 | 提示词未区分「谁触发」（自然语言下模型可能让子 agent 或 parent 各自理解） | 无明确触发权归属约束 |
| D7 | 模型认知来源 | 模型靠 tool description 认识 subagent 工具 | 提示词与 tool description 语法/agent 名不对齐 | 提示词措辞无法借力工具描述 |

---

## 四、R7 上报项（无法达高置信度的叶节点）

1. **「触发概率低」的量化/复现证据**——理由：human 观察，S 阶段只读无法在真实 pi 模型会话复现触发率（延续 N3.3）。
2. **「提示词缺口 ⇒ 触发概率低」的因果归属**——理由：结构性缺口（D1/D2/D6/D7）已证实，但因果归属属假设，非事实可证（延续 N3.4）。
3. **「按最佳实践对齐后触发确定性是否/能提升到多少」**——理由：需真实 pi 会话实测，属运行时行为，本次未做 live spawn，无法高置信。

---

## 整合摘要

最佳实践 9 条（BP-1~BP-9，均高置信）| 差距对照 7 项（D1~D7）| R7 上报 3 项。
