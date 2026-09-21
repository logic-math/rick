# research-L6 简报 —— 生产提升（promote）与 job 不中断的恢复语义

- 阶段：L6（KR3 受控提升 + KR4 切换无损恢复）
- 主题：dev 产物 → 生产（`~/.rick`）的原子替换、重启方式与中断边界、重启后自动恢复运行中会话/job、门禁、人类确认留痕、生态先例、风险清单
- 时间基准：2026-09-20 22:10–22:40（生产实例 = pid 172761，`./bin/rick web --listen 0.0.0.0 --port 8413`，cwd=/workdir/sunquan20/AI_CODING/rick；开机时长 15 分钟）
- 叶子调研：research-L6-leaf-1.md（生态先例，外部信源）、research-L6-leaf-2.md（pi resume 实测）、research-L6-leaf-3.md（仓库事实：门禁/构建/鉴权/隔离/后台任务）；本简报的置信度由 research 自行计算，不采信叶子自报值

## 结论速览（12 条，详情见对应节）

1. **生产二进制就是仓库工作树里的 `bin/rick`**（非 `~/.rick/bin/rick`）：`/proc/172761/exe → /workdir/.../rick/bin/rick`、cmdline `./bin/rick`、md5(running)=md5(bin/rick)=`643b54a3…`；`~/.rick/bin/rick` 是 8/24 的旧 7.4MB 遗留。（§1）
2. **同文件系统 `mv` 覆盖正在运行的二进制是原子的且安全**（实测：`cp` → `Text file busy` 失败；`mv` → 成功，旧进程继续跑，`/proc/<pid>/exe` 显示 `(deleted)`）；`/workdir` 与 `~/.rick` 同属 overlayfs(dev=15933053) → rename 原子；跨文件系统的 `mv` 会退化为 copy+unlink，**路径短暂消失（非原子）**。（§1）
3. **`~/.rick/web.pid` 现在根本不存在**（prod 在跑却不判活）→ 单例门禁目前失效：一个同 HOME、不同端口的第二个 `rick web` 不会被拒绝，会**双写 `sessions.json`**，并且它的 `ReconcileOnStart` 会把生产 active 会话改写为 error 落盘。（§1/§5/§7，高危）
4. **隔离手段只有 HOME**（无 `--state-dir`）：`--port/--listen/--token` 是全部 flag。实测 `HOME=/tmp/l6home ./bin/rick web --port 18413` 可同时起（自己的 web.pid/config/sessions.json；`/api/sessions` = `[]`；生产 sessions.json md5/mtime 不变）；代价：工作区注册表、config（`pi_extra_args` 丢失）、pi agent dir 全部另起一套，需 `RICK_PI_AGENT_DIR` 显式指向共享 agent dir。（§1/§5）
5. **重启的必然中断来自进程内 worker**：`ShutdownWorkers()`（`internal/web/server.go:130-137`，在 HTTP drain 之前）SIGTERM→5s→SIGKILL 收掉全部 rick web 托管的 pi worker；`pumpWorker` 见到 channel close → `markWorkerLost` → 状态落盘 error + 推 SSE（这就是线上那条 `closed_at=21:59:18` 的来源）。**drain 期间浏览器先收到 error 事件，再断线**。（§2）
6. **SSE 切换对浏览器是无感的（自愈）**：重启后 hub 缓冲为空 + 客户端携带陈旧游标 → 服务端回 `server_info{reason:"replay_overflow"}` → 客户端清游标、采纳新 seq、派发 `SSE_RESYNC_EVENT` → stores 拉 REST 全量快照（`internal/web/sse.go:130-152`；`web/src/api/sse.ts:290-320,375-430`）。**但"无感"仅限 UI：in-flight 回合的计算已死**，恢复靠 §3。（§2）
7. **`ReconcileOnStart` 现在只做「active/running → error（worker lost）」且不落盘任何 reason**（`internal/web/sessions.go:154-173`）：`updateStatus` 的 reason 只进 hub 事件，不进 `sessions.json`；`Busy` 是 `json:"-"`（`registry.go:234-254`）→ **"重启前是否正在跑/是否回合中途"两个信息都无法从磁盘推断**。（§3）
8. **需要新增 intent 持久化**：推荐「关停前写 shutdown 快照（session id + pi session file + `Worker.LastEntryID()` + busy）」+ `params["_auto_resume"]` 标记；`Entry.Params` 是自由 map、`_` 前缀字段会被 API 投影剥离（`sessions.go:283-291`），**无需改 schema**（`sessions.json` `version:1`，Go json 忽略未知字段）。（§3）
9. **"回合被中断"可从 pi jsonl 精确判定**（实测 8 个真实 session 的记录结构）：记录类型只有 `session/model_change/thinking_level_change/message/custom_message/compaction`，`message.role ∈ {user,assistant,toolResult}`，`message.stopReason ∈ {toolUse,stop,aborted,error}`；**末条 assistant 且 `stopReason=toolUse`（content 含 `toolCall`）而无后续 toolResult = 回合中途中断**；末条 toolResult = 步骤间中断；末条 `stopReason=stop/aborted` = 正常收尾/用户中止（幂等空闲）。（§3）
10. **pi 侧 resume 行为以叶子实测为准**（见 leaf-2 节）；风险面：被中断的 toolCall 可能已产生副作用但未记录 → 自动续跑有**重复副作用**风险，故推荐「默认只恢复空闲 worker，不自动续跑回合」。（§3）
11. **doing 后台任务不能 resume，只能重跑**（`SessionResume` 对 doing 直接 409，`sessions.go:1018-1020`；`DoingIn` 按 `status != success` 续跑剩余 task，`handler/doing.go:144-166`）；但**确定性门禁把遗留 `running` 判为 zombie 而失败**（`.rick/skills/rick-gates/helper.py:47-49`）→ 自动恢复必须先归一化 `running → pending`。（§3/§7）
12. **提升门禁与留痕**：现有 gate1–6 均为 python 脚本、输出 `{"pass","errors"}`；`go test ./...` 实测通过（leaf-3 实测，EXIT=0）；`rick web` 只有单用户 token 鉴权（`auth.go:29-50`），**全仓无 approve/confirm 语义**，最接近的是 `rick web reset` 的 `[y/N]`（`internal/cmd/web.go:192-210`）+ level 门禁 `gate_cmd`（`builder/orchestration.go:381`）→ 推荐「一次性 approval token 的 POST /api/promote」。（§4/§5）

## 问题索引

| # | 问题 | 本简报节 | 主要证据 |
|---|------|----------|----------|
| 1 | 原子替换三方案对比 + 回滚命令 | §1 | 实测 cp/mv/ETXTBSY、符号链接切换、`/proc/*/exe` |
| 2 | 重启方式与中断边界（kill/蓝绿/代理） | §2 | server.go 关停序、sse.go/hub 重放、sse.ts 游标自愈 |
| 3 | 恢复语义（intent/resume 行为/续跑策略/降级） | §3 | sessions.go、registry.go、helper.py、真实 sessions.json 与 jsonl 实测 |
| 4 | 提升前校验与门禁 | §4 | gate1-6、tasks.json、Makefile、go test 实测 |
| 5 | 人类确认留痕 | §5 | authWrap/TokenAuth、无 approve 语义、reset 的 y/N |
| 6 | 生态先例可迁移性 | §6 | leaf-1（外部 URL）+ 本仓结构判断 |
| 7 | 风险清单与缓解 | §7 | 汇总 |

> 说明：正文按调研完成顺序追加（§4 早于 §3 落盘，§3 依赖最后一个叶子完成）。各节标题自带编号，按上面的索引跳转即可。

---

## §1 原子替换：dev 产物 → 生产

### 1.1 生产二进制路径（事实，实测）

```
$ ls -l /proc/172761/exe /proc/172761/cwd
lrwxrwxrwx ... /proc/172761/exe -> /workdir/sunquan20/AI_CODING/rick/bin/rick
lrwxrwxrwx ... /proc/172761/cwd -> /workdir/sunquan20/AI_CODING/rick
$ tr '\0' ' ' < /proc/172761/cmdline   →  ./bin/rick web --listen 0.0.0.0 --port 8413
$ md5sum bin/rick ~/.rick/bin/rick
643b54a3ee75bb7980be8303bf45ed17  bin/rick          (16.4MB, Sep 20 21:59)
f18498d279c37059ed3faa7e01866091  ~/.rick/bin/rick  (7.4MB,  Aug 24)   ← 遗留副产物，未被使用
$ cat ~/.rick/start-web.sh   →  cd /workdir/sunquan20/AI_CODING/rick && exec ./bin/rick web --listen 0.0.0.0 --port 8413
```

