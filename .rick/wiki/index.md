# Rick CLI Wiki 知识库

> Context-First AI Coding Framework

## 📚 文档导航

### 🚀 快速开始
- **[快速入门指南](./getting-started.md)** - 5 分钟快速上手 Rick CLI
- **[最佳实践](./best-practices.md)** - 任务分解、依赖设计、测试方法
- **[任务设计最佳实践](./task-design-best-practices.md)** - 零重试的任务设计秘诀 ⭐

### 📖 实践教程
- **[教程系列](./tutorials/)** - 通过实践学习 Rick CLI
  - [Tutorial 1: 管理简单项目](./tutorials/tutorial-1-simple-project.md)
  - [Tutorial 2: 自我重构](./tutorials/tutorial-2-self-refactor.md)
  - [Tutorial 3: 并行版本管理](./tutorials/tutorial-3-parallel-versions.md)
  - [Tutorial 4: 自定义提示词](./tutorials/tutorial-4-custom-prompts.md)
  - [Tutorial 5: CI/CD 集成](./tutorials/tutorial-5-cicd-integration.md)

### 核心文档
- [架构设计](./architecture.md) - 系统架构、模块关系、技术栈
- [核心概念](./core-concepts.md) - Context Loop、DAG 调度、提示词管理

### 模块详解

#### 核心模块
- [CMD 模块](./modules/cmd.md) - 命令行接口（Cobra、参数解析、命令执行）
- [Workspace 模块](./modules/workspace.md) - 工作空间管理（目录结构、Job 管理）
- [Prompt Manager 模块](./modules/prompt_manager.md) - 提示词管理（模板、构建、上下文）
- [Executor 模块](./modules/dag_executor.md) - DAG 执行器（拓扑排序、任务执行）
- [Parser 模块](./modules/parser.md) - 内容解析（Markdown、task.md、debug.md）
- [Git 模块](./modules/git.md) - Git 操作（自动初始化、提交）
- [Config 模块](./modules/config.md) - 配置管理（加载、验证、持久化）
- [Logging 模块](./modules/logging.md) - 日志系统（分级日志、文本格式）

#### 辅助模块
- [CLI Commands 模块](./modules/cli_commands.md) - 命令处理（plan、doing、learning）
- [CallCLI 模块](./modules/callcli.md) - Claude Code CLI 交互
- [Infrastructure 模块](./modules/infrastructure.md) - 基础设施（CLI、工作空间、配置）

## 🎯 快速开始

### 核心公式
```
AICoding = Humans + Agents
其中：Agents = Models + Harness
```

### 核心命令
```bash
rick plan "任务描述"        # 规划任务
rick doing job_n            # 执行任务
rick learning job_n         # 知识积累
```

### 设计原则
1. **简化设计** - 最小化日志、配置、依赖
2. **人类控制** - 由人类完全控制 Context Loop
3. **自动化最小化** - 避免过度自动化，保持决策权

## 📖 学习路径

### 新手入门（~1 小时）
1. 阅读 **[快速入门指南](./getting-started.md)** - 了解安装和基本使用
2. 完成 **[Tutorial 1: 管理简单项目](./tutorials/tutorial-1-simple-project.md)** - 动手实践
3. 阅读 [核心概念](./core-concepts.md) - 理解 Context Loop vs Agent Loop

### 进阶开发（~2 小时）
1. 完成 **[Tutorial 2: 自我重构](./tutorials/tutorial-2-self-refactor.md)** - 掌握版本管理
2. 完成 **[Tutorial 3: 并行版本管理](./tutorials/tutorial-3-parallel-versions.md)** - 掌握双版本工作流
3. 阅读 **[最佳实践](./best-practices.md)** - 学习任务分解和依赖设计
4. 深入学习 [DAG Executor 模块](./modules/dag_executor.md) - 理解任务调度

### 高级定制（~2 小时）
1. 完成 **[Tutorial 4: 自定义提示词](./tutorials/tutorial-4-custom-prompts.md)** - 定制提示词模板
2. 完成 **[Tutorial 5: CI/CD 集成](./tutorials/tutorial-5-cicd-integration.md)** - 集成到自动化流程
3. 研究 [Prompt Manager 模块](./modules/prompt_manager.md) - 掌握提示词管理
4. 实践：贡献代码或自定义模块

## 🔗 相关资源

- [项目 README](../../README.md)
- [OKR 文档](../OKR.md)
- [SPEC 文档](../SPEC.md)
- [内存库](~/.claude/projects/-Users-sunquan-ai-coding-CODING-rick/memory/MEMORY.md)

## 📝 文档贡献

本 Wiki 知识库遵循以下原则：
- 清晰简洁，避免冗余
- 包含代码示例和图表
- 及时更新，保持同步
- 交叉引用，便于导航

---

*最后更新: 2026-03-14*
