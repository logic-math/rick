# 交接单：macOS/arm64 上 rick web 起不来（2026-10-30）

> **接手人**：rick 新会话 ｜ **现场机**：macOS 27.0.1 / arm64 / 用户 `logic`
> **仓库**：`/Users/logic/workdir/AI_CODING/rick`（main，`ahead 92`，工作树干净）
> **状态目录**：`/Users/logic/.rick` ｜ **端口**：生产 6137 ／ dev 8414
> 本文只写**已验证的事实**与**待办**；推断处均显式标注「推断」。所有命令都可直接复制执行。

---

## 0. 一句话结论

**不是文件权限问题。** `bin/rick` 当时指向的是一份 **linux/amd64 的 ELF**，macOS（arm64）根本 exec 不了，报错伪装成了 `cannot execute binary file` / `Undefined error: 0`。
随后 release 的自动回滚把 `current` 指回**另一份同样不可执行的 ELF**，于是「发布失败」升级成「服务彻底起不来」。

已手工切到本机可用的 arm64 构建 `4fa5dd8-261030115302` 并拉起服务（见 §1），**但代码层没有任何修复** —— 下一次 `rick release` 仍可能复发。

---

## 1. 当前状态（2026-10-30 16:50 实测）

| 项 | 值 |
|---|---|
| 监听 | `*:6137`，pid **34061** |
| 运行二进制 | `bin/releases/4fa5dd8-261030115302/rick`（Mach-O arm64，5.1.0） |
| `/api/health` | `{"status":"ok","build_id":"4fa5dd8-261030115302"}` |
| 工作区 | 4 个：MT_INFER / **RICK-RSI**（本仓库）/ MOGIC_HOME / PLATOIM |
| 启动方式 | **本次由人工** `./bin/rick web --daemon ...` 拉起（不是 `start-web.sh`） |
| 状态文件 | `~/.rick/web.pid`=`34061`，`~/.rick/web.lock`=`34061` |
| dev 实例 | `*:8414` pid 76547，state=`rick-dev-home`，**与本案无关，勿动** |

> `~/.rick/config.json` 的 `web_token` 是访问凭证；本文不抄录。浏览器入口：`http://127.0.0.1:6137/?token=<web_token>`。

---

## 2. 现象（用户视角）

执行 `~/.rick/start-web.sh` 后：

- `~/.rick/web.log` 反复出现
  - `.../bin/rick: cannot execute binary file`
  - `.../bin/rick: Undefined error: 0`
- 间歇出现
  - `状态目录 /Users/logic/.rick 已被另一个 rick web 实例占用（web.lock 锁持有者 pid=11135）`

两条都像「权限/占用」问题，实际成因不同（见 §3、§4）。

---

## 3. 已核实的事实

### F1 — 生产二进制是 Linux 的，机器是 macOS arm64

```
$ uname -m
arm64
$ file -L bin/rick
bin/rick: ELF 64-bit LSB executable, x86-64, ... interpreter /lib64/ld-linux-x86-64.so.2
```

`bin/rick` 是软链：`bin/rick -> releases/current/rick -> releases/eb8048c-261012235415/rick`（ELF）。
`bin/releases/.last` 当时也是 `eb8048c-261012235415`。

### F2 — 同树里躺着一份**能用的** arm64 构建（今天 11:53 产出）

```
$ file bin/releases/4fa5dd8-261030115302/rick
Mach-O 64-bit executable arm64
$ ./bin/releases/4fa5dd8-261030115302/rick --version
Rick CLI version 5.1.0
```

用临时 state-dir 在 6199 端口冒烟：`/api/health` → `{"status":"ok","build_id":"4fa5dd8-261030115302"}`（指纹正确，**不是** `dev`）。

### F3 — 11:53 那次 promote 的真实失败明细

`~/.rick/release-state.json`：

```json
{"action":"promote","version":"4fa5dd8-261030115302","stage":"health","ok":false,
 "error":"health: 自动回滚后生产仍未就绪: ... 运行中的 build_id=\"dev\" 与本次提升版本 ... 不一致",
 "pid":11135,"build_id":"dev",
 "restart":{"stopped":null,"skipped":[11135],"build_id":"dev","build_id_matches":false,...}}
```

