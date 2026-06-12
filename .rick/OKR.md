<!-- 变更说明：本次 job_14 执行后更新
- 新增：O4 - 建立 act-path 进化循环（原因：v2.0 核心升级，通过程序性 NDJSON 解析建立负反馈机制）
- 新增：KR4.1 - act-path 自动生成（原因：doing 执行后需产出可机读的行为轨迹）
- 新增：KR4.2 - learning 六步 SOP + act-path 注入（原因：learning 需消费 act-path 提取优化信号）
- 新增：KR4.3 - dream 命令落地（原因：人工触发的进化层，消费 act-path + run_log）
- 新增：KR4.4 - core-skills 精准注入（原因：不同 SOP 阶段需要不同 skill 组合，避免信息污染）
- 修改：KR3.1 - 补充 rick dream（原因：核心命令扩展为四个）
-->
# OKR

**愿景**: 打造以促进人类深度学习、思考、表达为目的的可控人工智能系统。

## O1: 构建上下文优先的可控人工智能系统

Rick 的核心假设是：AI 的输出质量取决于上下文质量。通过结构化的上下文管理（SPEC、OKR、debug、skills、wiki），让 AI agent 在每次任务执行时都能获得完整、准确、可控的上下文，从而产出高质量的结果。

### 关键结果 (Key Results)

- KR1.1: doing 提示词自动注入 SPEC、已完成任务历史、debug 记录、项目 skills、项目 tools、job OKR，覆盖率 100%
- KR1.2: `rick tools plan_check` 能检测 6 类上下文结构错误，确保进入 doing 阶段的任务格式正确
- KR1.3: debug.md 作为强制工作日志，每次任务执行必须记录，确保失败上下文可追溯
- KR1.4: 任务重试时自动加载 debug.md 作为上下文，重试成功率相比无上下文提升可测量
- KR1.5: `projectRoot/tools/*.py` 自动扫描并注入 plan/doing 提示词，项目特定工具对 AI agent 可见

## O2: 构建使人成长、使 AI 进化的双循环学习引擎

每次 job 执行后，人类通过审核 learning 产出获得深度思考和总结的机会；AI 通过 skills/wiki 的积累在下次任务中获得更好的起点。两者形成正向循环，随时间共同进化。

### 关键结果 (Key Results)

- KR2.1: learning 阶段产出四类标准化文档（SUMMARY / skills / OKR / SPEC），每类有明确格式规范
- KR2.2: learning 产出经人工审核后手动合并到 `.rick/`，审核 SUMMARY.md 确认质量后逐文件 `git add` 提交
- KR2.3: `.rick/skills/index.md` 在下次 doing/plan 时自动注入提示词（优先于 .py 扫描），含触发场景描述，形成知识复用闭环
- KR2.4: 每次 job 的 SUMMARY.md 包含可量化的执行指标（完成率、重试次数、问题数量）

## O3: 构建开发者体验优先、生产级可用的 AI Coding 框架

Rick 应该足够简单，让开发者能在 5 分钟内上手；足够健壮，能在真实项目中稳定运行；足够通用，不绑定特定项目或团队。

### 关键结果 (Key Results)

- KR3.1: 核心命令只有四个（`rick plan` / `rick doing` / `rick learning` / `rick dream`），无需 init，自动初始化
- KR3.2: 核心模块（cmd/executor/prompt）单元测试覆盖率 ≥ 70%，集成测试覆盖所有 tools 子命令
- KR3.3: 移除所有硬编码项目名称，Rick 可用于任意 Git 项目，零配置启动
- KR3.4: 支持生产版（`rick`）和开发版（`rick_dev`）并行运行，用于 Rick 自我重构场景
- KR3.5: `--auto-fix` 标志为 opt-in 设计，check 命令默认行为确定性，可在 CI 中稳定使用
- KR3.6: plan/doing/learning/dream 的 `--dry-run` 标志输出完整 prompt 内容，便于调试和验证上下文注入效果

## O4: 建立可靠的 act-path 进化循环

通过程序性 NDJSON 解析建立 act-path 负反馈机制，使 learning/dream 层能够从真实行为轨迹中提取优化信号，形成"执行→观测→进化"的闭环，而非依赖 LLM 自觉记录。

### 关键结果 (Key Results)

- KR4.1: `rick doing` 执行后自动生成 `doing/tasks/{taskID}/act-path.md`，包含工具调用轨迹（含行号链接）、报错次数、执行时长，原始日志双写到 `raw_session.log`
- KR4.2: `rick learning` 升级为七步 RFC SOP，自动收集所有 act-path 内容注入 `{{act_path_content}}`，Step 2 评估更优轨迹，Step 6 写入 `.rick/dream/run_log_{n}.md` 度量文件
- KR4.3: `rick dream` 命令可运行，`--dry-run` 正常输出完整提示词，消费 act-path + run_log，执行 SENSE 反思和 evolve-skills 进化
- KR4.4: 8 个 core-skill 文件通过 `embed.FS` 编译进二进制，按 SOP 阶段精准注入（plan/doing/learning/dream 各不相同），无跨阶段污染
