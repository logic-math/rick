# 依赖关系
task3, task8

# 任务名称

迁移 easy + dream prompt builders + 模板：移除 SPEC/OKR/wiki/tools，注入 loops_context

# 任务目标

修改 easy 和 dream 的 prompt builder + 模板，对齐 task4-6 的架构变更：

**easy (`internal/prompt/easy_prompt.go` + `templates/easy.md`)**：
- 移除 `{{okr_content}}`、`{{spec_content}}` 的注入（仅这两个，**`{{debug_content}}` 必须保留**，它是当前 session 的 debug 记录，与 SPEC/OKR 无关）
- 添加 `{{loops_context}}` 注入（`LoadLoopsContext(loopsDir)`）
- `easy.md` 模板移除"读取 OKR/SPEC"相关上下文章节，添加 loops_context 章节

**dream (`internal/prompt/dream_prompt.go` + `templates/dream.md`)**：
- 移除注入 wiki_dir、tools_dir、spec_path 相关变量（dream 不再读/写 wiki 和 tools）
- 添加 `{{loops_context}}`、`{{loops_dir}}`、`{{skills_dir}}` 注入
- `dream.md` 模板更新：dream 产出写到 `.rick/loops/` 和 `.rick/skills/`，不写 wiki/SPEC

关键约束：easy 中原有的 `{{debug_content}}`（当前 job debug 记录）保留，这是当前 session debug 上下文，不是 SPEC/OKR。

# 关键结果

1. `easy.md` 不含 `{{okr_content}}`、`{{spec_content}}`，新增 `{{loops_context}}`
2. `dream.md` 不含 `{{wiki_dir}}`、`{{tools_dir}}`、`{{spec_path}}`，新增 `{{loops_context}}`、`{{loops_dir}}`、`{{skills_dir}}`
3. `easy_prompt.go` 中 `GenerateEasyPromptFile()` 的 contextMgr 调用移除 GetOKRRaw/GetSPECRaw
4. `dream_prompt.go` 中对应注入同步更新（**先执行 `ls internal/prompt/dream_prompt.go` 确认文件存在**，若不存在则 grep dream 相关代码找到真实位置）
5. `go test ./internal/prompt/... -run TestEasy` 通过
6. `./bin/rick dream --dry-run` 输出包含 "loops_dir"，不包含 "{{wiki_dir}}"
7. `dream.md` 模板中写完候选文件后的 check 步骤只需调用 `rick tools dream_check`，loops/skills 格式校验已由 task8 集成进 dream_check，不需要单独调用
8. `dream.md` 模板中包含候选文件命名规范：写 loop 候选用 `candidate_loop_N.md`，写 skill 候选用 `candidate_skill_N.md`（与 learning 对称，N 为递增数字）

# 测试方法

1. **easy 正常路径 - dry-run**：
   - 前置条件：二进制已重建，`.rick/loops/` 存在
   - 操作：生成 easy prompt（`GenerateEasyPromptFile` 返回路径后 cat 文件）
   - 预期输出：包含 "可用的项目 Loops"，不包含 "spec_content" 或 "okr_content" 字面量

2. **dream 正常路径 - dry-run**：
   - 前置条件：binary 已重建，job_22 已有 tasks.json
   - 操作：`./bin/rick dream --dry-run 2>&1`
   - 预期输出：包含 "loops_dir" 路径文本，不包含 "wiki_dir" 或 "spec_path" 字面量

3. **easy 模板验证**：
   - 操作：`grep -c "okr_content\|spec_content" internal/prompt/templates/easy.md`
   - 预期输出：0

4. **dream 模板验证**：
   - 操作：`grep -c "wiki_dir\|tools_dir" internal/prompt/templates/dream.md`
   - 预期输出：0

5. **easy debug_content 保留验证**（边界用例 - 不该删的别删）：
   - 操作：`grep -c "debug_content\|debug" internal/prompt/templates/easy.md`
   - 预期输出：≥ 1（debug_content 变量仍在 easy 模板中，因为它是 session 级别的 debug 上下文）

6. **编译 + 单测**：
   - 操作：`./scripts/build.sh && go test ./internal/prompt/... -run TestEasy -v`
   - 预期输出：编译成功，测试全部通过

# 调试方法

遇到任何不符合预期的行为，必须加载并遵循 debug-skill：
`/Users/sunquan/ai_coding/CODING/rick/.rick/jobs/job_22/doing/prompts/skill_debug_skill.md`

执行顺序：按 debug-skill Phase 1-6 执行，每阶段达上限后升级人工协作
