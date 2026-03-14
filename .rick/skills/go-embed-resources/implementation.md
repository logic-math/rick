# Go 嵌入式资源 - 实现细节

## 嵌入单个文件

### string 类型
```go
package main

import (
    _ "embed"
    "fmt"
)

//go:embed config.json
var defaultConfig string

func main() {
    fmt.Println(defaultConfig)
}
```

### []byte 类型
```go
//go:embed logo.png
var logoBytes []byte

func serveLogo(w http.ResponseWriter, r *http.Request) {
    w.Write(logoBytes)
}
```

## 嵌入多个文件

### embed.FS（文件系统）
```go
package main

import (
    "embed"
    "fmt"
)

//go:embed templates/*.md
var templatesFS embed.FS

func main() {
    // 读取单个文件
    content, _ := templatesFS.ReadFile("templates/plan.md")
    fmt.Println(string(content))

    // 列出所有文件
    files, _ := templatesFS.ReadDir("templates")
    for _, file := range files {
        fmt.Println(file.Name())
    }
}
```

## Rick CLI 实现

### 完整代码
```go
package prompt

import (
    _ "embed"
    "fmt"
    "os"
    "path/filepath"
)

var (
    //go:embed templates/plan.md
    planTemplate string

    //go:embed templates/doing.md
    doingTemplate string

    //go:embed templates/learning.md
    learningTemplate string

    //go:embed templates/test.md
    testTemplate string
)

type PromptManager struct {
    templateDir string
    cache       map[string]*PromptTemplate
}

func (pm *PromptManager) LoadTemplate(name string) (*PromptTemplate, error) {
    var content string

    // 优先从文件系统加载（支持自定义模板）
    if pm.templateDir != "" {
        templatePath := filepath.Join(pm.templateDir, name+".md")
        fileContent, err := os.ReadFile(templatePath)
        if err == nil {
            content = string(fileContent)
        } else {
            // Fallback 到嵌入的模板
            content = pm.getEmbeddedTemplate(name)
        }
    } else {
        // 直接使用嵌入的模板
        content = pm.getEmbeddedTemplate(name)
    }

    if content == "" {
        return nil, fmt.Errorf("template %s not found", name)
    }

    return &PromptTemplate{
        Name:    name,
        Content: content,
    }, nil
}

func (pm *PromptManager) getEmbeddedTemplate(name string) string {
    switch name {
    case "plan":
        return planTemplate
    case "doing":
        return doingTemplate
    case "learning":
        return learningTemplate
    case "test":
        return testTemplate
    default:
        return ""
    }
}
```

## 最佳实践

### 1. Fallback 机制
```go
// 优先文件系统，然后嵌入资源
if content, err := os.ReadFile(path); err == nil {
    return content
}
return embeddedContent
```

### 2. 版本信息
```go
//go:embed VERSION
var version string

func GetVersion() string {
    return strings.TrimSpace(version)
}
```

### 3. 多级目录
```go
//go:embed templates/**/*
var templatesFS embed.FS

// 访问: templates/plan/v1.md
content, _ := templatesFS.ReadFile("templates/plan/v1.md")
```

---

*参考: `internal/prompt/manager.go`*
*最后更新: 2026-03-14*
