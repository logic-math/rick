# Rick CLI Go 开发研究总结

## 研究完成时间
2026-03-13

## 研究成果

### 1. 核心设计文档
- ✅ **研究报告**: `research/使用_golang_开发_rick_命令行程序.md`
  - 项目概述与核心理论
  - Morty 参考实现分析
  - Rick 的简化设计原则
  - 技术栈选择（最小化依赖）
  - 版本管理与安装机制详细设计
  - 核心算法与执行流程
  - 实现阶段规划
  - 潜在挑战与解决方案

### 2. 开发规范文档
- ✅ **开发指南**: `DEVELOPMENT_GUIDE.md`
  - 版本管理规范（生产版 vs 开发版）
  - 安装脚本使用规范（build.sh, install.sh, uninstall.sh, update.sh）
  - 开发工作流（4个典型场景）
  - 代码组织规范
  - 测试规范
  - 提交规范
  - 常见问题解答

### 3. 内存记录
- ✅ **项目记忆**: `~/.claude/projects/.../memory/MEMORY.md`
  - 项目概述
  - 核心设计原则
  - 关键技术决策
  - 项目结构
  - 安装脚本规范
  - 开发工作流

## 关键设计决策

### 设计原则（vs Morty）
1. **最小化日志系统**
   - ❌ 不需要复杂的日志轮转、多格式支持
   - ✅ 仅使用 Go 标准库 `log`，文本格式

2. **移除状态追踪命令**
   - ❌ 不需要 `status` 和 `reset` 命令
   - ✅ 通过 Git 本身进行版本管理

3. **简化配置系统**
   - ❌ 不需要 5 层级配置
   - ✅ 仅需 `~/.rick/config.json` 一个全局配置

### 独特模块设计
1. **提示词管理模块** (`internal/prompt/`)
   - 独立的提示词管理器
   - 提示词构建器
   - 模板目录（plan.md, doing.md, test.md, learning.md）

2. **版本管理机制**
   - 生产版本：`~/.rick/bin/rick`
   - 开发版本：`~/.rick_dev/bin/rick_dev`
   - 支持并行运行，便于自我重构

### 执行流程
1. **串行执行**：按拓扑排序顺序执行任务
2. **失败重试**：可配置重试次数（默认5次）
3. **问题记录**：每次失败记录到 debug.md
4. **人工干预**：超过重试限制后退出，由人工修改 task.md

## 项目结构

```
rick/
├── cmd/rick/main.go                 # 入口点
├── internal/
│   ├── cmd/                         # 命令处理器（init, plan, doing, learning）
│   ├── config/                      # 配置管理（简化版）
│   ├── workspace/                   # 工作空间管理
│   ├── parser/                      # 内容解析（task.md, debug.md）
│   ├── executor/                    # 任务执行引擎
│   ├── prompt/                      # ⭐ 提示词管理模块
│   ├── git/                         # Git 操作
│   └── callcli/                     # Claude Code CLI 交互
├── scripts/
│   ├── build.sh                     # 构建脚本
│   ├── install.sh                   # 安装脚本（--dev 支持）
│   ├── uninstall.sh                 # 卸载脚本
│   └── update.sh                    # 更新脚本
└── README.md
```

## 技术栈

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

## 实现阶段规划

1. **Phase 1**: 基础设施（go mod, cobra, 工作空间, 配置, 日志）
2. **Phase 2**: 核心解析（Markdown, DAG, 提示词管理）
3. **Phase 3**: 执行引擎（Claude Code 集成, 测试, 重试循环）
4. **Phase 4**: Git 与提交（自动提交）
5. **Phase 5**: 安装机制（build/install/uninstall/update 脚本）
6. **Phase 6**: Learning 阶段
7. **Phase 7**: 测试与文档

## 开发工作流示例

### 场景1：开发新功能
```bash
./install.sh --source --dev        # 安装 dev 版本
rick_dev plan "新功能"              # 使用 dev 版本规划
rick_dev doing job_1                # 使用 dev 版本执行
rick plan "集成新功能"              # 使用生产版本集成
rick doing job_2                    # 使用生产版本执行
./uninstall.sh --dev               # 卸载 dev 版本
```

### 场景2：使用 rick 重构 rick
```bash
rick plan "重构 Rick 架构"           # 使用生产版本规划
./install.sh --source --dev        # 安装 dev 版本
rick doing job_1                    # 使用生产版本执行
rick_dev plan "验证重构"            # 使用 dev 版本验证
rick_dev doing job_2                # 使用 dev 版本执行
./update.sh                         # 更新生产版本
./uninstall.sh --dev               # 卸载 dev 版本
```

## 关键文件格式

### task.md
```markdown
# 依赖关系
task1, task2

# 任务名称
任务标题

# 任务目标
具体目标

# 关键结果
1. 结果1
2. 结果2

# 测试方法
1. 测试步骤1
2. 测试步骤2
```

### tasks.json
```json
[
  {
    "task_id": "task1",
    "task_name": "任务名称",
    "dep": [],
    "state_info": {"status": "pending"}
  }
]
```

### debug.md
```markdown
# debug1: 问题描述

**问题描述**
...

**解决状态**
已解决/未解决

**解决方法**
...
```

## 下一步行动

1. ✅ 完成研究和规范文档
2. ⏳ 创建项目初始化（go mod init）
3. ⏳ 搭建 Cobra CLI 框架
4. ⏳ 实现 Phase 1 基础设施
5. ⏳ 逐步实现后续阶段

## 参考资源

- [研究报告](research/使用_golang_开发_rick_命令行程序.md) - 详细技术分析
- [开发指南](DEVELOPMENT_GUIDE.md) - 开发规范和工作流
- [项目记忆](~/.claude/projects/.../memory/MEMORY.md) - 快速参考
- [Rick 规范](Rick_Project_Complete_Description.md) - 完整项目规范
- [Morty 参考](../morty/) - 参考实现

---

**研究状态**: ✅ 完成
**文档版本**: 1.0
**最后更新**: 2026-03-13
