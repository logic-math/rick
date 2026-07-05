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

## human-loop 模块

- 命令：`rick human-loop <topic>`
- 通过 SENSE 方法论模板引导深度分析，产出存入 `.rick/RFC/` 目录
- 三个 sub agent 模板（think/learn/express）通过 Go embed 编译进二进制
- 运行时写出到系统 tmp，路径注入主控 prompt

### human_loop_think.md 扩展（job_26）

- **判断记录协议**：每个 SENSE 阶段推进条件满足后，提取 1-3 条关键判断追加到 `{{draft_dir}}/human-learning/judgment.md`
- **概念展开标记**：Perspective 阶段识别到值得深入的概念时，写入 `{{draft_dir}}/human-learning/loops.md`

### human_loop_express.md 扩展（job_26）

- **第零步：judgment.md review**：读取 `{{draft_dir}}/human-learning/judgment.md`，文件不存在时直接跳过（不报错）
- **第五步：ZPD 显式评价**：会话结束后引导用户回答 3 个问题，追加写入 `{{draft_dir}}/progress.md` 和 `{{draft_dir}}/loops.md`

### draft_dir 变量注入（job_26）

learning 阶段现在注入 `{{draft_dir}}` 变量，路径为 `.rick/jobs/{job_id}/draft/`，由 `internal/cmd/learning.go` 的 `buildLearningPrompt` 函数注入。

## dream 模块（internal/cmd/dream.go）

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
