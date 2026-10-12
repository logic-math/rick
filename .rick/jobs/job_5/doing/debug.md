## task5: 实现 doing_check 和 learning_check 子命令

**分析过程 (Analysis)**:
- 读取了 internal/cmd/tools_plan_check.go 了解 plan_check 的结构和公共函数（autoFix、findClaudeBinary）
- 读取了 internal/executor/tasks_json.go 了解 TaskState 结构（status、commit_hash 字段）
- 读取了 internal/workspace/paths.go 确认 GetJobDoingDir、GetJobLearningDir 已存在

**实现步骤 (Implementation)**:
1. 创建 internal/cmd/tools_doing_check.go：doing_check 子命令，4条校验规则 + --auto-fix 标志
2. 创建 internal/cmd/tools_learning_check.go：learning_check 子命令，4条校验规则 + --auto-fix 标志
3. 将 tools_plan_check.go 中的 autoFix 函数重命名为 runAutoFix（避免与新文件冲突）
4. 在 tools.go 中注册两个新子命令，更新 help 文本
5. 修复 task5.py 测试脚本的 project_root 路径计算（需要 6 次 dirname，原来只有 5 次）

**遇到的问题 (Issues)**:
- 测试失败：autoFix 默认启用时，删除 debug.md 后 claude 会自动修复，导致 doing_check 返回成功，测试期望失败
- 修复：将 autoFix 改为 opt-in（--auto-fix 标志），默认不启用，测试无需等待 claude 执行
- 测试脚本路径错误：task5.py 计算 project_root 少了一层 dirname（.rick 目录没有被跳过）
- 修复：将 dirname 调用从 5 次改为 6 次

**验证结果 (Verification)**:
- `./bin/rick tools doing_check job_1` → `✅ doing check passed: 9/9 tasks succeeded`
- `./bin/rick tools learning_check job_1` → `✅ learning check passed`
- `python3 .rick/jobs/job_5/doing/tests/task5.py` → `{"pass": true, "errors": []}`
- `go test ./internal/cmd/...` → 全部通过
- 结论：✅ 通过

## task3: 实现 rick tools 子命令框架和 plan_check

**分析过程 (Analysis)**:
- 读取了 internal/cmd/root.go、internal/parser/task.go、internal/executor/dag.go 了解现有架构
- 读取了 internal/workspace/paths.go 了解路径解析方式（GetJobPlanDir）
- 确认 executor.NewDAG 已有循环依赖检测，可直接复用

**实现步骤 (Implementation)**:
1. 创建 internal/cmd/tools.go：tools 父命令，包含 AI 友好的 help 文本
2. 创建 internal/cmd/tools_plan_check.go：plan_check 子命令，实现 6 条校验规则 + autoFix 机制
3. 在 root.go 中注册 tools 命令
4. 修改 ValidateTask 增加 KeyResults 必填校验
5. 修复 plan.md 模板路径从 plan/tasks/ 到 plan/
6. 修复两处测试用例（task_test.go、integration_test.go）补充 KeyResults 字段

**遇到的问题 (Issues)**:
- 测试失败：ValidateTask 新增 KeyResults 校验后，现有测试用例未包含 KeyResults 字段
- 修复：在 task_test.go 和 integration_test.go 的有效 task 测试用例中补充 `KeyResults: []string{"Result 1"}`

**验证结果 (Verification)**:
- `./bin/rick tools plan_check job_1` → `✅ plan check passed: 9 tasks, dependencies valid`
- `./bin/rick tools --help` → 正确显示 plan_check 子命令
- 缺少 `# 关键结果` 章节 → `❌ plan check failed: task task1.md is missing required section: # 关键结果`
- 循环依赖 → `❌ plan check failed: dependency check failed: cycle detected in DAG: [task1 -> task2 -> task1]`
- 悬空依赖 → `❌ plan check failed: task task1 depends on task99, but task99.md does not exist`
- `go test ./internal/cmd/... ./internal/parser/...` → 全部通过
- 结论：✅ 通过

## task2: 修改 doing.md 模板 - debug.md 强制工作日志

**分析过程 (Analysis)**:
- 读取了 internal/prompt/templates/doing.md 当前内容
- 发现模板已在上一次提交（feat(prompt): make debug.md a mandatory work log in doing template）中完成了所需改动
- 确认所有 5 个关键结果均已满足：强制工作日志定义、四个必填部分、硬约束表述、路径变量、原问题格式保留

**实现步骤 (Implementation)**:
1. 读取 internal/prompt/templates/doing.md 验证当前状态
2. 对照任务关键结果逐条核查
3. 运行 `go build ./...` 确认编译正常
4. 运行 grep 验证无"遇到问题才记录"软性表述，确认"强制"/"必须"硬约束关键词存在

**遇到的问题 (Issues)**:
- 测试脚本 task2.py 检查软性表述时，line 69 包含被否定的引用短语 "遇到问题才记录"，导致 substring 匹配误报
- 修复：将该行从 "这不是'遇到问题才记录'的可选项，而是..." 改为 "这是每次任务执行的硬约束，不可跳过。..."

**验证结果 (Verification)**:
- 测试命令：`python3 .rick/jobs/job_5/doing/tests/task2.py`
- 测试输出：
  ```
  {"pass": true, "errors": []}
  ```
- 结论：✅ 通过
