# 运行时模块（internal/runtime）

## 职责

`internal/runtime` 是三层金字塔第三层「执行」的一员，对 agent runtime（当前只有 pi）的调用逻辑做封装。它不安装 pi、不拼提示词、不维护 dag 调度与门禁。

- 参数解析 + 调用 pi（合并 provider/model/api-key 等 flags）
- 内部校验 session 就绪
- 采集行为轨迹（trace）
- 返回 `(sessionID, trace)`

## Runtime 契约

```go
type Runtime interface {
    Name() string
    Run(methodText string, promptFile string, cfg *config.Config) (sessionID string, trace *Trace, err error)
}
```

```go
type Trace struct {
    SessionID    string
    ToolCalls    []ToolCall
    FinalMessage string
    RawLogPath   string
    Duration     time.Duration
    Settled      bool
}
```

`piRuntime` 是当前唯一实现。`Run` 返回非空 sessionID 表示运行成功；若 JSONL 流从未产生 session id 或从未发出 `agent_settled`（会话未 settle），`Run` 返回错误，由 handler 应用重试安全网。

## 三层注入

`Run` 把方法层（methodText）写临时文件，经 `--append-system-prompt <methodFile>` 注入，保留 pi 默认骨架；promptFile 作为 user prompt（实例上下文）最后传入。

## pi 配置目录隔离

所有 pi 调用入口注入 `PI_CODING_AGENT_DIR=~/.rick/pi/agent`（`AgentEnv()`），与用户 `~/.pi` 完全隔离。

## Web 会话执行层：supervisor + pi daemon worker（2026-11-28 单轨架构）

`supervisor.go` + `daemon.go` 实现 web 会话（rick web 页面里的聊天/后台会话）的 pi 常驻进程管理。CLI 一次性执行（`--mode json`）不走这里。

**单轨 daemon 架构**（旧「直连管道」架构已删除——web 重启杀 pi 的根本缺陷）：

```
Go web（中控状态机，可重启/升级）          ←─ 重启/版本升级只影响这层
   │  unix socket: <AgentDir>/sockets/<session>.sock
   ▼
pi-daemon-shim（node，独立进程组）          ←─ 持有 pi stdin 永不 EOF + 断线环形缓冲 2000 行
   │  stdout/stderr 落 <stem>.log（与 web 进程零共享 FD）
   ▼
pi --mode rpc（工具调用/子代理全程不断）    ←─ 完全无感知
```

四条生命周期路径：

| 路径 | 行为 |
|------|------|
| Spawn | `node daemon_shim.mjs <sock> -- pi --mode rpc ...`（Setpgid 同组：shim+pi+工具+子代理）；活 socket 存在 → `ErrDaemonAlive`（幂等：一会话永远只有一个 pi worker） |
| 重启/升级（ShutdownWorkers） | 只 detach socket 连接，**绝不杀 pi**；注册表 status 保持 active 落盘；shim 缓冲 pi stdout，重连补发（`shim_attached` marker） |
| 启动对账（ReconcileOnStart） | 逐会话探活 socket（dial + `get_state` 端到端验证——防 SIGKILL 后内核 teardown 窗口内的幻影 attach）→ 活 → re-attach + Busy 同步零中断（active/suspended 都探活）；死 → suspended + auto-resume 兜底 + 孤儿清理 |
| 显式 close / 孤儿 | `kill(-pgid)` 整组收割 + 清 `.sock/.meta.json/.log`；`CleanupOrphansExcept` 杀无人认领的孤儿 |

关键事实（实现时踩过的坑，改架构前必读）：
- **pi `--mode rpc` stdin EOF 即退出**——所以必须有 shim 持有 pi 的 stdin/管道，web 重启永远碰不到它们
- **shim/pi 的 stdio 绝不能是连回 web 的管道**——web 死亡时管道断裂，pi 任何一次 stderr 写入就 EPIPE 崩溃（dev 真 e2e 实测：detach 正常执行后 2 秒内 pi 仍死）；必须落日志文件
- Worker 事件管线（scanLoop/intercept/pump/heartbeat）传输层无关——`stdin` 换 `net.Conn` 即整体复用
- macOS `sun_path` 104 字节上限——socket 路径超长是 EINVAL 不是神秘失败（sha256 截断守卫）

配置：`web_daemon_socket_dir`（默认 `<AgentDir()/sockets`，跟 `RICK_PI_AGENT_DIR` 走实现测试隔离）、`web_node_bin`（默认 `node`）。

## pi 解析优先级（全链路一致）

`cfg.PiPath` → 托管运行时（`RuntimeBin()` = `~/.rick/pi/agent/runtime/node_modules/.bin/pi`）→ PATH `pi`。

## 相关文件

| 文件 | 职责 |
|------|------|
| `runtime.go` | Runtime 接口 + piRuntime 实现 + Trace/ToolCall |
| `executor.go` | pi `--mode json` 事件流解析（session/tool/agent_settled） |
| `cli.go` | CallCLI 交互入口 + pi 路径解析 |
| `supervisor.go` | web 会话 supervisor：Worker 生命周期 + 事件管线（scanLoop/heartbeat/pump）+ 组信号收割 |
| `daemon.go` | daemon 单轨传输：spawnDaemon/Attach/DetachAll/CleanupOrphansExcept + socket 路径守卫 |
| `daemon_shim.mjs` | node shim：持有 pi 管道 + unix socket 服务 + 断线环形缓冲 + stdio 落日志 |
| `agentdir.go` | 共享路径工具（AgentDir/RuntimeDir/RuntimeBin/AgentEnv） |
