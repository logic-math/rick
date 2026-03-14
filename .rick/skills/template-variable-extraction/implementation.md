# 模板变量提取 - 实现细节

## 核心算法

### extractVariables
```go
func extractVariables(content string) []string {
    var variables []string
    seen := make(map[string]bool)

    // 扫描字符串，查找 {{variable}}
    for i := 0; i < len(content)-3; i++ {
        if content[i:i+2] == "{{" {
            // 找到开始标记，查找结束标记 }}
            for j := i + 2; j < len(content)-1; j++ {
                if content[j:j+2] == "}}" {
                    variable := content[i+2 : j]
                    variable = trimSpace(variable)
                    if variable != "" && !seen[variable] {
                        variables = append(variables, variable)
                        seen[variable] = true
                    }
                    i = j + 1
                    break
                }
            }
        }
    }

    return variables
}
```

### trimSpace（无 strings 包）
```go
func trimSpace(s string) string {
    start := 0
    end := len(s)

    // 去除前导空格
    for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n' || s[start] == '\r') {
        start++
    }

    // 去除尾随空格
    for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\n' || s[end-1] == '\r') {
        end--
    }

    return s[start:end]
}
```

## 测试用例

### 测试1: 简单变量
```go
content := "Hello {{name}}, you are {{age}} years old"
vars := extractVariables(content)
// Output: [name, age]
```

### 测试2: 带空格的变量
```go
content := "Hello {{ name }}, you are {{ age }} years old"
vars := extractVariables(content)
// Output: [name, age]  // 自动 trim
```

### 测试3: 重复变量
```go
content := "{{name}} is {{age}}, and {{name}} is happy"
vars := extractVariables(content)
// Output: [name, age]  // 自动去重
```

### 测试4: 无变量
```go
content := "Hello World"
vars := extractVariables(content)
// Output: []
```

## 使用示例
```go
template := `
# Task: {{task_name}}

## Goal
{{goal}}

## Context
{{okr}}
{{spec}}
`

variables := extractVariables(template)
fmt.Printf("Variables: %v\n", variables)
// Output: Variables: [task_name, goal, okr, spec]
```

---

*参考: `internal/prompt/manager.go`*
*最后更新: 2026-03-14*
