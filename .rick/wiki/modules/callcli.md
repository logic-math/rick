# CallCLI Module（CLI 交互模块）

## 概述
CallCLI Module 负责与 Claude Code CLI 交互，传递提示词并处理输出。

## 模块位置
`internal/callcli/`

## 核心功能

### 1. 调用 Claude Code CLI
**职责**: 调用 Claude Code CLI 执行任务

**核心函数**:
```go
// CallClaudeCLI 调用 Claude Code CLI
func CallClaudeCLI(prompt string) error {
    // 1. 获取 Claude Code CLI 路径
    cliPath := config.Get().ClaudeCodePath
    if cliPath == "" {
        cliPath = "claude" // 默认使用 PATH 中的 claude
    }

    // 2. 构建命令
    cmd := exec.Command(cliPath, "code", "--prompt", prompt)

    // 3. 设置标准输入/输出
    cmd.Stdin = os.Stdin
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr

    // 4. 执行命令
    err := cmd.Run()
    if err != nil {
        return fmt.Errorf("claude code failed: %w", err)
    }

    return nil
}
```

### 2. 提示词传递
**方式1**: 通过命令行参数传递
```go
cmd := exec.Command("claude", "code", "--prompt", prompt)
```

**方式2**: 通过临时文件传递（大提示词）
```go
func CallClaudeCLIWithFile(prompt string) error {
    // 1. 创建临时文件
    tmpFile, err := os.CreateTemp("", "rick-prompt-*.md")
    if err != nil {
        return err
    }
    defer os.Remove(tmpFile.Name())

    // 2. 写入提示词
    _, err = tmpFile.WriteString(prompt)
    if err != nil {
        return err
    }
    tmpFile.Close()

    // 3. 调用 Claude Code CLI
    cmd := exec.Command("claude", "code", "--prompt-file", tmpFile.Name())
    cmd.Stdin = os.Stdin
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr

    err = cmd.Run()
    if err != nil {
        return fmt.Errorf("claude code failed: %w", err)
    }

    return nil
}
```

### 3. 输出处理
**职责**: 处理 Claude Code CLI 的输出

**实时输出**:
```go
func CallClaudeCLIWithRealtime(prompt string) error {
    cmd := exec.Command("claude", "code", "--prompt", prompt)

    // 创建管道
    stdout, _ := cmd.StdoutPipe()
    stderr, _ := cmd.StderrPipe()

    // 启动命令
    cmd.Start()

    // 实时读取输出
    go io.Copy(os.Stdout, stdout)
    go io.Copy(os.Stderr, stderr)

    // 等待完成
    err := cmd.Wait()
    if err != nil {
        return fmt.Errorf("claude code failed: %w", err)
    }

    return nil
}
```

**捕获输出**:
```go
func CallClaudeCLIWithCapture(prompt string) (string, error) {
    cmd := exec.Command("claude", "code", "--prompt", prompt)

    // 捕获输出
    output, err := cmd.CombinedOutput()
    if err != nil {
        return "", fmt.Errorf("claude code failed: %s", output)
    }

    return string(output), nil
}
```

## 使用示例

### 示例1: Plan 阶段调用
```go
func executePlanStage(objective string) error {
    // 1. 构建提示词
    context := PromptContext{
        ProjectName: "Rick CLI",
        Objectives:  objective,
    }
    prompt, _ := prompt.BuildPlanPrompt(context)

    // 2. 调用 Claude Code CLI
    err := callcli.CallClaudeCLI(prompt)
    if err != nil {
        return err
    }

    return nil
}
```

### 示例2: Doing 阶段调用
```go
func executeDoingStage(task *Task) error {
    // 1. 构建提示词
    context := PromptContext{
        TaskID:      task.TaskID,
        TaskName:    task.TaskName,
        Objectives:  task.Objectives,
        KeyResults:  task.KeyResults,
        TestMethods: task.TestMethods,
    }
    prompt, _ := prompt.BuildDoingPrompt(context)

    // 2. 调用 Claude Code CLI
    err := callcli.CallClaudeCLI(prompt)
    if err != nil {
        return err
    }

    return nil
}
```

