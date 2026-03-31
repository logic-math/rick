## task4: 为 `rick plan` 命令增加 `--job` flag 支持重进已有 job

**分析过程 (Analysis)**:
- 阅读了 `internal/cmd/plan.go`，了解现有 `RunE` 逻辑：直接调用 `executePlanWorkflow`
- 阅读了 `internal/cmd/root.go`，确认 `--job` 已通过 `PersistentFlags()` 定义为全局 flag，`GetJobID()` 可直接读取
- 阅读了 `internal/workspace/paths.go`，确认 `GetJobPlanDir(jobID)` 可获取已有 job 的 plan 目录
- 关键约束：不能在 `plan.go` 中重复定义 `--job` flag，否则 cobra 会 panic；直接复用全局 flag

**实现步骤 (Implementation)**:
1. 在 `plan.go` 的 `RunE` 开头插入 `GetJobID()` 检查：非空走"重进"路径，空走原有路径
2. dry-run 模式下，若指定 `--job`，输出 `[DRY-RUN] Would re-enter plan for job: <id>` 并退出
3. 新增 `reEnterPlanWorkflow(existingJobID string)` 函数：检查 plan 目录是否存在，不存在返回明确错误；存在则复用现有目录，重新生成 prompt 并启动 Claude 交互
4. 在 `plan_test.go` 中新增两个测试：`TestPlanCmdWithJobFlagDryRun`（dry-run 下 --job 输出正确）、`TestReEnterPlanWorkflow_NonExistentJob`（不存在 job 返回正确错误）

**遇到的问题 (Issues)**:
- shell CWD 持续重置导致 `go build ./...` 报错（directory prefix does not contain main module），通过 `go -C <abs_path> build ./...` 解决

**验证结果 (Verification)**:
- 测试命令：`go -C /opt/meituan/dolphinfs_sunquan20/ai_coding/Coding/rick test -timeout 30s -v -run "TestPlanCmd|TestReEnterPlan|TestGenerateJobID" ./internal/cmd/...`
- 测试输出：
  ```
  === RUN   TestPlanCmdCreation
  --- PASS: TestPlanCmdCreation (0.00s)
  === RUN   TestGenerateJobID
  --- PASS: TestGenerateJobID (0.00s)
  === RUN   TestPlanCmdWithDryRun
  [DRY-RUN] Would create a plan
  --- PASS: TestPlanCmdWithDryRun (0.00s)
  === RUN   TestPlanCmdWithEmptyRequirement
  --- PASS: TestPlanCmdWithEmptyRequirement (0.00s)
  === RUN   TestPlanCmdWithJobFlagDryRun
  [DRY-RUN] Would re-enter plan for job: job_1
  --- PASS: TestPlanCmdWithJobFlagDryRun (0.00s)
  === RUN   TestReEnterPlanWorkflow_NonExistentJob
  --- PASS: TestReEnterPlanWorkflow_NonExistentJob (0.00s)
  PASS
  ok  	github.com/sunquan/rick/internal/cmd	0.027s
  ```
- 结论：✅ 通过

## task2: 实现 human-loop 命令的 prompt 生成函数和 RFC 目录管理

**分析过程 (Analysis)**:
- 阅读了 `internal/prompt/doing_prompt.go` 了解现有 prompt 生成函数的模式
- 阅读了 `internal/prompt/manager.go` 了解模板加载机制（支持嵌入模板）
- 阅读了 `internal/workspace/paths.go` 了解路径函数风格（均使用 `GetRickDir()` 作为基础）
- 发现 `NewPromptManager` 签名为 `NewPromptManager(templateDir string)`，但 task2.py 测试脚本调用 `prompt.NewPromptManager()`（无参数），需要改为可变参数
- 选择将 `NewPromptManager` 改为 variadic 函数，保持向后兼容

**实现步骤 (Implementation)**:
1. 在 `internal/workspace/paths.go` 末尾追加 `GetRFCDir()` 函数，返回 `.rick/RFC` 绝对路径
2. 创建 `internal/prompt/templates/human_loop.md` 模板，包含 `{{topic}}` 和 `{{rfc_dir}}` 变量
3. 在 `internal/prompt/manager.go` 中添加 `human_loop.md` 的 embed 声明，并在 `getEmbeddedTemplate` 中注册
4. 将 `NewPromptManager(templateDir string)` 改为 `NewPromptManager(templateDir ...string)` 以支持无参调用
5. 创建 `internal/prompt/human_loop_prompt.go`，实现 `GenerateHumanLoopPromptFile(topic, rfcDir string, manager *PromptManager) (string, error)`
6. 在 `internal/workspace/workspace_test.go` 中添加 `TestGetRFCDir` 测试，并补充 `strings` import

**遇到的问题 (Issues)**:
- task2.py 测试脚本路径计算有误（多了一层 `..`），导致 project_root 指向父目录，无法直接通过脚本测试；已通过手动运行 go 命令验证
- `NewPromptManager` 原为必须传参，但 task2.py 动态生成的测试代码调用无参版本，需改为 variadic

**验证结果 (Verification)**:
- 测试命令：`go build ./...` 及 `go test ./internal/prompt/... ./internal/workspace/...`
- 测试输出：
  ```
  BUILD OK
  ok  	github.com/sunquan/rick/internal/prompt	0.029s
  ok  	github.com/sunquan/rick/internal/workspace	0.015s
  ```
- 结论：✅ 通过
