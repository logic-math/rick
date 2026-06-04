# Wiki

| 文件 | 标题 | 摘要 |
|------|------|------|
| act_path_mechanism.md | act-path 生成机制与 DIP 全链路 | `act-path` 是 Rick v2.0 的核心负反馈机制：`rick doing` 执行任务时，程序性解析 Claude Code 的 NDJSON 流式输出，自动生成包含工具调用轨迹、报错次数、执行时长的 `act-path-{taskID}.md` 文件，供 `rick learning` 和 `rick dream` 提取优化信号。 |
| check_mechanism.md | Check 机制工作原理与强制集成 | Rick 的 Check 机制是一组验证工具（`plan_check`、`doing_check`、`learning_check`），用于验证各阶段 Agent 产出文件是否符合规范格式。从 job_11 起，这些工具被强制集成到各阶段的 Agent 提示词模板中，形成"产出 → 自验证 → 修复 → 再验证"的闭环。 |
| core_skills_injection.md | core-skills 精准注入机制 | Rick 将 8 个核心 skill 文件通过 `embed.FS` 编译进二进制，在不同 SOP 阶段精准注入对应 skill，避免信息污染（如 doing 阶段不注入 gen-skill，plan 阶段不注入 debug）。 |
| dream_command.md | dream 命令工作原理 | `rick dream` 是 Rick 三层控制架构中的**进化层**，在 learning 阶段积累足够 act-path 和 run_log 后，由人触发 dream 会话，让 AI 对已处理 job 进行反思、整理 wiki、精简 SPEC、进化 skills。 |
| failure_feedback_propagation.md | Doing 重试循环的失败信息传递机制 | 当 doing 阶段的 Agent 执行任务失败时，Rick 会将失败信息（测试输出、错误详情）传递给下一轮 Agent，帮助其快速定位和修复问题。从 job_11 起，失败信息传递机制经过优化：移除了 500 字符硬截断，改为智能截断策略，并确保传递完整的测试输出（含 stderr/traceback）。 |
| human_loop_command.md | human-loop 命令工作原理与使用指南 | `rick human-loop <topic>` 是一个基于 SENSE 方法论的深度思考辅助命令。它为指定主题生成结构化的引导提示词，并启动 Claude Code CLI 会话，帮助用户对复杂问题进行系统化分析和决策。 |
| human_loop_subagent_pattern.md | Human-Loop Sub Agent 路径注入模式 | human-loop 命令通过"路径注入"方式将三个 sub agent 模板文件的路径写入主控 prompt，AI 在执行时按需读取对应文件内容。这是"渐进式加载"设计——主控 prompt 保持精简，sub agent 规则只在需要时才加载到上下文。 |
| job_okr_design.md | Job 级 OKR 设计 | Rick 将 OKR 从全局级（`.rick/OKR.md`）改为 job 级（`job_N/plan/OKR.md`）。每个 job 有独立的 OKR，由 plan 阶段的 Claude 根据用户需求自动生成，doing/learning 阶段读取并注入提示词。 |
| learning_phase_workflow.md | Learning 阶段工作流 | Rick 的 learning 阶段是 plan→doing→learning 循环的最后一步，负责从 job 执行过程中提取可复用的知识，并将其沉淀到项目的知识库（`.rick/`）中。 |
| rick_tools_commands.md | rick tools 命令体系 | `rick tools` 是 Rick CLI 的工具链子命令体系，提供 plan/doing/learning 三阶段的自动校验和知识合并功能。设计目标是让 AI agent 和人类都能快速验证每个阶段的产出质量，并在出错时提供清晰的错误信息。 |
| sense_merge_decision.md | rick 与 sense 的架构决策：将 sense 合并为 rick 的上下文引擎 | > **文档类型：** SENSE 思考记录 |
| skills_and_tools_injection.md | Skills 与 Tools 注入机制 | Rick 在 plan 和 doing 阶段会自动将项目中可用的 Python 工具脚本注入到提示词中，让 AI agent 在规划和执行任务时能感知并复用现有能力，避免重复造轮子。 |
| skills_tools_separation.md | Skills/Tools 分离机制 | Rick 将可复用知识分为两类：**Tools**（确定性工具脚本）和 **Skills**（组合技能说明书）。两者存放在不同目录，服务于不同目的，共同注入 doing 提示词，为 AI agent 提供完整的执行能力。 |
| test_wiki.md | 测试 Wiki | This is a test wiki page. |
| testing.md | Rick CLI 测试与验证文档 | - [测试策略概览](#测试策略概览) |
