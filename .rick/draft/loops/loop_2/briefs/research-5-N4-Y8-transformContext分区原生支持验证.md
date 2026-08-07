# research-5 N4-Y8 transformContext 分区原生支持验证

节点路径:[根 > Y8 上下文分区 > N4-transformContext 分区原生支持验证]
事实陈述:验证 pi transformContext 是否支持"上下文分区"原生概念(context regions / segments / zones),还是需在 extension 内自行实现分区逻辑;区域化保留/压缩是否可实现

## 执行动作

1. Read `/tmp/pi_repo/packages/agent/src/types.ts` line 175-205(transformContext 签名 + 注释)
2. Read `/tmp/pi_repo/packages/agent/src/agent.ts` line 90-125(AgentOptions)+ line 170-225(Agent 类属性)
3. Read `/tmp/pi_repo/packages/agent/src/agent-loop.ts` line 280-320(streamAssistantResponse 中 transformContext 调用点)
4. Read `/tmp/pi_repo/packages/agent/README.md` line 50-70(transformContext 用法说明)
5. Read `/tmp/pi_repo/packages/coding-agent/src/core/session-manager.ts` line 40-154(SessionEntryBase / SessionEntry 类型)
6. Read `/tmp/pi_repo/packages/coding-agent/src/core/extensions/types.ts` line 560-660(SessionBeforeCompactEvent / SessionCompactEvent / SessionBeforeTreeEvent)
7. Grep `region|segment|zone|partition|tag.*context|context.*tag` 全 pi_repo(排除 TUI 文本分段无关结果)
8. Grep `SessionEntryBase|interface.*Entry|metadata|tags` extensions/types.ts
9. Read `/tmp/pi_repo/packages/coding-agent/src/core/sdk.ts` line 340-360(sdk.ts 中的 transformContext 示例)
10. Read `/tmp/pi_repo/packages/agent/src/harness/agent-harness.ts` line 485-500(harness 中的 transformContext 示例)

## 各信源验证结果

### 代码原文(权重 0.4)✅

**transformContext 签名(决定性证据)**(`packages/agent/src/types.ts` line 175-195):
```typescript
/**
 * Optional transform applied to the context before `convertToLlm`.
 *
 * Use this for operations that work at the AgentMessage level:
 * - Context window management (pruning old messages)
 * - Injecting context from external sources
 *
 * Contract: must not throw or reject. Return the original messages or another
 * safe fallback value instead.
 */
transformContext?: (messages: AgentMessage[], signal?: AbortSignal) => Promise<AgentMessage[]>;
```

**AgentOptions 中的 transformContext**(`packages/agent/src/agent.ts` line 101):
```typescript
transformContext?: (messages: AgentMessage[], signal?: AbortSignal) => Promise<AgentMessage[]>;
```

**Agent 类属性**(`packages/agent/src/agent.ts` line 180):
```typescript
public transformContext?: (messages: AgentMessage[], signal?: AbortSignal) => Promise<AgentMessage[]>;
```

**agent-loop.ts 调用点**(`packages/agent/src/agent-loop.ts` line 288-292):
```typescript
// Apply context transform if configured (AgentMessage[] → AgentMessage[])
let messages = context.messages;
if (config.transformContext) {
    messages = await config.transformContext(messages, signal);
}
```

**关键事实 1:transformContext 签名是单参 `messages: AgentMessage[]`,返回 `AgentMessage[]`**
- 输入:完整的 messages 数组(扁平,无分区标记)
- 输出:转换后的 messages 数组(扁平,无分区标记)
- **无 region/tag/zone/category 原生概念**
- **无 ctx.getRegion(name) / ctx.setTag() / ctx.setRetentionPolicy() 等 API**

**关键事实 2:transformContext 调用时机**(agent-loop.ts line 288-292):
- 每次 LLM 调用前(streamAssistantResponse 函数内)
- 在 `convertToLlm`(AgentMessage[] → LLM Message[])之前
- **不在 compaction 之前**(compaction 是独立流程,由 session-manager 触发)

**关键事实 3:SessionEntry 类型无 tag/metadata/region 字段**(`session-manager.ts` line 46-153):

```typescript
export interface SessionEntryBase {
    type: string;
    id: string;
    parentId: string | null;
    timestamp: string;
}
// 9 个子类型:SessionMessageEntry / ThinkingLevelChangeEntry / ModelChangeEntry /
// CompactionEntry / BranchSummaryEntry / CustomEntry / CustomMessageEntry /
// LabelEntry / SessionInfoEntry
```

- SessionEntryBase 只有 `type/id/parentId/timestamp` 4 个字段
- **无 tag/region/zone/category/metadata 字段**
- CustomMessageEntry 有 `customType: string` 字段(line 106),可作为"分区标记"的载体,但需 extension 自行实现分区逻辑
- LabelEntry 有 `targetId + label` 字段(line 111-115),是"书签/标记"机制,但 label 不参与 LLM context(buildSessionContext 忽略 LabelEntry)

