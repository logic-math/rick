# Rick CLI 快速入门指南

> 从零开始使用 Rick CLI，5 分钟内完成第一个 AI 编程任务

## 📋 目录

- [安装 Rick CLI](#安装-rick-cli)
- [第一个 Job 示例](#第一个-job-示例)
- [常用命令速查](#常用命令速查)
- [故障排查](#故障排查)
- [下一步](#下一步)

---

## 安装 Rick CLI

### 前置要求

- **Go 1.21+** (仅源码安装需要)
- **Claude Code CLI** 已安装并配置
- **Git** 已安装

### 安装生产版本

```bash
# 克隆项目
git clone https://github.com/anthropics/rick.git
cd rick

# 从源码安装（推荐）
./scripts/install.sh

# 或从 GitHub 下载预编译二进制（Linux only）
./scripts/install.sh --binary
```

安装完成后，确保 `~/.local/bin` 在你的 PATH 中：

```bash
# 添加到 ~/.bashrc 或 ~/.zshrc
export PATH="$HOME/.local/bin:$PATH"

# 重新加载配置
source ~/.bashrc  # 或 source ~/.zshrc
```

### 验证安装

```bash
rick --version
rick --help
```

### 安装开发版本（可选）

如果你需要同时运行生产版本和开发版本（例如使用 Rick 重构 Rick）：

```bash
# 安装开发版本到 ~/.rick_dev
./scripts/install.sh --dev

# 验证
rick_dev --version
```

---

## 第一个 Job 示例

让我们通过一个完整的示例来体验 Rick 的工作流程。

### 场景：创建一个简单的 Go Web 服务

#### Step 1: 规划任务

```bash
# 在你的项目目录中运行
rick plan "创建一个简单的 Go HTTP 服务器，监听 8080 端口，提供 /health 健康检查接口"
```

**发生了什么？**
- Rick 自动创建 `.rick/` 工作空间
- 调用 Claude Code 生成任务分解
- 创建 `.rick/jobs/job_0/plan/` 目录，包含：
  - `tasks/` - 任务定义（task1.md, task2.md, ...）
  - `tasks.json` - 任务依赖关系（JSON 格式）

#### Step 2: 审核任务计划

```bash
# 查看生成的任务
ls .rick/jobs/job_0/plan/tasks/

# 查看任务依赖关系
cat .rick/jobs/job_0/plan/tasks.json
```

**示例 tasks.json**:
```json
[
  {
    "task_id": "task1",
    "task_name": "初始化 Go 模块",
    "dep": [],
    "state_info": {"status": "pending"}
  },
  {
    "task_id": "task2",
    "task_name": "实现 HTTP 服务器",
    "dep": ["task1"],
    "state_info": {"status": "pending"}
  },
  {
    "task_id": "task3",
    "task_name": "添加健康检查接口",
    "dep": ["task2"],
    "state_info": {"status": "pending"}
  },
  {
    "task_id": "task4",
    "task_name": "编写测试",
    "dep": ["task3"],
    "state_info": {"status": "pending"}
  }
]
```

**如果需要修改**：
- 编辑 `tasks/*.md` 调整任务描述
- 编辑 `tasks.json` 调整依赖关系
- 添加或删除任务

#### Step 3: 执行任务

```bash
rick doing job_0
```

**发生了什么？**
- Rick 自动在项目根目录初始化 Git（首次运行）
- 按拓扑排序顺序执行任务
- 对每个任务：
  1. 生成测试脚本
  2. 调用 Claude Code 执行任务
  3. 运行测试脚本验证
  4. 如果通过：git commit + 标记为 done
  5. 如果失败：记录到 `debug.md`，重试（最多 5 次）
- 创建 `.rick/jobs/job_0/doing/` 目录，包含：
  - `tasks.json` - 任务执行状态
  - `debug.md` - 失败记录（如有）
  - `logs/` - 执行日志

#### Step 4: 知识积累

```bash
rick learning job_0
```

**发生了什么？**
- 调用 Claude Code 分析任务执行过程
- 提取可复用的知识和经验
- 创建 `.rick/jobs/job_0/learning/` 目录，包含：
  - `summary.md` - 任务总结
  - `skills.md` - 提取的技能
  - `patterns.md` - 识别的模式
- 更新全局知识库 `.rick/skills/` 和 `.rick/patterns/`

#### Step 5: 查看结果

```bash
# 查看 Git 提交历史
git log --oneline

# 查看工作空间结构
tree .rick/jobs/job_0/

# 查看知识积累
cat .rick/jobs/job_0/learning/summary.md
```

---

## 常用命令速查

### 核心命令

| 命令 | 功能 | 示例 |
|------|------|------|
| `rick plan <description>` | 规划任务 | `rick plan "添加用户认证功能"` |
| `rick doing <job_id>` | 执行任务 | `rick doing job_0` |
| `rick learning <job_id>` | 知识积累 | `rick learning job_0` |
| `rick --version` | 查看版本 | `rick --version` |
| `rick --help` | 查看帮助 | `rick --help` |

### 安装脚本

| 脚本 | 功能 | 示例 |
|------|------|------|
| `./scripts/install.sh` | 安装生产版本 | `./scripts/install.sh` |
| `./scripts/install.sh --dev` | 安装开发版本 | `./scripts/install.sh --dev` |
| `./scripts/install.sh --binary` | 从二进制安装 | `./scripts/install.sh --binary` |
| `./scripts/uninstall.sh` | 卸载生产版本 | `./scripts/uninstall.sh` |
| `./scripts/uninstall.sh --dev` | 卸载开发版本 | `./scripts/uninstall.sh --dev` |
| `./scripts/update.sh` | 更新生产版本 | `./scripts/update.sh` |
| `./scripts/update.sh --dev` | 更新开发版本 | `./scripts/update.sh --dev` |

### 工作空间结构

```
.rick/
├── config.json                    # 全局配置
├── jobs/
│   └── job_0/
│       ├── plan/                  # 规划阶段
│       │   ├── tasks/             # 任务定义
│       │   └── tasks.json         # 任务依赖
│       ├── doing/                 # 执行阶段
│       │   ├── tasks.json         # 执行状态
│       │   ├── debug.md           # 失败记录
│       │   └── logs/              # 执行日志
│       └── learning/              # 学习阶段
│           ├── summary.md         # 任务总结
│           ├── skills.md          # 提取技能
│           └── patterns.md        # 识别模式
├── skills/                        # 全局技能库
├── patterns/                      # 全局模式库
└── wiki/                          # 知识库
```

---

## 故障排查

### 问题 1: 命令未找到

**错误**:
```
bash: rick: command not found
```

**解决方案**:
```bash
# 检查 ~/.local/bin 是否在 PATH 中
echo $PATH | grep -q "$HOME/.local/bin" && echo "OK" || echo "NOT IN PATH"

# 如果不在 PATH 中，添加到 shell 配置
echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.bashrc
source ~/.bashrc
```

### 问题 2: Go 版本不兼容

**错误**:
```
Go version 1.20 is not supported. Please use Go 1.21 or later.
```

**解决方案**:
```bash
# 安装 Go 1.21+
# macOS
brew install go

# Ubuntu/Debian
sudo apt update
sudo apt install golang-1.21

# 或从官网下载：https://go.dev/dl/
```

### 问题 3: Claude Code CLI 未安装

**错误**:
```
ERROR: Claude Code CLI not found
```

**解决方案**:
```bash
# 安装 Claude Code CLI
# 参考：https://docs.anthropic.com/claude-code/installation
```

### 问题 4: 任务执行失败

**现象**:
- 任务执行后状态为 "failed"
- `.rick/jobs/job_0/doing/debug.md` 包含错误记录

**解决方案**:
```bash
# 1. 查看 debug.md 了解失败原因
cat .rick/jobs/job_0/doing/debug.md

# 2. 根据错误原因修改 task.md
vim .rick/jobs/job_0/plan/tasks/task1.md

# 3. 重新执行
rick doing job_0
```

### 问题 5: Git 未初始化

**错误**:
```
ERROR: Not a git repository
```

**解决方案**:
```bash
# Rick 会自动初始化 Git，但如果出错，可以手动初始化
cd <项目根目录>
git init
git add .
git commit -m "Initial commit"

# 然后重新执行
rick doing job_0
```

### 问题 6: 权限错误

**错误**:
```
Permission denied: ~/.rick/bin/rick
```

**解决方案**:
```bash
# 修复权限
chmod +x ~/.rick/bin/rick

# 或重新安装
./scripts/uninstall.sh
./scripts/install.sh
```

---

## 下一步

恭喜！你已经完成了第一个 Rick CLI 任务。接下来你可以：

### 深入学习

1. **[核心概念](./core-concepts.md)** - 理解 Context Loop vs Agent Loop
2. **[架构设计](./architecture.md)** - 了解 Rick 的模块化架构
3. **[最佳实践](./best-practices.md)** - 学习任务分解和依赖设计

### 实践教程

1. **[Tutorial 1: 管理简单项目](./tutorials/tutorial-1-simple-project.md)** - 完整项目示例
2. **[Tutorial 2: 自我重构](./tutorials/tutorial-2-self-refactor.md)** - 使用 Rick 重构 Rick
3. **[Tutorial 3: 并行版本管理](./tutorials/tutorial-3-parallel-versions.md)** - rick + rick_dev
4. **[Tutorial 4: 自定义提示词](./tutorials/tutorial-4-custom-prompts.md)** - 定制提示词模板
5. **[Tutorial 5: CI/CD 集成](./tutorials/tutorial-5-cicd-integration.md)** - 集成到 CI/CD 流程

### 进阶主题

1. **[DAG Executor 模块](./modules/dag_executor.md)** - 理解任务调度机制
2. **[Prompt Manager 模块](./modules/prompt_manager.md)** - 掌握提示词管理
3. **[开发指南](../DEVELOPMENT_GUIDE.md)** - 贡献代码

---

## 获取帮助

- **文档**: [Rick Wiki](./index.md)
- **快速参考**: [QUICK_REFERENCE.md](../QUICK_REFERENCE.md)
- **常见问题**: [FAQ](./faq.md)
- **问题反馈**: [GitHub Issues](https://github.com/anthropics/rick/issues)

---

*最后更新: 2026-03-14*
