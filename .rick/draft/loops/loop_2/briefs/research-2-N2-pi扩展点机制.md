# research-2 N2-pi 扩展点机制

节点路径:[根 > N2-pi 扩展点机制]
事实陈述:pi 提供多种扩展点类型与粒度,需对比 claude code 的 `--append-system-prompt` + MCP,判断 pi 扩展粒度是否更细。

## 执行动作

1. 读取 `/tmp/pi_coding_agent_readme.md`(32KB,完整 README)
2. 读取 `/tmp/pi_extensions.md`(120KB,extensions 专项文档)
3. 读取 `/tmp/pi_agent_readme.md`(pi-agent-core README,17KB)
4. 读取 `/tmp/pi_sdk.md`(SDK 文档,36KB)
5. grep extensions.md 关键能力(registerTool / registerCommand / on event / transformContext / convertToLlm / beforeToolCall / afterToolCall)

## 信源验证结果

### 代码原文(权重 0.4)✅

pi 提供 **6 类扩展点**(粒度从粗到细):

1. **Prompt Templates**(粗粒度):Markdown 文件,`/name` 展开,支持 `{{var}}` 占位符。放置 `~/.pi/agent/prompts/` 或 `.pi/prompts/`。
   - 等同 claude code `--append-system-prompt` 的 prompt 注入能力

2. **Skills**(中粒度):Markdown SKILL.md,遵循 Agent Skills standard(agentskills.io),`/skill:name` 调用。自动加载 `~/.pi/agent/skills/`、`~/.agents/skills/`、`.pi/skills/`、`.agents/skills/`(向上递归)。
   - 等同 rick 的 skill 体系(目录结构一致)

3. **Extensions**(细粒度,核心):TypeScript 模块,通过 `ExtensionAPI` 注册:
   - `pi.registerTool({...})` — 自定义 LLM 可调用工具(含参数 schema via typebox)
   - `pi.registerCommand("name", {...})` — 自定义 `/command`
   - `pi.registerShortcut("ctrl+x", {...})` — 键盘快捷键
   - `pi.registerFlag("my-flag", {...})` — CLI flag
   - `pi.registerProvider("name", {...})` — 自定义 LLM provider(支持 OpenAI/Anthropic/Google API 兼容)
   - `pi.on(eventName, handler)` — 事件订阅(生命周期/资源/会话/agent/model/tool 共 6 类事件)
   - `pi.appendEntry(...)` — 会话持久化自定义条目
   - `ctx.ui.{notify,confirm,input,select,custom,setStatus,setWidget}` — UI 交互
   - 通过 jiti 加载,无需编译

4. **Themes**(UI 粒度):热重载主题文件

5. **Pi Packages**(分发粒度):npm/git 分发 extensions+skills+prompts+themes 包,`pi install npm:@foo/bar`

6. **Agent Core 钩子**(最细粒度,程序化):`@earendil-works/pi-agent-core` 的 `Agent` 类暴露:
   - `transformContext: async (messages, signal) => pruneOldMessages(messages)` — 上下文裁剪/注入(每次 LLM 调用前)
   - `convertToLlm: (messages) => ...` — 自定义消息→LLM 格式转换
   - `shouldStopAfterTurn: async ({message, toolResults, context, newMessages}) => boolean` — turn 后停止判定
   - `beforeToolCall` hook — 工具调用前拦截(可 block)
   - `afterToolCall` hook — 工具调用后处理
   - `AgentTool.executionMode: "parallel" | "sequential"` — 每工具执行模式
   - `AgentTool.execute` 返回 `terminate: true` 可终止 loop

**事件清单**(extensions.md):
- Lifecycle:`session_start` / `session_shutdown` / `project_trust`
- Resource:`resources_discover`
- Session:`session_before_switch` / `session_before_fork`
- Agent:`agent_start` / `agent_end` / `turn_start` / `turn_end` / `message_start` / `message_update` / `message_end`
- Model:`model_changed`
- Tool:`tool_call`(可 block)/ `tool_execution_start` / `tool_execution_update` / `tool_execution_end`

### 运行时行为(权重 0.3)✅

- README 明确对比 claude code:"Make pi look like Claude Code"(列为 extension 能做到的事之一)
- README "Philosophy" 段:pi 显式选择不内置 MCP/sub-agents/permission/plan-mode/todos/background-bash,全部交由 extension——**这是设计哲学上的扩展性宣言**
- extensions.md Quick Start 示例:`pi.on("tool_call", ...)` 可在 bash 工具执行前拦截 `rm -rf`,block 工具调用
- extensions.md 示例用例列表:permission gates / git checkpointing / path protection / custom compaction / conversation summaries / interactive tools / stateful tools / external integrations / games

### 文档(权重 0.2)✅

- README Customization 章节:Prompt Templates / Skills / Extensions / Themes / Pi Packages 各有专节
- extensions.md 120KB,涵盖 Quick Start / Locations / Imports / Events / ExtensionContext / ExtensionAPI / State / Custom Tools / Dynamic Tool Loading / Custom UI / Error Handling / Mode Behavior / Examples
- agent README:transformContext / convertToLlm / shouldStopAfterTurn / beforeToolCall / afterToolCall 程序化钩子
- SDK 文档:AgentSession 暴露 prompt/steer/followUp/subscribe/setModel/compact/navigateTree/abort/dispose 全套 API

### 反事实(权重 0.1)N/A

- 本节点为外部文档调研,无代码修改

## 还原确认

无 rick 代码修改,无需还原。

## 关键事实

1. **扩展点类型**:6 类(Prompt Templates / Skills / Extensions / Themes / Pi Packages / Agent Core 钩子)
2. **粒度对比 claude code**:
   - claude code:`--append-system-prompt`(文本注入)+ MCP(外部工具协议)+ `--system-prompt`(替换)
   - pi:上述文本注入能力全有(prompt templates + SYSTEM.md/APPEND_SYSTEM.md)+ **程序化钩子**(transformContext/convertToLlm/beforeToolCall/afterToolCall/shouldStopAfterTurn)+ **事件系统**(6 类生命周期事件,可 block)+ **UI 替换**(editor/widget/status/footer/overlay)+ **provider 自定义**(可接任意 LLM API)
3. **关键差异**:pi 的 extension 是 **TypeScript 代码注入**,可访问 AgentState、拦截 tool_call、替换 convertToLlm、自定义 compaction——粒度显著细于 claude code 的 flag+MCP
4. **MCP 立场**:pi 显式 "No MCP",但 extension 可添加 MCP 支持(README 列为能做的事)
5. **加载机制**:jiti 运行时加载 TS,无需编译;auto-discovery + 显式 `-e` flag + pi package 安装

## 疑问点

- pi extension 是否能完全模拟 claude code 的 `--session-id`/`--resume` 行为?→ 见 N4 调研
- transformContext/convertToLlm 钩子在 RPC 模式下是否可用?→ 需 N3/N4 确认(SDK 文档显示 AgentSession 暴露 agent 实例,理论上可用)

## 置信度评估(由 research 主调度计算)

- 代码原文 ✅ × 0.4 = 0.4
- 运行时行为 ✅ × 0.3 = 0.3
- 文档 ✅ × 0.2 = 0.2
- 反事实 N/A × 0.1 = 0
- 合计 = 0.9(高,≥ 0.8 终止)