**关键事实 4:grep `region|segment|zone|partition` 全 pi_repo 结果**:
- `region`:仅 AWS_REGION / OSC133_ZONE_PREFIX(TUI)/ 环境变量,与 context 无关
- `segment`:仅 TUI 文本分段(Intl.Segmenter / word-navigation),与 context 无关
- `zone`:仅 OSC133_ZONE_PREFIX(TUI 终端转义),与 context 无关
- `partition`:无匹配
- **pi 无 "context region" / "context segment" / "context zone" / "context partition" 概念**

**关键事实 5:grep `tag.*context|context.*tag` 全 pi_repo 结果**:
- 无 context tagging 机制
- `tag` 出现在:git tag(version)/ syntax-highlight span tag(HTML)/ tools-manager tagPrefix(版本前缀),均与 context 无关

**关键事实 6:transformContext 官方示例**(`agent/types.ts` line 186-193 + `agent/README.md` line 193 + `agent-harness.ts` line 493 + `coding-agent/sdk.ts` line 350):
```typescript
// 官方示例 1(types.ts 注释)
transformContext: async (messages) => {
    if (estimateTokens(messages) > MAX_TOKENS) {
        return pruneOldMessages(messages);
    }
    return messages;
}

// 官方示例 2(README.md)
transformContext: async (messages, signal) => pruneOldMessages(messages)

// 官方示例 3(agent-harness.ts)
transformContext: async (messages) => { /* 自定义逻辑 */ }

// 官方示例 4(sdk.ts)
transformContext: async (messages) => { /* 自定义逻辑 */ }
```
- **所有官方示例都是"整体 transform"**(prune old messages / inject external context)
- **无"按区域 transform"示例**(无 `if message.region === "debug" then keep else compact` 模式)

**关键事实 7:compaction 扩展点组合能力**(前序 research-4-N3 已确认 + 本轮交叉验证):
- `session_before_compact` event:可 cancel/custom(compaction 前,可读取 branchEntries + preparation.messagesToSummarize)
- `session_compact` event:compaction 后通知
- `ctx.compact()` API:主动触发 compaction
- `before_agent_start` event:每次 agent loop 重建 system prompt(可动态修改)
- `transformContext`:每 turn 转换 messages(agent-core 层)
- **5 类扩展点可组合使用**,但组合方式由 extension 自行实现(pi 无内置"区域化保留/压缩"组合逻辑)

### 运行时行为(权重 0.3)✅

**Y8 human 论点回顾**:"内嵌到 agent loop 内部,pi 扩展内部,自由组合上下文,给上下文分区,不同区域在 loop 过程中实现保留或压缩"

**验证结论**:
- ✅ "内嵌到 agent loop 内部":transformContext 在 agent-loop.ts line 288 每次 LLM 调用前执行,是真"内嵌"
- ✅ "pi 扩展内部":transformContext 是 AgentOptions 配置项,extension 可通过 Agent 类注入
- ✅ "自由组合上下文":transformContext 输入输出都是 `AgentMessage[]`,可任意重排/过滤/注入
- ❌ "给上下文分区(原生)":pi **无原生分区概念**,SessionEntryBase 无 tag/region 字段,messages 数组扁平
- ⚠️ "给上下文分区(自建)":可通过 CustomMessageEntry.customType 字段或 AgentMessage 自定义属性自建分区标记,但需 extension 自行实现分区逻辑(pi 不识别)
- ⚠️ "不同区域在 loop 过程中实现保留或压缩":transformContext 每 turn 调用,可基于自建分区标记实现"区域化保留";compaction 通过 session_before_compact + customInstructions 可实现"区域化压缩",但**组合逻辑由 extension 自行实现**

**实现路径候选**(若 human 接受"自建分区抽象"):
1. 在 transformContext 中按 messages 的 customType/自定义属性分区:
   ```typescript
   transformContext: async (messages) => {
       const debugRegion = messages.filter(m => m.customType === "debug");
       const taskRegion = messages.filter(m => m.customType === "task");
       const historyRegion = messages.slice(-10); // 最近 10 条
       return [...debugRegion, ...taskRegion, ...historyRegion];
   }
   ```
2. 在 session_before_compact 中按区域自定义压缩:
   ```typescript
   pi.on("session_before_compact", async (event, ctx) => {
       const { preparation, branchEntries } = event;
       // debug 区域不压缩,task 区域压缩,history 区域保留最近 N 条
       const debugEntries = branchEntries.filter(e => e.customType === "debug");
       const taskEntries = branchEntries.filter(e => e.customType === "task");
       const summary = await customSummarize(taskEntries);
       return { compaction: { summary, firstKeptEntryId: ..., details: { debugEntries } } };
   });
   ```
3. 在 before_agent_start 中按区域注入 system prompt:
   ```typescript
   pi.on("before_agent_start", async (event, ctx) => {
       const debugRegion = loadDebugRegion();
       const taskRegion = loadTaskRegion();
       ctx.setSystemPrompt(`${debugRegion}\n${taskRegion}\n${event.systemPrompt}`);
   });
   ```

