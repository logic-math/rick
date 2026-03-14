# Rick CLI 核心概念

## 核心公式

```
AICoding = Humans + Agents
其中：Agents = Models + Harness
```

**解释**:
- **Humans**: 人类负责规划、决策、知识沉淀
- **Agents**: AI 代理负责执行具体编码任务
- **Models**: 大语言模型（如 Claude）
- **Harness**: 工具框架（Rick CLI）

## Context Loop vs Agent Loop

### Agent Loop（完全自动化）
**代表**: Morty 的 `init` 命令

```
┌─────────────────────────────────┐
│  Agent Loop (自动化)            │
│                                 │
│  init → plan → doing → learning │
│   ↑                         ↓   │
│   └─────────────────────────┘   │
│        (自动循环)                │
└─────────────────────────────────┘
```

**问题**:
- Claude Code CLI 是交互式工具，导致命令阻塞
- 人类失去控制权，无法及时干预
- 过度自动化，不适合复杂项目

### Context Loop（人类控制）⭐
**代表**: Rick 的 `plan` → `doing` → `learning`

```
┌─────────────────────────────────────────────────┐
│  Context Loop (人类控制)                        │
│                                                 │
│  ┌──────┐      ┌────────┐      ┌──────────┐   │
│  │ Plan │  →   │ Doing  │  →   │ Learning │   │
│  └──────┘      └────────┘      └──────────┘   │
│     ↑              ↓                  ↓         │
│     │              │                  │         │
│     │         (人类审核)          (人类沉淀)     │
│     │              │                  │         │
│     └──────────────┴──────────────────┘         │
│              (人类决策何时进入下一阶段)          │
└─────────────────────────────────────────────────┘
```

**优势**:
- 人类完全控制循环节奏
- 每个阶段结束后可以审核和调整
- 避免 AI 过度自动化导致的问题
- 适合复杂项目和团队协作

**Rick 的设计哲学**:
> "AI 是助手，不是主导。人类应该保持对开发流程的完全控制。"

## DAG 任务调度原理

### 什么是 DAG？
**DAG (Directed Acyclic Graph)**: 有向无环图，用于表示任务之间的依赖关系。

```
task1 ──┐
        ├──→ task3 ──→ task5
task2 ──┘              ↑
                       │
        task4 ─────────┘
```

**特点**:
- **有向**: 任务有明确的依赖方向（A → B 表示 B 依赖 A）
- **无环**: 不存在循环依赖（A → B → C → A）

### 拓扑排序（Kahn 算法）
**目标**: 找到一个任务执行顺序，确保所有依赖都被满足。

**算法步骤**:
1. 计算每个任务的入度（依赖数量）
2. 将入度为 0 的任务加入队列
3. 从队列中取出任务执行
4. 执行后，将依赖该任务的其他任务入度 -1
5. 重复步骤 3-4，直到队列为空

**示例**:
```
输入: task1 → task3, task2 → task3, task3 → task5, task4 → task5
输出: [task1, task2, task4, task3, task5]
```

### Rick 的实现
- **串行执行**: 按拓扑排序顺序串行执行（简单可靠）
- **未来优化**: 可支持并行执行（同一层级的任务可并行）

```go
// DAG 执行流程
func ExecuteDAG(tasks []Task) error {
    // 1. 构建 DAG
    dag := BuildDAG(tasks)

    // 2. 拓扑排序
    sortedTasks := TopologicalSort(dag)

    // 3. 串行执行
    for _, task := range sortedTasks {
        err := ExecuteTask(task)
        if err != nil {
            return err
        }
    }

    return nil
}
```

## 提示词管理机制

### 为什么需要提示词管理？
传统 AI 编码工具的问题：
- 提示词硬编码，难以维护
- 缺乏上下文注入，AI 不了解项目背景
- 无法支持多阶段任务（Plan、Doing、Learning）

### Rick 的提示词管理架构

```
┌─────────────────────────────────────────────────┐
│         Prompt Manager Module                   │
├─────────────────────────────────────────────────┤
│                                                 │
│  ┌───────────────┐      ┌──────────────────┐   │
│  │   Templates   │      │  Prompt Builder  │   │
│  ├───────────────┤      ├──────────────────┤   │
│  │ plan.md       │  →   │ Stage: plan      │   │
│  │ doing.md      │  →   │ Stage: doing     │   │
│  │ learning.md   │  →   │ Stage: learning  │   │
│  └───────────────┘      └──────────────────┘   │
│                                ↓                │
│                         ┌──────────────────┐   │
│                         │ Context Injection│   │
│                         ├──────────────────┤   │
│                         │ - 项目背景       │   │
│                         │ - 任务依赖       │   │
│                         │ - 问题记录       │   │
│                         │ - 重试次数       │   │
│                         └──────────────────┘   │
│                                ↓                │
│                         ┌──────────────────┐   │
│                         │ Final Prompt     │   │
│                         └──────────────────┘   │
└─────────────────────────────────────────────────┘
```

