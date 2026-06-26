# 依赖关系
task1

# 任务名称
更新 easy.md 模板 + easy_prompt.go：注入 grilling skill，添加 requirement.md 追加指令

# 任务目标
1. 在 `internal/prompt/templates/easy.md` 中添加 grilling 步骤：在"项目上下文"加载完成后、正式开始工作之前，先执行 grilling 追问（引用 `{{grilling_skill_path}}`）
2. 在 easy.md 中添加明确指令：grilling 结束后将澄清结论**追加**到 `{{doing_dir}}/requirement.md`（禁止覆写，保留原始用户输入）
3. 在 `internal/prompt/easy_prompt.go::GenerateEasyPromptFile` 中新增 `grillingFile := WriteSkillFile(promptsDir, "skill_grilling.md", "grilling")` 并 `SetVariable("grilling_skill_path", grillingFile)`

注意：easy_prompt.go 中已有 senseFile 逻辑（供 debug_skill 使用），grilling 是**新增**，不替换 sense。修改后 skillFiles = `[]string{tddFile, debugSkillFile, senseFile, grillingFile}`。

# 关键结果
1. `easy.md` 包含 grilling 步骤，明确引用 `{{grilling_skill_path}}`
2. `easy.md` 包含"grilling 结束后追加 requirement.md"的指令（含"禁止覆写"说明）
3. `easy_prompt.go` 中有 `WriteSkillFile(..., "skill_grilling.md", "grilling")` 调用，`skillFiles` slice 包含 grillingFile
4. 生成的 `doingDir/prompts/skill_grilling.md` 文件实际存在（可通过集成测试或 stat 验证）

# 测试方法
1. 正常路径（生成文件验证）：
   - 前置条件：task1 完成，`./scripts/build.sh` 构建成功，存在测试用 job 目录
   - 操作：在测试环境运行 `prompt.GenerateEasyPromptFile(jobID, "test requirement", rickDir, "")` 
   - 预期输出：返回的 mainFile 内容包含 `skill_grilling.md` 路径，`doingDir/prompts/skill_grilling.md` 文件存在
2. 边界用例（requirement.md 追加不覆写）：
   - 前置条件：easy.md 已修改含 grilling + requirement.md 追加指令，`./scripts/build.sh` 构建完成
   - 输入参数：调用 `GenerateEasyPromptFile(jobID, "原始需求文本", rickDir, "")`，doingDir/requirement.md 已预写"原始需求文本"
   - 操作序列：读取 requirement.md 初始内容 → 调用函数 → 读取 requirement.md 新内容 → 对比
   - 预期输出：requirement.md 包含原始"原始需求文本"（未被删除），且 easy_prompt.md 中包含"追加"字样的指令（`grep "追加" easy.md` 有匹配，`grep "覆写\|重写" easy.md` 无匹配）
3. 异常路径：
   - 前置条件：grilling.md 不存在（模拟 embed 失败）
   - 输入：`GenerateEasyPromptFile` 调用
   - 预期输出：返回包含 "grilling" 的 error，mainFile 为空字符串

# 调试方法
遇到任何不符合预期的行为，必须加载并遵循 debug-skill：
`/Users/sunquan/ai_coding/CODING/rick/.rick/jobs/job_18/doing/prompts/skill_debug_skill.md`

执行顺序：按 debug-skill 三阶段调试法（源码推理法→增量调试法→科学实验法）执行，每阶段达上限后升级人工协作
