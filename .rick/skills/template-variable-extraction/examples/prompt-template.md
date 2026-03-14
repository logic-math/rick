# 案例: 提示词模板变量提取

## 模板文件（doing.md）
```markdown
# Task Execution

## Task Information
- Task ID: {{task_id}}
- Task Name: {{task_name}}
- Goal: {{goal}}

## Context
### OKR
{{okr}}

### SPEC
{{spec}}

### Debug Context
{{debug}}
```

## 提取变量
```go
template, _ := os.ReadFile("templates/doing.md")
variables := extractVariables(string(template))

fmt.Printf("Variables: %v\n", variables)
// Output: [task_id, task_name, goal, okr, spec, debug]
```

## 验证变量
```go
data := map[string]string{
    "task_id":   "task1",
    "task_name": "创建 Wiki",
    "goal":      "建立索引",
    "okr":       "...",
    "spec":      "...",
    // 缺少 "debug"
}

for _, v := range variables {
    if _, exists := data[v]; !exists {
        fmt.Printf("Error: variable %s not provided\n", v)
    }
}
// Output: Error: variable debug not provided
```

---

*最后更新: 2026-03-14*
