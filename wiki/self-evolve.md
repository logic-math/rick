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

## 4b. ⚠️ release 必须用 --detach（2026-09-22 生产中断事故的教训）

`rick tools release` **永远**带 `--detach` 执行（setsid 脱离调用方进程树）。原因：release 会「先停生产再起新」，若调用方（AI 会话的 bash、终端复用器）在中间被中断，release 子进程被连带杀死 → 生产停留在「已停未起」。已实测：执行 shell 被中断导致生产中断 ~10 分钟，恢复 = 直接重跑 `~/.rick/start-web.sh`（换链已原子完成）。

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

---

## 8. RSI 自进化：让 rick 改进 rick（制度化的唯一入口）

自进化**不是**靠人肉记住流程，而是靠一条 loop —— `.rick/loops/rick-rsi-loop.md`（`rsi-loop`）。
它是 rick 源码的一部分，与代码同仓、同发布链，因此流程本身也会随迭代演进。

### 8.1 怎么用（三步）

1. **在生产 UI 里新建会话** → 类型选 **`RSI 自进化`**（`rsi`）→ 工作区选 **dev 工作区**（`rick-dev`，见 §10 注册步骤）
   - CLI 等价入口：`rick rsi`（在 dev 工作树内执行）
2. **loop 自动加载**：后端把 `rick-rsi-loop` 的**全文**注入本次会话的系统提示词，并把 loop 文件登记为
   `_method_file`（resume 后由 `--append-system-prompt` 重新注入，所以 loop 的后续修改在 resume 后依然生效）。
   → 也就是说：**启动这个会话 = 必然按 loop 执行**，不依赖 agent 自觉读文档。
3. **按 loop 的状态机 S0→S7 走完**一次迭代：

| 状态 | 做什么 | 命令 | 出口判据 |
|---|---|---|---|
| S0 设计 | grilling 出设计树，判断节点交人类裁决 | — | 设计树每层达标 + `grilling_gate` |
| S1 隔离开发 | 只在 dev 工作区改；前端热更 / 后端重建 | `rick tools dev-web init/status/up`；前端 `npm run build`（dev 树）；后端 `rick tools dev-web restart` | dev 侧 `build_id` 已换（新构建真的在跑） |
| S2 层门禁 | 逐层跑门禁 | `python3 .rick/jobs/<job>/plan/gates/gateN.py` | 全绿才下钻 |
| S3 演练 | 给人类看计划 | `rick tools release --dry-run` | `prod_untouched=true` |
| **S4 人类确认** | 唯一放行点：把计划 + 影响面呈报人类 | 人工 | `doing/rsi/approval.md` 写入 `APPROVED by=human at=<时间>` |
| S5 提升 | 源码合并 + 产物换链 + 重启 + 指纹校验 | `rick tools release --merge-source` | `/api/health` 的 `build_id` == 本次 version |
| S6 恢复 | 重启后会话**挂起**，人类在 UI 逐条点「恢复继续」 | UI 按钮 / `POST /api/sessions/{id}/continue` | `doing/rsi/resume.md` |
| S7 留痕校验 | 机器校验产出评估表 | `rick tools rsi_check --job <job> --json` | `pass=true` → 迭代成功退出 |

**人类确认点只有一个：S4。** 其余步骤都可自动执行，但**没有 S4 的证据就不算完成** ——
`rsi_check` 把 `approval.md` 里是否有 `APPROVED by=human` + 时间戳作为最关键的硬门槛。

### 8.2 为什么必须走 loop（而不是"记得这么干"）

- **机制 vs 制度**：`dev-web` / 门禁 / `release` / 挂起恢复是**机制**；loop 是**制度**。只有机制时，
  每次（或换个人、换个会话）改 rick 都要重走试错；loop 把「怎么安全地改」固化成可复制流程。
- **注入而非自觉**：`rsi` 会话类型把 loop 全文塞进系统提示词 —— "必须使用"由入口保证，不靠 agent 记性。
- **产出可机器校验**：loop 的「产出评估」表（6 项）由 `rsi_check` 逐项校验。
  证据目录：`<ws>/.rick/jobs/<job>/doing/rsi/{dev-iterations,gates,approval,release,resume}.md`；
  第 6 项 `prod-health` 是**实时探测** `GET /api/health` 并要求 `build_id` == release 记录的 version ——
  这是**证伪**手段（`release.md` 只是自述，只有探测能证明生产真的跑上了这次构建）。
- **防自欺**：`rsi_check --init` 生成的骨架**永远不通过**（残留 `<!-- TODO` 标记即视为未填写）。
- **工作区守卫**：`rsi` 会话拒绝三种非法工作区（HTTP 400 + 中文原因）：
  非 rick 源码树 / 缺 `.rick/loops/rick-rsi-loop.md` / **指向生产仓库根**（后者最危险：会绕过 release 门禁直接改生产源码）。

