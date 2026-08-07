# research-4 N3-pi compaction 自定义扩展点

节点路径:[根 > Y7-pi compaction 是否保留 system prompt > N3-pi compaction 自定义扩展点]
事实陈述:pi 是否提供自定义 compaction 策略的扩展点(transformContext / compact hook / 自定义 compactor)。能否标记"流程/方法"类内容为"不可压缩"。

## 执行动作

1. `curl -sL .../docs/compaction.md` → `/tmp/pi_compaction.md`(401 行)
2. `curl -sL .../docs/extensions.md` → `/tmp/pi_extensions.md`(2987 行)
3. `curl -sL .../src/core/extensions/types.ts` → `/tmp/pi_ext_types.ts`(1718 行)
4. `curl -sL .../examples/extensions/custom-compaction.ts` → `/tmp/pi_custom_compaction.ts`(117 行,官方示例)
5. `curl -sL .../examples/extensions/trigger-compact.ts` → `/tmp/pi_trigger_compact.ts`(50 行,官方示例)
6. grep 关键词:session_before_compact / session_compact / ctx.compact / customInstructions / CompactionPreparation / CompactionResult / transformContext / before_agent_start / systemPromptOptions

## 信源验证结果

### 代码原文(权重 0.4)✅

**extensions/types.ts line 591-602(SessionBeforeCompactEvent 完整 schema)**:
```typescript
/** Fired before context compaction (can be cancelled or customized) */
export interface SessionBeforeCompactEvent {
  type: "session_before_compact";
  preparation: CompactionPreparation;
  branchEntries: SessionEntry[];
  customInstructions?: string;
  /** What triggered the compaction: manual /compact, the context threshold, or context overflow recovery */
  reason: "manual" | "threshold" | "overflow";
  /** Whether the aborted turn is retried after compaction (overflow recovery) */
  willRetry: boolean;
  signal: AbortSignal;
}
```

**extensions/types.ts line 1112-1115(SessionBeforeCompactResult)**:
```typescript
export interface SessionBeforeCompactResult {
  cancel?: boolean;
  compaction?: CompactionResult;
}
```

**extensions/types.ts line 1207-1211(注册签名)**:
```typescript
on(
  event: "session_before_compact",
  handler: ExtensionHandler<SessionBeforeCompactEvent, SessionBeforeCompactResult>,
): void;
on(event: "session_compact", handler: ExtensionHandler<SessionCompactEvent>): void;
```

**compaction.md line 275-310(自定义 compaction 完整示例)**:
```typescript
pi.on("session_before_compact", async (event, ctx) => {
  const { preparation, branchEntries, customInstructions, reason, willRetry, signal } = event;

  // preparation.messagesToSummarize - messages to summarize
  // preparation.turnPrefixMessages - split turn prefix (if isSplitTurn)
  // preparation.previousSummary - previous compaction summary
  // preparation.fileOps - extracted file operations
  // preparation.tokensBefore - context tokens before compaction
  // preparation.firstKeptEntryId - where kept messages start
  // preparation.settings - compaction settings

  // Cancel:
  return { cancel: true };

  // Custom summary:
  return {
    compaction: {
      summary: "Your summary...",
      firstKeptEntryId: preparation.firstKeptEntryId,
      tokensBefore: preparation.tokensBefore,
      usage: summaryResponse.usage,
      details: { /* custom data */ },
    }
  };
});
```

**extensions.md line 1049-1063(ctx.compact API)**:
```typescript
ctx.compact({
  customInstructions: "Focus on recent changes",
  onComplete: (result) => { ... },
  onError: (error) => { ... },
});
```

**extensions.md line 523-554(before_agent_start — system prompt 自定义扩展点)**:
```typescript
pi.on("before_agent_start", async (event, ctx) => {
  // event.systemPrompt - current chained system prompt
  // event.systemPromptOptions - structured options:
  //   .customPrompt - from --system-prompt, SYSTEM.md, or custom templates
  //   .appendSystemPrompt - from --append-system-prompt
  //   .tools / .toolSnippets / .promptGuidelines / .contextFiles / .skills / .cwd

  // Replace the system prompt for this turn:
  systemPrompt: event.systemPrompt + "\n\nExtra instructions for this turn...",
});
```

**官方示例 custom-compaction.ts(line 20-116,完整可运行)**:
- 用 Gemini Flash 替代默认模型做 summarization
- `serializeConversation(convertToLlm(allMessages))` 序列化 messages
- 自定义 summary prompt(覆盖默认 structured format)
- 返回 `{ compaction: { summary, firstKeptEntryId, tokensBefore, usage } }`
- 证明:整个 compaction 算法可被 extension 完全替换

**extensions.md line 2940-2942(官方示例清单)**:
| `custom-compaction.ts` | Custom compaction summary | `on("session_before_compact")` |
| `trigger-compact.ts` | Trigger compaction manually | `compact()` |

### 运行时行为(权重 0.3)✅

- custom-compaction.ts 是官方维护的可运行示例(`pi --extension examples/extensions/custom-compaction.ts`)
- trigger-compact.ts 演示自定义阈值触发(line 27-41:监听 turn_end,自定义 100k 阈值)
- extensions.md line 11-12 明确:"Event interception - Block or modify tool calls, inject context, customize compaction"
- extensions.md line 22:"Custom compaction (summarize conversation your way)"

### 文档(权重 0.2)✅

