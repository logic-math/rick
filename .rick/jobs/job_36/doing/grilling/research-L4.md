# research-L4 简报 —— 隔离层（dev 实例与生产的隔离维度）

元信息：阶段 L4 / KR1 隔离实例 | 时间基准 2026-09-20 22:10–22:30 (+08:00) | 仓库 `/workdir/sunquan20/AI_CODING/rick` @ `a60035a` (main) | 生产实例：pid 172761 `./bin/rick web --listen 0.0.0.0 --port 8413`（`~/.rick/start-web.sh` 启动，HOME=/home/hadoop-recsys），实测 `GET /api/config` → 200 `{"rick_version":"4.4.15","port":"8413"}`（`curl --noproxy '*'`，本机 http_proxy 拦截 127.0.0.1）

## 结论速览（先读这 9 条）

1. **HOME 是唯一的全局隔离开关**：rick web 的机器级状态（web.json / sessions.json / archived.json / job-names.json / web/dist 覆盖层 / web.pid）全部由 `os.UserHomeDir()` 派生，**没有任何环境变量或 flag 可重定向**。→ dev 实例必须用独立 HOME（实测：不同 HOME 两实例并存 OK）。
2. **已存在但残缺的 `_dev` 约定**：二进制名以 `_dev` 结尾只切换 `config.json`（`~/.rick_dev/config.json`）与 CLI 的 cwd 级 `.rick_dev`；**web 状态、web.pid、pi agent dir 仍是 `.rick`**（实测 `rick_dev` 只生成 `~/.rick_dev/config.json` + `~/.rick/web.pid`）。→ 单靠改名**不足以**交付 KR1，且会与生产抢同一个 pid 文件。
3. **执行兜底靠 bind 而非 singleton**：pid 文件缺失/被删时，同 HOME 的第二实例可启动并被拒于 `EADDRINUSE`（实测 18996 与 18999 并存）。→ dev 用独立 HOME 后 singleton 天然不冲突，但**不能靠现有 singleton 阻止 dev 误共享生产状态**。
4. **现网 singleton 已失效（观察事实 + 机制实测）**：生产 172761 运行中而 `/home/hadoop-recsys/.rick/web.pid` **不存在**（`ls` exit=2）。机制已复现：删除 pid 文件后同 HOME 可再起实例（18996 成功）。→ 当前若有人用**同一 HOME、不同端口**起 dev，会被放行并直接读写生产 sessions.json/web.json。置信度：漏洞高；历史成因（TOCTOU 竞态 + EADDRINUSE 退出路径的 `defer os.Remove`）中。
5. **端口**：默认 `--port 6137`（`internal/cmd/web.go:65`），生产用 `--port 8413`。实测 `ss -ltnp` 占用：8410/8411/8412/8413/8415/8420（0.0.0.0）+ 22/111/…  ；**8414 空闲**（start-web.sh 注释称历史上曾用 8414）。→ 建议 dev = `--port 8414 --listen 127.0.0.1`（不暴露局域网）。
6. **pi agent dir 是唯一 env 可重定向项**：`RICK_PI_AGENT_DIR` > `$HOME/.rick/pi/agent`（`internal/runtime/agentdir.go:22-30`），经 `AgentEnv()` 注入 `PI_CODING_AGENT_DIR` 给所有 pi 子进程。→ 是否共享生产 pi 配置（1.6G / 1943 个 session jsonl / auth.json / settings.json / `runtime` 自托管 pi 副本）是 L4 的独立决策点（见 §2）。
7. **前端 overlay 随 HOME 自然隔离**：`overlayDist = WebStateDir()/dist`（`internal/web/server.go:66-68`、`154-158`），`static.go:52-91` all-or-nothing。但 `web/vite.config.ts:57-60` 的 dev proxy `/api` **硬编码 `http://127.0.0.1:6137`**（既不是生产 8413，也不是 dev 8414）→ 需参数化。
8. **共享源码工作树是最贵的泄漏**：生产工作区注册表里已有 `a8ceb938 = /workdir/sunquan20/AI_CODING/rick`（`~/.rick/web.json`），且 worker cwd = `ws.Path`（`internal/web/sessions.go:434,987,1053`）、job 目录硬编码 `<ws.Path>/.rick`（`sessions.go:355,908`、`routes.go:469`、`watcher.go:118`）→ dev 会话若指向生产仓库根，就会**直接编辑生产源码树并写生产 `.rick/jobs/`**，同时被生产 Jobs 看板看到。
9. **交付底线**：dev 实例 = 独立 HOME + 独立端口 + 工作树（不含生产 `.rick/jobs`）+ 自己的 web.json；pi agent dir 与二进制路径是**两个必须显式裁决**的共享点（§2 / §3 / §7）。

## 问题索引

| # | 问题 | 节 | 结论一句话 |
|---|---|---|---|
| Q1 | 状态与路径全依赖清单 | §1 | 机器级 6 项 + pi 沙盒 1 项 + workspace 级若干；仅 1 项可 env 重定向 |
| Q2 | pi 运行时依赖共享/独立 | §2 | 建议**独立 AgentDir**（代价：重建 auth/settings + runtime 安装） |
| Q3 | 源码工作树隔离 | §3 | 推荐 git worktree（但需处理 .rick 副本 / web/dist 已提交 / bin 输出） |
| Q4 | 单例与端口冲突 | §4 | singleton 按 HOME + pid 文件，无锁、可被删文件绕过；dev 用 8414 |
| Q5 | 双实例写冲突清单 | §5 | 机器级状态原子 rename 但无锁（丢更新）；workspace 级 tasks.json 非原子写 |
| Q6 | 前端 overlay 隔离 | §6 | overlay 随 HOME 隔离；vite proxy 6137 需改；frontend_reload 不跨实例 |
| Q7 | 最小隔离集结论 | §7 | dev 启动配置表 + 共享项 + 剩余泄漏点 |

---

## §1 状态与路径全依赖清单（Q1）

结论：**机器级状态 100% 由 `os.UserHomeDir()` 派生，零 env / flag 可重定向**；唯一 env 可重定向的是 pi agent 沙盒（`RICK_PI_AGENT_DIR`）；唯一的「改名切换」是 `config.json` 与 CLI 级 `.rick_dev`。

### 1.1 机器级（web 服务，HOME 派生，不可重定向）

