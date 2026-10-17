# research-L3-pi-token 简报

阶段/主题：L3-Q4 成本账本 token 双轨数据源（pi agent runtime 的 token/usage 计量）
时间基准：2026-10-10；核实对象 pi-coding-agent v0.84.2 + pi-subagents v0.51.0
信源等级：A=代码原文 B=官方文档 C=运行时产物

## 一、事实性结论

1. **json 模式无独立 usage 事件类型**：事件流只有 agent_start/end、turn_start/end、message_start/update/end、tool_execution_start/update/end、agent_settled、queue_update、compaction_*、auto_retry_* 等；不存在独立 `usage`/`tokenUsage`/`usageInfo` 事件。usage 挂在两个字段上。`message_update` 顶层携带 `usage`（流式累计）；`message_end.message.usage` 携带该轮最终值（权威）。[B] docs/json.md + [A] dist/modes/json-event.js（toJsonEvent 仅对 message_update 注入顶层 usage，其余事件原样透传）

2. **usage 数据结构（pi-ai `Usage` 接口）**：`input/output/cacheRead/cacheWrite`（number）+ 可选 `cacheWrite1h`(仅 Anthropic)/`reasoning`(output 子集) + `totalTokens`(number) + `cost:{input,output,cacheRead,cacheWrite,total}`。[A] node_modules/@earendil-works/pi-ai/dist/types.d.ts:255-283

3. **total 恒等式**：`totalTokens = input + output + cacheRead + cacheWrite`（运行时产物验证：10729+377+132608+0=143714）。[C] .pi/subagents/artifacts/*_transcript.jsonl 内 message.usage

4. **分模型明细**：`getUsageCostBreakdown()` 按 `provider/responseModel` 分组 assistant 消息；tool_result/compaction/branch_summary 归入 "Tools/summaries" 桶。[A] dist/core/usage-totals.js

5. **meta.json 是 pi-subagents 扩展写的，非 pi 本体**：`usage:{input,output,cacheRead,cacheWrite,cost,turns}`（注意 cost 已折叠为 number、无 totalTokens）。[A] pi-subagents/src/shared/types.ts:89-96 + src/runs/foreground/execution.ts:149-172(persistSingleResultMetadata) + src/runs/background/subagent-runner.ts:279(writeMetadata)

6. **meta.json 文件名规则**：`{runId}_{agent}{suffix}_meta.json`，suffix=`_<index>`（如 `006a12e7_worker_0_meta.json`）；路径 project 作用域 = `<cwd>/.pi/subagents/artifacts/`。[A] pi-subagents/src/shared/artifacts.ts:182-192 + :7-15

7. **⚠ py 直调 pi 不会写 meta.json**：meta.json 由"父 pi agent 调用 subagent 工具"这一动作触发；py subprocess 直调 `pi --mode json -p` 时无 subagent 工具参与 → 不落 meta.json。py 侧唯一可靠数据源 = stdout NDJSON。[A] 同上 + [C] rick 自身 .pi/subagents/artifacts/ 均为 subagent 产物

8. **py 直调可用计量路径仅①**：解析 stdout NDJSON，聚合 `message_end`(role==assistant) 的 `message.usage`——这正是 pi-subagents 父进程解析子进程 stdout 的同一逻辑。[A] pi-subagents/src/runs/background/subagent-runner.ts:714-730

9. **rpc vs json usage 差异**：两者事件 schema 相同（同走 JsonAgentSessionEvent，message_update 均带顶层 usage）。rpc 额外有 `get_session_stats` 命令，返回聚合 `tokens:{input,output,cacheRead,cacheWrite,total}`+`cost`+`contextUsage`（含 assistant+tool+compaction 全 session）。json 模式无此命令，只能自己按 message_end 累加。[B] docs/rpc.md:533-570 + [A] dist/core/agent-session.js:2490-2545

10. **rick 现状确认**：executor.go 的 piEvent/piMessage 结构确无 usage/token 字段，message.usage 被 json.Unmarshal 丢弃；但 message_end 事件本身带 usage，只是 rick 没接。[A] rick/internal/runtime/executor.go:29-49

11. **cost 字段**：事件内 `message.usage.cost` 是对象，py 记账取 `cost.total`（美元）；meta.json 已折叠为 number。总成本可用 input/output/cacheRead/cacheWrite 各单价×量自算。[A] usage-totals.js + subagent-runner.ts:729

12. **降级信号**：provider 不上报 usage 时字段为 0（faux/test provider 全 0）；message_update 流式阶段可能全 0，完成时才有效；message.usage 可能 undefined（runner 用 `if(eventUsage)` 守卫）。[A] pi-ai/dist/providers/faux.js:11-16 + docs/rpc.md:960-961

## 二、最小可用接入方案（py 侧）

```
for line in pi --mode json -p <task> 的 stdout:
    ev = json.loads(line)
    if ev.type == "message_end" and ev.message.role == "assistant":
        u = ev.message.get("usage")
        if u: input+=u.input; output+=u.output; cacheRead+=u.cacheRead; cacheWrite+=u.cacheWrite; cost+=u.cost.total
total = input+output+cacheRead+cacheWrite   # 或直接取最后一次 u.totalTokens
```

- 每类 agent(adjust/builder/coding/debug) 一次独立 subprocess = 一个 role 的一次会话，累加结果写入该 role 的 `agent_ended` payload。
- 兜底：若全 session 累加后 input+output 仍为 0（provider 不上报）→ ①回退取最后一次非零 `message_update.usage`；②仍无 → token 记 null/0，账本降级为 wall-clock + tool_call 数（现有字段）；③可选 char/4 粗估。
- 字段路径最终建议：`{input_tokens, output_tokens, cache_read, cache_write, total, cost_usd}` ← 映射自 u.input/u.output/u.cacheRead/u.cacheWrite/totalTokens/u.cost.total。

## 三、R7 上报项

1. 无法 100% 确认所有真实 provider 在 json 模式 message_end 均必带非零 usage（faux 全 0 证明存在零上报路径）——需对实际 deepseek/claude provider 跑一次 smoke 校准。
2. `reasoning`(思考 token) 是 output 子集、可选，若账本要单独列 reasoning 需注意 provider 差异。
3. meta.json 的 `cost` 折叠逻辑（cost.total 累加）与事件流 cost 对象不一致，两路不可混用。
4. py 直调下 get_session_stats(rpc) 不可用（那是 rpc 长连接命令，非 json 单发），无法后置补取 usage。
5. agent_end 事件含全量 messages（可重复、超大，runner 会截断），不建议 py 从 agent_end 取 usage。
6. 未验证 compaction 场景下 json 模式的 usage 是否含 compaction 生成的 token（getSessionStats 会单独计入，py 直读 message_end 会漏 compaction 部分）。
7. 分 role token 无 provider 侧直接字段，只能靠 py 按 subprocess 会话隔离统计。
8. cacheWrite1h/1h 留存拆分仅 Anthropic，非 Anthropic 恒 0。
