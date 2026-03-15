# Rick CLI Wiki

欢迎来到 Rick CLI 的知识库！这是一个 Context-First AI Coding Framework 的完整文档中心。

## 目录结构

本 Wiki 采用模块化组织，分为以下几个主要部分：

```
wiki/
├── README.md                    # 本文件：Wiki 索引和导航
├── CONTRIBUTING.md              # 贡献指南
├── architecture.md              # 系统架构设计
├── runtime-flow.md              # 运行时流程详解
├── dag-execution.md             # DAG 执行和依赖管理
├── prompt-system.md             # 提示词系统详解
├── testing.md                   # 测试与验证
├── installation.md              # 安装部署指南
└── modules/                     # 模块详细文档
    ├── cmd.md                   # 命令处理模块
    ├── workspace.md             # 工作空间管理
    ├── parser.md                # 内容解析模块
    ├── executor.md              # 任务执行引擎
    ├── prompt.md                # 提示词管理模块
    ├── git.md                   # Git 集成
    └── config.md                # 配置管理
```

## 文档导航

### 🚀 新手入门

如果你是第一次使用 Rick CLI，建议按以下顺序阅读：

1. **[安装部署指南](installation.md)** - 安装、配置和环境准备
2. **[系统架构设计](architecture.md)** - 理解 Rick 的整体架构和设计理念
3. **[运行时流程详解](runtime-flow.md)** - 了解任务执行的完整流程

### 📚 深入理解

当你熟悉基本使用后，可以深入了解：

1. **[DAG 执行和依赖管理](dag-execution.md)** - 任务依赖关系和拓扑排序
2. **[提示词系统详解](prompt-system.md)** - 提示词管理和构建机制
3. **[测试与验证](testing.md)** - 测试策略和验证方法
4. **[模块详细文档](modules/)** - 各个核心模块的详细说明

### 🔧 模块文档

深入了解各个核心模块：

- **[命令处理模块](modules/cmd.md)** - plan、doing、learning 命令实现
- **[工作空间管理](modules/workspace.md)** - .rick 目录结构和管理
- **[内容解析模块](modules/parser.md)** - Markdown 解析、task.md、debug.md 解析
- **[任务执行引擎](modules/executor.md)** - DAG 构建、拓扑排序、任务执行、重试机制
- **[提示词管理模块](modules/prompt.md)** - 模板管理、提示词构建、上下文注入
- **[Git 集成](modules/git.md)** - Git 操作和版本管理
- **[配置管理](modules/config.md)** - 配置文件格式和加载机制

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
