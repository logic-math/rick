# research-5 N3-rick cli 职责变化

节点路径:[根 > Y12 交互协议 > N3-rick cli 职责变化]
事实陈述:迁移到 pi 后 rick cli 本身职责是否有所变化——现状职责清单、迁移后保留/迁移/jointly 实现、最小职责集、模板/skill/subagent 归属

## 执行动作

1. Read `internal/cmd/` 全部命令文件(plan/doing/learning/easy/dream/ctrl/human_loop/tools_*)
2. Read `internal/prompt/` 模板体系(manager.go / builder.go / context.go / doing_prompt.go / plan_prompt.go / learning 等)
3. Read `internal/prompt/templates/` 模板文件清单(plan/doing/learning/dream/easy/ctrl/sense_loop/think/research/exporter/test_python)
4. Read `internal/prompt/templates/skills/` skill 文件清单(doing_loop/learning_loop/debug_skill/gen-skill/gen-loop/import_ctx 等)
5. Read `internal/executor/` 执行器(runner.go / executor.go / dag.go / retry.go / doing_check.go)
6. Read `internal/actpath/` 行为轨迹捕获
7. ls `.rick/` 运行时结构(domain/loops/skills/draft/jobs/dream/RFC)
8. Read pi `/tmp/pi_repo/packages/coding-agent/docs/skills.md`(pi skill 体系)
9. Read pi `/tmp/pi_repo/packages/coding-agent/docs/prompt-templates.md`(pi prompt-templates 体系)
10. Read pi `/tmp/pi_repo/packages/coding-agent/docs/extensions.md` 部分(pi extension 体系)

## 各信源验证结果

### 代码原文(权重 0.4)✅

**rick 现状职责清单**(基于 internal/ 目录扫描):

| 职责类别 | 实现位置 | 内容 |
|---|---|---|
| 1. 命令分发 | `internal/cmd/root.go` + 各命令文件 | cobra 命令注册:plan/doing/learning/easy/dream/ctrl/human-loop/tools |
| 2. prompt 模板管理 | `internal/prompt/manager.go` + `templates/` | embed.FS 嵌入 10 个 .md 模板 + 19 个 skill 文件,LoadTemplate / SaveToFile |
| 3. prompt 构建 | `internal/prompt/builder.go` + `doing_prompt.go` 等 | PromptBuilder.SetVariable + SaveToFile,WriteSkillFile / WriteSkillFileWithVars |
| 4. context 管理 | `internal/prompt/context.go` + `context_helpers.go` | ContextManager 管理 debug/loop/skill/domain 注入 |
| 5. skill 加载 | `internal/prompt/templates/skills/` embed.FS | 19 个 skill 文件,WriteSkillFileWithVars 动态注入 job_id/learning_dir 等 |
| 6. loop 加载 | `.rick/loops/{name}-loop.md` 运行时目录 | dream 维护,doing/learning 模板内嵌 doing_loop.md / learning_loop.md |
| 7. agent 调用 | `internal/agent/interface.go` + `claudecode/executor.go` | AgentExecutor 接口 + ClaudeCodeExecutor 实现(13 处调用点) |
| 8. 任务执行 | `internal/executor/runner.go` + `executor.go` | TaskRunner.RunTask + ExecuteJob,DAG 拓扑执行 |
| 9. 行为轨迹捕获 | `internal/actpath/` | act-path.md 生成(基于 AgentSession.ToolCalls) |
| 10. debug 体系 | `internal/executor/debug_dir.go` + `doing_check.go` | debug/bug*.md frontmatter + doing_check 校验 |
| 11. DAG 拓扑 | `internal/executor/dag.go` + `topological.go` | 任务依赖图 + 拓扑排序 |
| 12. retry 机制 | `internal/executor/retry.go` | MaxRetries + testErrorFeedback |
| 13. 三阶段流程 | `internal/cmd/plan.go` / `doing.go` / `learning.go` | plan → doing → learning 工作流 |
| 14. easy 模式 | `internal/cmd/easy.go` | 交互式 + session resume |
| 15. dream 模式 | `internal/cmd/dream.go` | 跨 job 反思 + loop/skill 进化 |
| 16. human-loop | `internal/cmd/human_loop.go` | SENSE 方法论 + 草稿/rfc/loops 目录 |
| 17. ctrl 模式 | `internal/cmd/ctrl.go` | 后台 doing session 监控 + 人工干预 |
| 18. tools 校验 | `internal/cmd/tools_*.go` | plan/doing/learning/dream/loops_skills 各阶段格式校验 |
| 19. git 集成 | `internal/git/` | init/commit/rollback/version/tag |
| 20. workspace 管理 | `internal/workspace/` | .rick 目录定位 + jobID 分配 |

**pi 等价能力清单**(基于 /tmp/pi_repo 调研):

