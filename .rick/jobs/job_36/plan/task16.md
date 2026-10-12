# 依赖关系
无

# 写域
internal/web/statedir.go
internal/web/web.go
internal/web/registry.go
internal/cmd/web.go
internal/handler/web.go
internal/web/statedir_test.go
internal/web/registry_test.go
internal/handler/web_test.go

# 任务目标
把「web 状态目录」从隐含的 HOME 派生变成**显式契约**，并用跨进程锁把 singleton 从「pid 文件尽力而为」升级为「file lock 强约束」；同时加上 dev 实例的 fail-fast 隔离守卫。这是 KR1 的地基：调研实测（research-L4 §4）当前生产 `~/.rick/web.pid` 竟然不存在，同 HOME 起第二实例会被放行，并因 `ReconcileOnStart` 把生产 active 会话改写成 error 落盘。

# 关键结果
1. `internal/web/statedir.go`（新）：
   - `ResolveStateDir(flagValue string) (string, error)`：优先级 `--state-dir` flag > 环境变量 `RICK_STATE_DIR` > `$HOME/.rick`；返回绝对路径（不存在则 MkdirAll）。
   - `AcquireStateLock(stateDir string) (release func(), err error)`：对 `<stateDir>/web.lock` 取 `syscall.Flock(LOCK_EX|LOCK_NB)`；已被占用 → 返回错误（含「另一个 rick web 正在使用该状态目录」中文提示 + 尽力读出占用者 pid）。release 幂等（解锁 + 关闭 fd）。
   - `IsDevStateDir(stateDir string) bool`：`stateDir != $HOME/.rick`（显式指定 = dev 语义）。
2. `internal/web/web.go`：`WebStateDir()/WebConfigPath()/SessionsPath()/ArchivedPath()/JobNamesPath()/PidPath()` 全部改为「读已解析的 state dir」——新增包级 `SetStateDir(dir)`（进程启动时调用一次，未调用时保持现有 HOME 派生行为，**既有测试与默认行为零变化**）。
3. `internal/web/registry.go`：`WorkspaceRegistry.Add` 增加守卫钩子 `SetAddGuard(func(path string) error)`；dev 实例（`IsDevStateDir` 为真）注册时校验：**拒绝注册「生产注册表（`$HOME/.rick/web.json`，只读）中已存在的工作区路径」**（防 dev 会话写生产 workspace 的 `.rick`）；非 dev 实例不设守卫（零行为变化）。
4. `internal/cmd/web.go`：新增 `--state-dir` flag；启动序列 = `ResolveStateDir` → `AcquireStateLock`（失败 → 非零退出 + 中文提示）→ 组装；**隔离守卫**（fail-fast）：
   - `--state-dir` 显式指定（dev 模式）时，必须同时显式给出 `RICK_PI_AGENT_DIR`，否则拒绝启动并说明理由（否则 web 状态隔离了、pi 沙盒仍指向生产 → dev 会覆写生产 145MB runtime）。
   - dev 模式下打印显著 banner（`[rick-web] DEV MODE state-dir=... agent-dir=...`）。
5. `internal/handler/web.go`：pid 文件路径改用同一个已解析 state dir（消除调研指出的「两处 pid 路径都硬编码 `.rick`」的第二处）。
6. 测试 `internal/web/statedir_test.go`：ResolveStateDir 优先级表驱动（flag > env > HOME）；AcquireStateLock 同目录二次获取失败、release 后可再获取；`t.Setenv` 隔离（勿碰真实 `~/.rick`）。
7. `internal/web/registry_test.go` + `internal/handler/web_test.go`：Add 守卫拒绝/放行（放行时行为与旧版一致）；pid 路径随 state dir。

# 测试方法
go test ./internal/web/ ./internal/handler/ ./internal/cmd/ -timeout 600s
go build ./... && go vet ./internal/web/ ./internal/cmd/
python3 .rick/jobs/job_36/plan/gates/gate7.py

# 上下文提示
- 调研依据：research-L4.md §1（路径归属清单）、§4（singleton 机制/4 个探针/现网 pid 缺失）、§7.2（残留泄漏点 2 与 4）。
- 兼容性红线：**不得改变默认（无 flag/env）行为**——生产就是默认路径，任何变化都可能影响运行中的 prod；dev 语义只在显式指定时启用。
- flock 需 `syscall`（Linux）；不要在 `internal/web` 里引入平台分支复杂度（本项目仅 Linux 云 IDE）。
- ⚠️ 本 task 只做「能力与守卫」，不要顺手改 sessions 的恢复语义（那是 task19）。

---

# 补丁需求 F1（由 task22 的 E2E 实测暴露，2026-09-21 复核确认）

**缺陷**：隔离守卫的启用条件写成 `IsDevStateDir(resolved)`（= `resolved != $HOME/.rick`），只在 `--state-dir` 形态下成立。而 `rick tools dev-web` 实际启动 dev 实例的形态是**换 HOME**（`HOME=<dev-home>`，状态目录仍是默认的 `$HOME/.rick`）→ `IsDevStateDir` 为 false → **守卫未安装**。

**实测复现**（换 HOME、无 --state-dir）：
```
POST /api/workspaces {"path":"/workdir/sunquan20/BERT_KEETA", ...}  → HTTP 201（本应 4xx 拒绝）
```
后果：dev 会话可注册并写入**生产工作区**的 `.rick`（jobs/tasks.json）——L4-6 的 fail-fast 纪律被绕过。

**要求**：
1. 新增谓词 `web.DevModeInfo(resolved string) (isDev bool, prodState string, homeSwapped bool)`：
   - `prodState` = `RICK_PROD_STATE_DIR` > 真实用户家目录（`user.Current()`）的 `.rick`；
   - `isDev` = `Clean(resolved) != Clean(prodState)`（**两种形态都算 dev**：换 HOME 后默认状态目录 ≠ 生产状态目录）；
   - `homeSwapped` = 当前 `$HOME` ≠ 真实用户家目录（= `ProductionStateDir()` 非空）。
2. `internal/cmd/web.go`：
   - `isDev` 时**一律**安装工作区守卫（`ProdWorkspaceGuard(prodState)`）+ 打印 DEV MODE banner（含形态说明：`home-swapped` / `state-dir-only`）；
   - **`RICK_PI_AGENT_DIR` 强制要求只在 `!homeSwapped` 时生效**（状态目录隔离了但 HOME 没换 → pi 沙盒仍指生产）；换 HOME 形态下 pi 沙盒已随 HOME 隔离，不强制。
3. 测试：`internal/web/statedir_test.go` 增加 `DevModeInfo` 表驱动（生产 HOME+默认 / 生产 HOME+--state-dir / 换 HOME+默认 / 换 HOME+--state-dir，用 `RICK_PROD_STATE_DIR` 与 `t.Setenv` 隔离）；`internal/cmd/` 至少补一条「换 HOME 形态下守卫已安装」的断言（可用 fake registry/guard 注入）。
4. `plan/gates/gate7.py` 增加**换 HOME 形态**断言（第 ⑨ 条）：换 HOME（不给 --state-dir）启动 → `POST /api/workspaces` 注册生产注册表里的工作区路径 → 必须 4xx 拒绝；同时断言 dev 会话列表为空（隔离成立）。
