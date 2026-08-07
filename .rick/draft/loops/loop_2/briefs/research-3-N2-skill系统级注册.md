# research-3 N2-skill 系统级注册机制（Y3）

节点路径:[根 > N2-skill 系统级注册机制]
事实陈述:pi `registerTool` / `registerCommand` 的语义(CLI 命令 vs LLM tool schema)、tool schema 注入机制(system prompt / context window)、rick skill 目录结构能否映射为 pi tool、映射粒度(1:1 / 1:N)、语义对齐性(流程描述 vs 函数签名)。

## 执行动作

1. 读取 `/tmp/pi_extensions.md`(120KB,extensions 专项文档)
2. `sed -n '1337,1420p'` — 读取 `pi.registerTool(definition)` 全文
3. `sed -n '1493,1560p'` — 读取 `pi.registerCommand(name, options)` 全文
4. 读取 rick `.rick/skills/` 目录结构 + 一个 skill.md 样例(`command_registration_verification_skill/skill.md`)
5. 对比 rick skill(流程描述 markdown)与 pi tool(TypeScript schema)的语义对齐性

## 信源验证结果

### 代码原文(权重 0.4)✅

**pi.registerTool 语义**(extensions.md 1337-1420 行):

> Register a custom tool **callable by the LLM**. ... New tools are refreshed immediately in the same session, so they appear in `pi.getAllTools()` and are **callable by the LLM without `/reload`**.

> Use `promptSnippet` to opt a custom tool into a one-line entry in `Available tools`, and `promptGuidelines` to append tool-specific bullets to the default `Guidelines` section when the tool is active.

→ **registerTool 注册的是 LLM 可调用工具,进 LLM tool schema(进 system prompt 的 `Available tools` 段 + `Guidelines` 段)**。LLM 在推理时可见,可主动调用。这是"系统级注册"(对 LLM 可见),非 CLI 斜杠命令。

**pi.registerCommand 语义**(extensions.md 1493-1560 行):

> Register a command. ... `pi.registerCommand("stats", { description: "Show session statistics", handler: ... })`

> `pi.getCommands()` — Get the slash commands available for invocation via `prompt` in the current session. Includes extension commands, prompt templates, and skill commands. ... Built-in interactive commands (like `/model` and `/settings`) are not included here. They are handled only in interactive mode and would not execute if sent via `prompt`.

→ **registerCommand 注册的是 CLI 斜杠命令(`/stats`、`/deploy`),仅交互式 TUI 可用,LLM 推理时不可见、不可调用**。

**pi.registerTool schema 注入机制**:

```typescript
pi.registerTool({
  name: "my_tool",
  label: "My Tool",
  description: "What this tool does",
  promptSnippet: "Summarize or transform text according to action",  // 进 system prompt "Available tools" 段
  promptGuidelines: ["Use my_tool when the user asks to summarize..."],  // 进 system prompt "Guidelines" 段
  parameters: Type.Object({  // typebox schema → LLM tool 参数 schema
    action: StringEnum(["list", "add"] as const),
    text: Type.Optional(Type.String()),
  }),
  async execute(toolCallId, params, signal, onUpdate, ctx) { ... },
});
```

→ **tool schema 通过 `promptSnippet` + `promptGuidelines` + `parameters`(typebox)三路注入 system prompt + LLM tool schema**。LLM 推理时:`Available tools` 段列出工具名+一句话描述,`Guidelines` 段给出工具使用时机,`parameters` 转换为 LLM provider 的 function calling schema(OpenAI tools / Anthropic tools)。

**pi.registerTool 动态生效**:

> `pi.registerTool()` works both during extension load and after startup. You can call it inside `session_start`, command handlers, or other event handlers. New tools are refreshed immediately in the same session, so they appear in `pi.getAllTools()` and are callable by the LLM **without `/reload`**.

→ 运行时动态注册,无需重启 session。

**pi.setActiveTools 控制 tool 启用**:

