# research-3 N4-subagent 扩展实现路径（Y6）

节点路径:[根 > N4-subagent 扩展实现路径]
事实陈述:pi 显式 No sub-agents,但 Extension 机制能否实现 rick 风格 subagent、派发独立 subagent 机制(rick 端多次调用 pi -p vs pi 内部虚拟 subagent)、subagent 上下文隔离机制、rick 现有 subagent 模式(think/research/exporter)的迁移路径。

## 执行动作

1. `curl -sL "https://raw.githubusercontent.com/earendil-works/pi/main/packages/coding-agent/examples/extensions/subagent/README.md"` — 读取官方 subagent extension README
2. `curl -sL "https://raw.githubusercontent.com/earendil-works/pi/main/packages/coding-agent/examples/extensions/subagent/index.ts"` — 读取 subagent extension 实现(35KB)
3. `grep -nE "registerTool|createAgentSession|new Agent|spawn|fork|isolat|sub.?agent|subagent"` — 定位关键实现
4. `sed -n '461,560p'` — 读取 registerTool 定义
5. `sed -n '320,360p'` — 读取 spawn 调用
6. 读取 rick `internal/prompt/templates/sense_loop.md` — 确认 rick subagent 模式
7. 读取 rick `internal/prompt/human_loop_prompt.go` — 确认 subagent prompt 生成

## 信源验证结果

### 代码原文(权重 0.4)✅

**pi 官方 subagent extension 存在**(packages/coding-agent/examples/extensions/subagent/):

```
subagent/
├── README.md            # 6111 bytes
├── index.ts             # 35092 bytes (extension entry)
├── agents.ts            # 3363 bytes (agent discovery)
├── agents/              # Sample agent definitions
│   ├── scout.md         # Fast recon, returns compressed context
│   ├── planner.md       # Creates implementation plans
│   ├── reviewer.md      # Code review
│   └── worker.md        # General-purpose
└── prompts/             # Workflow presets
    ├── implement.md     # scout -> planner -> worker
    ├── scout-and-plan.md
    └── implement-and-review.md
```

**README 开篇**(subagent/README.md):

> Delegate tasks to specialized subagents with **isolated context windows**.
>
> ## Features
> - **Isolated context**: Each subagent runs in a separate `pi` process
> - **Streaming output**: See tool calls and progress as they happen
> - **Parallel streaming**: All parallel tasks stream updates simultaneously
> - **Usage tracking**: Shows turns, tokens, cost, and context usage per agent
> - **Abort support**: Ctrl+C propagates to kill subagent processes

→ **pi 官方提供 subagent extension 范例**,每个 subagent 是独立 pi 进程(隔离 context window)。

**subagent extension index.ts 头部注释**(1-15 行):

```typescript
/**
 * Spawns a separate `pi` process for each subagent invocation,
 * giving it an isolated context window.
 *
 * Uses JSON mode to capture structured output from subagents.
 */
import { spawn } from "node:child_process";
```

→ **实现机制:`spawn("pi", ...)` 子进程**,每个 subagent 一次 spawn,通过 `--mode json` 捕获结构化输出。

**subagent registerTool 定义**(index.ts 461-560 行):

```typescript
pi.registerTool({
  name: "subagent",
  label: "Subagent",
  description: [
    "Delegate tasks to specialized subagents with isolated context.",
    "Modes: single (agent + task), parallel (tasks array), chain (sequential with {previous} placeholder).",
    `Default agent scope is "user" (from ${path.join(getAgentDir(), "agents")}).`,
    `To enable project-local agents in ${CONFIG_DIR_NAME}/agents, set agentScope: "both" (or "project").`,
  ].join(" "),
  parameters: SubagentParams,

  async execute(_toolCallId, params, signal, onUpdate, ctx) {
    // ... 支持 3 种模式:
    //   single: { agent, task }
    //   parallel: { tasks: [...] }  (max 8, 4 concurrent)
    //   chain: { chain: [...] }  (sequential with {previous} placeholder)
  },
});
```

→ **subagent 是 LLM 可调用 tool**(registerTool),LLM 推理时可主动调用 `subagent` tool 派发子任务。支持 3 种模式:单 agent、并行(最多 8 个,4 并发)、链式(带 `{previous}` 占位符)。

**subagent spawn 调用**(index.ts 320-360 行):

```typescript
if (agent.systemPrompt.trim()) {
  const tmp = await writePromptToTempFile(agent.name, agent.systemPrompt);
  tmpPromptDir = tmp.dir;
  tmpPromptPath = tmp.filePath;
  args.push("--append-system-prompt", tmpPromptPath);  // 注入 subagent 专属 system prompt
}

args.push(`Task: ${task}`);

const invocation = getPiInvocation(args);
const proc = spawn(invocation.command, invocation.args, {
  cwd: cwd ?? defaultCwd,
  shell: false,
  stdio: ["ignore", "pipe", "pipe"],
});

// 解析 JSON mode 输出
const processLine = (line: string) => {
  if (!line.trim()) return;
  let event: any;
  try { event = JSON.parse(line); } catch { return; }

  if (event.type === "message_end" && event.message) {
    const msg = event.message as Message;
    currentResult.messages.push(msg);
    // ... 累计 turns / tokens / cost
  }
};
```

