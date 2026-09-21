# 依赖关系
无

# 写域
internal/cmd/root.go
internal/web/routes.go
internal/web/server.go
internal/web/sse.go
web/vite.config.ts
internal/web/buildid_test.go

# 任务目标
让「新构建是否真的在跑」可被一条命令判定（构建指纹贯穿 `/api/health`、`/api/config`、SSE `server_info`），并把前端开发模式参数化（overlay 软链 + 可选 vite HMR）。调研依据：research-L5 §4（现无任何构建指纹：`/api/health` 只有 `{status:ok}`、`rick_version` 是静态常量）。

# 关键结果
1. `internal/cmd/root.go`：新增包级 `BuildID string`（默认 `"dev"`）+ `FuncVersion()`；由 `-ldflags "-X github.com/sunquan/rick/internal/cmd.BuildID=<sha7>-<ts>"` 注入（`dev-web build` 与 `tools release` 都会带）。
2. `internal/web/routes.go`：
   - `handleHealth` 响应扩展为 `{"status":"ok","build_id":"...","started_at":"..."}`（**保持免认证**与 `status` 字段向后兼容）。
   - `/api/config` 响应增加 `build_id`（与既有 `rick_version/port/auth_required` 并存）。
   - 新增 `GET /api/health` 的兼容断言测试（旧前端只读 `status` 不受影响）。
3. `internal/web/server.go` + `internal/web/sse.go`：`server_info` 事件与 hub 初始化携带 `build_id`（复用既有 `rick_version` 字段位置新增，不改既有字段语义）。
4. `web/vite.config.ts`：dev proxy 目标参数化——`process.env.RICK_DEV_API ?? "http://127.0.0.1:8414"`（原硬编码 6137 是死值）；`server.host` 允许 `0.0.0.0`、`server.port` 可用 `RICK_DEV_VITE_PORT` 覆盖；PWA 插件在 `RICK_DEV_NO_PWA=1` 时 `disable: true`（dev 构建禁用 service worker，避免缓存干扰热更；**不修** SW 的 `non-precached-url` 缺陷——那是独立议题，见 task22 文档）。
5. 测试 `internal/web/buildid_test.go`：health/config 响应含 `build_id` 且与注入值一致；`build_id` 为空时回退 `"dev"`；SSE `server_info` payload 含该字段。

# 测试方法
go test ./internal/web/ ./internal/cmd/ -timeout 600s
go build ./... && npx --prefix web tsc --noEmit -p web
python3 .rick/jobs/job_36/plan/gates/gate7.py

# 上下文提示
- 调研依据：research-L5 §1（overlay all-or-nothing + 缓存策略 + SW 实测失败）、§4（指纹方案）、§5（构建依赖）。
- 兼容红线：`/api/health` 是免认证探针，新增字段可以，**不得**要求 token、不得改 `status` 语义（现有 gate6/E2E 依赖它）。
- 与 task17 的契约：`build_id` 字面值格式 `<sha7>-<YYMMDDHHMMSS>`；task17 的 `Up` 会用它做「新构建在跑」判定。
