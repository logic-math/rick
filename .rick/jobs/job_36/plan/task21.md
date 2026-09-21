# 依赖关系
task19

# 写域
internal/cmd/tools_release.go
internal/cmd/tools_release_test.go
internal/env/release/
internal/env/release/release_test.go

# 任务目标
交付 `rick tools release`：把 dev 树的产物（二进制 + dist）**原子提升**为生产、重启生产、并保证刷新页面依旧连通；支持 `--dry-run` 与 `--rollback`。这是 KR3 的唯一入口（**人类执行命令 = 人类确认**，human 裁决 J-L5-2/J-L6-5）。

# 关键结果
1. `internal/env/release/`（新包，纯逻辑可测）：
   - `type Plan struct { ProdRepo, DevTree, ProdHome, Version string; ... }`；`Build(plan) (Release, error)`：跑门禁（`go test ./...`、`npm run build`）→ 产出 `$PROD_REPO/bin/releases/<version>/{rick,dist}`（`version=<sha7>-<YYMMDDHHMMSS>`；`bin/` 已 gitignore，产物不入库）。
   - `Promote(plan, rel) (Result, error)`：① 备份当前 `current` 链目标到 `bin/releases/.last`；② `ln -sfn <version> .tmp && mv -T .tmp bin/releases/current`（**原子换链**，实测 `mv -T` 是原子替换——research-L6 §1.3）；③ `bin/rick` 改为指向 `releases/current/rick` 的**相对软链**（保持 `start-web.sh` 的 `./bin/rick` 可用）；④ 前端同版推进：把 `releases/<ver>/dist` 原子投放到 `$PROD_HOME/.rick/web/dist`（`mv dist dist.prev && mv dist.new dist`，保留 `dist.prev`）；⑤ 保留最近 3 版，GC 更早版本。
   - `Restart(plan) (Result, error)`：停生产（读 `$PROD_HOME/.rick/web.pid` → TERM → ≤12s → KILL；**必须**用 `start-web.sh` 的语义重启：`setsid nohup <prod>/bin/rick web --listen 0.0.0.0 --port 8413` 或复用 `$PROD_HOME/start-web.sh`）→ 健康轮询 `GET /api/health` ≤20s → **校验 `build_id` == 本次版本**（判定「新构建真的在跑」）→ 打印恢复报告（挂起清单，来自 task19 的 `recovery-report.json`）。
   - `Rollback(plan) (Result, error)`：把 `current` 切回 `.last`（`mv -T` 原子）→ 前端 dist 回 `dist.prev` → 重启 → 健康校验。
   - `DryRun(plan) (Result, error)`：只做门禁 + 构建 + 计划打印，**不动生产、不重启**。
2. `internal/cmd/tools_release.go`：`rick tools release [--yes] [--rollback] [--dry-run] [--detach]`
   - 无 `--yes` 时交互确认（打印版本、门禁结果、将执行的 5 个步骤、影响面「生产将重启，所有在跑会话将变为挂起，可一键恢复」→ `[y/N]`）；
   - **自保**：默认检测「当前进程是否由生产实例托管（env 含 `PI_SESSION_ID` 且父链指向 prod web）」→ 若是，提示「本命令会重启承载你的实例」并要求 `--yes` 或建议 `--detach`（`setsid` 脱离进程组后执行，避免自己被杀导致提升半途而废）；
   - 退出码 `{0 ok, 2 gate fail, 3 build fail, 4 restart fail, 5 health/build_id fail}`；输出结构化回执（供 AI 会话解析）。
3. 测试 `internal/env/release/release_test.go`（`t.TempDir()` 造「假生产」：临时 repo + 临时 home + 假二进制 + 假 start 脚本 + 假健康端点）：换链原子性与 `current` 目标正确；`.last` 记录与回滚；保留 3 版 GC；门禁失败 → 不触碰生产；健康失败 → 自动回滚；`build_id` 校验失败 → 失败并回滚。
4. `internal/cmd/tools_release_test.go`：flag 解析与退出码映射（用注入的假实现）。

# 测试方法
go test ./internal/cmd/ ./internal/env/release/ -timeout 900s
go build ./... && go vet ./internal/cmd/ ./internal/env/release/
python3 .rick/jobs/job_36/plan/gates/gate9.py

# 上下文提示
- 调研依据：research-L6 §1（`mv` 覆盖运行中二进制原子且安全、`cp` 会 ETXTBSY、`~/.rick/bin/rick` 是遗留、生产二进制真身在 `<prod>/bin/rick`）、§2（kill 重启 + SSE 自愈；有 SSE 时优雅退出 ~10s）、§4（门禁清单）、§5（现无 approve 语义）。
- **绝对纪律**：gate 与测试**只对「模拟生产」（临时目录 + 临时端口）操作**；任何代码路径都不得在测试中触碰真实 `~/.rick` 与 8413（用注入的 `Plan` 参数隔离）。
- 与 task19 契约：`Restart` 之后读 `<prodHome>/.rick/recovery-report.json` 汇总挂起清单打印；不自动恢复会话（human 裁决）。
