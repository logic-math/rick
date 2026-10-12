# 依赖关系
task3

# 任务名称
在 plan/doing/learning 提示词模板中强制集成 check 机制

# 任务目标
当前 plan.md 和 doing.md 模板完全没有 check 调用指令，导致 Agent 产出文件后不会自我验证格式正确性。learning.md 的 Step 3 有 learning_check 调用但措辞较弱（仅说"运行命令"，未强调"必须通过才能继续"）。

需要修改三个模板文件，让 Agent 在每个阶段完成产出后强制运行对应的 check 命令，根据错误信息修复，直到 check 通过为止。

同时需要在 plan_prompt.go 和 doing_prompt.go 中注入 `rick_bin_path` 和 `job_id` 两个模板变量（目前这两个文件中没有注入这两个变量，但模板中会用到）。

**具体修改点**：

1. `internal/prompt/templates/plan.md`：在末尾"开始执行"章节之前，新增"五.1、强制验证步骤"章节，内容为：完成所有 task*.md 和 OKR.md 产出后，必须运行 `{{rick_bin_path}} tools plan_check {{job_id}}`，如果失败则根据错误信息修复文件，循环直到通过再结束。

2. `internal/prompt/templates/doing.md`：在"行为约束"章节末尾新增第 7 条约束："强制 doing check：在 git commit 之后，必须运行 `{{rick_bin_path}} tools doing_check {{job_id}}`，如果失败则根据错误信息修复（如补充 debug.md、解决 zombie 任务等），循环直到通过"。

3. `internal/prompt/templates/learning.md`：Step 3 当前措辞是先说"运行以下命令了解可用的元技能"（--help），然后才运行 check。需要强化为：明确说明 learning_check 必须通过才能进入 Step 4，如果失败则根据错误信息修复产出文件，循环直到通过。

4. `internal/prompt/plan_prompt.go`：在 `GeneratePlanPromptFile` 和 `GeneratePlanPrompt` 函数中注入 `rick_bin_path` 和 `job_id` 变量。`rick_bin_path` 的值参考 `internal/cmd/learning.go` 中的逻辑（查找 `./bin/rick`，找不到则用 `rick`）；`job_id` 需要从 `jobPlanDir` 路径中提取（路径格式为 `.rick/jobs/job_N/plan`，提取 `job_N` 部分）。

5. `internal/prompt/doing_prompt.go`：在 `GenerateDoingPromptFile` 和 `GenerateDoingPrompt` 函数中注入 `rick_bin_path` 和 `job_id` 变量。`job_id` 需要从 `tr.config.WorkspaceDir`（格式为 `.rick/jobs/job_N/doing`）提取，或者通过新增参数传入。由于 `GenerateDoingPromptFile` 是 `prompt` 包的函数，需要从调用方（`internal/executor/runner.go`）传入 jobID，或者从 WorkspaceDir 路径解析。

   注意：`GenerateDoingPromptFile` 的签名在 `internal/executor/runner.go` 中被调用，修改时要保持向后兼容（可用 variadic 参数）。

# 关键结果
1. plan.md 模板包含强制 plan_check 步骤，使用 `{{rick_bin_path}}` 和 `{{job_id}}` 变量
2. doing.md 模板包含强制 doing_check 步骤，使用 `{{rick_bin_path}}` 和 `{{job_id}}` 变量
3. learning.md 模板 Step 3 措辞强化，明确"必须通过才能进入 Step 4"
4. plan_prompt.go 中 `rick_bin_path` 和 `job_id` 被正确注入（变量替换后不含 `{{` 占位符）
5. doing_prompt.go 中 `rick_bin_path` 和 `job_id` 被正确注入（变量替换后不含 `{{` 占位符）
6. 所有修改编译通过，现有测试不受破坏

# 测试方法
1. 运行 `go build ./...` 确认编译通过
2. 运行 `go test ./internal/prompt/... -v` 确认 prompt 相关测试通过（重点检查 plan_prompt_test.go 和 doing_prompt_test.go）
3. 运行 `rick plan --dry-run "测试需求"` 检查输出中：
   a. 包含 `plan_check` 关键词
   b. 不含原始占位符 `{{rick_bin_path}}` 或 `{{job_id}}`（已被替换）
4. 运行 `rick doing job_1 --dry-run` 检查输出中：
   a. 包含 `doing_check` 关键词
   b. 不含原始占位符 `{{rick_bin_path}}` 或 `{{job_id}}`
5. 检查 learning.md 模板文件，Step 3 包含"必须通过"或"才能进入 Step 4"等强制性措辞
6. 运行 `go test ./... -count=1` 确认全量测试通过