| 路径 | 用途 | 决定者 | 代码位置 |
|---|---|---|---|
| `~/.rick/web.json` | workspace 注册表（生产已注册 7 个工作区，含 `a8ceb938=/workdir/sunquan20/AI_CODING/rick`） | `os.UserHomeDir()` | `internal/web/web.go:34-40`；加载 `internal/cmd/web.go:81` |
| `~/.rick/web/sessions.json` | 会话注册表（生产 26397 B，22:13 仍在写） | 同上 | `web.go:44-46`；`cmd/web.go:89` |
| `~/.rick/web/archived.json` | 归档注册表（110 B） | 同上 | `web.go:51-53`；`cmd/web.go:85` |
| `~/.rick/web/job-names.json` | job 显示名（181 B，权限 0600） | 同上 | `web.go:58-60`；`cmd/web.go:93` |
| `~/.rick/web/dist/` | **前端 overlay（优先于 embed）** | 同上 | `web.go:25-31` → `server.go:66-68`（装配）、`154-158`（watcher） |
| `~/.rick/web.pid` | **singleton 判活文件** | 同上 | `internal/handler/web.go:115-121`（镜像 `web.go:63-68`） |
| `~/.rick/web/src/` + `~/.rick/web/.rick-managed` | `rick web customize` 抽取的前端源码基线（幂等 marker） | 同上 | `internal/env/web.go:32-38, 51-76` |
| `~/.rick/config.json` | 全局配置（`web_token` / `pi_path` / `pi_extra_args` / git identity / human_loop 权重） | `UserHomeDir()` **+ 二进制名以 `_dev` 结尾则用 `.rick_dev`** | `internal/config/loader.go:12-27`（路径）、`:35-68`（默认值） |

实测（`ls -1a ~/.rick/`）：`bin config.json config.json.bak.20260817110141 deploy-web.sh pi start-web.sh web web.json web.log` +（`~/.rick/web/`）`archived.json dist job-names.json sessions.json`。**注意 web.pid 缺失**（见 §4）。

### 1.2 pi 沙盒（唯一 env 可重定向项）

| 路径 | 用途 | 决定者 | 代码位置 |
|---|---|---|---|
| `<AgentDir>`（默认 `~/.rick/pi/agent`，实测 1.6 G） | 托管 pi 配置根 | **`RICK_PI_AGENT_DIR` > `$HOME/.rick/pi/agent`** | `internal/runtime/agentdir.go:22-30` |
| `<AgentDir>/settings.json`（367 B，0600）、`auth.json`（95 B，0600） | pi 设置与认证 | 随 AgentDir | `agentdir.go:33-35`、`internal/env/settings.go:31-33` |
| `<AgentDir>/sessions/--<cwd-slug>--/*.jsonl`（实测 1943 个 jsonl） | pi 会话日志；**目录名由 cwd 派生** | 随 AgentDir + 子进程 cwd | `PI_SESSION_FILE` 实测样例（生产进程 env）；`internal/prompt/dream_prompt.go:182` |
| `<AgentDir>/runtime/` + `runtime/node_modules/.bin/pi` | **rick 自托管 pi 副本**（npm prefix；可被 `pi install`/升级覆盖） | 随 AgentDir | `agentdir.go:41-48` |
| `<AgentDir>/{extensions,themes,skills,subagents,missions,npm,bin,models-store.json,run-history.jsonl,web-search-cache}` | 扩展/主题/技能/子 agent/缓存 | 随 AgentDir | 实测 `ls ~/.rick/pi/agent/` |
| `~/.rick/pi/agent/extensions/rick-gates/helper.py` | doing 终态门禁兜底 | **`UserHomeDir()` 硬编码 —— 不读 `RICK_PI_AGENT_DIR`** | `internal/handler/doing.go:235-238` |
| `~/.pi/agent/settings.json` | 一次性迁移读取（theme/packages 种子） | `UserHomeDir()` | `internal/env/settings.go:37-44` |

### 1.3 workspace 级（由注册表内容或 cwd 决定）

| 路径 | 用途 | 决定者 | 代码位置 |
|---|---|---|---|
| `<ws>/.rick/jobs/<job>/{plan,doing,learning,debug}` | 会话产物、tasks.json、session_id | **注册表里的绝对路径 `ws.Path`**（web 侧硬编码 `.rick`，**不认 `_dev`**） | `internal/web/sessions.go:355`（createSession）、`:908`（import CLI session）、`:1842,1868,2019`（建 job/doing/learning 目录）、`routes.go:347,436,469` |
| `<ws>/.rick/{loops,skills,domain,jobs,dream}` | 结构初始化 | 同上 | `internal/workspace/workspace.go:44,80-84`；`routes.go:347` |
| `<cwd>/.rick` 或 `<cwd>/.rick_dev` | **CLI 命令**的工作目录锚定（非 web） | 二进制名后缀 `_dev` | `internal/workspace/paths.go:34-55`（`getRickDirName()`）、`:188-213`（NextJobID） |
| 文件浏览器可读根 = `ws.Path` + `$HOME/.rick` | 聊天引用文件读取白名单 | `UserHomeDir()` | `internal/web/routes.go:742-749` |
| 子进程 cwd = `ws.Path` | pi worker / CLI 的工作目录（父目录是 `.rick`） | 注册表 | `internal/web/sessions.go:434, 987, 1053` + `internal/runtime/supervisor.go:211`（`cmd.Dir = spec.Dir`） |

### 1.4 web 启动即写（对「共享状态」的放大效应）

`internal/cmd/web.go:106-111` 在服务启动时串行执行 `ReconcileOnStart()` + `BackfillTitles()` + `BackfillJobParams()`，三者都会**写 `sessions.json`**（把 active/running 但无 worker 的会话标记 error）。→ 任何第二个实例启动都会覆盖式重写会话注册表（§5）。

### 1.5 可重定向性判定（工程结论）

| 维度 | 能否 env 重定向 | 手段 |
|---|---|---|
| web 状态目录 / sessions / archived / job-names / web.json | ❌ | 只能换 `HOME` |
| web.pid | ❌ | 只能换 `HOME` |
| 前端 overlay dist | ❌ | 只能换 `HOME` |
| config.json | ⚠️ 仅靠二进制改名 `_dev` | 或换 `HOME` |
| pi agent dir + runtime + sessions | ✅ | `RICK_PI_AGENT_DIR` |
| workspace `.rick`（web 侧） | ❌ | 由注册表绝对路径决定；改注册表指向 dev 工作树 |
| workspace `.rick`（CLI 侧） | ⚠️ 靠二进制改名 `_dev` | `paths.go:41-44` |

⇒ **`HOME` 是唯一能一次性覆盖「web 状态 + pid + overlay + config + pi agent」的开关**（pi agent 可再用 `RICK_PI_AGENT_DIR` 单独指回生产，实现「状态隔离但 pi 共享」）。置信度：高（代码 + 实测 `HOME=/tmp/l4-h` 起实例，全部状态落在该 HOME 下）。

---

## §2 pi 运行时依赖：共享还是独立 agent dir（Q2）

### 2.1 解析优先级与语义（已独立复核）

| 项 | 规则 | 位置 |
|---|---|---|
| `AgentDir()` | `RICK_PI_AGENT_DIR`（非空即用）> `$HOME/.rick/pi/agent`；`UserHomeDir()` 失败返回 `""` | `internal/runtime/agentdir.go:21-30` |
| `SettingsPath()` / `RuntimeDir()` / `RuntimeBin()` | `<AgentDir>/{settings.json, runtime, runtime/node_modules/.bin/pi}` | `agentdir.go:33-35, 42-44, 48-50` |
| `AgentEnv()` | `os.Environ()` **追加** `PI_CODING_AGENT_DIR=<AgentDir()>`（Go exec 重复键取末值 → 覆盖继承值）；**不清洗**其他 `PI_*` | `agentdir.go:68-70` |

