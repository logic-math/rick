# Parser Module（内容解析模块）

## 概述
Parser Module 负责解析各种 Markdown 格式的文件，包括 task.md、debug.md、OKR/SPEC 文档等。

## 模块位置
`internal/parser/`

## 核心功能

### 1. Markdown 解析
**依赖**: Goldmark 库

**职责**:
- 解析 Markdown 文件为 AST（抽象语法树）
- 提取标题、段落、列表等元素
- 支持 GFM（GitHub Flavored Markdown）

### 2. task.md 解析
**格式**:
```markdown
# 依赖关系
task1, task2

# 任务名称
任务标题

# 任务目标
具体目标描述

# 关键结果
1. 结果1
2. 结果2

# 测试方法
1. 测试步骤1
2. 测试步骤2
```

**数据结构**:
```go
type Task struct {
    TaskID       string   `json:"task_id"`
    TaskName     string   `json:"task_name"`
    Dependencies []string `json:"dep"`
    Objectives   string   `json:"objectives"`
    KeyResults   []string `json:"key_results"`
    TestMethods  []string `json:"test_methods"`
    StateInfo    struct {
        Status string `json:"status"` // pending, doing, done, failed
    } `json:"state_info"`
}
```

**核心函数**:
```go
// ParseTaskMD 解析 task.md 文件
func ParseTaskMD(filePath string) (*Task, error)
```

### 3. debug.md 解析
**格式**:
```markdown
# debug1: 问题描述

**问题描述**
...

**复现步骤**
1. ...
2. ...

**可能原因**
...

**解决状态**
已解决/未解决

**解决方法**
...
```

**数据结构**:
```go
type DebugEntry struct {
    ID          string
    Title       string
    Description string
    Steps       []string
    Cause       string
    Status      string // 已解决/未解决
    Solution    string
}
```

**核心函数**:
```go
// ParseDebugMD 解析 debug.md 文件
func ParseDebugMD(filePath string) ([]DebugEntry, error)
```

### 4. OKR/SPEC 解析
**职责**:
- 解析 OKR.md 文件（目标和关键结果）
- 解析 SPEC.md 文件（项目规范）
- 提取项目背景信息

**核心函数**:
```go
// ParseOKR 解析 OKR.md 文件
func ParseOKR(filePath string) (*OKR, error)

// ParseSPEC 解析 SPEC.md 文件
func ParseSPEC(filePath string) (*SPEC, error)
```

## 实现细节

### Markdown 解析器
```go
import (
    "github.com/yuin/goldmark"
    "github.com/yuin/goldmark/extension"
)

// ParseMarkdown 解析 Markdown 文件
func ParseMarkdown(content []byte) (ast.Node, error) {
    md := goldmark.New(
        goldmark.WithExtensions(extension.GFM),
    )

    parser := md.Parser()
    reader := text.NewReader(content)
    return parser.Parse(reader), nil
}
```

### 标题提取
```go
// ExtractHeadings 提取所有标题
func ExtractHeadings(node ast.Node) map[string]string {
    headings := make(map[string]string)

    ast.Walk(node, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
        if !entering {
            return ast.WalkContinue, nil
        }

        if heading, ok := n.(*ast.Heading); ok {
            // 提取标题文本
            text := extractText(heading)
            headings[text] = ""
        }

        return ast.WalkContinue, nil
    })

    return headings
}
```

### 列表提取
```go
// ExtractList 提取列表项
func ExtractList(node ast.Node) []string {
    var items []string

    ast.Walk(node, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
        if !entering {
            return ast.WalkContinue, nil
        }

        if listItem, ok := n.(*ast.ListItem); ok {
            text := extractText(listItem)
            items = append(items, text)
        }

        return ast.WalkContinue, nil
    })

    return items
}
```

## 测试

### 单元测试
```bash
go test ./internal/parser/
```

### 测试用例
```go
func TestParseTaskMD(t *testing.T) {
    task, err := ParseTaskMD("testdata/task1.md")
    if err != nil {
        t.Fatal(err)
    }

    if task.TaskID != "task1" {
        t.Errorf("expected task_id=task1, got %s", task.TaskID)
    }

    if len(task.Dependencies) != 2 {
        t.Errorf("expected 2 dependencies, got %d", len(task.Dependencies))
    }
}
```

## 最佳实践

1. **错误处理**: 解析失败时返回详细错误信息
2. **格式验证**: 验证 Markdown 格式是否符合规范
3. **空值处理**: 处理可选字段为空的情况
4. **编码支持**: 支持 UTF-8 编码

## 常见问题

### Q1: 如何处理格式不规范的 task.md？
**A**: 返回错误，提示用户修正格式。

### Q2: 如何支持自定义字段？
**A**: 扩展 Task 结构体，添加 `ExtraFields map[string]interface{}`。

### Q3: 如何提高解析性能？
**A**: 使用缓存机制，避免重复解析同一文件。

## 未来优化

1. **缓存机制**: 缓存已解析的文件
2. **增量解析**: 仅解析修改的部分
3. **格式验证**: 更严格的格式验证
4. **自定义扩展**: 支持自定义 Markdown 扩展

---

*最后更新: 2026-03-14*