### 示例3: Learning 阶段调用
```go
func executeLearningStage(jobID string) error {
    // 1. 构建提示词
    context := PromptContext{
        JobID:     jobID,
        TaskCount: getTaskCount(jobID),
        DebugInfo: readAllDebugInfo(jobID),
    }
    prompt, _ := prompt.BuildLearningPrompt(context)

    // 2. 调用 Claude Code CLI
    err := callcli.CallClaudeCLI(prompt)
    if err != nil {
        return err
    }

    return nil
}
```

## 配置

### Claude Code CLI 路径配置
```json
{
  "claude_code_path": "/usr/local/bin/claude"
}
```

**查找顺序**:
1. 配置文件中的 `claude_code_path`
2. 环境变量 `CLAUDE_CODE_PATH`
3. PATH 中的 `claude` 命令

### 环境变量
```bash
export CLAUDE_CODE_PATH=/usr/local/bin/claude
```

## 错误处理

### 常见错误
1. **Claude Code CLI 未安装**: 提示用户安装
2. **提示词过长**: 使用临时文件传递
3. **执行失败**: 记录详细错误信息
4. **超时**: 添加超时控制

### 错误处理示例
```go
func CallClaudeCLI(prompt string) error {
    // 1. 检查 Claude Code CLI 是否可用
    cliPath, err := findClaudeCodeCLI()
    if err != nil {
        return fmt.Errorf("Claude Code CLI not found: %w", err)
    }

    // 2. 检查提示词长度
    if len(prompt) > 10000 {
        // 使用临时文件传递
        return CallClaudeCLIWithFile(prompt)
    }

    // 3. 执行命令
    cmd := exec.Command(cliPath, "code", "--prompt", prompt)
    cmd.Stdin = os.Stdin
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr

    err = cmd.Run()
    if err != nil {
        return fmt.Errorf("claude code failed: %w", err)
    }

    return nil
}

func findClaudeCodeCLI() (string, error) {
    // 1. 检查配置文件
    cliPath := config.Get().ClaudeCodePath
    if cliPath != "" {
        if _, err := os.Stat(cliPath); err == nil {
            return cliPath, nil
        }
    }

    // 2. 检查环境变量
    cliPath = os.Getenv("CLAUDE_CODE_PATH")
    if cliPath != "" {
        if _, err := os.Stat(cliPath); err == nil {
            return cliPath, nil
        }
    }

    // 3. 检查 PATH
    cliPath, err := exec.LookPath("claude")
    if err == nil {
        return cliPath, nil
    }

    return "", errors.New("Claude Code CLI not found")
}
```

## 测试

### 单元测试
```bash
go test ./internal/callcli/
```

### 测试用例
```go
func TestCallClaudeCLI(t *testing.T) {
    // Mock Claude Code CLI
    oldPath := config.Get().ClaudeCodePath
    config.Get().ClaudeCodePath = "/bin/echo"
    defer func() {
        config.Get().ClaudeCodePath = oldPath
    }()

    prompt := "测试提示词"
    err := CallClaudeCLI(prompt)
    if err != nil {
        t.Fatal(err)
    }
}

func TestFindClaudeCodeCLI(t *testing.T) {
    cliPath, err := findClaudeCodeCLI()
    if err != nil {
        t.Skip("Claude Code CLI not installed")
    }

    if cliPath == "" {
        t.Error("cliPath should not be empty")
    }
}
```

## 最佳实践

1. **提示词优化**: 控制提示词长度，避免过长
2. **错误处理**: 详细记录错误信息，便于调试
3. **超时控制**: 添加超时机制，避免长时间阻塞
4. **日志记录**: 记录每次调用的提示词和输出

## 常见问题

### Q1: 如何调试提示词？
**A**: 在调用前将提示词输出到文件：
```go
os.WriteFile("prompt.md", []byte(prompt), 0644)
```

### Q2: 如何处理超时？
**A**: 使用 `context.WithTimeout`:
```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
defer cancel()

cmd := exec.CommandContext(ctx, "claude", "code", "--prompt", prompt)
```

### Q3: 如何支持非交互式模式？
**A**: 使用 `--non-interactive` 参数（如果 Claude Code CLI 支持）。

## 未来优化

1. **超时控制**: 添加可配置的超时时间
2. **重试机制**: 网络错误时自动重试
3. **输出解析**: 解析 Claude Code CLI 的结构化输出
4. **进度显示**: 显示执行进度（如果 Claude Code CLI 支持）

---

*最后更新: 2026-03-14*