**实测注入点仅 4 处（穷举，已 grep 复核）**：`internal/runtime/cli.go:109`（CallCLI：plan/easy/ctrl/human-loop）、`internal/runtime/runtime.go:148`（Executor：doing `--mode json`）、`internal/runtime/supervisor.go:213`（web 会话 worker；同处 `:211 cmd.Dir=spec.Dir`）、`internal/env/pi.go:25`（`pi install/list/--version`）。binary 解析同源：`cli.go:31-55`（`cfg.PiPath` > `RuntimeBin` > PATH `pi`），`supervisor.go:187-189` / `runtime.go:94-96` 复用。

**关键的隐藏耦合**：`AgentEnv()` 不清理继承来的 `PI_SESSION_FILE` / `PI_SESSION_ID` / `PI_PROVIDER` / `PI_MODEL` / `PI_SUBAGENT_PARENT_SESSION`。实测生产 web 进程 environ（`tr '\0' '\n' </proc/172761/environ`）确实含 `PI_SESSION_FILE=.../sessions/--workdir-sunquan20-AI_CODING-rick--/2026-08-29T00-43-37-686Z_c5c14e43-....jsonl`、`PI_SESSION_ID`、`PI_PROVIDER=deepseek`、`PI_MODEL=deepseek-flash`、`PI_SUBAGENT_PARENT_SESSION`。→ **dev 实例必须从干净 shell（或 `env -i`）启动**，否则这些变量会透传进每个 pi 子进程（"我是谁/用哪个模型"被生产值污染）。confidence：高（env 实测 + 代码）；「是否真被 pi 当作默认值采纳」中（未做行为实验）。

### 2.2 后果矩阵（共享 vs 独立；证据：`agentdir.go`、`internal/env/pi.go:65-90`）

| 维度 | 共享生产 AgentDir | 独立 AgentDir |
|---|---|---|
| 认证 `auth.json`（实测 95 B / 0600，键仅 `deepseek`） | 免配置可用；dev 误改即污染生产密钥（风险**高**） | 需复制/重登；误改不影响生产（低） |
| `settings.json`（实测 367 B：packages=pi-web-access,pi-subagents；theme=rick；defaultProvider/Model） | dev 改动作用于生产**新启** worker（中高） | 完全隔离（低） |
| session jsonl（`sessions/` 实测 1.4 G / 44 目录 / 1957 jsonl） | 目录名由子进程 **cwd** 派生（`--<cwd-path-slug>--`）；same AgentDir + same cwd ⇒ 同目录（文件名 `<ts>_<uuid>.jsonl` 不撞名）；rick 按 uuid 全树搜索（`web/sessions.go:1536-1553`）⇒ 跨实例互相可读（中） | 隔离（低） |
| `runtime/`（实测 145 M，rick 自托管 pi 副本） | **`npm install --prefix <RuntimeDir>` 原地覆写、无锁**（`env/pi.go:65-90`）→ 正在跑的 pi worker 面对被替换/半写的树（**高**） | 各自一份（低，代价 +≈145 M 磁盘） |
| 升级 pi | dev 升级 = 生产升级 | 各自可控 |
| 触发时机（重要） | 仅 `rick tools init-pi`（`cmd/tools_init_pi.go:40` → `env/env.go:56`）与 `rick tools update-pi`（`env/update.go:120`）；**`rick web` 启动路径不调用任何 `env.*`**（`cmd/web.go:77-148` 无 env 调用）⇒ 单纯起 dev 服务不会破坏生产 | 同 |
| `bin/` 工具（rg/fd） | 现成 | pi 首次自动下载到 `<agent>/bin`（需联网） |

### 2.3 推荐：**独立 AgentDir**（`RICK_PI_AGENT_DIR=$DEV_HOME/.rick/pi/agent`）

理由：唯一能阻断 2.2 表中「runtime 原地覆写（高）」与 auth/settings 双向污染；代价可接受。
代价与前置步骤（首次，需 node/npm 联网）：
```bash
D=$DEV_HOME/.rick/pi/agent; mkdir -p "$D"
cp ~/.rick/pi/agent/{auth.json,settings.json,models-store.json} "$D/"; chmod 600 "$D/auth.json"
RICK_PI_AGENT_DIR="$D" ./bin/rick tools init-pi   # 装独立 runtime + agents/skills/themes
```
代价清单：+≈145 M 磁盘、首次联网安装、auth.json 复制涉及密钥保管（生产 `auth.json` 含 API key，落盘勿入 git；`.gitignore` 已忽略 `.pi/`，注意 `.rick_dev/` 也在仓库外）。
若选共享（省钱省事）的可接受前提：**dev 永不执行 `rick tools init-pi` / `tools update-pi`**，且接受 dev 会话与生产会话在 `sessions/` 同目录混放。

---

## §3 源码工作树隔离（Q3）

### 3.1 入库事实（实测，决定隔离方式的边界）

| 项 | 事实 | 证据 |
|---|---|---|
| `web/dist` | **被跟踪 12 文件**（`index.html` + `assets/index-C3ifqTyg.js` + `index-BWKqsxDY.css` + workbox/sw/图标…），且 `//go:embed all:dist` 编译期进二进制 | `git ls-files web/dist \| wc -l`=12；`web/embed.go:31-41`；`web/vite.config.ts:53-56`（`outDir:"dist"`, `manifest:true`） |
| `web/src` | 被跟踪 66 文件（React 18 + TS） | `git ls-files web/src \| wc -l`=66 |
| `.rick/` | **被跟踪 1114 文件**（体积 ≈180–201 M；`.rick/jobs` 197 M / 747 文件；`.rick/jobs/job_36` 82 文件） | `git ls-files -z \| tr '\0' '\n' \| grep -c '^\.rick/'`=1114；`git ls-files -- .rick \| wc -l`=1114 |
| `.gitignore` | 忽略 `bin/`(:2)、`node_modules/`(:3, 另有 `web/.gitignore:1`)、`.pi/`(:13)、`web/dist.old/`、`coverage*`；**不忽略** `web/dist` 与 `.rick/` | `git check-ignore -v bin/rick web/node_modules .pi/agent/settings.json` 命中；`web/dist/index.html` 无命中=被跟踪 |
| 生产二进制 | `ls -l /proc/172761/exe → /workdir/sunquan20/AI_CODING/rick/bin/rick`，与磁盘文件 **同 inode 7904654** | `/proc/172761/exe` |
| npm/go 缓存 | `~/.npm` 1.4 G、`GOCACHE=/home/hadoop-recsys/.cache/go-build`(442 M)、`GOMODCACHE=/home/hadoop-recsys/go/pkg/mod`(602 M) —— **全部 HOME 派生** | `go env`、`du -sh` |

