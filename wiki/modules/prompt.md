# prompt - 提示词管理模块

## 模块职责

`prompt` 模块是 Rick CLI 的核心创新之一，负责管理和生成各阶段的 AI 提示词。该模块提供了模板化的提示词管理机制，支持变量替换、上下文注入和动态生成，确保 Claude Code 能够接收到清晰、完整、结构化的任务指令。

**核心职责**：
- 管理提示词模板（plan.md, doing.md, learning.md）
- 构建和生成各阶段的提示词
- 支持变量替换和上下文注入
- 提供提示词的保存和临时文件管理
- 集成项目上下文（OKR, SPEC, Wiki, Skills）

## 核心类型

### PromptTemplate
提示词模板，定义模板的结构和变量。

```go
type PromptTemplate struct {
    Name      string   // 模板名称（plan, doing, learning）
    Content   string   // 模板内容
    Variables []string // 需要替换的变量列表
}
```

### PromptBuilder
提示词构建器，负责变量替换和上下文注入。

```go
type PromptBuilder struct {
    Template  *PromptTemplate
    Variables map[string]string         // 变量映射
    Context   map[string]interface{}    // 上下文数据
}
```

### PromptManager
提示词管理器，管理所有模板的加载和访问。

```go
type PromptManager struct {
    templates map[string]*PromptTemplate
    rickDir   string
}
```

## 关键函数

### NewPromptManager(rickDir string) (*PromptManager, error)
创建提示词管理器实例。

**参数**：
- `rickDir`: .rick 目录路径

**示例**：
```go
pm, err := prompt.NewPromptManager("/path/to/.rick")
if err != nil {
    log.Fatal(err)
}
```

### LoadTemplate(name string) (*PromptTemplate, error)
加载指定名称的提示词模板。

**支持的模板**：
- `plan`: 任务规划阶段
- `doing`: 任务执行阶段
- `learning`: 知识积累阶段

**示例**：
```go
template, err := pm.LoadTemplate("doing")
if err != nil {
    log.Fatal(err)
}
```

### NewPromptBuilder(template *PromptTemplate) *PromptBuilder
创建提示词构建器。

**示例**：
```go
builder := prompt.NewPromptBuilder(template)
```

### SetVariable(key, value string) *PromptBuilder
设置模板变量。

**支持链式调用**。

**示例**：
```go
builder.SetVariable("task_id", "task1").
        SetVariable("task_name", "实现功能").
        SetVariable("retry_count", "0")
```

### SetContext(key string, value interface{}) *PromptBuilder
设置上下文数据。

**支持任意类型的值**。

**示例**：
```go
builder.SetContext("okr_content", okrText).
        SetContext("spec_content", specText).
        SetContext("completed_tasks", []string{"task1", "task2"})
```

### Build() (string, error)
构建最终的提示词内容。

**流程**：
1. 替换所有 `{{variable}}` 变量
2. 注入上下文数据
3. 返回完整的提示词

**示例**：
```go
content, err := builder.Build()
if err != nil {
    log.Fatal(err)
}
fmt.Println(content)
```

### BuildAndSave(prefix string) (string, error)
构建提示词并保存到临时文件。

**参数**：
- `prefix`: 临时文件名前缀（如 "doing-task1"）

**返回**：
- `string`: 临时文件路径
- `error`: 错误信息

**示例**：
```go
tmpFile, err := builder.BuildAndSave("doing-task1")
if err != nil {
    log.Fatal(err)
}
defer os.Remove(tmpFile)

// 使用临时文件调用 Claude Code
callClaudeCode(tmpFile)
```

### SaveToFile(filePath string) error
构建提示词并保存到指定文件。

**示例**：
```go
err := builder.SaveToFile("prompt.md")
if err != nil {
    log.Fatal(err)
}
```

### GetMissingVariables() []string
获取未设置的变量列表。

**用于验证提示词是否完整**。

**示例**：
```go
missing := builder.GetMissingVariables()
if len(missing) > 0 {
    log.Fatalf("Missing variables: %v", missing)
}
```

## 类图

