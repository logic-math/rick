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
