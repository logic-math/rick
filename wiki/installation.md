# Rick CLI 安装与部署指南

## 系统要求

| 依赖 | 要求 | 说明 |
|------|------|------|
| Go | >= 1.21 | 编译 rick 源码 |
| Node.js + npm | >= 22.19.0 | pi 运行时（rick **不**替用户装 node，缺失时 `init-pi` 会提示自行安装） |
| Git | 任意 | 自动 commit |

## 安装方法

### 源码构建

```bash
./scripts/build.sh      # 构建 → ./bin/rick
./scripts/install.sh    # 安装到 ~/.rick/bin/rick（可选）
```

测试时直接用 `./bin/rick`，无需安装。

### 初始化 pi（必须）

```bash
rick tools init-pi
```

`init-pi` 幂等，检查后跳过已就绪项，缺什么补什么：

1. 前置检查：若 pi 未安装，检查 node/npm 是否在 PATH（缺失则终止）
2. 检查 rick 自闭环 pi 运行时（`~/.rick/pi/agent/runtime`），缺失则 `npm install --prefix` 安装独立副本
3. 注册 `pi-subagents` 扩展（`subagent` 工具）
4. 注册 `pi-web-access` 扩展（`web_search`/`fetch` 工具）
5. 剔除 Tokyo Night 包（`@wishx127/pi-tokyo-night`）
6. 最终验证：`pi list` 确认扩展注册 + 主题字段
7. 落盘 rick 自有定制（rick-gates hook / rick skills / think-research-exporter agent）

## 配置文件

`~/.rick/config.json`：

```json
{
  "max_retries": 5,
  "runtime": "pi",
  "pi_path": "",
  "pi_extra_args": ["--provider", "deepseek", "--model", "deepseek-v4-pro", "--api-key", "sk-..."],
  "git": {
    "user_name": "Your Name",
    "user_email": "your.email@example.com"
  }
}
```

> pi **不读**环境变量 provider/model/api-key，必须通过 `pi_extra_args` 或命令行配置。

## 版本管理

Rick 支持生产版本（`rick` → `~/.rick`）与开发版本（`rick_dev` → `~/.rick_dev`）并行，用于自我重构：

```bash
./scripts/install.sh --source --dev   # 安装开发版
rick_dev plan "重构 Rick 架构"        # 用开发版自我重构
```

## 常见问题

- **pi 未就绪**：先跑 `rick tools init-pi`；node/npm 缺失则先装 node。
- **门禁失败**：`helper.py` 报 zombie / 缺 commit_hash，检查 doing/tasks.json。
- **pi 扩展"假成功"**：用 `pi --mode json -p '...' | grep toolName` 验证真实生效，不只看 `pi list`。