### 3.2 三种方式的实测对比（leaf-1 在 `/tmp` 实做，已清理；本机 `/tmp`=`/dev/md0p1` 与仓库所在 `overlay` **跨设备**）

| 方式 | 命令 | 耗时 | 磁盘 | node_modules | bin/ | `.rick` 副本 | 陷阱 |
|---|---|---|---|---|---|---|---|
| **worktree（推荐）** | `git worktree add --detach <dir> HEAD` | **2.13 s** | **202 M** | 无（需 `npm ci`） | 无（需 `go build`） | 有 180 M | 共享主仓 `.git` 的 refs/objects（见 3.3）；`main` 已被主仓占用 **必须 `--detach` 或 `-b`** |
| clone | `git clone --local --no-hardlinks <repo> <dir>` | 4.32 s | 713 M | 无 | 无 | 有 180 M | 裸 `--local` **跨设备直接失败**：`fatal: failed to create link … Invalid cross-device link`（硬链接 .git/objects 无法跨 FS） |
| cp -a | `cp -a <repo> <dir>` | 8.36 s | **958 M** | **带 185 M** | **带 16 M** | 有 200 M | 拷贝 `.git`(511 M) 与生产 `bin/rick`；两仓完全独立但也彻底失去「一次提交两处可见」 |

补充实测：worktree 内 `npm ci` = 3.55 s（**依赖热缓存 `~/.npm`**）→ 171 M；`go build -o bin/rick ./cmd/rick` = 0.983 s（**依赖共享 `GOCACHE`**）；worktree 里 `npm run build`（不改源码）= 4.13 s 且 `git status` 干净，**改 1 个前端可见字节** → `M web/dist/index.html`、`M .vite/manifest.json`、`M sw.js`、`D assets/index-C3ifqTyg.js`、`?? assets/index-D6T850Wu.js`。

### 3.3 关键陷阱（按风险排序）

1. **`.rick` 快照是最大的语义陷阱（major）**：worktree/clone 都会带一份 180 M 的 `.rick`（含 `job_36` 在内的 29 个 job、747 个跟踪文件）。dev 实例若把该工作树注册为工作区，会看到**冻结的 job 状态快照**（生产在跑时尤其误导），并可能对这份快照做 resume/重命名/归档。
   → 落地建议：dev 工作树里把 `.rick/jobs` 视为**只读基线**；或直接在 dev 树上 `git rm -r --cached .rick` 后加 `.rick/` 到本地 `.git/info/exclude`（不入库改动）以免两处提交打架。**这条必须由 human/后续层裁决**（涉及仓库纪律）。
2. **ref 共享（major）**：worktree 的 `commondir=../..`，dev 里的 `git commit`/`branch` 会出现在**主仓 ref 列表**；反之主仓 `git checkout main` 不影响 worktree 的 HEAD/index（各自独立），但**在 dev 树执行涉及 `main` 的操作（如 `git branch -D main`、`git push`、`git worktree prune` 误操作）仍会打到主仓**。→ 纪律：dev 树只用自己的 `-b dev/...` 分支，绝不触碰 `main`。
3. **`bin/rick` 覆盖（minor，但影响 L6）**：`bin/` 被忽略，prod `exe` 就是主仓 `bin/rick`。在 worktree 构建只写自己的 `bin/rick`（实测 prod 二进制 mtime 21:59 未变、`strings` 无 dev 标记）。**用 `/bin/sleep` 复现了「rename 覆盖运行中二进制」**：`/proc/$P/exe → …(deleted)` 而进程存活 ⇒ 主仓直接覆盖 `bin/rick` **不会立刻打断生产，但下一次重启会加载新二进制**（这正是 L6 提升机制可利用的性质）。
4. **`web/dist` 已提交 ⇒ 改前端必脏 git status（minor）**：见 3.2 末行。要么把 dist 变更一起提交，要么用前端 overlay 路径绕开（`~/.rick/web/dist` 生效**不需要重新编译 Go**，因为 overlay 优先于 embed —— `static.go:67-72`）。→ 这是 L5 自迭代回路的关键杠杆：**纯前端改动走 overlay 可零重启生效；后端改动才需 `go build` + 重启**。
5. **HOME 隔离的副作用：编译器/包管理器缓存变冷（minor，但会显著拖慢 dev 循环）**：`GOCACHE`/`GOMODCACHE`/`~/.npm` 全部 HOME 派生，dev 一旦用独立 HOME，实测的 0.983 s 构建与 3.55 s `npm ci` **不再成立**（冷缓存 + 联网）。
   → 缓解：dev 启动配置里显式 `export GOCACHE=$HOME_PROD/.cache/go-build GOMODCACHE=$HOME_PROD/go/pkg/mod`，npm 用 `--cache <prod>/.npm` 或 `npm_config_cache`。这三条 env 是「隔离状态但复用缓存」的关键。
6. **`/tmp` 位置**：本机 `/tmp` 跨设备且重启即失（leaf-1 实测跨设备导致 `clone --local` 失败）；长期 dev 树建议放 `/workdir/sunquan20/…`（与生产同设备，且便于人类查看）。

### 3.4 推荐命令（可复制）

```bash
# 1) 源码工作树（同设备、独立分支、不碰 main）
cd /workdir/sunquan20/AI_CODING/rick
git worktree add -b dev/web-self-iterate /workdir/sunquan20/rick-dev HEAD
# 2) 前端依赖（复用生产 npm 缓存，避免冷启动）
cd /workdir/sunquan20/rick-dev/web && npm ci --cache /home/hadoop-recsys/.npm
# 3) 二进制（复用生产 go 缓存）
cd /workdir/sunquan20/rick-dev
GOCACHE=/home/hadoop-recsys/.cache/go-build GOMODCACHE=/home/hadoop-recsys/go/pkg/mod \
  go build -o bin/rick ./cmd/rick
```
**已确认零残留**：leaf-1 的 `/tmp` 实验产物（`/tmp/l4-wt|l4-clone|l4-cp|l4-inode`）已删除，`git worktree list` 仅剩主仓，主仓 `git status` 与实验前基线一致，生产 8413 实测 200。

---

## §4 单例与端口冲突（Q4）

### 4.1 singleton 的实现与顺序（`internal/handler/web.go`）

`Web()` 的执行顺序（`:59-110`）：① `checkSingleton(pidPath)`（`:64-68`）→ ② 写 pid 文件（`:73`）+ `defer os.Remove(pidPath)`（`:76`）→ ③ `config.LoadConfig()`（`:81-84`）→ ④ token 解析（`:88-96`）→ ⑤ 信号 ctx（`:99-100`）→ ⑥ `serve(...)`（`:103-107`，EADDRINUSE 有专门提示 `:104-106`）。

