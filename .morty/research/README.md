# Rick CLI Go 开发研究文档

## 📚 文档导航

本目录包含 Rick CLI 使用 Go 语言开发的完整研究和规范文档。

### 🚀 快速开始（5分钟）
- **[QUICK_REFERENCE.md](QUICK_REFERENCE.md)** - 快速参考卡片
  - 核心命令速查
  - 常用脚本命令
  - 文件位置速查
  - 故障排查

### 📖 详细文档

#### 1. 研究报告（深度分析）
- **[research/使用_golang_开发_rick_命令行程序.md](research/使用_golang_开发_rick_命令行程序.md)** - 完整研究报告
  - 项目概述与核心理论
  - Morty 参考实现分析
  - Rick 的简化设计原则
  - 技术栈选择详解
  - 版本管理与安装机制
  - 核心算法与执行流程
  - 实现阶段规划（7个阶段）
  - 潜在挑战与解决方案
  - **规模**: 11KB, 11个主要章节

#### 2. 开发指南（实操规范）
- **[DEVELOPMENT_GUIDE.md](DEVELOPMENT_GUIDE.md)** - 完整开发指南
  - 版本管理规范
  - 安装脚本详解（build.sh, install.sh, uninstall.sh, update.sh）
  - 开发工作流（4个典型场景）
  - 代码组织规范
  - 测试规范
  - 提交规范
  - 常见问题解答
  - **规模**: 11KB, 完整实操指南

#### 3. 研究总结（快速参考）
- **[RESEARCH_SUMMARY.md](RESEARCH_SUMMARY.md)** - 研究成果总结
  - 研究成果清单
  - 关键设计决策
  - 项目结构概览
  - 技术栈总结
  - 实现阶段规划
  - 开发工作流示例
  - **规模**: 5.9KB, 快速参考

#### 4. 项目记忆（持久化）
- **[~/.claude/projects/.../memory/MEMORY.md](~/.claude/projects/-Users-sunquan-ai-coding-CODING-rick/memory/MEMORY.md)** - 项目记忆库
  - 项目概述
  - 核心设计原则
  - 关键技术决策
  - 项目结构
  - 参考资源
  - **用途**: 跨会话记忆，自动加载到上下文

---

## 🎯 按使用场景选择文档

### 场景1：我是新手，想快速了解 Rick CLI
1. 阅读 [QUICK_REFERENCE.md](QUICK_REFERENCE.md) - 5分钟快速入门
2. 查看 [RESEARCH_SUMMARY.md](RESEARCH_SUMMARY.md) - 了解核心概念
3. 参考 [DEVELOPMENT_GUIDE.md](DEVELOPMENT_GUIDE.md) - 学习开发流程

### 场景2：我要开始开发 Rick CLI
1. 阅读 [RESEARCH_SUMMARY.md](RESEARCH_SUMMARY.md) - 理解设计决策
2. 查看 [research/使用_golang_开发_rick_命令行程序.md](research/使用_golang_开发_rick_命令行程序.md) - 深入技术细节
3. 遵循 [DEVELOPMENT_GUIDE.md](DEVELOPMENT_GUIDE.md) - 按规范开发

### 场景3：我要维护或扩展 Rick CLI
1. 查看 [DEVELOPMENT_GUIDE.md](DEVELOPMENT_GUIDE.md) - 了解代码组织
2. 参考 [QUICK_REFERENCE.md](QUICK_REFERENCE.md) - 快速查找信息
3. 查阅 [research/使用_golang_开发_rick_命令行程序.md](research/使用_golang_开发_rick_命令行程序.md) - 深入理解设计

### 场景4：我要使用 rick_dev 进行自我重构
1. 阅读 [DEVELOPMENT_GUIDE.md](DEVELOPMENT_GUIDE.md) 中的"开发工作流"部分
2. 参考 [QUICK_REFERENCE.md](QUICK_REFERENCE.md) 中的安装脚本命令
3. 查看 [RESEARCH_SUMMARY.md](RESEARCH_SUMMARY.md) 中的"开发工作流示例"

---

## 📊 关键设计决策一览

### 简化原则（vs Morty）
- ✅ 最小化日志系统：仅使用 Go 标准库 `log`，文本格式
- ✅ 移除状态追踪命令：通过 Git 本身进行版本管理
- ✅ 简化配置系统：仅需 `~/.rick/config.json` 一个全局配置

### 独特模块设计 ⭐
- **提示词管理模块** (`internal/prompt/`)：独立管理提示词、构建器、模板
- **版本管理机制**：生产版本 + 开发版本，支持并行运行和自我重构

### 核心特性
- **串行执行**：按拓扑排序顺序执行任务
- **失败重试**：可配置重试次数（默认5次）
- **问题记录**：每次失败记录到 debug.md
- **人工干预**：超过重试限制后退出，由人工修改 task.md

---

## 🛠 技术栈速览

