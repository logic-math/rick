# 依赖关系

task1

# 任务名称

升级 learning 阶段：注入 draft_dir 变量并添加 domain 事实同步到 draft/progress.md 步骤

# 任务目标

在 `internal/prompt/templates/learning.md` 中添加 `{{draft_dir}}` 变量和"Draft 同步"可选步骤（如 draft/ 存在，将本次 job 产出的关键 domain 事实追加到 `draft/progress.md`）；在 `internal/cmd/learning.go` 的 `buildLearningPrompt()` 中注入 `draft_dir`（通过 `workspace.GetDraftDir()`），同时在 `runLearningDryRun()` 中注入占位路径。

# 关键结果

1. `learning.md` 末尾新增"## Draft 同步（可选）"章节：如 `{{draft_dir}}` 目录存在，在所有 loop 步骤完成后将本次 job 关键 domain 事实追加到 `{{draft_dir}}/progress.md`（格式：`## [job_id] 学习记录` + 本次新增知识点列表）；如目录不存在则跳过
2. `internal/cmd/learning.go` 的 `buildLearningPrompt()` 函数调用 `workspace.GetDraftDir()`，将结果通过 `builder.SetVariable("draft_dir", draftDir)` 注入
3. `runLearningDryRun()` 同样注入 draft_dir（如 GetDraftDir 失败则用空字符串，不中断 dry-run）
4. 新增单元测试验证以上行为

# 测试方法

**前提：使用 skill:tdd，先写失败测试，再实现，再看绿。**

### 测试 1：learning 模板包含 draft_dir 变量

- 前置条件：内嵌模板已加载
- 输入：`pm.LoadTemplate("learning")`
- 操作序列：检查模板 Content 字段
- 预期输出：包含 `{{draft_dir}}`

```bash
go test ./internal/prompt/... -run TestLearningTemplateContainsDraftDir -v
```

### 测试 2：buildLearningPrompt 注入 draft_dir 并替换变量

- 前置条件：tmpDir 结构（rickDir + doingDir + tasks.json），os.Chdir 到 tmpDir
- 输入：构造 `&ExecutionData{JobID: "job_test", RickDir: rickDir}`，调用 `buildLearningPrompt`
- 操作序列：读取输出 promptFile 内容
- 预期输出：内容包含 `draft` 路径字符串，不含未替换的 `{{draft_dir}}`

```bash
go test ./internal/cmd/... -run TestBuildLearningPromptContainsDraftDir -v -timeout 30s
```

### 测试 3：dry-run 输出不含未替换的 {{draft_dir}}

- 前置条件：workDir 下有 `.rick/` 目录
- 输入：`./bin/rick learning job_N --dry-run`（job_N 不存在也输出 prompt）
- 操作序列：捕获 stdout
- 预期输出：stdout 不含 `{{draft_dir}}`

```bash
./bin/rick learning --dry-run | grep -c '{{draft_dir}}'
# 应为 0
```

# 约束说明

1. **`buildLearningPrompt` 签名不变**：函数保持 `(data *ExecutionData, learningDir, promptsDir string)` 不加参数，在函数内部调用 `workspace.GetDraftDir()` 获取 draft_dir
2. **错误处理**：`GetDraftDir()` 失败时 draft_dir 赋空字符串，继续构建 prompt 不中断（dry-run 也适用）
3. 两处调用方（`runLearningDryRun` line 106 和 `callClaudeForAnalysis` line 227）**无需修改**

# 参考 Loops/Skills

- `tdd-red-green-refactor-loop`：Go 代码（learning.go）改动时先写失败测试再实现
- `.rick/skills/verify_go_changes_skill/skill.md`：Go 和模板改动后构建验证
- `.rick/skills/template_injection_skill/skill.md`：`{{draft_dir}}` 变量注入规范
- `.rick/skills/mark_task_success_skill/skill.md`：task 完成后标记 success

### 边界测试：draft/ 目录不存在时 learning 正常执行

- 前置条件：rickDir 下无 draft/ 目录
- 预期输出：buildLearningPrompt 不 panic，draft_dir 变量被注入（值为路径，目录不存在），模板中"如不存在则跳过"的描述使 agent 不执行同步步骤

```bash
go test ./internal/cmd/... -run TestBuildLearningPromptNoDraftDir -v
```