| 组件 | 位置 | 语义 |
|---|---|---|
| pid 路径 | `handler/web.go:115-121`（`$HOME/.rick/web.pid`） | 「mirrors `internal/web.PidPath` without importing internal/web (import cycle: web → handler)」——**两处重复实现**，`internal/web/web.go:63-68` 是另一份 |
| 判活 | `handler/web.go:126-149` | 读文件 → `IsNotExist` 即放行（`:128-131`）→ 不可解析则删文件放行（`:136-140`）→ **`syscall.Kill(pid, 0)` 成功即 `ErrWebAlreadyRunning`**（`:142-145`）→ 死 pid 删文件放行（`:147-148`） |

**注意 `:69-77` 的条件**：只有 `pidPath != ""`（HOME 可解析）才写 pid 与注册 defer；因此「pid 文件只覆盖同一 HOME」。

### 4.2 实测（4 个探针，全部用临时 HOME / 临时端口，生产未受影响）

| 探针 | 命令 | 实测输出 | 结论 |
|---|---|---|---|
| A | `HOME=/tmp/l4-home ./bin/rick web --port 18999` | `~/.rick/web.pid` 内容 `177823`；日志 `listening on http://127.0.0.1:18999` | 同 HOME 首个实例正常写 pid |
| B | 同 HOME 再起 `--port 18998` | `Error: rick web is already running：PID 177823（先停旧实例，或删除 /tmp/l4-home/.rick/web.pid 后重试）` | **同 HOME + pid 存活 = 被拒** |
| C | A 存活期间 `rm .../web.pid`，再起 `--port 18996` | `listening on http://127.0.0.1:18996`，pid 文件被改写为 `178005`，A 与 C **同时存活** | **删掉 pid 文件即可绕过 singleton**（advisory only） |
| D | `HOME=/tmp/l4-home2 ./bin/rick web --port 18997 --token devtok` | 与 A 同时存活（18997 + 18999 并存，各自 pid 文件） | **不同 HOME = 天然共存**（dev 的正路） |

清理已验证：4 个探针进程已全部退出（`ps` 无匹配）、`/tmp/l4-home*` 已删除、生产 172761 全程存活（`GET /api/config` → 200）。

### 4.3 现网异常（重要，实测）

**生产 172761 正在运行，但 `/home/hadoop-recsys/.rick/web.pid` 不存在**（`ls` exit=2；`ls -1a ~/.rick/` 无该条目）。
- 事实：singleton 判活**只**依赖该文件 ⇒ **当前生产的 singleton 保护处于失效状态**。
- 机制已复现（探针 C）：pid 文件缺失时，同 HOME 的第二实例会被放行。
- 推断成因（confidence 中）：`checkSingleton` 的「读」与 `:73` 的「写」非原子（无 `O_EXCL`/`O_CREATE|O_EXCL`），两个同时启动的进程可都通过检查；后启动者绑定 8413 失败（EADDRINUSE）退出时，`:76` 的 `defer os.Remove` 会删掉 pid 文件 → 留下「有实例在跑但无 pid 文件」的状态（生产 `~/.rick/web.log` 显示 21:59 启动，之后无更多行；`strings bin/rick` 确认该二进制含 `web.pid` 逻辑，排除「旧二进制无此功能」）。**未做并发启动实测复现（会干扰生产），标 待验证。**
- 对 L4 的直接后果：**不能用「反正 singleton 会挡住」来说服 dev 复用生产 HOME**——现在它挡不住；且真挡住的场景是「dev 被彻底拒绝启动」（探针 B），也不是隔离。

### 4.4 端口占用实测与建议

- 默认值：`--port 6137`（`internal/cmd/web.go:65`）；`DefaultWebAddr = "127.0.0.1:6137"`（`internal/web/server.go:42`，仅当 `Listen=="" && Port==0` 时使用，`cmd/web.go:113-116`）。
- 生产：`~/.rick/start-web.sh` → `exec ./bin/rick web --listen 0.0.0.0 --port 8413`（ppid=1，cwd=生产仓库，stdout/stderr → `~/.rick/web.log`）。
- `ss -ltnp` 实测 0.0.0.0 上已占用：**8410、8411、8412、8413、8415、8420**（另 22/111/5050/5051/5087/5088/5266…/6942/7888 等无关端口）。**8414 空闲**（`start-web.sh` 注释称历史上曾用 8414 + 代理，用户确认只需 8413 后移除）。
- 建议：dev = **`--port 8414 --listen 127.0.0.1`**。理由：8414 紧邻且当前空闲（避免撞 8410–8413/8415/8420 这一簇）；`127.0.0.1` 刻意不暴露局域网（生产是 `0.0.0.0`，dev 若继承 `0.0.0.0` 会被同网段访问，且 dev 认证弱化时风险更高）。
- **flag/env 设计结论**：现有 `--port` / `--listen` **够用**（端口维度）；但**缺 `--state-dir` / `--home`** —— 没有任何 flag/env 能重定向 web 状态与 pid（§1.5）。最小可行（零代码）：`HOME=$DEV_HOME`；推荐（需改代码，留给 L5/L6）：新增 `RICK_STATE_DIR`（或 `--state-dir`）统一重定向 `WebStateDir/WebConfigPath/PidPath`，并把现有 `_dev` 后缀约定扩展到这三处（现在只覆盖 `config.json` 与 CLI 的 `.rick_dev`，§3 会再证）。

---

## §5 双实例对同一 workspace 的写冲突（Q5）

前提事实（已独立复核）：**全仓无任何跨进程锁** —— `grep -rniE "flock|LOCK_EX|O_EXCL|syscall.Flock" internal/ pkg/ --include=*.go`（排除测试）**空**。所有状态一致性只靠「进程内 mutex + tmp+rename 原子替换」，因此**两实例必然出现丢更新**。

### 5.1 机器级状态文件（写方式均 tmp+rename 原子，但无跨进程互斥）

| 文件 | 写入链 | 后果 | 严重度 |
|---|---|---|---|
| `~/.rick/web.json` | `registry.go:133/150/191` → `save()` `:218` → `writeFileAtomic` `:355-371` | 丢工作区/顺序（整份内存快照覆盖） | major |
| `~/.rick/web/sessions.json` | `registry.go:291/304` → `save()` `:341` → `:355` | **整份覆盖**：A 只改 1 个会话却顶掉 B 刚写的全部内容 → 会话状态抖动/丢失 | **blocker**（与 5.3 叠加） |
| `~/.rick/web/archived.json` | `archived.go:70/87` → `persist()` `:106-124` | 别名/归档丢失 | minor |
| `~/.rick/web/job-names.json` | `jobnames.go:139` → `saveLocked()` `:143-172`（CreateTemp+Rename） | job 显示名丢失 | minor |

### 5.2 workspace 级（`<ws>/.rick/`）

