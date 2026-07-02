# Go 编码规范与模式

## 代码风格

- Go 标准格式（gofmt），函数命名 camelCase，导出函数 PascalCase
- 每个任务完成后独立 commit，commit message 包含 task ID

## Cobra flag 定义规范

**全局 flag**（跨命令共享）：在 `root.go` 用 `rootCmd.PersistentFlags()` 定义，通过 `GetXxx()` 函数统一暴露。

```go
// root.go
func init() {
    rootCmd.PersistentFlags().StringP("job", "j", "", "Job ID")
}

func GetJobID() string {
    v, _ := rootCmd.PersistentFlags().GetString("job")
    return v
}
```

**命令级 flag**：在对应命令文件用 `cmd.Flags()` 定义。

**禁止**：在子命令文件中重复定义已在 root.go 定义的全局 flag（会导致 flag redefined 错误）。

## Go variadic 改造模式

当需要让现有**必传参数**变为可选时，使用 variadic（`...T`）而非新增无参构造函数：

```go
// 改造前：必传
func NewPromptManager(skillsDir string) *Manager { ... }

// 改造后：variadic（调用方无需修改）
func NewPromptManager(skillsDir ...string) *Manager {
    dir := ""
    if len(skillsDir) > 0 { dir = skillsDir[0] }
    ...
}
```

好处：保持接口唯一性，调用方无需修改，向后兼容。

## embed.FS 目录嵌入

```go
// 目录嵌入：必须绑定 embed.FS 类型
//go:embed templates
var templatesFS embed.FS

// 单文件嵌入：可绑定 string
//go:embed templates/doing.md
var doingTemplate string

// 注意：import "embed"（不是 _ "embed"）才能使用 embed.FS
import "embed"
```

**嵌入文件修改后必须重新 build**：`./scripts/build.sh`，改模板后不 build 不生效。

## 包内函数共享

同一 Go 包内的函数可在多个文件中直接调用，不需要重新声明或导出：

```go
// plan.go 中定义
func callClaudeCodeCLI(cfg Config, promptFile string) error { ... }

// human_loop.go 中直接复用（同一 internal/cmd 包）
func runHumanLoop(topic string) error {
    return callClaudeCodeCLI(cfg, promptFile)  // ✅ 直接调用
}
```

## 接口签名协商（并行 task 中）

并行 task 中若涉及接口定义和实现：
- 接口 task **先完成**后，实现 task 才开始
- 或在 plan 阶段明确接口签名

**约定**：接口签名**不含 `context.Context`**，避免标准库强制依赖影响可测试性：

```go
// ✅ 推荐
type AgentExecutor interface {
    Execute(task Task) (Result, error)
}

// ❌ 避免（context.Context 引入标准库依赖链）
type AgentExecutor interface {
    Execute(ctx context.Context, task Task) (Result, error)
}
```

## 配置污染防护（测试）

全局 `~/.rick/config.json` 的 `max_retries` 高值会导致测试超时（retry sleep 累计）。测试开头注入临时 HOME：

```go
func TestXxx(t *testing.T) {
    dir := t.TempDir()
    t.Setenv("HOME", dir)
    _ = os.MkdirAll(filepath.Join(dir, ".rick"), 0755)
    _ = os.WriteFile(filepath.Join(dir, ".rick", "config.json"),
        []byte(`{"max_retries":2}`), 0644)
    // ...
}
```

识别信号：本地测试卡 > 30s，stack trace 卡在 `retry.go time.Sleep`，CI 正常。