### 提示词模板示例

**doing.md 模板**:
```markdown
# Rick 项目执行阶段提示词

你是一个资深的软件工程师。你的任务是执行规划好的任务，完成具体的编码工作。

## 任务信息
**任务 ID**: {{task_id}}
**任务名称**: {{task_name}}
**重试次数**: {{retry_count}}

### 任务目标
{{objectives}}

### 关键结果
{{key_results}}

### 测试方法
{{test_methods}}

## 项目背景
{{project_context}}

## 执行上下文
### 已完成的任务
{{completed_tasks}}

### 任务依赖
{{dependencies}}

{{#if retry_count > 0}}
### 前次执行的问题记录
{{debug_info}}
{{/if}}

## 执行要求
1. 理解需求
2. 设计方案
3. 编写代码
4. 测试验证
5. 提交代码
```

### 上下文注入机制

**动态数据注入**:
```go
type PromptContext struct {
    TaskID          string
    TaskName        string
    RetryCount      int
    Objectives      string
    KeyResults      []string
    TestMethods     []string
    ProjectContext  string
    CompletedTasks  []string
    Dependencies    []string
    DebugInfo       string
}

func BuildPrompt(template string, context PromptContext) string {
    // 使用模板引擎注入上下文
    return RenderTemplate(template, context)
}
```

**上下文来源**:
- **项目背景**: 从 OKR.md、SPEC.md 读取
- **任务依赖**: 从 tasks.json 的 DAG 关系推导
- **问题记录**: 从 debug.md 读取（如果重试）
- **已完成任务**: 从 tasks.json 的状态读取

## 失败重试机制

### 为什么需要重试？
AI 编码可能因为以下原因失败：
- 理解错误（误解需求）
- 代码错误（语法错误、逻辑错误）
- 测试失败（功能不符合预期）
- 环境问题（依赖缺失、配置错误）

### Rick 的重试策略

```
┌─────────────────────────────────────────────────┐
│  Task Execution with Retry                      │
├─────────────────────────────────────────────────┤
│                                                 │
│  ┌──────────────┐                               │
│  │ Execute Task │                               │
│  └──────┬───────┘                               │
│         │                                       │
│         ▼                                       │
│  ┌──────────────┐     Pass                     │
│  │  Run Tests   │ ─────────→ ✅ git commit     │
│  └──────┬───────┘                               │
│         │ Fail                                  │
│         ▼                                       │
│  ┌──────────────────┐                           │
│  │ Record to        │                           │
│  │ debug.md         │                           │
│  └──────┬───────────┘                           │
│         │                                       │
│         ▼                                       │
│  ┌──────────────────┐                           │
│  │ Retry Count < 5? │                           │
│  └──────┬───────┬───┘                           │
│         │ Yes   │ No                            │
│         ▼       ▼                               │
│  ┌──────────┐  ┌──────────────┐                │
│  │ Retry    │  │ Exit Process │                │
│  │ (Load    │  │ (Human       │                │
│  │ debug.md)│  │ Intervention)│                │
│  └──────────┘  └──────────────┘                │
└─────────────────────────────────────────────────┘
```

### 重试配置
```json
{
  "max_retries": 5,
  "retry_strategy": "incremental_context"
}
```

### debug.md 格式
```markdown
# debug1: 测试失败 - TestParseTask

**问题描述**
执行 `go test ./internal/parser/` 时，TestParseTask 失败。
错误信息：expected 2 dependencies, got 1

**复现步骤**
1. 运行 `go test ./internal/parser/`
2. 观察 TestParseTask 失败

**可能原因**
解析 task.md 时，依赖关系解析逻辑有误。

**解决状态**
未解决

**解决方法**
（待 AI 在重试时填写）
```

### 重试时的上下文增强
- **第 1 次执行**: 仅包含任务信息
- **第 2 次执行**: 包含任务信息 + debug.md（第1次失败记录）
- **第 3 次执行**: 包含任务信息 + debug.md（第1-2次失败记录）
- ...
- **第 6 次执行**: 超过限制，退出进程，需人工干预

## 版本管理机制（rick vs rick_dev）

### 为什么需要两个版本？
**问题**: 如何使用 Rick 来重构 Rick 自身？

**解决方案**: 双版本机制
- **生产版本（rick）**: 稳定版本，用于日常开发
- **开发版本（rick_dev）**: 开发版本，用于自我重构

