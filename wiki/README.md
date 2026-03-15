# Rick CLI Wiki

欢迎来到 Rick CLI 的知识库！这是一个 Context-First AI Coding Framework 的完整文档中心。

## 目录结构

本 Wiki 采用模块化组织，分为以下几个主要部分：

```
wiki/
├── README.md                    # 本文件：Wiki 索引和导航
├── getting-started.md           # 快速入门指南
├── core-concepts.md             # 核心概念和设计理念
├── architecture.md              # 系统架构设计
├── best-practices.md            # 最佳实践和使用建议
├── modules/                     # 模块详细文档
│   ├── README.md                # 模块概览
│   ├── infrastructure.md        # 基础设施模块
│   ├── parser.md                # 内容解析模块
│   ├── dag_executor.md          # DAG 执行模块
│   ├── prompt_manager.md        # 提示词管理模块
│   ├── cli_commands.md          # 命令处理模块
│   ├── workspace.md             # 工作空间管理
│   ├── config.md                # 配置管理
│   ├── logging.md               # 日志系统
│   ├── git.md                   # Git 集成
│   └── callcli.md               # Claude Code CLI 交互
└── tutorials/                   # 教程和实战案例
    ├── README.md                # 教程索引
    ├── tutorial-1-simple-project.md      # 教程1：简单项目实战
    ├── tutorial-2-self-refactor.md       # 教程2：使用 Rick 重构 Rick
    ├── tutorial-3-parallel-versions.md   # 教程3：并行开发版本
    ├── tutorial-4-custom-prompts.md      # 教程4：自定义提示词模板
    └── tutorial-5-cicd-integration.md    # 教程5：CI/CD 集成
```

## 文档导航

### 🚀 新手入门

如果你是第一次使用 Rick CLI，建议按以下顺序阅读：

1. **[快速入门指南](getting-started.md)** - 安装、配置和第一个项目
2. **[核心概念](core-concepts.md)** - 理解 Rick 的设计理念和核心概念
3. **[教程1：简单项目实战](tutorials/tutorial-1-simple-project.md)** - 通过实际项目学习基本工作流

### 📚 深入理解

当你熟悉基本使用后，可以深入了解：

1. **[系统架构设计](architecture.md)** - Rick 的整体架构和模块关系
2. **[模块详细文档](modules/README.md)** - 各个模块的详细说明
3. **[最佳实践](best-practices.md)** - 高效使用 Rick 的技巧和建议

### 🔧 模块文档

深入了解各个核心模块：

- **[基础设施模块](modules/infrastructure.md)** - Go 项目初始化、CLI 框架、工作空间
- **[内容解析模块](modules/parser.md)** - Markdown 解析、task.md、debug.md 解析
- **[DAG 执行模块](modules/dag_executor.md)** - DAG 构建、拓扑排序、任务执行、重试机制
- **[提示词管理模块](modules/prompt_manager.md)** - 模板管理、提示词构建、上下文注入
- **[命令处理模块](modules/cli_commands.md)** - plan、doing、learning 命令实现
- **[工作空间管理](modules/workspace.md)** - .rick 目录结构和管理
- **[配置管理](modules/config.md)** - 配置文件格式和加载机制
- **[日志系统](modules/logging.md)** - 简化的日志设计
- **[Git 集成](modules/git.md)** - Git 操作和版本管理
- **[Claude Code CLI 交互](modules/callcli.md)** - 与 Claude Code 的集成

### 📖 实战教程

通过实际案例学习 Rick CLI：

1. **[简单项目实战](tutorials/tutorial-1-simple-project.md)** - 创建一个简单的 Go Web 项目
2. **[使用 Rick 重构 Rick](tutorials/tutorial-2-self-refactor.md)** - 自我重构的完整流程
3. **[并行开发版本](tutorials/tutorial-3-parallel-versions.md)** - 生产版本和开发版本并行运行
4. **[自定义提示词模板](tutorials/tutorial-4-custom-prompts.md)** - 定制化提示词以适应特定需求
5. **[CI/CD 集成](tutorials/tutorial-5-cicd-integration.md)** - 将 Rick 集成到持续集成流程

## 使用指南

### 如何阅读本 Wiki

- **顺序阅读**：如果你是新手，建议从"新手入门"部分开始，按顺序阅读
- **主题阅读**：如果你对某个特定主题感兴趣，可以直接跳转到相应的模块文档
- **问题驱动**：如果你遇到具体问题，可以在相关模块文档中查找解决方案

### 文档约定

本 Wiki 使用以下约定：

- **代码块**：使用 \`\`\` 标记的代码块表示可执行的命令或代码示例
- **文件路径**：使用 `path/to/file` 格式表示文件路径
- **命令**：使用 `rick command` 格式表示 CLI 命令
- **重要提示**：使用 ⚠️ 标记重要注意事项
- **最佳实践**：使用 ✅ 标记推荐做法
- **反模式**：使用 ❌ 标记不推荐的做法

### 术语表

- **Context-First**：上下文优先的设计理念，强调完整的上下文信息是高质量输出的关键
- **DAG**：有向无环图（Directed Acyclic Graph），用于表示任务依赖关系
- **Job**：一个完整的工作单元，包含 plan、doing、learning 三个阶段
- **Task**：Job 中的一个具体任务，是最小的执行单元
- **OKR**：目标与关键结果（Objectives and Key Results），用于定义项目目标
- **SPEC**：技术规范（Specification），详细描述项目的技术实现

## 核心命令速查

```bash
# 规划任务
rick plan "任务描述"

# 执行任务
rick doing job_n

# 知识积累
rick learning job_n
```

## 版本信息

- **当前版本**：v1.0.0-dev
- **Go 版本要求**：>= 1.21
- **支持平台**：macOS, Linux

## 贡献指南

如果你发现文档中的错误或有改进建议，欢迎：

1. 在项目仓库提交 Issue
2. 提交 Pull Request 改进文档
3. 在社区讨论中分享你的使用经验

## 相关资源

- **项目仓库**：[GitHub](https://github.com/yourusername/rick)
- **OKR 文档**：`.rick/OKR.md` - 项目目标和关键结果
- **SPEC 文档**：`.rick/SPEC.md` - 技术规范
- **Skills 库**：`.rick/skills/` - 可复用技能库
- **Learning 记录**：`.rick/jobs/*/learning/` - 历史项目的知识积累

## 更新日志

- **2026-03-15**：创建 Wiki 目录结构和索引文件
- 更多更新记录请查看项目的 Git 提交历史

---

**提示**：本文档会随着项目的发展持续更新。建议定期查看最新版本以获取最新信息。
