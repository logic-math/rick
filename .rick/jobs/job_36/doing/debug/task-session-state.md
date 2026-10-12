# 依赖关系
无

# 写域
internal/web/sessions.go
internal/web/sessions_test.go
web/src/stores/sessions.ts
web/src/components/chat/ChatView.tsx
web/src/api/client.ts

# 任务目标
会话状态根治（用户反馈：非活跃会话仍显示 active，进入发送就 409，应标记中断并引导 resume）。

# 关键结果

## 后端（internal/web/sessions.go）
1. **command() 的 worker 失活处理**：`worker == nil || worker.IsDead()` 分支——在返回 409 之前，把该会话**自动标记 error**（reason: "worker not alive: "+...）并广播 session_state（前端据此立即切换 UI 到 error 态显示 Resume）。同时 `worker.Send()` 失败（进程管道断）也标记 error。
2. **worker 死亡监听强化**：pumpWorker 的 channel 关闭路径已转 closed——改为**区分**：非用户主动关闭（!closing）时转 **error**（reason: "worker exited: "+worker.Reason()）而非 closed——「终止/中断」语义（用户要求：活跃/终止/完成 三态，worker 异常退出=终止=error）。
3. **supervisor 心跳超时联动**：Worker.IsDead() 因 get_state 心跳超时置位时（supervisor markDead），SessionManager 也应收到信号标记 error——若 pumpWorker 的 events channel 因进程死而关闭则 2 已覆盖；若进程活着但 pi 无响应（挂起）则 Events() 不关——需在 command() 的 IsDead 检查覆盖（1 已做）。确认 Worker.IsDead 的判定包含心跳超时（自查 supervisor.go）。
4. 测试：TestSessionState 补充——worker 标记 dead（fake pi 脚本主动退出）→ 会话转 error 而非 closed；command 对 IsDead worker → 自动 error + 409。

## 前端（web/src/stores/sessions.ts + ChatView.tsx + api/client.ts）
5. **session_state 事件应用**：sessions store 的 applyState 已处理 SSE 状态变更——确保 error 状态正确落入 byWorkspace/index（自查现有 applyState，若只处理部分状态补全 active/running/closed/error 全映射）。
6. **ChatView 409 自动降级**：send() 的 catch 里，若 ApiError.status===409（state_conflict/worker not alive）→ 调 applyState(sessionId, "error", "worker lost")（本地立即降级）+ 显示错误横幅「会话已中断，点击 Resume 恢复」；SteerBar 已对 error 显示 Resume——验证 phase 计算把 error 正确映射（自查 phase: status==="error" → "error" 分支已有）。
7. **Resume 引导按钮强化**：error 态的 SteerBar Resume 按钮旁加说明「将重新加载完整历史并恢复 agent 进程」；Resume 成功后（202）本地 applyState active + 前端重拉 entries（触发 history 刷新）。

# 测试方法
- go build ./... + go test ./internal/web/...（含新状态测试）
- npx tsc --noEmit + npm run build
- playwright 冒烟（6173 验证实例）：打开被 reconcile 标 error 的会话 → 显示「会话异常终止 + Resume」；若新建 active 会话（无 worker 场景由 reconcile 处理）——重点验证 error 态 UI 与 Resume 请求发出

# 上下文提示
- 后端 reconcile（重启对账）已实现（ReconcileOnStart）；本 task 补「运行时失活」路径
- 状态语义对齐用户要求：active=活跃 / error=终止（中断）/ closed=完成（主动关闭）
- applyState 在 stores/sessions.ts 已有（自查现状补全）
