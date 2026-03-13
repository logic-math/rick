# Plan 索引

**生成时间**: 2026-03-14T10:30:00+08:00

**对应 Research**:
- `.morty/research/使用_golang_开发_rick_命令行程序.md` - 项目概述、技术栈、实现阶段
- `.morty/research/DEVELOPMENT_GUIDE.md` - 开发规范、安装脚本规范
- `.morty/research/RESEARCH_SUMMARY.md` - 关键设计决策、技术栈总结
- `.morty/research/QUICK_REFERENCE.md` - 快速参考卡片

**现有实现探索**: 否

## 模块列表

| 模块名称 | 文件 | Jobs 数量 | 依赖模块 | 状态 |
|----------|------|-----------|----------|------|
| 基础设施 | infrastructure.md | 7 | 无 | 规划中 |
| 内容解析 | parser.md | 6 | infrastructure | 规划中 |
| DAG 执行 | dag_executor.md | 7 | infrastructure, parser | 规划中 |
| 提示词管理 | prompt_manager.md | 8 | infrastructure, parser | 规划中 |
| Git 集成 | git_integration.md | 5 | infrastructure | 规划中 |
| CLI 命令 | cli_commands.md | 7 | infrastructure, parser, dag_executor, prompt_manager, git_integration | 规划中 |
| 安装脚本 | installation.md | 7 | infrastructure, cli_commands | 规划中 |
| E2E 测试 | e2e_test.md | 8 | 所有模块 | 规划中 |

## 依赖关系图

```text
infrastructure
    ├─→ parser
    │    └─→ dag_executor
    │         └─→ cli_commands
    │              └─→ installation
    │                   └─→ e2e_test
    │
    ├─→ prompt_manager
    │    └─→ dag_executor
    │         └─→ cli_commands
    │              └─→ installation
    │                   └─→ e2e_test
    │
    ├─→ git_integration
    │    └─→ dag_executor
    │         └─→ cli_commands
    │              └─→ installation
    │                   └─→ e2e_test
    │
    └─→ cli_commands
         └─→ installation
              └─→ e2e_test
```

## 执行顺序

**第 1 轮**: 基础设施层
1. infrastructure (无依赖)

**第 2 轮**: 核心功能层（可并行）
2. parser (依赖 infrastructure)
3. prompt_manager (依赖 infrastructure)
4. git_integration (依赖 infrastructure)

**第 3 轮**: DAG 执行层
5. dag_executor (依赖 infrastructure, parser)

**第 4 轮**: 命令层
6. cli_commands (依赖 infrastructure, parser, dag_executor, prompt_manager, git_integration)

**第 5 轮**: 发布层
7. installation (依赖 infrastructure, cli_commands)

**第 6 轮**: 测试层
8. e2e_test (依赖所有模块)

**说明**: 执行顺序基于拓扑排序，确保依赖关系正确。同一轮中的模块可以并行开发。

## 统计信息

- **总模块数**: 8（包括 e2e_test 模块）
- **总 Jobs 数**: 54（包括所有模块的集成测试 Job）
- **预计执行轮次**: 6 轮（基于依赖关系的最长路径）
- **探索子代理使用**: 否

## 模块详细说明

### 1. infrastructure.md - 基础设施（7 Jobs）

**职责**: 搭建 Rick CLI 的基础设施

**关键 Jobs**:
- Job 1: Go 项目初始化
- Job 2: Cobra CLI 框架搭建
- Job 3: 工作空间管理系统
- Job 4: 配置系统实现
- Job 5: 日志系统实现
- Job 6: 错误定义系统
- Job 7: 集成测试

**输出**: CLI 框架、工作空间、配置系统、日志系统

---

### 2. parser.md - 内容解析（6 Jobs）

**职责**: 实现 Markdown 内容解析系统

**关键 Jobs**:
- Job 1: Markdown 基础解析器
- Job 2: Task.md 解析器
- Job 3: Debug.md 处理器
- Job 4: OKR.md 和 SPEC.md 解析器
- Job 5: 多文件解析协调器
- Job 6: 集成测试

**输出**: Task 结构体、DebugInfo 结构体、上下文管理器

---

### 3. dag_executor.md - DAG 执行（7 Jobs）

**职责**: 实现 DAG 构建、拓扑排序和任务执行引擎

**关键 Jobs**:
- Job 1: DAG 构建器
- Job 2: 拓扑排序实现（Kahn 算法）
- Job 3: Tasks.json 生成器
- Job 4: 任务执行器（核心）
- Job 5: 重试机制实现
- Job 6: 执行协调器
- Job 7: 集成测试

**输出**: 拓扑排序结果、tasks.json、执行日志

---

### 4. prompt_manager.md - 提示词管理（8 Jobs）⭐

**职责**: 实现提示词管理模块（Rick 的核心创新）

**关键 Jobs**:
- Job 1: 提示词模板管理器
- Job 2: 提示词构建器
- Job 3: 上下文管理器
- Job 4: 规划阶段提示词生成
- Job 5: 执行阶段提示词生成
- Job 6: 测试脚本生成提示词
- Job 7: 学习阶段提示词生成
- Job 8: 集成测试

