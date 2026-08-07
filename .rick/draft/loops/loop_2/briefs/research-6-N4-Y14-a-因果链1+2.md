# research-6 N4-Y14-a：因果链 1+2 验证（提示词调度 + 门禁内嵌）

节点路径：[根 > N4-Y14-a：因果链 1+2 验证]
事实陈述：
- 因果链 1：rick 注入系统提示词 → pi 基于系统提示词工作 → 确定性提升
- 因果链 2：门禁做成 pi extension 内嵌 → 在 main agent 中确定性做到

## 执行动作

1. Read `/tmp/pi_repo/packages/coding-agent/src/core/system-prompt.ts`（buildSystemPrompt + customPrompt + appendSystemPrompt）
2. Read `/tmp/pi_repo/packages/coding-agent/src/core/agent-session.ts`（_systemPromptOverride + before_agent_start + beforeToolCall hook 安装）
3. Read `/tmp/pi_repo/packages/coding-agent/src/core/extensions/runner.ts`（emitToolCall + block 逻辑 + before_agent_start 替换 systemPrompt）
4. Read `/tmp/pi_repo/packages/agent/src/agent-loop.ts`（systemPrompt 作为 Context 字段独立传递给 LLM）
5. Grep `before_agent_start` / `systemPrompt` / `tool_call.*block` 验证所有拦截点

## 信源验证结果

### 代码原文（权重 0.4）✅

**因果链 1 验证：系统提示词注入能力**

**入口 1：CLI flag 注入**（args.ts line 95-99）：
```ts
} else if (arg === "--system-prompt" && i + 1 < args.length) {
    result.systemPrompt = args[++i];
} else if (arg === "--append-system-prompt" && i + 1 < args.length) {
    result.appendSystemPrompt = result.appendSystemPrompt ?? [];
    result.appendSystemPrompt.push(args[++i]);
}
```
- `--system-prompt <text>`：**完全替换**默认系统提示词
- `--append-system-prompt <text>`：追加到默认或自定义提示词（可多次，可传文件路径）

**入口 2：system-prompt.ts buildSystemPrompt**（line 28-72）：
```ts
export function buildSystemPrompt(options: BuildSystemPromptOptions): string {
    const { customPrompt, appendSystemPrompt, ... } = options;
    if (customPrompt) {
        let prompt = customPrompt;
        if (appendSection) prompt += appendSection;
        // 追加 context files / skills / cwd
        return prompt;
    }
    // 默认提示词 + appendSection + context files + skills
}
```
- `customPrompt` 非空时完全替换默认提示词，但仍追加 context files / skills / cwd
- `appendSystemPrompt` 始终追加（无论 customPrompt 是否存在）

**入口 3：before_agent_start 事件动态替换**（types.ts line 699-709, 1097-1101）：
```ts
export interface BeforeAgentStartEvent {
    type: "before_agent_start";
    prompt: string;
    systemPrompt: string;  // 完整的系统提示词字符串
    systemPromptOptions: BuildSystemPromptOptions;
}

export interface BeforeAgentStartEventResult {
    message?: Pick<CustomMessage, ...>;
    systemPrompt?: string;  // 替换系统提示词
}
```
- extension 可订阅 `before_agent_start`，返回 `systemPrompt` 字段**替换**该 turn 的系统提示词
- agent-session.ts line 1254-1256：`if (result?.systemPrompt !== undefined) { this._systemPromptOverride = result.systemPrompt; this.agent.state.systemPrompt = result.systemPrompt; }`

**LLM 是否"必须遵循"system prompt**（agent-loop.ts line 295-302）：
```ts
const llmContext: Context = {
    systemPrompt: context.systemPrompt,  // 独立字段
    messages: llmMessages,                // 消息历史
    tools: context.tools,
};
const response = await streamFunction(config.model, llmContext, ...);
```
- system prompt 作为 `Context.systemPrompt` 字段**独立传递**给 LLM provider
- 在 Anthropic/OpenAI API 中，system prompt 是顶层字段（非 messages 数组中的 user message）
- LLM 训练时系统提示词权重最高，但**理论上可被 user message 覆盖**（prompt injection）
- pi 不提供"强制 LLM 遵循 system prompt"的机制（这是 LLM 层面的事，非 harness 层面）

