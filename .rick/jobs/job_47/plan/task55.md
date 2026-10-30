# task55：super stop 进程树收割修复（防孤儿/多实例）

# 任务目标
真缺陷（红卫树 env 实跑实证：stop 后残留 4 个孤儿进程 → 双重启叠加 → 多实例并发写同一工作区，事件 seq 冲突）。`rick super stop` 只按 super.json.pid 单点 kill，未覆盖 daemon 的 supervisor 与 worker 进程树。

# 关键结果
- `internal/handler/super.go` `SuperStopIn`：
  - daemon 形态：**kill 整个进程组** `syscall.Kill(-pid, SIGTERM)`（daemon 用 setsid 启动，pid==pgid）→ 等 `term_grace_ms` + 余量 → 仍存活则 `Kill(-pid, SIGKILL)`
  - 同时读 super.json 的 `worker_pid` 兜底（若组信号未覆盖，单点 TERM/KILL 该 pid）
  - **杀后验证**：轮询两个 pid 都不存活（≤10s），仍未死 → 返回明确错误（含 pid 列表与建议 `ps -p`）
  - 清理：super.json state=stopped（RMW 保 cursor）；锁文件残留（`<ws>/locks/*.lock`）在停后清理（下次 start 的 release_orphans 也会兜底）
- `internal/super/supervisor.go`：确认 daemon 启动用 `Setsid`（若是 Setpgid 则 stop 用 `-pgid`；先 read 确认实际语义并在代码注释写明）
- 测试 `internal/handler/super_test.go`：
  ①fake 进程树（daemon → 子进程 supervisor → 孙 worker 各 sleep）→ stop 后全部消失（用 `kill -0` 验证）②worker_pid 单独存活时被兜底杀 ③不存在的 pid → 幂等返回 stopped ④grace 内不死 → SIGKILL 生效

# 写域
- internal/handler/super.go
- internal/handler/super_test.go
- internal/super/supervisor.go（仅在需要修正 Setsid/Setpgid 语义时）

# 依赖关系
- 无（L1，与 task54 写域互斥：Go vs py）

# 测试方法
`go build ./... && go test ./internal/handler/ ./internal/super/ -timeout 300s` 全绿 + 手工验证（真 daemon 起停后 `ps` 无残留）

# 禁止事项
- 不误杀非本工作区进程（保留所有权断言）；不改 CLI 参数契约