- **结论**：生产 = 仓库工作树 `bin/rick`（`bin/` 被 `.gitignore:2` 忽略、不入库）；`~/.rick/bin/rick` 与 `scripts/install.sh` 的 `prefix=~/.rick` 是「安装式」分发路径，**本机部署没走那条路**。提升对象因此是：① 仓库工作树源码（git）+ ② `bin/rick` + ③ 前端（内嵌 dist 或 `~/.rick/web/dist` 覆盖层）。confidence 0.98
- 代价提示：`start-web.sh` 硬编码仓库路径 → 提升无法把生产搬到独立目录，除非同时改脚本。

### 1.2 Unix 语义（实测，/workdir 即本机 overlayfs）

| 操作 | 结果（实测命令与输出） | 语义 |
|---|---|---|
| `cp 新 正在运行的二进制` | `cp: cannot create regular file 'live.bin': Text file busy` | ETXTBSY：**不能原地写正在执行的二进制** |
| `mv 新 正在运行的二进制`（同 FS 同目录） | `mv OK`；被替换进程继续运行；`readlink /proc/<pid>/exe` → `.../live.bin (deleted)` | rename(2) 原子；旧 inode 由进程持有到退出 |
| 版本目录 + `current` 符号链接切换 | `ln -sfn versions/v2 current` 后，运行中进程 `exe -> .../versions/v1/rick` 仍存活 | 切链对已运行进程零影响 |
| `go build -o 正在运行的二进制` | 成功（未报 ETXTBSY），旧进程继续跑 | cmd/go 走临时文件+rename，非原地写 |
| 跨文件系统 mv（/tmp=xfs → /workdir=overlayfs） | `mv` **成功**（对运行中的二进制也成功），旧 inode 变 `(deleted)` | mv 跨 FS 退化为 copy/或「先 unlink 再建」→ **路径会短暂不存在，非原子**，绝不能用于提升 |
| `df/stat -f` | `/workdir/.../rick/bin` 与 `/home/hadoop-recsys/.rick` 同为 `overlayfs dev=15933053` | 二者之间 rename 原子；但 overlayfs 对 lower 层文件的 rename 会 copy-up（本机未观测到失败/EXDEV） |

### 1.3 三方案对比与推荐

| 方案 | 原子性 | 回滚 | 代价/风险 | 判定 |
|---|---|---|---|---|
| (a) 直接 `mv` 覆盖 `bin/rick` | 同 FS 原子 ✅；跨 FS 非原子 ❌ | 只能靠「提升前 `cp bin/rick bin/rick.prev`」手工回滚 | 会**覆盖掉上一版**（回滚依赖副本），提升失败时上一版只剩一个裸文件；rollback 需重新验证 md5 | 可用，但不推荐单独用 |
| (b) 版本目录 + `current` 符号链接 | 换链需 `ln -s 新 .tmp && mv -T .tmp current`（`ln -sfn` 先 unlink 再建，**非原子**） | 一键：把链切回上一版目录（瞬时、原子、无需重新构建） | 多一层目录；`bin/rick` 需变成符号链接（`start-web.sh` 的 `./bin/rick` 仍可用）；旧版本目录需 GC 策略 | **推荐基线** |
| (c) 版本目录 + 链 + 保留前 N 版 + 覆盖层一并版本化 | 同上 | 二进制与前端一起切 | 需要「dist 覆盖层」也纳入版本目录（见 1.4） | **推荐完整形态** |

**推荐（c）**，理由：回滚是「切链 + 重启」，不需要重新构建（构建耗时与不确定性最大）；代价是要管理 `releases/` 目录与 GC，并保证 `bin/rick → releases/current/rick` 的相对/绝对路径在 `start-web.sh` 下可用。

### 1.4 前端产物与二进制的版本一致性（关键约束）

- 覆盖层 `~/.rick/web/dist` 优先级高于内嵌 baseline，**逐请求判断**（`internal/web/static.go:63-71`：`Lstat(overrideDir/index.html)` 存在即整树用覆盖层，all-or-nothing），覆盖层路径 = `$HOME/.rick/web/dist`（`WebStateDir()`，`internal/web/web.go:25`）。
- 因此「提升二进制」不会自动切前端：只要覆盖层在，新二进制的内嵌 dist 不生效。实测（leaf-3）：当前 `~/.rick/web/dist/index.html` 与 `web/dist/index.html` md5 相同 → 覆盖层与内嵌一致，属正常态。
- 两种一致策略（推荐后者）：
  1. 分别提升：`releases/<v>/dist` → 原子替换 `~/.rick/web/dist`（`mv dist dist.prev && mv dist.new dist`），回滚切回 `dist.prev`。
  2. **只提升二进制并撤掉覆盖层**：`npm run build` 产物随 `go build` 内嵌（`web/embed.go:35-36` `//go:embed all:dist`），`rm -rf ~/.rick/web/dist` 后前端与二进制**天然同一版本**，回滚二进制即回滚前端。日常热迭代（不重启）时再用覆盖层。
- 注意构建链：`scripts/build.sh` **不含前端构建**（`go build -o bin/rick ./cmd/rick`，`scripts/build.sh:88-92`）；前端进仓库靠 `Makefile:4-5 make web-dist`（`cd web && npm ci && npm run build`）。**不重跑 `make web-dist` 就 go build，会把旧前端内嵌进新二进制**。

### 1.5 一键回滚与提升命令（可执行）

```bash
# ---- 目录约定 ----
# REPO=/workdir/sunquan20/AI_CODING/rick
# REL=$REPO/bin/releases        （bin/ 已 gitignore → 产物不入库，符合现状）
# REL/<ver>/rick  REL/<ver>/dist   REL/current -> <ver>   （符号链接）

# ---- 提升（dev 产物 → 生产）----
set -euo pipefail
VER=$(date +%Y%m%d-%H%M%S)
cd web && npm ci && npm run build && cd ..          # 前端 → web/dist（内嵌源）
go test ./...                                        # 门禁①
go build -o "bin/releases/$VER/rick" ./cmd/rick      # 新文件，不碰运行中的二进制
cp -r web/dist "bin/releases/$VER/dist"
ln -sfn "$VER" bin/releases/.current.tmp && mv -T bin/releases/.current.tmp bin/releases/current   # 原子换链（若 bin/rick 为链接则切 bin/rick）
# 记录回滚点
readlink -f bin/releases/current > bin/releases/.last        # 或 cp 上一版链接目标
# ---- 重启（见 §2；顺序不可颠倒）----
ps -o pid= -C rick | head -1   # 找到 web pid（或 ss -ltnp | grep :8413）
kill -TERM <PID>; while kill -0 <PID> 2>/dev/null; do sleep 0.2; done   # 最多 ~11s（有 SSE 连接时）
rm -f ~/.rick/web.pid          # 若上面超时（进程未退干净）才需要，见 §2.3
nohup ./bin/rick web --listen 0.0.0.0 --port 8413 >> ~/.rick/web.log 2>&1 &
until curl -sf --noproxy '*' http://127.0.0.1:8413/api/health >/dev/null; do sleep 0.1; done
# ---- 回滚（一条命令 + 重启）----
ln -sfn "$(cat bin/releases/.last)" bin/releases/.rollback.tmp && mv -T bin/releases/.rollback.tmp bin/releases/current
rm -rf ~/.rick/web/dist && cp -r "$(readlink -f bin/releases/current)/dist" ~/.rick/web/dist   # 前端一并回滚（或直接删覆盖层用内嵌）
# 再次执行上面的「重启 + 健康检查」
```

- 回滚成本：切链 + 重启（~1s 启动 + ≤11s 关停），无需构建；上一版二进制始终在 `releases/` 内保留（建议保留最近 3 版，`ls -1t bin/releases | tail -n +4 | xargs rm -rf`）。
- 校验：`curl -H "Authorization: Bearer $TOK" :8413/api/config | jq .rick_version`（版本号硬编码于 `cmd/rick/main.go:8 const VERSION`，无 ldflags —— 提升前后据此确认生效，leaf-3 实测 `{"version":1,"rick_version":"4.4.15",...}`）。

---

## §2 重启方式与中断边界

### 2.1 关停/启动的实际时序（实测 + 代码）

关停序（`internal/web/server.go:119-141`）：`ctx.Done()` → ① `Sessions.ShutdownWorkers()`（SIGTERM → `TermGrace=5s` → SIGKILL，收掉**全部** rick web 托管的 pi worker，`internal/runtime/supervisor.go:43-47`；注释明确说明 pi worker 自成进程组，父进程退出不连带杀）→ ② `httpServer.Shutdown(10s)`；若超时 → `Close()` 并返回错误。

