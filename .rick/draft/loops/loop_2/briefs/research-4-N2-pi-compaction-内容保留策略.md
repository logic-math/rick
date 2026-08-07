# research-4 N2-pi compaction 内容保留策略

节点路径:[根 > Y7-pi compaction 是否保留 system prompt > N2-pi compaction 内容保留策略]
事实陈述:pi compaction 时哪些内容被压缩、哪些被保留,**重点确认 system prompt 是否被保留**。compaction 后的 context 结构(summary 替换 vs 关键 message 保留)。

## 执行动作

1. `curl -sL .../docs/compaction.md` → `/tmp/pi_compaction.md`(401 行,专项文档)
2. `curl -sL .../src/core/compaction/compaction.ts` → `/tmp/pi_compaction_ts.ts`(969 行,源码)
3. `curl -sL .../src/core/session-manager.ts` → `/tmp/pi_session_manager.ts`(1714 行,buildContextEntries 实现)
4. `curl -sL .../docs/extensions.md` → `/tmp/pi_extensions.md`(2987 行,system prompt 组装)
5. grep 关键词:system prompt / systemPrompt / buildContextEntries / firstKeptEntryId / messagesToSummarize / kept messages / What the LLM sees

## 信源验证结果

### 代码原文(权重 0.4)✅(决定性证据)

**compaction.md line 70-77(原文图示——"What the LLM sees")**:
```
What the LLM sees:

  ┌────────┬─────────┬─────┬─────┬──────┬──────┬─────┬──────┐
  │ system │ summary │ usr │ ass │ tool │ tool │ ass │ tool │
  └────────┴─────────┴─────┴─────┴──────┴──────┴─────┴──────┘
       ↑         ↑      └─────────────────┬────────────────┘
    prompt   from cmp          messages from firstKeptEntryId
```

**解读(直接证据)**:
- `system` = system prompt(独立列,标注 "prompt")
- `summary` = compaction 生成的摘要(从 cmp entry 来)
- `usr/ass/tool/...` = 从 `firstKeptEntryId` 开始的最近 messages
- **system prompt 独立于 summary,不被压缩**——它是每次 agent loop 开始时重建的固定 prefix

**compaction.md line 41-45(算法步骤)**:
> 1. **Find cut point**: Walk backwards from newest message, accumulating token estimates until `keepRecentTokens` (default 20k) is reached
> 2. **Extract messages**: Collect messages from the previous kept boundary (or session start) up to the cut point
> 3. **Generate summary**: Call LLM to summarize with structured format, passing the previous summary as iterative context when present
> 4. **Append entry**: Save `CompactionEntry` with summary and `firstKeptEntryId`
> 5. **Reload**: Session reloads, using summary + messages from `firstKeptEntryId` onwards

**关键**:算法步骤 1-5 全程操作"messages"(session entries),**未涉及 system prompt**。system prompt 不在 session entries 中(它是 Agent 类的属性,由 systemPromptOptions 构建)。

**session-manager.ts line 411-456(buildContextEntries 实现)**:
```typescript
// line 412-416 注释:
// Build the active, compaction-aware session entry list.
// This follows the current leaf path. If the path contains compaction entries,
// the latest compaction is represented by the compaction entry itself, followed
// by the kept entries starting at firstKeptEntryId and all entries after the
// compaction entry. Older summarized entries are omitted.

// line 418-452:返回 [compactionEntry, ...keptEntries]
//   - compactionEntry 转成 summary message 注入 context
//   - 不包含 system prompt(system prompt 不在 SessionEntry 类型中)
```

**session-manager.ts line 69-76(CompactionEntry 结构)**:
```typescript
export interface CompactionEntry<T = unknown> extends SessionEntryBase {
  type: "compaction";
  id: string;
  parentId: string;
  timestamp: number;
  summary: string;
  firstKeptEntryId: string;
  tokensBefore: number;
  usage?: Usage;
  fromHook?: boolean;
  details?: T;
}
```
CompactionEntry 字段中**无 system prompt 字段**——证明 system prompt 不是 compaction 操作的对象。

**extensions.md line 523-554(before_agent_start event,system prompt 重建机制)**:
> Fired after user submits prompt, before agent loop. Can inject a message and/or modify the system prompt.
> // event.systemPrompt - current chained system prompt for this handler
> // event.systemPromptOptions - structured options used to build the system prompt
> //   .customPrompt - any custom system prompt (from --system-prompt, SYSTEM.md, or custom templates)
> //   .appendSystemPrompt - text from --append-system-prompt flags

