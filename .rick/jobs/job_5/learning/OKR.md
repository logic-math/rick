<!-- Updated in learning phase: job_5 (2026-03-22) -->
# OKR

**愿景**: 打造以促进人类深度学习、思考、表达为目的的可控人工智能系统。

## O1: 构建上下文优先的可控人工智能系统

Rick 的核心假设是：AI 的输出质量取决于上下文质量。通过结构化的上下文管理（SPEC、OKR、debug、skills、wiki），让 AI agent 在每次任务执行时都能获得完整、准确、可控的上下文，从而产出高质量的结果。

### 关键结果 (Key Results)

- KR1.1: doing 提示词自动注入 SPEC、已完成任务历史、debug 记录、项目 skills，覆盖率 100%
- KR1.2: `rick tools plan_check` 能检测 6 类上下文结构错误，确保进入 doing 阶段的任务格式正确
- KR1.3: debug.md 作为强制工作日志，每次任务执行必须记录，确保失败上下文可追溯
- KR1.4: 任务重试时自动加载 debug.md 作为上下文，重试成功率相比无上下文提升可测量

## O2: 构建使人成长、使 AI 进化的双循环学习引擎

每次 job 执行后，人类通过审核 learning 产出获得深度思考和总结的机会；AI 通过 skills/wiki 的积累在下次任务中获得更好的起点。两者形成正向循环，随时间共同进化。

### 关键结果 (Key Results)

- KR2.1: learning 阶段产出四类标准化文档（SUMMARY / skills / OKR / SPEC），每类有明确格式规范
- KR2.2: `rick tools merge` 实现 learning 产出到 `.rick/` 的安全合并，分支隔离 + 人工审核双重保障
- KR2.3: `.rick/skills/*.py` 在下次 doing 时自动注入提示词，形成知识复用闭环
- KR2.4: 每次 job 的 SUMMARY.md 包含可量化的执行指标（完成率、重试次数、问题数量）

## O3: 构建开发者体验优先、生产级可用的 AI Coding 框架

Rick 应该足够简单，让开发者能在 5 分钟内上手；足够健壮，能在真实项目中稳定运行；足够通用，不绑定特定项目或团队。

### 关键结果 (Key Results)

- KR3.1: 核心命令只有三个（`rick plan` / `rick doing` / `rick learning`），无需 init，自动初始化
- KR3.2: 核心模块（cmd/executor/prompt）单元测试覆盖率 ≥ 70%，集成测试覆盖所有 tools 子命令
- KR3.3: 移除所有硬编码项目名称，Rick 可用于任意 Git 项目，零配置启动
- KR3.4: 支持生产版（`rick`）和开发版（`rick_dev`）并行运行，用于 Rick 自我重构场景
- KR3.5: `--auto-fix` 标志为 opt-in 设计，check 命令默认行为确定性，可在 CI 中稳定使用