**关键限制**:
- 上述 3 路径均需 extension 自行实现分区逻辑(pi 不原生支持)
- CustomMessageEntry 参与 LLM context(buildSessionContext 转为 user message),但 customType 字段本身不进 LLM(需 extension 在 transformContext 中显式提取并重组)
- 自建分区抽象的维护成本:分区标记一致性 / 跨 session 持久化 / 与 compaction 交互 / 与 tree navigation 交互

### 文档(权重 0.2)✅

- `packages/agent/README.md` line 58:"AgentMessage[] → transformContext() → AgentMessage[] → convertToLlm() → Message[] → LLM"(明确 transformContext 在 convertToLlm 之前)
- `packages/agent/README.md` line 62:"transformContext: Prune old messages, inject external context"(官方用途仅 2 类:pruning + injecting,**无 partitioning**)
- `packages/agent/CHANGELOG.md` line 621:"preprocessor → transformContext"(历史重命名,无分区概念引入)
- pi 全部 docs/ 下文档(compaction.md / extensions.md / sdk.md / session-format.md / skills.md / prompt-templates.md)无 "region" / "segment" / "zone" / "partition" 上下文分区相关词汇

### 反事实(权重 0.1)✅

**反事实验证**:若 pi 原生支持上下文分区,则应满足以下任一条件:
- SessionEntryBase 有 `region/tag/zone/category` 字段 → ❌ 实测无
- AgentMessage 类型有 `region/tag` 属性 → ❌ 实测 AgentMessage 是 union(UserMessage/AssistantMessage/ToolResultMessage 等),无 region 属性
- transformContext 签名有 `ctx` 参数(类似 `ctx.getRegion(name)`)→ ❌ 实测只有 `(messages, signal)`
- 官方示例有"按区域 transform"模式 → ❌ 实测全部是"整体 transform"
- docs 有 "context region" / "context segment" 章节 → ❌ 实测无

**反证结论**:pi **不原生支持**上下文分区,Y8 human 论点的"给上下文分区"需在 transformContext / session_before_compact / before_agent_start 之上**自建分区抽象**

## 还原确认

无 rick/pi 代码修改,无需还原。

## 关键事实

1. **transformContext 签名**:`(messages: AgentMessage[], signal?) => Promise<AgentMessage[]>`,单参 messages 数组,返回 messages 数组,**无原生分区概念**
2. **SessionEntryBase 字段**:仅 `type/id/parentId/timestamp`,**无 tag/region/zone/category/metadata**
3. **pi 无 "context region/segment/zone/partition" 概念**:grep 全 pi_repo 无匹配(排除 TUI 文本分段无关结果)
4. **transformContext 调用时机**:每 turn LLM 调用前(agent-loop.ts line 288),在 convertToLlm 之前,**不在 compaction 之前**
5. **官方示例全部是"整体 transform"**:prune old messages / inject external context,**无"按区域 transform"示例**
6. **Y8 human 论点验证**:
   - ✅ "内嵌 agent loop 内部":transformContext 在 agent-loop 每次 LLM 调用前执行,真内嵌
   - ✅ "pi 扩展内部":transformContext 是 AgentOptions,extension 可注入
   - ✅ "自由组合上下文":transformContext 输入输出都是 messages 数组,可任意重排/过滤/注入
   - ❌ "给上下文分区(原生)":pi **无原生分区概念**
   - ⚠️ "给上下文分区(自建)":可通过 CustomMessageEntry.customType 或 AgentMessage 自定义属性自建分区标记,但需 extension 自行实现分区逻辑
   - ⚠️ "区域化保留/压缩":transformContext(每 turn 保留)+ session_before_compact(压缩时保留)可实现,但**组合逻辑由 extension 自行实现**
7. **5 类扩展点组合能力**:transformContext + session_before_compact + session_compact + ctx.compact + before_agent_start 可组合使用,但 pi 无内置"区域化保留/压缩"组合逻辑
8. **实现路径候选**:3 路径(transformContext 分区 / session_before_compact 区域压缩 / before_agent_start 区域注入),均需 extension 自建分区抽象
9. **关键限制**:自建分区抽象的维护成本(分区标记一致性 / 跨 session 持久化 / 与 compaction 交互 / 与 tree navigation 交互)
10. **反事实验证**:pi 原生支持分区的 5 个条件(SessionEntryBase 字段 / AgentMessage 属性 / ctx 参数 / 官方示例 / docs 章节)**全部不满足**

## 疑问点

- human 是否接受"自建分区抽象"作为 Y8 "给上下文分区"的实现?(价值性判断,非事实调研)
- 自建分区抽象的维护成本是否可接受?(架构决策,非事实调研)
- rick 现有 doing_loop.md 内嵌 doing.md(模板文本内嵌)是否算"上下文分区"的弱形态?(语义边界,前序 r4 门禁已识别)

## 置信度评估(由 research 主调度计算)

- 代码原文 ✅ × 0.4 = 0.4
- 运行时行为 ✅ × 0.3 = 0.3
- 文档 ✅ × 0.2 = 0.2
- 反事实 ✅ × 0.1 = 0.1
- 合计 = 1.0(高,≥ 0.8 终止)
