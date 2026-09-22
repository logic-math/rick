# research-L4-leaf-3：pi 运行时依赖与「共享 vs 独立 agent dir」（Q2）

## 结论（3 条）

1. **解析优先级**：`RICK_PI_AGENT_DIR` > `$HOME/.rick/pi/agent`；`AgentEnv()` 只覆盖 `PI_CODING_AGENT_DIR`，其余 pi 变量原样继承 → dev 必须干净 env 启动。
2. **推荐独立 agent dir**：共享时 dev 的 init-pi/update-pi 会 `npm install --prefix` 原地覆写生产 runtime（145M），直接威胁运行中的 job。
3. **独立代价**：复制 auth.json（含密钥）+ settings.json，首次 `tools init-pi` 重装 runtime（+~145M，需 node/npm 联网）。

## 1. 语义与行号

| 函数 | 规则 | 位置 |
|---|---|---|
| `AgentDir` | `os.Getenv("RICK_PI_AGENT_DIR")` 非空即用；否则 `$HOME/.rick/pi/agent` | internal/runtime/agentdir.go:21-30 |
| `SettingsPath` | `AgentDir()/settings.json` | :33-35 |
| `RuntimeDir` | `AgentDir()/runtime`（npm prefix） | :42-44 |
| `RuntimeBin` | `RuntimeDir()/node_modules/.bin/pi` | :48-50 |
| `AgentEnv` | `os.Environ()` + `PI_CODING_AGENT_DIR=<AgentDir()>` | :68-70 |

pi 侧：`ENV_AGENT_DIR = ${APP_NAME}_CODING_AGENT_DIR` = `PI_CODING_AGENT_DIR`，`getAgentDir()` 优先读它，sessions/bin/themes/prompts 全部由其派生。
实测：`~/.rick/pi/agent/runtime/node_modules/@earendil-works/pi-coding-agent/dist/config.js:397,412-418,440-441,448-449`。
注意 `AgentEnv` 是**追加**（Go exec 重复键取最后一个 → 覆盖继承值），但**不清理**其它 pi 变量。

## 2. 注入点清单（哪些 pi 子进程读到该 env）

`cmd.Env = AgentEnv()` 共 4 处（穷举）：
- internal/runtime/cli.go:109 —— `CallCLI`：plan / easy / ctrl / human-loop 交互式 pi
- internal/runtime/runtime.go:148 —— `Executor`：doing `--mode json`
- internal/runtime/supervisor.go:213 —— web 会话 worker（同处 :211 `cmd.Dir=spec.Dir`，同 :216 Setpgid）
- internal/env/pi.go:25 —— `PiCommand`：`pi install`/`pi list`/`pi --version`

binary 解析同源：cli.go:31-55 `FindBinary`/`piPathOrDefault`（cfg.PiPath > RuntimeBin > PATH `pi`）；supervisor.go:187-189、runtime.go:94-96 复用同函数。web 侧读 agent dir：sessions.go:1542 `findSessionJSONL` 按 uuid 全树匹配（sessions.go:1536-1553）。

**待验证（confidence medium）**：`AgentEnv` 不清理 `PI_SESSION_FILE`/`PI_SESSION_ID`/`PI_SUBAGENT_PARENT_SESSION`/`PI_PROVIDER`/`PI_MODEL`。生产 web 进程 environ 实测（`tr '\0' '\n' </proc/172761/environ`）确实含这些值，故会透传给每个 pi 子进程；是否影响新建会话取决于 pi 是否让显式 `--session-id/--session` 覆盖它们。→ dev 实例务必在干净 shell（或 `env -i`）下启动。

## 3. 现网实况（只读，`~/.rick/pi/agent` = 1.6G）

sessions 1.4G（44 目录 / 1957 jsonl）｜runtime 145M｜npm 65M｜missions 8.6M｜bin 5.2M（仅 `rg`）｜skills 204K｜agents 36K（exporter/research/think.md）｜extensions 36K（catpaw-proxy.ts、rick-gates）｜themes 16K（rick.json…）｜auth.json 4K（键仅 `deepseek`）｜settings.json 4K（packages=pi-web-access,pi-subagents；theme=rick；defaultProvider=deepseek；defaultModel=deepseek-v4-flash）｜models-store.json 4K｜run-history.jsonl 208K。

会话子目录由 cwd 派生：`--<cwd 去首 /，[/\\:]→->--`（pi dist/core/session-manager.js:242-245），实测 `sessions/--workdir-sunquan20-AI_CODING-rick--`（142 jsonl）。
**判定**：AgentDir 相同且 cwd 相同 ⇒ 两实例写入**同一 sessions 子目录**（文件名 `<ts>_<uuid>.jsonl` 不撞名；rick 按 uuid 全树搜索 ⇒ 跨实例互相可读）。dev worktree 路径不同则子目录不同，但父树 auth/settings/runtime 仍共享。

## 4. 共享 vs 独立后果矩阵

| 维度 | 共享 | 独立 |
|---|---|---|
| auth.json（认证） | 免配置直接可用；dev 误改即污染生产密钥（**高**） | 需复制/重登；误改不影响生产（低） |
| settings.json（模型/theme/packages） | dev 改动立即作用于生产**新启** worker（**中高**） | 完全隔离（低） |
| session jsonl | 同 cwd 同目录、跨实例可读（中） | 隔离（低） |
| 子进程 env | 两实例指向同一树 | 各自指向（低） |
| runtime 副本 | `npm install --prefix` 原地覆写生产 node_modules（**高**） | 各自一份，+145M（低） |
| 升级 pi | dev 升级 = 生产升级 | 各自可控（低） |
| 磁盘 | 0 | +~145M |
| rg/fd 工具 | 现成 | pi 首次自动下载到 `<agent>/bin`（tools-manager.js:9,229-233；需联网） |

## 5. 关键风险核查（共享 AgentDir 的影响面）

`InstallManagedPI` = `npm install --prefix <RuntimeDir> --no-fund --no-audit <spec>`，无锁、无临时目录、直写 `AgentDir()/runtime/node_modules`（internal/env/pi.go:65-90）。
触发点仅 2 处且**非自动**：`rick tools init-pi`（cmd/tools_init_pi.go:40 → env.Ensure，env/env.go:56,68）与 `rick tools update-pi`（env/update.go:120）。
`rick web` 启动路径**不**调用 env.*（internal/cmd/web.go:77-148 无 env.* 调用）⇒ 生产 job 不会因 dev 单纯起服务被破坏；但 dev 一旦跑 init-pi/update-pi，正在运行的 pi worker 会面对被替换/半写的 145M 树 → **高风险**。

## 6. 推荐

**dev 用独立 AgentDir**：`RICK_PI_AGENT_DIR=$HOME/.rick_dev/pi/agent`。
准备步骤（首次）：
```bash
D=$HOME/.rick_dev/pi/agent; mkdir -p "$D"
cp ~/.rick/pi/agent/{auth.json,settings.json,models-store.json} "$D/"   # 复制后 chmod 600 auth.json
RICK_PI_AGENT_DIR="$D" ./bin/rick tools init-pi                          # 装独立 runtime + agents/skills/themes
```
理由：唯一能阻断 §5 覆写风险，且 auth/settings 双向不污染。
代价：+~145M 磁盘、首次联网装 pi、auth 复制需保管密钥（也可用只读挂载/每实例独立 key）。
