# 依赖关系
task19

# 写域
web/src/types.ts
web/src/components/sessions/SessionSettingsDialog.tsx
web/src/components/layout/SessionBadge.tsx
web/src/components/chat/ChatView.tsx
web/src/components/chat/SteerBar.tsx
web/src/components/jobs/JobsList.tsx
web/src/routes/Jobs.tsx
web/src/api/client.ts
web/src/App.tsx
web/src/components/recovery/RecoveryBanner.tsx

# 任务目标
让「挂起」在 UI 上是一个**清晰、可操作**的状态：因平台升级挂起的会话/任务显示专属徽标与说明，并提供**一键恢复/继续**入口；前端同时展示恢复报告（谁被挂起了、恢复结果如何）。

# 关键结果
1. `web/src/types.ts`：`SessionStatus` 联合类型加 `"suspended"`；`JobSummary`/`SessionInfo` 若有 reason 字段则加 `last_reason?: string`；新增 `RecoveryReport` 类型。
2. `web/src/api/client.ts`：`continueSession(id)`（`POST /api/sessions/{id}/continue`）、`getRecoveryReport()`（`GET /api/recovery`）。
3. `SessionBadge.tsx` / `SessionSettingsDialog.tsx`：`suspended` → 徽标「⏸ 已挂起（平台升级）」+ 会话设置里说明文案 + 「▶ 恢复继续」按钮（调 `continueSession`，成功后跳转会话页）；与现有 error（「已中断」）/closed（「已完成」）三态区分。
4. `ChatView.tsx` + `SteerBar.tsx`：`phase === "suspended"` 时输入区显示「会话因平台升级挂起 —— 点击恢复继续（不会自动续跑，避免副作用）」，按钮调 `/continue`；**不得**沿用「已中断（agent 进程不在）」的旧文案（语义不同）。
5. `JobsList.tsx` / `Jobs.tsx`：doing/dream job 卡片在挂起时显示「⏸ 已挂起 · 点继续执行」→ 调 `continueSession`（后端会归一化 `running→pending` 并续跑剩余 task）；归档区同款入口。
6. `RecoveryBanner.tsx`（新）+ `App.tsx`：顶部横幅展示最近一次平台升级的恢复报告（挂起 N 个、已恢复 M 个、失败 K 个），可展开明细；仅当存在未恢复项或报告时间在 24h 内时显示，可关闭并记住（localStorage）。
7. 关键：所有「恢复」动作都是**人工触发**——不引入任何自动重试/自动 resume（与 task19 的硬约束一致）。

# 测试方法
npx --prefix web tsc --noEmit -p web && npm --prefix web run build
go build ./...
python3 .rick/jobs/job_36/plan/gates/gate9.py

# 上下文提示
- 契约（task19 已定）：状态字符串 `suspended`；`POST /api/sessions/{id}/continue` 返回 `{ok:true}` 或 202（幂等）；`GET /api/recovery` 返回 `{at, suspended[], recovered[], failed[]}`。
- 样式沿用 R&M 主题 token（挂起用 `morty` 黄或 `ink-3` 灰，不要与 error 的红/danger 混用）。
- 会话页 `phase` 计算在 `ChatView.tsx`：`sseState.busy ?? info.busy ?? vm.streaming` 那套之外，需要新增基于 status 的分支——**server 权威**原则不变。