| pi 能力 | rick 现状对应 | 迁移动作 |
|---|---|---|
| Prompt Templates(`~/.pi/agent/prompts/*.md`) | `internal/prompt/templates/*.md` embed.FS | rick 模板可迁移为 pi prompt-templates(但 rick 用 embed.FS 编译期嵌入,pi 用文件系统运行时加载) |
| Skills(`~/.pi/agent/skills/` + SKILL.md 标准) | `internal/prompt/templates/skills/*.md` embed.FS + `.rick/skills/{name}_skill/skill.md` | rick skill 可迁移为 pi skills(Agent Skills 标准,pi 兼容 claude code skill 格式) |
| Extensions(registerTool / on() event hooks / transformContext) | rick 无对应(rick 用 prompt 文本 + claude CLI 实现"subagent") | rick subagent 概念(think/research/exporter)可迁移为 pi extension |
| Compaction(自定义 compactor) | rick 无对应 | 新能力,可启用 |
| Session(树结构 + fork) | rick 线性 session(easy.go session_id 文件) | 语义兼容,可增强 |
| Multi-provider(--provider --model) | rick 无对应(claude code 单 provider) | 新能力,可启用 |
| Standalone binary(Bun 编译) | rick Go binary | 部署形态可对齐 |

**rick 现有 prompt 模板体系细节**(`internal/prompt/manager.go` line 22-53):
- `//go:embed templates/plan.md` 等 10 个 embed 指令
- `//go:embed templates/skills` embed.FS 嵌入 19 个 skill 文件
- 编译期嵌入,运行时不可修改(需重新 build)
- WriteSkillFile / WriteSkillFileWithVars 把 skill 内容写到临时文件,prompt 中引用路径

**rick 现有 .rick/ 运行时结构**(ls 确认):
- `domain/` 描述性规范文档(architecture/commands/go-patterns/testing/project-conventions)
- `loops/` {name}-loop.md 迭代控制流(dream 维护)
- `skills/` {name}_skill/skill.md + 辅助脚本(dream 维护)
- `draft/` human-loop 产出(loops/rfc/concepts/human-learning)
- `jobs/` job 工作目录
- `dream/` dream_run_*_log.md + prompts/
- `RFC/` human-loop 产出

**pi prompt-templates / skills 加载机制**(pi docs/skills.md + prompt-templates.md):
- pi 启动时扫描 `~/.pi/agent/skills/` + `.pi/skills/` + packages + settings + `--skill` flag
- skill 内容 progressive disclosure:启动时仅 name+description 进 system prompt,agent 用 `read` 加载完整 SKILL.md
- pi 兼容 claude code skill 格式(skills.md "Using Skills from Other Harnesses" 段明示:`~/.claude/skills` 可加入 pi settings)
- prompt-templates:`/name` 命令展开(filename → command name)

### 运行时行为(权重 0.3)✅

**迁移后 rick 职责三分类**:

**A. 保留在 rick cli**(rick 核心调度,pi 无法替代):
1. 命令分发(cobra)
2. workspace 管理(.rick 目录 + jobID)
3. 任务执行调度(DAG 拓扑 + retry + ExecuteJob)
4. 行为轨迹捕获(actpath)
5. debug 体系(debug_dir + doing_check)
6. git 集成
7. tools 校验(plan/doing/learning/dream 格式)
8. 三阶段流程编排(plan → doing → learning 状态机)
9. human-loop 草稿管理(loops/rfc/concepts)
10. dream 跨 job 反思调度

**B. 迁移到 pi extension/skill/prompt-template**(模板内容,可重写为 TS):
1. prompt 模板体系(10 个 .md + 19 个 skill .md → pi prompt-templates + pi skills)
   - 但 rick 用 embed.FS 编译期嵌入 + Go template 变量替换;pi 用文件系统 + frontmatter + `/name` 命令展开
   - 重写为 TS extension 后,rick 可不再持有模板内容,但需保留"模板调度逻辑"(哪个 job 用哪个模板)
2. context 注入(debug/loop/skill/domain → pi transformContext / before_agent_start)
3. subagent 派发(think/research/exporter → pi subagent extension,前序 research-3-N4 已确认 pi 官方提供 subagent extension 范例)
4. doing_loop / learning_loop 内嵌逻辑 → pi extension 的 before_agent_start / transformContext hook

**C. jointly 实现**(rick 调度 + pi 执行):
1. agent 调用(rick 持 AgentExecutor 接口 + PiExecutor 实现,pi 执行)
2. session 管理(rick 持 session_id 文件 + pi `--session` flag 加载)
3. skill 触发(rick 模板中引用 skill 路径 + pi 加载 skill 内容)
4. retry 机制(rick MaxRetries + pi agent_settled 事件)
5. raw log 落盘(rick 写 raw_session_coding.log + pi stdout JSONL)

