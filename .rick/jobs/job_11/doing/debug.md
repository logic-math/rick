## task3: 修复三个 check 工具与实际产出格式的不一致问题

**分析过程 (Analysis)**:
- 阅读了 `internal/cmd/tools_plan_check.go`、`tools_doing_check.go`、`tools_learning_check.go` 源码
- 阅读了 `internal/cmd/tools_test.go` 了解现有测试结构
- 发现 plan_check 的 `runPlanCheck()` 在检查5项后直接通过，未检查 OKR.md 存在性
- 发现 doing_check 的 `runDoingCheck()` 只检查 debug.md 文件存在，不检查内容非空和 `## task` 记录
- 发现 learning_check 的 `runLearningCheck()` 只检查 SUMMARY.md 存在，不检查内容结构
- 选择最小化修改方案：在现有检查链后追加新检查，不重构现有逻辑

**实现步骤 (Implementation)**:
1. `tools_plan_check.go`：在第5项检查（DAG循环依赖）之后，新增第6项检查 OKR.md 存在性；更新 `writePlanCheckFixPrompt` 的 Instructions 加入第5条说明
2. `tools_doing_check.go`：添加 `"strings"` import；增强 debug.md 检查，读取内容验证非空和包含 `## task` 记录；更新 `writeDoingCheckFixPrompt` 的 Instructions 加入非空和 task 记录要求
3. `tools_learning_check.go`：增强 SUMMARY.md 检查，读取内容验证非空且包含 `# Job` 标题；更新 `writeLearningCheckFixPrompt` 的 Instructions 加入内容要求
4. `tools_test.go`：
   - `TestRunPlanCheck_Valid` 和相关 workspace 测试新增写入 OKR.md
   - 新增 `TestRunPlanCheck_MissingOKR` 测试
   - 将所有 `[]byte("# debug")` 替换为 `[]byte("## task1: did work\nsome content")`
   - 新增 `TestRunDoingCheck_EmptyDebugMD` 和 `TestRunDoingCheck_NoTaskRecords` 测试
   - 将所有 `[]byte("summary")` 替换为 `[]byte("# Job Summary\nsome content")`
   - 新增 `TestRunLearningCheck_EmptySummary` 和 `TestRunLearningCheck_MissingJobHeading` 测试

**遇到的问题 (Issues)**:
- 无

**验证结果 (Verification)**:
- 测试命令：`go test ./internal/cmd/... -v -run "TestRunPlanCheck|TestRunDoingCheck|TestRunLearningCheck"`
- 测试输出：35 tests PASS
- 测试命令：`go test ./... -count=1`
- 结论：✅ 全量测试通过
- 验证命令：`./bin/rick tools plan_check job_9`（auto-fix 创建了 OKR.md，之后通过）
- 验证命令：`./bin/rick tools doing_check job_9` → ✅ doing check passed: 4/5 tasks succeeded
- 验证命令：`./bin/rick tools learning_check job_9` → ✅ learning check passed
