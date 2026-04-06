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
