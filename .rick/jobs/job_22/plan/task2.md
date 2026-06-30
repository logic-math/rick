# 依赖关系


# 任务名称

将 `debug_skill.md` 三阶段结构替换为 diagnosing-bugs Phase 1-6

# 任务目标

用 RFC-001 中 diagnosing-bugs 的 Phase 1-6 抽象（构建反馈回路→复现最小化→可证伪假设→插桩观察→修复回归→清理事后分析）替换 `internal/prompt/templates/skills/debug_skill.md` 中现有的三阶段结构（源码推理法/增量调试法/科学实验法），保留 bug*.md frontmatter 格式规范和 review debug agent 协议，并重新构建二进制。

diagnosing-bugs 六阶段含义：
- Phase 1：构建反馈回路（初次复现 bug + 追求最小化复现路径，获得秒级可重复运行的测试）
- Phase 2：复现 + 最小化（基于 Phase 1，进一步裁剪到最小可复现单元）
- Phase 3：生成 3-5 个有优先级排序的可证伪假设
- Phase 4：插桩观察（log/gdb/delve/pprof 等均视为桩，选最合适的插入关键路径）
- Phase 5：修复 + 回归测试（基于确认的根因，最小改动修复，全量测试通过）
- Phase 6：清理 + 事后分析（移除临时桩，写 bug*.md 结论章节，提炼防范模式）

# 关键结果

1. `internal/prompt/templates/skills/debug_skill.md` 正文中不再有"阶段一：源码推理法""阶段二：增量调试法""阶段三：科学实验法"标题，替换为"Phase 1"至"Phase 6"六个章节
2. YAML frontmatter（name: debug-skill, description）保留不变
3. `bug*.md` 格式规范（frontmatter: summary/status + 合法状态值）保留，但正文章节从三阶段改为六阶段对应的记录格式
4. review debug agent 协议（含 sense 触发、输出格式、角色约束）完整保留
5. 运行 `./scripts/build.sh` 成功，`./bin/rick --version` 正常输出版本号
6. 运行 `go test ./internal/prompt/... -run TestEmbedded -v` 通过
7. **⚠️ 必须同步检查并更新**：`internal/prompt/templates/doing.md` 中 bug*.md 格式铁律说明（搜索"阶段一""阶段二""阶段三"关键词），若存在则同步改为 Phase 1-6；同步检查 `internal/executor/doing_check.go`（若存在）中校验 bug*.md 章节名称的逻辑，确保不与新 Phase 格式冲突

# 测试方法

1. **正常路径 - 内容替换验证**：
   - 前置条件：`debug_skill.md` 已按新格式更新，二进制已重新构建
   - 输入：读取文件内容检查
   - 操作：`grep -c "Phase [1-6]" internal/prompt/templates/skills/debug_skill.md && grep -c "源码推理法\|增量调试法\|科学实验法" internal/prompt/templates/skills/debug_skill.md`
   - 预期输出：第一条输出 6（Phase 1-6 全部存在），第二条输出 0（旧阶段名消失）

2. **保留内容验证**：
   - 前置条件：文件已更新
   - 操作：`grep -c "review debug agent\|sense_skill_path\|summary.*status" internal/prompt/templates/skills/debug_skill.md`
   - 预期输出：输出 ≥ 3（review debug agent 协议、sense 路径、frontmatter 字段均保留）

3. **二进制构建验证**：
   - 前置条件：debug_skill.md 已更新
   - 操作：`./scripts/build.sh && ./bin/rick --version`
   - 预期输出：构建成功，版本号正常输出（无 panic/编译错误）

4. **单元测试验证**：
   - 前置条件：二进制已构建
   - 操作：`go test ./internal/prompt/... -run TestEmbedded -v`
   - 预期输出：所有 Embedded 相关测试通过（模板嵌入 debug_skill.md 内容可读取）

5. **边界用例 - 直接读取 skill 文件验证 Phase 结构**（注意：debug_skill.md 通过 WriteSkillFileWithVars 写出为独立文件，不在 doing dry-run 主体输出中，须直接读取源文件验证）：
   - 前置条件：`debug_skill.md` 已按新格式更新
   - 操作：`grep -c "## Phase [1-6]" internal/prompt/templates/skills/debug_skill.md`
   - 预期输出：输出 6（六个 Phase 章节标题均存在）

# 调试方法

遇到任何不符合预期的行为，必须加载并遵循 debug-skill：
`/Users/sunquan/ai_coding/CODING/rick/.rick/jobs/job_22/doing/prompts/skill_debug_skill.md`

执行顺序：按 debug-skill Phase 1-6 执行，每阶段达上限后升级人工协作
