# 提示词文件化改造

## 改造目标

将 Rick 调用 Claude Code CLI 时的提示词传递方式从 **stdin pipe** 改为 **临时文件**，使命令行更简洁，提示词更易于调试。

## 改造前后对比

### 改造前

```go
// 通过 stdin pipe 传递提示词
cmd := exec.Command(claudePath, "--dangerously-skip-permissions")
stdin, _ := cmd.StdinPipe()
stdin.Write([]byte(promptContent))
stdin.Close()
cmd.Run()
```

**问题**：
- 命令行参数冗长（提示词内容通过 pipe 传递）
- 无法直接查看提示词内容，调试困难
- stdin pipe 管理复杂，容易出错

### 改造后

```go
// 先保存提示词到临时文件，然后传递文件路径
promptFile, _ := builder.BuildAndSave("plan")
defer os.Remove(promptFile)
cmd := exec.Command(claudePath, promptFile)
cmd.Run()
```

**优势**：
- ✅ 命令行简洁：`claude /tmp/rick-plan-xxx.md`
- ✅ 提示词可查看：直接打开 `.md` 文件查看内容
- ✅ 代码更简单：无需管理 stdin pipe
- ✅ 调试友好：可以手动修改临时文件测试

## 改造内容

### 1. 核心模块：PromptBuilder

**文件**: `internal/prompt/builder.go`

添加两个新方法：

#### BuildAndSave (推荐使用)
```go
// BuildAndSave builds the prompt and saves it to a temporary file
// Returns the file path and any error
// The caller is responsible for cleaning up the temporary file
func (pb *PromptBuilder) BuildAndSave(prefix string) (string, error)
```

**用途**: 构建提示词并保存到临时文件（`/tmp/rick-<prefix>-*.md`）
**返回**: 临时文件路径
**清理**: 调用者负责清理（使用 `defer os.Remove()`）

#### SaveToFile
```go
// SaveToFile builds the prompt and saves it to a specific file path
func (pb *PromptBuilder) SaveToFile(filePath string) error
```

**用途**: 构建提示词并保存到指定路径
**返回**: 错误信息
**清理**: 不自动清理（用于持久化保存）

### 2. Plan 阶段

**修改文件**:
- `internal/prompt/plan_prompt.go`
- `internal/cmd/plan.go`

#### plan_prompt.go
新增函数：
```go
func GeneratePlanPromptFile(requirement string, contextMgr *ContextManager,
                            manager *PromptManager) (string, error)
```

返回临时文件路径，调用者负责清理。

#### plan.go
修改调用方式：
```go
// 改造前
planPrompt, err := prompt.GeneratePlanPrompt(requirement, contextMgr, promptMgr)
if err := callClaudeCodeCLI(cfg, planPrompt); err != nil { ... }

// 改造后
planPromptFile, err := prompt.GeneratePlanPromptFile(requirement, contextMgr, promptMgr)
defer os.Remove(planPromptFile)
if err := callClaudeCodeCLI(cfg, planPromptFile); err != nil { ... }
```

修改 `callClaudeCodeCLI` 函数：
```go
// 改造前：接收提示词内容，通过 stdin pipe 传递
func callClaudeCodeCLI(cfg *config.Config, prompt string) error {
    cmd := exec.Command(claudePath, "--permission-mode", "plan")
    stdin, _ := cmd.StdinPipe()
    stdin.Write([]byte(prompt))
    stdin.Close()
    cmd.Run()
}

// 改造后：接收文件路径，直接传递给 Claude
func callClaudeCodeCLI(cfg *config.Config, promptFile string) error {
    cmd := exec.Command(claudePath, promptFile)
    cmd.Stdin = os.Stdin  // 保持交互模式
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr
    cmd.Run()
}
```

### 3. Doing 阶段

**修改文件**:
- `internal/prompt/doing_prompt.go`
- `internal/executor/runner.go`

#### doing_prompt.go
新增函数：
```go
func GenerateDoingPromptFile(task *parser.Task, retryCount int,
                              contextMgr *ContextManager,
                              manager *PromptManager) (string, error)
```

#### runner.go
修改三个关键方法：

##### 1. GenerateDoingPromptFile (重命名)
```go
// 改造前
func (tr *TaskRunner) GenerateDoingPrompt(task *parser.Task,
                                           debugContext string) (string, error)

// 改造后
func (tr *TaskRunner) GenerateDoingPromptFile(task *parser.Task,
                                               debugContext string) (string, error)
```

返回临时文件路径，如果有 debug context，会追加到文件末尾。

##### 2. CallClaudeCodeCLI
```go
// 改造前：接收提示词内容
func (tr *TaskRunner) CallClaudeCodeCLI(promptContent string) (string, error) {
    cmd := exec.Command(claudePath, "--dangerously-skip-permissions")
    stdin, _ := cmd.StdinPipe()
    stdin.Write([]byte(promptContent))
    stdin.Close()
    // ...
}

// 改造后：接收文件路径
func (tr *TaskRunner) CallClaudeCodeCLI(promptFile string) (string, error) {
    cmd := exec.Command(claudePath, "--dangerously-skip-permissions", promptFile)
    // 不再需要 stdin pipe
    // ...
}
```