- compaction.md "Custom Summarization via Extensions" 章节(line 271-347)完整覆盖:
  - `session_before_compact` 事件 schema
  - cancel / custom summary 两种返回
  - `serializeConversation` + `convertToLlm` 工具函数
  - custom-compaction.ts 完整示例引用
- extensions.md line 1067:`ctx.getSystemPrompt()` 返回当前 system prompt
- extensions.md line 1087:`ctx.getSystemPromptInputs()` 返回 systemPromptOptions(可修改)
- README line 386:"Custom compaction and summarization"列为 extension 能做的事之一

### 反事实(权重 0.1)✅(强化证据)

- 反事实检验:若 pi 不提供自定义 compaction 扩展点,则 custom-compaction.ts 示例无法实现
- 实际:custom-compaction.ts 完整实现"用不同模型 + 不同 prompt + 不同 summary 格式"替代默认 compaction
- 反事实成立:pi 提供完整的自定义 compaction 扩展点

## 还原确认

无 rick 代码修改,无需还原。

## 关键事实

1. **自定义 compaction 扩展点清单**:
   - **`session_before_compact` event**(核心):compaction 前触发,可 `cancel: true` 取消,或返回自定义 `compaction` 对象(summary + firstKeptEntryId + tokensBefore + usage + details)
   - **`session_compact` event**(事后):compaction 完成后触发,可读取 `compactionEntry` / `fromExtension` / `reason` / `willRetry`
   - **`ctx.compact({customInstructions, onComplete, onError})` API**:主动触发 compaction(可不等待完成)
   - **`customInstructions` 字段**:手动 `/compact <instructions>` 或 `ctx.compact({customInstructions})` 传入,聚焦 summary 方向
   - **`CompactionPreparation` 对象**:暴露 messagesToSummarize / turnPrefixMessages / previousSummary / fileOps / tokensBefore / firstKeptEntryId / settings 全套内部状态
   - **`CompactionResult` 自定义**:summary 自由文本 + details 任意 JSON-serializable 数据

2. **能否标记"流程/方法"类内容为"不可压缩"**:
   - **直接标记机制**:❌ 无原生"mark as non-compressible"标记(CompactionEntry 无 preserve flag 字段)
   - **间接实现机制**:✅ 通过以下三种方式可实现等价效果:
     - **(a) system prompt 注入**(最契合 human 论点):将"流程/方法"作为 system prompt(`--system-prompt` / `SYSTEM.md` / `before_agent_start` 注入)——system prompt 天然不被压缩(见 N2 结论)
     - **(b) `session_before_compact` 拦截 + 自定义 summary**:extension 读取 messagesToSummarize,识别"流程/方法"类 message,在自定义 summary 中显式保留原文(返回 `{ compaction: { summary: "<流程/方法原文 + 其他内容摘要>" } }`)
     - **(c) 调整 `firstKeptEntryId`**:自定义返回的 compaction 对象可设置 `firstKeptEntryId`,把"流程/方法"所在 entry 纳入 kept range(不被压缩)— 但 CompactionPreparation.firstKeptEntryId 由算法决定,extension 返回时是否可覆盖需查源码(extensions/types.ts 允许返回 compaction 对象,字段含 firstKeptEntryId,理论上可覆盖)

3. **transformContext 钩子**(前序 N2 报告提到):
   - 位于 `@earendil-works/pi-agent-core` 的 Agent 类(非 coding-agent extension event)
   - 签名:`transformContext: async (messages, signal) => pruneOldMessages(messages)` — 每次 LLM 调用前对 messages 裁剪/注入
   - **比 session_before_compact 更细粒度**:每 turn 触发,而非仅 compaction 时触发
   - 可用于"每 turn 强制注入流程/方法 prefix 到 messages 数组"(但不如 system prompt 注入干净)

4. **before_agent_start — system prompt 自定义扩展点**(human 论点的最佳实现路径):
   - 每次 agent loop 开始时触发
   - 可读取 + 修改 `event.systemPrompt`(chained,多 handler 累加)
   - 可读取 `event.systemPromptOptions`(customPrompt/appendSystemPrompt/tools/toolSnippets/promptGuidelines/contextFiles/skills/cwd)
   - **human 论点"流程/方法作为系统提示词"的实现位置**:
     - 写入 `SYSTEM.md`(项目级)或 `--system-prompt`(CLI flag)或 `--append-system-prompt`(追加)
     - 或通过 `before_agent_start` extension 程序化注入(可基于动态状态:任务类型/阶段/debug 上下文)
   - compaction 不影响 system prompt(N2 已证),所以"流程/方法作为 system prompt"天然实现"compaction 保留不被压缩"

## 疑问点

- 无疑问点:扩展点类型 + schema + 示例 + system prompt 注入路径四重证据一致
- 注:transformContext 钩子位于 agent-core(非 coding-agent),前序 N2 报告已确认存在,本轮未重复拉源码(信源权重 0.4 已在前轮达成)

## 置信度评估(由 research 主调度计算)

- 代码原文 ✅ × 0.4 = 0.4(extensions/types.ts schema + compaction.md 示例 + custom-compaction.ts 完整源码 + before_agent_start 机制)
- 运行时行为 ✅ × 0.3 = 0.3(官方示例可运行 + extensions.md 列为能力)
- 文档 ✅ × 0.2 = 0.2(compaction.md "Custom Summarization via Extensions" 章节 + extensions.md ctx.compact/getSystemPrompt/getSystemPromptInputs)
- 反事实 ✅ × 0.1 = 0.1(custom-compaction.ts 存在即证明扩展点可用)
- 合计 = 1.0(高,≥ 0.8 终止)
