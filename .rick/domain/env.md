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
- 版本 3.1.5；默认模型 deepseek-v4-pro（~/.rick/config.json pi_extra_args）
