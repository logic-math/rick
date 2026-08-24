# 调研报告 — 触发概率低的其他因素 + pi 自定义 agent 机制 — S 追加调研

日期：2026-08-13
阶段：S（问题确认）· research subagent 追加调研（human 追问：A6 其他因素 + A4 系统级 agent 方向）

---

## 信源配置

| 信源 | 权重 | 验证方式 | 本次使用 |
|---|---|---|---|
| 代码原文 | 0.4 | Read/Grep 直接读取 | ✅（pi-subagents 源码 tool-description.ts / agents.ts / frontmatter.ts / docs/*.md；rick settings.json / tools_init_pi.go / human_loop_prompt.go） |
| 运行时行为 | 0.3 | Bash 跑命令 | ⚠️ 部分（未做 live spawn；`pi list` 已在前序报告确认扩展注册） |
| 文档 | 0.2 | Read 官方文档 | ✅（pi-subagents docs/agents.md / configuration.md / tool-reference.md / models.md；pi docs/compaction.md / skills.md） |
| 反事实 | 0.1 | 修改后还原 | ❌ 未使用（S 阶段只读） |

置信度 = Σ(信源验证结果 × 权重)。高 ≥ 0.8 | 中 0.5-0.8 | 低 < 0.5（R7）。「文件包含/不包含」类事实由直接 Read/Grep 判定为确定（高），不机械套加权。

---

## 尽调树（快照）

```
根：触发概率低的其他因素 + 自定义 agent 机制
├─ A 其他因素层
│   ├─ A1 模型 tool-calling 能力差异 → R7（本地无 benchmark 信源）
│   ├─ A2 pi 运行时/配置因素 → ✅高（扩展已注册、工具可用、默认 full 描述、无禁用）
│   └─ A3 提示词结构性因素（软触发词/长脚本） → ✅高（并入 D3 形态问题）
├─ B 自定义 agent 机制层
│   ├─ B1 定义方式（frontmatter markdown） → ✅高
│   ├─ B2 注册作用域（builtin/package/user/project 优先级） → ✅高
│   ├─ B3 触发方式（runs.run + 显式 agent 名） → ✅高
│   └─ B4 rick 现状（think/research/exporter 非 pi agent，零自定义 agent） → ✅高
└─ C 佐证层
    ├─ C1 pi 官方要求显式 workflowScript+agent 名 → ✅高
    └─ C2 「软自然语言不可靠、需显式提示」官方旁证 → ✅高
```

树规模：深度 3 | 总节点 11 ≤ 30 ✅

---

## 分支 A：除「提示词未对齐」外，还有哪些因素影响 subagent 触发概率

### A1：模型 tool-calling 能力差异 → R7（无法达高置信）

- **事实陈述**：不同模型对工具调用（tool-calling）的遵循度存在差异，可能影响 subagent 工具被调用的概率。
- **证据现状**：本地 pi 文档仅间接涉及——`docs/models.md` 的「Recommended model tiering」提到 tier-3 高推理模型「tend to loop on vague goals」、tier-4 需「a model that reads human intent well」；`docs/agents.md` 说明内置 agent「do not pin a provider model; they inherit your current Pi default model」。
- **结论**：**模型选择会影响行为**是官方隐含承认的，但「deepseek/claude/gemini/gpt 之间 tool-calling 遵循度的量化差异」本地无可验证 benchmark 信源（本次无 WebSearch 工具）。→ **R7 上报**：需外部 benchmark/实测，S 阶段无法达高置信。

### A2：pi 运行时/配置因素 → 已排除（高置信）

逐项核查 rick 的实际配置（`~/.rick/pi/agent/settings.json` + 有无扩展 config）：

| 因素 | 检查结果 | 置信度 |
|---|---|---|
| `toolDescriptionMode` | 未配置 → 默认 `full`（最完整描述，`tool-description.ts` `resolveToolDescriptionMode` 返回 full） | 高 |
| 扩展注册 | `packages` 含 `npm:pi-subagents` + `npm:pi-web-access`；`tools_init_pi.go` `requiredExtensions=["pi-subagents","pi-web-access"]` | 高 |
| subagent 工具是否禁用 | 无 `disableBuiltins` / 无 `disabled` override / 无 `subagents` 配置块 → 内置 agent 全可用 | 高 |
| `toolBudget`/`turnBudget`/`usageBudget` | rick settings.json 无；这些是 launch 参数（非配置层），rick 提示词也未声明 | 高 |
| 扩展 config.json | `~/.rick/pi/agent/extensions/subagent/config.json` 不存在 → 扩展全默认（asyncByDefault=true 等） | 高 |

**结论**：配置层不是触发概率低的因素——扩展已注册、subagent 工具可用、工具描述为最完整的 `full` 模式、无任何禁用项。rick 当前使用 pi-subagents 的**全默认配置**。

### A3：提示词结构性因素（非「对齐」层面）→ 高置信，实质归入 D3

- **事实**：rick 模板 150 处自然语言 subagent 术语（「派发 subagent」「每个 subagent 独立启动」「SPAWN Sub Agent」「父 Agent 启动独立子 Agent」），触发条件均为**软性自然语言**，无硬性触发语法（延续 N3.1/N3.2）。
- **工具描述不被 compaction 破坏**（事实）：工具定义（tool description）在 system prompt 层，`compaction.md` 的 message 压缩只作用于会话消息（tool result 截断 2000 字符、summarize 工具调用），**不压缩工具定义**。因此「工具说明丢失」不是因素。
- **长 procedural 脚本被 summarize 的风险**（事实，非「对齐」）：rick 提示词是长流程脚本（sense_loop 五阶段 / plan 六维评审），长会话中被 summarize 时，subagent 段可能被压缩——这是「提示词形态（D3）」带来的风险，而非「未引用触发语法」。
- **结论**：结构性因素 = 软触发词 + 长脚本形态，二者均指向 D3（任务形态偏过程脚本而非契约），与 D1/D2（无语法/无 agent 名）共同构成「提示词未对齐」的完整面。

---

## 分支 B：pi 自定义 agent / 系统级 agent 机制

### B1：自定义 agent 定义方式（高置信）

- **事实**：一个 agent = 一个 markdown 文件：YAML frontmatter 在上（`name` / `description` / `tools` / `model` / `systemPromptMode` / `inheritProjectContext` / `inheritSkills` / `defaultContext` / `acceptanceRole` / `async` / `timeoutMs` 等），system prompt 在下。
- **最小示例**（`docs/agents.md`）：
  ```yaml
  ---
  name: scout
  description: Fast codebase recon
  tools: read, grep, find, ls
  ---
  Your system prompt goes here.
  ```
- **证据**：`docs/agents.md`（Frontmatter reference 全文）+ `src/agents/frontmatter.ts`（parseFrontmatter）+ `src/agents/agents.ts`（AgentConfig/BUILTIN_AGENT_NAMES）。

### B2：注册作用域与优先级（高置信）

| 作用域 | 路径 | 优先级 |
|---|---|---|
| Builtin | `~/.pi/agent/extensions/subagent/agents/`（pi-subagents 包内） | 最低 |
| Installed package | package.json `pi-subagents.agents` | 次低 |
| User | `~/.pi/agent/agents/**/*.md` | 高于 package |
| Project | `.pi/agents/**/*.md`（标准 Pi） | 最高（覆盖 user） |

- 遗留 `.agents/**/*.md` 也可被 project 发现。
- **rick 隔离环境映射**：rick 用 `PI_CODING_AGENT_DIR=~/.rick/pi/agent` 隔离配置（`tools_init_pi.go` `piCommand` 注入 `piagent.AgentEnv()`），故「user 级」= `~/.rick/pi/agent/agents/**/*.md`，「project 级」= 仓库 `.pi/agents/**/*.md`。

### B3：如何用明确提示词触发自定义 agent（高置信）

- **事实**：定义好 agent（name=think）后，parent 用 `runs.run` 显式引用其名：
  `subagent({ workflowScript: "return runs.run('think', { agent: 'think', task: '...' })" })`
- agent 名支持 `aliases`；`{action:"list"}` 发现、`{action:"create", config:{name, description, systemPrompt, ...}}` 动态创建、`{action:"eject"}` 复制内置为可编辑文件。
- 证据：`docs/tool-reference.md`（Execution examples + Management actions + create config）+ `docs/agents.md`。

### B4：rick 现状对照（高置信）

- **think/research/exporter 不是 pi 自定义 agent**：`human_loop_prompt.go` 生成的是**普通 markdown 提示词文件**，写到 `.rick/draft/loops/loop_N/prompts/*.md`（think.md/research.md/exporter.md/sense_loop.md + skill_*.md），**无 YAML frontmatter、无 name/description 字段、不在任何 pi agent 发现目录**。
- **rick 零自定义 agent**：仓库无 `.pi/agents/`、无 `.agents/`；`~/.rick/pi/agent/agents/` 不存在。
- **rick settings.json 无 `subagents` 配置块**：无 defaultModel / agentOverrides / disableBuiltins。
- **结论**：要把 think/research/exporter「内置为系统级 agent」，需把它们注册为 pi 自定义 agent——写 frontmatter markdown 到 `~/.rick/pi/agent/agents/`（user 级，随 init-pi 落盘）或 `.pi/agents/`（project 级），或由 rick 命令/init-pi 用 `{action:"create"}` 动态写入；随后提示词用显式 `agent:'think'` 触发。当前 rick 无此注册。

---

## 分支 C：佐证「提示词缺口是主要原因」的旁证

### C1：pi 官方明确要求显式 workflowScript + agent 名（高置信）

- `tool-description.ts` `FULL_SUBAGENT_TOOL_DESCRIPTION` 首句：「**Run subagents only through { workflowScript }; omit action.**」+ 示例 `runs.run('main', {agent:'worker', task:'...'})` + 安全护栏「Before executing, use { action: "list" } and run only executable/non-disabled configured agents.」
- `skills/pi-subagents/SKILL.md`：「Use `workflowScript` for all execution」+「Use `return runs.run("main", { agent, task })`」。
- `docs/agents.md`：「**Agent definitions are not loaded into context by default.** Management actions let the LLM discover, inspect, create, update, and delete agents...」→ 内置/自定义 agent 名**默认不在上下文**，模型需 `{action:"list"}` 或读 skill 才能知道真实 agent 名。
- **佐证强度**：rick 提示词写「派发 subagent」却无 workflowScript 语法、无真实 agent 名（自造 think/research/exporter/subagent_1~6 在 pi 中不存在），与官方唯一触发路径完全脱节 → 「自然语言触发不可靠、显式工具调用才可靠」由官方执行面设计直接佐证。

### C2：「软性自然语言指令不可靠、需显式提示」官方旁证（高置信）

- pi `docs/skills.md` 第 68 行：「When a task matches, the agent uses `read` to load the full SKILL.md (**models don't always do this; use prompting or /skill:name to force it**)」——官方明确：模型不总会遵守软性指令，需显式提示/强制。
- 此旁证支持「软性自然语言触发词（『当 X 时用 subagent』）不如显式语法/agent 名可靠」的因果方向。

---

## R7 上报项（无法达高置信的叶节点）

1. **A1 模型 tool-calling 能力差异的量化证据**——本地无 benchmark 信源，需 WebSearch/实测；本次无 WebSearch 工具，无法达高置信。
2. **（延续 N3.3）「触发概率低」的量化/复现证据**——human 观察，S 阶段只读无法复现真实 pi 会话触发率。
3. **（延续 N3.4）「提示词缺口 ⇒ 触发概率低」的因果归属**——结构性缺口已证实（D1/D2/D6/D7），因果归属属假设，非事实可证。

---

## 整合摘要

分支 A：3 项（A1 R7 / A2 已排除 / A3 归入 D3）｜分支 B：4 项（全高置信）｜分支 C：2 项（全高置信）｜R7 上报 3 项。

**关键结论（事实性，非建议）**：配置层已排除；模型能力差异无法本地验证（R7）；自定义 agent 机制完全可支撑「把 think/research/exporter 注册为系统级 agent」的方向（B1/B2/B3），而 rick 现状是零自定义 agent、think/research/exporter 仅为普通 markdown 文件（B4）。
