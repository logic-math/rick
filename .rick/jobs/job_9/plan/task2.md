# 依赖关系


# 任务名称
建立 skills/index.md 格式规范并重构 skills 注入机制

# 任务目标
当前 `doing_prompt.go:formatSkillsSection()` 通过扫描 `.rick/skills/*.py` 文件的第一行注释来获取 skill 描述，没有 index 文件，plan 阶段也完全不感知 skills。

RFC-001 要求：
- `.rick/skills/` 必须有 `index.md`，描述每个 skill 的名称和触发场景
- plan 和 doing 阶段都必须注入 skills index
- prompt 中强制要求 agent 优先考虑用已有 skills 解决问题

本任务目标：
1. 定义 `index.md` 的标准格式
2. 重构 `workspace/skills.go`：新增 `LoadSkillsIndex()` 读取 `index.md` 内容
3. 重构 `doing_prompt.go:formatSkillsSection()`：优先读取 `index.md`，若不存在则降级为扫描 `.py` 文件
4. 重构 `plan_prompt.go`：新增注入 skills index（通过新增模板变量 `{{skills_index}}`）
5. 更新 `.rick/skills/index.md`：为现有 skills 补充触发场景描述
6. 更新 `workspace.GenerateSkillsREADME()` → 改名为 `GenerateSkillsIndex()`，生成标准 `index.md` 格式

# 关键结果
1. `workspace/skills.go` 新增 `LoadSkillsIndex(rickDir string) (string, error)` 函数，读取 `.rick/skills/index.md` 原始内容
2. `doing_prompt.go:formatSkillsSection()` 优先使用 `index.md` 内容，降级时保留原有扫描逻辑
3. `plan_prompt.go` 在 prompt 末尾注入 skills index 内容（模板变量 `{{skills_index}}`），并在 plan.md 模板中添加对应章节
4. `.rick/skills/index.md` 存在且包含所有现有 skills 的名称和触发场景
5. `go test ./...` 全部通过

# 测试方法
1. 运行 `go build ./...` 确认编译通过
2. 运行 `go test ./internal/workspace/ -v` 确认 skills 相关测试通过
3. 运行 `go test ./internal/prompt/ -v` 确认 prompt 相关测试通过
4. 运行 `rick doing --dry-run job_9` 或构建后检查 doing prompt 包含 skills index 内容
5. 运行 `go test ./...` 确认全量测试通过
