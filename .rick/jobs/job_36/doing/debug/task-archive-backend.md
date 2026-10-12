# 依赖关系
无

# 写域
internal/web/archived.go
internal/web/archived_test.go
internal/web/routes.go
internal/web/jobs.go
internal/web/web.go

# 任务目标
job 归档后端：已完成 job 可归档（列表默认隐藏），可恢复；归档状态存 web 层机器级状态（不污染 rick job 文件）。

# 关键结果

1. `internal/web/archived.go`（新建）：
   - `type ArchivedStore struct`——归档状态存储：`{version:1, archived:{<workspace_id>: ["job_5","job_9"]}}`
   - 文件：`~/.rick/web/archived.json`（HOME 跟随，与 web.json 同目录；路径常量在 web.go 加 `ArchivedPath()`）
   - 方法：`LoadArchived(path) (*ArchivedStore, error)`（缺失→空）、`Archive(wsID, jobID) error`（幂等）、`Unarchive(wsID, jobID) error`、`IsArchived(wsID, jobID) bool`、`List(wsID) []string`；原子写（tmp+rename）
2. `internal/web/jobs.go`：`ListJobs` 保持不变（纯读取）；**新增 `ListJobsFiltered(rickDir string, archived []string) []JobSummary`**——从 ListJobs 结果中过滤掉 archived 里的 job_id
3. `internal/web/routes.go`：
   - `GET /api/workspaces/{ws}/jobs`：默认过滤已归档；`?include_archived=true` 返回全部（含 archived 标记字段 `"archived": true`，JobSummary 加 `Archived bool \`json:"archived,omitempty"\``）
   - 新增 `POST /api/workspaces/{ws}/jobs/{job}/archive` → 204（仅 status=success 的 job 可归档——读 tasks.json 校验；非 success → 409 state_conflict；已归档幂等 204）
   - 新增 `POST /api/workspaces/{ws}/jobs/{job}/unarchive` → 204（幂等）
   - Deps 加 `Archived *ArchivedStore` 注入（handler.Web 组装处 NewArchivedStore 加载）
4. `internal/web/archived_test.go` + routes 测试：归档/恢复/幂等/过滤（filtered 列表不含归档）/非 success 409/include_archived 标记

# 测试方法
go build ./... + go test ./internal/web/...（含新测试）；重启 6173 验证实例（pkill -f '[p]ort 6173' + /tmp/start-verify.sh）后 curl：归档一个 success job → 列表不含它 → include_archived=true 含且 archived:true → unarchive 恢复

# 上下文提示
- web.go 已有 WebStateDir/WebConfigPath（HOME 跟随）——ArchivedPath 同风格
- 归档是软归档（只隐藏不删文件）——rick 命令（dream/learning）照常扫描 jobs，互不影响
- 前端并行 task 会调这三个接口（契约：archive/unarchive → 204；jobs 列表含 archived 字段；include_archived 参数）