| 写点 | 位置 | 方式 / 冲突后果 | 严重度 |
|---|---|---|---|
| job 目录创建（plan/easy） | `web/sessions.go:1836+1842`、`:1863+1868`；`scanNextJobID` `:1987-2006` | **TOCTOU**：先扫 max `job_N` 再 +1 → 两实例同扫必撞同一 `job_N+1`，后写者覆盖前者 `tasks.json`/`requirement.md` | major |
| doing 初始 tasks.json | `internal/builder/orchestration.go:408-445`（WriteFile 445）← `handler/doing.go:115-117` | 非原子；幂等（已存在即跳过，不重置状态） | minor |
| easy 合成 tasks.json | `handler/easy.go:235` ← `easy.go:189`、`web/sessions.go:1877` | 非原子 | minor |
| hook 改任务状态 | `.rick/skills/mark_task_success_skill/mark_task_success.py:73-74`（`open(w)` 截断写） | 非原子；契约见 `orchestration.go:392`「tasks.json 只由 hook 写」 | minor |
| easy close 翻 success | `web/sessions.go:808`（`updateEasyTasksStatus`，def `:783`）← 调用点 `:768` | 非原子 | minor |
| git 提交（doing） | `handler/doing.go:311-332` `ensureGitRepo`：`git init` + **`git add -A`**(319) + commit(322)；`repoRoot=filepath.Dir(rickDir)`(311) | 两实例并发 doing 同一 workspace：`add -A` 把对方半成品纳入同一 commit | major |
| task 级 commit | 不在 Go 内（`grep -rn "git commit" internal/ .rick/skills/ scripts/` 仅命中 doing.go:329、orchestration.go:392 契约文本、scripts/version.sh:212） | 由 pi agent 自己 bash 提交；同 job 双实例会交错提交 | major |
| session_id / method.md | `web/sessions.go:456-457`、`:1979` | 覆盖写 | minor |

### 5.3 「互相误杀」——最高危的一条（已独立复核 `web/sessions.go:158-173`）

`ReconcileOnStart()`（调用点 `internal/cmd/web.go:107`，**每次 web 启动必跑**）对注册表里 `active`/`running` 的会话：
- `doing`/`dream` 类型 → **无条件** `updateStatus(error, "server restarted: background task lost")`（`:163-166`，不检查本进程是否有该 worker）；
- 交互型 → 本进程 `sup.Get(e.ID) == nil || IsDead()` 即标 error（`:168-171`）。

⇒ dev 实例一旦启动并读到**共享的** `sessions.json`，就会立刻把生产所有在跑会话标 error 并全量回写；生产随后用自己内存快照回写又顶掉 → 状态反复抖动，前端显示 error/Resume。**这正是 KR1「生产不中断」的直接反例。严重度：blocker。**

### 5.4 worker 归属与同 session 双跑

supervisor 的 worker 表是**纯内存 map**（注册 `supervisor.go:294`，`Get` `:316-321`），两实例互不可见；spawn 用 `--session-id`/`--session` 恢复既有 pi session（`supervisor.go:204-206`）。同一 web session 被两实例同时 resume → **两个 pi 进程写同一 `<AgentDir>/sessions/**/*.jsonl`**，且 `sessions.go:457` 的 `session_id` 互相覆盖。严重度：**blocker**（会话文件交错/损坏风险；jsonl 追加交错程度未做实测，标 待验证）。

### 5.5 缓解（必需项）

1. **独立 HOME** → 机器级 4 个注册表 + pid + overlay 天然隔离（§1.5）。
2. **dev 的 web.json 只注册 dev 工作树**，绝不注册 `/workdir/sunquan20/AI_CODING/rick`（否则 dev 会话直接编辑生产源码树并写生产 `.rick/jobs`，且被生产 Jobs 看板看到）。
3. **独立 `RICK_PI_AGENT_DIR`** → 阻断 5.4 的 jsonl 互写。
4. 仅隔离 HOME 而共享 workspace 是**不可接受**的：5.2/5.3 的 workspace 级与 `sessions.json` 冲突依旧成立。

---

## §6 前端 overlay 隔离（Q6）

### 6.1 overlay 解析链（已复核）

`StateDir = web.WebStateDir() = $HOME/.rick/web`（`internal/web/web.go:25-31`）→ 装配时 `overlayDist = joinPath(StateDir, "dist")`（`internal/web/server.go:66-68`，注入见 `:73`）与 watcher 侧（`server.go:154-158`）→ `StaticHandler(cfg.WebFS, overlayDist)`（`static.go:52`）。
生效判定 = `os.Lstat(overlayDir/index.html)` 存在即整树接管（`static.go:67-72`）；**all-or-nothing**：overlay 缺某 asset → 直接 404（`:74-87`、`:95-112`），非 asset 路径回落 overlay 的 `index.html`（SPA）；overlay 无效则回落编译期 embed `webembed.DistWeb()`（`cmd/web.go:121`、`web/embed.go:31-41`）。
**⇒ overlay 只由 HOME 决定：dev 用独立 HOME 即自然隔离；共享 HOME 则两实例服务同一目录，dev 一构建就改生产 UI（无锁、无实例标记）。**

### 6.2 scaffold 与现网 dist 的来源（已实测）

- `env.DeployWebScaffold` 抽取到 `$HOME/.rick/web/`（硬编码，`env/web.go:32-38, 51-76`），源 = `webembed.SrcFS`（**不含 dist**）；幂等靠 marker `~/.rick/web/.rick-managed`。
- `env.ResetWebCustomization`（`env/web.go:83-108`）删除 `WebStateDir()` 下**除 `sessions.json` 以外的一切**。
  - **发现（verified，严重度 major）**：这会连 `archived.json` 与 `job-names.json` 一起删掉，与 `env/web.go:80-82` 的注释（只保留 sessions.json 之外的 registry）及 `cmd/web.go:186-189` 的用户文档（「`~/.rick/web.json` 与 `~/.rick/web/sessions.json` 不受影响——reset 只清前端自定义」）**不一致**：归档表与 job 展示名会被清空。在共享 HOME 下，dev 侧一次 `rick web reset` 就会抹掉生产的归档与改名。
- 实测生产 `~/.rick/web/`：仅 `archived.json dist job-names.json sessions.json`（**无 src、无 marker**）⇒ `customize` 从未执行；现网 overlay 由 `~/.rick/deploy-web.sh`（`npm run build` → `rm -rf $HOME/.rick/web/dist` → `cp -r dist/.`）产出。
- 实测 overlay 与仓库产物**字节相同**：`sha256sum web/dist/index.html ~/.rick/web/dist/index.html` → 二者均为 `d03b90a1…`；assets 同名 `index-C3ifqTyg.js` / `index-BWKqsxDY.css`。

### 6.3 vite 开发链（已复核 `web/vite.config.ts`）

| 项 | 值 | 行 |
|---|---|---|
| dev proxy | `"/api": "http://127.0.0.1:6137"` —— **硬编码**，既不等于生产 8413 也不等于建议的 dev 8414 | `:57-60` |
| build.outDir / manifest | `dist` / `true`（**产物提交仓库**，注释 `:7`） | `:53-56` |
| PWA | `VitePWA`（autoUpdate；workbox 只缓存 `assets/**`；`devOptions.enabled=false`） | `:18-51` |
| base | 未设置（默认 `/`）⇒ 各端口根路径互不冲突 | — |
| 前端后端地址 | 同源：`web/src/api/client.ts:88` 用 `window.location.origin`、`web/src/api/sse.ts` 相对 `/api/events` ⇒ **dev 只需改 proxy target**，无需改前端代码 | — |
| scripts / 依赖 | `dev=vite`、`build=vite build`、`typecheck=tsc --noEmit`；vite ^6.3.5、React 18 + tailwind v4（非 Svelte）；无 `engines`；本机 node v24.16.0 / npm 11.13.0 满足 | `web/package.json` |