##### 3. GenerateTestWithAgent
```go
// 改造前
func (tr *TaskRunner) GenerateTestWithAgent(task *parser.Task) (string, error) {
    testPrompt := tr.buildTestGenerationPrompt(task)
    cmd := exec.Command(claudePath, "--dangerously-skip-permissions")
    stdin, _ := cmd.StdinPipe()
    stdin.Write([]byte(testPrompt))
    // ...
}

// 改造后
func (tr *TaskRunner) GenerateTestWithAgent(task *parser.Task) (string, error) {
    testPromptFile, _ := tr.buildTestGenerationPromptFile(task, testScriptPath)
    defer os.Remove(testPromptFile)
    cmd := exec.Command(claudePath, "--dangerously-skip-permissions", testPromptFile)
    // ...
}
```

修改辅助函数：
```go
// 改造前
func (tr *TaskRunner) buildTestGenerationPrompt(task *parser.Task) string

// 改造后
func (tr *TaskRunner) buildTestGenerationPromptFile(task *parser.Task,
                                                     testScriptPath string) (string, error)
```

### 4. Learning 阶段

**文件**: `internal/cmd/learning.go`

**无需修改** - 已经在使用临时文件方式（line 168-177）：

```go
tmpFile, err := os.CreateTemp("", "rick-learning-*.md")
if err != nil { ... }
defer os.Remove(tmpFile.Name())

tmpFile.WriteString(prompt)
tmpFile.Close()

cmd := exec.Command(claudePath, tmpFile.Name())
cmd.Run()
```

## 文件清理策略

### 自动清理（推荐）
使用 `defer os.Remove()` 在函数返回时自动清理：

```go
promptFile, err := builder.BuildAndSave("plan")
if err != nil {
    return err
}
defer os.Remove(promptFile) // 自动清理

// 使用 promptFile
err = callClaude(promptFile)
return err
```

### 手动清理
在某些场景下需要手动清理（如调试时保留文件）：

```go
promptFile, err := builder.BuildAndSave("plan")
if err != nil {
    return err
}

// 调试模式：不清理，打印路径
if debug {
    fmt.Printf("Prompt saved to: %s\n", promptFile)
    return nil
}

// 正常模式：使用后清理
err = callClaude(promptFile)
os.Remove(promptFile)
return err
```

## 临时文件命名规范

| 阶段 | 文件名模式 | 示例 |
|------|-----------|------|
| Plan | `rick-plan-*.md` | `rick-plan-123456.md` |
| Doing | `rick-doing-<task_id>-*.md` | `rick-doing-task1-789012.md` |
| Test Gen | `rick-test-gen-<task_id>-*.md` | `rick-test-gen-task1-345678.md` |
| Learning | `rick-learning-*.md` | `rick-learning-901234.md` |

**位置**: 系统临时目录（`/tmp` on Unix, `%TEMP%` on Windows）

## 调试技巧

### 1. 查看生成的提示词

在调用 Claude 前，打印临时文件路径：

```go
promptFile, _ := builder.BuildAndSave("plan")
fmt.Printf("📝 Prompt saved to: %s\n", promptFile)
defer os.Remove(promptFile)
```

然后可以：
```bash
# 查看提示词内容
cat /tmp/rick-plan-123456.md

# 手动测试
claude /tmp/rick-plan-123456.md
```

### 2. 保留提示词文件

添加 `--keep-prompts` 标志（需要在 config 中实现）：

```go
if !cfg.KeepPrompts {
    defer os.Remove(promptFile)
}
```

### 3. 自定义提示词位置

使用 `SaveToFile` 保存到指定位置：

```go
builder.SaveToFile(".rick/debug/plan-prompt.md")
```

## 兼容性

### 保留旧接口
为了向后兼容，保留了原有的 `GeneratePlanPrompt` 和 `GenerateDoingPrompt` 函数，返回提示词内容字符串。

如果需要使用旧接口：
```go
// 旧接口（返回字符串）
prompt, err := prompt.GeneratePlanPrompt(requirement, contextMgr, promptMgr)

// 新接口（返回文件路径）
promptFile, err := prompt.GeneratePlanPromptFile(requirement, contextMgr, promptMgr)
```

### 测试兼容性
所有现有测试应该继续通过，因为核心逻辑未改变，只是提示词传递方式改变。

## 测试

### 编译测试
```bash
go build -o bin/rick cmd/rick/main.go
```

### 功能测试
```bash
# Plan 阶段
rick plan "测试提示词文件化"

# Doing 阶段
rick doing job_1

# Learning 阶段
rick learning job_1
```

### 验证点
- ✅ 编译无错误
- ✅ Plan 阶段能正常调用 Claude
- ✅ Doing 阶段能正常执行任务
- ✅ Learning 阶段能正常分析
- ✅ 临时文件能正常创建和清理

## 总结

这次改造的核心价值：

1. **简化代码**: 移除了复杂的 stdin pipe 管理逻辑
2. **提升调试体验**: 提示词可见、可修改、可重放
3. **统一接口**: 所有阶段都使用文件方式传递提示词
4. **保持兼容**: 旧接口保留，不影响现有代码

**改造规模**:
- 修改文件: 4 个
- 新增函数: 4 个
- 修改函数: 5 个
- 代码行数: ~200 行

**影响范围**:
- Plan 阶段: ✅ 已改造
- Doing 阶段: ✅ 已改造
- Learning 阶段: ✅ 已使用文件方式（无需改造）
- Test 阶段: ✅ 已改造
