# 本机环境事实

## GitHub 推送凭据

- **GH_TOKEN 定义在 `~/.zshrc`**（`export GH_TOKEN=github_pat_...`）
- 当前 bash 环境（如 rick/pi 会话）**不会自动加载 zshrc** → push 报 "could not read Username"
- 解决：`export GH_TOKEN=$(grep -oP 'export GH_TOKEN=\K.*' ~/.zshrc | head -1)` 再 push
- credential helper 已配置（读 GH_TOKEN），无 ~/.git-credentials、无 gh CLI
- 仓库 `logic-math/rick` 公开可读（api.github.com 200），push 需写 token
- SSH 不可用（内网 DNS 无法解析 github.com），走 HTTPS + HTTP 代理

## rick 安装位置

- `~/.local/bin/rick`（PATH 生效，符号链接曾被替换为真实文件）
- `~/go/bin/rick`（go install 产物）
- 项目内 `bin/rick`（仓库跟踪二进制，gitignore 但 -f 提交）

## pi 托管目录结构（job_34 确认）

- `~/.rick/pi/agent/` — pi 配置目录（PI_CODING_AGENT_DIR 注入点）：settings.json（theme/hideThinkingBlock/packages）、themes/、npm/node_modules（扩展）
- `~/.rick/pi/agent/runtime/` — rick 自闭环 pi 运行时（npm --prefix 安装的独立包副本），二进制 `node_modules/.bin/pi`
- 全局 pi（用户独立会话用）：`~/.local/lib/node_modules/@earendil-works/pi-coding-agent`（v0.84.1，PATH `pi` → `~/.local/bin/pi`）

## rick 主题（job_34 最终）

- 当前主题：**VSCode Dark+** 配色（bg #1e1e1e，keyword #c586c0，string #ce9178，comment #6a9955）
- 源头：`internal/cmd/themes/rick.json`（embedded）；运行时激活副本：`~/.rick/pi/agent/themes/rick.json`
- 版本 3.1.5；默认模型 **catpaw-proxy / glm-5.3**（thinking **high**）——落点 `~/.rick/config.json` 的 `pi_extra_args`（prod 与 dev HOME 均已设置）

## rick 自定义 agent 落盘（job_35 确认）

- `~/.rick/pi/agent/agents/` — rick 自定义 agent（think/research/exporter）落盘目录，由 env 职责 3 `deployRickAgents` 幂等写入（YAML frontmatter + wiki 正文）
- 自定义 agent 清单注册在 `internal/env/agents.go` 的 `rickAgents`，就绪 check 在 `CheckRickAgents`

## 本机 Mac 布局（job_46 迁移后，2026-10-15）

- **本机已从 Linux 服务器（10.146.199.52，/workdir/sunquan20）整体迁移到 Mac**
- 工作区：`~/workdir`（= 源机 /workdir/sunquan20 全树；HUMANS/SLM/BERT_KEETA/.../AI_CODING/rick 11 个注册工作区）
- rick 状态：`~/.rick`（标准位；config.json / web.json / pi/agent 含 3,778 个会话文件）
- rick 二进制：`~/.local/bin/rick`（darwin/arm64，main 构建 5.0.15——**不含 dump/import 功能**；该功能在 `/Users/logic/workdir/rick-dev` 树的 6+2 个提交里，gitfile 断链待修，修好后可构建/release）
- 迁移方式：`rick tools dump`（8.6GB 归档）→ import `--target ~/workdir`（源机 /workdir/sunquan20 marker id `6c925625032d269f4ae0019c9cc224e7` 跨机不变）
- web 访问：`rick web --listen 0.0.0.0 --port 8413`（端口约定沿源机）；token 在 `~/.rick/config.json` 的 `web_token`

## 目标机迁移依赖（job_46 E2E 实测）