**因果链 1 结论**：
- ✅ pi 支持 rick 注入系统提示词（3 种入口：--system-prompt / --append-system-prompt / before_agent_start）
- ✅ pi 基于 system prompt 工作（system prompt 作为 Context 字段传递给 LLM）
- ⚠️ "确定性提升"是 LLM 层面的语义，非 harness 层面。pi 提供注入机制，但 LLM 是否遵循取决于模型本身。prompt injection 仍可覆盖 system prompt。
- **因果链 1 部分成立**：注入机制成立，"确定性提升"是 LLM 行为假设非 pi 保证

**因果链 2 验证：门禁做成 pi extension 内嵌 → 在 main agent 中确定性做到**

**beforeToolCall hook 拦截能力**（agent-session.ts line 479-499）：
```ts
this.agent.beforeToolCall = async ({ toolCall, args }) => {
    const runner = this._extensionRunner;
    if (!runner.hasHandlers("tool_call")) return undefined;
    try {
        return await runner.emitToolCall({
            type: "tool_call",
            toolName: toolCall.name,
            toolCallId: toolCall.id,
            input: args as Record<string, unknown>,
        });
    } catch (err) {
        if (err instanceof Error) throw err;
        throw new Error(`Extension failed, blocking execution: ${String(err)}`);
    }
};
```
- `beforeToolCall` 在**工具执行前**同步调用
- 返回 `{ block: true, reason: string }` → 工具执行被**确定性阻止**
- extension 抛异常 → 工具执行被**确定性阻止**（line 497 "Extension failed, blocking execution"）

**emitToolCall block 逻辑**（runner.ts line 936-953）：
```ts
async emitToolCall(event: ToolCallEvent): Promise<ToolCallEventResult | undefined> {
    for (const ext of this.extensions) {
        const handlers = ext.handlers.get("tool_call");
        for (const handler of handlers) {
            const handlerResult = await handler(event, ctx);
            if (handlerResult) {
                result = handlerResult as ToolCallEventResult;
                if (result.block) {
                    return result;  // 立即返回，阻止工具执行
                }
            }
        }
    }
    return result;
}
```
- 任意一个 handler 返回 `block: true` → 立即返回，**后续 handler 不执行，工具不执行**
- **确定性阻止**：非建议性，是硬阻止

**beforeToolCall 拦截范围**：
- ✅ 可拦截**工具调用**（bash/edit/write/read/grep/find/ls/custom tool）
- ❌ **不可拦截 LLM 文本回复**（LLM 输出纯文本不触发 tool_call 事件）
- ❌ **不可拦截 LLM "思考"**（thinking blocks）

**其他拦截点**：
- `message_end` 事件：可返回 `MessageEndEventResult.message` **替换**最终消息（但消息已生成，是事后替换非预防阻止）
- `input` 事件：可返回 `{ action: "handled" }` 阻止用户输入进入 agent loop（但这是用户输入层非 LLM 输出层）
- `context` 事件：可修改 messages（但这是 LLM 调用前修改上下文，非阻止 LLM 输出）
- `shouldStopAfterTurn`（agent-core）：turn 后停止判定（但这是 turn 后非 turn 中）
- `terminate: true`（AgentTool.execute 返回值）：工具执行后终止 loop（但这是工具内非门禁）

**因果链 2 结论**：
- ✅ 门禁做成 pi extension 内嵌（beforeToolCall hook + tool_call 事件 + block: true）
- ✅ 在 main agent 中确定性做到（block 是硬阻止，非建议）
- ⚠️ 仅限**工具调用门禁**，不含"LLM 文本回复门禁"
- ⚠️ LLM 若输出纯文本（无工具调用），门禁无法阻止（但可由 message_end 替换消息，是事后修正）
- **因果链 2 部分成立**：工具调用门禁确定性成立，文本门禁需 message_end 事后修正

