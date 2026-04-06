# 依赖关系
task1, task2, task3

# 任务名称
OKR 改为 job 级：plan 生成 job OKR，doing/learning 读取 job OKR

# 任务目标
当前 plan/doing/learning 都从全局 `.rick/OKR.md` 读取 OKR。RFC-001 要求 OKR 改为 job 级：
- agent 只需知道完成当前 job 所需的目标信息
- plan 阶段在 `job_N/plan/OKR.md` 生成 job 级 OKR（由 Claude 在 plan 会话中生成）
- doing 阶段从 `job_N/plan/OKR.md` 读取 OKR 注入 prompt
- learning 阶段从 `job_N/plan/OKR.md` 读取 OKR（task1 已实现读取，本任务确保路径正确）
- 全局 `.rick/OKR.md` 保留但不再注入任何 prompt

具体改动：
1. `plan.go:executePlanWorkflow()` 和 `reEnterPlanWorkflow()`：删除加载全局 OKR 的代码（`contextMgr.LoadOKRFromFile(okriPath)`），改为在 prompt 中告知 Claude 需要在 `job_N/plan/OKR.md` 生成 job 级 OKR
2. `plan.md` 模板：在"约束 0"章节明确要求 Claude 生成 `job_N/plan/OKR.md`，描述本 job 的聚焦目标（不是全局项目目标）；同时删除 `{{okr_content}}` 变量（全局 OKR 不再注入 plan prompt）
3. `plan_prompt.go`：删除 `okr_content` 变量的设置逻辑
4. `doing.go:executeDoingWorkflow()`：新增读取 `job_N/plan/OKR.md` 并通过 contextMgr 注入 doing prompt
5. `doing.md` 模板：新增 `{{job_okr_content}}` 变量，在"项目背景"章节展示 job 级 OKR
6. `doing_prompt.go`：新增 `job_okr_content` 变量设置

# 关键结果
1. `plan.go` 不再读取全局 `.rick/OKR.md`，`plan_prompt.go` 删除 `okr_content` 变量
2. `plan.md` 模板明确要求 Claude 生成 `{{job_plan_dir}}/OKR.md`，内容为本 job 的聚焦目标
3. `doing.go` 新增读取 `job_N/plan/OKR.md` 的逻辑，注入 doing prompt
4. `doing.md` 模板展示 job 级 OKR（`{{job_okr_content}}`）
5. `go test ./...` 全部通过

# 测试方法
1. 运行 `go build ./...` 确认编译通过
2. 运行 `go test ./internal/cmd/ -v` 确认 plan/doing 相关测试通过
3. 运行 `go test ./internal/prompt/ -v` 确认 prompt 相关测试通过
4. 运行 `go test ./...` 确认全量测试通过
5. 构建后运行 `rick plan --dry-run`，确认生成的 plan prompt 不包含全局 OKR 内容，但包含"生成 job 级 OKR.md"的要求
6. 构建后运行 `rick doing --dry-run job_9`，确认 doing prompt 包含 job OKR 章节（若 OKR.md 存在）
