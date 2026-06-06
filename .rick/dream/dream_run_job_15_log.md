# Dream Run: job_15

## 处理概述

- **处理时间**: 2026-06-06
- **Job 状态**: 已完成反思
- **数据来源**: debug.md（3 条目，全部已解决）+ tasks.json（1 task easy_session, success）

## 反思发现

1. **ctrl prompt 模板内容不足（debug2）**：初版 ctrl.md 只写了路径，未说明 NDJSON 格式、目录结构、act-path.md 内容、干预场景。根因：模板编写时未充分阅读 `executor.go` 和 `actpath/generator.go` 源码。修复：重写 ctrl.md，补全四个干预场景（A/B/C/D）和文件结构说明。
2. **ctrl 命令实现零问题（debug1）**：`ctrl.go` + `ctrl_prompt.go` 首次实现即通过，证明"接口规范清晰 + 复用 callClaudeCodeCLI"的零重试模式有效。
3. **super-debugging skill 合并零问题（debug3）**：合并两个旧 skill 文件为 `super-debugging-zh.md`，删除旧文件，build 和 dry-run 验证全部通过。
4. **测试引用过时（subagent_6 发现）**：`manager_test.go:199` 仍引用 `"debug"` skill 名称，但文件已改为 `super-debugging-zh.md`，已记录为 RFC-refactor-1。

## 变更记录

### Skills 变更

- 新增: 无（super-debugging 是同 job 的 doing 产出，非 dream 新增）
- 修改: 无
- 删除: 无

### SPEC.md 变更

- 新增 `rick ctrl` 命令规范（场景A/B/干预指令章节名称、场景B 重置约束、dry-run 要求）
- 新增 `Cobra flag 定义规范`（全局 flag vs 命令 flag 区分规则）
- 修复 `.rick/dream/readme.md` 断链引用（文件已被删除，改为描述 dream/ 目录用途）

### Wiki 文档

- 新增: `ctrl_command.md`（ctrl 命令架构、工作流程、四场景干预、NDJSON 格式、Prompt 生成机制）
- 修改: `core_skills_injection.md`（`debug.md` → `super-debugging-zh.md`，更新注入表和示例）
- 修改: `dream_command.md`（四维 → 六维质量验证）
- 修改: `README.md`（添加 ctrl_command.md 条目，修正 core_skills_injection.md 摘要）

### RFC

- 新增: `RFC-refactor-1.md`（P0: manager_test.go 中 "debug" skill 名称过时，应改为 "super-debugging-zh"）

## 下次建议关注

1. RFC-refactor-1 的 manager_test.go:199 修复 — 低风险，建议下个 job 顺手修复
2. 观察 ctrl 命令在后续 job 中的实际使用情况（汇报格式是否清晰，/loop 监控是否实用）
3. RFC-refactor-go-codebase.md 中记录的 workspace 死代码是否已完全清理（skills.go 已删除，但 paths.go 的 SkillsDirName 常量状态待确认）
