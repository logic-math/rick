# 依赖关系

task1

# 任务名称

更新 doing.md / plan.md 模板引用，删除 super-debugging-zh.md

# 任务目标

将两个提示词模板从 super-debugging 切换到 debug_skill，同时清理 doing.md 中冗余的 debug{N} 记录格式（改为加载 debug_skill），删除已废弃的 super-debugging-zh.md 文件。

# 关键结果

1. 删除 `internal/prompt/templates/skills/super-debugging-zh.md`（用 `git rm` 或 `rm`，确认文件不再存在）
2. `internal/prompt/templates/doing.md` 更新：
   - 第 3 行声明由 `skill:super-debugging` → `skill:debug-skill`
   - `{{super_debugging_path}}` 替换为 `{{debug_skill_path}}`（出现在 core skills 列表和 DEBUG 铁律章节）
   - 在 core skills 列表中**新增一行** `- skill:sense（系统化思维，供 review debug agent 使用）：{{sense_skill_path}}`
   - DEBUG 铁律章节：将 "I will use skill:super-debugging." 声明 + 五阶段流程描述替换为 "I will use skill:debug-skill." + 三阶段调试流程（加载 debug_skill 文件后执行）
   - 工作日志规范章节的 `debug{N}:` 详细问题记录格式整体删除，改为一句：`遇到 bug 时，加载并严格遵循 skill:debug-skill，在 debug/ 目录下按 bug{n}-{描述}.md 格式记录`
   - 保留 `task{N}:` 执行日志格式（分析过程/实现步骤/遇到的问题（写"无"或引用 debug/{file}）/验证结果）不变
3. `internal/prompt/templates/plan.md` 更新：
   - `{{super_debugging_skill_path}}` 替换为 `{{debug_skill_path}}`
   - 将 "super-debugging skill" 改为 "debug-skill"
   - 将执行顺序说明从 "S（还原问题）→ E（视角分析）→ N（验证假设）→ 修复实现 → 3 次失败则停止找人类协作者" 改为 "按 debug-skill 三阶段调试法（源码推理法→增量调试法→科学实验法）执行，每阶段达上限后升级人工协作"
4. 以上所有变更在 `git diff` 中可见，`git grep super_debugging internal/prompt/templates/` 返回空

# 测试方法

**前置条件**：task1 已完成（debug_skill.md 存在）；git 工作区干净

**测试1：super-debugging-zh.md 已删除**
```bash
test ! -f internal/prompt/templates/skills/super-debugging-zh.md && echo "✅ 已删除" || echo "❌ 仍存在"
```
- 预期：✅ 已删除

**测试2：模板中无旧引用**
```bash
git grep "super_debugging\|super-debugging" internal/prompt/templates/ && echo "❌ 有残留" || echo "✅ 无残留"
```
- 预期：✅ 无残留

**测试3：doing.md 包含新引用**
```bash
grep "debug_skill_path" internal/prompt/templates/doing.md && echo "✅ doing.md 已更新" || echo "❌"
grep "debug-skill" internal/prompt/templates/doing.md && echo "✅ 声明已更新" || echo "❌"
```
- 预期：两行均 ✅

**测试4：plan.md 包含新引用**
```bash
grep "debug_skill_path" internal/prompt/templates/plan.md && echo "✅ plan.md 已更新" || echo "❌"
```
- 预期：✅

**测试5：doing.md 保留 task 执行日志格式**
```bash
grep "## task" internal/prompt/templates/doing.md && echo "✅ 执行日志格式保留" || echo "❌"
```
- 预期：✅

**边界用例：doing.md 中 debug{N} 格式已清理**
```bash
grep "## debug{N}" internal/prompt/templates/doing.md && echo "❌ 旧格式残留" || echo "✅ 已清理"
```
- 预期：✅ 已清理

**测试6：sense_skill_path 占位符存在于 doing.md**
```bash
grep "sense_skill_path" internal/prompt/templates/doing.md && echo "✅ sense_skill_path 占位符存在" || echo "❌ 缺少 sense_skill_path 占位符"
```
- 预期：✅

# 调试方法

遇到任何不符合预期的行为，必须加载并遵循 super-debugging skill：
`/Users/sunquan/ai_coding/CODING/rick/.rick/jobs/job_16/doing/prompts/skill_super_debugging_zh.md`

执行顺序：S（还原问题）→ E（视角分析）→ N（验证假设）→ 修复实现 → 3 次失败则停止找人类协作者
