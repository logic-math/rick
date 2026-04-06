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
- 测试脚本 Test 5 假设 `rick tools plan_check job_9` 会因缺少 OKR.md 而失败，但实际上：
  1. 上一轮 auto-fix 已经在 job_9/plan 创建了 OKR.md，所以检查通过
  2. 即使临时删除 OKR.md，plan_check 的 auto-fix 机制会自动调用 Claude 恢复它
  - 解决方案：改为静态检查源码是否包含 OKR.md 检查逻辑，依赖 Go 单元测试（TestRunPlanCheck_MissingOKR）验证行为
- 测试脚本使用系统 `rick` 命令（安装版），而非本地构建的 `./bin/rick`（含新代码）
  - 解决方案：测试脚本先构建 `./bin/rick`，所有 `rick tools` 命令改用 `{rick_bin} tools`

**验证结果 (Verification)**:
- 测试命令：`python3 .rick/jobs/job_11/doing/tests/task3.py`
- 测试输出：
  ```
  TestPlanCheck passed
  TestDoingCheck passed
  TestLearningCheck passed
  plan_check OKR.md check found in source
  {"pass": true, "errors": []}
  ```
- 结论：✅ 通过

## task2: 优化 doing 重试循环的测试失败信息传递

**分析过程 (Analysis)**:
- 阅读了 `internal/executor/runner.go`：发现 `result.Error` 在 test-failed 路径只包含 `strings.Join(testResult.Errors, "; ")`，完整 `testOutput`（含 stderr/traceback）未传入 `result.Error`
- 阅读了 `internal/executor/retry.go`：发现 `output[:500] + "... (truncated)"` 硬截断，且 `testErrorFeedback` 无限累积所有历史失败（前置拼接）
- 确认 `TaskExecutionResult` 和 `RetryResult` 结构体无需新增字段，只需修改 `result.Error` 内容和 feedback 累积逻辑

**实现步骤 (Implementation)**:
1. `runner.go`：
   - test execution error 路径（line 101）：`result.Error` 改为包含完整 `testOutput`（`"test execution failed: %v\n\nFull test output:\n%s"`）
   - test did not pass 路径（line 113）：`result.Error` 改为包含完整 `testOutput`（`"test did not pass: %s\n\nFull test output:\n%s"`）
2. `retry.go`：
   - 添加 `strings` import
   - 移除 500 字符硬截断和 `Output` 的额外追加
   - 改为调用新的 `appendFailureFeedback(existing, newEntry, 2, 3000)` 函数
   - 在 `err != nil` 分支同样使用 `appendFailureFeedback`
   - 新增 `appendFailureFeedback` 函数：按 `=== Attempt ` 分割条目，保留最近 maxEntries 条，超过 maxBytes 时从尾部截断并对齐到行边界

**遇到的问题 (Issues)**:
- 无

**验证结果 (Verification)**:
- 测试命令：`go build ./... && go test ./internal/executor/... -v && go test ./... -count=1`
- 测试输出：
  ```
  ok  github.com/sunquan/rick/internal/executor  3.556s
  ok  github.com/sunquan/rick/internal/cmd       27.249s
  ... all packages PASS
  ```
- 结论：✅ 通过
