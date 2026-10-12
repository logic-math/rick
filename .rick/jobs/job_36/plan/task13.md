# 依赖关系
task11, task6, task8, task3

# 写域
internal/web/server.go
internal/web/routes.go
internal/web/static.go
internal/web/watcher.go
internal/web/server_test.go

# 任务目标
Web server 组装：路由表挂载 + 静态资源（embed baseline/覆盖层 fallback）+ fs watcher + ServerConfig。

# 关键结果
1. `internal/web/routes.go`：`RegisterRoutes(mux *http.ServeMux, deps Deps)`——按 api-contract.md 挂全部路由（config/workspaces/sessions 全命令面/jobs/knowledge/events(SSE)/web customize/reset + 静态）；GET /api/workspaces 响应由 routes 层 join ListJobs 补 `jobs_count`（读取失败计 -1 或省略字段）；`Deps struct{Hub, Sessions *SessionManager, Workspaces *WorkspaceRegistry, Jobs deps..., Token string, Static http.Handler, Version string}`（Version 由 task14 的 NewWebCmd(version) 注入 → /api/config 的 rick_version）；错误统一 JSON（WebError→status 映射 helper）；404 JSON；CORS 不开（同源部署）
2. `internal/web/static.go`：`StaticHandler(embedFS fs.FS, overrideDir string) http.Handler`——overrideDir/index.html 存在→服务 overrideDir 全部（目录穿越清洗）；否则 embedFS；index.html/manifest/sw.js no-cache，/assets/* 长缓存（immutable）；SPA fallback（未知 GET 非 /api/ → index.html）；`//go:embed` 经参数传入（import webassets "github.com/sunquan/rick/web" 在 cmd 层做——本包收 fs.FS 接口保持可测）
3. `internal/web/watcher.go`：`StartWatchers(ctx, hub, paths...)`——**自实现轮询 watcher（2s mtime）**：监听 `~/.rick/web/dist`（变更 debounce 500ms→Hub frontend_reload 事件）；**监听目录不存在→跳过并轮询等待目录出现，不得 fatal**；v1 禁止引入 fsnotify 等新依赖（不改 go.mod）+ 各注册工作区 `<ws>/.rick/jobs/*/doing/tasks.json`（变更→读 diff→jobs_update 事件；工作区增删时 watcher 动态增删）；轮询兜底（fsnotify 不可用时 2s 轮询 mtime——保持零外部依赖可降级：fsnotify 引入与否按 go.mod 现状决定，倾向轻量自实现轮询+可选 fsnotify）
4. `internal/web/server.go`：`type ServerConfig struct{Addr string; Token string; WebFS fs.FS; StateDir string}` + `func NewServer(cfg, deps...) (*http.Server, error)` 组装 + `Serve(ctx)` 优雅关闭（ctx cancel→Shutdown 10s）+ 启动打印（listening 地址+token 提示+workspace 数）；健康端点 GET /api/health（无鉴权——探活用）
5. `internal/web/server_test.go`：httptest 全路由 smoke——401 矩阵（无 token/错 token/Bearer/query）；workspaces CRUD 往返；静态 embed fallback 与 overrideDir 优先级（tmpdir 造 override）；SPA fallback（GET /foo → index.html 内容）；health 200；SSE 建立即收 server_info；customize/reset 端点接 env 层（本 task 以函数注入 `CustomizeFunc/ResetFunc` 解耦——env 实现在 task14）
6. `go build ./...` + `go test ./internal/web/... -timeout 300s` 全包绿（此前各 task 测试合流回归）

# 测试方法
go test ./internal/web/... -timeout 300s -v 全绿；go build ./... 

# 上下文提示
- 路由统一 /api/ 前缀；http.ServeMux（Go1.22 pattern 语法可用 "POST /api/sessions"）
- 契约补充（与 task12 对齐）：POST /api/web/customize → {ok:true,scaffolded:true|false}；POST /api/web/reset → {ok:true}
- watcher 的 tasks.json diff 逻辑可复用 handler.doing 的 watchTasksJSON 思路（读快照 diff）——但写域限制在 internal/web/ 内自实现轻量版