**输出**: 完整的提示词字符串、提示词构建器实例

---

### 5. git_integration.md - Git 集成（5 Jobs）

**职责**: 实现 Git 集成，支持自动提交和版本管理

**关键 Jobs**:
- Job 1: Git 基础操作
- Job 2: 自动提交系统
- Job 3: 版本管理
- Job 4: 回滚和恢复
- Job 5: 集成测试

**输出**: Git 操作接口、自动提交系统

---

### 6. cli_commands.md - CLI 命令（7 Jobs）

**职责**: 实现四个核心命令，整合所有底层模块

**关键 Jobs**:
- Job 1: init 命令实现
- Job 2: plan 命令实现
- Job 3: doing 命令实现
- Job 4: learning 命令实现
- Job 5: 命令行参数解析
- Job 6: 错误处理和用户反馈
- Job 7: 集成测试

**输出**: 完整的 CLI 命令集

---

### 7. installation.md - 安装脚本（7 Jobs）

**职责**: 实现安装、卸载、更新脚本，支持版本管理

**关键 Jobs**:
- Job 1: build.sh 脚本实现
- Job 2: install.sh 脚本实现
- Job 3: uninstall.sh 脚本实现
- Job 4: update.sh 脚本实现
- Job 5: 版本管理脚本
- Job 6: 环境检查和配置
- Job 7: 集成测试

**输出**: 4 个安装脚本、版本管理机制

---

### 8. e2e_test.md - 端到端测试（8 Jobs）

**职责**: 验证整个系统的端到端功能、性能和稳定性

**关键 Jobs**:
- Job 1: 开发环境启动验证
- Job 2: 完整工作流测试
- Job 3: 并行版本管理测试
- Job 4: 失败重试和恢复测试
- Job 5: 性能和稳定性测试
- Job 6: 自我重构能力验证
- Job 7: 文档和示例验证
- Job 8: 集成测试

**输出**: E2E 测试报告、性能测试结果

## 关键设计亮点

### 1. 简化设计原则
- ✅ 最小化日志系统：仅使用 Go 标准库 log
- ✅ 移除状态追踪命令：通过 Git 本身管理版本
- ✅ 简化配置系统：仅需 ~/.rick/config.json 一个文件

### 2. 独特模块设计
- ⭐ **提示词管理模块** (prompt_manager)：Rick 的核心创新，独立管理提示词、构建器、模板
- ⭐ **版本管理机制**：支持生产版本 + 开发版本并行运行，便于自我重构

### 3. 执行流程特性
- 串行执行：按拓扑排序顺序执行任务
- 失败重试：可配置重试次数（默认5次）
- 问题记录：每次失败记录到 debug.md
- 人工干预：超过重试限制后退出，由人工修改 task.md

### 4. 技术栈最小化
- CLI 框架：Cobra
- JSON 处理：encoding/json（标准库）
- Markdown 解析：Goldmark
- DAG 算法：自实现（Kahn 算法）
- Git 操作：go-git
- 日志系统：Go 标准库 log

## 预期交付物

### 代码层面
- 完整的 Go 项目结构（cmd/, internal/, pkg/, scripts/)
- 8 个功能模块，54 个 Jobs
- 所有模块都包含集成测试
- 代码覆盖率 >= 80%

### 文档层面
- README.md - 项目说明
- DEVELOPMENT_GUIDE.md - 开发指南
- 代码注释和 docstring

### 脚本层面
- build.sh - 构建脚本
- install.sh - 安装脚本（支持 --dev）
- uninstall.sh - 卸载脚本
- update.sh - 更新脚本

### 发布层面
- 生产版本：~/.rick/bin/rick
- 开发版本：~/.rick_dev/bin/rick_dev
- 全局配置：~/.rick/config.json

## 下一步行动

1. ✅ 完成 Plan 模式（当前）
2. ⏳ 运行 `morty plan validate --verbose` 验证所有 Plan 文件
3. ⏳ 开始 Doing 模式：`morty doing`
4. ⏳ 按执行顺序逐个完成各模块
5. ⏳ 完成所有 Jobs 后进入 Learning 模式

## 快速开始

### 开发流程
```bash
# 1. 查看 Plan 文件
cat .morty/plan/infrastructure.md

# 2. 开始 Doing 模式
morty doing

# 3. 按照 Jobs 逐个完成任务
# 每个 Job 完成后会自动提交

# 4. 完成所有 Jobs 后进入 Learning 模式
morty learning
```

### 自我重构
```bash
# 使用 rick 规划重构
rick plan "重构 Rick 架构"

# 安装开发版本
./install.sh --source --dev

# 使用生产版本执行重构
rick doing job_1

# 使用开发版本验证
rick_dev plan "验证重构"
rick_dev doing job_2

# 更新生产版本
./update.sh

# 卸载开发版本
./uninstall.sh --dev
```

---

**Plan 文件生成完成**: 2026-03-14
**总 Jobs 数**: 54
**预计执行轮次**: 6 轮
**状态**: ✅ 已生成，等待验证
