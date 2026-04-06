# Wiki

| 文件 | 标题 | 摘要 |
|------|------|------|
| human_loop_command.md | human-loop 命令工作原理与使用指南 | `rick human-loop <topic>` 是一个基于 SENSE 方法论的深度思考辅助命令。它为指定主题生成结构化的引导提示词，并启动 Claude Code CLI 会话，帮助用户对复杂问题进行系统化分析和决策。 |
| job_okr_design.md | Job 级 OKR 设计 | Rick 将 OKR 从全局级（`.rick/OKR.md`）改为 job 级（`job_N/plan/OKR.md`）。每个 job 有独立的 OKR，由 plan 阶段的 Claude 根据用户需求自动生成，doing/learning 阶段读取并注入提示词。 |
| learning_phase_workflow.md | Learning 阶段工作流 | Rick 的 learning 阶段是 plan→doing→learning 循环的最后一步，负责从 job 执行过程中提取可复用的知识，并将其沉淀到项目的知识库（`.rick/`）中。 |
| rick_tools_commands.md | rick tools 命令体系 | `rick tools` 是 Rick CLI 的工具链子命令体系，提供 plan/doing/learning 三阶段的自动校验和知识合并功能。设计目标是让 AI agent 和人类都能快速验证每个阶段的产出质量，并在出错时提供清晰的错误信息。 |
| sense_merge_decision.md | rick 与 sense 的架构决策：将 sense 合并为 rick 的上下文引擎 | > **文档类型：** SENSE 思考记录 |
| skills_and_tools_injection.md | Skills 与 Tools 注入机制 | Rick 在 plan 和 doing 阶段会自动将项目中可用的 Python 工具脚本注入到提示词中，让 AI agent 在规划和执行任务时能感知并复用现有能力，避免重复造轮子。 |
| test_wiki.md | 测试 Wiki | This is a test wiki page. |
| testing.md | Rick CLI 测试与验证文档 | - [测试策略概览](#测试策略概览) |
