# task11：Go 命令面（cmd + handler + session）

## 任务目标

打通 `rick super` CLI：session（中控会话，照 easy 模式默认加载 super-loop + gen_env skill）/ start [--daemon] / stop / status / event 五个子命令 + super 会话 prompt 构建。Go 侧只编排与监督，不碰进化逻辑（py 调度器的事）。

## 关键结果

- 新建 `internal/cmd/super.go`：`func NewSuperCmd(version string) *cobra.Command`——默认子命令 session；start（flags: --name/--env/--daemon/--max-wall-time/--max-iterations/--adjust-interval）、stop（--name）、status（--name/--json）、event（--name/--type/--payload）；照 easy.go 模板（56 行范式：RunE→handler 转发）
- 新建 `internal/handler/super.go`：
  - `func SuperSession(opts Options) error`——中控会话：BuildSuperPrompt → pi 交互会话（照 easy.go RunInteractive 同型）
  - `func SuperStart(opts Options, name, env string, daemon bool, cfg overrides) error`——工作区初始化（调 `python3 -m superopt.config --init`？**取 Go 直做**：Go 侧建目录骨架 + super.json（复用 py config.py 的 DEFAULTS 值——Go 写一份同值默认，双语言默认一致性由 gate4 断言）→ spawn PySupervisor（传 --preverify-red：调度器先门先证红，失败即退出非 0，start 返回错误「门先证红失败: leaked=[...]」）；daemon 模式 re-exec 自身（--daemon 内部 flag，照 web DoingRunner --daemon 先例）
  - `func SuperStop(opts Options, name string) error`——读 super.json pid → 优雅停（SIGTERM 落完当前事件 → grace → SIGKILL）→ super.json state=stopped
  - `func SuperStatus(opts Options, name string, jsonOut bool) error`——读 super.json + heartbeat.json + events.jsonl 尾（迭代游标/北极星摘要：入格数/总迭代/token 花费——从事件流聚合）+ supervisor 状态；--json 输出机器可读
  - `func SuperEvent(opts Options, name, etype, payload string) error`——写 `<ws>/interventions/` 队列文件（时间戳命名 JSON；**不直写 events.jsonl**）
- 新建 `internal/super/session.go`：`func BuildSuperPrompt(rickDir, ws string) (methodFile, promptFile string, err error)`——LoadLoopsContext（super-loop 会在其中）+ LoadSkillsContext + super-loop inline + gen_env skill inline + 中控任务说明（env 建设引导/启停/监控/干预 SOP 摘要 + CLI 速查）
- 修改 `internal/cmd/root.go`：AddCommand(NewSuperCmd(version)) 一行
- 修改 `internal/config/config.go`：SuperConfig 嵌套结构体 {DefaultSuperDir, PythonPath, HeartbeatIntervalMs, HeartbeatTimeoutMs, TermGraceMs, AdjustInterval, DebugMaxAttempts}（omitempty 先例，照 HumanLoopConfig）
- 新建 `internal/super/supervisor_test.go` 已在 task10（本 task 不动）；本 task 测试进 `internal/handler/super_test.go` + `internal/cmd/` 编译级：
  - SuperStart（mock python：fake superopt 脚本写心跳+写事件后常驻）→ status 读到 running/心跳 → stop 优雅停 → super.json state=stopped
  - SuperEvent 写入 interventions/ 队列文件（内容/命名断言）
  - BuildSuperPrompt：产出含 super-loop 与 gen_env 内容、无未替换变量
  - config 默认值与 py DEFAULTS 一致性（adjust_interval=5/debug_max_attempts=3/k=3 断言）
  - start 的 --preverify-red 失败路径：fake superopt 退出非 0 → SuperStart 返回错误含「门先证红」

# 写域

- internal/cmd/super.go（新建）
- internal/handler/super.go（新建）
- internal/handler/super_test.go（新建）
- internal/super/session.go（新建）
- internal/cmd/root.go（修改：AddCommand 一行）
- internal/config/config.go（修改：SuperConfig 结构体）

## 依赖关系

- task10（Kahn L2；import internal/super.PySupervisor）
- 与 task5/6/7 同层并行（写域互斥：本 task 全在 internal/，py 侧在 super/）——集成（真 superopt 跑通）在 task14

## 实现要点

- 测试用 fake superopt py 脚本（不是真 superopt——并行层解耦；真集成 gate4）
- super 会话 = 照 easy：SavePrompt 落 workspace prompts + RunInteractive
- status 北极星聚合直接读 events.jsonl 尾 N 行（Go 读 jsonl 逐行 parse——不调 py）
- heartbeat.json 读用 task10 的 PyWorker.LastHeartbeat 复用

## 测试方法

`go build ./... && go test ./internal/super/... ./internal/cmd/... ./internal/handler/... -timeout 120s` 全绿；`rick super --help` 列五子命令（临时 bin 冒烟）

## 禁止事项

- 不得在 Go 里实现任何进化逻辑（采样/门禁/档案——py 侧职责）；不得碰 internal/web/**（J5：本期 CLI only，web 留下个 job 但不改现有 web 代码）
