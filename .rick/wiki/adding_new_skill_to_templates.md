---
name: adding_new_skill_to_templates
description: 向 plan/easy 模板注入新 skill 的完整流程（WriteSkillFile 模式）
type: project
---

# 向 plan/easy 模板注入新 Skill

## 触发场景

当需要在 `rick plan` 或 `rick easy` 会话中嵌入新的结构化行为（如追问协议、安全检查、格式约束）时使用。
信号词：「在 plan/easy 中增加 X 步骤」「让 Claude 在开始前先执行 Y」

## 预期效果

- plan/easy 生成的 prompt 文件包含 skill 绝对路径（运行时注入，可点击验证）
- dry-run 输出不含未替换的 `{{xxx_skill_path}}` 占位符
- `go test ./internal/prompt/...` 全绿

## 使用方法

### 1. 创建 skill 文件

```
internal/prompt/templates/skills/<skill_name>.md
```

内容格式：参考 `grilling.md`（核心指令 + 追问协议 + 终止条件）。禁止在 skill 文件中使用 `{{变量}}` 占位符。

### 2. 在模板中添加步骤

编辑 `internal/prompt/templates/plan.md` 或 `easy.md`，在合适位置插入：

```markdown
**加载并执行 skill:xxx**：`{{xxx_skill_path}}`
```

### 3. 更新 prompt.go 注入变量

在 `plan_prompt.go` 或 `easy_prompt.go` 中：

```go
// 写出 skill 文件到 prompts 目录
xxxFile, err := WriteSkillFile(promptsDir, "skill_xxx.md", "xxx")
if err != nil {
    return "", nil, err
}
// 注入路径变量
builder.SetVariable("xxx_skill_path", xxxFile)
```

dry-run 占位符（`GeneratePlanPrompt` 分支）也需同步更新：

```go
builder.SetVariable("xxx_skill_path", "<tmp>/rick-plan-prompts/skill_xxx.md")
```

### 4. 写 RED 测试（先失败）

```go
func TestGeneratePlanPrompt_HasXxxSkillPath(t *testing.T) {
    prompt := GeneratePlanPrompt(...)
    // 注意：断言字符串必须与 WriteSkillFile 写出的文件名一致
    if !strings.Contains(prompt, "skill_xxx.md") {
        t.Error("Expected prompt to contain skill_xxx.md reference")
    }
}
```

**关键**：断言用 `"skill_xxx.md"`（文件名），不要用 `"xxx_skill"` 等部分字符串，否则与实际路径不匹配。

### 5. 验证

```bash
./scripts/build.sh
go test ./internal/prompt/... -v
./bin/rick plan --dry-run | grep -c '{{xxx_skill_path}}'  # 应为 0
./bin/rick plan --dry-run | grep 'skill_xxx'              # 应有输出
```
