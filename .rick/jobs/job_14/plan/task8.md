# 依赖关系

task4, task6

# 任务名称

升级 doing 双 agent 提示词为红绿 TDD SOP，实现 RED 验证逻辑

# 任务目标

在 task4（Cialdini 框架）和 task6（runner 接线完成）的基础上，将双 agent 流程升级为标准红绿 TDD 架构，并在 runner.go 中实现 RED 验证。

依赖 task4（doing.md 已有 Cialdini 框架，task8 在其上添加 TDD 专项指令）；依赖 task6（runner.go 接线完成，task8 在其上添加 RED 验证逻辑）。

**双 agent 职责：**
- **testing agent（红阶段）**：`test_python.md` → skill:tdd + skill:tc + skill:testing，生成脚本后立即验证 RED
- **coding agent（绿阶段）**：`doing.md` → skill:tdd + skill:debug，遭遇 bug 强制走 systematic-debugging

# 关键结果

1. **`tools/check_prompt_variables.py` 新增 `"testing"` phase**：
   - `--phase` choices 追加 `"testing"`，映射到 `test_python.md` dry-run 输出

2. **`test_python.md` 更新**：
   - 新增模板变量 `{{okr_content}}`、`{{spec_content}}`、`{{debug_content}}`
   - 开头承诺：`YOU MUST declare: "I will use skill:tdd and skill:tc for test generation."`
   - RED 验证指令：生成脚本后运行 `python3 <test_script>`，确认输出含 `"pass": false`；若为 `true` 则 MUST 重新生成（最多 2 次）

3. **`doing.md` 更新**（在 task4 Cialdini 基础上追加）：
   - 开头承诺：`YOU MUST declare: "I will use skill:tdd for implementation."`
   - TDD 铁律 section：先红→再绿→再重构，不得跳过
   - debug 指令：`When encountering ANY bug, YOU MUST declare: "I will use skill:debug." No random fixes. No exceptions.`

4. **`runner.go` 的 `buildTestGenerationPromptFile` 使用 variadic `TestGenContext`**：
   ```go
   type TestGenContext struct {
       OKRContent   string
       SpecContent  string
       DebugContent string
   }
   func (tr *TaskRunner) buildTestGenerationPromptFile(
       task *parser.Task, testScriptPath string, ctx ...TestGenContext,
   ) (string, error)
   ```
   `runner_test.go` 旧调用 `buildTestGenerationPromptFile(task, path)` **零修改**（variadic 向后兼容）

5. **`runner.go` 的 `RunTask` 新增 RED 验证**：
   - testing agent 完成后立即调用 `ExecuteTestScript`
   - 若 `pass==true`（意外绿态）→ 写入 debug.md 警告 + 重新触发 test generation（最多 2 次重试）
   - 若 2 次重试后仍为 `pass==true` → 继续执行（记录 warn，不阻断）

# 测试方法

1. 编译：`python3 tools/build_and_get_rick_bin.py`
2. runner 向后兼容：`go test ./internal/executor/... -v`，所有旧测试通过（buildTestGenerationPromptFile 旧调用无需改动）
3. **RED 验证逻辑单元测试**（最重要的新行为，`runner_test.go` 中新增）：
   - `TestRunTask_REDPass_TriggersRetry`：mockAgentExecutor 第一次 Execute 让 ExecuteTestScript 返回 `pass=true`（意外绿态），验证：
     - debug.md 中写入含 "RED" 或 "unexpected pass" 的警告
     - testing agent 被重新调用（Execute 调用次数 ≥ 2）
   - `TestRunTask_REDPass_MaxRetry`：连续 2 次 `pass=true`，第 3 次无论结果都继续执行，不无限循环
   - `TestRunTask_REDFail_Normal`：`pass=false`（正常 RED），testing agent 只调用 1 次，无警告
   ```
   go test ./internal/executor/... -v -run TestRunTask_RED
   ```
4. testing agent TDD skill：`python3 tools/check_prompt_variables.py --phase testing --keywords "skill:tdd"`，`{"pass": true}`
5. testing agent RED 指令：`python3 tools/check_prompt_variables.py --phase testing --keywords "pass.*false"` 或 `--keywords "RED phase"`，确认 RED 验证指令存在
6. coding agent debug skill：`python3 tools/check_prompt_variables.py --phase doing --keywords "skill:debug"`，`{"pass": true}`
7. 无 gen-skill 污染：`python3 tools/check_prompt_variables.py --phase doing --keywords "gen-skill"` → "关键词未找到"
8. 完整测试：`go test ./...`，无新增失败