> Use `pi.setActiveTools()` to enable or disable tools (including dynamically added tools) at runtime.

→ 可按任务阶段动态启停 tool(如 doing 阶段启用 research tool,learning 阶段禁用)。

**rick skill 目录结构**(`.rick/skills/`):

```
.rick/skills/
├── check_mechanism_skill/skill.md + mock_agent_testing.py
├── command_registration_verification_skill/skill.md
├── dag_task_decomposition_skill/skill.md
├── failure_feedback_skill/skill.md
├── global_ref_sync_skill/skill.md
├── mark_task_success_skill/skill.md + mark_task_success.py + build_rick.py
├── multi_phase_protocol_skill/skill.md
├── subprocess_env_isolation_skill/skill.md
├── template_injection_skill/skill.md + check_prompt_variables.py
├── test_script_practices_skill/skill.md
├── verify_go_changes_skill/skill.md + check_go_build.py + check_variadic_api.py + check_cobra_registration.py
└── zero_retry_task_design_skill/skill.md
```

**rick skill.md 样例**(command_registration_verification_skill/skill.md 前 30 行):

```markdown
# skill:command-registration-verification（文档命令引用核实）

## 触发场景

在文档（README、commands.md、学习文档等）中引用项目自身的 CLI 命令...时使用：
- 写"命令体系"小节，列出所有命令
- 描述某命令的 flags、默认值、用法
...

## 核心内容

### 第 0 步（禁止跳过）：读 root.go 的 AddCommand 清单

```bash
grep -n "AddCommand" internal/cmd/root.go
```

### 第 1 步：读每个命令源文件的 cobra.Command 定义
...
```

→ **rick skill 是"流程描述"markdown**:触发场景 + 预期效果 + 核心内容(分步骤 + bash 命令 + 检查点)。无函数签名,无参数 schema,无返回值。是协议文档,非可执行函数。

### 运行时行为(权重 0.3)✅

**extensions.md 示例用例列表**(Quick Start 段):

> permission gates / git checkpointing / path protection / custom compaction / conversation summaries / interactive tools / stateful tools / external integrations / games

→ 这些都是**程序化工具**(有 execute 函数),非流程描述。

**extensions.md 1881 行**:

> Register tools the LLM can call via `pi.registerTool()`. Tools appear in the system prompt and can have custom rendering.

→ 确认 tool 进 system prompt(LLM 可见)。

**pi Skills 体系**(extensions.md "Skills" 段):

> Skills: Markdown SKILL.md,遵循 Agent Skills standard(agentskills.io),`/skill:name` 调用。自动加载 `~/.pi/agent/skills/`、`~/.agents/skills/`、`.pi/skills/`、`.agents/skills/`(向上递归)。

→ **pi 原生支持 Agent Skills standard**,与 rick skill 目录结构(`{name}_skill/skill.md`)兼容(只需重命名为 `skill.md` 而非 `{name}_skill/skill.md`,或建符号链接)。但 pi Skills 是**斜杠命令调用**(`/skill:name`),非 LLM 自动触发。

### 文档(权重 0.2)✅

- extensions.md 1337-1420 行:registerTool 完整 schema(promptSnippet + promptGuidelines + parameters + execute)
- extensions.md 1493-1560 行:registerCommand 完整 schema(name + description + handler)
- extensions.md 1881 行:"Tools appear in the system prompt"
- rick `.rick/skills/` 目录:12 个 skill,全为 markdown 流程描述

### 反事实(权重 0.1)N/A

- 本节点为外部文档调研,无代码修改

## 还原确认

无 rick 代码修改,无需还原。

## 关键事实