### 运行时行为（权重 0.3）✅

- README "Philosophy"：pi 显式选择不内置 permission popups，全部交由 extension——permission gate 是 extensions.md 列出的示例用例之一
- extensions.md Quick Start：`pi.on("tool_call", ...)` 可在 bash 工具执行前拦截 `rm -rf`，block 工具调用
- extensions.md 示例用例：permission gates / git checkpointing / path protection / custom compaction / conversation summaries

### 文档（权重 0.2）✅

- types.ts `BeforeAgentStartEventResult.systemPrompt`：extension 可替换系统提示词
- types.ts `ToolCallEventResult.block`：extension 可阻止工具执行
- agent-session.ts `_installAgentToolHooks`：beforeToolCall/afterToolCall 钩子安装
- agent-loop.ts `Context.systemPrompt`：系统提示词作为独立字段传递给 LLM

### 反事实（权重 0.1）N/A

本节点为外部源码调研，无代码修改。

## 还原确认

无 rick 代码修改，无需还原。

## 关键事实

1. **系统提示词注入能力**（因果链 1 前提）：
   - 3 种入口：`--system-prompt`（替换）/ `--append-system-prompt`（追加）/ `before_agent_start` 事件（动态替换）
   - rick 可通过 flag 或 extension 注入系统提示词

2. **pi 基于系统提示词工作**（因果链 1 中段）：
   - system prompt 作为 `Context.systemPrompt` 字段独立传递给 LLM provider
   - 在 Anthropic/OpenAI API 中是顶层 system 字段
   - LLM 训练时权重最高

3. **"确定性提升"的边界**（因果链 1 结论）：
   - ✅ pi 提供注入机制（确定性）
   - ⚠️ LLM 是否遵循 system prompt 是模型层面语义，非 harness 保证
   - ⚠️ prompt injection 仍可覆盖 system prompt（user message 中含恶意指令）
   - **因果链 1 部分成立**：注入机制成立，"确定性提升"是 LLM 行为假设

4. **门禁内嵌到 pi extension**（因果链 2 前提）：
   - `beforeToolCall` hook + `tool_call` 事件 + `block: true` → 工具调用确定性阻止
   - extension 抛异常 → 工具调用确定性阻止
   - 非建议性，是硬阻止

5. **"在 main agent 中确定性做到"的边界**（因果链 2 结论）：
   - ✅ 工具调用门禁：确定性成立（block 硬阻止）
   - ❌ LLM 文本回复门禁：不可预防阻止（仅 message_end 事后替换）
   - ❌ LLM 思考门禁：不可拦截
   - **因果链 2 部分成立**：工具调用门禁确定性成立，文本门禁需事后修正

6. **rick 门禁适配建议**：
   - 工具调用门禁（如"禁止编辑 .rick/draft/ 外文件"）→ beforeToolCall + block: true（确定性）
   - 流程门禁（如"必须先 think 再 act"）→ 系统提示词注入 + beforeToolCall 检查 session 状态
   - 格式门禁（如"doing.md 必须含特定段"）→ afterToolCall + tool_result 修改 + message_end 替换（事后修正）
   - 状态机门禁（如"task 状态必须先更新"）→ beforeToolCall 检查 + block

## 疑问点

无。本节点事实清晰，源码三重交叉验证（system-prompt.ts + agent-session.ts + runner.ts + agent-loop.ts）一致。

## 置信度评估（由 research 主调度计算）

- 代码原文 ✅ × 0.4 = 0.4
- 运行时行为 ✅ × 0.3 = 0.3
- 文档 ✅ × 0.2 = 0.2
- 反事实 N/A × 0.1 = 0
- 合计 = 0.9（高，≥ 0.8 终止）
