# 依赖关系

# 任务名称
创建 grilling skill 文件（templates/skills/grilling.md）

# 任务目标
在 `internal/prompt/templates/skills/` 下新建 `grilling.md`，内容包含 grilling 追问协议、终止条件（所有决策落实到具体代码路径或工具调用）、操作规范，供 plan 和 easy 模板通过 `{{grilling_skill_path}}` 变量注入使用。文件通过 embed.FS 的 `//go:embed templates/skills` 指令自动包含，无需修改 manager.go。

## 前置阅读（必读）
- RFC 文档：`/Users/sunquan/ai_coding/CODING/rick/.rick/RFC/grilling-integration-2026-06-26.md`
  - 包含 grilling 协议的设计决策、核心指令（"Interview me relentlessly..."）、与 plan/easy 的集成方案
  - 终止条件：用户已确认，所有设计决策落实到具体代码路径或工具调用
  - 排除方案：不新增独立命令、不独立前置会话

# 关键结果
1. `internal/prompt/templates/skills/grilling.md` 文件存在，内容包含核心 grilling 指令（含 "Interview me relentlessly"）、终止条件、逐问规范
2. `LoadCoreSkills([]string{"grilling"})` 调用返回非空字符串（需重新 build 使 embed 生效）
3. 文件内容不含未替换的 `{{variable}}` 占位符（grilling.md 本身是纯内容 skill，不用变量替换）

# 测试方法
1. 正常路径：
   - 前置条件：文件已写入 `internal/prompt/templates/skills/grilling.md`，执行 `./scripts/build.sh` 构建成功
   - 输入：`go test ./internal/prompt/... -run TestLoadCoreSkills_Grilling -v`（新增测试用例）
   - 操作：测试调用 `prompt.LoadCoreSkills([]string{"grilling"})`
   - 预期输出：返回字符串包含 "Interview me relentlessly"，长度 > 100 字符
2. 边界用例：
   - 输入：`grep -r '{{' internal/prompt/templates/skills/grilling.md`
   - 预期输出：无匹配（退出码 1），确认 grilling.md 不含未替换变量
3. 异常路径：
   - 输入：`LoadCoreSkills([]string{"nonexistent"})` 
   - 预期输出：返回空字符串，不 panic（现有行为保持不变）

# 调试方法
遇到任何不符合预期的行为，必须加载并遵循 debug-skill：
`/Users/sunquan/ai_coding/CODING/rick/.rick/jobs/job_18/doing/prompts/skill_debug_skill.md`

执行顺序：按 debug-skill 三阶段调试法（源码推理法→增量调试法→科学实验法）执行，每阶段达上限后升级人工协作