`restart.log_tail` 里同时有：

```
Error: 状态目录 /Users/logic/.rick 已被另一个 rick web 实例占用（web.lock 锁持有者 pid=11135）
/Users/logic/.rick/start-web.sh: line 12: .../bin/rick: cannot execute binary file
```

→ 「旧进程 11135 没停掉」+「新进程起不来」+「health 读到的是旧进程（`build_id=dev`）」三件事同时成立。

### F4 — 回滚把生产指向了不可执行的 ELF

回滚后 `releases/current -> eb8048c-261012235415`（ELF），`.last -> 4fa5dd8-...`（arm64）。
即：**回滚点本身也是坏的**，回滚动作没有做任何「能否执行」校验。

### F5 — 死 pid 的 `web.lock` **不会**阻塞启动（更正一个初期误判）

探针：在临时 state-dir 里写入 `web.lock = 999999`（不存在的 pid），服务照常启动，锁被改写为新 pid。
→ **本案里删 `web.lock` 并不是必要的**；「已被占用」只在锁持有者**仍存活**时出现。
（所以：不要再把“清锁”当成修复项，重点在 F3 的漏杀。）

### F6 — 生产树里混着多份 Linux 产物

| 路径 | 格式 |
|---|---|
| `bin/releases/eb8048c-261012235415/rick` | ELF x86-64 |
| `bin/releases/59ac918-260922143616/rick` | ELF x86-64 |
| `bin/rick.legacy-20260922-133423` | ELF x86-64 |
| `~/.rick/bin/rick` | ELF x86-64 |
| `~/.rick/bin/rick-import-darwin-arm64` | Mach-O arm64 ✅ |

`internal/env/relocate/import.go` 里已有 `wrongPlatformBinary()`（按 magic 拒绝错平台），但显然没覆盖 `~/.rick/bin/rick` 这条写入路径。

### F7 — `rick` 命令自身版本不统一

| 入口 | 版本 |
|---|---|
| `which rick` → `/Users/logic/bin/rick` | **5.0.15** |
| `~/.local/bin/rick` | 5.1.0 |

即：用户敲的 `rick release` 可能跑的是**旧代码**（早于 `4fa5dd8` 的归一化修复）。

### F8 — 前端覆盖层比后端旧

```
~/.rick/web/dist/assets/index-COEOVYQX.js        (10-15 构建)
bin/releases/4fa5dd8-.../dist/assets/index-B0jkiPLb.js   (10-30 构建)
```

`~/.rick/web/dist` 覆盖层优先于二进制内嵌 dist（`wiki/web-ui.md` 的约定），所以**当前页面 = 10-15 的前端 + 10-30 的后端**。

### F9 — `~/.rick/deploy-web.sh` 在 macOS 上必失败

脚本第二行 `cd /workdir/sunquan20/AI_CODING/rick/web` —— 这是旧 Linux 机路径，本机 `/workdir` 不存在。

---

## 4. 根因链

```
R1(直接)  current → linux/amd64 ELF  →  exec 失败（报错不指向架构）
   ↓
R2(机制)  promote 时旧实例 11135 未被 stopProd 认领（skipped=[11135], stopped=null）
          → 新进程被单实例锁/端口挡下、立即退出
          → waitHealthy 轮询到的是**旧进程**（build_id="dev"）
          → 与期望版本不等 → 判 health 失败
   ↓
R3(设计)  自动回滚不做可执行性校验 → current 切到 .last（同为 ELF）
          → 从“新版本没生效”变成“生产彻底不可启动”
   ↓
R4(卫生)  生产树长期混放 Linux 构建产物；错误信息把架构问题伪装成权限问题
```

