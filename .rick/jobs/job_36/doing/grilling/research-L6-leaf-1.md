# Research: promote 新二进制+重启而不丢会话（外部事实 L6）

1. **Unix 语义** 进程持 inode 非路径：rename/unlink 后旧 inode 仍被引用，`/proc/pid/exe` 指旧文件（unlink 后加 " (deleted)"）；rename(2) 同 FS 原子替换 newpath，跨 FS 返 EXDEV，mv 退化为 copy+unlink（非原子）；写正在执行的文件得 ETXTBSY（rename/unlink 不触发）。软链须 `ln -s 新 .tmp && mv -T .tmp current` 才原子。→**可迁移**：换二进制对已运行进程安全，但不阻止新连接进旧二进制。
   man7.org/linux/man-pages/man2/rename.2.html ; man7.org/.../man5/proc_pid_exe.5.html ; deployer.org/blog/atomic-symlinks

2. **K8s 滚动更新** terminationGracePeriodSeconds 自 preStop 执行前起算，preStop 返回后才发 SIGTERM；PDB 只约束 eviction（自愿中断），不约束 RollingUpdate；K8s 不感知会话，长连接随进程死，重连/续传责任在应用层，原生会话恢复仍是 KEP 提案。→**部分可迁移**：drain 顺序可借鉴，协调者与恢复机制不可。
   kubernetes.io/docs/concepts/containers/container-lifecycle-hooks/ ; kubernetes.io/docs/tasks/run-application/configure-pdb/ ; github.com/kubernetes/kubernetes/issues/140018

3. **systemd socket activation** Accept=no 时 systemd 持监听 socket，listen FD 从 fd 3 起传入（sd_listen_fds/LISTEN_FDS）；FileDescriptorStoreMax>0 + sd_notify 发 `FDSTORE=1`（用 sd_pid_notify_with_fds 带 fd）可把 listening FD 存回 manager，重启后再传回。结论：**端口零中断**，但只保监听 FD，**已建 TCP/SSE 连接随旧进程断**。Go：go-systemd `activation.Listeners()` → `net.FileListener`。→**部分可迁移**：rick 无 systemd，但 FD 继承模式可自实现。
   freedesktop.org/software/systemd/man/latest/sd_listen_fds.html ; systemd.io/FILE_DESCRIPTOR_STORE/

4. **nginx / SO_REUSEPORT** USR2 起新 master（新二进制），旧 worker 收 WINCH+QUIT 后优雅退出、继续服务既有连接——靠 master 传 listen FD。SO_REUSEPORT（Linux 3.9+）允许多 socket 绑同端口、内核按 4 元组哈希分发，但有连接失败实现坑。Go 官方不暴露该选项（golang/go#23696：「don't currently use SO_REUSEPORT」），需 ListenConfig.Control 自行 setsockopt；更稳为 FD 继承+net.FileListener。→**可迁移**：Go 单进程可行，但须一并解决进程内 pi worker/SSE 归属。
   nginx.org/en/docs/control.html ; lwn.net/Articles/542629/ ; github.com/golang/go/issues/23696

5. **Erlang hot code swap** 同模块可并存 current/old 两版本，旧进程跑旧码直到全限定调用；code:load_file、sys:change_code、code_change/3（appup `{advanced,Extra}`）、release_handler 做整 release 升级，soft_purge 只清无进程在跑的旧码。前置：同 VM 内载新模块+版本化状态+幂等 upgrade 回调。Go 静态二进制无 VM/模块表，做不到。→**不可迁移**（仅「显式状态迁移回调」思路可借）。
   erlang.org/doc/system/code_loading.html ; erlang.org/doc/system/release_handling.html

6. **Docker live-restore** live-restore 使 daemon 重启不杀 running 容器，进程/端口延续；官方明言「networking and user input are interrupted」，docker exec 会话断。restart policy 语义不同（容器退出/daemon 启动时拉起）。→**部分可迁移**：重启对象与业务进程可解耦，但只保住进程，保不住交互通道。
   docs.docker.com/engine/daemon/live-restore/ ; docs.docker.com/engine/containers/start-containers-automatically/

7. **dev/prod 隔离先例** Vite 的 `vite dev` 与 `vite build`+`preview` 是两命令两进程，preview 明示不可作生产服务器；Next.js `next dev` vs `next build`+`next start` 同理。air/nodemon 是杀进程重启，内存态（缓存/连接池）丢失（webpack-hot-middleware#21 抱怨 forced exit）。→结论：**dev server 用专用端口、生产不动是业界常规**（换的是构建产物，不是同一监听进程）；但这些先例都不解决进程内会话。
   vite.dev/guide/cli ; nextjs.org/docs/app/api-reference/cli/next

## 可复用模式提炼
1. **谁持端口**：由寿命最长者持 listen FD（systemd/父进程），业务进程只 accept；重启只换业务进程。
2. **谁持会话**：热重载/重启工具都保不住进程内会话（Erlang 例外，且靠同 VM 双版本代码）；rick 的 pi worker+SSE 在进程内 → 重启必断，须外部化或明确接受中断。
3. **状态放哪**：落盘且可重建（现 sessions.json + jsonl），新进程 re-attach/重放，不指望迁内存。
4. **升级协议**：版本目录 + `mv -T` 原子切 current 软链；跨 FS 退化为非原子 copy。
5. **drain 顺序**：停收新连接 → 等既有流结束或超时 → 退出（K8s preStop+grace 同构）。
