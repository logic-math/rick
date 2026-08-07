# 调研报告 — 替换 claude code 引擎，ai_cli 支持 PI agent 可行性调研 — 2026-08-04

## 信源配置

| 信源 | 权重 | 验证方式 |
|---|---|---|
| 代码原文 | 0.4 | Read/Grep |
| 运行时行为 | 0.3 | Bash 跑命令/日志 |
| 文档 | 0.2 | Read |
| 反事实 | 0.1 | 修改代码看影响后还原 |

置信度 = Σ(信源验证结果 × 权重), 高 ≥ 0.8(终止) | 中 0.5-0.8(续研) | 低 < 0.5(R7)

## 尽调树(快照)

```
根:替换 claude code 引擎,ai_cli 支持 PI agent 可行性调研
├─ N1-ai_cli 现状 ✅1.0(高)
├─ N2-claude code 耦合点 ✅1.0(高)
├─ N3-agent 类型与启动路径 ✅1.0(高)
├─ N4-PI agent 概念 ❌0.0(低,R7)
└─ N5-ai_cli 扩展点结构 ✅1.0(高)
```

## 节点详情

### N1-ai_cli 现状:rick 中 ai_cli 的代码位置、扩展机制、调用 claude code 的路径
- 置信度:1.0(高)
- 信源验证:代码原文 ✅(ai_cli 字符串零匹配,human 口语对应 internal/agent/ 抽象层) | 运行时 ✅(ClaudeCodeExecutor.Execute 调 claude -p --output-format stream-json) | 文档 ✅ | 反事实 ✅
- 调研报告:briefs/research-1-N1-ai_cli现状.md
- 关键事实:
  - `internal/agent/interface.go` 定义 AgentExecutor/AgentSession 接口
  - `internal/agent/claudecode/executor.go` 实现 ClaudeCodeExecutor
  - `config.ClaudeCodePath` 配置 claude 二进制路径

### N2-claude code 耦合点:rick 模块依赖 claude code 的位置
- 置信度:1.0(高)
- 信源验证:代码原文 ✅(13 处调用点穷举) | 运行时 ✅(仅 doing.go 走接口) | 文档 ✅(subagent 是 rick 概念非 claude) | 反事实 ✅
- 调研报告:briefs/research-2-N2-claude-code耦合点.md
- 关键事实:
  - 13 处 claude 调用点,仅 doing.go:204 走 AgentExecutor 接口
  - 其余 12 处直接 exec.Command(plan/easy/dream/learning/human_loop/ctrl/tools_plan_check/runner)
  - claude 特有协议:NDJSON(stream-json)、--dangerously-skip-permissions/--session-id/--resume/-p
  - prompt 模板无 claude 专有 API,均为文本+路径占位符

### N3-agent 类型与启动路径:rick 支持的 agent 类型/启动路径
- 置信度:1.0(高)
- 信源验证:代码原文 ✅(9 命令入口穷举) | 运行时 ✅ | 文档 ✅(MEMORY.md 三阶段) | 反事实 ✅
- 调研报告:briefs/research-3-N3-agent类型与启动路径.md
- 关键事实:
  - 9 个命令入口:plan/doing/easy/learning/dream/human-loop/ctrl/tools/tools-plan-check
  - AgentExecutor 接口仅 1 个实现(ClaudeCodeExecutor),仅 doing.go 注入
  - 其余 8 命令直接 exec.Command claude

### N4-PI agent 概念:PI agent 在仓库中是否出现
- 置信度:0.0(低)
- 信源验证:代码原文 ❌(零匹配) | 运行时 ❌(无实现) | 文档 ❌(无定义) | 反事实 ❌(无代码可改)
- 调研报告:briefs/research-4-N4-PI-agent概念.md
- R7 上报:PI agent 在 rick 仓库无任何定义性引用,需 human 澄清

### N5-ai_cli 扩展点结构:是否已抽象 agent 类型接口,PI agent 接入改动
- 置信度:1.0(高)
- 信源验证:代码原文 ✅(AgentExecutor 接口已存在) | 运行时 ✅(反事实验证 doing.go 注入点松耦合) | 文档 ✅ | 反事实 ✅(修改 doing.go:204 → go build 报错 → git restore 还原)
- 调研报告:briefs/research-5-N5-ai_cli扩展点结构.md
- 关键事实:
  - 已有抽象:agent.AgentExecutor/AgentSession 接口
  - PI agent 接入需 5 处改动:新增 piagent 实现 + doing.go 替换 + 12 处 exec.Command 重构为 CLI agent 接口 + 配置泛化 + flag 按 agent 分发
  - 反事实还原确认:git restore 后 grep -c "claudecode.NewExecutor" = 1

## R7 上报项(无法达高置信度的叶节点)

- **N4-PI agent 概念**(置信度 0.0):PI agent 在 rick 仓库(Go 代码 + .rick 文档)中零匹配,仅在本次 sense_loop 协议文件中出现。无法通过代码/文档调研达高置信度。需 human 澄清:
  1. PI agent 全称/来源(外部框架?美团内部?自创?)
  2. PI agent 协议规格(输入输出格式、调用方式、与 claude code 差异)
  3. PI agent 实现位置(已有可执行文件?还是需 rick 实现?)

## 整合摘要

总节点数 6(含根) | 高置信度叶节点 4(N1/N2/N3/N5) | R7 上报 1(N4)
