# task10：Go PySupervisor 监督模块 + fake py 测试

## 任务目标

新建 `internal/super` Go 包：PySupervisor——监督 py 调度器进程组生命周期（Setpgid 进程组隔离 / 文件心跳监视 / kill 梯 SIGTERM→grace→SIGKILL 整树收割 / OnDead 游标重启回调）。复用既有 supervisor.go 的六原语思想但**新写**（Spawn 硬编码 pi RPC 协议不能给 py 用，J7 裁决）。

## 关键结果

- 新建 `internal/super/supervisor.go`：
  - `type PySupervisorConfig struct { HeartbeatInterval, HeartbeatTimeout, TermGrace time.Duration; PythonPath string; OnDead func(ws string, lastSeq int64) }`
  - `func NewPySupervisor(cfg PySupervisorConfig) *PySupervisor`
  - `func (p *PySupervisor) Start(ws string) (*PyWorker, error)`——spawn `python3 -m superopt --workdir <ws>`（Setpgid=true 进程组；stdout/stderr 管道捕获 tail；cwd=仓库根——superopt 包以 `super/superopt` 相对仓库根可导入；模块路径经 PYTHONPATH 或 cwd 保证）
  - `func (p *PySupervisor) Get(ws string) (*PyWorker, bool)` / `Count() int` / `StopAll()`
  - `type PyWorker struct`——SessionID(ws 名)/StartTime/IsDead()/Reason()/StderrTail(n)/LastHeartbeat()/Close()
  - `func (w *PyWorker) Close() error`——kill 梯：SIGTERM(-pgid) → TermGrace → SIGKILL(-pgid) 整树收割
  - 心跳协议：py 侧写 `<ws>/heartbeat.json` {ts, seq, state}；Go 监视 goroutine 每 HeartbeatInterval 检查 mtime/内容，超 HeartbeatTimeout 判死 → 走 kill 梯 → OnDead(ws, lastSeq)（lastSeq 读 heartbeat.json 的 seq）
- 新建 `internal/super/supervisor_test.go`（照既有 supervisor_test.go 21 用例范式，fake py 脚本）：
  - fake_py_ok.sh：写心跳循环 + sleep——Start→IsDead=false；心跳文件更新；Close→进程组全灭（含子进程——fake 脚本 spawn 一个 sleep 子进程验证整树收割）
  - fake_py_stall.sh：启动后不写心跳——HeartbeatTimeout 到 → IsDead=true Reason=heartbeat_timeout → OnDead 被调且 lastSeq 正确 → 自动重启计数
  - fake_py_exit.sh：立即退出——IsDead=true Reason=exit_nonzero
  - kill 梯时序：TermGrace 内 SIGTERM 生效不升级 SIGKILL
  - 并发多 ws：Count/Get/StopAll
- **不接生产**：本 task 只交付 internal/super 包 + 测试；命令面接线是 task11

# 写域

- internal/super/supervisor.go（新建）
- internal/super/supervisor_test.go（新建）

## 依赖关系

- 无（Kahn L1；与 py 侧零 import 耦合——测试用 fake 脚本，不需要 superopt 真代码）

## 实现要点

- 参照 internal/runtime/supervisor.go 的进程组/kill 梯实现模式（读它，但不 import 其 pi RPC 部分）
- fake 脚本写在 t.TempDir()，注意 macOS bash 差异（bugs.md：fake 脚本恢复 PATH、不用 bashism）
- OnDead 重启**策略**（重启几次/退避）不在本 task——Supervisor 只回调，策略由上层（task11 handler）决定
- 心跳读取要容错 JSON 半写（py fsync 前崩溃）：解析失败=沿用上一次，只有超时才判死

## 测试方法

`go build ./... && go test ./internal/super/... -timeout 120s -v` 全绿（心跳超时用例用短 interval：50ms/200ms 级，测试秒级完成）

## 禁止事项

- 不得修改 internal/runtime/supervisor.go（只读参照）；不得实现 CLI 命令（task11）；不得碰 web/
