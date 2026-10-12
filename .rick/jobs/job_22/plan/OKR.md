# Job OKR: 实现 RFC-001 上下文架构重设计

## 目标 (Objective)

将 rick 的上下文架构从 `SPEC.md → wiki → tools` 三层迁移到 `loops → skills` 两层，使项目级 loop 和 skill 由 learning 阶段动态产出，agent 通过 loops_context 获取执行时可用的结构化工作流。

## 关键结果 (Key Results)

- KR1: `.rick/loops/` 和 `.rick/skills/` 目录建立，loop.md 三要素格式规范明确（frontmatter: name/trigger/scope）
- KR2: `debug_skill.md` 替换为 diagnosing-bugs Phase 1-6，更精炼的调试抽象落地
- KR3: `LoadLoopsContext()` 函数实现并通过单元测试，遍历 `.rick/loops/*.md` 正确提取 trigger 字段
- KR4: doing/plan/learning/easy/dream 五个 prompt builder 完成迁移：移除 SPEC/OKR/wiki/tools 注入，添加 loops_context 注入
- KR5: 所有模板文件同步更新，`rick tools plan_check job_22` 通过
- KR6: `loop_protocol.md` 通过 embed.FS 内嵌，单一维护；doing/easy 的 dry-run 输出包含真实路径（非字面量 `{{loop_protocol_path}}`），Loop 执行协议正文只存在于 `loop_protocol.md` 一处
