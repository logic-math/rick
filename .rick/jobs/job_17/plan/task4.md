# 依赖关系


# 任务名称
更新 SPEC 和 wiki 文档，清理 rick easy 独立命令引用和 tools merge 残留引用

# 任务目标
更新 `.rick/SPEC.md`、`.rick/wiki/` 下所有文档，删除或修正：① `rick easy`（独立命令形式）的描述（`rick doing --easy` 功能仍存在，只删除将其描述为独立命令的部分）；② `rick tools merge` "尚未实现"引用（merge 功能已决策不做）；③ OKR 中 KR2.2 的 merge 相关描述。确保文档与实现现状一致。

注：`core_skills_injection.md` 中描述 `doing/easy` 阶段 skill 注入的内容**保留**（`--easy` 功能本身仍存在）。

# 关键结果
1. `.rick/SPEC.md` 中无 `rick tools merge` 相关描述；重点：`工程实践` 章节的 `知识合并` 条目（当前含 `rick tools merge 命令尚未实现，见 RFC-005`）改为人工手动合并说明；`--easy` flag 的使用说明可保留
2. `.rick/wiki/learning_phase_workflow.md`：删除 "rick tools merge 尚未实现" 相关步骤，替换为人工手动合并说明
3. `.rick/wiki/rick_tools_commands.md`：删除 "rick tools merge 尚未实现（见 RFC-005）" 相关内容
4. `.rick/OKR.md`：KR2.2 更新为描述人工审核 + 手动合并流程（不再提 merge 命令）
5. `grep -rn "tools merge\|RFC-005" .rick/SPEC.md .rick/wiki/ .rick/OKR.md` 无残留引用（`tests/` 目录不在本次文档清理范围内）

# 测试方法
1. **SPEC 无 tools merge 相关内容**
   - 操作：`grep -n "tools merge\|rick merge\|RFC-005" .rick/SPEC.md`
   - 预期输出：无输出（0 匹配）

2. **wiki 文档无 tools merge / RFC-005 残留**
   - 操作：`grep -rn "tools merge\|RFC-005" .rick/wiki/ --include="*.md"`
   - 预期输出：无输出（0 匹配）

3. **OKR KR2.2 已更新，且 learning_phase_workflow.md 有人工合并说明**
   - 操作 A：`grep -A3 "KR2.2" .rick/OKR.md`；预期：不含 `rick tools merge`，且含 `手动` 或 `人工` 关键词
   - 操作 B：`grep -n "手动\|人工.*合并\|git merge" .rick/wiki/learning_phase_workflow.md`；预期：至少 1 行匹配（确认人工合并说明已写入）

# 调试方法
遇到任何不符合预期的行为，必须加载并遵循 debug-skill：
`/Users/sunquan/ai_coding/CODING/rick/.rick/jobs/job_17/doing/prompts/skill_debug_skill.md`

执行顺序：按 debug-skill 三阶段调试法（源码推理法→增量调试法→科学实验法）执行，每阶段达上限后升级人工协作