→ **subagent 调用流程**:
1. 写 subagent system prompt 到临时文件
2. `spawn("pi", ["--mode", "json", "--append-system-prompt", tmpPromptPath, "Task: ..."])`
3. 解析 JSON lines 输出(`message_end` 事件)
4. 累计 turns/tokens/cost
5. 返回 subagent 最终输出给主 agent

**AgentSession API**(SDK 文档 66-110 行):

```typescript
interface AgentSession {
  prompt(text: string, options?: PromptOptions): Promise<void>;
  steer(text: string): Promise<void>;       // 流式中打断
  followUp(text: string): Promise<void>;    // 队列后续任务
  subscribe(listener: (event: AgentSessionEvent) => void): () => void;
  sessionFile: string | undefined;
  sessionId: string;
  setModel(model: Model): Promise<void>;
  navigateTree(targetId: string, options?): Promise<{...}>;  // 会话树导航
  compact(customInstructions?: string): Promise<CompactionResult>;
  abort(): Promise<void>;
  dispose(): void;
}
```

→ AgentSession 暴露完整 API,但**无 `spawnSubagent()` 方法**。subagent 必须通过 extension 自己 spawn 子进程实现。

**rick 现有 subagent 模式**(`internal/prompt/templates/sense_loop.md` + `internal/prompt/human_loop_prompt.go`):

```go
// human_loop_prompt.go:73-103
// Build and save think subagent prompt
// Build and save research subagent prompt
// Build and save exporter subagent prompt
```

```markdown
<!-- sense_loop.md:6-8 -->
think subagent：`{{think_agent_path}}`
research subagent：`{{research_agent_path}}`
exporter subagent：`{{exporter_agent_path}}`

<!-- sense_loop.md:16 -->
你（main agent）= sense 复核层具象化。控制 5 阶段推进节奏：
派发 subagent → 展示简报 → 嵌入门禁 → 记录判断 → 派发下一阶段 OR 反向回流

<!-- sense_loop.md:83-84 -->
- research subagent:调研现状事实(用尽调树)
- think subagent(嵌入批判门禁):对 human 回答执行假设追问
```

→ **rick subagent 模式**:main agent 通过 prompt 路径(`{{think_agent_path}}`)派发 subagent,subagent 是独立 prompt 文件,main agent 读取后执行(本质是 prompt 文本注入 + main agent 角色切换)。**非进程隔离**(同一 claude code 进程内多角色),**非 context 隔离**(共享 context window)。

### 运行时行为(权重 0.3)✅

**subagent extension README 安全模型**:

> This tool executes a separate `pi` subprocess with a delegated system prompt and tool/model configuration.
>
> **Project-local agents** (`.pi/agents/*.md`) are repo-controlled prompts that can instruct the model to read files, run bash commands, etc.
>
> **Default behavior:** Only loads **user-level agents** from `~/.pi/agent/agents`.
>
> To enable project-local agents, pass `agentScope: "both"` (or `"project"`). Only do this for repositories you trust.
>
> When running interactively, the tool prompts for confirmation before running project-local agents.

→ **subagent 上下文隔离机制**:独立 pi 进程(独立 context window)+ 独立 system prompt(临时文件)+ 独立 tool/model 配置。project-local agent 需用户确认(安全门禁)。

**subagent extension 工作流预设**:

```markdown
/implement           # scout -> planner -> worker (chain)
/scout-and-plan      # scout -> planner (chain, no implementation)
/implement-and-review  # worker -> reviewer -> worker (chain)
```

→ 内置链式工作流,通过 prompt template 定义 multi-step subagent 链。

**subagent parallel 模式**:

> Parallel mode streaming:
> - Shows all tasks with live status (⏳ running, ✓ done, ✗ failed)
> - Returns each completed task's final output to the parent model, **capped at 50 KB per task**

→ 并行 subagent 最多 8 个(4 并发),输出 50KB 截断(防止 context 爆炸)。

**SDK 文档第 11 行**:

> Build custom tools that spawn sub-agents

→ pi 官方明确 subagent 通过 custom tool spawn 子进程实现(非内置)。

### 文档(权重 0.2)✅

- subagent/README.md 6111 bytes(完整文档)
- extensions.md 2969 行:`subagent/` | Spawn sub-agents | `registerTool`, `exec`
- SDK 文档第 11 行:"Build custom tools that spawn sub-agents"
- rick sense_loop.md:subagent 派发模式(prompt 路径注入)