**rick 作为"基本 cli"的最小职责集**(Y11 human 疑问句的解答候选):
- **必保留**:命令分发 + workspace 管理 + DAG 调度 + retry + actpath + debug 体系 + git + tools 校验 + 三阶段流程状态机 + human-loop 草稿管理 + dream 调度
- **可迁移**:prompt 模板内容(重写为 TS extension)+ skill 内容(迁移到 pi skills 目录)+ subagent 派发(迁移到 pi extension)
- ** jointly**:agent 调用(接口在 rick,实现在 pi)+ session 协调(rick 持久化 session_id,pi 加载)+ skill 触发(rick 引用路径,pi 加载内容)

### 文档(权重 0.2)✅

- rick `internal/prompt/manager.go` 注释明示 embed.FS 用法
- pi `docs/skills.md` line 1-80 明示 skill 加载机制 + 兼容 claude code skill 格式
- pi `docs/prompt-templates.md` line 1-60 明示 prompt-template 加载机制 + `/name` 命令展开
- pi `docs/extensions.md`(前序 research-2-N4 已调研)明示 6 类扩展点
- rick `.rick/domain/commands.md` 描述 rick 命令体系(三阶段 + easy/dream/human-loop/ctrl)
- rick MEMORY.md 记录 v2.9.0 架构迁移:wiki/tools/SPEC.md/OKR.md 已删除,知识重构为 loops/skills/domain 三层

### 反事实(权重 0.1)N/A

- 本节点为现状对比调研,无代码修改

## 还原确认

无 rick 代码修改,无需还原。

## 关键事实

1. **rick 现状职责**:20 类(命令分发/prompt 模板/skill 加载/loop 加载/agent 调用/任务执行/行为轨迹/debug 体系/DAG/retry/三阶段/easy/dream/human-loop/ctrl/tools 校验/git/workspace + context 管理)
2. **pi 等价能力**:pi 有 prompt-templates / skills / extensions / compaction / session(树结构)/ multi-provider / standalone binary 完整体系
3. **pi 兼容 claude code skill 格式**:skills.md 明示 `~/.claude/skills` 可加入 pi settings,意味着 rick 现有 `.rick/skills/{name}_skill/skill.md` 可直接被 pi 加载(若格式符合 Agent Skills 标准)
4. **迁移后 rick 职责三分类**:
   - **保留**(10 类):命令分发/workspace/DAG/retry/actpath/debug/git/tools 校验/三阶段状态机/human-loop 草稿/dream 调度
   - **迁移**(4 类):prompt 模板内容 / skill 内容 / subagent 派发 / doing_loop/learning_loop 内嵌逻辑
   - **jointly**(5 类):agent 调用 / session 管理 / skill 触发 / retry / raw log
5. **rick 最小职责集**:命令分发 + workspace + DAG 调度 + retry + actpath + debug + git + tools 校验 + 三阶段状态机 + human-loop 草稿 + dream 调度(11 类,核心是"调度 + 状态机 + 校验")
6. **关键问题解答**:
   - **rick 是否仍持有 prompt 模板?**:可不再持有(迁移到 pi prompt-templates),但需保留"模板调度逻辑"(哪个 job 用哪个模板的决策)
   - **rick 是否仍管 skill 注册?**:可不再管(迁移到 pi skills 目录),但需保留"skill 触发引用"(prompt 中引用 skill 路径)
   - **rick 是否仍管 subagent 派发?**:可不再管(迁移到 pi subagent extension),但需保留"subagent 调度时机"(sense_loop 各阶段何时派发)
7. **架构决策候选**:rick 作为"基本 cli"= 调度层 + 状态机层 + 校验层,pi 作为"执行层 + 模板层 + skill 层"
8. **Y11 human 疑问句解答**:rick 作为基本 cli 存在 = 保留调度/状态机/校验职责,模板/skill/subagent 内容迁移到 pi extension

## 疑问点

- rick 现有 embed.FS 编译期嵌入 vs pi 文件系统运行时加载,是否影响"rick 作为可分发的 standalone binary"目标?(若 rick 仍 embed.FS 持有模板,则 rick binary 自包含;若迁移到 pi 文件系统,需配套分发 .pi/skills/ 目录)
- rick 现有 `.rick/skills/{name}_skill/skill.md` 是否符合 pi Agent Skills 标准?(需对比 frontmatter / SKILL.md 命名 / 目录结构)
- rick 现有 doing_loop.md 内嵌 doing.md(模板文本内嵌)的语义,迁移到 pi extension 后如何实现?(before_agent_start hook 注入 loop 协议到 system prompt?)

## 置信度评估(由 research 主调度计算)

- 代码原文 ✅ × 0.4 = 0.4
- 运行时行为 ✅ × 0.3 = 0.3
- 文档 ✅ × 0.2 = 0.2
- 反事实 N/A × 0.1 = 0
- 合计 = 0.9(高,≥ 0.8 终止)