- **node ≥22.19**：目标机无 node 时从 nodejs.org 经代理下 tarball 解压到自管目录即可（无需 root/系统安装）：`curl -sSL -x http://10.229.18.27:8412 -o node.tar.xz https://nodejs.org/dist/v22.20.0/node-v22.20.0-linux-x64.tar.xz && tar xf && export PATH=<dir>/bin:$PATH`
- **模型 API 经代理可达**：内网机器实测 `10.229.18.27:8412` 代理出网（github 200 / api.deepseek.com 401=TLS 通）；pi worker 需 `export http_proxy/https_proxy` + `no_proxy=localhost,127.0.0.1`
- **`--state-dir` 必须与 `RICK_PI_AGENT_DIR` 成对**：`rick web --state-dir X` 不带 `RICK_PI_AGENT_DIR` 会被启动守卫拒绝（防 pi 沙盒指向生产、init-pi/update-pi 覆写生产 runtime）——迁移部署的标准布局（state 落默认 ~/.rick）无此问题
- **web 端口约定 8413**（沿生产惯例；目标机若占用需换 --port）

## 网络与代理（job_44 实测）

- **直连 DNS 失败**：`github.com` 解析报 EAI_AGAIN；经代理可达：
  - 验证命令: `curl -x http://10.229.18.27:8412 -sI https://github.com`（→ HTTP 200）
  - 来源: job_44  状态: ✅ 已确认
- **git push 凭据**：仓库 credential helper 依赖 `GH_TOKEN` 环境变量（fine-grained PAT，一次性使用不落盘；提取方法见上方「GitHub 推送凭据」节）
  - 来源: job_44  状态: ✅ 已确认

## 生产树属主地图（agent=hadoop-recsys，job_44 实测）

- **可写**：代码路径 / `.git` / `bin`（不含 releases）/ `~/.rick`
- **不可写**：`.rick/jobs/job_44|45|46`（属主 sankuai，ACL mask r-x）
- **bin/releases 曾不可写**（属主 sankuai，ACL mask r-x）——现已授权：`sudo -n -u sankuai setfacl ... && sudo -n -u sankuai chmod 775`（775 生效后 agent 可写）
- **sudo 事实**：`sudo -n -u sankuai` 免密可用；sudo 到 root 需密码（不可用）
  - 验证命令: `getfacl <dir> | grep mask`（#effective:r-x = 被锁）；`sudo -n -u sankuai true && echo SUDO_OK`
  - 来源: job_44  状态: ✅ 已确认

## .git/info/exclude 为 dev/main 两个 worktree 共享（job_44）

- 写 `.git/info/exclude` 同时影响 dev/main 两侧的 ignore 行为（同一份文件）
  - 来源: job_44  状态: ✅ 已确认

## .rick/jobs 载荷策略（job_44）

- 生产树 `.rick/jobs/**` 属主 sankuai 且 agent 无写权 → 发布载荷必须排除 base 中不存在的新 job 路径（job_44/45）；job 记录以 dev 树为准
- `.git/info/exclude` 本地忽略相应新 job 路径（三条），以保证 merge 干净性检查通过
  - 来源: job_44  状态: ✅ 已确认

## 模型网关路由（job_49 实测，Mac 主机）

- **mcli.sankuai.com 是内网直达域名**：pi catpaw-proxy provider 的 baseUrl（key 在 `~/.config/mcopilot-cli/.config.yaml` 的 AUTHORIZATION 行）。本机 proxy `10.229.18.27:8412` 能转发外网（github 200）但**不转发它**（HTTP 000）——带 proxy env 访问模型网关必 `Connection error`，no_proxy 必须含 mcli.sankuai.com（已写入 ~/.rick/start-web.sh）。与上方 job_46「模型 API 经代理可达」为目标机 Linux 实测——机器不同结论相反，部署环境迁移后网络假设须实测校准（同域名在本机与在 Linux 目标机路由性质可反转）
  - 验证命令: `no_proxy=localhost,127.0.0.1,mcli.sankuai.com pi --provider catpaw-proxy --model glm-5.3 --print "通"`（带 proxy env 时对照实验复现 Connection error）
  - 来源: job_49  状态: ✅ 已确认