实测（dev 实例 18414，一个 `curl -N /api/events` SSE 客户端在线）：

| 观测 | 结果 |
|---|---|
| `kill -TERM` → 进程退出耗时 | **10094 ms**（= 完整 `shutdownGrace=10s`） |
| 关停期日志 | `Error: graceful shutdown: context deadline exceeded` → cobra 打印 usage → **退出码非 0** |
| 关停期 `~/.rick/web.pid` | **仍然存在**（defer 删除只在进程真正退出时执行）→ 此时启动新实例必被 `checkSingleton` 拒绝（`internal/handler/web.go:123-149`） |
| 关停期新进程能否 bind 同端口 | **能**（`Shutdown` 先关监听器；实测 python `bind 18414` 成功）→ 端口不是瓶颈，pid 文件才是 |
| 无 SSE 客户端时 `kill -TERM` | 立即退出（无 blocker） |
| 启动 → `/api/health` 200 | **73 ms**（空状态 dev 实例；`/api/health` 免鉴权实测 `{"status":"ok"}`，`internal/web/routes.go:94-96,849`） |
| 杀掉后立即重启同端口 | 成功（Go listener 默认 SO_REUSEADDR，无 EADDRINUSE） |

**中断窗口的构成**：`max(0, 10s − 0s) + 启动 0.1s`。即**只要浏览器开着页面，每次提升重启 = ~10s 连接不可用**（浏览器侧 1s 起指数退避自动重连，`web/src/api/sse.ts:434-446`）。这 10s 不是"用户按了才发生"，而是 `Shutdown` 等 SSE handler 返回——而 `ServeSSE` 只在请求 ctx 结束（客户端断开）时返回，服务端关停**不会**主动结束流。**这是可以消除的**：给 hub 加 shutdown 广播，让 `ServeSSE` 在关停时 return（改动面：`internal/web/sse.go` ServeSSE 的 select 增加一路 `<-shutdownCh`；`server.go` 关停前先广播）→ 关停退化为 ~0.1s。

### 2.2 三种重启方式对比（本机约束：无 root、无 systemd user manager、无 nginx/socat/haproxy、单机单用户）

实测环境：`pid1=systemd` 且 `/run/dbus/system_bus_socket` 存在、`systemctl list-units` 可列（`is-system-running=starting`），但 **`systemctl --user` 不可用**（`/run/user/1423176628` 为空、`loginctl` 报 "not logged in or lingering"），且我们非 root → **不能建 system/user unit**；`nginx/socat/haproxy/caddy` 均未安装（`which` 全 miss）。

