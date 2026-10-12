# 依赖关系


# 任务名称

创建 core-skills 内嵌目录，按 SOP 阶段精准注入对应 skill

# 任务目标

在 `internal/prompt/templates/skills/` 创建 8 个 core-skill 文件，通过 `embed.FS` 编译进 rick 二进制，并在 `manager.go` 中新增 `LoadCoreSkills()` 函数供各 prompt 构建函数按阶段精准注入。

## 关键事实（实测验证）

`manager.go` 当前 embed 模式是**独立 string 变量**，不存在 `embed.FS`：
```go
import _ "embed"   // blank import，不暴露 embed 包

//go:embed templates/plan.md
planTemplate string   // 共 8 个独立 string 变量
```
**没有 `templateFS`**。`//go:embed templates/skills`（目录）必须绑定 `embed.FS` 类型，不能绑定 string。

## 精准注入映射

| SOP 阶段 | 注入 skills |
|---------|------------|
| plan | sense、tc |
| doing / testing agent | tdd、testing、tc |
| doing / coding agent | tdd、tdd/testing-anti-patterns、debug |
| learning | gen-skill |
| dream | sense、evolve-skills |

# 关键结果

1. **创建 8 个 skill 文件**（关键内容要求如下）：

   - `templates/skills/sense/skill.md`：**SENSE 框架 4 维度必须完整**
     - Subject（主体）：谁在做这件事，动机是什么
     - Perspective（视角）：从不同角色/约束看待问题
     - Judgment（判断）：基于证据得出结论
     - Critique（批判）：找出判断的漏洞和风险
   
   - `templates/skills/evolve-skills/skill.md`：**进化决策逻辑必须明确**
     - 保留：run_log 中触发频次 ≥ 3 且出错次数 < 1/3 触发次数
     - 升级：有效但描述不清晰，需要重写 description/content
     - 淘汰：触发频次 = 0 或出错次数 ≥ 触发次数的 1/2
   
   - `templates/skills/debug/skill.md`：systematic-debugging 4 阶段（Phase1 信息收集/Phase2 假设/Phase3 验证/Phase4 修复）
   - `templates/skills/tdd/skill.md`：RED→GREEN→REFACTOR 铁律
   - `templates/skills/tdd/testing-anti-patterns.md`：禁止伪测试的反模式清单
   - `templates/skills/gen-skill/skill.md`：从 act-path 生成 skill 的格式（触发场景/预期效果/核心内容）
   - `templates/skills/tc/skill.md`：测试用例四要素（前置条件/输入参数/操作序列/预期输出）
   - `templates/skills/testing/skill.md`：测试执行规范
   
   所有文件 description 字段遵循 CSO 规则：`"Use when..."` 开头，仅写触发条件，不超过 200 词

2. **更新 `manager.go` embed 声明**（不影响现有 8 个 string 变量）：
   ```go
   import "embed"  // 从 _ "embed" 改为 "embed"（暴露包以使用 embed.FS）

   // 现有 8 个 string 变量保持不变（planTemplate 等）

   //go:embed templates/skills
   var skillsFS embed.FS   // 新增，递归嵌入整个 skills 目录树
   ```

3. **在 `manager.go` 新增 `LoadCoreSkills(names []string) string`**：
   - 遍历 names，从 `skillsFS` 读取 `templates/skills/{name}/skill.md`
   - 特例：`tdd/testing-anti-patterns` → 路径 `templates/skills/tdd/testing-anti-patterns.md`
   - 拼接内容用 `\n\n---\n\n` 分隔
   - 文件不存在时 `log.Printf` warn，跳过，不 panic

4. **各 prompt 构建函数精准调用 `LoadCoreSkills`**（**只注入对应阶段**）：
   - `GeneratePlanPromptFile` → `LoadCoreSkills([]string{"sense", "tc"})`
   - `buildTestGenerationPromptFile` → `LoadCoreSkills([]string{"tdd", "testing", "tc"})`
   - `GenerateDoingPromptFile` → `LoadCoreSkills([]string{"tdd", "tdd/testing-anti-patterns", "debug"})`
   - `buildLearningPrompt` → `LoadCoreSkills([]string{"gen-skill"})`
   - `GenerateDreamPromptFile` → `LoadCoreSkills([]string{"sense", "evolve-skills"})`

# 测试方法

1. 编译：`python3 tools/build_and_get_rick_bin.py`
2. **`TestCoreSkillsEmbed` 单元测试**（本 task 新增于 `internal/prompt/manager_test.go`）：
   - 逐一调用 `LoadCoreSkills([]string{"sense"})` 等验证 8 个 skill 文件均非空
   - 验证 `tdd/testing-anti-patterns` 路径（斜杠名称）可正确读取
   - 验证多 skill 拼接结果包含 `"---"` 分隔符
   - 验证不存在的 skill 不 panic，只输出空（或 warn）
   ```
   go test ./internal/prompt/... -v -run TestCoreSkillsEmbed
   ```
3. **各阶段精准注入**（覆盖所有 5 个阶段）：
   ```bash
   python3 tools/check_prompt_variables.py --phase plan    --keywords "SENSE"              # sense skill
   python3 tools/check_prompt_variables.py --phase doing   --keywords "systematic-debug"   # debug skill
   python3 tools/check_prompt_variables.py --phase learning --keywords "gen-skill"          # gen-skill
   python3 tools/check_prompt_variables.py --phase dream   --keywords "skill:sense"        # sense in dream
   ```
4. **无污染验证**：
   ```bash
   python3 tools/check_prompt_variables.py --phase doing --keywords "gen-skill"   # → 未找到
   python3 tools/check_prompt_variables.py --phase plan  --keywords "skill:debug" # → 未找到
   ```
5. 完整测试：`go test ./...`，无新增失败
