# 依赖关系
task16

# 写域
internal/cmd/tools_dev_web.go
internal/cmd/tools_dev_web_test.go
internal/cmd/tools.go
internal/env/devweb/
internal/env/devweb/devweb_test.go

# 任务目标
交付 `rick tools dev-web {init|build|up|restart|status|down}`：一键起/停/重建**隔离的 dev 实例**，并把「构建 → 停旧 → 起新 → 健康 + 指纹校验」做成幂等、可被 AI 会话直接调用的闭环（KR2 的可操作面）。调研依据：research-L5 §2（五方案对比：进程重启 + 唯一二进制名 + 健康轮询是最低成本可行解）、§4（反馈回路：脚本 exit code + 指纹，别让 AI 依赖 SSE）、§5（构建依赖：HOME 覆写会让 Go/npm 缓存失效，必须显式固定）。

# 关键结果
1. `internal/env/devweb/`（新包，纯逻辑、可测）：
   - `type Layout struct { Tree, Home, AgentDir, Port, Token string }` + `DefaultLayout(prodRepo string) (Layout, error)`：`Tree=$HOME_PARENT/rick-dev`（默认 `/workdir/sunquan20/rick-dev`，可用 `RICK_DEV_TREE` 覆盖）、`Home=.../rick-dev-home`、`AgentDir=<Home>/.rick/pi/agent`、`Port=8414`、`Token` 读 `<Home>/DEV_TOKEN`（不存在则生成 `devtok-<8hex>` 并 chmod 600）。
   - `Init(l Layout, prodRepo string) error`：幂等——缺 worktree 则 `git worktree add -b dev/<...> <tree> HEAD`（tree 已存在则跳过）；建 `<Home>/.rick/pi/agent` 并从生产 `<prodHome>/.rick/pi/agent` 拷 `auth.json/settings.json/models-store.json` 种子（chmod 600，已存在不覆盖）；建 overlay 软链 `<Home>/.rick/web/dist → <Tree>/web/dist`（缺目录则先 npm build 或跳过）；写 `<Home>/dev.env`（记录 Tree/Home/AgentDir/Port/Token 供脚本与人读）。
   - `Build(l Layout, goCache, goModCache string) (bin string, err error)`：`npm run build`（web/，缓存 `npm_config_cache` 指向生产 `.npm`）+ `go build -o <Home>/bin/rick.dev.<sha7>-<HHMMSS> ./cmd/rick`；返回唯一路径（**文件名即构建指纹**）；失败返回 stderr 尾部（≤2KB）。
   - `Up(l Layout, bin string) (result UpResult, err error)`：停旧（读 `<Home>/.rick/web.pid` → TERM → 最多 12s → KILL；无 pid 时按 `pgrep -f "<Home>"` 兜底）→ `setsid nohup` 起新（env 白名单：`HOME/ RICK_PI_AGENT_DIR/ RICK_STATE_DIR=<Home>/.rick/ GOCACHE/GOMODCACHE/npm_config_cache/PATH`）→ 健康轮询（`GET /api/health` ≤15s）→ **指纹校验**（`/api/config` 的 `build_id` 与 bin 内注入值一致，task18 提供；未实现时降级为比对 `/proc/<pid>/exe` 的 sha256）。
   - `Down(l Layout) error`、`Status(l Layout) (StatusResult, error)`（读 `<Home>/dev-state.json` + 健康 + 指纹）。
   - `UpResult {Bin, PID, HealthMS, BuildID, LogTail}`；状态文件 `<Home>/dev-state.json`：`{bin, sha7, build_id, pid, built_at, health_ms, port}`。
2. `internal/cmd/tools_dev_web.go`：`rick tools dev-web` 子命令（`init|build|up|restart|status|down`），退出码语义 `{0 ok, 2 build fail, 3 start fail, 4 health fail}`，输出**一行回执**（`DEV_UP bin=... pid=... health_ms=... build_id=...` / `DEV_FAIL stage=... detail=...`）供 AI 会话解析；`up --no-build` 复用最近二进制；`status --json`。
3. `internal/cmd/tools.go`：挂载子命令（与既有 tools 子命令风格一致）。
4. 测试 `internal/env/devweb/devweb_test.go`（`t.TempDir()` 全隔离，勿碰真实 worktree）：Layout 默认值与 env 覆盖；Token 生成持久化；Build 的失败路径返回 stderr 尾部（用 `PATH` 注入假 npm/go 脚本，脚本需恢复 PATH）；Up 的停旧逻辑（构造假 pid 文件 + 假进程）；状态文件写入字段完整性。

# 测试方法
go test ./internal/cmd/ ./internal/env/devweb/ -timeout 600s
go build ./... && go vet ./internal/cmd/ ./internal/env/devweb/
python3 .rick/jobs/job_36/plan/gates/gate8.py

# 上下文提示
- 调研依据：research-L5 §2/§4/§5；research-L4 §3（worktree 命令）、§7（启动配置表：GOCACHE/GOMODCACHE/npm cache 必须显式指向生产缓存，否则冷构建）。
- 关键纪律：**本命令只操作 dev 树与 dev HOME，绝不触碰生产 `~/.rick`（除只读读种子文件）与生产 8413 进程**；任何 TERM/KILL 必须先断言目标 pid 的 cwd/env 属于 dev HOME。
- 与 task18 的契约（并行开发，按此实现）：`build_id` 由 task18 注入二进制并由 `/api/health`、`/api/config` 暴露；本 task 的指纹校验读 `build_id`，缺失时降级 `/proc/<pid>/exe` sha256。
