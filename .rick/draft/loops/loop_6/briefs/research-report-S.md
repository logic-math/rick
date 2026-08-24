# 调研报告 — subagent 在 pi runtime 下触发概率低（优化提示词） — S 问题确认

日期：2026-08-13
阶段：S（问题确认）· research subagent 输出

---

## 信源配置

| 信源 | 权重 | 验证方式 | 本次使用 |
|---|---|---|---|
| 代码原文 | 0.4 | Read/Grep 直接读取 | ✅（rick Go 源码、嵌入模板、pi-subagents 扩展源码） |
| 运行时行为 | 0.3 | Bash 跑命令/测试/日志 | ✅（`pi list`、`go test ./internal/prompt/...`） |
| 文档 | 0.2 | Read 官方文档/规范 | ✅（pi-subagents SKILL.md + references、内置 agent 定义） |
| 反事实 | 0.1 | 修改代码看影响后还原 | ❌ 本次未使用（S 阶段只读，不改代码） |

置信度 = Σ(信源验证结果 × 权重)。高 ≥ 0.8 | 中 0.5-0.8 | 低 < 0.5（R7 上报）。

**计分说明**：「某文件包含/不包含某文字」类事实，由直接 Read/Grep 判定即为**确定**（标注高置信），不机械套加权公式；加权公式用于「外部机制如何工作」等需多信源交叉的事实。二者在节点详情中分别注明证据。

---

## 尽调树（快照）

```
根：subagent 在 pi runtime 下触发概率低（优化提示词）
├─ G1 运行时层：pi runtime 的 subagent 触发机制
│   ├─ N1.1 pi-subagents 扩展已注册 ✅0.9 高
│   ├─ N1.2 subagent 触发工具形态（subagent 工具 + workflowScript + runs.run/runs.all）✅0.9 高
│   └─ N1.3 内置 agent 名单（scout/worker/reviewer/researcher/delegate/oracle/advisor）✅0.9 高
├─ G2 提示词层：rick 各命令提示词如何要求 subagent
│   ├─ N2.1 human-loop（sense_loop 模板）要求派发 think/research/exporter ✅高
│   ├─ N2.2 plan 六维评审 subagent_1~6 ✅高
│   ├─ N2.3 dream 质量验证 subagent_1~4 ✅高
│   ├─ N2.4 doing/easy 经 doing_loop skill 要求 Main/Sub Agent ✅高
│   ├─ N2.5 learning 经 learning_loop skill 要求 父/子 Agent ✅高
│   └─ N2.6 ctrl 无 subagent 要求 ✅高
└─ G3 缺口层：提示词 subagent 语言 vs 运行时触发机制的对应关系
    ├─ N3.1 提示词未引用 pi 触发工具（subagent 工具/workflowScript/runs.run）→ 属实 ✅高
    ├─ N3.2 提示词未引用内置 agent 名 → 属实 ✅高
    ├─ N3.3 「触发概率低」的量化/复现证据 → R7（无法只读复现）
    └─ N3.4 「提示词缺口 ⇒ 触发概率低」的因果归属 → R7（属假设，非事实可证）
```

树规模：深度 3 ≤ 5 | 每层子节点 ≤ 7 | 总节点 17 ≤ 30 ✅

---

## 节点详情

### G1 运行时层：pi runtime 的 subagent 触发机制

#### N1.1：pi-subagents 扩展已注册
- **事实陈述**：rick 托管的 pi 运行环境已注册 pi-subagents 扩展（提供 subagent 工具）。
- **置信度**：0.9（高）
- **信源验证**：
  - 代码原文 ✅0.4：`internal/cmd/tools_init_pi.go` 中 `ensureNpmExtension("pi-subagents", "pi-subagents")`、`requiredExtensions = ["pi-subagents", "pi-web-access"]`。
  - 运行时 ✅0.3：`pi list` 输出含 `npm:pi-subagents`；`~/.rick/pi/agent/settings.json` 的 `packages` 含 `"npm:pi-subagents"`。
  - 文档 ✅0.2：`pi-subagents/skills/pi-subagents/SKILL.md` 为 subagent 委托技能。
  - 反事实 ❌ 未使用。

#### N1.2：subagent 触发工具形态
- **事实陈述**：pi 下触发 subagent 的唯一执行入口是 `subagent` 工具的 `{ workflowScript: "..." }`，脚本内用全局 `runs.run(...)`（单子）/ `runs.all([...])`（并行/编排）启动子 agent，省略 `action` 字段表示执行、`action` 字段用于管理/控制。
- **置信度**：0.9（高）
- **信源验证**：
  - 代码原文 ✅0.4：`pi-subagents/src/extension/tool-description.ts` 明确定义 `FULL_SUBAGENT_TOOL_DESCRIPTION`（"Run subagents only through { workflowScript }"、`runs.run('main', {agent:'worker', task:'...'})`、`runs.all([...])`）。
  - 运行时 ✅0.3：扩展已注册（`pi list`）即工具可用；**未做 live spawn 复现**（需真实模型会话，S 阶段只读）。
  - 文档 ✅0.2：SKILL.md + `references/execution-controls.md`、`prompting-and-roles.md`。
  - 反事实 ❌ 未使用。

