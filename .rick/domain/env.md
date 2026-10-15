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