# Rick 架构设计

## 技术栈

- 语言: Go 1.21+（主程序），Python 3.8+（辅助脚本和测试脚本）
- 框架: Cobra（CLI 命令框架），Goldmark（Markdown 解析）
- 测试: Go testing 标准库，Python unittest，Bash integration tests
- 其他: Git（版本管理），Claude Code CLI（AI agent 集成）

## 模块划分

```
cmd/rick/main.go              # 入口，VERSION 常量
internal/
  cmd/                        # Cobra 命令处理（plan/doing/learning/easy/dream/ctrl/human-loop）
  executor/                   # 任务执行引擎（runner.go/doing_check.go）
  prompt/                     # 提示词管理（builder/manager/templates via embed.FS）
  workspace/                  # 路径管理
  parser/                     # 内容解析
  git/                        # Git 操作
  callcli/                    # Claude Code CLI 集成
  agent/                      # 接口契约（AgentSession/AgentExecutor/ToolCall）
    claudecode/               # 唯一实现，仅在 doing.go 组合根中实例化
  actpath/                    # act-path 生成
```

## Prompt Context 注入体系

`internal/prompt/context_helpers.go` 提供两个并行的 context loader，均在 doing/easy/plan/dream 启动时注入：

| 函数 | 读取源 | 注入变量 | 提取字段 |
|------|--------|---------|---------|
| `LoadLoopsContext` | `.rick/loops/*.md` | `{{loops_context}}` | frontmatter `name` + `trigger` |
| `LoadSkillsContext` | `.rick/skills/*_skill/skill.md` | `{{skills_context}}` | `# skill:` 标题 + `## 触发场景` 首行 |

两者格式对称：`- **{name}**：{trigger/触发场景首行}`。新增同类 context loader 时，遵循相同模式。

**来源 Job**: job_23

## DIP 组合根模式

**核心约束**：`doing.go` 是唯一 import `internal/agent/claudecode` 的地方。

```
doing.go（组合根）
  └── import claudecode ✅
runner.go / executor.go / actpath/
  └── import agent（接口）✅  NOT claudecode ❌
```

**验证命令**：
```bash
grep -r "claudecode" internal/executor/ internal/actpath/
# 应无输出
```

**nil guard**：`actpath.Generate(session, outputFile)` 中 session 为 nil 时跳过，不 panic。

## agent 接口模块（internal/agent/）

```go
type AgentSession interface { /* Claude Code 会话 */ }
type AgentExecutor interface {
    Execute(session AgentSession, task Task) (Result, error)
}
type ToolCall struct { /* 工具调用记录 */ }
```

`claudecode` 子包为唯一实现，仅在组合根 `doing.go` 中实例化。

## act-path 生成模块（internal/actpath/）

```go
func Generate(session AgentSession, outputFile string) error
```

- 不 import 任何具体 agent 实现（仅依赖 `internal/agent` 接口）
- 输出三节：执行摘要 / 行为轨迹 / Agent 最终输出
- 每个 task 的 `agentExecutor.Execute` 完成后调用

## human-loop 模块(v3.1, job_28 重构)

- 命令:`rick human-loop <topic>`
- 通过 SENSE 方法论引导 5 阶段深度思考,产出存入 `.rick/draft/rfc/` 目录
- **四文件架构**(go embed 内嵌模板):
  - `templates/sense_loop.md` — main agent 协议(5 阶段调度+批判门禁嵌入+反向回流)
  - `templates/think.md` — subagent(推理识别+假设提取+4维打分+3启发性问题)
  - `templates/research.md` — subagent(尽调树+信源加权+subagent 上下文隔离)
  - `templates/exporter.md` — subagent(RFC 输出,大纲+内容两阶段)
- 对应 skill 文件:`templates/skills/{sense,think,research,exporter}.md`
- 运行时写到 `{{loop_dir}}/prompts/`,路径注入主控 prompt

### 5 阶段流程(v3,替代 v2 7 步线性)

