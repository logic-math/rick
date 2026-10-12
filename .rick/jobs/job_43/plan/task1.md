# task1：pi 会话持久化（Go 后端）

## 任务目标

web 会话的 pi worker 改为持久持有：只要 server 存续、用户不主动断开，worker 就不被空闲回收；并发上限可配置（默认 64）；心跳杀保留（真卡死进程仍会被清理）。版本号提升至 5.0.12。

## 关键结果

- KR1a：`internal/cmd/web.go` 组合根不再使用 30m 默认空闲回收——传负值（或等价显式禁用语义）禁用 reapLoop；`internal/runtime/supervisor.go` 补注释澄清负值语义
- KR1b：`internal/config` 新增 `web_max_active` 字段（缺省/0 → 64）；`internal/cmd/web.go` 将其接入 `NewSupervisor` 的 MaxActive
- KR1c：心跳超时路径**不动**（HeartbeatTimeout 语义保持）
- KR1d：`cmd/rick/main.go` VERSION → "5.0.12"
- KR1e：新增 Go 测试：①负 IdleTimeout 不启动回收（worker 空闲后仍存活）；②MaxActive 解析（缺省 64 / 配置值生效）——建议把 supervisor 配置解析抽为可测函数（如 `resolveSupervisorConfig`），单测覆盖

# 写域

- internal/runtime/
- internal/config/
- internal/cmd/web.go
- internal/cmd/web_test.go
- cmd/rick/main.go

# 依赖关系

无

## 实现要点

- 组合根现状：`internal/cmd/web.go:236-240`（`runtime.NewSupervisor(runtime.SupervisorConfig{PiPath, ExtraArgs})`，注释说明默认 8/30m/30s）
- `internal/runtime/supervisor.go:140-141`：`IdleTimeout==0 → DefaultIdleTimeout(30m)`；负值原样保留，`:310-311` 附近 `idle > 0 才启动 reapLoop`——负值即禁用，已有语义，无需改逻辑，只补注释
- MaxActive 建议单一来源：`DefaultMaxActive` 调整为 64（同步注释）或组合根解析缺省 64——二选一，保证「缺省=64，config 可覆盖」
- config 结构见 `internal/config/config.go`（已有扁平 `web_token` 先例，新字段 `web_max_active` 风格一致）
- 现有 `TestSupervisor_IdleReap`（supervisor_test.go）显式传值不受影响，勿破坏
- 测试用小超时驱动（如 IdleTimeout: 100ms 的对照组 + 负值组断言 worker 存活），勿真等 30m

## 测试方法

- `go build ./...` 绿
- `go test ./internal/runtime/ ./internal/config/ ./internal/cmd/` 全绿（含新增测试）
- 手动推演：config.json 无 web_max_active → 生效 64；写 4 → 生效 4；IdleTimeout 负值 → reapLoop 不启动

## 禁止事项

- 只改写域内文件；不碰 web/ 前端；不碰心跳语义；不执行任何 git 操作（层检查点由 parent 统一提交）；一切改动只在 dev 树（/workdir/sunquan20/rick-dev）