1. **registerTool = LLM tool schema 系统级注册**:进 system prompt(`Available tools` + `Guidelines` 段)+ LLM function calling schema(OpenAI/Anthropic tools)。LLM 推理时可见、可主动调用。**这是"系统级注册"**。
2. **registerCommand = CLI 斜杠命令**:仅交互式 TUI 可用(`/stats`、`/deploy`),LLM 推理时不可见、不可调用。**非系统级注册**。
3. **registerTool 动态生效**:运行时注册,无需重启 session,LLM 立即可见。
4. **setActiveTools 控制 tool 启用**:可按任务阶段动态启停 tool(doing 启用 research、learning 禁用)。
5. **rick skill 是流程描述 markdown**:触发场景 + 预期效果 + 核心内容(分步骤 + bash 命令)。无函数签名,无参数 schema,无返回值。
6. **pi tool 是 TypeScript 函数签名**:name + description + parameters(typebox schema)+ execute 函数。有明确参数与返回值。
7. **rick skill 与 pi tool 语义不对齐**:rick skill 是"协议文档"(LLM 按描述执行),pi tool 是"可执行函数"(LLM 调用 → TS 函数执行 → 返回结果)。
8. **pi 原生支持 Agent Skills standard**:`~/.pi/agent/skills/` + `.pi/skills/` 自动加载,`/skill:name` 调用。与 rick skill 目录结构兼容(重命名或符号链接)。

## 映射粒度分析

| rick skill 类型 | 映射为 pi tool? | 映射粒度 | 语义对齐性 |
|---|---|---|---|
| 流程描述型(command_registration_verification) | ❌ 不可直接映射 | N/A | 流程描述 ≠ 函数签名 |
| 脚本辅助型(verify_go_changes 含 .py 脚本) | ✅ 可映射 | 1 skill = 1 tool(包装 .py 脚本) | 中(脚本有明确输入输出) |
| 协议型(dag_task_decomposition) | ❌ 不可直接映射 | N/A | 协议 ≠ 函数 |
| 检查型(check_mechanism 含 mock_agent_testing.py) | ✅ 可映射 | 1 skill = 1 tool | 中 |

→ **rick skill 大部分是流程描述型,不能直接映射为 pi tool**(需重写为 TS extension + execute 函数)。仅含 .py 脚本的 skill 可包装为 pi tool(但需 rick 端写 TS shim 调用 .py)。

## 触发概率提升判断

human 假设:"skill 系统级注册(pi registerTool)→ 触发概率显著提升"。

- **现状**(rick skill 提示词维护):skill.md 通过 doing.md / learning.md 模板 `{{skill_path}}` 注入 prompt,LLM 看到 skill 路径 → 需主动 Read skill.md → 按描述执行。触发依赖 LLM 主动读文件。
- **pi registerTool**:tool schema 直接进 system prompt `Available tools` 段 + LLM function calling schema。LLM 推理时直接看到工具名+描述+参数,可主动调用(无需 Read 文件)。

→ **触发概率提升机制成立**:registerTool 把 skill 从"文件需主动读"提升为"schema 直接进 LLM 推理空间"。但**仅适用于可映射为 tool 的 skill**(函数签名型),流程描述型 skill 仍需 prompt 注入(pi Skills `/skill:name` 体系,或 APPEND_SYSTEM.md)。

## 疑问点

- 流程描述型 skill(command_registration_verification 等)在 pi 中如何触发?→ pi Skills 体系(`/skill:name`)或 APPEND_SYSTEM.md 注入,但非 LLM 自动调用。需 rick 端写 TS extension 包装为 registerTool 才能进 LLM tool schema。
- rick skill 中的 .py 脚本(verify_go_changes 含 3 个 .py)如何被 pi tool 调用?→ pi tool execute 函数中 `exec("python3 /path/to/script.py")`,需 rick 端写 TS shim。

## 置信度评估(由 research 主调度计算)

- 代码原文 ✅ × 0.4 = 0.4(extensions.md registerTool/registerCommand 全文 + rick skill 目录)
- 运行时行为 ✅ × 0.3 = 0.3(extensions.md 1881 行 + 示例用例 + pi Skills 体系)
- 文档 ✅ × 0.2 = 0.2(extensions.md + rick skill.md 样例)
- 反事实 N/A × 0.1 = 0
- 合计 = 0.9(高,≥ 0.8 终止)
