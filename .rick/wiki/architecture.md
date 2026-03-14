# Rick CLI 架构设计

## 系统架构概览

```
┌─────────────────────────────────────────────────────────────┐
│                        Rick CLI                             │
│                  (Context-First Framework)                  │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
        ┌─────────────────────────────────────────┐
        │          CLI Commands Module            │
        │  (plan, doing, learning commands)       │
        └─────────────────────────────────────────┘
                 │              │              │
        ┌────────┘              │              └────────┐
        ▼                       ▼                       ▼
┌──────────────┐      ┌──────────────────┐    ┌──────────────┐
│ Prompt       │      │  DAG Executor    │    │  Workspace   │
│ Manager      │◄─────│  Module          │────│  Module      │
└──────────────┘      └──────────────────┘    └──────────────┘
        │                      │                       │
        │                      ▼                       │
        │              ┌──────────────┐                │
        │              │   Parser     │                │
        │              │   Module     │                │
        │              └──────────────┘                │
        │                      │                       │
        ▼                      ▼                       ▼
┌──────────────────────────────────────────────────────────┐
│              Infrastructure Module                       │
│  (Config, Logger, Git, CallCLI, Workspace)               │
└──────────────────────────────────────────────────────────┘
```

## 核心模块职责

### 1. Infrastructure Module（基础设施模块）
**位置**: `internal/config/`, `internal/git/`, `internal/callcli/`, `internal/workspace/`

**职责**:
- **Config**: 全局配置管理（`~/.rick/config.json`）
- **Logger**: 简化日志系统（Go 标准库 `log`）
- **Git**: Git 操作（自动初始化、提交、分支管理）
- **CallCLI**: Claude Code CLI 交互
- **Workspace**: 工作空间管理（`.rick/` 目录结构）

**关键特性**:
- 使用 Go 标准库为主，最小化外部依赖
- 自动初始化机制（首次 `plan` 创建工作空间，首次 `doing` 初始化 Git）

### 2. Parser Module（内容解析模块）
**位置**: `internal/parser/`

**职责**:
- 解析 Markdown 文件（使用 Goldmark 库）
- 解析 `task.md` 格式（依赖关系、任务目标、关键结果、测试方法）
- 解析 `debug.md` 格式（问题记录）
- 解析 OKR/SPEC 文档

**关键数据结构**:
```go
type Task struct {
    TaskID      string
    TaskName    string
    Dependencies []string
    Objectives  string
    KeyResults  []string
    TestMethods []string
}
```

### 3. DAG Executor Module（DAG 执行模块）
**位置**: `internal/executor/`

**职责**:
- 构建任务依赖图（DAG）
- 拓扑排序（Kahn 算法，自实现）
- 串行执行任务（按拓扑排序顺序）
- 失败重试机制（默认最多5次）
- 状态管理（pending, doing, done, failed）

**执行流程**:
```
1. 加载 tasks.json
2. 构建 DAG（拓扑排序）
3. 对每个 task:
   a. 生成测试脚本
   b. 调用 Claude Code 执行
   c. 运行测试脚本
   d. 通过 → git commit + 标记 done
   e. 失败 → 记录 debug.md + 重试
4. 超过重试限制 → 退出，人工干预
```

### 4. Prompt Manager Module（提示词管理模块）⭐
**位置**: `internal/prompt/`

**职责**:
- 提示词模板管理（`templates/plan.md`, `doing.md`, `learning.md`）
- 提示词构建（动态生成上下文）
- 多阶段提示词生成（Plan、Doing、Learning）
- 上下文注入（项目背景、任务依赖、问题记录）

**核心创新**:
```go
// PromptManager 管理提示词模板和构建
type PromptManager struct {
    TemplateDir string
}

// PromptBuilder 构建特定阶段的提示词
type PromptBuilder struct {
    Stage       string // plan, doing, learning
    Context     map[string]interface{}
    Template    string
}
```

### 5. CLI Commands Module（命令处理模块）
**位置**: `internal/cmd/`

**职责**:
- `plan.go`: 规划任务（创建 job_n/plan/）
- `doing.go`: 执行任务（创建 job_n/doing/，调用 DAG Executor）
- `learning.go`: 知识积累（创建 job_n/learning/）

**命令流程**:
```bash
rick plan "任务描述"
  → 创建 .rick/jobs/job_n/plan/
  → 生成 tasks/*.md
  → 生成 tasks.json

rick doing job_n
  → 自动初始化 Git（如果需要）
  → 加载 tasks.json
  → 调用 DAG Executor 执行
  → 每个 task 完成后 git commit

rick learning job_n
  → 创建 job_n/learning/
  → 人工审核和知识沉淀
  → 更新 .rick/knowledge/ 知识库
```

### 6. Git Module（Git 操作模块）
**位置**: `internal/git/`

**职责**:
- 自动初始化 Git 仓库（项目根目录）
- Git 提交（每个 task 完成后）
- 分支管理
- 检查 Git 状态

### 7. CallCLI Module（CLI 交互模块）
**位置**: `internal/callcli/`

**职责**:
- 调用 Claude Code CLI
- 传递提示词和上下文
- 处理 CLI 输出

### 8. Workspace Module（工作空间模块）
**位置**: `internal/workspace/`

**职责**:
- 管理 `.rick/` 目录结构
- 创建 job 目录（job_n/plan/, doing/, learning/）
- 读写 tasks.json
- 管理知识库（.rick/knowledge/）

