# 依赖关系


# 任务名称
修复三个 check 工具与实际产出格式的不一致问题

# 任务目标
通过对照 check 源码与 job_9 实际产出文件（task*.md、tasks.json、debug.md、learning/SUMMARY.md、learning/OKR.md、learning/SPEC.md），发现以下不一致问题需要修复，确保 check 工具能准确反映各阶段产出是否符合规范：

**plan_check 缺失检查**：
- plan.md 模板约束 0 明确要求必须生成 `job_N/plan/OKR.md`，doing 阶段的 `runner.go` 也会读取 `job_N/plan/OKR.md`，但 plan_check 完全不检查 OKR.md 是否存在。

**doing_check 缺失检查**：
- debug.md 只检查文件存在，不检查内容是否非空。Agent 可能生成空的 debug.md，check 会误报通过。
- doing.md 模板要求 debug.md 包含 `## task{N}: ...` 格式的记录，但 check 不验证。

**learning_check 缺失检查**：
- SUMMARY.md 只检查文件存在，不检查是否包含基本内容结构（如 `# Job` 标题行或 `## 执行概述` 章节）。
- SUMMARY.md 需要在最终确认后添加 `APPROVED: true` 第一行，但 check 不验证（注意：check 是在 merge 前运行的，此时 APPROVED 可能还未写入，所以这一项**不应该**检查 APPROVED）。

**具体修改**：

1. `internal/cmd/tools_plan_check.go` 的 `runPlanCheck()` 函数：
   - 在现有检查之后，新增第 6 项检查：验证 `plan/OKR.md` 文件存在
   - 错误信息：`"OKR.md not found in plan directory: %s"`

2. `internal/cmd/tools_doing_check.go` 的 `runDoingCheck()` 函数：
   - 增强 debug.md 检查：验证文件内容非空（`len(strings.TrimSpace(content)) > 0`）
   - 验证 debug.md 包含至少一个 `## task` 开头的记录（`strings.Contains(content, "## task")`）
   - 错误信息：`"debug.md exists but is empty"` / `"debug.md contains no task records (missing ## task section)"`

3. `internal/cmd/tools_learning_check.go` 的 `runLearningCheck()` 函数：
   - 增强 SUMMARY.md 检查：读取文件内容，验证非空且包含 `# Job` 标题（`strings.Contains(content, "# Job")`）
   - 错误信息：`"SUMMARY.md exists but is empty or missing required '# Job' heading"`

同时更新对应的 `writePlanCheckFixPrompt`、`writeDoingCheckFixPrompt`、`writeLearningCheckFixPrompt` 函数，在 Instructions 中加入新增检查项的修复说明。

# 关键结果
1. `runPlanCheck()` 新增检查 `plan/OKR.md` 存在性，缺失时返回明确错误
2. `runDoingCheck()` 增强 debug.md 检查：内容非空 + 包含 `## task` 记录
3. `runLearningCheck()` 增强 SUMMARY.md 检查：内容非空 + 包含 `# Job` 标题
4. 三个 fix prompt 函数的 Instructions 更新，包含新检查项的修复指导
5. `go test ./internal/cmd/... -v` 全部通过（重点：tools_test.go 中的 plan/doing/learning check 测试）
6. 对 job_9 的实际产出运行三个 check，全部通过（验证新增检查不会误报）

# 测试方法
1. 运行 `go build ./...` 确认编译通过
2. 运行 `go test ./internal/cmd/... -v -run TestPlanCheck` 确认 plan_check 测试通过
3. 运行 `go test ./internal/cmd/... -v -run TestDoingCheck` 确认 doing_check 测试通过
4. 运行 `go test ./internal/cmd/... -v -run TestLearningCheck` 确认 learning_check 测试通过
5. 运行 `rick tools plan_check job_9` 验证 job_9 的 plan 目录通过检查（注意：job_9/plan 没有 OKR.md，预期失败——这说明新检查是正确的，job_9 确实缺少 OKR.md）
6. 运行 `rick tools doing_check job_9` 验证 job_9 的 doing 目录通过检查
7. 运行 `rick tools learning_check job_9` 验证 job_9 的 learning 目录通过检查
8. 运行 `go test ./... -count=1` 确认全量测试通过
