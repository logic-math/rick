# CMD 模块详解

> 命令行接口（CLI）模块 - 基于 Cobra 的命令行架构

## 📋 模块概述

CMD 模块是 Rick CLI 的命令行接口层，负责处理用户输入、参数解析和命令执行。该模块基于 [Cobra](https://github.com/spf13/cobra) 框架构建，提供了三个核心命令：`plan`、`doing` 和 `learning`。

### 功能职责
- 命令行参数解析和验证
- 全局标志（flags）管理
- 命令执行流程控制
- 错误处理和用户交互
- 工作流协调（调用其他模块）

### 模块位置
```
internal/cmd/
├── root.go           # 根命令和全局配置
├── plan.go           # plan 命令实现
├── doing.go          # doing 命令实现
├── learning.go       # learning 命令实现
└── feedback_helper.go # 用户反馈辅助函数
```

---

## 🏗️ 核心类型和接口

### 全局标志变量

```go
var (
    verbose bool   // 启用详细输出
    dryRun  bool   // 干运行模式（不执行实际操作）
    jobID   string // 指定 Job ID
)
```

### 命令结构

Rick CLI 使用 Cobra 的 `Command` 结构：

```go
type cobra.Command struct {
    Use     string              // 命令使用方式
    Short   string              // 简短描述
    Long    string              // 详细描述
    Args    cobra.PositionalArgs // 参数验证
    RunE    func(cmd *cobra.Command, args []string) error // 执行函数
}
```

---

## 🔧 主要函数说明

### 1. Root 命令 (`root.go`)

#### `NewRootCmd(version string) *cobra.Command`

创建根命令，配置全局标志和子命令。

**参数**：
- `version`: 版本号字符串

**返回值**：
- `*cobra.Command`: 配置好的根命令

**关键特性**：
- 注册全局标志：`--verbose`, `--dry-run`, `--job`
- 添加子命令：`plan`, `doing`, `learning`
- 配置版本显示

**示例**：
```go
rootCmd := cmd.NewRootCmd("v1.0.0")
if err := rootCmd.Execute(); err != nil {
    log.Fatal(err)
}
```

#### `validateJobID(id string) error`

验证 Job ID 格式。

**验证规则**：
- 不能为空
- 只允许字母、数字、下划线和连字符
- 最小长度为 1

**示例**：
```go
if err := validateJobID("job_1"); err != nil {
    // 处理错误
}
```

### 2. Plan 命令 (`plan.go`)

#### `NewPlanCmd() *cobra.Command`

创建 `plan` 命令。

**用法**：
```bash
rick plan [requirement]
rick plan "实现用户登录功能"
```

**工作流程**：
1. 获取需求描述（参数或交互式输入）
2. 加载配置和工作空间
3. 生成规划提示词
4. 调用 Claude Code CLI
5. 等待用户完成规划

#### `executePlanWorkflow(requirement string) error`

执行完整的规划工作流。

**步骤详解**：
```
1. 加载配置 (config.LoadConfig)
   └─> 获取 MaxRetries, ClaudeCodePath 等配置

2. 初始化工作空间 (workspace.New)
   └─> 自动创建 .rick 目录结构

3. 生成规划提示词
   ├─> 加载 OKR.md (如果存在)
   ├─> 加载 SPEC.md (如果存在)
   └─> 使用 prompt.GeneratePlanPrompt 生成提示词

4. 调用 Claude Code CLI (交互模式)
   └─> 传递提示词，用户与 AI 交互完成规划

5. 提示用户下一步
   └─> 显示 "rick doing <job_id>" 命令
```

**错误处理**：
- 配置加载失败
- 工作空间创建失败
- 提示词生成失败
- Claude Code CLI 调用失败

### 3. Doing 命令 (`doing.go`)

#### `NewDoingCmd() *cobra.Command`

创建 `doing` 命令。

**用法**：
```bash
rick doing job_1
rick doing --job job_1
```

**工作流程**：
1. 验证 Job ID
2. 加载任务（从 plan 目录）
3. 构建 DAG 并执行任务
4. 自动提交成功的任务
5. 显示执行摘要

#### `executeDoingWorkflow(jobID string) error`

执行完整的任务执行工作流。

**步骤详解**：
```
1. 加载配置和工作空间
   └─> 验证 job 目录结构

2. 自动初始化 Git (ensureGitInitialized)
   ├─> 检查 .git 目录
   └─> 不存在则初始化 Git 仓库

3. 加载任务 (loadTasksFromPlan)
   ├─> 读取 plan/task*.md 文件
   ├─> 解析任务内容
   └─> 提取依赖关系

4. 创建执行器 (executor.NewExecutor)
   ├─> 构建 DAG
   ├─> 拓扑排序
   └─> 生成 tasks.json

5. 执行任务 (exec.ExecuteJob)
   ├─> 按拓扑顺序串行执行
   ├─> 每个任务支持重试
   └─> 失败记录到 debug.md

6. 提交结果 (commitDoingResults)
   └─> 使用 git 提交成功的任务
```

**关键函数**：

##### `loadTasksFromPlan(planDir string) ([]*parser.Task, error)`

从 plan 目录加载所有任务。

**实现细节**：
- 扫描 `task*.md` 文件
- 按文件名数字排序
- 解析每个任务文件
- 提取 task ID（从文件名）

##### `ensureGitInitialized(rickDir string) error`

确保项目根目录已初始化 Git。

**行为**：
- 检查项目根目录的 `.git`
- 不存在则运行 `git init`
- 创建默认 `.gitignore`

##### `printExecutionSummary(result *executor.ExecutionJobResult)`

打印执行摘要。

**显示内容**：
- Job ID
- 执行状态（completed/partial/failed）
- 执行时长
- 任务统计（总数/成功/失败）
- 每个任务的详细状态

### 4. Learning 命令 (`learning.go`)

#### `NewLearningCmd() *cobra.Command`

创建 `learning` 命令。

**用法**：
```bash
rick learning job_1
rick learning --job job_1
```

**工作流程**：
1. 加载执行结果
2. 生成学习提示词
3. 调用 Claude Code CLI
4. 更新文档（OKR.md, SPEC.md, Wiki）
5. 提交学习结果

#### `executeLearningWorkflow(jobID string) error`

执行完整的学习工作流。

**步骤详解**：
```
1. 验证 job 目录结构
   └─> 检查 doing 目录是否存在

2. 加载执行结果 (loadExecutionResults)
   ├─> 读取 execution.log
   ├─> 读取 debug.md
   └─> 获取 git 历史

3. 生成学习提示词 (generateLearningPrompt)
   └─> 包含执行摘要、日志、调试记录

4. 调用 Claude Code CLI
   └─> 生成学习总结

5. 更新文档 (updateDocumentation)
   ├─> 保存 learning_summary.md
   ├─> 更新 OKR.md
   └─> 更新 SPEC.md

6. 提交学习结果 (commitLearningResults)
   └─> git commit 学习文档
```

**关键函数**：

##### `loadExecutionResults(doingDir string, jobID string) (*ExecutionResults, error)`

加载执行结果。

**返回结构**：
```go
type ExecutionResults struct {
    JobID            string
    TotalTasks       int
    SuccessfulTasks  int
    FailedTasks      int
    ExecutionLog     string
    DebugRecords     string
    GitHistory       string
}
```

##### `updateDocumentation(rickDir string, learningResult string, learningDir string) error`

更新项目文档。

**更新内容**：
- `learning_summary.md`: 完整学习总结
- `OKR.md`: 追加关键洞察
- `SPEC.md`: 追加实现笔记

---

## 💡 使用示例

### 示例 1: 完整工作流

```bash
# 1. 规划任务
rick plan "实现用户认证系统"
# AI 会生成 task1.md, task2.md, task3.md

# 2. 执行任务
rick doing job_1
# 自动执行所有任务，失败自动重试

# 3. 知识积累
rick learning job_1
# 生成学习总结，更新文档
```

### 示例 2: 使用全局标志

```bash
# 详细输出模式
rick --verbose doing job_1

# 干运行模式（不执行实际操作）
rick --dry-run doing job_1

# 指定 Job ID（全局标志）
rick --job job_1 doing
```

### 示例 3: 错误处理

```go
// 在代码中使用 CMD 模块
package main

import (
    "log"
    "github.com/sunquan/rick/internal/cmd"
)

func main() {
    rootCmd := cmd.NewRootCmd("v1.0.0")

    if err := rootCmd.Execute(); err != nil {
        log.Fatalf("Command execution failed: %v", err)
    }
}
```

---

## ❓ 常见问题

### Q1: 如何添加新的命令？

**A**: 在 `internal/cmd/` 创建新文件，实现 `NewXxxCmd()` 函数，然后在 `root.go` 中注册：

```go
// 在 NewRootCmd 中添加
rootCmd.AddCommand(NewXxxCmd())
```

### Q2: 如何自定义全局标志？

**A**: 在 `root.go` 中添加新的全局变量和标志：

```go
var customFlag string

func NewRootCmd(version string) *cobra.Command {
    // ...
    rootCmd.PersistentFlags().StringVar(&customFlag, "custom", "", "Custom flag")
    // ...
}
```

### Q3: 命令执行失败如何调试？

**A**: 使用 `--verbose` 标志查看详细日志：

```bash
rick --verbose doing job_1
```

### Q4: 如何实现命令的交互式输入？

**A**: 使用 `promptForRequirement()` 模式：

```go
func promptForRequirement() (string, error) {
    reader := bufio.NewReader(os.Stdin)
    fmt.Print("Enter your requirement: ")
    requirement, err := reader.ReadString('\n')
    if err != nil {
        return "", err
    }
    return strings.TrimSpace(requirement), nil
}
```

### Q5: 命令之间如何共享状态？

**A**: 通过工作空间和文件系统共享状态：
- `plan` 生成 `task*.md` 文件
- `doing` 读取 `task*.md`，生成 `tasks.json` 和 `execution.log`
- `learning` 读取执行结果，更新文档

---

## 🔗 相关模块

- [Workspace 模块](./workspace.md) - 工作空间管理
- [Executor 模块](./dag_executor.md) - 任务执行
- [Prompt Manager 模块](./prompt_manager.md) - 提示词生成
- [Parser 模块](./parser.md) - 内容解析
- [Git 模块](./git.md) - Git 操作
- [Config 模块](./config.md) - 配置管理

---

## 📚 设计原则

### 1. 命令独立性
每个命令（plan/doing/learning）都是独立的，可以单独运行，不依赖其他命令的运行时状态。

### 2. 交互式优先
优先支持交互式输入，提供更好的用户体验：
- `rick plan` 无参数时提示输入需求
- `rick doing` 失败时提示重试

### 3. 渐进式增强
- 基础功能：串行执行任务
- 增强功能：自动重试、Git 集成
- 未来扩展：并行执行、远程执行

### 4. 错误透明化
所有错误都明确返回给用户，附带上下文信息：
```
failed to load tasks: failed to read task file task1.md: open task1.md: no such file or directory
```

---

## 🎯 最佳实践

### 1. 命令设计
- 使用清晰的命令名称（plan/doing/learning）
- 提供简短和详细描述（Short/Long）
- 支持参数和标志两种输入方式

### 2. 错误处理
- 使用 `fmt.Errorf` 包装错误，保留错误链
- 在错误信息中包含上下文
- 区分用户错误和系统错误

### 3. 用户体验
- 提供详细的进度信息（使用 `--verbose`）
- 在关键步骤后提示下一步操作
- 使用表情符号增强可读性（✓ ✗ ⚠）

### 4. 模块协调
- CMD 模块只负责命令行接口
- 业务逻辑委托给专门模块
- 保持命令处理函数简洁

---

*最后更新: 2026-03-14*
