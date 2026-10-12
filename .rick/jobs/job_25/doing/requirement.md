修复 debug skill 丢失的问题
---

## Grilling 澄清结论（2026-07-03）

**根因**：`debug_skill.md` 模板存在但从未被写出到 prompts 目录；`doing_loop.md` 引用了 `skill:debug-skill` 但无具体路径。

**修改范围**：easy + doing（不含 plan）

**具体变更**：

1. `internal/prompt/templates/skills/doing_loop.md` — 在 `"I will use skill:debug-skill."` 后补加 `` `{{debug_skill_path}}` ``

2. `doing_prompt.go:loadDoingLoopContent()` — 签名改为 `(domainDir, debugSkillPath string)`，内部替换 `{{debug_skill_path}}`

3. `easy_prompt.go:GenerateEasyPromptFile` — 调用 `WriteSkillFile(promptsDir, "skill_debug_skill.md", "debug_skill")`，路径传给 `loadDoingLoopContent()`，加入 `skillFiles`

4. `doing_prompt.go:GenerateDoingPromptFile` + `GenerateDoingPrompt` — 同上，写文件并传路径

5. `internal/prompt/templates/plan.md` — 移除 `# 调试方法` 章节（含 `{{debug_skill_path}}`）

6. `plan_prompt.go` — 移除所有 `debug_skill_path` 变量引用（`GeneratePlanPromptFile` 和 `GeneratePlanPrompt` 两处）
