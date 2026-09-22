# 写域
web/src/api/sse.ts
web/src/App.tsx（连接状态指示挂载点，如需）
web/src/components/layout/**（状态指示组件，如需新增）
web/src/stores/events.ts（仅在需要配合 cursor 时最小改动）

# 背景（已实测确认的事实）
- server 端 SSE **已完整实现** Last-Event-ID 续传：GET /api/events 支持 `Last-Event-ID` header 或 `?lastEventID=` query；Hub 有全局单调 seq + 1000 条环形重放缓冲；溢出时发 server_info{reason:"replay_overflow",reconnect:true}。
  实测证据：`curl -N ".../api/events?lastEventID=5"` → 立即补发 id:12624（当前 seq）等事件。
- pi worker 与 doing/dream 后台任务都跑在 **Go server 进程内**（pumpWorker goroutine / startBackground goroutine），与浏览器无关 → 刷新页面不影响会话执行（已具备）。
- **缺口（本次任务）**：前端 `SseClient.lastSeq` 只存内存 → 浏览器 HTTP 刷新（F5）后新 EventSource 不带 cursor，只能从 seq 0 全量 replay（flush 1000 条 → 易触发 overflow → resync 丢进行中流式内容）。刷新期间的事件本可由 server 缓冲补发，前端却没带 cursor。

# 任务目标
页面刷新/断网后**无缝续传**：前端持久化 SSE cursor，新连接带 `lastEventID` 增量补拉断连期间事件（含进行中的 thinking/text 流）——实现「刷新后后台会话不断连、状态由 server 维护」的用户体感。

# 关键结果
1. **cursor 持久化**（sse.ts）：
   - `lastSeq` 初始化时从 `localStorage["rick-web-sse-cursor"]` 读取（无则 -1）
   - 每次 seq 前进后**节流持久化**（如 500ms 或每 N 条；避免每条事件写 localStorage 的性能开销；卸载/隐藏时（visibilitychange/pagehide）立即 flush）
   - seq 回绕（server 重启，已有 1_000_000 阈值探测处）→ 清 cursor
2. **新连接带 cursor**（sse.ts.open）：cursor >= 0 时 URL 加 `&lastEventID=<cursor>`（token 已在 query，server 支持 query fallback）
3. **溢出降级**：收到 `server_info` 且 `data.reason === "replay_overflow"` → **清 cursor**（下次全量）+ 触发既有 `SSE_RESYNC_EVENT`（stores 全量刷新对齐 server）；避免重复拖垮
4. **连接状态可见**（用户诉求「server 维护状态、页面只是显示」）：
   - sse.ts 已 `emitState(connecting|open|reconnecting|closed)`——导出订阅入口（若无）
   - UI：连接断开/重连中时显示轻量提示条（如顶栏/侧栏底部小字「重连中…后台会话继续运行」；恢复后自动消失）——桌面+移动端都可见且不打扰
5. **验证（必须实测）**：
   - 场景 A（刷新续传）：dev/verify 实例（6173 或 8412）打开会话 → 记录当前 cursor（localStorage）→ 触发若干新事件（可 mock SSE 或真实会话操作）→ `page.reload()` → 断言：新连接 URL 带 lastEventID；刷新期间事件在 vm 中可见（thinking/text 内容不丢）
   - 场景 B（worker 解耦）：造 active 会话（plan 或 dream interactive，server 端 spawn）→ 关闭页面（context.close）→ 数秒后调 GET /api/sessions/{id} 断言仍 active（worker 未因页面断开而死）→ 重新打开页面 → 会话 active + 历史完整
   - 场景 C（溢出）：模拟 cursor 过期（手动设 localStorage cursor=1 且 buffer 已滑出）→ 连上后收到 replay_overflow → cursor 被清 + resync 触发 + 页面正常（无卡死/无错误）
   - 回归：正常流式、resync、消息顺序与折叠状态不受影响；tsc + build 绿

# 约束
- 不改 server（已具备能力；如需在报告中建议后续 server 改动，仅记录不实施）
- 写域外文件不动；不碰 git
- 回执：改动清单 + 三场景实测数据（URL cursor、事件补齐证据、worker 存活断言）+ 回归结果
