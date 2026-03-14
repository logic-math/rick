# Markdown 解析 - 实现细节

## 核心数据结构

### MarkdownDocument
```go
type MarkdownDocument struct {
    AST    ast.Node  // Goldmark AST 根节点
    Source []byte    // 原始 Markdown 源码
}
```

## Goldmark 解析流程

### 1. 解析 Markdown
```go
func ParseMarkdownWithSource(content string) (*MarkdownDocument, error) {
    md := goldmark.New()
    source := []byte(content)
    reader := text.NewReader(source)
    node := md.Parser().Parse(reader)

    return &MarkdownDocument{
        AST:    node,
        Source: source,
    }, nil
}
```

### 2. 提取标题
```go
func ExtractHeadingWithSource(node ast.Node, level int, source []byte) []string {
    var headings []string
    walkNode(node, func(n ast.Node, entering bool) ast.WalkStatus {
        if !entering {
            return ast.WalkContinue
        }
        heading, ok := n.(*ast.Heading)
        if ok && heading.Level == level {
            text := extractTextFromNode(heading, source)
            headings = append(headings, text)
        }
        return ast.WalkContinue
    })
    return headings
}
```

### 3. 提取列表项
```go
func ExtractListItemsWithSource(node ast.Node, source []byte) []string {
    var items []string
    walkNode(node, func(n ast.Node, entering bool) ast.WalkStatus {
        if !entering {
            return ast.WalkContinue
        }
        item, ok := n.(*ast.ListItem)
        if ok {
            text := extractTextFromNode(item, source)
            if text != "" {
                items = append(items, text)
            }
        }
        return ast.WalkContinue
    })
    return items
}
```

### 4. AST 遍历
```go
func walkNode(node ast.Node, fn func(ast.Node, bool) ast.WalkStatus) {
    if node == nil {
        return
    }

    queue := []ast.Node{node}
    for len(queue) > 0 {
        current := queue[0]
        queue = queue[1:]

        status := fn(current, true)
        if status == ast.WalkStop {
            return
        }
        if status == ast.WalkSkipChildren {
            continue
        }

        child := current.FirstChild()
        for child != nil {
            queue = append(queue, child)
            child = child.NextSibling()
        }
    }
}
```

### 5. 文本提取
```go
func extractTextFromNode(node ast.Node, source []byte) string {
    var buf bytes.Buffer
    extractTextRecursive(node, &buf, source)
    return buf.String()
}

func extractTextRecursive(node ast.Node, buf *bytes.Buffer, source []byte) {
    if node == nil {
        return
    }

    switch n := node.(type) {
    case *ast.Text:
        segment := n.Segment
        if !segment.IsEmpty() && source != nil {
            buf.Write(segment.Value(source))
        }
    case *ast.String:
        buf.Write(n.Value)
    case *ast.Heading, *ast.Paragraph, *ast.ListItem:
        for child := n.FirstChild(); child != nil; child = child.NextSibling() {
            extractTextRecursive(child, buf, source)
        }
    }
}
```

## 使用示例

### 解析 task.md
```go
package main

import (
    "fmt"
    "os"
    "github.com/sunquan/rick/internal/parser"
)

func main() {
    // 1. 读取 task.md
    content, err := os.ReadFile("task.md")
    if err != nil {
        panic(err)
    }

    // 2. 解析 Markdown
    doc, err := parser.ParseMarkdownWithSource(string(content))
    if err != nil {
        panic(err)
    }

    // 3. 提取标题
    headings := parser.ExtractHeadingWithSource(doc.AST, 1, doc.Source)
    fmt.Printf("Headings: %v\n", headings)

    // 4. 提取列表项（依赖关系）
    deps := parser.ExtractListItemsWithSource(doc.AST, doc.Source)
    fmt.Printf("Dependencies: %v\n", deps)

    // 5. 提取段落（任务目标）
    goals := parser.ExtractParagraphWithSource(doc.AST, doc.Source)
    fmt.Printf("Goals: %v\n", goals)
}
```

## 最佳实践

### 1. 始终传递源码
```go
// ✅ 正确：传递源码
headings := ExtractHeadingWithSource(ast, 1, source)

// ❌ 错误：不传递源码（会返回空字符串）
headings := ExtractHeading(ast, 1)
```

### 2. 检查节点类型
```go
heading, ok := n.(*ast.Heading)
if ok && heading.Level == 1 {
    // 处理 h1 标题
}
```

### 3. 处理空值
```go
if node == nil {
    return
}
```

---

*参考: `internal/parser/markdown.go`*
*最后更新: 2026-03-14*