#### N1.3：内置 agent 名单
- **事实陈述**：pi-subagents 内置 agent 为 scout / worker / reviewer / researcher / delegate / oracle / advisor（advisor 为 oracle 的别名）。
- **置信度**：0.9（高）
- **信源验证**：
  - 代码原文 ✅0.4：`pi-subagents/src/agents/agents.ts` 第 39-45 行枚举 advisor/delegate/oracle/researcher/reviewer/scout/worker；`pi-subagents/agents/` 目录含 delegate.md / oracle.md / researcher.md / reviewer.md / scout.md / worker.md。
  - 运行时 ✅0.3：扩展已注册（工具可用），未做 live `{action:"list"}` 枚举。
  - 文档 ✅0.2：`prompting-and-roles.md` 的 Builtin Agents 表格。
  - 反事实 ❌ 未使用。

---

### G2 提示词层：rick 各命令提示词如何要求 subagent

（以下均为「文件内容」类事实，直接 Read 确定，置信高。模板为 `internal/prompt/templates/*.md`，运行时以 `go test ./internal/prompt/...` 通过佐证模板可正确嵌入。）

#### N2.1：human-loop（sense_loop 模板）要求派发 think/research/exporter
- **置信度**：高
- **证据**：`templates/sense_loop.md` 反复出现「派发 subagent」；第 6-8 行声明 think/research/exporter 三个 subagent 路径；第 16 行「派发 subagent → 展示简报 → 嵌入批判门禁…」。`internal/prompt/human_loop_prompt.go` 生成并落盘 think/research/exporter 三个 subagent prompt 文件（第 73/88/103 行）。

#### N2.2：plan 六维评审 subagent_1~6
- **置信度**：高
- **证据**：`templates/plan.md` 第 118-125 行「六维评审（每个 subagent 独立启动，串行执行）」列出 subagent_1~6。

#### N2.3：dream 质量验证 subagent_1~4
- **置信度**：高
- **证据**：`templates/dream.md` 第 124-166 行「质量验证（4 个子 Agent 串行）」列出 subagent_1~4，并全文使用「父 Agent / 子 Agent」术语。

#### N2.4：doing/easy 经 doing_loop skill 要求 Main/Sub Agent
- **置信度**：高
- **证据**：`templates/doing.md` 注入 `{{doing_loop_content}}`；`templates/skills/doing_loop.md` Step 3「每轮迭代由 Main Agent 启动一个独立 Sub Agent」「SPAWN Sub Agent」。`internal/prompt/easy_prompt.go` 用 `LoadTemplate("doing")` + 注入 doing_loop 内容（第 70/83 行），故 easy 与 doing 同源。

#### N2.5：learning 经 learning_loop skill 要求 父/子 Agent
- **置信度**：高
- **证据**：`templates/learning.md` 注入 `{{learning_loop_path}}`；`templates/skills/learning_loop.md`「每个 Step 由父 Agent 启动一个独立子 Agent执行」并给出 父 Agent/Step N 子 Agent 结构图。

#### N2.6：ctrl 无 subagent 要求
- **置信度**：高
- **证据**：`templates/ctrl.md` 全文为监控/干预 agent 职责，无「subagent」「子 Agent」「Sub Agent」字样（grep 0 命中）。

---

### G3 缺口层：提示词 subagent 语言 vs 运行时触发机制的对应关系

#### N3.1：提示词未引用 pi 触发工具
- **事实陈述**：rick 全部模板（含 skills/）中，`subagent` 工具、`workflowScript`、`runs.run`、`runs.all` 均零出现。
- **置信度**：高（grep 判定为确定）
- **证据**：`grep -rn "workflowScript\|runs\.run\|runs\.all" internal/prompt/templates/` → 0 命中。模板对 subagent 的描述全部是自然语言（「每个 subagent 独立启动」「派发 subagent」「SPAWN Sub Agent」「父 Agent 启动独立子 Agent」），无任何一行给出 pi 下实际触发 subagent 的 `subagent({workflowScript:...})` 语法。

#### N3.2：提示词未引用内置 agent 名
- **事实陈述**：rick 全部模板中未出现 pi-subagents 内置 agent 名（scout/worker/reviewer/researcher/delegate/oracle）。
- **置信度**：高（grep 判定为确定）
- **证据**：`grep -rn "scout\|reviewer\|researcher\|delegate\|oracle\|worker" internal/prompt/templates/` → 0 命中（排除 "research" 词干干扰后）。模板用 `subagent_1~6`、`子 Agent`、`Sub Agent`、`think/research/exporter` 等自造角色名，与 pi 运行时的内置 agent 名无映射关系。

#### N3.3：「触发概率低」的量化/复现证据 → R7
- **理由**：human 观察为「触发概率低」，本次 S 阶段只读调研无法在真实 pi 模型会话中复现/量化触发率（需真实 provider/model 会话，非确定性、成本高）。无高置信可验证的触发率数据。
- **状态**：无法澄清（低置信，无疑问点可下钻）→ R7 上报，供 human 决策。

#### N3.4：「提示词缺口 ⇒ 触发概率低」的因果归属 → R7
- **理由**：结构性缺口（N3.1/N3.2）已证实：提示词要求「用 subagent」却未给出 pi 运行时的触发语法与 agent 名。但「该缺口是触发概率低的（唯一/主要）原因」属**因果假设**，非事实可证（可能还涉及 model 能力、pi 配置、其他运行时约束）。因果归属留待 think/human 判断。
- **状态**：无法澄清 → R7 上报。

---

## R7 上报项（无法达高置信度的叶节点）

- **N3.3**：「触发概率低」的量化/复现证据——理由：human 观察，S 阶段只读无法复现真实 pi 会话触发率。
- **N3.4**：「提示词缺口 ⇒ 触发概率低」的因果归属——理由：结构性缺口已证实，但因果归属属假设，非事实可证，留待 think/human 判断。

---

## 整合摘要

总节点数 17 | 高置信度叶节点 11 | R7 上报 2
