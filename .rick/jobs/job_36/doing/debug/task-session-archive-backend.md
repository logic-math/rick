# 写域
internal/web/registry.go
internal/web/sessions.go
internal/web/routes.go
internal/web/*_test.go（本任务相关）

# 任务目标
会话级「人工归档」后端：每个会话可被用户手动标记完成并归档（不做 agent 自动判定）；归档列表可分页查询；可恢复单个归档会话。

# API 契约（前端将按此对接——必须严格一致）
## SessionEntry 注册表扩展（~/.rick/web.json sessions[]，向后兼容：omitempty 不升 schema version）
Archived   bool      `json:"archived,omitempty"`
ArchivedAt time.Time `json:"archived_at,omitzero"`

## sessionInfo wire 投影（toSessionInfo）
加 Archived bool `json:"archived,omitempty"` + ArchivedAt *time.Time `json:"archived_at,omitempty"`（零值省略，参照 ClosedAt 指针模式）

## GET /api/sessions?workspace=<ws>（现有 ListSessions）
- 现状：返回裸数组；新增行为：**默认响应不含 archived=true 的会话**（含 ArchivedAt 非零 / Archived 标记）
- 新参数 archived=true：返回**分页对象**（区别于裸数组——前端据此分支）：
  {"items":[sessionInfo...], "total":N, "limit":L, "offset":O}
  按 created_at desc 排序；limit 默认 50 上限 200；offset 默认 0
- 不带 archived 参数 → 保持裸数组兼容

## POST /api/sessions/{id}/archive → 204（幂等）
语义：人工「标记完成归档」。
- 若会话 status=active/running（worker 活着或后台在跑）→ 先终止（复用 close 的终止逻辑：closing map + sup worker Close/Remove 或后台 cancel+bgWait）+ 状态置 closed（reason "archived"）
- 已 closed/error → 仅标记
- 设 Archived=true + ArchivedAt=now，registry 持久化
- Hub 广播 session_state（status 变更照常；archived 由 GET 透出——hub 事件不新增字段）
- 已归档再次调用 → 204 幂等

## POST /api/sessions/{id}/unarchive → 204（幂等）
- 清 Archived/ArchivedAt（回默认列表）
- 不改变 status（closed 仍 closed——前端点开会走 Resume）

# 现有代码坐标
- registry.go：SessionEntry struct（~233 行）+ LoadSessionRegistry/Add/Update/持久化；sessionRegistryVersion=1
- sessions.go：sessionInfo struct（177）+ toSessionInfo（189）；ListSessions（~415）；SessionClose（~525-570：m.closing + cancel/bgWait + worker.Close/Remove + updateStatus closed）——把终止逻辑抽成内部函数（如 closeSessionEntry(entry, reason)）供 SessionClose 与 archive 复用，避免重复
- routes.go：mux 注册（~645 行：GET /api/sessions / POST /api/sessions/{id}/resume 等）；新增 POST /api/sessions/{id}/archive、/unarchive 注册
- SessionResume 对 archived 会话：允许（恢复执行后仍 archived？——resume 语义：closed→active 重新 spawn。**决定：resume 一个已归档会话时自动清 archived**（用户要重新用了）——在 SessionResume 成功后清标记）
- markWorkerLost/ReconcileOnStart 不动

# 测试方法（go test ./internal/web/ 全绿）
- archive active 会话：worker 被终止 + status=closed + archived=true
- archive error/closed/已归档幂等
- unarchive 幂等 + 回默认列表
- 默认列表不含归档；archived=true 分页（limit/offset/total/排序 created_at desc）
- registry 重启持久化（archived 保留）
- resume archived 会话 → archived 被清

# 纪律
不碰 git；回执：改动清单 + 测试结果