**关键**:`before_agent_start` 在每次 agent loop 开始时触发,**重建 system prompt**(基于 systemPromptOptions:customPrompt + appendSystemPrompt + tools + guidelines + context files + skills)。compaction 后下一次 agent loop 仍然走 `before_agent_start`,system prompt 被完整重建——**与 compaction 无关,天然不被压缩**。

### 运行时行为(权重 0.3)✅

- compaction.md 图示明确 LLM 看到的 context = `[system, summary, ...kept_messages]`
- compaction 算法(line 41-45)只操作 messages,不涉及 system prompt
- buildContextEntries 源码(line 418-452)返回值不含 system prompt
- before_agent_start 每次 agent loop 重建 system prompt(compaction 后仍触发)
- 官方示例 custom-compaction.ts(line 20-116)自定义 compaction 时也只处理 messagesToSummarize + turnPrefixMessages,**不涉及 system prompt**

### 文档(权重 0.2)✅

- compaction.md "How It Works" + "What the LLM sees" 双重图示
- extensions.md "before_agent_start" 章节:system prompt 重建机制
- README line 274:"Compaction summarizes older messages while keeping recent ones"——明确只 summarize messages
- session-format.md:SessionEntry 类型清单(UserMessage/AssistantMessage/ToolResultMessage/CompactionEntry/...),**无 SystemPromptEntry**

### 反事实(权重 0.1)✅(强化证据)

- 反事实检验:若 compaction 压缩 system prompt,则 `before_agent_start` event 的 `event.systemPrompt` 在 compaction 后应为空或被裁剪
- 实际:extensions.md line 556 明确 "Inside `before_agent_start`, `event.systemPrompt` and `ctx.getSystemPrompt()` both reflect the chained system prompt as of the current handler"——compaction 后 system prompt 完整存在,可被读取/修改
- 反事实成立:system prompt 不被压缩

## 还原确认

无 rick 代码修改,无需还原。

## 关键事实(Y7 核心结论)

1. **✅ system prompt 被保留(不被压缩)**——Y7 澄清
   - 直接证据:compaction.md "What the LLM sees" 图示,system 独立列为 prefix
   - 机制证据:system prompt 不在 SessionEntry 类型中,不在 buildContextEntries 返回值中,不在 CompactionEntry 字段中
   - 重建证据:每次 agent loop 通过 `before_agent_start` 重建 system prompt(based on systemPromptOptions),compaction 不影响

2. **被压缩的内容**:
   - 较老的 messages(UserMessage/AssistantMessage/ToolResultMessage/BashExecutionMessage)
   - 范围:从 session 起点到 cut point(cut point = 从最新 message 倒退累计达到 keepRecentTokens=20k 处)
   - 压缩产物:structured summary(Goal/Constraints/Progress/Key Decisions/Next Steps/Critical Context + read-files/modified-files)

3. **被保留的内容**:
   - **system prompt**(完整保留,独立于 compaction)
   - **最近 messages**(从 firstKeptEntryId 开始,约 20k tokens)
   - **CompactionEntry 本身**(作为 summary message 注入 context)
   - **tool result 与 tool call 配对**(cut point 规则:never cut at tool results)

4. **compaction 后 context 结构**(LLM 视角):
   ```
   [system_prompt] + [compaction_summary] + [recent_messages_from_firstKeptEntryId]
   ```
   - 不是"summary 替换全部 messages",而是"summary + 最近 20k tokens messages"
   - 也不是"保留关键 message 删除中间",而是"老 messages → summary,最近 messages 原样保留"

5. **split turn 处理**:当单个 turn 超过 keepRecentTokens,cut point 落在 assistant message 中间,生成两个 summary(history summary + turn prefix summary)合并——仍不涉及 system prompt

## 疑问点

- 无疑问点:文档图示 + 源码 + 类型定义 + event 机制四重证据一致,system prompt 保留事实清晰

## 置信度评估(由 research 主调度计算)

- 代码原文 ✅ × 0.4 = 0.4(compaction.md 图示 + session-manager.ts buildContextEntries + CompactionEntry 类型 + extensions.md before_agent_start 四源一致)
- 运行时行为 ✅ × 0.3 = 0.3(算法步骤 + custom-compaction.ts 示例 + 官方图示)
- 文档 ✅ × 0.2 = 0.2(compaction.md + extensions.md + README + session-format.md 四文档交叉验证)
- 反事实 ✅ × 0.1 = 0.1(before_agent_start 在 compaction 后仍能读取完整 system prompt,反证 system prompt 不被压缩)
- 合计 = 1.0(高,≥ 0.8 终止)
