# Rick CLI 模块文档索引

> 本目录包含 Rick CLI 所有核心模块的详细文档

## 📚 核心模块（8个）

### 1. [CMD 模块](./cmd.md)
**命令行接口模块** - 基于 Cobra 的命令行架构

- 命令行参数解析和验证
- 全局标志管理
- 命令执行流程控制
- 三大命令：plan、doing、learning

**关键文件**: `internal/cmd/`

---

### 2. [Workspace 模块](./workspace.md)
**工作空间管理模块** - .rick 目录结构管理

- 工作空间初始化
- Job 目录创建和管理
- 路径管理工具
- 目录结构规范

**关键文件**: `internal/workspace/`

---

### 3. [Prompt Manager 模块](./prompt_manager.md)
**提示词管理模块** - 多阶段提示词生成

- 提示词模板管理
- 嵌入式模板机制
- 上下文构建器
- 变量替换系统

**关键文件**: `internal/prompt/`

---

### 4. [Executor 模块](./dag_executor.md)
**DAG 执行模块** - 任务调度和执行

- DAG 构建和拓扑排序
- 任务执行引擎
- 失败重试机制
- tasks.json 生成和更新

**关键文件**: `internal/executor/`

---

### 5. [Parser 模块](./parser.md)
**内容解析模块** - Markdown 文件解析

- Markdown 解析（Goldmark）
- task.md 解析
- debug.md 解析
- 依赖关系提取

**关键文件**: `internal/parser/`

---

### 6. [Git 模块](./git.md)
**Git 操作模块** - 版本控制集成

- Git 初始化
- 自动提交
- 版本管理
- 回滚机制

**关键文件**: `internal/git/`

---

### 7. [Config 模块](./config.md)
**配置管理模块** - 全局配置管理

- 配置文件加载
- 默认配置
- 配置验证
- 版本隔离（rick vs rick_dev）

**关键文件**: `internal/config/`

---

### 8. [Logging 模块](./logging.md)
**日志系统模块** - 简化日志系统

- 分级日志（INFO/WARN/ERROR/DEBUG）
- 文本格式输出
- 支持文件和标准输出
- 基于 Go 标准库

**关键文件**: `internal/logging/`

---

## 📊 文档统计

| 模块 | 字数 | 章节数 | 代码示例 |
|------|------|--------|----------|
| CMD | 1072 | 10+ | ✅ |
| Workspace | 1323 | 10+ | ✅ |
| Prompt Manager | 708 | 8+ | ✅ |
| Executor | 716 | 8+ | ✅ |
| Parser | 507 | 8+ | ✅ |
| Git | 975 | 10+ | ✅ |
| Config | 1406 | 12+ | ✅ |
| Logging | 1439 | 10+ | ✅ |

**总字数**: 8,146 字  
**平均字数**: 1,018 字/模块

---

## 🎯 文档质量标准

每个模块文档都包含以下章节：

1. ✅ **模块概述** - 功能、职责、模块位置
2. ✅ **核心类型和接口** - 数据结构、类型定义
3. ✅ **主要函数说明** - 详细的函数文档
4. ✅ **使用示例** - 实际代码示例
5. ✅ **常见问题** - FAQ 和解决方案
6. ✅ **相关模块链接** - 模块间交叉引用

---

## 📖 阅读建议

### 新手入门路径
1. [CMD 模块](./cmd.md) - 理解命令行接口
2. [Workspace 模块](./workspace.md) - 了解目录结构
3. [Executor 模块](./dag_executor.md) - 理解任务执行

### 进阶开发路径
1. [Prompt Manager 模块](./prompt_manager.md) - 掌握提示词管理
2. [Parser 模块](./parser.md) - 理解内容解析
3. [Git 模块](./git.md) - 理解版本控制

### 系统架构路径
1. [Config 模块](./config.md) - 配置系统设计
2. [Logging 模块](./logging.md) - 日志系统设计
3. [Executor 模块](./dag_executor.md) - DAG 调度系统

---

## 🔗 相关资源

- [Wiki 首页](../index.md)
- [架构设计](../architecture.md)
- [核心概念](../core-concepts.md)
- [项目 OKR](../../../OKR.md)
- [项目 SPEC](../../../SPEC.md)

---

*最后更新: 2026-03-14*
*文档版本: v1.0.0*
