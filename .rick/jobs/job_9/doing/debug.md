## task1: 重构 learning 输入：直接读取 OKR/task/debug，移除 git 历史依赖

**分析过程 (Analysis)**:
- 阅读了 `internal/cmd/learning.go`：`collectExecutionData()` 只读取 `debug.md` 和 `tasks.json`；`buildLearningPrompt()` 注入 `task_execution_results` 和 `debug_records`
- 阅读了 `internal/prompt/learning_prompt.go`：`GenerateLearningPrompt()` 调用 4 个占位函数（`formatGitHistory`、`formatNewFeatures`、`formatCodeImprovements`、`formatTechnicalDebt`），全部返回硬编码模板文本
- 阅读了 `internal/prompt/templates/learning.md`：Step 1 要求用 `git show <commit_hash>` 读取代码变更
- 设计方案：在 `ExecutionData` 新增 `OKRContent`/`TaskMDContent` 字段；`collectExecutionData()` 读取 `plan/OKR.md`（可选）和 `plan/task*.md`；`buildLearningPrompt()` 注入两个新变量；删除 4 个占位函数；更新模板 Step 1

**实现步骤 (Implementation)**:
1. `internal/cmd/learning.go`：`ExecutionData` 新增 `OKRContent string` 和 `TaskMDContent string`
2. `internal/cmd/learning.go`：`collectExecutionData()` 新增读取 `plan/OKR.md`（不存在则跳过）
3. `internal/cmd/learning.go`：`collectExecutionData()` 新增用 `filepath.Glob` 读取所有 `plan/task*.md`，拼接为带文件名标题的字符串
4. `internal/cmd/learning.go`：`buildLearningPrompt()` 注入 `okr_content` 和 `task_md_content` 变量
5. `internal/prompt/learning_prompt.go`：删除 `formatGitHistory`、`formatNewFeatures`、`formatCodeImprovements`、`formatTechnicalDebt` 四个函数及其在 `GenerateLearningPrompt()` 中的调用
6. `internal/prompt/learning_prompt_test.go`：删除对应的 4 个测试函数，更新 `TestGenerateLearningPrompt_VariableReplacement` 中的模板（移除已删除变量）
7. `internal/prompt/templates/learning.md`：新增 OKR/task_md 章节；Step 1 改为"读取上方注入的 OKR、task*.md、debug.md，按需读取源码"

**遇到的问题 (Issues)**:
- `TestGenerateLearningPrompt_VariableReplacement` 的测试模板仍包含 `{{git_history}}` 等已删除变量，导致测试失败；删除测试模板中对应变量后通过

**验证结果 (Verification)**:
- 测试命令：`go build ./... && go test ./... && grep "本周期内新增" internal/ -r`
- 测试输出：
  ```
  ok  github.com/sunquan/rick/internal/cmd
  ok  github.com/sunquan/rick/internal/prompt
  ok  github.com/sunquan/rick/internal/...（全部通过）
  grep: 无结果
  ```
- 结论：✅ 通过

## task3: 新增 tools/ 扫描机制并注入 plan/doing prompt

