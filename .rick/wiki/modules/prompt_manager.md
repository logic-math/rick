# Prompt Manager Module（提示词管理模块）⭐

## 概述
Prompt Manager Module 是 Rick CLI 的核心创新，负责提示词模板管理、提示词构建和多阶段提示词生成。

## 模块位置
`internal/prompt/`

## 核心功能

### 1. 提示词模板管理
**职责**: 管理不同阶段的提示词模板

**模板目录**:
```
internal/prompt/templates/
├── plan.md       # Plan 阶段提示词模板
├── doing.md      # Doing 阶段提示词模板
└── learning.md   # Learning 阶段提示词模板
```

**核心结构**:
```go
type PromptManager struct {
    TemplateDir string
}

// LoadTemplate 加载模板
func (pm *PromptManager) LoadTemplate(stage string) (string, error)
```

### 2. 提示词构建
**职责**: 根据模板和上下文构建最终提示词

**核心结构**:
```go
type PromptBuilder struct {
    Stage    string
    Context  PromptContext
    Template string
}

type PromptContext struct {
    // 任务信息
    TaskID       string
    TaskName     string
    RetryCount   int
    Objectives   string
    KeyResults   []string
    TestMethods  []string

    // 项目背景
    ProjectName  string
    ProjectDesc  string
    ProjectSPEC  string
    ProjectArch  string

    // 执行上下文
    CompletedTasks []string
    Dependencies   []string
    DebugInfo      string
}

// Build 构建提示词
func (pb *PromptBuilder) Build() (string, error)
```

