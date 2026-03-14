# 案例: Rick CLI 提示词模板嵌入

## 背景

Rick CLI 需要使用 4 个提示词模板（plan.md, doing.md, learning.md, test.md）。使用 `//go:embed` 嵌入模板文件，实现单文件部署。

## 目录结构
```
internal/prompt/
  manager.go          # 使用 //go:embed
  templates/
    plan.md
    doing.md
    learning.md
    test.md
```

## 实现代码
```go
package prompt

import _ "embed"

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

## 效果

### 编译前
```
ls -lh rick
# rick 不存在

ls templates/
# plan.md doing.md learning.md test.md
```

### 编译后
```
go build -o rick cmd/rick/main.go
ls -lh rick
# -rwxr-xr-x  rick  8.5M

# 删除 templates/ 目录
rm -rf templates/

# Rick 仍然可以正常工作
./rick plan "新任务"
# ✓ 使用嵌入的 plan.md 模板
```

## 优势

1. **单文件部署**: 只需要 `rick` 可执行文件
2. **无运行时依赖**: 不需要额外的 templates/ 目录
3. **版本一致性**: 模板和代码版本同步

---

*最后更新: 2026-03-14*