```
S ⇄ E ⇄ N ⇄ S-R ⇄ EC
↑                    ↑
└── 跃迁/反向回流 ──┘
```

| 阶段 | 名称 | 核心动作 |
|---|---|---|
| S | 问题确认 | research 调研现状+对 human 假设追问,完成现状/期望/差距 |
| E | 视角生成 | research 跨领域调研→多视角候选→human 给原创视角 |
| N1 | 矛盾生成 | 用系统论描述符(node/input/output/inner/edge)描述系统 |
| N2 | 主要矛盾判断 | think 三维打分(根本性/全局性/决定性)+ human 选定 |
| S-R | 辩证逆转 | "若 X 必然,实现 Y 应当如何?" |
| EC | 良知批判 | sense_loop 呈现回顾,**human 自判**(无 subagent) |

### 关键设计决策

1. **5 阶段非线性**:替代 v2 7 步线性,允许反向回流(后续可重启前序,上限 `sense_max_backflows` 默认 3)
2. **批判门禁嵌入**:think 不再独立步骤,嵌入各阶段(human 实质性回答后触发)
3. **EC human 自判**:不替 human 提议跃迁方向,AI 只呈现回顾
4. **系统论描述符**(N1 阶段):5 要素 node/input/output/inner/edge,替代模糊概念地图
5. **简化产物**:2 产物(briefs+judgment.md),删除 loops.md/progress.md(v2 4 产物)
6. **配置化所有阈值**:`max_retries`/`sense_max_backflows`/`think_top_n`/`think_min_assumptions`/`research_source_weights`

### 假设数量保障 + 3 启发性问题(think v3.1)

- **最低假设数**:`think_min_assumptions` 默认 5,多视角强制(演绎+归纳+溯因+交叉)
- **补强流程**:低于 min 则反事实/边缘/隐含假设迭代 2 轮
- **每假设 3 启发性问题**:
  - Q1 信念:关于 [Y],你内心最确信的是什么?最不确定的是什么?
  - Q2 前提:[Y] 成立需要什么前提?这些前提你确认过吗?
  - Q3 反例:什么证据会让你改变对 [Y] 的判断?
- 总提问数 ≥ 5 × 3 = 15 问题(默认配置)

### 尽调树 + 信源加权(research v2)

- **尽调树**:MECE 划分,深度 ≤5,每层 ≤7,总节点 ≤30
- **信源加权**:代码原文 0.4 / 运行时 0.3 / 文档 0.2 / 反事实 0.1(可配置)
- **置信度** = Σ(信源验证结果 × 权重),高 ≥ 0.8(终止)/ 中 0.5-0.8 / 低 < 0.5(R7 上报)
- **subagent 上下文隔离**:主 research 维护树+主报告,具体调研派给 subagent 落盘

### draft_dir 变量注入(job_26 保留)

learning 阶段注入 `{{draft_dir}}` 变量,路径为 `.rick/jobs/{job_id}/draft/`,由 `internal/cmd/learning.go` 的 `buildLearningPrompt` 函数注入。

## dream 模块(internal/cmd/dream.go)

- 不生成 act-path
- 自动扫描 `.rick/jobs/*/doing/tasks.json` 发现已完成 jobs
- 与 `dream_run_*_log.md` 对比得出待处理列表
- 支持 `--background`/`-p` 背景模式（`--dangerously-skip-permissions`）

## Tools 子命令体系

`rick tools` 提供四个校验子命令，统一输出格式：

| 命令 | 验证对象 |
|------|---------|
| `plan_check` | tasks/*.md 格式 + tasks.json 存在 |
| `doing_check` | tasks.json 可解析 + debug/bug*.md 格式 |
| `learning_check` | SUMMARY.md 非空且含 `# Job` 标题 |
| `dream_check` | loops/*.md 五要素格式 |

**exit code**: 0=pass / 1=fail；输出 `✅ PASS` 或 `❌ FAIL + 描述`。

`--auto-fix` 标志才触发 Claude 修复，默认只报告（保持确定性）。