### 6.4 frontend_reload 的跨实例性

watcher 轮询**本实例**的 `cfg.StateDir/dist`（`watcher.go:41-44`、`50-99`：2 s 轮询、500 ms debounce、>5 s 强制 flush；publish 见 `:76`/`:95`）→ SSE（`sse.go:18, 409-413`）→ 前端 `web/src/api/sse.ts:422-424` `location.reload()`。
**⇒ 不同 HOME：互不触发（天然隔离）；共享 HOME：dev 构建 → 两实例各自广播 → 生产所有浏览器被强制 reload**（后端 job 不受影响，但用户视角是"生产 UI 自己变了"）。

### 6.5 前端维度的最小改动清单

- **无需改**：overlay 路径、embed 兜底、缓存策略（`static.go:35-43,146-154`）、SSE 同源、PWA。
- **需改（1 行，可不入库）**：`vite.config.ts:57-60` proxy target 参数化，例如 `"/api": process.env.RICK_DEV_API ?? "http://127.0.0.1:6137"`；dev 态 `RICK_DEV_API=http://127.0.0.1:8414 npm run dev`（vite 起 5173，HMR 免构建免 overlay）。
- **UI 迭代的两种回路**：① `npm run dev`（HMR，改 src 即时可见，需 proxy 指向 dev 后端）；② 构建产物路径 `npm run build` → 拷贝到 `<dev HOME>/.rick/web/dist`（触发本实例 frontend_reload → 浏览器自动刷新）。
- **注意**：`web/src`（66 文件）与 `web/dist`（12 文件）**均被 git 跟踪** ⇒ 在源码工作树里 `npm run build` 会让工作树变脏（dist 差异），需要明确「提交 dist 还是丢弃」的纪律（与 §3 交叉）。

---

## §7 最小隔离集结论（Q7）

### 7.1 dev 实例启动配置表（可直接落地）

| 项 | 取值 | 理由（证据） |
|---|---|---|
| 工作目录 | `/workdir/sunquan20/rick-dev`（`git worktree add -b dev/... HEAD`，与生产同设备） | §3.2/3.3：worktree 2.13 s / 202 M；同设备避免 `clone --local` 跨设备失败 |
| **HOME** | **`$DEV_HOME=/workdir/sunquan20/rick-dev-home`（全新，勿指向 `/home/hadoop-recsys`）** | §1.5：唯一能同时隔离 web.json / sessions.json / archived / job-names / web.pid / overlay dist / config.json 的开关；实测探针 D 证明两实例共存 |
| `RICK_PI_AGENT_DIR` | `$DEV_HOME/.rick/pi/agent`（独立；首次 `cp` 生产 `auth.json/settings.json/models-store.json` 后 `chmod 600 auth.json`，再跑 `./bin/rick tools init-pi`） | §2.2/2.3：阻断「`npm install --prefix` 原地覆写生产 145 M runtime」这一高风险路径；代价 +145 M + 首次联网 |
| `GOCACHE` | `/home/hadoop-recsys/.cache/go-build`（**显式共享生产缓存**） | §3.3.5：GOCACHE 是 HOME 派生的；不显式指定则 dev 构建从 0.98 s 退化到冷构建 |
| `GOMODCACHE` | `/home/hadoop-recsys/go/pkg/mod`（显式共享） | 同上（602 M，内容寻址只读复用） |
| npm 缓存 | `npm ci --cache /home/hadoop-recsys/.npm` | 同上（1.4 G 热缓存使 `npm ci` 仅 3.55 s） |
| 端口 / 监听 | `--port 8414 --listen 127.0.0.1` | §4.4：8414 实测空闲；生产占 8410–8413/8415/8420；loopback-only 刻意不暴露局域网 |
| 认证 | `--token <dev 随机值>`（勿复用生产 `web_token`） | 生产 token 存于 `~/.rick/config.json:web_token`；dev 独立 `config.json` 会自行生成（实测日志「已自动生成 web token」） |
| 二进制名 | **普通 `rick`**（不要用 `rick_dev`） | §3.3.6/§4：`_dev` 后缀只切 `config.json` 与 CLI `.rick_dev`，**不隔离 web 状态/pid**，反而与生产抢同一个 `web.pid`（§4.1：两处 pid 路径都硬编码 `.rick`） |
| 启动环境 | `env -i` 或干净 shell 中显式给出上述变量 | §2.1：生产 web 进程 environ 含 `PI_SESSION_ID/PI_PROVIDER/PI_MODEL` 等，`AgentEnv()` 不清洗，会透传给每个 pi 子进程 |
| dev 注册表（`$DEV_HOME/.rick/web.json`） | **只注册 dev 工作树**（`/workdir/sunquan20/rick-dev`），绝不注册 `/workdir/sunquan20/AI_CODING/rick` | §5.2/5.5：worker cwd=`ws.Path`、job 目录硬编码 `<ws.Path>/.rick`；注册生产路径 = dev 会话直接编辑生产源码树 |
| 前端 | ① HMR：`RICK_DEV_API=http://127.0.0.1:8414 npm run dev`（需先参数化 `vite.config.ts:57-60`）；② 产物：`npm run build` → 拷到 `$DEV_HOME/.rick/web/dist` | §6.3/6.5：前端同源（`window.location.origin`），只需 proxy target |
| 参考启动行 | `HOME=/workdir/sunquan20/rick-dev-home RICK_PI_AGENT_DIR=… GOCACHE=… GOMODCACHE=… /workdir/sunquan20/rick-dev/bin/rick web --port 8414 --listen 127.0.0.1 --token dev-$(date +%s)` | 组合 §1.5 + §4.4 + §2.3 |

**不需要隔离的部分（共享即纯收益或无害）**：Go 构建缓存与模块缓存（内容寻址，无状态串味，实测无「产物串味」）；npm 缓存；生产 `auth.json` 的一次性拷贝（只读种子）；`web/` 之外的生产源码（dev 树自带一份）；生产正在跑的 job/会话（只要 §5.5 两条纪律成立，两实例互不可见 worker：`supervisor.go:294` 纯内存表）。

### 7.2 该方案下仍然存在的泄漏点（residual risks）