## darwin 进程探测字段语义（job_49 实测）

- **macOS shebang 进程的 `ps -ww -o command=` 输出**：argv[0] 是解释器路径（如 `/bin/sh`、`/Library/.../Python`），脚本路径在 argv[1]——名义判定要扫全 argv（只看 argv[0] 会漏）；`ps -o comm=` 同样是解释器名（如 `/bin/sh`），不是脚本名
  - 验证命令: `printf '#!/bin/sh\nsleep 20\n' > /tmp/sb.sh && chmod +x /tmp/sb.sh && /tmp/sb.sh & sleep 1; ps -ww -o command= -p $!; ps -o comm= -p $!`
  - 来源: job_49  状态: ✅ 已确认

- **`lsof -F` 结构化输出字段语义**：p=pid、c=comm（进程名）、n=name（文件路径）——批量 cwd/txt 采集用 `-F pn`；把 c 当路径会让 Snapshot.Cwd 恒为进程名（见 bugs.md job_49 procfs Snapshot 条目）
  - 验证命令: `lsof -w -F pc -d cwd | head`（c 行=进程名）vs `lsof -w -F pn -d cwd | head`（n 行=路径）
  - 来源: job_49  状态: ✅ 已确认
## darwin procfs Environ 不可用——进程所有权断言的 env 兜底在 macOS 恒失效（job_47 实证）

- **`darwinInfo.Environ(pid)` 返回 `ErrUnsupported`**（`internal/env/procfs/procfs_darwin.go:113`：darwin 无 `/proc/<pid>/environ` 等价物，KERN_PROCARGS2 的 env 部分不可靠）→ `ownedByProd` 三重断言（cwd/exe/HOME env）在 macOS 上 **HOME env 断言永远不生效**（`err != nil` 即跳过该断言）——进程所有权只剩 cwd/exe 两条腿。job_47 release 事故链一环：生产实例从 `~/bin/rick`（裸构建，cwd=家目录）启动时 cwd/exe 都不在生产树 → 三断言全不命中 → stopProd 跳过不杀 → 新实例被 web.lock 挡住（job_49 P0-3 以「持生产 web.lock/占生产端口即认领」根治；此处记录事实本身）。
  - 验证命令: `grep -n "Environ" internal/env/procfs/procfs_darwin.go`（返回 ErrUnsupported 的注释即事实源）
  - 来源: job_47 / release 跨机事故分析 + wiki/handover-2026-10-30-web-start-macos.md  状态: ✅ 已确认（job_49 P0-3 另行根治认领逻辑）

## 2026-11-01 网络出口变更（human 裁决：搜索/出网一律直连）

- **事实**：`10.229.18.27:8412` = hk02-llm-spider-proxy37.mt 的 Squid 3.1.23；对本机源 IP（11.3.4.158 / utun4 VPN）一律 `403 ERR_ACCESS_DENIED`（ACL 拒绝）——npm、web_search provider fetch、fetch_content 全挂；同时本机**直连外网完全正常**（google/baidu/bing 200、api.exa.ai API 端点可达）。旧结论「机器无直连外网、模型 API 经代理可达」已过时（环境漂移：代理 ACL 变了 + 本机直连恢复）。
- **裁决与落地**：`~/.rick/start-web.sh` 已删 http_proxy/https_proxy export（保留 mcli 的 no_proxy）；dev web 已带干净 env 重启。生产 web 重启后生效。
- **回滚条件**：若未来需要恢复代理（如直连被掐），先 `curl -x http://10.229.18.27:8412 -sI https://github.com` 验证 ACL 放行再 export。
- **单点**：api.search.brave.com 直连超时（000）——Brave provider 受影响，pi auto 选路走 Exa/OpenAI 等可用通道，不阻塞 web_search。
- **排查方法**（环境漂移通用）：① `env | grep -i proxy` 找出口配置 → ② `curl -x <proxy> -sI https://github.com` vs `curl --noproxy '*' -sI https://github.com` 对照（代理 vs 直连）→ ③ 代理 403 时看 Squid 错误页（`curl -s http://x/ 2>/dev/null | grep -oE "ERR_[A-Z_]+"`）拿代理身份与拒绝原因 → ④ 顺藤找 env 来源（grep shell profile / 启动脚本 / launchctl getenv / ps eww <pid>）。