| 功能 | 选择 | 理由 |
|------|------|------|
| CLI 框架 | Cobra | 业界标准 |
| JSON 处理 | encoding/json | 标准库 |
| Markdown 解析 | Goldmark | 功能完整 |
| DAG/拓扑排序 | 自实现 | 简单场景 |
| 日志 | log | 标准库 |
| 配置管理 | encoding/json | 标准库 |
| 文件操作 | os/io | 标准库 |
| Git 操作 | go-git | 纯 Go 实现 |

**原则**：最小化外部依赖，优先使用 Go 标准库

---

## 📁 项目结构

```
rick/
├── cmd/rick/main.go                 # 入口点
├── internal/
│   ├── cmd/                         # 命令处理器（4个）
│   ├── config/                      # 配置管理（简化版）
│   ├── workspace/                   # 工作空间管理
│   ├── parser/                      # 内容解析
│   ├── executor/                    # 任务执行引擎
│   ├── prompt/                      # ⭐ 提示词管理模块
│   ├── git/                         # Git 操作
│   └── callcli/                     # Claude Code CLI 交互
├── scripts/
│   ├── build.sh                     # 构建脚本
│   ├── install.sh                   # 安装脚本
│   ├── uninstall.sh                 # 卸载脚本
│   └── update.sh                    # 更新脚本
└── README.md
```

---

## 🚀 实现阶段规划

| 阶段 | 内容 | 周期 |
|------|------|------|
| Phase 1 | 基础设施（go mod, cobra, 工作空间, 配置, 日志） | Week 1 |
| Phase 2 | 核心解析（Markdown, DAG, 提示词管理） | Week 1-2 |
| Phase 3 | 执行引擎（Claude Code 集成, 测试, 重试循环） | Week 2-3 |
| Phase 4 | Git 与提交（自动提交） | Week 3 |
| Phase 5 | 安装机制（build/install/uninstall/update 脚本） | Week 3-4 |
| Phase 6 | Learning 阶段 | Week 4 |
| Phase 7 | 测试与文档 | Week 4-5 |

---

## 📝 核心命令

```bash
# 项目初始化
rick init

# 规划任务
rick plan "需求描述"

# 执行任务
rick doing job_1

# 知识积累
rick learning job_1

# 安装开发版本
./install.sh --source --dev

# 卸载开发版本
./uninstall.sh --dev

# 更新生产版本
./update.sh
```

---

## 💡 开发工作流示例

### 开发新功能
```bash
./install.sh --source --dev        # 安装 dev 版本
rick_dev plan "新功能"              # 使用 dev 版本规划
rick_dev doing job_1                # 使用 dev 版本执行
rick plan "集成新功能"              # 使用生产版本集成
rick doing job_2
./uninstall.sh --dev               # 卸载 dev 版本
```

### 使用 rick 重构 rick
```bash
rick plan "重构 Rick 架构"           # 使用生产版本规划
./install.sh --source --dev        # 安装 dev 版本
rick doing job_1                    # 使用生产版本执行
rick_dev plan "验证重构"            # 使用 dev 版本验证
rick_dev doing job_2
./update.sh                         # 更新生产版本
./uninstall.sh --dev               # 卸载 dev 版本
```

---

## 🔍 文档版本信息

| 项目 | 版本 | 最后更新 | 状态 |
|------|------|---------|------|
| 研究报告 | 2.0 | 2026-03-13 | ✅ 完成 |
| 开发指南 | 1.0 | 2026-03-13 | ✅ 完成 |
| 研究总结 | 1.0 | 2026-03-13 | ✅ 完成 |
| 快速参考 | 1.0 | 2026-03-13 | ✅ 完成 |
| 项目记忆 | 1.0 | 2026-03-13 | ✅ 完成 |

---

## ✅ 研究成果

本研究已完成以下工作：

- ✅ 分析 Rick 项目核心理论和 Morty 参考实现
- ✅ 设计简化的 Rick CLI 架构
- ✅ 设计独特的提示词管理模块
- ✅ 设计优雅的版本管理机制
- ✅ 规划 7 个实现阶段
- ✅ 编写完整的开发规范和工作流
- ✅ 创建 4 个详细文档和 1 个快速参考
- ✅ 记录项目记忆用于跨会话参考

---

## 🎓 关键学习成果

1. **Rick 的核心价值**在于 Context Loop，而非单纯的 Agent Loop
2. **简化设计原则**：删除不必要的复杂性，保留核心功能
3. **提示词管理**是 AI 编程工具的关键模块
4. **版本管理机制**需要支持自我重构的灵活性
5. **失败重试与人工干预**的平衡至关重要

---

## 📞 支持

如有问题或建议，请参考：
- [常见问题解答](DEVELOPMENT_GUIDE.md#常见问题)
- [故障排查](QUICK_REFERENCE.md#故障排查)
- [Rick 项目规范](Rick_Project_Complete_Description.md)
- [Morty 参考实现](../morty/)

---

**研究完成**: 2026-03-13
**文档版本**: 1.0
**状态**: ✅ 已完成
