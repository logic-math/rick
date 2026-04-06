# 依赖关系


# 任务名称
重构 learning 输入：直接读取 OKR/task/debug，移除 git 历史依赖

# 任务目标
当前 `learning.go` 的 `collectExecutionData()` 只读取 `debug.md` 和 `tasks.json`，而 `learning_prompt.go` 的 `buildLearningPrompt()` 中存在大量返回假数据的占位函数（`formatGitHistory`、`formatNewFeatures`、`formatCodeImprovements`、`formatTechnicalDebt`），这些函数硬编码了无意义的模板文本。同时 `learning.md` 模板的 Step 1 要求 agent 用 `git show` 读取代码变更，效率低且噪声多。

本任务目标：让 learning 的输入直接来自 `OKR.md`（job 级，plan 目录下）、`task*.md`（plan 目录下）、`debug.md`（doing 目录下），遵循"渐进式披露"原则——agent 拿到这三份文档后自行判断是否需要进一步读取源码，不再强制 git 历史。

# 关键结果
1. `collectExecutionData()` 新增读取 `plan/` 目录下所有 `task*.md` 文件内容，汇总为字符串注入 prompt
2. `collectExecutionData()` 新增读取 `plan/OKR.md`（job 级，若不存在则跳过，不报错）
3. `buildLearningPrompt()` 删除 `formatGitHistory`、`formatNewFeatures`、`formatCodeImprovements`、`formatTechnicalDebt` 四个占位函数及其调用，改为注入真实的 task 内容和 OKR 内容
4. `learning.md` 模板 Step 1 删除 `git show <commit_hash>` 指令，改为"读取上方注入的 OKR、task.md、debug.md，按需读取源码"
5. `go test ./internal/...` 全部通过

# 测试方法
1. 运行 `go build ./...` 确认编译通过
2. 运行 `go test ./internal/cmd/ -run TestLearning -v` 确认 learning 相关测试通过
3. 运行 `go test ./internal/prompt/ -run TestLearning -v` 确认 prompt 相关测试通过
4. 运行 `go test ./...` 确认全量测试通过
5. 手动检查 `buildLearningPrompt()` 不再包含任何硬编码的假数据字符串（grep "本周期内新增" 应无结果）