**R2 的具体成因是本案最需要 rick 复现的一环**（`skipped` 恰恰说明「宁可不杀」的所有权断言退化了）。
`internal/env/release/release.go:1276` 的注释已经记录过同类事故（`linux→macOS 跨机暴露`），`4fa5dd8` 提交也声称修了 `EvalSymlinks` 归一与 `spawnDetached` 相对路径；但 11:53 仍 `skipped=[11135]`，说明还有未覆盖场景（**推断**：darwin 上 `proc.Cwd` 取不到（lsof/TCC 限制）时，归属断言直接退化为 false，而 `HOME` 兜底又因 `prodEnv` 只透了白名单变量而失效——需实测）。

---

## 5. 本次人工处置（临时状态，**不是修复**）

```bash
cd /Users/logic/workdir/AI_CODING/rick/bin/releases
ln -sfn 4fa5dd8-261030115302 current          # 1) 指回可执行的 arm64 构建

cd /Users/logic/workdir/AI_CODING/rick
rm -f ~/.rick/web.lock                        # 2) 清死锁（经 F5 证明非必要，仅为整洁）
export http_proxy=http://10.229.18.27:8412 https_proxy=http://10.229.18.27:8412 no_proxy=localhost,127.0.0.1
./bin/rick web --daemon --listen 0.0.0.0 --port 6137 --state-dir /Users/logic/.rick
```

效果：§1 的表格。**注意**：`current` 与 `web.pid` 都是手工态，任何一次 `rick release` 都可能重新打回原形。

---

## 6. 待办（按优先级；每条都给了落点与验收）

### P0 — 必须修，否则下次发布必复发

#### P0-1 promote / restart 前校验「目标二进制在本机能执行」

- **现状**：`file` 一看就知道是 ELF，但流水线照单全收。
- **期望**：在 promote 之前（而非 health 阶段）拦截：
  1. 读文件头 magic（darwin 上拒绝 ELF）——直接复用 `internal/env/relocate/import.go` 的 `wrongPlatformBinary()`；
  2. 跑一次 `<bin> --version`（带超时）做冒烟。
- **落点**：`internal/env/release/release.go`（promote / restart 阶段）。
- **验收**：
  - 单测：往 `releases/vX/` 放一个 ELF → `Promote` 返回 `build`(或 `gate`) 阶段错误，`current` 不变、**没有进程被杀**；
  - 手工：把 `current` 指回 `eb8048c` 态再 promote → 必须在 promote 前就拒绝。

#### P0-2 回滚目标同样要校验（不能把生产切到坏二进制）

- **现状**：`Rollback()` 无条件 `current → .last`。
- **期望**：回滚前校验 `.last`；不可执行则**不切链**、保留现场、错误信息点名「回滚点不可执行」，并提示可用的健康版本。
- **验收**：`TestRollbackRejectsNonExecutableTarget`；手工：`echo eb8048c-... > .last` 后 `rick release rollback` → 拒绝且 `current` 不变、服务仍在跑。

#### P0-3 复现并修复 `stopProd` 漏杀（skipped=[11135]）

- **复现建议**：临时 state-dir + 6199 端口起一个实例，再用 release 指向同一 state-dir 做 promote；观察 `stopped` / `skipped`。
- **必查**：
  - darwin 上 `proc.Cwd(pid)` / `proc.Exe(pid)` 是否因 lsof 权限返回 error；
  - `prodEnv()` 只透传白名单变量，`HOME` 兜底断言在什么条件下失效；
  - `findProdPIDs` 的批量快照在 darwin 上是否漏了这个进程。
- **验收**：连续两次 promote，第二次 `stopped=[旧pid]`、`skipped=[]`；补一条单测覆盖「cwd 不可读」场景（应回退到 exe/HOME 断言，而不是直接判 false）。

#### P0-4 health 失败信息要能直接定位

- **现状**：只报 `运行中的 build_id="dev" 与 ... 不一致`，看不出端口上是谁。
- **期望**：失败时输出 `old_pid` / `old_build_id` / 端口占用者（lsof）/ 新进程日志最后 20 行。
- **验收**：人为制造「旧进程占 6137」→ 报错直接写明 `6137 被 pid=X 占用，build_id=dev`。

### P1 — 健壮性与卫生

#### P1-1 `~/.rick/start-web.sh` 加固（用户明确要求）