| 方案 | 本机可行性 | 端口连续性 | 会话/worker 连续性 | 代价与风险 |
|---|---|---|---|---|
| (i) kill + 重启 | ✅ 直接可用 | 断 ~10s（有 SSE 客户端时）/ 立即可（无客户端）；bind 同端口无问题 | worker 全部被杀（`ShutdownWorkers`），恢复靠 §3 | 唯一代价是那 10s 与「必须等进程退出否则被 singleton 拒绝」。**推荐基线** |
| (ii) 蓝绿（新实例先起、健康检查、停旧） | ⚠️ 同 HOME 不可行：第二个实例要么被 pid 门禁拒（文件在时），要么**共享 `sessions.json` 双写**（§5.2）；同 HOME 下新实例的 `ReconcileOnStart` 会把旧实例的 active 会话改写成 error（`sessions.go:154-173`）。要用蓝绿必须换 HOME，则工作区注册表/config/pi agent dir 全部另起（§1 隔离） | 端口只能换（浏览器书签固定 8413）→ 必须前置代理才有意义 | **零收益**：worker 不能迁移（supervisor 每会话一进程、pi 单会话文件被旧进程持有），新实例 resume 同一个 pi session 会与旧 worker **争抢同一 session 文件** | 复杂度高、收益≈0 → **不推荐** |
| (iii) 稳定代理端口（浏览器只连代理） | ⚠️ 需自研/安装代理（本机无 nginx/socat） | 代理可避免"连接被拒"的一瞬，但**已建立的 SSE 长连接在切换时仍必断**（代理不搬 TCP 连接；nginx 的 upstream failover 只在建连期生效） | 同上，worker 仍在后端进程内 → 无收益 | 需要新增长期运行的组件；只在「想保持 8413 不变但想换后端实现/多后端」时有价值 → **暂不需要** |
| (iii') 同端口的「FD 继承」变体（Go 版 socket activation） | 可自实现：父进程持 listener，把 `*os.File`（`Listener.File()`）传给新子进程（`ExtraFiles`），新进程 `net.FileListener` 接管，旧进程只做 drain | 端口与监听 FD 零中断（systemd socket activation 的等价物，`sd_listen_fds` 语义） | worker 仍在旧进程 → 仍需 §3 恢复；等于「(i) + 不丢端口」 | 需新增一套 supervisor-shim 进程模型（谁拉起 shim？需 daemon 化脚本/自守护），改动面最大；仅当"连接被拒的一瞬"被证明为痛点时才值得 |

**推荐**：先做 (i) 的**受控重启脚本**（顺序、等待、健康检查、失败回滚），并把 §2.1 的 10s 用一个 20 行的 SSE shutdown 广播消掉——这比引入代理/蓝绿便宜一个数量级；(iii') 留作后续可选（若将来要做到"本地内核热切"或需要多后端）。

### 2.3 SSE 在现有实现下切换能否"用户无感"（对照实现给结论）

服务端：`Hub` 全局单调 seq + 1000 条环形重放缓冲（`internal/web/sse.go:50-60,73-108`）；重连时按 `Last-Event-ID`/`?lastEventID` 重放 seq 之后的缓冲；**游标早于缓冲窗口（或重启后缓冲为空）→ 直接发 `server_info{reason:"replay_overflow"}`**（`sse.go:117-155`）。每次连接先发 `server_info`（`routes.go:828-836`）。

客户端（`web/src/api/sse.ts`）：游标持久化在 `localStorage["rick-web-sse-cursor"]`（节流 500ms，`pagehide`/`visibilitychange` 立即落盘，`:186-238`）；断线主动 close + 指数退避 1s→30s 重连（`:434-446`）；拿到 `replay_overflow` → 清游标、**无条件采纳服务端 seq**、派发 `rick-web:sse-resync`（`:290-320`）→ `stores/events.ts:243-247`、`stores/sessions.ts`、`stores/jobs.ts:135-136` 重新拉 REST 全量快照；seq 回退（重启后 hub 归零）同样触发 resync（`:395-410`）；`frontend_reload` 用 sessionStorage 去重防重载循环（`:339-348`）。

**结论（confidence 0.9）**：
1. **浏览器侧无感**：切换后 1–11 s 内自动重连，重连即全量 resync 对齐；页面不需要用户刷新，也不会长期停在旧状态（seq 回退/replay_overflow 两条路径都覆盖了"服务端重启"）。
2. **无感 ≠ 无损**：REST 快照能重建 **状态**（会话列表、jobs、任务名），但**重建不了正在流式输出的那半个回合**——worker 已被杀，hub 缓冲为空，那段 thinking/text 不在 REST 里（部分在 pi jsonl 里，见 §3）。UI 上表现为：会话卡片先收到 `session_state=error`（关停时序里 `ShutdownWorkers` 早于 HTTP drain → 浏览器**能**先收到这条，见 §2.1），随后按钮变为 Resume。
3. 因此 KR4「前端事件流不断档」在实现上已具备（游标续传 + resync），**真正的缺口是 §3 的自动恢复**，不是 SSE。

---

## §4 提升前的校验与门禁

### 4.1 现有门禁清单（事实，逐文件实测）

| 门禁 | 覆盖 | 关键动作（文件:行） |
|---|---|---|
| gate1 | task3 前端骨架/embed + task5 registry | `go build ./...`；`npm run build`（`gate1.py:47-49`）；查 Makefile 含 `web-dist`（`:43-44`） |
| gate2/3/4 | task7+ 各层 | `go build ./...`；`npx tsc --noEmit` + `npm run build`（`gate2.py:48-53`、`gate3.py:37-42`、`gate4.py:37-42`）；**残留 pi 检查** `pgrep -f '[m]ode rpc'`（`gate2.py:55-57`） |
| gate5 | task14 env+cmd 收口 | `go build ./...`、四个包 `go test`、**`go build -o bin/rick ./cmd/rick`**（`gate5.py:27`）+ `bin/rick web --help` 与 customize/reset 子命令（`:31-38`） |
| gate6 | 真 E2E | `go build -o bin/rick`（`gate6.py:43`）→ `HOME=<tempdir>` + `--port 16374`(占用则 16375) + `--token e2etest-token` + `no_proxy=127.0.0.1,localhost` 起真实服务（`:52-58`）→ 轮询 `/api/health`（`:63-66`）→ 401/静态/SPA/工作区/jobs/SSE `server_info` → SIGINT 后校验 `<tmp>/.rick/web.pid` 已清理（`:132-133`） |
| 统一入口 | — | **不存在**：`Makefile` 只有 `web-dist`/`web-dev`；`web/package.json` 只有 `dev/build/typecheck`；无 `make test` |
| 全量测试 | Go 全包 | `go test ./...` 实测全 `ok`、`EXIT=0`（约 60s，多为 cached；证据 `/tmp/l6leaf3_gotest.txt`） |
| 现状一致性 | — | 覆盖层 `~/.rick/web/dist/index.html` md5 == `web/dist/index.html`（leaf-3 实测）；`git status web/` 干净 → 覆盖层==内嵌==HEAD |

### 4.2 现有门禁在「提升」场景下的三个缺陷（必须处理）

1. **门禁会覆盖它要验证的产物**（severity: high）。gate5/gate6 执行 `go build -o bin/rick ./cmd/rick`，而 `bin/rick` **就是生产二进制路径**。实测 `go build -o` 覆盖运行中的二进制**不会** ETXTBSY（走临时文件 + rename，与 `cp` 相反，§1.2）→ 静默成功。后果：跑完门禁后 `bin/rick` 可能已不是「人类批准的那份」（构建时间戳/md5 变了），且失败时**上一版已被覆盖**。→ 结论：**门禁必须在 dev 工作树里跑**（gate 脚本的 `ROOT` 由文件位置上溯 5 层推出，`gate5.py:5-6`，所以在 dev worktree 里复制一份即可正确指向 dev），生产侧只做只读校验（§4.3 步骤 6）。
2. **门禁会改写仓库受版本控制的前端产物**（severity: medium）。gate1/2/3/4/6 在 `ROOT/web` 里 `npm run build` → 重写 `web/dist/`（该目录**入库**，`.gitignore` 未忽略）。若构建不完全确定，git 会变脏、提升提交会混入 dist churn；更糟的是**在没有覆盖层的机器上，门禁一跑前端就实时变了**（`static.go` 是逐请求判断，无缓存屏蔽）。缓解：门禁在 dev 树跑；生产树改前端只走 `make web-dist` + 明确提交。
3. **残留 pi 检查（gate2/3/4）既失灵又易误报**（severity: medium）。实测 `pgrep -af 'mode rpc'` 对真实 worker **零命中**：生产 web(172761) 现有一个真实 rpc worker `pi`（pid 176315，`/proc/176315/cmdline` = `pi`，`exe → /usr/local/node-v24.16.0/bin/node`，即 pi 的 argv[0] 被改写为 `pi`），而 `pgrep -f '[m]ode rpc'` 找不到它；反过来，任何命令行里出现该字符串的无关 shell 都会命中（我在本次调研中就被自身命令行命中 `count=2`）。→ 残留检测应改为：按**父进程链/进程组**枚举（`ShutdownWorkers` 用 Setpgid，见 §3.4），或让 pi 启动时写 pid 文件（更可靠）。

### 4.3 可执行的「提升 checklist」（失败即阻断 + 回滚）

```bash
# ============ 阶段 A：dev 侧（不碰生产）============
cd $DEV_WORKTREE                       # 独立 git worktree（KR1），非生产工作树
set -euo pipefail
gofmt -l ./cmd ./internal ./pkg        # 0 输出才继续
go vet ./...                           # 编译期静态检查
go test ./...                          # 全量单测（实测 ~60s）
(cd web && npm ci && npx tsc --noEmit && npm run build)   # 前端 + 类型
go build -o bin/dev-rick ./cmd/rick    # 只建 dev 产物，绝不 -o 生产 bin/rick
for g in plan/gates/gate[1-6].py; do (cd .rick/jobs/job_36 && python3 $g) || exit 1; done   # 现有 6 道门禁（在 dev 树）
# 以上任一失败 → 阻断提升，不进入 B

# ============ 阶段 B：产物封装与报价（只读生产）============
VER=$(date +%Y%m%d-%H%M%S)
mkdir -p bin/releases/$VER && cp bin/dev-rick bin/releases/$VER/rick && cp -r web/dist bin/releases/$VER/dist
( cd bin/releases/$VER && sha256sum rick > SHA256SUMS && find dist -type f -exec sha256sum {} + | sort >> SHA256SUMS )
git -C $DEV_WORKTREE log --oneline $(git -C $PROD rev-parse HEAD)..HEAD > bin/releases/$VER/CHANGES
git -C $DEV_WORKTREE diff --stat $(git -C $PROD rev-parse HEAD)..HEAD >> bin/releases/$VER/CHANGES
bin/releases/$VER/rick web --help >/dev/null   # 冒烟：新二进制可执行
# ↑ 人类审核的输入 = CHANGES + SHA256SUMS（§5）

# ============ 阶段 C：人类批准（gate，§5）============
# 无批准记录 → 拒绝执行 D。批准记录写入 ~/.rick/web/promotions/<id>.json

# ============ 阶段 D：提升 + 重启（原子、可回滚）============
PREV=$(readlink -f bin/releases/current 2>/dev/null || echo "")
cp -r bin/releases/$VER "$PREV_DIR/../$VER" 2>/dev/null || true
ln -sfn "$VER" bin/releases/.current.tmp && mv -T bin/releases/.current.tmp bin/releases/current
[ -n "$PREV" ] && cp -r "$PREV/dist" ~/.rick/web/dist.prev 2>/dev/null || true
rm -rf ~/.rick/web/dist && cp -r bin/releases/$VER/dist ~/.rick/web/dist   # 前端与二进制同版（也可留覆盖层由嵌入兜底）
SESS_BEFORE=$(python3 -c "import json;d=json.load(open('$HOME/.rick/web/sessions.json'));print(len(d['sessions']))")
WEBPID=$(ss -ltnp 2>/dev/null | sed -n 's/.*:8413 .*pid=\([0-9]*\).*/\1/p' | head -1)
kill -TERM "$WEBPID"
for i in $(seq 1 75); do kill -0 "$WEBPID" 2>/dev/null || break; sleep 0.2; done   # ≤15s（有 SSE 客户端时约 10s，§2.1）
kill -0 "$WEBPID" 2>/dev/null && { echo "关停超时"; rm -f ~/.rick/web.pid; }      # 端口已释放（Shutdown 先关监听器）
(cd $PROD && nohup ./bin/rick web --listen 0.0.0.0 --port 8413 >> ~/.rick/web.log 2>&1 &)
for i in $(seq 1 100); do curl -sf --noproxy '*' http://127.0.0.1:8413/api/health >/dev/null && break; sleep 0.1; done

# ============ 阶段 E：提升后验收（失败 → 回滚）============
curl -sf --noproxy '*' http://127.0.0.1:8413/api/health                    # {"status":"ok"}
curl -sf --noproxy '*' -H "Authorization: Bearer $TOK" http://127.0.0.1:8413/api/config | grep -o '"rick_version":"[^"]*"'
curl -sf --noproxy '*' -H "Authorization: Bearer $TOK" http://127.0.0.1:8413/api/sessions | python3 -c "import json,sys;d=json.load(sys.stdin);print('sessions',len(d));print('error',sum(1 for s in d if s['status']=='error'))"
[ "$SESS_BEFORE" = "$(…count…)" ] || echo "会话数变化 → 需人工核对（§3 恢复报告）"
# 恢复报告：期望「重启前 active 的会话数 == 自动恢复成功数 + 明确失败数」，全部有据可查
# 任一验收失败 → 执行 §1.5 回滚（切链 + 重启），并把失败写入 promote 日志
```

---

## §3 恢复语义（核心）

### 3.1 「重启前它确实在跑」——现有可依据的信号与缺口（事实）

| 信号 | 现状 | 能否作为 auto-resume 依据 |
|---|---|---|
| `SessionEntry.Status ∈ {active,running}` | 落盘（`registry.go:243`） | ⚠️ 部分：正常关停时 `ShutdownWorkers` 杀 worker → `pumpWorker` 见 channel close → `markWorkerLost` → **落盘 error**（`sessions.go:1660-1671`、`1775-1786`）。即**优雅关停会把 intent 抹掉**；只有崩溃（SIGKILL）才留下 `active` |
| 关停原因 | `updateStatus(entry,status,reason)` 的 reason **只进 hub 事件**，不入 `sessions.json`（`sessions.go:1782-1793`） | ❌ 无法区分「关停导致」与「三天前 agent 崩了」 |
| `Busy`（本回合是否在流式） | `json:"-"`（`registry.go:247-254`），仅内存 | ❌ 不落盘（且代码注释明确「重启后落盘的 busy 是谎言」） |
| pi 会话文件尾部 | `~/.rick/pi/agent/sessions/<cwd编码>/*.jsonl`，与 web 注册表无关联字段（只有 `PISessionID`） | ✅ 可判定「回合是否中断」（§3.4），但需先定位文件 |
| `_prompt_file`/`_method_file` | `params` 内的保留键，resume 时复用 | ✅ 实测：9 条 active/error 行中 6 条文件仍存在；3 条 easy（疑似 CLI 导入）没有 → 这些行 resume 后**不带方法层提示词** |

**线上实证**（`~/.rick/web/sessions.json`，42 行，2026-09-20 22:13）：`active` 恰好 1 条（type=easy，`closed_at=2026-09-20T21:59:18` = 上一次服务重启的关停时刻，`status=active`），`error` 9 条，其余 32 closed。→ 这就是当前流程的写照：**关停把 active 打成 error（带 closed_at），人工 Resume 又把它拉回 active 但不清理 `closed_at`**（`SessionResume` 只写 status，`sessions.go:1073-1074`；`updateStatus` 只在 closed/error 时写 ClosedAt）。附带缺陷（severity: low）：resume 后 `closed_at` 残留，UI「关闭时间」显示错误。

### 3.2 intent 持久化方案（推荐，无需改 schema）

| 需要回答的问题 | 推荐落点 | 理由 |
|---|---|---|
| 这次关机是我主动做的吗？ | **关停快照** `~/.rick/web/shutdown.json`：`{stopped_at, pid, sessions:[{id,type,workspace_id,status,busy,pi_session_id,pi_session_file,last_entry_id,worker_pgid}]}`，在 `ShutdownWorkers()` **之前**写 | 一次写入覆盖全部候选，不受「worker 死亡回调是否来得及落盘」的竞态影响；`Worker.LastEntryID()`（`supervisor.go:445`）与 `WorkerState.SessionFile` 现成可用 |
| 这条会话该不该自动恢复？ | `params["_auto_resume"]`（缺省 = auto；`"off"` 显式关闭）——`Params` 是自由 map，`_` 前缀键在 API 投影中被剥离（`sessions.go:283-291`），旧版本读新 JSON 也安全（`version:1` + Go json 忽略未知字段） | 零 schema 变更，前端/CLI 立即可用 |
| 上次恢复失败过吗？ | `params["_resume_attempts"]` + 新增顶层 `LastError string`（可选） | 防「恢复风暴」，让 UI 能解释为何没自动恢复 |
| 谁是孤儿 worker？ | 每次 spawn 写 `~/.rick/web/workers/<session_id>.pid`（pgid），正常退出删除 | pi 的 argv 已被改写为 `pi`（实测 `/proc/176315/cmdline` = `pi`），**无法靠命令行识别**；pidfile 同时可修 gate2/3/4 的残留检查（§4.2-3） |

启动时候选集 = `shutdown.json.sessions`（主）∪ 注册表中 `status ∈ {active,running}` 的行（兜底：崩溃路径无快照）∪ 无快照但 `status=error` 且 `closed_at` 在本次启动前 60s 内的行（次兜底，需 `_auto_resume` 未显式关闭）。

### 3.3 pi `--session` resume 的实测行为（leaf-2 实测 + 本简报复核）

环境：pi 0.84.1（`@earendil-works/pi-coding-agent`）。实验全部在 `/tmp/l6/` 的 session **副本**上做，未改动 sessions 目录任何原文件（源文件 mtime 未变），生产 172761 全程存活。

| 编号 | 结论 | 证据 |
|---|---|---|
| P1 | **resume 不自动续跑**：对「正常收尾 / 悬挂 toolCall / toolResult 结尾 / 末条 user」四种副本，`pi --mode rpc --session <path>` 后 `get_state` 均 `isStreaming:false`、`pendingMessageCount:0`，无 `agent_start` | `{"success":true,"data":{"isStreaming":false,"messageCount":1076,"pendingMessageCount":0}}` |
| P2 | **pi 不修复悬挂 toolCall**：构造「user → assistant[toolCall, stopReason=toolUse]」的最小 session 再发 prompt，悬挂条目**原样保留**、不补 toolResult、不丢弃 | `grep -c call_l6_dangling synth-dangling-A.jsonl` = 1；源码层仅做 `toolCall→tool_use` 机械转换，无配对校验（pi-ai `dist/api/anthropic-messages.js:932-978`） |
| P3 | **pi 对 session 文件无排他锁**：两进程同时 `--session 同一副本` 均 `success:true`、`sessionFile` 同路径、无 lock 文件、载入不写文件 | 并发实测；**⟹ 重启后孤儿 worker + 自动 resume = 双写同一 jsonl** |
| P4 | **session 查找是 cwd-scoped**（目录名由 cwd 编码）：在 `/tmp` 下用 uuid 打开 rick 项目的 session → `Session found in different project: /workdir/.../rick` → `Fork this session into current directory? [y/N] Aborted.` | 实测；**⟹ resume 必须以工作区根为 cwd**（现实现 `SpawnSpec.Dir = ws.Path` 正确，`sessions.go:1063-1070`；但若工作区路径变了/被重新注册到别处，rpc 下这个 y/N 会吃掉 stdin 造成挂死） |
| P5 | 配额耗尽**不报错**：catpaw-proxy 返回普通 assistant 文本 + `stopReason:"stop"`、usage 全 0（"额度已使用完毕"） | `dangling-A-raw.txt`；**⟹ rick 无法用 stopReason 区分「答完」与「配额拒绝」**，自动续跑可能静默空转 |
| P6 | `get_entries` 忠实返回含悬挂条目的完整分支（1082 行 → 1081 entry） | 实测 ⟹ 前端可回放未完成回合 |
| P7 | jsonl 记录类型（全库 137 文件 census）：`message`（role ∈ user/assistant/toolResult）、`custom_message`、`session`、`model_change`、`thinking_level_change`、`session_info`、`compaction`；**无** agent_end/message_end 落盘；`stopReason ∈ {toolUse, stop, aborted, error(, length)}` | 与本简报独立抽样复核一致（我另外抽 8 个真实文件：role/stopReason 值域相同；正在进行的本会话尾部 = `stopReason=toolUse` + toolCall） |

**关键推论**：pi 的 resume = 「加载历史 + 待命」。**续跑必须由 rick 主动投递一条 prompt/steer**；而悬挂 toolCall 的会话在投递后会把「未配对的 toolCall」送进 provider（是否 400 未端到端验证，leaf-2 标 U1，confidence 中高：源码 + Anthropic 契约）。

### 3.4 「回合被中断」判定算法（可落地）

```
输入：session 副本的 jsonl 最后一条 type=="message" 记录 + rick 侧 intent
1) 无 message                     → EMPTY（无内容，不值得恢复）
2) role == "toolResult"           → INTERRUPTED/after_tool   （工具已执行完，下一步未发生）
3) role == "assistant":
     stopReason == "stop"         → COMPLETE
     stopReason == "aborted"      → INTERRUPTED/aborted      （有标记；含用户主动 abort）
     stopReason == "error"        → FAILED
     stopReason == "length"       → TRUNCATED               （上下文/输出超长）
     stopReason == "toolUse" 且末toolCall 的 id 无对应 toolResult → INTERRUPTED/dangling（危险）
4) role == "user"                 → PENDING（已提交未应答）
```
真实分布（leaf-2 全库 633 session 尾部 census）：`assistant/stop` 558、`toolResult` **22**、`assistant/aborted` 19、悬挂 `assistant/toolUse` **19**、`error` 5、`length` 2 ⟹ **约 10% 的会话停在未完成回合，其中 3% 是悬挂态**。注意 **jsonl 无法区分「进程被杀」与「用户 abort」**（都是 aborted/空白），故必须叠加 3.2 的 intent。

### 3.5 「续跑未完成回合」策略对比与推荐

| 策略 | 可行性（实测依据） | 风险 | 判定 |
|---|---|---|---|
| A 只恢复 worker（不续跑） | 100%（P1：resume 即待命） | 无 token 消耗；用户需自己再发一句 | ✅ 必做基线 |
| **B 安全形状才续跑**：resume 后 rick 主动投递一条「服务重启中断了上一回合，请从已有工具结果继续」的 prompt | P1 证明必须 rick 投递；P2 证明悬挂态前序非法 → 排除悬挂态 | 每次重启耗 token；长上下文重放成本高；语义漂移（agent 被系统话术打断） | ✅ **推荐（A+B）** |
| C 重发最后一条 user 消息 | 可行但等于重跑整回合 | 高：工具副作用重放（写文件/commit/发请求）+ 悬挂态前序非法 | ❌ 不推荐 |
| D steer/follow_up 续跑 | rpc 层有 `Steer`/`FollowUp`（`internal/runtime/rpc.go:185-196`） | 对 idle worker 的语义等价性未验证（U2） | ⏸ 备选，待验证 |

**推荐 = A（全量恢复 worker）+ B（仅对安全形状、且 `_auto_resume != off` 的会话投递一次续跑）+ 悬挂态降级为「恢复 worker + UI 显式提示需人工确认继续」**。理由：A 零成本且立刻让所有会话「可继续对话」（这是 KR4 的主诉求）；B 只在 90% 正常收尾之外的 10% 里有价值，但必须排除 3% 悬挂态（P2 的未配对 toolCall 是最可能的 400 源），并把「投递」限制为每次重启每会话最多一次。

**防风暴/防配额（必须做，P5 证明配额耗尽不可见）**：
- 并发上限：`Supervisor` 上限 8（`supervisor.go:43`）→ 恢复要用队列 + 抖动（每条 0.5–5s 随机）+ 串行化 spawn，超限排队而非失败重试。
- 总预算：每次重启最多投递 N 条续跑 prompt（建议 N ≤ 3，可配置），其余只恢复 worker。
- 幂等：`sup.Get(id)` 非空且存活 → 跳过（`SessionResume` 已有此模式，`sessions.go:1040-1055`）。
- 空闲回收：idle 30min 自动收 worker（`supervisor.go:44`、`reapLoop`）→ 恢复出来的 worker 不会永久占坑，这是天然的安全阀。
- 观测：把「恢复 N 条 / 失败 M 条 / 续跑 K 条 / 跳过原因」写 SSE `resume_report` + `~/.rick/web/resume-report.json`。

### 3.6 恢复执行时序与失败降级（硬要求：服务必须起得来）

```
NewServer 起来 → /api/health 200（先让服务可用，恢复是后台事）
  └─ go AutoResumeOnStart(ctx)   // 全部包 recover()，绝不 panic 到主流程
       1. 读 shutdown.json（无 → 空）；读注册表；算候选集（§3.2）
       2. 对每条候选：
          a. 孤儿检测：workers/<id>.pid 存在且进程活着 → 先 SIGTERM(-pgid) → 5s → SIGKILL，再恢复
             （理由：P3 无锁；不做这一步就是双写同一 jsonl）
          b. 定位 pi session 文件（快照里的 pi_session_file；否则按 cwd 编码目录 + PISessionID 前缀扫）
          c. 判定形状（§3.4）→ COMPLETE/EMPTY：只恢复 worker；INTERRUPTED(after_tool/aborted/PENDING)：恢复 worker + （预算内）投递一次续跑
             INTERRUPTED/dangling 或 FAILED/TRUNCATED：恢复 worker，状态置 error 并写明原因，等待人工
          d. spawn：SessionID/ Dir=ws.Path / MethodFile/PromptFile(若文件仍存在) / SessionIDFlag=PISessionID / CreateNew=false
             —— 复用 SessionResume 的 spec 构造（sessions.go:1063-1070），不要另写一套
          e. 失败（文件被删/工作区未注册/spawn 超时/pi 探活失败）→ 该条 status=error + 持久化原因 + SSE 事件，继续下一条
       3. 结束：写 resume-report.json + hub.Publish(resume_report)
```
- 降级要求：单条失败不影响其它；**注册表损坏也不允许阻断启动**——但注意当前 `webServeComposition` 在 `LoadSessionRegistry` 出错时直接 return error（`internal/cmd/web.go`），即 **`sessions.json` 损坏 = 生产起不来**（severity: high，见 §7-R2），提升脚本必须先备份并在失败时回滚恢复该文件。
- 报告字段（UI 可读）：`{at, candidates, resumed, failed:[{id,type,reason}], continued, skipped:[{id,reason}], orphans_killed}`。

### 3.7 doing / dream 后台会话的特殊处理（不能 resume）

- `SessionResume` **明确拒绝 doing**（409 "monitor-only; re-run via a new doing session"，`sessions.go:1018-1020`）与 background dream（`sessions.go:1022-1025`）。
- 可重跑：`DoingIn` 每轮用 builder 重新生成「仅剩非 success task」的编排，`status != success` 计数为 pending（`internal/handler/doing.go:144-166,357-361`），attempt 上限 = `config.max_retries`（本机 = 5）。⇒ **重跑是幂等收敛的**（已完成 task 及其 commit_hash 保留）。
- 三个必须处理的坑：
  1. **遗留 `running`**：确定性门禁把 task status=running 判为 zombie 并 fail（`.rick/skills/rick-gates/helper.py:47-49`；`doing.go:227` 注释同款语义）→ 自动恢复前必须把它归一化为 `pending`（写回 tasks.json 前先备份 `tasks.json.bak`）。
  2. **孤儿 doing pi**：doing 的 pi 是 `exec.Command` + **无 Setpgid、不绑 ctx**（`internal/runtime/runtime.go:143-146,166`），而关停只收 supervisor workers（`server.go:130-137`）⇒ **服务重启后 doing 的 pi 仍可能活着继续写 tasks.json/commit**。若新进程再重跑 DoingIn → 两个 pi 同时改同一 job（重复 commit、tasks.json 互相覆盖）。缓解：重跑前用 §3.2 的 pidfile/进程组枚举清理孤儿（doing 应在 spawn 时也 Setpgid 并登记 pid）。
  3. **doing 会话的 status 语义**：现 `ReconcileOnStart` 对 doing 一律 error（`sessions.go:162-167`），重跑需新建/复用会话并写清 reason，否则 UI 显示「错误但实际在跑」。

---

## §5 人类确认留痕（「审核通过」才允许提升）

### 5.1 现有素材（事实）

| 素材 | 事实 | 可否直接用作「人类确认」 |
|---|---|---|
| 鉴权 | `authWrap = TokenAuth`（`routes.go:62-64` → `auth.go:29-50`）：`Authorization: Bearer` 或 `?token=`；仅 `/api/health` 与静态资源豁免（`routes.go:849,908-909`）。token 来源 flag > `config.web_token` > 自动生成（`handler/web.go:153-172`），当前值 `c2f268ce…` | ❌ 前端把 token 存在 **localStorage**（`web/src/api/client.ts:51-57`）→ 任何一次页面加载都"自动携带"= 无人确认 |
| 现有 web 自管理 API | `POST /api/web/customize`、`POST /api/web/reset`（`routes.go:900-902`，env 函数注入，nil → 501）——**无确认体，仅 token** | ⚠️ 可复用其形（endpoint 命名/错误体），不可复用其"无确认"语义 |
| UI 确认弹窗 | 已存在：`Settings.tsx:224-242`（Reset to baseline：「agent 的全部自迭代修改将丢失——确认？」）、`WorkspaceTree.tsx:9,55`（注销工作区二次确认，失败原因显示在弹窗内）、通用 `Dialog.tsx:16`（`false = 禁止关闭（关键确认场景）`） | ✅ 可直接复用（展示变更摘要 + diff 的载体） |
| CLI 确认 | `rick web reset` 的 `[y/N]` 读 stdin（`internal/cmd/web.go:192-210`，`--yes` 跳过） | ✅ 终端路径的现成范式 |
| 门禁即确认 | doing 编排里每层有 `gate_cmd`「人类确认的门禁」（`internal/builder/orchestration.go:381`）；gate 脚本约定输出 `{"pass":bool,"errors":[]}` + exit code（gate1-6） | ✅ 提升前门禁沿用同一约定（gate7） |
| 全仓 approve/approval 语义 | **零命中**（`grep -rni approve internal/` 无结果；前端无 `window.confirm`） | — |

### 5.2 推荐：三层确认，凭证与「会话凭证」分离

1. **CLI 路径（成本最低，先做）**：`rick web promote --from <dev-worktree-or-release-dir> [--yes]`
   - 打印「报价单」：`git log --oneline <prod_head>..<dev_head>` + `git diff --stat` + 新二进制 sha256 + dist 文件数与 sha256 汇总 + 门禁结果 JSON（§4.3 阶段 A/B 产物）。
   - 无 `--yes` 时要求**手输版本号或 commit 短 sha**（不是 `y`）——`y/N` 对不可逆操作太廉价（`rick web reset` 只是删覆盖层，提升是替换生产内核）。
   - 留痕：追加一行到 `~/.rick/web/promote-log.jsonl`（`{at, actor:"tty", changes, sha256, gate_result, result, rollback_point}`）。
2. **Web API 路径（在一个 UI 会话里闭环）**：
   - `POST /api/web/promote/request`（token 鉴权）：服务端冻结一份「报价单」→ 写 `~/.rick/web/promotions/<id>.json`（`status:"pending"`），返回 `{id, changes, sha256, gate_result, approval_token}`；**`approval_token` 必须经带外通道交付人类**（CLI 里 `rick web promote --show-token <id>` 打印；或写入 `~/.rick/web/promotions/<id>.token` 由人类 `cat`）。理由：web token 在 localStorage（§5.1），沿用即等于无人确认；带外 token 才构成「人确实在场」。
   - `POST /api/web/promote/{id}/approve`（**同时要求 web token + `approval_token`，一次性、TTL≤10 分钟、用后作废**）：服务端校验 → 记录 `{approved_by:"web", approved_at, client_ip, ua}` → 触发提升；执行结果写回同一记录（`status: approved|running|succeeded|failed|rolled_back`，附 stderr 尾巴）。
   - UI：`Settings`（或 Jobs 页新增「提升」区）展示报价单 + diff & 文件清单 + 门禁结果，确认弹窗复用 `Dialog`（`false` 禁关闭），输入框要求粘贴 `approval_token` 后才激活「确认提升」按钮（等价于 CLI 的「手输版本号」）。
   - SSE：提升过程推 `promote_state` 事件（复用 hub），浏览器据此显示进度与最终结果；**重启必然断连**，客户端会 resync（§2.3）→ 状态由 REST 快照 + promote 记录补齐，UI 不会卡在「进行中」。
3. **门禁作为批准的前置条件**：`request` 阶段就跑 gate7（§4.3 阶段 A 全量 + E2E 冒烟），把 `{"pass":bool,"errors":[]}` 附在报价单里；`pass=false` 时 `approve` 直接 409 并在 UI 展示 errors。人类批准的是「已通过门禁的确定产物」（sha256 冻结），不是"一个分支"。

### 5.3 必须记录的审计字段（可核查 = 留痕的实质）

`{promotion_id, requested_at, requested_by(页面/CLI), dev_head, prod_head, changed_files(diff --stat), binary_sha256, dist_manifest_sha256, gate7_result, approved_at, approved_channel(cli|web), approval_token_id, executed_at, restart_window_ms, health_after, sessions_before/after, resume_report, rollback_point, result}`

---

## §6 生态先例可迁移性（逐条判定）

外部事实来源见 leaf-1（含 URL 与机制名）；下表是「本场景（Go 单进程 + 进程内 pi worker + 单机单用户 + 无 root/无 user systemd）」的迁移判定。

| 先例 | 关键机制 | 本场景判定 | 原因（谁持端口/会话/状态） |
|---|---|---|---|
| Kubernetes rollout / StatefulSet | `terminationGracePeriodSeconds`（自 preStop 起算）、`preStop`、PDB（只约束自愿驱逐）；**不感知会话**，长连接随进程死，重连与续传由应用层负责 | ⚠️ 部分可迁移 | **只借 drain 顺序与 graceful 预算**（= rick 已有：worker 先收 → HTTP drain 10s，§2.1）。会话恢复 K8s 不管 → 正是 §3 要做的事 |
| systemd socket activation（`Accept=no` + FD 传递；`FileDescriptorStoreMax`+`sd_notify FDSTORE=1` 让服务重启后拿回 listen FD） | systemd 持监听 socket，服务只 accept → **端口零中断**，但**已建连接随旧进程断** | ❌ 本机不可用（架构上可迁移） | 实测：`pid1=systemd`、`systemctl list-units` 可列，但 `systemctl --user` 报 "Failed to connect to bus"（`/run/user/<uid>` 为空、无 linger），且无 root ⇒ 建不了 unit。自研 FD 继承等价物 = §2.2 (iii')，成本高收益低 |
| nginx reload（USR2 起新 master + 旧 worker 优雅退出，靠 master 传 listen FD）/ SO_REUSEPORT | 多进程共享同一端口，旧 worker 继续服务既有连接 | ❌ 不装 / ⚠️ 仅当需要多后端 | `nginx/socat/haproxy/caddy` 均未安装；SO_REUSEPORT Python 侧可见（`hasattr(socket,'SO_REUSEPORT')`=True）但 Go 官方不暴露（leaf-1: golang/go#23696），需 `ListenConfig.Control` 自行 setsockopt。且**代理/多监听不解决进程内 worker 迁移** |
| Erlang/OTP hot code swap | 同 VM 内 current/old 双版本代码共存 + `code_change/3` 状态迁移回调 | ❌ 不可迁移 | Go 静态二进制无 VM/模块表；唯一可借的是**「显式状态迁移回调」思路** → 对应 §3.2 的启动对账 + intent |
| Docker live-restore | daemon 重启不杀容器（进程与端口延续），但官方明确"network & user input interrupted" | ⚠️ 部分可迁移 | 可借「重启对象与业务进程解耦」的架构直觉；但保不住交互通道，且 rick 没有 daemon/container 层 |
| Vite dev server / Next.js dev vs build+start；air/reflex/nodemon | dev 与 prod 是两个命令两个进程，产物不同；热重载工具是**杀进程重启**，内存态丢失 | ✅ 可迁移（做法上） | 「dev server 专用端口 + 生产不动」是业界常规；但**这些工具都不解决进程内会话**——rick 的 pi worker 在进程内，所以必须显式做 §3 |
| **可复用模式（提炼）** | ① **谁持端口**：由寿命最长的组件持 listen FD（systemd/父进程/自研 shim）；rick 当前是本进程持 → 重启必断端口（改造成本见 §2.2）② **谁持会话**：所有热重载先例都保不住进程内会话；rick 的会话 = 进程内 pi worker + 磁盘 jsonl ⇒ 「重启即中断」是结构性的，只能靠**恢复**而非**迁移** ③ **状态放哪**：落盘且可重建（sessions.json + jsonl + 新增 shutdown 快照/intent）④ **升级协议**：版本目录 + `mv -T` 原子切链（跨 FS 会退化为非原子）⑤ **drain 顺序**：停收新活 → 等既有流结束或超时 → 退出 |

---

## §7 风险清单（自进化链路全量失败模式）

| ID | 级别 | 失败模式 | 证据 | 缓解（推荐） |
|---|---|---|---|---|
| R1 | **高** | **同 HOME 的第二个实例污染生产会话态**：dev 实例启动即 `ReconcileOnStart` 把生产 active 会话写成 error 落盘；两实例对 `sessions.json` 全量互覆（last-writer-wins）；overlay `~/.rick/web/dist` 也共享，dev 构建直接改生产前端 | `sessions.go:154-173`（改状态并落盘）、`registry.go:102-125`（全量 save）、`static.go:63-71`（逐请求读 `$HOME/.rick/web/dist`）；**实测当前 `~/.rick/web.pid` 不存在 ⇒ 单例门禁形同虚设**（`handler/web.go:127-135`：文件缺失直接放行） | dev 实例一律 **HOME 隔离**（实测可行，§1 隔离节）；同时修单例：pid 文件按 `HOME+port` 命名或用 flock，并在启动时校验「端口已被占用但 pid 文件缺失」→ 显式告警 |
| R2 | **高** | **注册表损坏 ⇒ 生产起不来**（提升后新二进制读到坏 `sessions.json`/`web.json` 直接 return error，服务不启动；此时连 UI 都没有，只能靠文件恢复） | `internal/cmd/web.go` 的 `LoadSessionRegistry/LoadWorkspaceRegistry` 出错即 `return fmt.Errorf`；`registry.go` 解析失败即 error | 提升前把 `sessions.json/web.json/config.json` 复制到 `releases/<ver>/state-backup/`；启动失败时脚本自动回滚文件 + 切回上一版二进制；另建议把「注册表损坏」降级为「重命名坏文件 + 从空表启动 + 显式告警」 |
| R3 | **高** | **孤儿 doing pi × 自动重跑 = 同一 job 双写**：doing 的 pi 无 Setpgid、不绑 ctx，服务死后仍可能继续写 tasks.json/commit；恢复若直接重跑 DoingIn → 重复 commit、任务状态互相覆盖 | `internal/runtime/runtime.go:143-146,166`（`exec.Command` + `cmd.Wait()`，无 SysProcAttr）；关停只收 supervisor worker（`server.go:130-137`、`sessions.go:1745-1752`） | ① doing spawn 也 Setpgid + 写 `workers/<id>.pid`；② 重跑前按 pidfile/pgid 清理孤儿；③ 重跑前把遗留 `running` 归一化为 `pending`（`.rick/skills/rick-gates/helper.py:47-49` 会判 zombie 而 fail） |
| R4 | **高** | **提升后新内核起不来 ⇒ 全部会话不可用**（user 诉求的「不中断」彻底落空） | 提升 = 换二进制 + 重启；`/api/health` 是唯一免鉴权探针（`routes.go:849`） | 保留 N 版 + 原子切链（§1.5）；启动后 3 次健康检查失败即自动回滚（切链 + 重启 + 状态文件恢复）+ 报告；提升前 `bin/releases/<ver>/rick web --help` 冒烟 |
| R5 | 高 | **悬挂 toolCall 会话续跑 → 非法请求/语义错误**（未配对的 toolCall 被送进 provider） | leaf-2 P2/F9/F10：pi 不补 toolResult、不丢弃；全库 19/633 为悬挂态 | 悬挂态**只恢复 worker 不续跑**，UI 显式提示；续跑前用 `get_entries` 校验末尾配对 |
| R6 | 中高 | **自动恢复风暴打爆模型配额**：每条 resume 都加载大上下文（实测会话 12MB/3253 条），并在续跑时重放历史 | `supervisor.go:43` `MaxActive=8`；leaf-2 P5：配额耗尽**不报错**（返回普通 `stopReason:"stop"` + usage=0） | 队列 + 抖动 + 每次重启续跑预算（≤3）+ 每会话一次 + 明确「只恢复不续跑」默认；恢复后校验 usage/文本特征并报告 |
| R7 | 中 | **重复副作用**：被中断的 toolCall 可能已生效（写了文件/commit/推送）但未记录 → 续跑再执行一次 | leaf-2 P2 + 中断样本（assistant/toolUse 后无 toolResult） | 默认不续跑；续跑 prompt 要求「先核验上一工具是否已生效」；doing 只重跑 `status != success` 的 task（已完成 task 的 `commit_hash` 保留，`doing.go:144-166`） |
| R8 | 中 | **无锁双写同一 jsonl** → 会话历史交错/损坏 | leaf-2 P3（两进程可同开同一 session，无 lock 文件） | `workers/<id>.pid` 孤儿检测 + 恢复前强杀孤儿进程组；resume 前拒绝「已有活 worker」的同 ID spawn（supervisor 已做内存级防重，但跨进程无效） |
| R9 | 中 | **前后端版本错配**：只更 dist（覆盖层）不重启 → 旧后端服务新前端；只更二进制不撤覆盖层 → 新后端服务旧前端 | `static.go` 覆盖层优先；watcher 2s 轮询 + `frontend_reload` 自动刷新（`watcher.go:33-36`、`sse.ts:422-427`） | 提升时「二进制 + dist」同版推进（§1.4 推荐撤覆盖层让前端随二进制内嵌）；API 契约保持向后兼容（`sessions.json version:1` 未变） |
| R10 | 中 | **门禁自伤**：跑 gate 会重建 `bin/rick`（=生产二进制路径）、重写入库的 `web/dist`；残留 pi 检查既失灵又误报 | `gate5.py:27`、`gate6.py:43`、`gate1.py:47-49`/`gate2.py:53`、`gate2.py:55-57`；实测真实 worker argv 是 `pi`（`/proc/176315/cmdline`）→ `pgrep -f '[m]ode rpc'` 零命中 | 门禁只在 dev 工作树跑（gate 的 ROOT 由文件位置推导，复制即生效）；生产侧只读验收（health/config/sessions 计数）；残留检测改用 pidfile/pgid |
| R11 | 中 | **10s 关停窗口 + 非零退出码**：有浏览器在线时 `kill -TERM` 需 ~10s（SSE handler 不返回），日志报 `Error: graceful shutdown: context deadline exceeded`，`set -e` 脚本误判为失败 | 实测 10094ms + 该错误串（§2.1） | 提升脚本以「进程退出 / 端口释放 / health 200」判成功，忽略退出码；建议给 hub 加 shutdown 广播让 `ServeSSE` 退出（关停降至 ~0.1s）；另注意**关停窗口内 pid 文件仍在** → 新实例会被单例拒绝，必须等进程退出或先删 pid 文件（此时端口已释放） |
| R12 | 低-中 | **状态语义混淆**：`closed_at` 在 resume 后不清除（实测线上 active 行带着 21:59:18 的 closed_at）；`error` 既表示崩溃也表示"关停导致"；恢复报告若基于 status 会失真 | `sessions.go:1782-1793`（仅 closed/error 写 ClosedAt）、`:1073-1074`（resume 不清） | resume 时清 `ClosedAt`；新增持久化 `LastError/LastStatusReason`；恢复以 shutdown 快照 + intent 为准而非仅 status |
| R13 | 中 | **cwd-scoped session 查找**：工作区路径变化/重新注册到别处，resume 触发 pi 的交互式 y/N（"Fork this session into current directory?"），rpc 模式会吞掉 stdin 造成挂死 | leaf-2 P4 实测 | resume 前比对「session 文件所在 cwd 编码目录」↔ `ws.Path`，不一致则标 error 不 spawn；错误消息里给出两个路径 |
| R14 | 低 | UI 在提升/恢复期间显示假象（卡在「进行中」/不知道恢复了多少） | 现有 `session_state` 事件只覆盖单会话状态 | 增加 `promote_state` + `resume_report` SSE 事件 + REST 字段（报告落 `~/.rick/web/resume-report.json`） |
| R15 | 中 | **配额耗尽被当成正常收尾**：自动续跑静默无效、用户以为在跑 | leaf-2 P5/F14（usage 全 0 + `stopReason:"stop"`） | 续跑后检查 usage/消息特征，异常时标记会话为 `error(quota?)` 并进报告；恢复预算天然限流（R6） |

---

## R7 上报项（无法澄清 / 缺条件验证）

1. **悬挂 toolCall 是否真的触发 provider 400**：pi 侧已证不修复（P2，源码 + 构造实验），但端到端未验证——实验当日配额耗尽（P5），请求未到达消息校验。缺：可用配额 + 一次受控 E2E。
2. **SIGKILL 打断"流式生成中"的回合会落盘哪条记录**：是否写 `aborted` 标记未复现（现有 aborted 样本推测来自优雅 abort）。缺：可控中断注入实验。
3. **`steer`/`follow_up` 对 idle worker 是否等价于 prompt**（决定 §3.5 策略 D 是否可用）。
4. **`~/.rick/web.pid` 为何缺失**：当前生产单例门禁失效的原因是推测（曾被删除/写在别处），无直接证据；建议查 shell history 与 `~/.rick/web.log` 首行时间戳。
5. **生产重启瞬间"确实在跑"的精确集合**：无 shutdown 快照，无法回溯历史；只能由 `status=active` + `closed_at≈关停时刻` 间接推定。
6. **doing 孤儿 pi 的真实存活形态与写坏结果**：未构造「中断 doing + 重启」实验，R3 的严重性上限未实测。
7. **`npm run build` 的确定性**：是否会因工具链差异让 `web/dist` 变脏未验证（本研究刻意未在生产树跑构建，避免污染）。
8. **双写同一 jsonl 的损坏形态**：仅证"无锁"（P3），未构造并发写坏的样本。

## 叶子文件

- `research-L6-leaf-1.md` —— 生态先例（Unix 语义/K8s/systemd/nginx/OTP/docker/Vite，含外部 URL 与可迁移性）
- `research-L6-leaf-2.md` —— pi resume 实证（P1-P7、中断判定算法、续跑策略成本、未验证清单）
- `research-L6-leaf-3.md` —— 仓库事实（前端投放/构建版本/门禁/鉴权/端口与状态隔离/后台任务，逐条 file:line）
- 本次调研的实测原始输出：关键结论均已内联在本文档与各叶子中（cp/mv 的 ETXTBSY 与 `(deleted)`、符号链接切换、10094ms 关停窗口、隔离实例启动与 `/api/sessions=[]`、`pid 文件缺失`、gate 的 `pgrep` 失配）。保留的原文：`/tmp/l6leaf3_gotest.txt`（`go test ./...` 全 ok，EXIT=0）、`/tmp/l6gate.py`；leaf-2 的 /tmp/l6/ 会话副本与我的 .l6test/、/tmp/l6home/ 隔离实例在核对完毕后已清理。
- 安全性复核（本次调研结束时）：生产 pid 172761 仍监听 8413；`~/.rick/web/sessions.json` md5 与调研开始时一致（`11b1a724b7d0165d140e3fb2b865badf`，mtime 22:13:17 未变）；无 18413/18414 残留监听；无测试残留进程；仓库工作树无新增修改（仅本简报与既有 untracked 文件）。
