# Job 9 OKR：实现 RFC-001 Context 注入增强

## Objective
提升 rick 的 Context 注入质量，让 plan/doing/learning 各阶段的 agent 获得更精准、更完整的上下文，减少 agent 依赖 git 历史或硬编码占位数据的问题。

## Key Results

### KR1：Learning 输入重构（task1）
- `collectExecutionData()` 直接读取 `plan/OKR.md`、`plan/task*.md`、`doing/debug.md`
- 删除 `buildLearningPrompt()` 中四个硬编码占位函数（`formatGitHistory` 等）
- `learning.md` 模板移除强制 `git show` 指令，改为渐进式披露

### KR2：Skills Index 机制（task2）
- 建立 `.rick/skills/index.md` 标准格式
- `workspace.LoadSkillsIndex()` 读取 index.md 内容
- plan/doing prompt 均注入 skills index，优先引导 agent 复用已有 skills

### KR3：Tools 扫描注入（task3）
- `workspace.LoadToolsList()` 扫描项目根目录 `tools/*.py`
- plan/doing prompt 注入 tools 列表，引导 agent 优先使用确定性工具

### KR4：OKR 改为 Job 级（task4）
- plan 阶段生成 `job_N/plan/OKR.md`（聚焦当前 job 目标）
- doing/learning 从 job 级 OKR.md 读取，不再使用全局 `.rick/OKR.md`

### KR5：集成测试与 Dry-run 增强（task5）
- 覆盖 task1~4 所有关键变更的集成测试
- `rick plan --dry-run` 和 `rick learning --dry-run` 打印完整构建 prompt