**分析过程 (Analysis)**:
- 阅读了 `internal/workspace/skills.go`：`LoadSkillsList()` 扫描 `.rick/skills/*.py`，结构与 tools 扫描类似
- 阅读了 `internal/prompt/doing_prompt.go`：`formatSkillsSection()` append 到文件末尾；`GenerateDoingPromptFile()` 用 variadic `rickDir` 注入 skills
- 阅读了 `internal/prompt/plan_prompt.go`：`formatSkillsIndexSection()` 返回 index.md 内容；通过模板变量 `{{skills_index}}` 注入
- 阅读了 `internal/prompt/templates/plan.md` 和 `doing.md`：plan 用模板变量，doing 用 append
- 设计方案：tools 扫描放在 `workspace/tools.go`（projectRoot/tools/*.py）；doing 沿用 append 模式；plan 沿用模板变量 `{{tools_list}}`；doing.md 不加模板变量（避免 GenerateDoingPrompt 遗漏设置）

**实现步骤 (Implementation)**:
1. `internal/workspace/tools.go`：新增 `ToolInfo` 结构体和 `LoadToolsList(projectRoot string)` 函数，扫描 `projectRoot/tools/*.py`，提取 `# Description:` 注释
2. `internal/workspace/tools_test.go`：新增 `TestLoadToolsList` 覆盖 5 个子场景
3. `internal/prompt/doing_prompt.go`：新增 `formatToolsSection(projectRoot string)`；`GenerateDoingPromptFile()` 中 `os.Getwd()` 获取 projectRoot，append tools section
4. `internal/prompt/plan_prompt.go`：新增 `formatToolsListSection()`（内部调用 `os.Getwd()`）；两个 Generate 函数均注入 `tools_list` 变量
5. `internal/prompt/templates/plan.md`：在"九、可用的项目 Skills"后新增"九.1、可用的项目 Tools"章节（`{{tools_list}}`）

**遇到的问题 (Issues)**:
- 无

**验证结果 (Verification)**:
- 测试命令：`go build ./... && go test ./internal/workspace/ -run TestLoadToolsList -v && go test ./...`
- 测试输出：
  ```
  === RUN   TestLoadToolsList
  --- PASS: TestLoadToolsList (0.01s)
  ok  	github.com/sunquan/rick/internal/workspace	0.456s
  ok  	github.com/sunquan/rick/internal/cmd	26.659s
  ok  	github.com/sunquan/rick/internal/prompt	0.783s
  ok  	github.com/sunquan/rick/internal/workspace	0.497s
  （全部通过）
  ```
- 结论：✅ 通过

## task2: 建立 skills/index.md 格式规范并重构 skills 注入机制

**分析过程 (Analysis)**:
- 阅读了 `internal/workspace/skills.go`：现有 `LoadSkillsList()` 扫描 `.py` 文件第一行注释，`GenerateSkillsREADME()` 生成 README.md
- 阅读了 `internal/prompt/doing_prompt.go`：`formatSkillsSection()` 调用 `LoadSkillsList()`，`GenerateDoingPromptFile()` 使用 variadic `rickDir` 参数注入 skills
- 阅读了 `internal/prompt/plan_prompt.go`：`GeneratePlanPrompt/File()` 无 skills 注入，无 `skills_index` 变量
- 阅读了 `internal/prompt/templates/plan.md`：无 `{{skills_index}}` 模板变量
- 阅读了 `.rick/skills/README.md`：现有 skills 描述表格
- 设计方案：新增 `LoadSkillsIndex()` 读取 index.md；重构 `formatSkillsSection()` 优先用 index.md；为 plan 新增 `formatSkillsIndexSection()` + `{{skills_index}}` 变量；创建 `.rick/skills/index.md`；保留 `GenerateSkillsREADME` 向后兼容

**实现步骤 (Implementation)**:
1. `workspace/skills.go`：新增 `LoadSkillsIndex(rickDir string) (string, error)` 读取 index.md 内容
2. `workspace/skills.go`：新增 `GenerateSkillsIndex()` 生成标准 index.md 格式（含触发场景列）
3. `workspace/skills.go`：保留 `GenerateSkillsREADME()` 作为 `GenerateSkillsIndex()` 的 alias（向后兼容）
4. `doing_prompt.go`：重构 `formatSkillsSection()` 优先读取 index.md，fallback 扫描 .py 文件
5. `plan_prompt.go`：新增 `formatSkillsIndexSection()` 辅助函数；`GeneratePlanPrompt/File()` 改为 variadic `rickDir` 参数；注入 `skills_index` 变量
6. `templates/plan.md`：在"用户需求"前新增"可用的项目 Skills"章节（`{{skills_index}}`）
7. `.rick/skills/index.md`：创建包含所有现有 skills 的 index，补充触发场景描述

**遇到的问题 (Issues)**:
- 无

**验证结果 (Verification)**:
- 测试命令：`go build ./... && go test ./internal/workspace/ -v && go test ./internal/prompt/ -v && go test ./...`
- 测试输出：
  ```
  ok  	github.com/sunquan/rick/internal/workspace	2.369s
  ok  	github.com/sunquan/rick/internal/prompt	2.210s
  ok  	github.com/sunquan/rick/internal/cmd	27.971s
  ok  	github.com/sunquan/rick/internal/config	0.292s
  ok  	github.com/sunquan/rick/internal/executor	3.983s
  ok  	github.com/sunquan/rick/internal/git	8.519s
  ok  	github.com/sunquan/rick/internal/logging	1.829s
  ok  	github.com/sunquan/rick/internal/parser	1.457s
  ok  	github.com/sunquan/rick/pkg/errors	2.009s
  ```
- 结论：✅ 通过
