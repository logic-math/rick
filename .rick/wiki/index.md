# Rick CLI Wiki 知识库

> Context-First AI Coding Framework

## 📚 文档导航

### 核心文档
- [架构设计](./architecture.md) - 系统架构、模块关系、技术栈
- [核心概念](./core-concepts.md) - Context Loop、DAG 调度、提示词管理

### 模块详解
- [Infrastructure 模块](./modules/infrastructure.md) - 基础设施（CLI、工作空间、配置）
- [Parser 模块](./modules/parser.md) - 内容解析（Markdown、task.md、debug.md）
- [DAG Executor 模块](./modules/dag_executor.md) - DAG 执行器（拓扑排序、任务执行）
- [Prompt Manager 模块](./modules/prompt_manager.md) - 提示词管理（模板、构建、上下文）
- [CLI Commands 模块](./modules/cli_commands.md) - 命令处理（plan、doing、learning）
- [Git 模块](./modules/git.md) - Git 操作（自动初始化、提交）
- [CallCLI 模块](./modules/callcli.md) - Claude Code CLI 交互
- [Workspace 模块](./modules/workspace.md) - 工作空间管理

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

### 新手入门
1. 阅读 [核心概念](./core-concepts.md) 理解 Context Loop vs Agent Loop
2. 阅读 [架构设计](./architecture.md) 了解系统架构
3. 实践：运行 `rick plan` → `rick doing` → `rick learning` 完整流程

### 进阶开发
1. 深入学习 [DAG Executor 模块](./modules/dag_executor.md) 理解任务调度
2. 研究 [Prompt Manager 模块](./modules/prompt_manager.md) 掌握提示词管理
3. 实践：使用 `rick_dev` 开发新功能

### 高级定制
1. 学习模块化架构设计
2. 理解提示词模板机制
3. 实践：贡献代码或自定义模块

## 🔗 相关资源

- [项目 README](../../README.md)
- [OKR 文档](../../OKR.md)
- [SPEC 文档](../../SPEC.md)
- [内存库](~/.claude/projects/-Users-sunquan-ai-coding-CODING-rick/memory/MEMORY.md)

## 📝 文档贡献

本 Wiki 知识库遵循以下原则：
- 清晰简洁，避免冗余
- 包含代码示例和图表
- 及时更新，保持同步
- 交叉引用，便于导航

---

*最后更新: 2026-03-14*