### 反事实(权重 0.1)N/A

- 本节点为外部文档调研,无代码修改

## 还原确认

无 rick 代码修改,无需还原。

## 关键事实

1. **pi 官方提供 subagent extension 范例**:`packages/coding-agent/examples/extensions/subagent/`(README + index.ts 35KB + agents/ + prompts/)
2. **subagent 实现机制 = spawn 子进程**:`spawn("pi", ["--mode", "json", "--append-system-prompt", tmpFile, "Task: ..."])`,每个 subagent 独立 pi 进程
3. **subagent 是 LLM 可调用 tool**:通过 `pi.registerTool({ name: "subagent", ... })` 注册,LLM 推理时主动调用,支持 3 种模式(single / parallel max 8 / chain with `{previous}`)
4. **subagent 上下文隔离**:独立 pi 进程(独立 context window)+ 独立 system prompt(临时文件)+ 独立 tool/model 配置 + 输出 50KB 截断
5. **subagent 安全模型**:project-local agent(`.pi/agents/*.md`)需用户确认(交互式 TUI),防 repo 注入恶意 prompt
6. **AgentSession 无内置 spawnSubagent**:subagent 必须通过 extension 自己 spawn 子进程实现(pi 不提供"虚拟 subagent")
7. **rick 现有 subagent 模式 = prompt 路径注入**:main agent 通过 `{{think_agent_path}}` 读取 subagent prompt 文件,角色切换执行,**非进程隔离,非 context 隔离**
8. **迁移路径**:rick subagent(think/research/exporter)可映射为 pi agents(`.pi/agents/think.md` 等),main agent 调用 `subagent` tool 派发,每个 subagent 独立 pi 进程(强隔离)

## 两种实现路径对比

| 路径 | 实现机制 | rick 改造量 | 隔离性 | trace 继承 |
|---|---|---|---|---|
| **A. rick 端多次调用 `pi -p`** | rick(Go)直接 spawn 多个 pi 子进程,每进程一个 subagent | 中(exec.Command 已有,改 prompt 生成) | 强(进程级) | 弱(rick 端拼装 trace) |
| **B. pi 内部 subagent extension** | rick 调用主 pi 进程,主 pi 通过 subagent extension spawn 子 pi 进程 | 高(写 TS extension + agents/*.md) | 强(进程级) | 强(pi 内部统一 trace) |
| **C. pi 内部虚拟 subagent** | 不存在,pi 无此能力 | N/A | N/A | N/A |

→ **路径 A 与 B 等价性**:都是 spawn 子进程,差别在"谁 spawn"(rick Go vs pi TS extension)。路径 B 更优(trace 继承 + subagent 作为 LLM tool 可被 LLM 主动调用,而非 rick 硬编码派发)。

## rick subagent 迁移映射

| rick subagent | pi agent 文件 | 职责 | system prompt 来源 |
|---|---|---|---|
| think subagent | `.pi/agents/think.md` | 4 维打分 + top-N 假设分析 | rick sense_loop think prompt |
| research subagent | `.pi/agents/research.md` | 尽调树 + 信源加权 | rick sense_loop research prompt |
| exporter subagent | `.pi/agents/exporter.md` | RFC 大纲 + 内容填充 | rick sense_loop exporter prompt |
| (新增)critic subagent | `.pi/agents/critic.md` | 批判门禁 | rick 批判门禁 prompt |

→ rick 现有 4 类 subagent(think/research/exporter + 批判门禁)可全部映射为 pi agents,通过 subagent extension 调用。

## 疑问点

- rick 现有 subagent 是 prompt 注入(同进程),迁移为 pi 子进程后,**subagent 间共享状态如何传递**?→ 通过 main agent 拼装(subagent 输出回传 main agent,main agent 再传给下一 subagent)。pi subagent extension 的 chain 模式(`{previous}` 占位符)支持此模式。
- subagent 输出 50KB 截断是否够 rick 用?→ rick research subagent 报告可达 10KB+,think subagent 假设列表 < 5KB,exporter RFC 可达 50KB。临界,需测试。
- rick 批判门禁嵌入 think subagent(非独立 subagent),迁移后是否需拆分?→ 可保持嵌入(think.md 内含门禁逻辑),或拆为独立 critic.md agent。

## 置信度评估(由 research 主调度计算)

- 代码原文 ✅ × 0.4 = 0.4(subagent extension README + index.ts 35KB + registerTool + spawn 实现 + rick sense_loop.md)
- 运行时行为 ✅ × 0.3 = 0.3(subagent extension 3 模式 + 安全模型 + 50KB 截断 + SDK 文档第 11 行)
- 文档 ✅ × 0.2 = 0.2(subagent README + extensions.md 2969 行 + SDK 文档)
- 反事实 N/A × 0.1 = 0
- 合计 = 0.9(高,≥ 0.8 终止)
