## task3: 创建 core-skills 内嵌目录，按 SOP 阶段精准注入对应 skill

**分析过程 (Analysis)**:
- 阅读了 `internal/prompt/manager.go`：发现 embed 模式是 8 个独立 string 变量，使用 `_ "embed"` 空白导入
- 确认无 `templateFS embed.FS`，需要新增 `skillsFS embed.FS` 用于目录嵌入
- 阅读了 `plan_prompt.go`、`doing_prompt.go`、`learning_prompt.go` 确认现有注入模式
- 发现 `buildTestGenerationPromptFile` 在 `executor/runner.go`，`buildLearningPrompt` 在 `cmd/learning.go`
- 无 dream 命令，决定创建 `dream_prompt.go` stub 以满足 KR4 要求
- 核实 test3.py 期望的 skill 文件结构为 flat 格式（`skills/sense.md`），而非 nested（`skills/sense/skill.md`）

**实现步骤 (Implementation)**:
1. 创建 `internal/prompt/templates/skills/` 目录及 `tdd/` 子目录
2. 创建 7 个 flat skill 文件：`sense.md`, `tc.md`, `tdd.md`, `testing.md`, `debug.md`, `gen-skill.md`, `evolve-skills.md`
3. 创建 1 个 nested skill 文件：`tdd/testing-anti-patterns.md`
4. 更新 `manager.go`：`_ "embed"` → `"embed"`，新增 `skillsFS embed.FS`，添加 `log` + `strings` 导入，实现 `LoadCoreSkills(names []string) string`
5. 更新 `plan_prompt.go` (`GeneratePlanPromptFile`)：追加 `LoadCoreSkills([]string{"sense", "tc"})` 到 prompt 文件
6. 更新 `doing_prompt.go` (`GenerateDoingPromptFile`)：追加 `LoadCoreSkills([]string{"tdd", "tdd/testing-anti-patterns", "debug"})`
7. 更新 `executor/runner.go` (`buildTestGenerationPromptFile`)：在写入前追加 `LoadCoreSkills([]string{"tdd", "testing", "tc"})`
8. 更新 `cmd/learning.go` (`buildLearningPrompt`)：拼接 `LoadCoreSkills([]string{"gen-skill"})` 到 prompt 字符串
9. 创建 `internal/prompt/dream_prompt.go` 实现 `GenerateDreamPromptFile`，注入 `LoadCoreSkills([]string{"sense", "evolve-skills"})`
10. 在 `manager_test.go` 新增 `TestCoreSkillsEmbed` 单元测试（4 个子测试）

**遇到的问题 (Issues)**:
- task3.md 描述 nested 结构（`skills/sense/skill.md`），但 test3.py 期望 flat 结构（`skills/sense.md`）—— 以 test3.py 为准，采用 flat 结构
- `plan_prompt.go` 需要调用 `readAndAppend`（定义在 `doing_prompt.go`），同包可直接调用，无需重复定义

**验证结果 (Verification)**:
- 测试命令：`python3 /Users/sunquan/ai_coding/CODING/rick/.rick/jobs/job_14/doing/tests/task3.py`
- 测试输出：
  ```
  {"pass": true, "errors": []}
  ```
- TestCoreSkillsEmbed 命令：`go test ./internal/prompt/... -run TestCoreSkillsEmbed -v`
- 测试输出：
  ```
  --- PASS: TestCoreSkillsEmbed (0.00s)
      --- PASS: TestCoreSkillsEmbed/all_eight_skill_files_non_empty (0.00s)
      --- PASS: TestCoreSkillsEmbed/tdd_testing_anti_patterns_slash_path (0.00s)
      --- PASS: TestCoreSkillsEmbed/multi_skill_contains_separator (0.00s)
      --- PASS: TestCoreSkillsEmbed/nonexistent_skill_no_panic (0.00s)
  ```
- 全量测试：`go test ./...` 全部 PASS，无新增失败
- 结论：✅ 通过

## task1: 定义 AgentSession/AgentExecutor 稳定接口与 act-path 生成器

**分析过程 (Analysis)**:
- 确认 `internal/agent/` 和 `internal/actpath/` 目录不存在，需新建
- 无 claudecode 包，无需处理依赖隔离问题（actpath 只依赖 agent 接口）
- 梳理测试断言：RawLogPath="/tmp/raw_session.log"，basename="raw_session.log"，FinalMessageLine=42 → 输出应含 "raw_session.log:42"
- 任务要求 FinalMessage 截断到 ≤200 字符，使用 []rune 处理 Unicode 安全

**实现步骤 (Implementation)**:
1. 创建 `internal/agent/interface.go`：定义 `ToolCall` struct、`AgentSession` 接口（ID/Duration/ToolCalls/FinalMessage/FinalMessageLine/RawLogPath）、`AgentExecutor` 接口
2. 创建 `internal/actpath/generator.go`：实现 `Generate(session AgentSession, outputFile string) error`
   - `os.MkdirAll(filepath.Dir(outputFile), 0755)` 自动创建目录
   - 输出三个 Markdown 节：执行摘要 / 行为轨迹 / Agent 最终输出
   - FinalMessage 用 []rune 截断到 200 字符
   - 行号链接格式 `[L{n}]({rawLogPath}:{n})`
   - Agent 最终输出尾注：`> [{base}:{finalLine}]({rawLogPath})`
3. 创建 `internal/actpath/generator_test.go`：
   - `mockSession` struct 实现 `AgentSession` 接口
   - `TestGenerate_Format`：2 ToolCall（1 IsError），含截断子测试
   - `TestGenerate_EmptyToolCalls`：零 ToolCall，仅表头
   - `TestGenerate_CreatesDir`：嵌套目录自动创建

**遇到的问题 (Issues)**:
- 无

**验证结果 (Verification)**:
- 编译：`go build ./internal/agent/... ./internal/actpath/...` → 无报错
- 接口隔离：`grep -r "claudecode" internal/actpath/` → 空（exit 1 = 无匹配）
- 单元测试：`go test ./internal/actpath/... -v`
  ```
  --- PASS: TestGenerate_Format (0.01s)
  --- PASS: TestGenerate_EmptyToolCalls (0.00s)
  --- PASS: TestGenerate_CreatesDir (0.00s)
  PASS
  ok  	github.com/sunquan/rick/internal/actpath	0.488s
  ```
- task1.py：`python3 .rick/jobs/job_14/doing/tests/task1.py` → `{"pass": true, "errors": []}`
- 全量测试：`go test ./...` 全部 PASS，无新增失败
- 结论：✅ 通过