```mermaid
classDiagram
    class PromptManager {
        -map~string,*PromptTemplate~ templates
        -string rickDir
        +NewPromptManager(rickDir) (*PromptManager, error)
        +LoadTemplate(name) (*PromptTemplate, error)
        +GetTemplate(name) *PromptTemplate
        +ListTemplates() []string
    }

    class PromptTemplate {
        +string Name
        +string Content
        +[]string Variables
        +ExtractVariables() []string
    }

    class PromptBuilder {
        -PromptTemplate Template
        -map~string,string~ Variables
        -map~string,interface{}~ Context
        +NewPromptBuilder(template) *PromptBuilder
        +SetVariable(key, value) *PromptBuilder
        +SetContext(key, value) *PromptBuilder
        +Build() (string, error)
        +BuildAndSave(prefix) (string, error)
        +SaveToFile(path) error
        +GetMissingVariables() []string
    }

    class DoingPromptBuilder {
        +BuildDoingPrompt(task, config) (string, error)
        -loadTaskContext(task) (string, error)
        -loadDebugContext(debugFile) (string, error)
        -loadProjectContext() (string, error)
    }

    class PlanPromptBuilder {
        +BuildPlanPrompt(description, config) (string, error)
        -loadProjectContext() (string, error)
    }

    class LearningPromptBuilder {
        +BuildLearningPrompt(jobID, config) (string, error)
        -loadExecutionResults(jobPath) (string, error)
        -loadTaskFiles(planDir) ([]string, error)
    }

    PromptManager --> PromptTemplate : manages
    PromptBuilder --> PromptTemplate : uses
    DoingPromptBuilder --> PromptBuilder : extends
    PlanPromptBuilder --> PromptBuilder : extends
    LearningPromptBuilder --> PromptBuilder : extends
```

## 使用示例

### 示例 1: 构建 Doing 阶段提示词
```go
package main

import (
    "fmt"
    "log"
    "github.com/sunquan/rick/internal/prompt"
    "github.com/sunquan/rick/internal/parser"
)

func main() {
    // 1. 创建提示词管理器
    pm, err := prompt.NewPromptManager(".rick")
    if err != nil {
        log.Fatal(err)
    }

    // 2. 加载 doing 模板
    template, err := pm.LoadTemplate("doing")
    if err != nil {
        log.Fatal(err)
    }

    // 3. 创建构建器
    builder := prompt.NewPromptBuilder(template)

    // 4. 设置变量
    task := &parser.Task{
        ID:   "task1",
        Name: "实现用户认证",
        Goal: "完成用户登录和注册功能",
    }

    builder.SetVariable("task_id", task.ID).
            SetVariable("task_name", task.Name).
            SetVariable("task_goal", task.Goal).
            SetVariable("retry_count", "0")

    // 5. 设置上下文
    builder.SetContext("okr_content", loadOKR()).
            SetContext("spec_content", loadSPEC()).
            SetContext("completed_tasks", []string{})

    // 6. 构建并保存
    tmpFile, err := builder.BuildAndSave("doing-task1")
    if err != nil {
        log.Fatal(err)
    }
    defer os.Remove(tmpFile)

    fmt.Println("Prompt saved to:", tmpFile)
}
```

### 示例 2: 使用专用构建器
```go
func buildDoingPrompt(task *parser.Task, retryCount int) (string, error) {
    // 使用专用的 Doing 提示词构建器
    builder := prompt.NewDoingPromptBuilder(".rick", "job_1")

    // 构建提示词
    promptFile, err := builder.BuildDoingPrompt(task, retryCount)
    if err != nil {
        return "", fmt.Errorf("failed to build prompt: %w", err)
    }

    return promptFile, nil
}
```

### 示例 3: 验证变量完整性
```go
func validatePrompt(builder *prompt.PromptBuilder) error {
    missing := builder.GetMissingVariables()
    if len(missing) > 0 {
        return fmt.Errorf("missing required variables: %v", missing)
    }

    // 构建提示词
    content, err := builder.Build()
    if err != nil {
        return fmt.Errorf("failed to build prompt: %w", err)
    }

    // 检查是否还有未替换的变量
    if strings.Contains(content, "{{") {
        return fmt.Errorf("prompt contains unreplaced variables")
    }

    return nil
}
```

## 提示词模板格式

### doing.md 模板示例
```markdown
# Rick 项目执行阶段提示词

你是一个资深的软件工程师。你的任务是执行规划好的任务，完成具体的编码工作。

## 任务信息

**任务 ID**: {{task_id}}
**任务名称**: {{task_name}}
**重试次数**: {{retry_count}}

### 任务目标
{{task_goal}}

### 关键结果
{{key_results}}

### 测试方法
{{test_method}}

## 项目背景

**项目名称**: {{project_name}}
**项目描述**: {{project_description}}

### 项目 OKR
{{okr_content}}

### 项目 SPEC
{{spec_content}}

## 执行上下文

### 已完成的任务
{{completed_tasks}}

### 任务依赖
该任务依赖以下任务的完成：
{{dependencies}}

{{#if retry_count > 0}}
### 前次执行的问题记录
{{debug_content}}
{{/if}}

## 执行要求

1. **理解需求**: 仔细阅读任务目标和关键结果
2. **设计方案**: 根据项目架构和现有代码，设计实现方案
3. **编写代码**: 实现所有必要的功能
4. **测试验证**: 按照测试方法验证功能的正确性
5. **提交代码**: 使用 git 提交代码，提交信息应该清晰明确

## 重要提示

1. 如果遇到问题，请详细记录问题现象、复现步骤、可能原因和解决方案
2. 确保所有测试都通过
3. 代码应该能够在生产环境中正确运行
```

