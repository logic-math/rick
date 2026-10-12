# research-L5-leaf-2｜端口交接 / 进程自替换 / 进程树（实测）

环境：Go go1.25.0（toolchain 走 module cache）、Linux amd64、pid1=systemd 249、node v24.16.0。
红线遵守：未触碰生产 8413 进程、未改仓库文件、实验全在 /tmp。

## 1 SO_REUSEPORT 是否可用

| 项 | 实测 |
|---|---|
| `syscall.SO_REUSEPORT` 常量 | **不存在**（linux/amd64, Go 1.25）。`zerrors_linux_amd64.go` 只有 `SO_REUSEADDR = 0x2`；编译探针报 `undefined: syscall.SO_REUSEPORT` |
| 硬编码 `0xf` + `net.ListenConfig.Control` + `syscall.SetsockoptInt(fd,SOL_SOCKET,0xf,1)` | 两个 listener 同绑 `127.0.0.1:19702` **都成功**（`both non-nil: true`） |
| 对照组（不加 SO_REUSEPORT） | `[no-reuseport] listen#2: err=listen tcp 127.0.0.1:19701: bind: address already in use` |
| 依赖面 | 无需 x/sys（硬编码常量即可）；模块缓存已有 `golang.org/x/sys@v0.2.0`（离线可装） |

## 2 端口交接窗口 + TIME_WAIT

| 场景 | 结果 |
|---|---|
| 旧进程 SIGTERM（默认终止）后每 5ms 尝试重绑 | `rebind SUCCESS on attempt 1 (2ms)` |
| 服务端先 close 制造**真实 TIME_WAIT**（`ss` 实测：`TIME-WAIT 127.0.0.1:19999 127.0.0.1:47132`） | 仍 `attempt 1 (2ms)` 成功 |

结论：Go `net.Listen` 默认带 SO_REUSEADDR → TIME_WAIT 不阻塞重绑；交接中断量级 = 进程退出+启动本身（主报告实测：无 SSE 时退出 8.7ms / 有 SSE 时 9.9s 且 exit code=1）。

## 3 syscall.Exec 保 pid 自替换（/tmp/exectest）

```
[parent] pid=192524 listener fd=4 FD_CLOEXEC=1 (1=set)
[parent] cleared CLOEXEC errno=errno 0 FD_CLOEXEC=0 (0=cleared)
[parent] exec /tmp/exectest/exectest → expect same pid after exec
[child]  pid=192524 inherited fd=4 FileListener err=<nil>
[child]  accept OK from 127.0.0.1:49388
[client] received "hello-from-inherited-listener\n"
```

可行：pid 不变、监听 socket 跨 exec 存活、可继续 Accept。注意点：① Go 建的所有 socket 默认 `FD_CLOEXEC=1`，必须 `syscall.Syscall(SYS_FCNTL, fd, F_SETFD, 0)` 显式清除（并复核 F_GETFD=0）；② fd 号只能经 argv/env 传给新镜像；③ exec 不换 cwd/env；④ 只清该 fd，别误清其余 fd（否则 epoll/io_uring/日志 fd 泄漏进新镜像）；⑤ 新镜像必须自己重建 mux、Registry、Supervisor（旧进程内存态全丢）。

## 4 socket activation 依赖面

```
/proc/1/comm = systemd ; systemctl --version = systemd 249 ; systemctl is-system-running = starting
which systemd-socket-activate = /usr/bin/systemd-socket-activate   ← 实测可用（无需 daemon）
Listening on [::]:19998 as 3. / Execing /tmp/sa/sa
[sa] my pid=192267 LISTEN_FDS="1" LISTEN_PID="192267" inherited listener: [::]:19998
client: served from inherited fd 3, pid=192267
```

结论：**机制**（LISTEN_FDS + fd 3 继承）在本环境完全可用，等价于「fd 交接」；但依赖 systemd 托管 unit 不可靠（`is-system-running=starting`，容器内 unit 管理不可信）。

## 5 生产进程关系（只读）

```
172761  PPID=1       PGID=172761  SESS=172761  ./bin/rick web --listen 0.0.0.0 --port 8413
176315  PPID=172761  NSpgid=176315 NSsid=172761  pi        ← 当前唯一 rpc worker
/proc/176315/fd: 0 -> pipe:[483394971]   1 -> pipe:[483394972]
/proc/172761/fd: 10(w) -> pipe:[483394971]   11(r) -> pipe:[483394972]
```

worker 与 web 同 session、**独立 PGID**、I/O 走**匿名管道**。孤儿实验（node 子进程 stdout 接管道，父进程 SIGKILL）：

```
Error: write EPIPE (Unhandled 'error' event) ... Node.js v24.16.0   ← 子进程随即退出
```

结论：web 被杀 → 内核层不连带杀 worker（PGID 独立、reparent 到 1），但 worker 下一次写 stdout 即 **EPIPE 自杀**；匿名管道无名字、pi 无 attach/socket 模式 → 新实例**无法接管已存在 worker**。

## 结论（4 条）

1. SO_REUSEPORT 可用，但需硬编码 `0xf`（stdlib 无该常量）；本项目零新依赖即可。
2. 端口交接本身无成本（2ms，TIME_WAIT 不影响）；重启成本全在优雅关停 worker/SSE。
3. `syscall.Exec` 保 pid + 保监听 socket 可行（清 CLOEXEC + argv 传 fd 即可）；但 worker 仍会死，救不了「开发会话不断」。
4. socket activation 机制可自建（免 daemon），unit 托管不可靠。

## 待验证点

- SO_REUSEPORT 双 listener 的内核分流/连接分配行为（本次只验证绑定成功）。
- exec 自替换后新镜像的 http.Server 重建路径（需自研 fd 交接约定）。
- pi worker 对 EPIPE 的退出行为是否随 node/pi 版本变化（本机 = 抛未处理 error 退出）。