### 3. 多阶段提示词生成
**支持阶段**:
- **Plan**: 规划任务，生成 tasks/*.md
- **Doing**: 执行任务，完成编码工作
- **Learning**: 知识沉淀，总结经验

**核心函数**:
```go
// BuildPlanPrompt 构建 Plan 阶段提示词
func BuildPlanPrompt(context PromptContext) (string, error)

// BuildDoingPrompt 构建 Doing 阶段提示词
func BuildDoingPrompt(context PromptContext) (string, error)

// BuildLearningPrompt 构建 Learning 阶段提示词
func BuildLearningPrompt(context PromptContext) (string, error)
```

## 模板示例

### Plan 阶段模板（plan.md）
```markdown
# Rick 项目规划阶段提示词

你是一个资深的软件架构师。你的任务是规划项目任务，将大目标分解为可执行的小任务。

## 项目信息
**项目名称**: {{project_name}}
**项目描述**: {{project_desc}}

## 任务目标
{{objectives}}

## 规划要求
1. 分析需求，理解项目背景
2. 将大目标分解为小任务（每个任务 1-3 天完成）
3. 确定任务依赖关系（构建 DAG）
4. 为每个任务定义关键结果和测试方法

## 输出格式
为每个任务创建 task.md 文件，格式如下：
...
```

### Doing 阶段模板（doing.md）
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
{{#each key_results}}
{{@index}}. {{this}}
{{/each}}

### 测试方法
{{#each test_methods}}
{{@index}}. {{this}}
{{/each}}

## 项目背景
{{project_context}}

## 执行上下文
### 已完成的任务
{{#each completed_tasks}}
- {{this}}
{{/each}}

### 任务依赖
{{#each dependencies}}
- {{this}}
{{/each}}

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

### Learning 阶段模板（learning.md）
```markdown
# Rick 项目学习阶段提示词

你是一个知识管理专家。你的任务是总结项目经验，沉淀知识。

## 任务信息
**Job ID**: {{job_id}}
**任务数量**: {{task_count}}

## 学习要求
1. 总结项目目标达成情况
2. 提取可复用的设计模式
3. 记录最佳实践
4. 总结经验教训

## 输出格式
...
```

## 上下文注入机制

### 动态数据来源
```go
func BuildPromptContext(task *Task, jobDir string) (PromptContext, error) {
    context := PromptContext{
        // 1. 任务信息（从 task.md 读取）
        TaskID:      task.TaskID,
        TaskName:    task.TaskName,
        RetryCount:  task.RetryCount,
        Objectives:  task.Objectives,
        KeyResults:  task.KeyResults,
        TestMethods: task.TestMethods,

        // 2. 项目背景（从 OKR.md、SPEC.md 读取）
        ProjectName: readProjectName(),
        ProjectDesc: readProjectDesc(),
        ProjectSPEC: readSPEC(),
        ProjectArch: readArchitecture(),

        // 3. 执行上下文
        CompletedTasks: getCompletedTasks(jobDir),
        Dependencies:   task.Dependencies,
        DebugInfo:      readDebugInfo(jobDir, task.TaskID),
    }

    return context, nil
}
```

### 模板渲染
```go
func RenderTemplate(template string, context PromptContext) (string, error) {
    // 使用 text/template 或 Handlebars 渲染
    tmpl, err := template.New("prompt").Parse(template)
    if err != nil {
        return "", err
    }

    var buf bytes.Buffer
    err = tmpl.Execute(&buf, context)
    if err != nil {
        return "", err
    }

    return buf.String(), nil
}
```

## 实现细节

### PromptManager 实现
```go
type PromptManager struct {
    TemplateDir string
}

func NewPromptManager(templateDir string) *PromptManager {
    return &PromptManager{
        TemplateDir: templateDir,
    }
}

func (pm *PromptManager) LoadTemplate(stage string) (string, error) {
    templatePath := filepath.Join(pm.TemplateDir, stage+".md")
    content, err := os.ReadFile(templatePath)
    if err != nil {
        return "", err
    }
    return string(content), nil
}
```

### PromptBuilder 实现
```go
type PromptBuilder struct {
    Stage    string
    Context  PromptContext
    Template string
}

func (pb *PromptBuilder) Build() (string, error) {
    // 渲染模板
    prompt, err := RenderTemplate(pb.Template, pb.Context)
    if err != nil {
        return "", err
    }

    return prompt, nil
}
```

## 测试

### 单元测试
```bash
go test ./internal/prompt/
```

### 测试用例
```go
func TestLoadTemplate(t *testing.T) {
    pm := NewPromptManager("templates")
    template, err := pm.LoadTemplate("doing")
    if err != nil {
        t.Fatal(err)
    }

    if !strings.Contains(template, "任务信息") {
        t.Error("template should contain '任务信息'")
    }
}

func TestBuildPrompt(t *testing.T) {
    context := PromptContext{
        TaskID:   "task1",
        TaskName: "测试任务",
    }

    pb := &PromptBuilder{
        Stage:    "doing",
        Context:  context,
        Template: loadTemplate("doing"),
    }

    prompt, err := pb.Build()
    if err != nil {
        t.Fatal(err)
    }

    if !strings.Contains(prompt, "task1") {
        t.Error("prompt should contain task1")
    }
}
```

## 最佳实践

1. **模板复用**: 提取公共部分为子模板
2. **上下文完整**: 确保所有必要信息都注入上下文
3. **错误处理**: 处理模板加载和渲染失败
4. **版本控制**: 模板文件纳入版本控制

## 常见问题

### Q1: 如何自定义提示词模板？
**A**: 直接修改 `internal/prompt/templates/*.md` 文件。

### Q2: 如何添加新的上下文字段？
**A**: 扩展 `PromptContext` 结构体，更新模板。

### Q3: 如何支持多语言提示词？
**A**: 添加语言参数，加载不同的模板文件（如 `doing_en.md`, `doing_zh.md`）。

## 未来优化

1. **模板热重载**: 支持运行时重载模板
2. **模板验证**: 验证模板语法和必需字段
3. **多语言支持**: 支持英文、中文等多语言提示词
4. **模板市场**: 支持从社区下载和分享模板

---

*最后更新: 2026-03-14*
