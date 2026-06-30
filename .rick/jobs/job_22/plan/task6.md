# 依赖关系
task3, task8

# 任务名称

迁移 learning prompt builder + learning.md 模板：移除 wiki/tools 输出，改写为 loops/skills 产出

# 任务目标

修改 `internal/prompt/` 中 learning prompt 相关代码和 `internal/prompt/templates/learning.md`：
1. 移除注入 `{{spec_path}}`、`{{wiki_dir}}`、`{{tools_dir}}` 三个变量（learning 不再引导 agent 写 wiki 文件或 tools 脚本）
2. 添加 `{{loops_context}}` 注入（调用 `LoadLoopsContext`）
3. 添加 `{{loops_dir}}`（`.rick/loops/` 绝对路径）和 `{{skills_dir}}`（`.rick/skills/` 绝对路径）注入，供 learning agent 写候选文件使用
4. 更新 `learning.md` 模板的 7 步 SOP：
   - Step 3（原"提取可复用 Python 工具"）改为：提取可复用 skill，写入 `{{skills_dir}}/candidate_skill_{n}.md`
   - Step 4（原"创建 wiki 文档"）改为：识别新 loop 模式，写候选 `{{loops_dir}}/candidate_loop_{n}.md`
   - Step 5（原"更新 SPEC.md"）删除（不再更新 SPEC）
5. 更新 `learning.md` 中 Step 6（SUMMARY.md）格式，移除 wiki/tools 对 SPEC 注册的要求

关键代码路径：
- `internal/prompt/easy_prompt.go`：`GenerateEasyLearningPromptFile()` 函数（learning prompt 生成）；**实现前先执行 `grep -r "GenerateEasyLearningPromptFile" --include="*.go" .` 找到所有调用方，同步更新**
- 可能还有独立的 `learning_prompt.go`（如存在）
- `internal/prompt/templates/learning.md`

# 关键结果

1. `learning.md` 模板不含 `{{spec_path}}`、`{{wiki_dir}}`、`{{tools_dir}}` 变量，新增 `{{loops_dir}}`、`{{skills_dir}}`、`{{loops_context}}`
2. `learning.md` 模板 Step 3-5 改为写候选 loop/skill 文件到对应目录，不写 wiki 或 tools
3. learning prompt builder 中三个新 SetVariable 调用全部实现：
   - `builder.SetVariable("loops_context", LoadLoopsContext(loopsDir))`（loopsDir 由 `workspace.GetRickDir()` 推导）
   - `builder.SetVariable("loops_dir", loopsDir)`
   - `builder.SetVariable("skills_dir", skillsDir)`（skillsDir = `{rickDir}/skills/`）
4. `go test ./internal/prompt/... -run TestLearning` 全部通过（或相关 embedded 测试通过）
5. `./bin/rick learning --job job_22 --dry-run 2>&1` 包含 "loops_dir" 和 "skills_dir" 路径，不包含 "wiki_dir" 或 "spec_path"
6. `learning.md` 模板中写完候选文件后的 check 步骤只需调用 `rick tools learning_check {{job_id}}`，不需要单独调用 loops/skills check（task8 已将其集成进 learning_check）

# 测试方法

1. **正常路径 - dry-run 验证新变量**：
   - 前置条件：job_22 doing 目录存在；`.rick/loops/` 和 `.rick/skills/` 存在（task1 已完成）；二进制重新构建
   - 操作：`./bin/rick learning --job job_22 --dry-run 2>&1`
   - 预期输出：包含 "loops_dir" 和 "skills_dir" 路径，包含 "可用的项目 Loops"，不包含 "{{wiki_dir}}" 或 "{{spec_path}}" 字面量

2. **模板内容验证**：
   - 操作：`grep -c "wiki_dir\|tools_dir\|spec_path" internal/prompt/templates/learning.md`
   - 预期输出：0（旧变量已完全移除）

3. **新变量存在性验证**：
   - 操作：`grep -c "loops_dir\|skills_dir\|loops_context" internal/prompt/templates/learning.md`
   - 预期输出：≥ 3（三个新变量均在模板中存在）

4. **边界用例 - 候选文件命名验证**：
   - 操作：确认 `learning.md` 中包含 `candidate_loop_` 或 `candidate_skill_` 前缀说明
   - 预期输出：`grep -c "candidate_loop\|candidate_skill" internal/prompt/templates/learning.md` ≥ 2

5. **异常路径 - 编译验证**：
   - 操作：`./scripts/build.sh`
   - 预期输出：编译成功，无未使用变量或未定义变量错误

# 调试方法

遇到任何不符合预期的行为，必须加载并遵循 debug-skill：
`/Users/sunquan/ai_coding/CODING/rick/.rick/jobs/job_22/doing/prompts/skill_debug_skill.md`

执行顺序：按 debug-skill Phase 1-6 执行，每阶段达上限后升级人工协作