- **现状**：前台 `exec`，失败无自检；终端一关服务就没了。
- **期望**：`--daemon`；启动后 `curl /api/health` 校验 `build_id`；失败打印 `web.log` 尾部并非零退出；开头加一行 `file -L bin/rick` 架构自检（非 Mach-O/arm64 直接给「二进制架构不匹配」而不是 `cannot execute binary file`）。
- **验收**：杀掉服务后执行脚本 → 自动后台起、回显 health；把 `current` 指回 ELF → 脚本给出架构错误、退出码非零。

#### P1-2 清理 Linux 产物 + import 兜底

- 处理 F6 的四份 ELF（删除或移到 `bin/legacy-linux/` 并标注「不可在本机运行」）；
- `~/.rick/bin/rick` 的写入路径补 `wrongPlatformBinary` 检查（`rick-import-*` 已按平台分拣，`rick` 本体没有）。
- **验收**：`find /Users/logic/workdir/AI_CODING/rick /Users/logic/.rick -name rick -exec file {} \;` 不再出现 `ELF`。

#### P1-3 `~/.rick/deploy-web.sh` 路径修正

- `cd /workdir/sunquan20/AI_CODING/rick/web` → `$HOME/workdir/AI_CODING/rick/web`（或从脚本自身位置推导）。
- **验收**：执行后 `~/.rick/web/dist` 与 `web/dist` 内容一致，浏览器刷新生效。

#### P1-4 统一 `rick` 版本

- `which rick` 是 5.0.15（F7），建议软链到 5.1.0 产物。
- **验收**：`rick --version` = 5.1.0；`rick release --help` 含最新参数。

#### P1-5 前端覆盖层对齐（F8）

- 处置二选一：`rm -rf ~/.rick/web/dist`（回落内嵌 dist），或从 `bin/releases/current/dist` 覆盖。
- **验收**：覆盖层 `assets/index-*.js` 文件名 == 发布包；页面无 404/白屏。

### P2 — 决策项

- **P2-1 20 个挂起会话**：`~/.rick/recovery-report.json` 里有 20 条（job_45、酒旅搜索、GTC 提报…），按设计**不会自动恢复**（`internal/web/recovery.go` 注释 J-L6-6/J-L6-7：pi 不修复悬挂 toolCall，自动续跑会重复副作用）。需人工在 Web 里逐个 `continue`，或明确归档。
- **P2-2 开机自启**：如需重启后自动可用，写 launchd plist（本次未做）。
- **P2-3 文档**：`wiki/installation.md` / `wiki/web-ui.md` 补一句「macOS 上生产二进制必须是 Mach-O arm64；Linux 构建产物不可直接投放」。

---

## 7. 安全复现环境（不要碰生产）

```bash
cd /Users/logic/workdir/AI_CODING/rick
TMPD=$(mktemp -d /tmp/rick-probe.XXXXXX); mkdir -p "$TMPD/pi/agent"
RICK_PI_AGENT_DIR="$TMPD/pi/agent" \
  ./bin/rick web --listen 127.0.0.1 --port 6199 --state-dir "$TMPD"
```

- 自定义 `--state-dir` 时 **必须**同时给 `RICK_PI_AGENT_DIR`（否则会拒绝启动，防止就地覆写生产的 pi runtime）。
- 演练 release 先加 `--dry-run` 看计划。
- **不要动** `*:6137`（生产）与 `*:8414`（dev，pid 76547）。
- 本机无直连外网：任何拉起 pi worker 的操作都要带
  `http_proxy/https_proxy=http://10.229.18.27:8412`、`no_proxy=localhost,127.0.0.1`。

---

## 8. 约束与禁区

1. **单实例语义**：`--state-dir` 上的 `web.lock` 只认「持有者是否活着」（F5），别再加「有锁就拒绝」的粗暴逻辑。
2. **别 kill dev**：8414 是独立 `rick-dev-home`，与本案无关。
3. **token 不要进日志/文档**：从 `~/.rick/config.json` 的 `web_token` 读。
4. **`~/.rick` 在工作区之外**：agent 写它需要显式授权（本机沙箱默认只允许 `workdir`）。
5. **回滚点必须保持可执行**：`.last` 是救命绳，不能指向坏二进制（P0-2）。