## 9. 源码合并：`release --merge-source`

`release` 的默认行为只提升**产物**（二进制 + 前端覆盖层）。带上 `--merge-source` 才把 **dev 分支合并进生产分支**，
让源码与二进制同版生效：

```bash
rick tools release --merge-source --dry-run     # 先看计划（不碰生产）
rick tools release --merge-source               # 人类确认后执行
rick tools release --no-merge-source            # 显式只要产物、不动源码
```

- **前置条件**：生产工作树必须干净（否则中止且**零改动**，报错会列出脏文件）；生产仓库若存在未完成的合并（`MERGE_HEAD`）也会被拒绝。
- **冲突语义（刻意设计）**：冲突发生 → 执行 `git merge --abort` 把生产工作树**恢复到干净状态** → 命令报错并列出冲突文件清单。
  **交 AI 修复后重跑**：在 dev 树内 `git merge main` → 解冲突 → 提交 → 重新 `rick tools release --merge-source`。
  （刻意不做自动解冲突：错误合并的代价远高于多跑一次。）
- **回滚语义**：`rick tools release --rollback` **只回滚产物**（二进制/覆盖层切回上一版），**不回滚源码** ——
  需要回退源码请用 `git`（例如 `git revert` 那个 merge commit）。
- 留痕：`RELEASE_MERGE merged=true branch=<dev 分支> main=<生产分支> commit=<merge commit> files=<n>`；
  合并结果同时写进 `doing/rsi/release.md` 的 `version=` / `rollback_point=` 供 `rsi_check` 校验。

## 10. 把 dev 工作区注册到生产 UI（让 RSI 会话能在生产里启动）

RSI 会话必须在 **dev 工作区**运行（生产仓库根会被守卫拒绝），所以要让**生产 UI** 能选到它，需要在生产实例的
注册表里加一条 **additive 且可逆** 的记录：

```bash
PROD_TOKEN="$(python3 -c 'import json,os;print(json.load(open(os.path.expanduser("~/.rick/config.json")))["web_token"])')"

# 1) 注册 dev 工作区（rick-dev）——只新增一条记录，不影响任何现有会话/job
curl -s --noproxy '*' -X POST -H "Authorization: Bearer $PROD_TOKEN" -H 'Content-Type: application/json' \
  -d '{"path":"/workdir/sunquan20/rick-dev","name":"rick-dev"}' \
  http://127.0.0.1:8413/api/workspaces

# 2) 复查：应能看到 rick-dev（其余工作区原样）
curl -s --noproxy '*' -H "Authorization: Bearer $PROD_TOKEN" http://127.0.0.1:8413/api/workspaces

# 3) 撤销（可逆）：删掉该工作区注册（只移除注册，不删目录、不动它的 .rick）
curl -s --noproxy '*' -X DELETE -H "Authorization: Bearer $PROD_TOKEN" \
  http://127.0.0.1:8413/api/workspaces/<rick-dev 的 id>
```

之后在**生产 UI**：新建会话 → 类型 `RSI 自进化` → 工作区 `rick-dev` → agent 会按 loop 走完 S0→S7。

> 若不想动生产注册表：**dev 实例自己的 UI**（`http://<host>:8414/?token=<dev token>`）注册表里已经有 `rick-dev`，
> 可以作为 RSI 会话的替代入口 —— 两种入口跑的是同一套 loop 与工具。

## 11. 验收（本增量）

```bash
bash scripts/rsi-loop-e2e.sh                     # RSI 全链路（模拟生产；末行 {"pass":true,...,"prod_touched":false}）
python3 .rick/jobs/job_36/plan/gates/gate11.py   # loop 制度载体 + loops_check
python3 .rick/jobs/job_36/plan/gates/gate12.py   # rsi 入口绑定 + 守卫
python3 .rick/jobs/job_36/plan/gates/gate13.py   # rsi_check + release --merge-source
python3 .rick/jobs/job_36/plan/gates/gate14.py   # E2E + 文档 + 生产回归
```

`rsi-loop-e2e.sh` 覆盖：loop 注入（提示词含 loop 全文与机制命令）、三类非法工作区守卫（400 + 中文原因）、
`rsi_check` 的 fail/骨架-fail/pass 与两条负例（门禁 `pass=false`、`version` 与生产 `build_id` 不符）、
`--merge-source` 的「无冲突成功 / 冲突中止且工作树恢复干净 / AI 修复后重跑成功」、以及真实生产只读回归。

> **真实生产的第一次端到端仍由人类执行**：脚本只覆盖可在隔离环境验证的部分；真实 release 会重启生产
> （所有在跑会话变「挂起」，需人工逐条恢复）。
