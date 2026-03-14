# Go 嵌入式资源技能

## 技能概述

Go 1.16+ 引入的 `//go:embed` 指令允许将文件内容直接嵌入到编译后的二进制文件中。这使得应用程序可以自包含所有必需的资源文件（模板、配置、静态文件），无需在运行时从文件系统读取。

核心特性：**编译时嵌入**。资源文件在编译时被打包到可执行文件中，运行时直接访问，无需额外的文件部署。

## 使用场景

### 1. 模板文件嵌入
- **场景**: 将提示词模板嵌入到 CLI 工具中
- **示例**: Rick CLI 嵌入 plan.md、doing.md、learning.md、test.md 模板
- **价值**: 单文件部署，无需额外配置

### 2. 配置文件嵌入
- **场景**: 嵌入默认配置文件
- **示例**: 默认的 config.json、schema.json
- **价值**: 提供开箱即用的默认配置

### 3. 静态资源嵌入
- **场景**: Web 应用嵌入 HTML、CSS、JS 文件
- **示例**: 前端资源直接打包到后端二进制
- **价值**: 简化部署流程

### 4. SQL 脚本嵌入
- **场景**: 嵌入数据库迁移脚本
- **示例**: schema.sql、migrations/*.sql
- **价值**: 避免脚本文件丢失

### 5. 文档嵌入
- **场景**: 嵌入帮助文档、README
- **示例**: 内置用户手册
- **价值**: 离线访问文档

## 核心优势

### ✅ 优点

1. **单文件部署**: 所有资源打包到一个可执行文件中
2. **无运行时依赖**: 不需要在目标机器上部署资源文件
3. **版本一致性**: 资源和代码版本同步
4. **简化分发**: 只需分发一个二进制文件
5. **性能优化**: 避免文件 I/O，直接从内存访问
6. **安全性**: 资源文件不会被意外修改或删除

### ⚠️ 注意事项

1. **二进制体积增大**: 嵌入大量资源会增加可执行文件大小
2. **编译时固定**: 修改资源需要重新编译
3. **内存占用**: 嵌入的资源会占用内存
4. **不支持动态更新**: 无法在运行时修改嵌入的资源
5. **路径限制**: 只能嵌入相对于 Go 文件的路径

## 适用条件

- ✅ 资源文件相对静态，不需要频繁修改
- ✅ 需要单文件部署
- ✅ 资源文件总大小适中（< 10MB）
- ✅ 可以接受编译时固化资源

## 不适用场景

- ❌ 资源文件非常大（如视频、大型数据集）
- ❌ 需要动态更新资源（如用户配置）
- ❌ 资源文件频繁变化
- ❌ 需要根据环境加载不同资源

## 关键概念

### //go:embed 指令
```go
//go:embed templates/plan.md
var planTemplate string

//go:embed templates/*.md
var templates embed.FS
```

### 嵌入类型

#### string
```go
//go:embed file.txt
var content string
```

#### []byte
```go
//go:embed image.png
var imageData []byte
```

#### embed.FS
```go
//go:embed templates/*
var templatesFS embed.FS
```

## Rick CLI 中的应用

### 嵌入提示词模板
```go
package prompt

import (
    _ "embed"
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
```

### 使用嵌入的模板
```go
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

## 实际效果

在 Rick CLI 项目中：
- **二进制大小**: 增加约 20KB（4 个模板文件）
- **加载速度**: < 1μs（直接内存访问）
- **部署便利性**: 单个可执行文件，无需额外文件

## 扩展阅读

- [Go embed 包文档](https://pkg.go.dev/embed)
- [Go 1.16 Release Notes](https://go.dev/doc/go1.16#library-embed)
- Rick CLI 源码: `internal/prompt/manager.go`

---

*难度: ⭐⭐*
*分类: Go 语言特性*
*最后更新: 2026-03-14*
