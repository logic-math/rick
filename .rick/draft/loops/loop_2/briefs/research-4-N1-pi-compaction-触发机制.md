# research-4 N1-pi compaction 触发机制

节点路径:[根 > Y7-pi compaction 是否保留 system prompt > N1-pi compaction 触发机制]
事实陈述:pi compaction 的触发机制——自动/手动/阈值/可配置。

## 执行动作

1. `curl -sL https://raw.githubusercontent.com/earendil-works/pi/main/packages/coding-agent/README.md` → `/tmp/pi_readme.md`(710 行)
2. `curl -sL https://raw.githubusercontent.com/earendil-works/pi/main/packages/coding-agent/docs/extensions.md` → `/tmp/pi_extensions.md`(2987 行)
3. `curl -sL https://raw.githubusercontent.com/earendil-works/pi/main/packages/coding-agent/docs/compaction.md` → `/tmp/pi_compaction.md`(401 行,专项文档)
4. `curl -sL https://raw.githubusercontent.com/earendil-works/pi/main/packages/coding-agent/src/core/compaction/compaction.ts` → `/tmp/pi_compaction_ts.ts`(969 行,源码)
5. `curl -sL https://raw.githubusercontent.com/earendil-works/pi/main/packages/coding-agent/examples/extensions/trigger-compact.ts` → `/tmp/pi_trigger_compact.ts`(50 行,官方示例)
6. grep 关键词:compaction / compact / threshold / overflow / reserveTokens / keepRecentTokens / enabled

## 信源验证结果

### 代码原文(权重 0.4)✅

**compaction.md line 27-35(触发条件原文)**:
> Auto-compaction triggers when:
> ```
> contextTokens > contextWindow - reserveTokens
> ```
> By default, `reserveTokens` is 16384 tokens (configurable in `~/.pi/agent/settings.json` or `<project-dir>/.pi/settings.json`). This leaves room for the LLM's response.

**compaction.md line 37**:
> You can also trigger manually with `/compact [instructions]`, where optional instructions focus the summary.

**extensions.md line 332-333(event 触发)**:
> /compact or auto-compaction
>   ├─► session_before_compact (can cancel or customize)
>   └─► session_compact

**extensions.md line 459-460(reason 字段)**:
> // reason - "manual" (/compact), "threshold", or "overflow"
> // willRetry - whether the aborted turn is retried after compaction (overflow recovery)

**README line 272-280(Compaction 章节)**:
> **Manual:** `/compact` or `/compact <custom instructions>`
> **Automatic:** Enabled by default. Triggers on context overflow (recovers and retries) or when approaching the limit (proactive). Configure via `/settings` or `settings.json`.

**compaction.md line 381-401(Settings 章节,可配置项)**:
```json
{
  "compaction": {
    "enabled": true,
    "reserveTokens": 16384,
    "keepRecentTokens": 20000
  }
}
```
| Setting | Default | Description |
| `enabled` | `true` | Enable auto-compaction |
| `reserveTokens` | `16384` | Tokens to reserve for LLM response |
| `keepRecentTokens` | `20000` | Recent tokens to keep (not summarized) |
> Disable auto-compaction with `"enabled": false`. You can still compact manually with `/compact`.

**trigger-compact.ts(官方示例,自定义阈值触发)**:
- 监听 `turn_end`,当 `previousTokens <= 100_000 && currentTokens > 100_000` 时调用 `ctx.compact()`
- 注册 `/trigger-compact` 命令,手动触发并支持 `customInstructions`
- 证明:触发阈值可由 extension 完全自定义(不限于默认 reserveTokens 逻辑)

### 运行时行为(权重 0.3)✅

- README 明确:"Automatic: Enabled by default"——开箱即用,无需配置
- compaction.md 明确两种自动触发场景:
  - **overflow**:context 超出窗口(已溢出),pi 中止当前 turn → compaction → 重试(willRetry=true)
  - **threshold**:接近窗口上限(主动触发,避免溢出)
- compaction.ts 源码 969 行实现了 prepareCompaction() / compact() 函数(与文档一致)
- 官方示例 trigger-compact.ts 演示了"自定义阈值 + 自定义命令"双路径触发

### 文档(权重 0.2)✅

- README Compaction 章节(line 272-280):Manual + Automatic 双触发
- compaction.md 专项文档 401 行,覆盖触发/算法/结构/自定义全链路
- extensions.md 列出 `session_before_compact` / `session_compact` 两事件 + `ctx.compact()` API

### 反事实(权重 0.1)N/A

- 本节点为外部文档+源码调研,无代码修改

## 还原确认

无 rick 代码修改,无需还原。

## 关键事实

1. **触发机制三态**:
   - **自动-阈值(threshold)**:`contextTokens > contextWindow - reserveTokens`(默认 reserveTokens=16384),接近窗口上限主动触发
   - **自动-溢出(overflow)**:context 已超出窗口,prompt 调用失败后恢复重试(willRetry=true)
   - **手动(manual)**:`/compact [instructions]` 命令或 `ctx.compact({customInstructions})` API
2. **默认开启**:auto-compaction `enabled=true`(开箱即用)
3. **可配置项**(settings.json):
   - `compaction.enabled`(bool,默认 true)
   - `compaction.reserveTokens`(默认 16384,LLM 响应预留)
   - `compaction.keepRecentTokens`(默认 20000,保留最近不压缩的 token 数)
4. **reason 字段三值**:`manual` / `threshold` / `overflow`(extensions/types.ts SessionBeforeCompactEvent 明确)
5. **extension 可完全自定义触发**:
   - 监听 `turn_end` 读取 `ctx.getContextUsage().tokens`,自定义阈值调用 `ctx.compact()`
   - 注册自定义 `/command` 触发
   - `session_before_compact` event 可 `return { cancel: true }` 取消本次 compaction

## 疑问点

- 无疑问点:触发机制文档+源码+示例三重证据,事实清晰

## 置信度评估(由 research 主调度计算)

- 代码原文 ✅ × 0.4 = 0.4(compaction.md + compaction.ts + extensions/types.ts + trigger-compact.ts 四源一致)
- 运行时行为 ✅ × 0.3 = 0.3(README 明确 enabled=true 默认开启 + 官方示例可运行)
- 文档 ✅ × 0.2 = 0.2(README + compaction.md + extensions.md 三文档交叉验证)
- 反事实 N/A × 0.1 = 0
- 合计 = 0.9(高,≥ 0.8 终止)