## 数据流向图

```
┌─────────────┐
│   Plan      │  人类规划任务
│   Stage     │
└──────┬──────┘
       │
       ▼
┌─────────────────────────────────────┐
│  .rick/jobs/job_n/plan/             │
│  ├── tasks/                         │
│  │   ├── task1.md                   │
│  │   └── task2.md                   │
│  └── tasks.json                     │
└──────┬──────────────────────────────┘
       │
       ▼
┌─────────────┐
│   Doing     │  自动执行任务
│   Stage     │
└──────┬──────┘
       │
       ▼
┌─────────────────────────────────────┐
│  1. Load tasks.json                 │
│  2. Build DAG (Topological Sort)    │
│  3. For each task:                  │
│     a. Generate test script         │
│     b. Call Claude Code CLI         │
│     c. Run test script              │
│     d. Pass → git commit            │
│     e. Fail → retry (max 5)         │
└──────┬──────────────────────────────┘
       │
       ▼
┌─────────────────────────────────────┐
│  .rick/jobs/job_n/doing/            │
│  ├── tasks.json (updated)           │
│  ├── debug.md (if failed)           │
│  └── test_scripts/                  │
└──────┬──────────────────────────────┘
       │
       ▼
┌─────────────┐
│  Learning   │  人工知识沉淀
│   Stage     │
└──────┬──────┘
       │
       ▼
┌─────────────────────────────────────┐
│  .rick/jobs/job_n/learning/         │
│  ├── summary.md                     │
│  ├── insights.md                    │
│  └── knowledge/                     │
└──────┬──────────────────────────────┘
       │
       ▼
┌─────────────────────────────────────┐
│  .rick/knowledge/ (知识库更新)       │
│  ├── patterns/                      │
│  ├── best_practices/                │
│  └── lessons_learned/               │
└─────────────────────────────────────┘
```

## 技术栈说明

### 编程语言
- **Go 1.23+**: 主要编程语言

### 核心依赖
- **标准库优先**: 使用 Go 标准库（`log`, `os`, `path/filepath`, `encoding/json`）
- **Goldmark**: Markdown 解析（唯一外部依赖）
- **Cobra**: CLI 框架（可选，可用标准库替代）

### 外部工具集成
- **Claude Code CLI**: AI 编码工具（通过 `callcli` 模块调用）
- **Git**: 版本控制（通过 `git` 模块操作）

### 设计模式
- **模块化设计**: 8个独立模块，职责清晰
- **依赖注入**: 模块间通过接口交互
- **策略模式**: 提示词管理支持多阶段策略
- **状态机**: 任务状态管理（pending → doing → done/failed）

## 版本管理机制

### 生产版本 vs 开发版本
```
~/.rick/                    # 生产环境
├── bin/rick                # 生产二进制
└── config.json             # 全局配置

~/.rick_dev/                # 开发环境
├── bin/rick_dev            # 开发二进制
└── config.json             # 独立配置
```

**使用场景**:
- 生产版本（`rick`）: 日常使用
- 开发版本（`rick_dev`）: 开发新功能、自我重构
- 支持并行运行，互不干扰

### 安装脚本
```bash
# 生产安装
./install.sh

# 开发安装
./install.sh --source --dev

# 更新
./update.sh [--dev]

# 卸载
./uninstall.sh [--dev]
```

## 关键设计决策

### 1. 简化设计（vs Morty）
- ❌ 移除复杂日志系统 → ✅ 使用 Go 标准库 `log`
- ❌ 移除 5 层配置系统 → ✅ 单一全局配置
- ❌ 移除 status/reset 命令 → ✅ 通过 Git 管理版本
- ❌ 移除 init 命令 → ✅ 自动初始化机制

### 2. Context Loop vs Agent Loop
- **Agent Loop**: 完全自动化（Morty 的 `init` 命令）
- **Context Loop**: 人类控制（Rick 的 `plan` → `doing` → `learning`）

**Rick 的选择**: Context Loop，由人类完全控制循环

### 3. DAG 拓扑排序
- **算法**: Kahn 算法
- **实现**: 自实现（无外部依赖）
- **执行**: 串行执行（简单实现）

### 4. 失败重试机制
- **默认重试次数**: 5 次
- **重试策略**: 记录 debug.md，下次执行加载上下文
- **超限处理**: 退出进程，需人工干预

### 5. 提示词管理（核心创新）
- **模板化**: 支持多阶段模板（plan.md, doing.md, learning.md）
- **上下文注入**: 动态生成提示词（项目背景、任务依赖、问题记录）
- **灵活性**: 支持自定义模板

## 性能和可扩展性

### 性能考虑
- **串行执行**: 当前实现为串行，简单可靠
- **未来优化**: 可支持并行执行（基于 DAG 的层级）

### 可扩展性
- **模块化设计**: 易于添加新模块
- **接口抽象**: 易于替换实现（如 Parser、Executor）
- **提示词模板**: 易于定制不同阶段的提示词

## 安全和稳定性

### 安全措施
- **Git 自动提交**: 每个 task 完成后自动提交，防止丢失代码
- **失败重试**: 最多5次重试，避免无限循环
- **人工干预**: 超过重试限制后退出，需人工修改

### 稳定性保证
- **最小化依赖**: 减少外部依赖，降低风险
- **错误处理**: 完善的错误处理和日志记录
- **测试覆盖**: 核心模块有单元测试

---

*最后更新: 2026-03-14*
