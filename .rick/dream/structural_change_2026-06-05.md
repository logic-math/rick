# Dream Run: 三层结构重组

## 处理概述

- **处理时间**: 2026-06-05
- **类型**: 架构性重组（非 job 执行反思）
- **触发原因**: 将 .rick/ 内部重组为 SPEC.md → wiki/ → tools/ 三层结构

## 变更内容

### 文件迁移
- `tools/*.py`（5 个）→ `.rick/tools/`，删除项目根 `tools/` 目录
- `.rick/skills/*.md`（5 个）→ `.rick/wiki/`，删除 `.rick/skills/` 目录
- `.rick/wiki/sense_merge_decision.md` → `.rick/RFC/RFC-003-sense-merge-decision.md`（决策文档归 RFC）

### Dead Code 确认
- `formatToolsSection` / `formatSkillsSection` 动态注入为 dead code：SPEC.md 本身即 agent 上下文，动态扫描注入冗余；待后续 Go job 清理

### SPEC.md 变更（126 行）
- 新增"三层上下文结构"条目，替换旧 Skills/Tools 分离规范
- 所有 `tools/` 路径更新为 `.rick/tools/`
- 移除变更注释 dead code 描述
- 路径约定：移除 `.rick/skills/` 条目，合并为 `.rick/wiki/` 单条目

### Wiki 文档变更
- `skills_tools_separation.md`：完全重写，描述新三层结构架构
- `wiki/README.md`：新增 5 个 skill 文件条目，更新已改写文档摘要
- 6 个 wiki 文档批量更新工具路径（`tools/` → `.rick/tools/`）
- `dag_task_decomposition.md` / `zero_retry_task_design.md`：统一 section 名称为"触发场景"

### dream_prompt.md 变更（358 → 212 行）
- 删除内嵌的 4 个 dream_run_*_log.md 全文（148 行冗余），改为文件引用一行
- 所有约束路径更新为 `.rick/wiki/`、`.rick/tools/`

## 四维质量验证结果

| 维度 | 结论 |
|------|------|
| 规范一致性 | ✅ SPEC→wiki→tools 引用链完整，2 处 section 命名已统一 |
| 无效上下文清理 | ✅ 148 行冗余内嵌清理，1 个决策文档移至 RFC/ |
| 运行仿真 | ✅ 新人仅凭文档可独立完成工具调用，全流程无断点 |
| 路径推演 | ✅ 5/5，工具路径错误在新上下文下不再会发生 |

## 下次建议关注
1. 开一个 Go job 清理 dead code（`LoadToolsList`、`LoadSkillsIndex`、`formatToolsSection`、`formatSkillsSection`、`tools_merge.go` merge 目标）
2. `modules/workspace.md`、`modules/prompt.md` 等 Go 模块文档仍引用旧 `.rick/skills/` 路径，待 Go 代码更新后同步
