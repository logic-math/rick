# rick web 自进化（dev-web / release）运维手册

> 目标：**在 rick web 里改进 rick web**。开发期在完全隔离的 dev 实例里改前端/后端/CLI/pi runtime，
> 生产（8413）与其上所有运行中会话**不受影响**；交付期用一条命令把 dev 产物原子提升为生产。
> 设计依据：`.rick/jobs/job_36/doing/grilling/design-tree.md` 第 2 棵设计树（KR1 隔离 / KR2 开发闭环 /
> KR3 受控提升 / KR4 挂起可恢复），调研简报 `research-L4.md`、`research-L5.md`、`research-L6.md`。

---

## 1. 心智模型：谁持有什么

| | 生产（prod） | 开发（dev） |
|---|---|---|
| 端口 | `8413`（LAN 暴露，`--listen 0.0.0.0`） | `8414`（默认，`RICK_DEV_PORT` 可改） |
| HOME / 状态 | `~/.rick`（`web.json`/`web/sessions.json`/`web/dist` 覆盖层/`web.pid`） | `<dev-home>`（默认 `<祖父目录>/rick-dev-home`，`RICK_DEV_HOME` 可改） |
| 源码树 | 生产仓库工作树（`/workdir/sunquan20/AI_CODING/rick`） | git worktree（默认 `/workdir/sunquan20/rick-dev`，`RICK_DEV_TREE` 可改） |
| 二进制 | `<prod-repo>/bin/rick` → `bin/releases/current/rick`（符号链接） | `<dev-home>/bin/rick.dev.<sha7>-<时间戳>`（**文件名即构建指纹**） |
| pi 沙盒 | `~/.rick/pi/agent`（1.6G，含 runtime 自托管副本） | `<dev-home>/.rick/pi/agent`（独立；首次从生产拷 `auth.json/settings.json/models-store.json` 做种子） |
| 版本历史 | `bin/releases/<version>/{rick,dist}` + `.last` 回滚点（保留最近 3 版） | 无（worktree 里的分支即历史） |

**隔离开关只有一个：`HOME`。** rick web 的全部机器级状态都由 `os.UserHomeDir()` 派生
（`internal/web/statedir.go`）。所以：**dev 实例必须用独立 HOME**；只换端口不换 HOME 会让 dev 与生产
共用 `web.json`/`sessions.json`，第二实例启动时的对账逻辑会把生产的 active 会话改写成 error（实测事故）。

**跨进程互斥**：每个状态目录有一把 `flock`（`<state-dir>/web.lock`）。同状态目录起第二实例会被拒绝——
`web.pid` 可以被删，内核锁不会被绕过。

---

## 2. 一次性初始化

```bash
# 1) 建 worktree + dev HOME + pi 沙盒种子 + overlay 软链（幂等，可重复执行）
rick tools dev-web init

# 2) 起 dev 实例（构建 → 停旧 → 起新 → 健康 + 指纹复核）
rick tools dev-web up
#    DEV_UP bin=<path> pid=<pid> health_ms=<n> build_id=<指纹> port=8414
```

dev 实例起来后在你的浏览器里打开 `http://<本机IP>:8414/?token=<dev token>` 即可看到 dev UI
（token 存在 `<dev-home>/DEV_TOKEN`，也会写进 `<dev-home>/dev.env`）。

**建议**：在**生产 UI** 里新建一个会话，工作区选 `rick-dev`（dev 工作树）——这就是「改进 rick web 的那个
AI 会话」。它由生产实例托管（进程关系与 dev 实例无关），所以 dev 侧可以随便重建、重启、甚至 `kill -9`，
这个开发会话都不会掉线（`spec.Dir = ws.Path`，worker 的 cwd 就是 dev 工作树）。

---

## 3. 日常命令

### 3.1 `rick tools dev-web`