---

## 9. 关键路径速查

| 用途 | 路径 |
|---|---|
| 生产仓库 | `/Users/logic/workdir/AI_CODING/rick` |
| 生产二进制 | `<repo>/bin/rick` → `bin/releases/current/rick` |
| 版本目录 | `<repo>/bin/releases/<ver>/`（含 `rick` + `dist/`） |
| 当前/回滚点 | `bin/releases/.last` |
| 状态目录 | `/Users/logic/.rick`（config.json / web.pid / web.lock / web.log / sessions.json） |
| 前端覆盖层 | `~/.rick/web/dist`（优先于内嵌，`rm -rf` 即回落） |
| 启动脚本 | `~/.rick/start-web.sh`（前台 exec） |
| 挂起/恢复 | `~/.rick/suspend.json`、`~/.rick/recovery-report.json` |
| 发布状态 | `~/.rick/release-state.json`、`~/.rick/release.log` |

---

## 10. 整体验收（DoD）

- [ ] 在 macOS 上从「干净状态」跑一次 `rick release promote`：无需人工干预即可切到新版本，`health` 的 `build_id` == 版本号，`stopped` 非空、`skipped` 为空。
- [ ] 把 `current` 手工指向 ELF 后：promote 在**构建/门禁阶段**就失败，`current` 不变，服务不中断。
- [ ] `.last` 指向 ELF 时 rollback 被拒绝，生产继续跑。
- [ ] `start-web.sh` 一条命令完成「后台起 + 自检 + 失败可读」。
- [ ] `find … -name rick -exec file {} \;` 不再有 ELF。
- [ ] 页面与后端版本一致（覆盖层 `index-*.js` == 发布包）。

---

## 11. 源码位置（已读过，可直接下手）

| 主题 | 位置 |
|---|---|
| promote / restart / rollback / health | `internal/env/release/release.go`（`Restart`≈L760、`waitHealthy`≈L895、`Rollback`≈L999、`stopProd`≈L793、`ownedByProd`≈L1276、`prodEnv`≈L865） |
| 进程归属断言与批量快照 | `internal/env/release/release.go` + `internal/env/procfs` |
| 错平台识别（可复用） | `internal/env/relocate/import.go` 的 `wrongPlatformBinary()` |
| 单实例锁 / 启动 | `internal/cmd/web.go`（`--daemon` 的 setsid+nohup） |
| `build_id` 默认值 `"dev"` | `internal/cmd/root.go:26`（未注入 `-X cmd.BuildID` 时就是这个值） |
| 挂起/恢复契约 | `internal/web/recovery.go`（顶部注释即设计依据） |
| 发布 E2E（含 build_id 断言） | `scripts/self-evolve-e2e.sh`、`scripts/rsi-loop-e2e.sh` |

> 既有测试 `internal/env/release/release_test.go` 里已有「工具链/回滚」相关用例，新增 P0-1/P0-2 的用例请放在同文件，风格对齐。

---

## 12. 给之后 rick 会话的起手式

```text
读 wiki/handover-2026-10-30-web-start-macos.md，按 P0-1 → P0-4 顺序修：
1) 先在临时 state-dir + 6199 端口复现 P0-3（stopProd 在 darwin 上 skip 活进程），
   确认是 proc.Cwd 取不到还是 HOME 兜底失效，再改断言；
2) 加 promote/rollback 的可执行性校验（复用 wrongPlatformBinary + --version 冒烟）及单测；
3) 改 health 失败信息（旧 pid / 旧 build_id / 端口占用者 / 新进程日志尾部）；
4) 最后跑 release 全链路 e2e，断言 current 永远不会指向非原生二进制。

约束：不要动 6137（生产）与 8414（dev）上的运行实例；所有演练写 /tmp；
改 ~/.rick 需要人类授权；改完给出手工回滚步骤。
```

---

*本文由 2026-10-30 排障会话整理；事实均来自 `file`、`~/.rick/release-state.json`、`~/.rick/web.log` 与实测探针。*