### 使用场景

#### 场景1: 日常使用
```bash
rick plan "实现新功能"
rick doing job_1
rick learning job_1
```

#### 场景2: 自我重构
```bash
# 安装开发版本
./install.sh --source --dev

# 使用开发版本开发新功能
rick_dev plan "重构 Prompt Manager"
rick_dev doing job_1

# 使用生产版本集成新功能
rick plan "集成 Prompt Manager 重构"
rick doing job_2

# 卸载开发版本
./uninstall.sh --dev
```

### 版本隔离

```
~/.rick/                    # 生产环境
├── bin/rick                # 生产二进制
├── config.json             # 生产配置
└── source/                 # 生产源码（可选）

~/.rick_dev/                # 开发环境
├── bin/rick_dev            # 开发二进制
├── config.json             # 开发配置
└── source/                 # 开发源码（可选）
```

**隔离特性**:
- 独立的配置文件
- 独立的工作空间（`.rick/` vs `.rick_dev/`）
- 独立的命令名称（`rick` vs `rick_dev`）
- 可以同时运行，互不干扰

## 自动初始化机制

### 为什么移除 init 命令？
**问题**: Morty 的 `init` 命令尝试自动执行 plan → doing → learning，但 Claude Code CLI 是交互式工具，导致命令阻塞。

**解决方案**: 移除 `init` 命令，改为自动初始化机制。

### 自动初始化流程

#### 首次 `rick plan`
```bash
rick plan "实现新功能"

# 自动执行：
# 1. 检查 .rick/ 目录是否存在
# 2. 不存在 → 创建 .rick/ 目录结构
# 3. 创建 .rick/jobs/job_1/plan/
# 4. 生成 tasks/*.md 和 tasks.json
```

#### 首次 `rick doing`
```bash
rick doing job_1

# 自动执行：
# 1. 检查项目根目录 .git/ 是否存在
# 2. 不存在 → 在项目根目录初始化 Git
# 3. 创建 .rick/jobs/job_1/doing/
# 4. 执行 DAG 任务
```

### 设计优势
- **无需手动初始化**: 首次使用自动创建
- **人类控制循环**: 由人类决定何时进入下一阶段
- **避免阻塞**: 不再尝试自动执行整个循环

## 工作空间结构

### .rick/ 目录结构
```
.rick/
├── config.json              # 全局配置
├── jobs/                    # 任务目录
│   ├── job_1/
│   │   ├── plan/            # 规划阶段
│   │   │   ├── tasks/
│   │   │   │   ├── task1.md
│   │   │   │   └── task2.md
│   │   │   └── tasks.json
│   │   ├── doing/           # 执行阶段
│   │   │   ├── tasks.json   # 更新后的状态
│   │   │   ├── debug.md     # 问题记录
│   │   │   └── test_scripts/
│   │   └── learning/        # 学习阶段
│   │       ├── summary.md
│   │       ├── insights.md
│   │       └── knowledge/
│   └── job_2/
│       └── ...
├── knowledge/               # 知识库
│   ├── patterns/
│   ├── best_practices/
│   └── lessons_learned/
└── wiki/                    # Wiki 知识库
    ├── index.md
    ├── architecture.md
    ├── core-concepts.md
    └── modules/
```

### 目录职责
- **jobs/**: 存储每个 job 的完整生命周期
- **knowledge/**: 存储跨 job 的知识积累
- **wiki/**: 存储项目文档和架构设计

## 测试策略

### 测试层级
1. **单元测试**: 测试单个模块（如 Parser、DAG Executor）
2. **集成测试**: 测试模块间交互（如 CLI Commands + DAG Executor）
3. **端到端测试**: 测试完整流程（plan → doing → learning）

### 测试脚本生成
Rick 在 `doing` 阶段为每个 task 生成测试脚本：

```bash
# 示例：test_task1.sh
#!/bin/bash

# 测试步骤1：编译项目
go build -o bin/rick cmd/rick/main.go
if [ $? -ne 0 ]; then
    echo "❌ 编译失败"
    exit 1
fi

# 测试步骤2：运行单元测试
go test ./internal/parser/
if [ $? -ne 0 ]; then
    echo "❌ 单元测试失败"
    exit 1
fi

echo "✅ 所有测试通过"
exit 0
```

### 测试驱动开发（TDD）
Rick 鼓励 TDD 流程：
1. **Plan 阶段**: 定义测试方法
2. **Doing 阶段**: 生成测试脚本 → 编写代码 → 运行测试
3. **Learning 阶段**: 总结测试经验

---

*最后更新: 2026-03-14*