| 子命令 | 作用 | 典型耗时 |
|---|---|---|
| `init` | 幂等准备：worktree / dev HOME / pi 沙盒种子 / 前端依赖 / overlay 软链 | 首次 ~3s，之后 <1s |
| `build` | 构建 dev 产物（`npm run build` + `go build`，二进制唯一命名 = 指纹） | ~10–40s（热缓存） |
| `up` | 停旧 + 起新 + 健康轮询 + **指纹复核**（`--no-build` 复用最近构建） | ~1s |
| `restart` | `build` + `up`（**改完后端代码后的标准动作**） | ~15–45s |
| `status` | 汇报进程/端口/期望指纹/运行中指纹/是否一致（`--json` 给脚本用） | <1s |
| `down` | 停止 dev 实例（只杀**经归属断言**属于该 dev HOME 的进程） | ~1s |

前端改动更快：`npm --prefix web run build` 之后 overlay 生效即可（dev 的 `<dev-home>/.rick/web/dist`
是指向 `<dev-tree>/web/dist` 的**目录软链**，零拷贝），静态层逐请求检查 overlay，**不需要重启实例**；
浏览器 F5 就是最新前端。需要 HMR 时：`RICK_DEV_API=http://127.0.0.1:8414 npm --prefix web run dev`
（vite dev proxy 已参数化，默认指向 8414）。

### 3.2 `rick tools release`

```bash
rick tools release --dry-run        # 只跑门禁 + 构建 + 打印计划，不动生产、不重启
rick tools release --yes            # 正式提升（人类执行这条命令 = 人类确认）
rick tools release --yes --rollback # 一键回滚到上一版并重启
rick tools release --yes --detach   # 由「被生产托管」的 AI 会话执行时用（setsid 脱离，避免半途被杀）
```

提升的 6 步（任一步失败即中止，失败自动尝试回滚）：

1. **门禁**：在 **dev 树**里跑 `go test ./...` + `npm run build`（绝不写生产 `bin/`）
2. **构建**：`<prod-repo>/bin/releases/<version>/{rick,dist}`，`version=<sha7>-<时间戳>`，
   并以 `-ldflags` 把 `build_id=version` 注入二进制
3. **原子换链**：记录回滚点 `.last` → `ln -sfn` + `mv -T` 原子替换 `bin/releases/current`
   → `bin/rick` 指向 `releases/current/rick`（同文件系统 `mv` 覆盖正在运行的二进制是原子的；
   `cp` 会被内核以 `Text file busy` 拒绝——不用担心"替换到一半"）
4. **前端同版推进**：把 `releases/<ver>/dist` 原子投放到 `<state-dir>/web/dist`（旧覆盖层留 `dist.prev`）
5. **重启生产**：停旧（pid 文件 + `/proc` 扫描 + **归属断言**，断言不过宁可不杀）→ 启动 → 轮询 `/api/health`
   并要求 `build_id == 本次版本`（不是则判失败，**不留一个"跑着旧版本却报成功"的假象**）
6. **恢复报告**：打印 `RELEASE_RECOVER suspended=N ...` + 挂起会话 id 清单

退出码：`0` 成功 / `2` 门禁失败 / `3` 构建失败 / `4` 提升或重启失败 / `5` 健康或指纹校验失败。

---

## 4. 恢复语义：平台自动回来，会话挂起待人工一键恢复

这是本设计**最重要的安全性决策**（human 裁决 J-L6-6/J-L6-7）：

| 对象 | 重启（升级）后 | 谁触发 |
|---|---|---|
| **平台**（web 服务 + UI + SSE） | **自动回来**：`/api/health` 可用、状态文件完整、浏览器 SSE 携带陈旧游标 → 服务端回 `replay_overflow` → 前端清游标 + REST 全量重拉（**UI 无感**） | 自动 |
| **会话**（plan/easy/ctrl/human-loop/learning/交互 dream） | 标记为 **`suspended`（挂起）**，**不自动恢复** | **人工**：UI 上点「▶ 恢复继续」→ `POST /api/sessions/{id}/continue` |
| **后台 job**（doing / 后台 dream） | 同样**挂起**，**不自动续跑** | **人工**：点「继续执行」→ 归一化 `running → pending` 后重跑剩余 task |

