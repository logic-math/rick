# Skills Index

本目录包含可在 doing 阶段参考的组合技能说明书（Markdown 格式）。工具脚本（`.py`）位于项目根目录 `tools/`。

## 可用 Skills

| Skill | 描述 | 触发场景 |
|-------|------|----------|
| dag_task_decomposition.md | DAG 任务分解方法论 | 当需要将复杂任务拆解为有依赖关系的子任务时 |
| doc_engineering_three_phases.md | 文档工程三阶段法 | 当任务涉及大规模文档生成时 |
| documentation_engineering.md | 文档工程完整指南 | 当需要系统性地创建或重构项目文档时 |
| zero_retry_task_design.md | 零重试任务设计原则 | 当设计任务粒度、编写 task.md 时 |

## 调用方式

参考对应的 `.md` 文件了解技能的使用场景和执行步骤。工具脚本请使用：

```bash
python3 tools/<filename>.py
```
