# Infrastructure Module（基础设施模块）

## 概述
Infrastructure Module 提供 Rick CLI 的基础设施支持，包括配置管理、日志系统、Git 操作、CLI 交互和工作空间管理。

## 模块位置
- `internal/config/` - 配置管理
- `internal/git/` - Git 操作
- `internal/callcli/` - Claude Code CLI 交互
- `internal/workspace/` - 工作空间管理

## 核心组件

### 1. Config（配置管理）
**文件**: `internal/config/config.go`

**职责**:
- 加载和保存全局配置（`~/.rick/config.json`）
- 提供配置访问接口
- 配置验证

**配置结构**:
```go
type Config struct {
    MaxRetries      int    `json:"max_retries"`       // 最大重试次数
    ClaudeCodePath  string `json:"claude_code_path"`  // Claude Code CLI 路径
    LogLevel        string `json:"log_level"`         // 日志级别
}
```

**使用示例**:
```go
cfg, err := config.Load()
if err != nil {
    log.Fatal(err)
}

maxRetries := cfg.MaxRetries
```

### 2. Logger（日志系统）
**设计原则**: 简化设计，使用 Go 标准库 `log`

**日志级别**:
- INFO: 一般信息
- WARN: 警告信息
- ERROR: 错误信息

**使用示例**:
```go
log.Println("[INFO] 开始执行任务")
log.Println("[ERROR] 任务执行失败:", err)
```

### 3. Git（Git 操作）
**文件**: `internal/git/git.go`

**职责**:
- 自动初始化 Git 仓库（项目根目录）
- Git 提交（每个 task 完成后）
- 分支管理
- 检查 Git 状态

**核心函数**:
```go
// EnsureGitInitialized 确保 Git 仓库已初始化
func EnsureGitInitialized(projectRoot string) error

// Commit 提交代码
func Commit(message string) error

// GetStatus 获取 Git 状态
func GetStatus() (string, error)
```

**使用示例**:
```go
// 自动初始化 Git（首次 doing 时）
err := git.EnsureGitInitialized(".")
if err != nil {
    log.Fatal(err)
}

// 提交代码（task 完成后）
err = git.Commit("feat: 完成 task1")
```

### 4. CallCLI（Claude Code CLI 交互）
**文件**: `internal/callcli/callcli.go`

**职责**:
- 调用 Claude Code CLI
- 传递提示词和上下文
- 处理 CLI 输出

**核心函数**:
```go
// CallClaudeCLI 调用 Claude Code CLI
func CallClaudeCLI(prompt string) error
```

**使用示例**:
```go
prompt := promptManager.BuildPrompt("doing", context)
err := callcli.CallClaudeCLI(prompt)
```

### 5. Workspace（工作空间管理）
**文件**: `internal/workspace/workspace.go`

**职责**:
- 管理 `.rick/` 目录结构
- 创建 job 目录（job_n/plan/, doing/, learning/）
- 读写 tasks.json
- 管理知识库（.rick/knowledge/）

**核心函数**:
```go
// EnsureWorkspace 确保工作空间存在
func EnsureWorkspace() error

// CreateJobDir 创建 job 目录
func CreateJobDir(jobID string, stage string) (string, error)

// LoadTasksJSON 加载 tasks.json
func LoadTasksJSON(jobDir string) ([]Task, error)

// SaveTasksJSON 保存 tasks.json
func SaveTasksJSON(jobDir string, tasks []Task) error
```

**使用示例**:
```go
// 确保工作空间存在（首次 plan 时）
err := workspace.EnsureWorkspace()

// 创建 job 目录
jobDir, err := workspace.CreateJobDir("job_1", "plan")

// 加载 tasks.json
tasks, err := workspace.LoadTasksJSON(jobDir)
```

## 设计原则

### 1. 最小化外部依赖
- 优先使用 Go 标准库（`log`, `os`, `path/filepath`, `encoding/json`）
- 避免引入重型框架

### 2. 简化设计
- 日志系统：仅使用标准库 `log`，文本格式
- 配置系统：单一全局配置文件
- 避免过度抽象

### 3. 自动初始化
- 首次 `plan` 自动创建工作空间
- 首次 `doing` 自动初始化 Git
- 无需手动 `init` 命令

## 测试

### 单元测试
```bash
go test ./internal/config/
go test ./internal/git/
go test ./internal/workspace/
```

### 测试覆盖
- Config: 配置加载、保存、验证
- Git: 初始化、提交、状态检查
- Workspace: 目录创建、JSON 读写

## 最佳实践

1. **配置管理**: 使用 `config.Load()` 加载配置，避免硬编码
2. **日志记录**: 使用统一的日志格式 `[LEVEL] message`
3. **错误处理**: 使用 `log.Fatal()` 处理致命错误，返回 `error` 处理可恢复错误
4. **工作空间操作**: 使用 `workspace` 包的函数，避免直接操作文件系统

## 未来优化

1. **配置热重载**: 支持动态重载配置
2. **结构化日志**: 使用 JSON 格式日志（可选）
3. **Git Hooks**: 支持 pre-commit、post-commit hooks
4. **工作空间清理**: 自动清理过期的 job 目录

---

*最后更新: 2026-03-14*
