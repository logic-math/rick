# 案例: task.md 解析

## task.md 格式
```markdown
# 依赖关系
task1, task2

# 任务名称
创建 Wiki 索引

# 任务目标
建立知识库索引系统

# 关键结果
1. 创建 index.md
2. 生成目录结构

# 测试方法
1. 检查文件是否存在
2. 验证格式是否正确
```

## 解析代码
```go
doc, _ := ParseMarkdownWithSource(content)

// 提取依赖: task1, task2
deps := ExtractListItemsWithSource(doc.AST, doc.Source)

// 提取任务名称: "创建 Wiki 索引"
headings := ExtractHeadingWithSource(doc.AST, 1, doc.Source)
taskName := headings[1]

// 提取目标: "建立知识库索引系统"
goals := ExtractParagraphWithSource(doc.AST, doc.Source)
```

## 输出
```
Dependencies: [task1, task2]
Task Name: 创建 Wiki 索引
Goal: 建立知识库索引系统
Key Results: [创建 index.md, 生成目录结构]
```

---

*最后更新: 2026-03-14*
