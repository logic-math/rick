# Markdown 解析技能

## 技能概述

Markdown 解析技能使用 Goldmark 库解析 Markdown 文档，提取结构化信息（标题、列表、段落、代码块）。通过遍历 AST（抽象语法树），可以精确提取和处理 Markdown 内容，支持 task.md、OKR.md、SPEC.md 等文档的自动解析。

核心库：**Goldmark** - 一个功能强大、符合 CommonMark 规范的 Go Markdown 解析器。

## 使用场景

### 1. 任务定义解析
- **场景**: 从 task.md 文件中提取任务名称、目标、依赖、测试方法
- **示例**: Rick CLI 解析任务文件，构建任务 DAG
- **价值**: 自动化任务管理，无需手工解析

### 2. 文档内容提取
- **场景**: 从 Markdown 文档中提取特定章节内容
- **示例**: 提取 OKR.md 中的所有目标和关键结果
- **价值**: 实现文档驱动的工作流

### 3. 代码块提取
- **场景**: 从技术文档中提取代码示例
- **示例**: 提取 Skills 文档中的所有 Go 代码片段
- **价值**: 自动化代码示例管理

### 4. 知识库索引
- **场景**: 为 Wiki 知识库建立索引
- **示例**: 提取所有标题，生成目录结构
- **价值**: 快速定位知识库内容

### 5. 文档转换
- **场景**: 将 Markdown 转换为其他格式
- **示例**: 生成提示词模板、HTML 报告
- **价值**: 实现多格式输出

## 核心优势

### ✅ 优点

1. **符合标准**: 完全符合 CommonMark 规范，兼容性好
2. **高性能**: 纯 Go 实现，无 CGO 依赖，速度快
3. **易扩展**: 支持自定义渲染器和扩展
4. **AST 遍历**: 提供完整的 AST，可以精确控制解析逻辑
5. **源码映射**: 保留原始源码位置，便于错误定位
6. **零依赖**: Goldmark 本身无外部依赖

### ⚠️ 注意事项

1. **需要源码**: 提取文本时必须传递原始源码字节数组
2. **AST 复杂**: 需要理解 AST 节点类型（Heading、List、Paragraph 等）
3. **遍历顺序**: 需要正确处理节点的父子关系和兄弟关系
4. **空值检查**: 节点可能为 nil，需要检查
5. **编码问题**: 源码必须是 UTF-8 编码

## 适用条件

- ✅ 输入是符合 CommonMark 规范的 Markdown 文档
- ✅ 需要精确提取结构化信息（标题、列表、代码块）
- ✅ 需要保留源码位置信息
- ✅ 可以接受 AST 遍历的复杂性

## 不适用场景

- ❌ 简单的字符串匹配（用正则表达式更简单）
- ❌ 非 CommonMark 格式的 Markdown（如 GitHub Flavored Markdown 特有语法）
- ❌ 需要渲染 HTML（直接用 Goldmark 的 HTML 渲染器）
- ❌ 实时解析大量文档（考虑缓存）

## 关键概念

### AST（抽象语法树）
- **定义**: Markdown 文档的树形结构表示
- **节点类型**: Heading, Paragraph, List, ListItem, CodeBlock, Text, etc.
- **遍历**: 使用 WalkFunc 或手动遍历子节点

### Goldmark 核心类型

#### ast.Node
```go
type Node interface {
    FirstChild() Node    // 第一个子节点
    LastChild() Node     // 最后一个子节点
    NextSibling() Node   // 下一个兄弟节点
    PreviousSibling() Node // 上一个兄弟节点
    Parent() Node        // 父节点
}
```

#### ast.Heading
```go
type Heading struct {
    Level int  // 1 for h1, 2 for h2, etc.
}
```

#### ast.Text
```go
type Text struct {
    Segment text.Segment  // 原始源码位置
}
```

### 源码提取
- **问题**: AST 节点不直接存储文本内容
- **解决**: 使用 Segment.Value(source) 从源码字节数组提取文本
- **示例**: `segment.Value(source)` 返回节点对应的原始文本

## Rick CLI 中的应用

### 解析 task.md
```go
// 提取依赖关系（第一个标题下的列表）
dependencies := ExtractListItems(ast, source)

// 提取任务名称（第二个标题）
headings := ExtractHeading(ast, 1, source)
taskName := headings[1]

// 提取任务目标（第二个标题后的段落）
goals := ExtractParagraph(ast, source)
```

### 提取代码块
```go
// 提取所有代码块
codeBlocks := ExtractCodeBlock(ast, source)

// 获取代码块语言
lang := GetCodeBlockLanguage(codeBlock, source)
```

## 实际效果

在 Rick CLI 项目中：
- **解析速度**: 1KB Markdown 文档 < 1ms
- **准确率**: 100%（符合 CommonMark 规范）
- **应用场景**: task.md 解析、OKR.md 提取、技能文档索引

## 扩展阅读

- [Goldmark GitHub](https://github.com/yuin/goldmark)
- [CommonMark 规范](https://commonmark.org/)
- Rick CLI 源码: `internal/parser/markdown.go`

---

*难度: ⭐⭐⭐*
*分类: 文档处理*
*最后更新: 2026-03-14*
