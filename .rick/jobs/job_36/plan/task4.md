# 依赖关系
task1

# 写域
internal/runtime/supervisor.go
internal/runtime/supervisor_test.go

# 任务目标
pi rpc 子进程 supervisor：per-session worker 生命周期管理（spawn/心跳/优雅终止/崩溃恢复/上限回收）。

# 关键结果
1. `internal/runtime/supervisor.go`：
   - `type Supervisor struct` + `NewSupervisor(cfg SupervisorConfig)`：`SupervisorConfig{MaxActive int(默认8), IdleTimeout(默认30min), HeartbeatInterval(默认30s), PiPath string, ExtraArgs []string}`
   - `type SpawnSpec struct{ SessionID string; Dir string; MethodFile string; PromptFile string; SessionIDFlag string; CreateNew bool }`（CreateNew=true 用 `--session-id`（创建新 pi 会话），false 用 `--session`（resume 已有——bugs.md 语义差异）
   - `(s *Supervisor) Spawn(spec) (*Worker, error)`：args 切片拼接（`[]string{"--mode","rpc"}` + extraArgs + systemPrompt flags + sessionFlag——注意 Go 语法不允许 ... 后跟参数）；`cmd.Env=AgentEnv()`；**Setpgid 进程组**；三管道（stdin pipe / stdout pipe / stderr drain goroutine）；spawn 后 500ms 探活（exitCode+stderr 缓存——subagent/index.ts 范本）
   - `(w *Worker)`：`Send(cmd []byte) error`（写 stdin，带 mutex）、`Events() <-chan *RpcEvent`（stdout 行扫描 goroutine → ParseEventLine → channel）、`Close()`（abort→SIGTERM(进程组 -pgid)→等 5s→SIGKILL；幂等）、`State()`（最近 get_state 结果缓存）、`LastEntryID() string`
   - 导出别名：`// BootstrapMessage re-exports the CLI bootstrap trigger for web sessions.
   var BootstrapMessage = bootstrapMessage`（cli.go:63 未导出常量，task11 web 会话 bootstrap 需要——同包引用零重复）
   - 心跳 goroutine：每 HeartbeatInterval 发 GetState 命令，超时/进程退出 → 标记 worker dead → 回调 `OnDead(sessionID, lastEntryID)`（supervisor 不自动 respawn——由 web 层决策，避免隐藏状态）
   - idle 回收：LastActivity 超过 IdleTimeout 且非 streaming → 自动 Close + 回调
   - 并发安全：workers map 带 RWMutex；channel 缓冲 256，满则丢最旧事件并标记 overflow（记日志）
2. `internal/runtime/supervisor_test.go`：**fake pi 脚本**测试（tmpdir 写 shell 脚本模拟 pi：读 stdin 行、对 get_state 回 canned JSON 响应、可控退出）——覆盖：spawn 探活失败（脚本立即 exit 1）；命令往返（Send get_state → Events 收到 response）；Close 的 SIGTERM 优雅路径（脚本 trap TERM 后退出）与强杀路径（脚本忽略 TERM）；多 worker 并发 spawn/close；MaxActive 上限拒绝（返回 ErrMaxActive）
3. `go build ./...` + `go test ./internal/runtime/ -run TestSupervisor -timeout 120s` 通过
4. ⚠️ 测试隔离：`t.Setenv("RICK_PI_AGENT_DIR", t.TempDir())` 防命中真实托管 pi（bugs.md job_34 坑位）；fake 脚本开头恢复系统 PATH（bugs.md job_33 坑位）

# 测试方法
go test ./internal/runtime/ -run TestSupervisor -v（fake pi 全绿）；ps 检查无残留 pi 进程（测试后 pgrep -f "mode rpc" 应空）

# 上下文提示
- 直接移植 pi 官方 subagent/index.ts 的进程管理范本（SIGTERM→5s→SIGKILL/并发上限/50KB 截断/启动探活——research-L1-r2.md 结论）
- Go1.20+ exec.Cmd 有 Cancel/WaitDelay 可简化优雅终止，但进程组需要自己 syscall.Setpgid（bash 工具会生孙进程，必须杀组）
- RpcClient 在 task1 已就绪（internal/runtime/rpc.go），直接复用其命令构建与解析
