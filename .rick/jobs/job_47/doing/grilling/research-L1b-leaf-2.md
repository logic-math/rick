# L1b-leaf-2 supervisor.go 原语盘点（信源=代码直读）

internal/runtime/supervisor.go(877L)：pi `--mode rpc` 子进程监督器（web 会话层的进程管理半，rpc.go=协议半）。与 pi rpc 协议耦合，非通用监督器。

## API（行号）
- L39 var BootstrapMessage；L42-70 常量 DefaultMaxActive=64/IdleDisabled=-1s/心跳 30s/Probe 500ms/TermGrace 5s
- L75 ErrMaxActive；L78 ErrEmptySessionFlag
- L82 SupervisorConfig{MaxActive,IdleTimeout,HeartbeatInterval,HeartbeatTimeout,ProbeTimeout,TermGrace,PiPath,ExtraArgs,OnDead func(sessionID,lastEntryID string)}
- L111 SpawnSpec{SessionID,Dir,MethodFile,PromptFile,SessionIDFlag,CreateNew}；L131 WorkerState；L140 Supervisor；L375 Worker
- L148 NewSupervisor(cfg)（零值取默认）
- L172 Spawn(SpawnSpec)(*Worker,error)：pi --mode rpc；L228 Setpgid；probe；并发上限；scan/pump/heartbeat/reap 四 goroutine
- L329 Get；L336 Count；L353 Remove；L360 CloseAll
- L412 SessionID；L415 IsDead；L425 Reason；L434 Dropped；L438 Events()<-chan *RpcEvent；L441 StderrTail；L445 State；L457 LastEntryID；L466 Send([]byte)error
- L504 Close()：abort→L519 kill(-pgid,SIGTERM)→5s→L525 kill(-pgid,SIGKILL)；幂等
- 私有：L552 markDead→单次 OnDead(sessionID,lastEntryID)；L786 heartbeatLoop（get_state 超时判死）；L820 reapLoop；L854 byteRing(4KB stderr)

## 生产调用方
- 唯一组装根 internal/cmd/web.go:236 NewSupervisor(resolveSupervisorConfig(cfg))；:293-312 PiPath/ExtraArgs←config、IdleTimeout=IdleDisabled、MaxActive←cfg.WebMaxActive
- internal/web/sessions.go：Spawn L452/1005/1158；Get/IsDead L671/717/1201；Send L460/646/1239；Remove L771；State L682；Events L1777（关→L1811 markWorkerLost）；CloseAll L1904
- internal/web/recovery.go:245 LastEntryID→关停快照（须先于 CloseAll，:221）
- 未接线：OnDead（组装根 nil）；Count/StderrTail/Dropped/Reason 仅测试

## 测试（21 例，fake pi 脚本 via PiPath+RICK_PI_AGENT_DIR）
L341 孙进程收割实证；L445 外杀→OnDead 携 lastEntryID 单次；L604 心跳超时判死；L285/316 优雅/强杀；L501/545 idle 回收/禁用；L632 溢出丢旧；L681 flag 语义；L734 Dir+Env；L196 probe；L800/856 超长行

## 结论
1. RFC 六原语（Setpgid/心跳/超时/整树收割/OnDead 游标/stderrTail）全部真实且有测试，孙进程收割有实证。
2. Spawn 硬编码 pi rpc 协议（心跳=get_state），监督 py 进程组需新变体或 py 实现同协议。
3. OnDead 生产未接线：web 走 Events 关闭判死+关停快照；super 可复用 OnDead+get_entries(since) 重放。