## macOS launchd 守护环境（job_49 追加沉淀，2026-11-02）

- **launchd 拉起的进程不继承 shell PATH**（homebrew `/opt/homebrew/bin` 是 zshrc 注入）——plist 必须显式注入 PATH，否则 pi worker 报 `env: node: No such file or directory`（exit 127，pi shebang 依赖 PATH 里的 node）
- **macOS 睡眠/唤醒周期会向后台 daemon 发 SIGTERM**——`--daemon`（setsid+nohup）防 SIGHUP 防不了它；web 两次同签名死亡实证（worker exit 143 → `graceful shutdown: context deadline exceeded`）。根治 = launchd KeepAlive 守护
- **launchd 正确持有方式 = 前台直跑**（不带 --daemon：setsid 脱离让 KeepAlive 追踪失效；不用幂等脚本入口：秒退 exit 0 会让 launchd 误判成功、旧实例死透后无人再拉）
- 完整操作模板与排障速查：`mac_launchd_web_skill/skill.md`（当前生产 plist：`~/Library/LaunchAgents/com.rick.web.plist`，KeepAlive+RunAtLoad+ThrottleInterval=5；**job_51 修订：env 只注入 no_proxy（含 mcli.sankuai.com），不注入 http_proxy/https_proxy**——旧「proxy 三件套」约定已反转，硬编码 proxy 会成 KeepAlive 复活的毒源）

## macOS 排查工具陷阱（job_51 实证，2026-11-02）

- **`ps eww <pid>` 有输出截断假象**：pi 进程显示无 proxy、其子 bash 实际带 proxy（pi 85393 显示 0 个 proxy 变量、子 bash `env` 里 http/https 都在）。排查 env 继承链时不可只信 ps——让可疑进程自己执行 `env | grep -i proxy`（进程内自证）交叉验证。
- **macOS 无 `setsid` 命令**（Linux 常用）：需要脱离进程组时用 Go 侧 `--daemon`（rick web --daemon 内部 syscall setsid，re-exec 子进程完全脱离）或 `nohup ... & disown`（父死后被 launchd 收养继续跑）。
- **「杀不死的进程」第一反应查 launchd**：`launchctl list | grep <label>`——KeepAlive=true 会 ≤ThrottleInterval 秒自动复活，**复活实例带的是 plist EnvironmentVariables 硬编码值**（改 start-web.sh / release prodEnv / 手动干净重启都会被下次复活盖掉）。修法 = 改 plist + `launchctl bootout gui/$(id -u)/<label> && launchctl bootstrap gui/$(id -u) <plist>` 成对重载。

## rick 二进制链回退（job_51，superopt 跨 workspace 解析）

- **`resolveRepoRoot` 两级查找（internal/super/supervisor.go）**：①进程 cwd 逐级上溯找 `super/superopt/__main__.py`（dev 树内/测试用）；②**fallback：rick 二进制自身位置**（`os.Executable` + `EvalSymlinks` 穿 `bin/rick → releases/current/rick` symlink 链到真实 repo）。外部 workspace（如 MT_INFER）的 web super session 里 pi worker cwd=ws.Path，①必失败——superopt 调度器全靠②救（曾实测 `rick super start` 报 `No module named superopt`）。
- **super session 资产是逐 workspace 的**：web 侧 `rickDir = <workspace>/.rick`，创建 super session 前置检查 `<ws>/.rick/loops/super-loop.md` + `<ws>/.rick/skills/gen_env_skill/skill.md`；运行还要 super 七件套（experiment_analysis/sense/experiment_design/debug_protocol/cell_management/cost_analysis）+ super_event_autopsy。外部 workspace 上 super 前必须整套部署（job_51 已部署 MT_INFER 9 件）。