为什么**不做**自动恢复（两条实测依据，都是真金白银的教训）：

1. **pi 不修复「悬挂 toolCall」**：会话末尾若是 `stopReason=toolUse` 的 assistant 消息、且它的 toolCall
   没有对应 toolResult，自动续跑会把这条非法前序重放给模型 → 可能**重复副作用**（重复写文件、重复提交、
   重复发请求）。人工确认天然避免了这个问题。
2. **配额耗尽不报错**：模型侧配额用完时返回的是普通 assistant 文本 + `stopReason=stop` + `usage` 全 0 —— 
   rick 无法用 stopReason 区分「答完」与「被拒」。自动恢复会静默空转、烧掉上下文，还让人以为在跑。

关停时的 intent 会落盘：`<state-dir>/suspend.json`（会话 id / pi 会话 / lastEntryID / 是否 busy），
`<state-dir>/recovery-report.json` 汇总「谁被挂起 / 谁已恢复 / 谁失败」，UI 顶部横幅与 `GET /api/recovery` 都会读它。

> 注意：**doing/dream 的续跑不是幂等重放**——被中断的那个 task 会重跑一次。这是设计取舍：
> doing 的语义本就是「按 task 状态续跑」，task 内的门禁与 commit 纪律负责兜底。

---

## 5. 故障排查

| 现象 | 原因 | 处置 |
|---|---|---|
| `dev-web up` 报 `health` 失败 | 端口被占 / 构建产物不可执行 / dev 状态目录不可写 | 看 `<dev-home>/dev.log` 尾部（`dev-web status` 也会带 log tail）；换 `RICK_DEV_PORT` |
| `dev-web up` 报 `fingerprint` 不一致 | 旧进程没被换掉（僵尸进程占着端口） | `rick tools dev-web down` 再 `up`；必要时按 `pgrep -af '<dev-home>'` 核对后手工收尸 |
| 端口被占（`EADDRINUSE`） | 生产占 8410-8413/8415/8420；dev 默认 8414 | `ss -ltnp \| grep <port>` 确认占用者，再改端口 |
| 启动报「另一个 rick web 正在使用该状态目录」 | flock 生效（**这是保护**，不是故障） | 用不同 `--state-dir`；或先停掉占用该状态目录的实例 |
| `release` 失败在 `stage=gate` | 测试/前端构建不过 | 先本地修绿；`--dry-run` 预演一遍再正式提升 |
| `release` 失败在 `stage=health` | 新进程没起来 / `build_id` 不匹配 | 看 `<state-dir>/web.log` 尾部；`rick tools release --yes --rollback` 回到上一版 |
| 提升后 UI 是旧前端 | 覆盖层未更新或浏览器缓存 | `release` 会同步投放 dist；硬刷新（`index.html` 是 `no-cache`，hash 化资源天然换名） |
| 有 `pi` 进程残留 | 服务被 `kill -9`（跳过收尸路径） | `rick tools dev-web down` / `release` 的 `stopProd` 各自带归属断言可清；**不要**无差别 `pkill pi`（会杀掉生产 worker） |
| AI 会话执行 `release` 时自己被杀 | 该会话由生产实例托管，重启必然连坐 | 用 `--detach`（setsid 脱离）；或请人在终端里执行 |

---

## 6. 已知边界（诚实记录）

1. **dev 工作区守卫的触发条件偏窄（本次验收发现）**：守卫只在「`--state-dir` 与 `$HOME/.rick` **不同**」
   时安装（`IsDevStateDir`）。而 `rick tools dev-web` 的实际形态是 **HOME 被换掉、state-dir 正好等于
   `$HOME/.rick`** → 该形态下守卫**不安装**，dev 实例可以注册生产已注册的工作区（`POST /api/workspaces`
   返回 201）。影响：若有人在这种 dev 实例里注册并起会话跑生产工作区，会话会写该工作区的 `.rick/`
   （与生产共享）。**缓解**：dev 侧只注册 dev 工作树（`dev-web init` 就是这么做的）；建议后续把守卫触发
   条件改为「`ProductionStateDir() != ""`（当前 HOME ≠ passwd home）**或** IsDevStateDir」，见
   `scripts/self-evolve-e2e.sh` 的 `FINDING guard_not_installed_for_home_swap` 复现点。
