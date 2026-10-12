# research-L6-leaf-2：pi `--session` resume 行为实证

环境：pi 0.84.1。全部实验在 `/tmp/l6/` 的 session **副本**上做；未改动 `/home/hadoop-recsys/.rick/pi/agent/sessions/` 任何原文件（源文件 mtime 仍 2026-09-01 18:49:50），生产 web pid 172761 全程存活。

## 0. 结论速览

1. **pi 不续跑未完成回合**：对 4 种尾部形状实测，`get_state` 均 `isStreaming:false`、`pendingMessageCount:0`、零 agent 事件。resume = 加载 + 待命，**续跑必须由 rick 主动发 prompt**。
2. **中断无统一标记**：633 个真实 session 的尾部有 5 种形状、4 种属中断；最危险的是 assistant `stopReason:toolUse` 且无 toolResult（pi 不回填也不丢弃）。
3. **pi 对 session 文件无排他锁**：两进程可同时打开同一 session。rick 的防重只在内存 supervisor，重启后孤儿 worker + 自动 resume = 双写同一 jsonl。
4. **session 查找按 cwd/project scoped**：cwd 不匹配时非交互直接 `Aborted.`（rpc 下会吃掉 stdin 一行 → 潜在挂死）。

## 1. 事实表

| # | 结论 | 证据 |
|---|---|---|
| F1 | pi 0.84.1，包 `@earendil-works/pi-coding-agent` | `readlink -f $(which pi)` → `.../dist/cli.js` |
| F2 | `--session <path\|id>` = 指定 session 文件或 UUID 前缀；`--session-id` = 建缺失；`--mode text\|json\|rpc`；`-p` = 非交互处理即退 | `pi --help` Options 段 |
| F3 | jsonl 记录类型：`message` 62893、`custom_message` 2147、`session` 137、`model_change` 185、`thinking_level_change` 171、`session_info` 90、`compaction` 8。role ∈ {user,assistant,toolResult}；content 块 ∈ {text,thinking,toolCall,image}。**无** agent_end/message_end 落盘 | 137 文件全量 census |
| F4 | assistant `stopReason` 值域：`toolUse` 27411、`stop` 2270、`aborted` 166、`error` 5 | 同上 |
| F5 | toolCall 块字段是 `arguments`（非 input），toolResult 用 `toolCallId`+`toolName`+`isError` 关联 | `{"type":"toolCall","id":"call_00_ET_BKw..","name":"bash","arguments":{...}}` |
| F6 | 真实中断样本：`2026-09-01T10-37-53-716Z_01a05c8b-*.jsonl` 末 2 条 = assistant(toolUse,bash) → toolResult，**此后无 assistant** | 末记录 id `ff8fe253` |
| F7 | 全库 633 session 尾部 census：`assistant/stop` 558；`toolResult` **22**；`assistant/aborted` 19；`assistant/toolUse` 悬挂 **19**；`error` 5；`length` 2 ⇒ 约 10% 停在未完成回合，3% 为危险悬挂态 | 全库 tail census |
| F8 | resume **不**自动续跑。对 `stop`/悬挂 `toolUse`/`toolResult`/`user` 四种副本 `pi --mode rpc --session <path>`+`get_state`：均 `success:true,isStreaming:false,pendingMessageCount:0`，无 agent_start（bogus 模型，观察 10~12s） | `{"command":"get_state","success":true,"data":{...,"isStreaming":false,"messageCount":1076,"pendingMessageCount":0}}` |
| F9 | pi **不修复**悬挂 toolCall：构造 5 行最小 session（…user→assistant[toolCall,stopReason=toolUse]）后发 prompt，悬挂条原样保留、无合成 toolResult：u1→a1(toolUse,[text,toolCall])→新 user→新 assistant | `grep -c call_l6_dangling synth-dangling-A.jsonl` = 1 |
| F10 | 源码同结论：仅机械转换 `toolCall→{type:tool_use,id,name,input:arguments??{}}`、`toolResult→user 消息的 tool_result`，**无配对校验/补全/丢弃** | pi-ai `dist/api/anthropic-messages.js:932-938,960-978`；pi-agent-core 搜 `unpaired/without tool` 无命中 |
| F11 | **无并发锁**：两进程同时 `--session concur-A.jsonl` 均 `success:true` 且 `sessionFile` 同路径；无 lock 文件；载入不改文件（1082 行不变，仅真写才追写） | 并发实测 |
| F12 | 解析 cwd scoped（目录 `--workdir-sunquan20-AI_CODING-rick--` 由 cwd 编码）。`/tmp` 下用 uuid 打开 rick 项目 session：`Session found in different project: /workdir/.../rick` → `Fork this session into current directory? [y/N] Aborted.` | 实测 |
| F13 | `get_entries` 返回完整分支含悬挂条目（1082 行 → 1081 entry，末 3 = toolResult/assistant(toolUse)/toolResult），web 可忠实回放未完成回合 | 实测 |
| F14 | 配额耗尽不报 error：catpaw-proxy 返回普通 assistant 文本 `stopReason:"stop"`、usage 全 0（"您的2026-09-20额度已使用完毕…"）⇒ rick 无法从 stopReason 区分「答完」与「配额拒绝」 | `dangling-A-raw.txt` 的 turn_end/agent_end |

