# Rick CLI Wiki 知识库

欢迎来到 Rick CLI Wiki 知识库！这是一个全面的文档系统，包含 Rick CLI 的架构设计、核心概念和模块详解。

## 📚 快速开始

1. **新手入门**: 从 [index.md](./index.md) 开始，了解文档结构
2. **核心概念**: 阅读 [core-concepts.md](./core-concepts.md) 理解 Rick 的设计哲学
3. **架构设计**: 查看 [architecture.md](./architecture.md) 了解系统架构
4. **模块详解**: 浏览 [modules/](./modules/) 目录深入学习各个模块

## 📖 文档结构

```
.rick/wiki/
├── README.md                   # 本文件
├── index.md                    # Wiki 索引（主入口）
├── architecture.md             # 架构设计文档
├── core-concepts.md            # 核心概念文档
├── verify_wiki.sh              # 文档验证脚本
└── modules/                    # 模块详解目录
    ├── infrastructure.md       # 基础设施模块
    ├── parser.md               # 内容解析模块
    ├── dag_executor.md         # DAG 执行模块
    ├── prompt_manager.md       # 提示词管理模块
    ├── cli_commands.md         # 命令处理模块
    ├── git.md                  # Git 操作模块
    ├── callcli.md              # CLI 交互模块
    └── workspace.md            # 工作空间模块
```

## 🎯 核心内容

### 架构设计（architecture.md）
- 系统架构图
- 8个核心模块职责说明
- 数据流向图
- 技术栈说明

### 核心概念（core-concepts.md）
- Context Loop vs Agent Loop
- DAG 任务调度原理
- 提示词管理机制
- 失败重试机制
- 版本管理机制

### 模块详解（modules/）
- 每个模块的职责、核心函数、使用示例
- 代码实现细节
- 测试方法
- 最佳实践

## ✅ 文档验证

运行验证脚本检查文档完整性：

```bash
./.rick/wiki/verify_wiki.sh
```

## 📊 统计信息

- **核心文档**: 3 个
- **模块文档**: 8 个
- **总行数**: 3500+ 行
- **总文件数**: 11 个

## 🔄 更新记录

- **2026-03-14**: 初始版本，创建完整的 Wiki 知识库

## 🤝 贡献指南

本 Wiki 知识库遵循以下原则：
- **清晰简洁**: 避免冗余，突出重点
- **代码示例**: 包含实际可运行的代码
- **图表辅助**: 使用 ASCII 图表说明架构
- **交叉引用**: 便于在文档间导航
- **及时更新**: 保持与代码同步

## 📝 文档规范

### Markdown 格式
- 使用 ATX 风格标题（`#`）
- 代码块使用三个反引号（` ``` `）
- 列表使用 `-` 或 `1.`
- 链接使用相对路径

### 文档模板
每个模块文档包含以下部分：
1. **概述**: 模块功能简介
2. **模块位置**: 代码路径
3. **核心功能**: 主要功能列表
4. **实现细节**: 代码示例
5. **使用示例**: 实际使用场景
6. **测试**: 测试方法
7. **最佳实践**: 推荐做法
8. **常见问题**: FAQ
9. **未来优化**: 改进方向

## 🔗 相关链接

- [项目 README](../../README.md)
- [OKR 文档](../../OKR.md)
- [SPEC 文档](../../SPEC.md)
- [内存库](~/.claude/projects/-Users-sunquan-ai-coding-CODING-rick/memory/MEMORY.md)

---

*最后更新: 2026-03-14*
