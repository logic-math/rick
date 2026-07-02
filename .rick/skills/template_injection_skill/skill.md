# skill:template-injection（向 plan/easy 模板注入新 skill）

## 触发场景

需要在 `rick plan` 或 `rick easy` 会话中嵌入新的结构化行为时：
- 在 plan/easy prompt 中添加新 skill 引用
- 修改模板变量名（如替换 `{{old_var}}` → `{{new_var}}`）
- 验证模板注入后 dry-run 输出中无残留占位符

信号词：「在 plan/easy 中增加 X 步骤」「注入新 skill」「模板变量未替换」

## 预期效果

- plan/easy 生成的 prompt 文件包含 skill 绝对路径（可点击验证）
- dry-run 输出不含未替换的 `{{xxx_skill_path}}` 占位符
- `go test ./internal/prompt/...` 全绿

## 核心内容

### Step 1：创建 skill 文件

```
internal/prompt/templates/skills/<skill_name>.md
```

禁止在 skill 文件内使用 `{{变量}}` 占位符。

### Step 2：在模板中添加步骤

编辑 `internal/prompt/templates/plan.md` 或 `easy.md`：

```markdown
**加载并执行 skill:xxx**：`{{xxx_skill_path}}`
```

### Step 3：更新 prompt builder 注入变量

在 `plan_prompt.go` 或 `easy_prompt.go` 中：

```go
xxxFile, err := WriteSkillFile(promptsDir, "skill_xxx.md", "xxx")
if err != nil { return "", nil, err }
builder.SetVariable("xxx_skill_path", xxxFile)
```

dry-run 分支也要同步：
```go
builder.SetVariable("xxx_skill_path", "<tmp>/rick-plan-prompts/skill_xxx.md")
```

### Step 4：写 RED 测试

```go
func TestGeneratePlanPrompt_HasXxxSkillPath(t *testing.T) {
    prompt := GeneratePlanPrompt(...)
    // 断言用文件名，不用部分字符串
    if !strings.Contains(prompt, "skill_xxx.md") {
        t.Error("Expected prompt to contain skill_xxx.md reference")
    }
}
```

### Step 5：验证

```bash
./scripts/build.sh
go test ./internal/prompt/... -v
./bin/rick plan --dry-run | grep -c '{{xxx_skill_path}}'  # 应为 0
./bin/rick plan --dry-run | grep 'skill_xxx'              # 应有输出
```

### 验证 section 内容的精确方法

```python
import re

output = subprocess.run([rick_bin, "plan", "--dry-run"], ...).stdout
match = re.search(r'## 可用的项目 Skills(.*?)(?=^##|\Z)', output, re.DOTALL | re.MULTILINE)
skills_section = match.group(1) if match else ""
assert "skill_xxx" in skills_section
```

### 修改变量名时（全局替换）

先参考 [global_ref_sync_skill](../global_ref_sync_skill/skill.md)，全局 grep 旧变量名找出所有引用，再批量修改。