### 变量说明
- `{{task_id}}`: 任务 ID
- `{{task_name}}`: 任务名称
- `{{task_goal}}`: 任务目标
- `{{retry_count}}`: 重试次数
- `{{okr_content}}`: OKR 内容
- `{{spec_content}}`: SPEC 内容
- `{{completed_tasks}}`: 已完成任务列表
- `{{dependencies}}`: 任务依赖
- `{{debug_content}}`: 调试信息（仅重试时）

## 上下文加载策略

### Plan 阶段
```go
- 项目描述（用户输入）
- OKR 内容（.rick/OKR.md）
- SPEC 内容（.rick/SPEC.md）
- Wiki 索引（.rick/wiki/index.md）
- Skills 索引（.rick/skills/index.md）
```

### Doing 阶段
```go
- 任务信息（task.md）
- 项目 OKR 和 SPEC
- 已完成任务列表
- 任务依赖信息
- Debug 信息（如果是重试）
- Wiki 相关文档
```

### Learning 阶段
```go
- Job 执行结果（tasks.json）
- 所有任务文件（task*.md）
- 执行日志（execution.log）
- Debug 记录（debug.md）
- Git 提交历史
```

## 错误处理

### 常见错误及解决方案

1. **模板文件不存在**
   ```
   Error: template file not found: doing.md
   Solution: 确保 .rick/templates/ 目录包含所有模板
   ```

2. **变量未设置**
   ```
   Error: missing required variable: task_id
   Solution: 调用 SetVariable 设置所有必需变量
   ```

3. **无法创建临时文件**
   ```
   Error: failed to create temporary file
   Solution: 检查 /tmp 目录的写权限
   ```

## 设计原则

1. **模板化**：所有提示词使用模板，便于维护和修改
2. **上下文丰富**：提供完整的项目上下文，确保 AI 理解任务
3. **结构化**：使用清晰的章节结构，便于 AI 解析
4. **可扩展**：易于添加新的变量和上下文
5. **类型安全**：使用 Go 类型系统确保数据正确性

## 测试覆盖

### builder_test.go
```go
func TestNewPromptBuilder(t *testing.T)
func TestSetVariable(t *testing.T)
func TestSetContext(t *testing.T)
func TestBuild(t *testing.T)
func TestBuildAndSave(t *testing.T)
func TestGetMissingVariables(t *testing.T)
```

### doing_prompt_test.go
```go
func TestBuildDoingPrompt(t *testing.T)
func TestLoadTaskContext(t *testing.T)
func TestLoadDebugContext(t *testing.T)
```

## 扩展点

### 添加新模板
```go
// 1. 创建模板文件
// .rick/templates/review.md

// 2. 在 PromptManager 中注册
func (pm *PromptManager) LoadReviewTemplate() (*PromptTemplate, error) {
    return pm.LoadTemplate("review")
}

// 3. 创建专用构建器
type ReviewPromptBuilder struct {
    builder *PromptBuilder
}

func (rpb *ReviewPromptBuilder) BuildReviewPrompt(code string) (string, error) {
    rpb.builder.SetVariable("code_content", code)
    return rpb.builder.BuildAndSave("review")
}
```

### 自定义变量替换
```go
func (pb *PromptBuilder) SetVariableWithFormat(key, value, format string) *PromptBuilder {
    formatted := fmt.Sprintf(format, value)
    pb.Variables[key] = formatted
    return pb
}

// 使用
builder.SetVariableWithFormat("date", time.Now().String(), "Date: %s")
```

## 与其他模块的交互

### cmd 模块
```go
// cmd 使用 prompt 生成提示词文件
promptFile, err := buildDoingPrompt(task, retryCount)
callClaudeCode(promptFile)
```

### executor 模块
```go
// executor 在执行任务时生成提示词
builder := prompt.NewDoingPromptBuilder(rickDir, jobID)
promptFile, _ := builder.BuildDoingPrompt(task, attempt)
```

### parser 模块
```go
// prompt 使用 parser 解析任务信息
task, _ := parser.ParseTask(content)
builder.SetVariable("task_name", task.Name)
builder.SetVariable("task_goal", task.Goal)
```
