# Wiki

| 文件 | 标题 | 摘要 |
|------|------|------|
| act_path_mechanism.md | act-path 生成机制与 DIP 全链路 | `act-path` 是 Rick v2.0 的核心负反馈机制：`rick doing` 执行任务时，程序性解析 Claude Code 的 NDJSON 流式输出，自动生成包含工具调用轨迹、报错次数、执行时长的 `act-path-{taskID}.md` 文件，供 `rick learning` 和 `rick dream` 提取优化信号。 |
| check_mechanism.md | Check 机制工作原理与强制集成 | Rick 的 Check 机制是一组验证工具（`plan_check`、`doing_check`、`learning_check`），用于验证各阶段 Agent 产出文件是否符合规范格式。从 job_11 起，这些工具被强制集成到各阶段的 Agent 提示词模板中，形成"产出 → 自验证 → 修复 → 再验证"的闭环。 |
| ctrl_command.md | ctrl 命令工作原理与使用指南 | `rick ctrl --job <id>` 是 Rick 的监控与干预命令，在 `rick doing` 后台运行时启动交互式 Claude 会话，支持实时观测进度（任务状态/流式日志/debug 记录）和四种干预场景（追加指令/重置状态/查看轨迹/查看原始日志）。 |
| core_skills_injection.md | core-skills 精准注入机制 | Rick 将核心 skill 文件通过 `embed.FS` 编译进二进制，在不同 SOP 阶段精准注入对应 skill，避免信息污染（如 doing 阶段不注入 gen-skill，plan 阶段不注入 super-debugging）。 |
| dag_task_decomposition.md | DAG 任务分解方法 | 将复杂任务分解为多个子任务并设计依赖关系的方法论。适用于 plan 阶段设计任务 DAG，确保任务之间的依赖清晰、可并行执行。 |
| dream_command.md | dream 命令工作原理 | `rick dream` 是 Rick 三层控制架构中的**进化层**，在 learning 阶段积累足够 act-path 和 run_log 后，由人触发 dream 会话，让 AI 对已处理 job 进行反思、整理 wiki、精简 SPEC、进化 skills。 |
| failure_feedback_propagation.md | Doing 重试循环的失败信息传递机制 | 当 doing 阶段的 Agent 执行任务失败时，Rick 会将失败信息（测试输出、错误详情）传递给下一轮 Agent，帮助其快速定位和修复问题。从 job_11 起，失败信息传递机制经过优化：移除了 500 字符硬截断，改为智能截断策略，并确保传递完整的测试输出（含 stderr/traceback）。 |
| human_loop_command.md | human-loop 命令工作原理与使用指南 | `rick human-loop <topic>` 是一个基于 SENSE 方法论的深度思考辅助命令。它为指定主题生成结构化的引导提示词，并启动 Claude Code CLI 会话，帮助用户对复杂问题进行系统化分析和决策。 |
| human_loop_subagent_pattern.md | Human-Loop Sub Agent 路径注入模式 | human-loop 命令通过"路径注入"方式将三个 sub agent 模板文件的路径写入主控 prompt，AI 在执行时按需读取对应文件内容。这是"渐进式加载"设计——主控 prompt 保持精简，sub agent 规则只在需要时才加载到上下文。 |
| job_okr_design.md | Job 级 OKR 设计 | Rick 将 OKR 从全局级（`.rick/OKR.md`）改为 job 级（`job_N/plan/OKR.md`）。每个 job 有独立的 OKR，由 plan 阶段的 Claude 根据用户需求自动生成，doing/learning 阶段读取并注入提示词。 |
| learning_phase_workflow.md | Learning 阶段工作流 | Rick 的 learning 阶段是 plan→doing→learning 循环的最后一步，负责从 job 执行过程中提取可复用的知识，并将其沉淀到项目的知识库（`.rick/`）中。 |
| rick_tools_commands.md | rick tools 命令体系 | `rick tools` 是 Rick CLI 的工具链子命令体系，提供 plan/doing/learning 三阶段的自动校验和知识合并功能。设计目标是让 AI agent 和人类都能快速验证每个阶段的产出质量，并在出错时提供清晰的错误信息。 |
| skills_tools_separation.md | .rick/ 三层上下文结构 | `.rick/` 内部形成 `SPEC.md → wiki/ → tools/` 三层结构：SPEC 是 agent 上下文入口，wiki/ 存放原理文档和技能说明书，`.rick/tools/` 存放可执行 Python 脚本。 |
| test_go_project_changes.md | 验证 Go 项目代码修改 | 修改了 Go 源文件后，确认编译通过、单元测试和集成测试通过的标准验证流程。 |
| test_script_best_practices.md | 测试脚本最佳实践 | 编写或调试任务测试脚本时的常见陷阱与修复方案，涵盖路径错误、binary 版本混用、section 误判、variadic 工具限制等。 |
| testing.md | Rick CLI 测试与验证文档 | - [测试策略概览](#测试策略概览) |
| verify_rick_check_commands.md | 验证 rick check 命令 | 验证 `rick tools plan_check`/`doing_check`/`learning_check` 命令行为是否符合预期的操作指南。 |
| zero_retry_task_design.md | 零重试任务设计模式 | plan 阶段设计 task.md 时，参考高成功率任务的设计要素，降低 doing 阶段重试率。 |