2. **`release --prod-home` 不会连带改默认启动脚本路径**：`--start-script` 默认仍取**真实家目录**的
   `~/.rick/start-web.sh`。演练/测试场景若只改 `--prod-home/--port` 而不给 `--start-script`，
   重启步骤会去跑**真生产**的启动脚本。**纪律**：任何非真生产的演练都必须显式传 `--start-script`。
3. **doing 门禁的 helper 路径硬编码**：`internal/handler/doing.go` 解析 gates `helper.py` 时用
   `UserHomeDir()`，不尊重 `RICK_PI_AGENT_DIR` → dev 实例里跑 doing 仍读生产的 helper（只读，无写风险）。
4. **PWA service worker 注册必然失败（独立议题）**：`web/dist/sw.js` 的 precache 清单不含 `index.html`，
   却调用 `createHandlerBoundToURL("index.html")` → workbox 抛 `non-precached-url`。好处是目前不会造成
   「改了看不到」；**一旦修好 SW 就必须同时解决 SW 缓存**，否则前端热更会退化（dev 构建可用
   `RICK_DEV_NO_PWA=1` 禁用 PWA）。
5. **`web/dist` 已入库**：dev 树自带的 `.rick/` 是**创建 worktree 时的快照**，可能与生产分叉（例如 job 进度）。
   dev 会话若读 `.rick/jobs/<job>/...` 读到的可能是旧副本——需要生产最新状态时用绝对路径读生产树。
6. **workspace 级写冲突仍靠纪律**：本次用 flock 只覆盖了「状态目录」这一层。两个实例若共享同一个
   workspace 路径，`tasks.json` 非原子写、job 目录 TOCTOU、`git add -A` 混提交依旧可能发生（机器级状态隔离了，
   workspace 级没有）。**纪律**：dev 实例只注册 dev 工作树。
7. **自动恢复缺失是有意的**：见 §4。若将来要加，必须先解决悬挂 toolCall 与配额静默两个前提。
8. **`~/.rick/bin/rick` 是历史遗留**（8/24 的旧 7.4MB 二进制，未被使用）；真身是生产仓库工作树里的
   `bin/rick` → `bin/releases/current/rick`。

---

## 7. 验收

```bash
# 全链路端到端（可重放、幂等；只读生产，release 只对模拟生产操作）
bash scripts/self-evolve-e2e.sh
# 末行输出：{"pass":true,"steps":[...],"prod_touched":false}

# 各层门禁（实现期）
python3 .rick/jobs/job_36/plan/gates/gate7.py    # 隔离底座 + 构建指纹
python3 .rick/jobs/job_36/plan/gates/gate8.py    # dev-web 闭环 + 挂起语义
python3 .rick/jobs/job_36/plan/gates/gate9.py    # 前端挂起 UI + release
python3 .rick/jobs/job_36/plan/gates/gate10.py   # 端到端 + 文档 + 生产回归
```

E2E 断言覆盖：隔离（状态/flock/守卫）、前端热更（进程不重启）、后端源码改动生效（`rick_version` 探针）、
挂起语义（`suspended` 而非 error + 未自动恢复 + `/continue` 归一化）、提升与回滚（模拟生产上 `build_id` 逐版校验）、
生产只读回归（`sessions.json`/`web.json` 指纹 + 8413 健康）。详细证据见
`.rick/jobs/job_36/plan/sim-report.md`。

> 生产是活的（在被使用的实例上，每次会话活动都会写注册表），所以「指纹不变」只在**同一次运行内**有意义——
> E2E/gate10 都是首尾各测一次、在单次运行内比对，不做跨运行比较。
