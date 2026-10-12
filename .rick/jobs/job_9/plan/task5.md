# 依赖关系
task1, task2, task3, task4

# 任务名称
补全集成测试与 dry-run 输出：覆盖 job_9 所有变更的关键断言

# 任务目标
job_9 的 4 个 task 完成后，需要一套可自动运行的集成测试来 check 住所有关键改动，同时增强 plan/learning 的 dry-run 使其打印构建好的 prompt（方便人工端到端验证）。

**测试覆盖范围**（对应 task1~4 的每个关键结果）：

**task1（learning 输入重构）**：
- `buildLearningPrompt()` 生成的 prompt 包含真实 task.md 内容（任务名称、关键结果）
- `buildLearningPrompt()` 生成的 prompt 包含 OKR.md 内容（若存在）
- `buildLearningPrompt()` 生成的 prompt 不包含任何硬编码假数据字符串（"本周期内新增"、"本周期内的代码改进"等）
- `learning.md` 模板不包含 `git show` 指令

**task2（skills index）**：
- `LoadSkillsIndex()` 在 index.md 存在时返回其内容
- `LoadSkillsIndex()` 在 index.md 不存在时返回空字符串（不报错）
- doing prompt 包含 skills index 内容（当 index.md 存在时）
- plan prompt 包含 skills index 内容（当 index.md 存在时）
- doing prompt 在 index.md 不存在但有 .py 文件时，降级为扫描 .py 文件的描述

**task3（tools 注入）**：
- `LoadToolsList()` 正确扫描 `tools/*.py` 并提取 `# Description:` 注释
- `LoadToolsList()` 在 tools/ 不存在时返回空列表（不报错）
- doing prompt 包含 tools 列表章节（当 tools/*.py 存在时）
- plan prompt 包含 tools 列表章节（当 tools/*.py 存在时）
- doing/plan prompt 在 tools/ 不存在时不包含 tools 章节

**task4（job 级 OKR）**：
- plan prompt 不包含全局 `.rick/OKR.md` 的内容
- plan prompt 包含要求 Claude 生成 `job_N/plan/OKR.md` 的指令
- doing prompt 包含 job 级 OKR 内容（当 `job_N/plan/OKR.md` 存在时）
- doing prompt 在 job OKR 不存在时不报错，正常生成

**dry-run 增强**：
- `rick plan --dry-run "需求"` 打印构建好的完整 plan prompt 到 stdout
- `rick learning --dry-run job_N` 打印构建好的完整 learning prompt 到 stdout

**测试文件位置**：
- prompt 内容断言：`internal/prompt/integration_rfc001_test.go`（新文件，package prompt）
- workspace 函数断言：`internal/workspace/tools_test.go`（新文件）和 `internal/workspace/skills_test.go`（已有，补充 LoadSkillsIndex 测试）
- dry-run 行为断言：`internal/cmd/dryrun_integration_test.go`（新文件，package cmd）

# 关键结果
1. `internal/prompt/integration_rfc001_test.go` 新建，包含覆盖 task1~4 所有关键断言的测试函数，每个断言独立的 `t.Run` 子测试
2. `internal/workspace/tools_test.go` 新建，包含 `TestLoadToolsList_*` 系列测试
3. `internal/workspace/skills_test.go` 补充 `TestLoadSkillsIndex_*` 测试
4. `internal/cmd/dryrun_integration_test.go` 新建，验证 plan/learning dry-run 打印 prompt 内容
5. `plan.go` / `learning.go` 的 dry-run 分支改为打印构建好的完整 prompt
6. `go test ./...` 全部通过，新增测试无 skip

# 测试方法
1. 运行 `go test ./internal/prompt/ -run TestIntegration_RFC001 -v` 确认所有 RFC001 相关断言通过
2. 运行 `go test ./internal/workspace/ -run TestLoadToolsList -v` 确认 tools 扫描测试通过
3. 运行 `go test ./internal/workspace/ -run TestLoadSkillsIndex -v` 确认 skills index 测试通过
4. 运行 `go test ./internal/cmd/ -run TestDryRun -v` 确认 dry-run 输出测试通过
5. 运行 `go test ./...` 确认全量测试通过，无新增失败
6. 手动运行 `bin/rick plan --dry-run "测试需求"` 确认输出包含完整 prompt 内容
7. 手动运行 `bin/rick learning --dry-run job_9` 确认输出包含完整 learning prompt 内容
