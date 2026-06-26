# 依赖关系
task1

# 任务名称
更新 plan.md 模板 + plan_prompt.go：移除 sense 追问，注入 grilling skill

# 任务目标
1. 从 `internal/prompt/templates/plan.md` 中删除"三、思考方法"section（sense S→E→N 分析块，约 20 行），删除 `{{sense_skill_path}}` 变量所有引用
2. 简化 12 步 SOP：删除 Step 1（还原问题，sense S）、Step 7（系统化思考 E→N），在合适位置（OKR/SPEC check 和探索项目之后）添加 grilling 步骤（引用 `{{grilling_skill_path}}`），重新连续编号
3. 在 `internal/prompt/plan_prompt.go` 中：
   - `GeneratePlanPromptFile`：删除 `senseFile` 写出逻辑，新增 `grillingFile := WriteSkillFile(promptsDir, "skill_grilling.md", "grilling")`，将 `sense_skill_path` SetVariable 替换为 `grilling_skill_path`
   - `GeneratePlanPrompt`（dry-run）：移除 `sense_skill_path` placeholder，新增 `grilling_skill_path` 占位符 `<tmp>/rick-plan-skill-grilling-*.md`

# 重要约束
- **不删除 `internal/prompt/templates/skills/sense.md` 文件本身**：sense skill 在 doing/easy 中仍被 debug_skill 依赖；本 task 只删除 plan.md 中的 sense 引用和 plan_prompt.go 中的 senseFile 写入逻辑
- `GeneratePlanPromptFile` 的返回值 `(promptFile, nil, nil)` 保持不变，无需修改 skillFiles

# 关键结果
1. `plan.md` 中无 `{{sense_skill_path}}` 字符串（grep 验证为空）
2. `plan.md` SOP 包含 grilling 步骤，内容引用 `{{grilling_skill_path}}`，步骤编号连续
3. `plan_prompt.go::GeneratePlanPromptFile` 中有 `WriteSkillFile(..., "skill_grilling.md", "grilling")` 调用
4. `plan_prompt.go::GeneratePlanPrompt` 中 `grilling_skill_path` 设置为占位符，无 `sense_skill_path`

# 测试方法
1. 正常路径（dry-run 验证）：
   - 前置条件：task1 完成（grilling.md 存在），`./scripts/build.sh` 构建成功
   - 输入：`./bin/rick plan --dry-run`
   - 操作：执行命令并检查输出
   - 预期输出：stdout 包含 `skill_grilling` 字样，不含 `sense_skill_path` 字样，不含 `{{grilling_skill_path}}` 未替换占位符
2. 边界用例：
   - 前置条件：plan.md 已删除 sense 引用并添加 grilling 步骤；plan_prompt.go 已更新
   - 输入：`grep -n 'sense_skill_path' internal/prompt/templates/plan.md internal/prompt/plan_prompt.go`
   - 操作：执行 grep，再对 grilling_skill_path 做正面验证
   - 预期输出：sense_skill_path 无匹配（退出码 1）；`grep -c 'grilling_skill_path' internal/prompt/templates/plan.md internal/prompt/plan_prompt.go` 每个文件至少有 1 处匹配
3. 异常路径（模板变量完整性）：
   - 输入：`./bin/rick tools plan_check job_18`（构建后运行格式检查）
   - 预期输出：通过，退出码 0

# 调试方法
遇到任何不符合预期的行为，必须加载并遵循 debug-skill：
`/Users/sunquan/ai_coding/CODING/rick/.rick/jobs/job_18/doing/prompts/skill_debug_skill.md`

执行顺序：按 debug-skill 三阶段调试法（源码推理法→增量调试法→科学实验法）执行，每阶段达上限后升级人工协作