1. **workspace 级写冲突未被 HOME 隔离覆盖**：只要 dev 与生产**共享任一工作区路径**，§5.2 的 `job_N` TOCTOU、`git add -A` 混提交、`tasks.json` 非原子写、watcher 双份事件依旧成立（机器级状态隔离了，workspace 级没有）。
2. **生产 singleton 当前失效**（`~/.rick/web.pid` 不存在，§4.3）：在修好之前，「同 HOME 起 dev」会被放行并直接读写生产 `sessions.json/web.json`；**这条既是 dev 风险也是现存生产缺陷**。
3. **`rick web reset` 会删 `archived.json` 与 `job-names.json`**（`env/web.go:83-108`，与 `cmd/web.go:186-189` 文档矛盾）：共享 HOME 时一次误操作即抹生产归档/改名；独立 HOME 下仅伤 dev。
4. **`_dev` 命名法仍是陷阱**：任何后续实现若沿用 `rick_dev` 二进制命名，会立刻与生产抢 `web.pid`（dev 先起则生产起不来；生产先起则 dev 直接被拒）。
5. **pi agent dir 若被裁决为共享**（为省 145 M）：只剩「dev 永不跑 `tools init-pi`/`tools update-pi`」这一条软约束在保护生产 runtime。
6. **无跨进程锁是结构性的**：即使 HOME 隔离，只要将来出现两个实例指向同一状态目录（例如人为 `--state-dir` 或备份脚本），丢更新会静默发生（`grep flock` 空）。
7. **`.rick` 快照陈旧**：dev 树自带 `.rick`（含 job_36/29 个 job），任何基于它的 resume/import 都会读到与生产分叉的历史。

### 7.3 事实 vs 推荐（分离声明）

**事实（有路径/行号/实测支撑）**：§1 全部路径归属与可重定向性；§2.1 优先级与 4 个注入点；§2.2 runtime 覆写机制；§3.1/3.2 入库与三种方式的耗时/磁盘/脏状态；§4.1–4.3 singleton 机制与 4 个探针结果、现网 pid 缺失；§5 全部写点行号与「无 flock」实测；§6 overlay/proxy/reset/前端同源。
**推荐（含代价，需 human 裁决）**：§2.3 独立 AgentDir（代价 +145 M + 联网）；§3.4 worktree + 显式缓存复用（代价：ref 共享需纪律）；§4.4 dev 端口 8414 + loopback；§7.1 整张配置表；§7.2 的 7 条泄漏点的处置优先级。
**置信度标注**：prod pid 缺失成因 = 中（漏洞高、成因未复现）；pi 是否采纳继承的 `PI_SESSION_ID` = 中；jsonl 交错损坏程度 = 未验证；其余为高（代码+实测）。

---

### 7.4 现场旁证（非本尽调所起，观察到的独立实现）

尽调末段在生产机上观察到**第二个 rick web 实例正在运行**（pid 186120，起于 22:21:47，`--listen 127.0.0.1 --port 18413 --token devtok`），经 `/proc/186120/environ` 判明它属于**并行的 L5 调研会话**（`HOME=/tmp/l5devhome`，日志 `/tmp/l5devhome/.rick/logs/web-h.log`），不是本 L4 派发的（L4 的探针用 18995–18999 且已全部退出）。实测其隔离效果：

| 观察项 | 实测值 | 结论 |
|---|---|---|
| 该实例 HOME | `/tmp/l5devhome`（独立） | 走的是「独立 HOME」路线 |
| 其 `PI_CODING_AGENT_DIR` | `/home/hadoop-recsys/.rick/pi/agent`（**生产 agent dir**） | 选的是共享 pi agent dir（§2.2 的中风险选项） |
| 其 cwd / exe | 生产仓库 cwd + 生产 `bin/rick` | 二进制与源码未隔离（§3.3.3 提示的 next-restart 风险 ） |
| 生产 `~/.rick/web/sessions.json` mtime | **仍为 22:13:17**（早于该实例启动 22:21:47） | **独立 HOME 确实让该实例没碰生产机器级状态**（对 §7.1 的现场验证） |
| 生产 `~/.rick/web.pid` | 依然**不存在** | 与 §4.3 一致：不同 HOME 各自一份 pid，互不影响；也再次说明生产 singleton 仍处于失效状态 |
| 生产 8413 | `api/config` = 200 | 两实例并存未影响生产可用性 |

**追加泄漏点（该项带入 §7.2）**：该实例共享了生产 `PI_CODING_AGENT_DIR` ⇒ ① 它启动的 pi 会话会写进生产的 `sessions/` 树（跨实例可读，§2.2）；② 若它执行 `rick tools init-pi`/`update-pi`，会原地覆写生产 145 M `runtime`（§2.2 高风险）。另 `/tmp` 作为 dev HOME 在机器重启后会被清空（`start-web.sh` 注释同款教训）。

---

## R7 上报项（无法在本次尽调内澄清，≤8 条）

1. **生产 `web.pid` 消失的确切历史成因**：需并发启动复现实验，但这会干扰生产实例（原则禁止）→ 缺「授权 + 可停机窗口」。
2. **pi agent dir 共享与否属成本/安全权衡**（+145 M 磁盘 vs runtime 覆写风险）：非事实问题，需 human 裁决。
3. **dev 工作树是否允许携带 `.rick`（1114 跟踪文件 / 180 M，含 job_36）**：涉及仓库纪律（是否 `git rm --cached .rick`），事实层无解。
4. **`rick web reset` 删 `archived.json`/`job-names.json` 是否判定为 bug 并修复**：代码与文档矛盾已确证，但「谁在哪个 task 里修」需 parent 裁决。
5. **同 session 双实例并发 resume 的 jsonl 实际损坏程度**：需构造双实例实验（当前生产在跑，不宜）；缺实验窗口。
6. **pi 是否把继承来的 `PI_SESSION_ID`/`PI_MODEL` 当默认值采纳**（leaf-3 标 medium）：需一次隔离行为实验。
7. **dev 的 token 策略**（复用生产 token 便于本机浏览器直连 vs 独立随机更安全）：产品偏好，未裁决。
8. **长期 dev 树与 dev HOME 的落盘位置/清理责任**（`/workdir/sunquan20/rick-dev{,-home}` 建议值）：需 human 确认磁盘与卫生约定。

## 叶子文件清单（本轮外包，均已落盘）

| 叶子 | 覆盖 | 路径 |
|---|---|---|
| leaf-1 | Q3 源码工作树隔离（三方式实测） | `.rick/jobs/job_36/doing/grilling/research-L4-leaf-1.md` |
| leaf-2 | Q5 双实例写冲突（写点穷举） | `.rick/jobs/job_36/doing/grilling/research-L4-leaf-2.md` |
| leaf-3 | Q2 pi 运行时依赖（共享/独立矩阵） | `.rick/jobs/job_36/doing/grilling/research-L4-leaf-3.md` |
| leaf-4 | Q6 前端 overlay 与 vite 链 | `.rick/jobs/job_36/doing/grilling/research-L4-leaf-4.md` |

> Q1（路径清单）、Q4（单例/端口）由 research 本体独立实测完成（§1、§4），未外包。
> 本轮共 5 次 subagent 派发（1 次 JS 语法失败重发 + 4 叶子），无叶子超时/空响应；全部 `/tmp` 实验残留已清理，主仓 `git status` 与基线一致，生产 8413 全程 200。
