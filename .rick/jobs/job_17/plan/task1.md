# 依赖关系


# 任务名称
合并 tc.md 内容到 tdd-zh.md 并删除死代码 skill 文件

# 任务目标
将 `internal/prompt/templates/skills/tc.md` 的"测试用例四要素"内容无损合并进 `tdd-zh.md`，然后删除 `tc.md`、`tdd.md`、`tdd/testing-anti-patterns.md` 三个从未被注入的 skill 文件，保持编译和测试通过。注：模板文件属于 embed.FS，变更后需 `./scripts/build.sh` 重新构建才能验证运行时二进制行为。

# 关键结果
1. `tdd-zh.md` 末尾新增"测试用例四要素"章节，内容来自 `tc.md`，无删减
2. `tc.md`、`tdd.md`、`tdd/testing-anti-patterns.md` 三个文件被 `git rm` 删除
3. `internal/prompt/manager_test.go` 同步更新（此三处修改必须作为一个整体完成，不可在修改测试之前单独运行 `go test`）：① 从 `all_eight_skill_files_non_empty` 的 skills slice 中删除 `"tc"`、`"tdd"`；② 删除 `tdd_testing_anti_patterns_slash_path` 整个子测试；③ 将 `multi_skill_contains_separator` 中的 `["sense", "tc"]` 改为 `["sense", "tdd-zh"]`（或其他两个均存在的 skill）
4. `go test ./internal/prompt/...` 通过，无编译错误
5. `go build ./...` 通过

# 测试方法
1. **正常路径 — 合并内容验证（含章节结构完整性）**
   - 前置条件：`tdd-zh.md` 已修改，`tc.md` 已删除
   - 操作 A：`grep -n "测试用例四要素\|前置条件\|输入参数\|操作序列\|预期输出\|INSUFFICIENT_FUNDS" internal/prompt/templates/skills/tdd-zh.md`；预期：至少 5 行匹配
   - 操作 B：`grep -c "^### " internal/prompt/templates/skills/tdd-zh.md`；预期：比合并前多 4 个（对应四要素的四个 `###` 子节标题）

2. **死代码文件已删除**
   - 前置条件：执行删除操作后
   - 操作：`ls internal/prompt/templates/skills/`
   - 预期输出：无 `tc.md`、`tdd.md`；`tdd/` 目录下无 `testing-anti-patterns.md`（`tdd-zh.md` 和 `testing-anti-patterns-zh.md` 保留）

3. **测试无断裂**
   - 前置条件：删除完成后
   - 操作：`go test ./internal/prompt/...`
   - 预期输出：PASS，exit code 0

4. **manager_test.go 已更新，无残留引用**
   - 操作：`grep -n '"tc"\|"tdd"\|tdd/testing-anti-patterns"' internal/prompt/manager_test.go`
   - 预期输出：无输出（已删除的三个 skill 不再被测试引用）

# 调试方法
遇到任何不符合预期的行为，必须加载并遵循 debug-skill：
`/Users/sunquan/ai_coding/CODING/rick/.rick/jobs/job_17/doing/prompts/skill_debug_skill.md`

执行顺序：按 debug-skill 三阶段调试法（源码推理法→增量调试法→科学实验法）执行，每阶段达上限后升级人工协作