## 2. 中断判定算法

```
输入：jsonl 最后一条 message（role/stopReason/content），可选 rick 侧 intent
1) role == toolResult                     → INTERRUPTED(工具后中断)   # F6，真实 22 例
2) role == assistant:
     stopReason == aborted                → INTERRUPTED(有标记)       # 19 例
     stopReason == error                  → FAILED                    # 5 例
     stopReason == toolUse 且末尾 toolCall 无 toolResult
                                          → INTERRUPTED(悬挂，禁自动续跑) # 19 例
     stopReason == stop                    → COMPLETE
3) role == user                           → PENDING(未被应答)
4) 无 message                             → EMPTY
```

- jsonl **无法**区分「进程被杀」与「用户 abort」（都是 aborted/空白）。
- 需 rick 持久化 intent：`SessionEntry` 无该字段，但 `Params map[string]any` 可放 `_` 前缀保留键（`toSessionInfo` 已隐藏 `_` 键，无需改 schema）。`Busy` 是 `json:"-"` 不落盘（internal/web/registry.go:247-254），**不可**作依据。
- 自动恢复条件 = intent=auto_resume ∧ 判定 ∈ {toolResult, aborted, PENDING}；判定为悬挂 toolUse 时只恢复 worker、不续跑。

## 3. 续跑策略（推荐）

| 策略 | 依据 | 代价 |
|---|---|---|
| A 只恢复 worker，不续跑 | F8 安全 | 无 token；需人工发消息 |
| **B 仅对安全形状自动续跑**：resume 后 rick 注入一次「重启中断，请从上次工具结果继续」 | F8 证明必须 rick 主动发；排除悬挂态避免 400（F9/F10） | 每次重启耗 token，语义漂移 |
| C 重发最后一条 user 消息 | 悬挂态前序非法 | 高：400 + 副作用重放 |
| D steer/follow_up 续跑 | rpc 有 `Steer`/`FollowUp`（internal/runtime/rpc.go:185-196） | 语义等价性未测 |

推荐 **B+A**：默认全量恢复 worker；仅 intent 标记且形状安全者注入一次续跑 prompt；悬挂态降级为「恢复 + 人工确认」；限「每会话一次 / 全机 N 次每重启」防风暴。

## 4. 未验证 / 不确定

- U1 悬挂 toolCall 是否真 400：本地已证 pi 不修复（F9/F10），但 provider 端到端未验（实测日配额耗尽，请求未进入消息校验）。confidence 中高（源码 + Anthropic 契约）。
- U2 steer 对 idle worker 是否等价 prompt（未测）。
- U3 SIGKILL 打断**流式生成中**落盘哪条：现有 aborted 有显式标记，推测仅优雅 abort 写；未能复现（配额受限）。
- U4 `--session-id` 与 `--session` 在 uuid 冲突时的差异未测。
- U5 双写同 jsonl 的实际损坏形态未构造，仅证无锁（F11）。

## 5. 对 L6 的影响

- 自动恢复必须 rick 自实现；需新增 intent 持久化（Params 保留键即可，sessions.json `version:1` 后向兼容）。
- 提升重启时 rick 必 `ShutdownWorkers()` 强杀整个 worker 进程组（internal/web/sessions.go:1746；supervisor.go Setpgid 注释）⇒「重启即批量产生中断回合」是设计内生事实，恢复逻辑必须覆盖 F7 的 22+19 类形状。
