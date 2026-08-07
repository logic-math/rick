# 调研报告 — 替换 claude code 引擎,ai_cli 支持 PI agent 可行性调研(S 阶段二次调研 - pi 框架事实澄清) — 2026-08-04

## 信源配置

| 信源 | 权重 | 验证方式 |
|---|---|---|
| 代码原文 | 0.4 | git clone pi 仓库 + Read package.json/README/docs |
| 运行时行为 | 0.3 | git log/shortlog/tag 分析(活跃度验证) |
| 文档 | 0.2 | WebFetch pi.dev/raw.githubusercontent(README/extensions/rpc/json/sdk/session-format) |
| 反事实 | 0.1 | N/A(本轮纯外部调研,未改 rick 代码) |

置信度 = Σ(信源验证结果 × 权重), 高 ≥ 0.8(终止) | 中 0.5-0.8(续研) | 低 < 0.5(R7)

## 尽调树(快照)

```
根:替换 claude code 引擎,ai_cli 支持 PI agent 可行性调研(S 二次 - pi 框架事实澄清)
├─ N1-pi 框架定位与维护活跃度 ✅0.9(高)
│  事实:Pi = minimal terminal coding harness;MIT;v0.83.0;5394 commits;563 commits/30天;当天有 commit;Mario Zechner/earendil-works
├─ N2-pi 扩展点机制 ✅0.9(高)
│  事实:6 类扩展点(Prompt Templates/Skills/Extensions/Themes/Pi Packages/Agent Core 钩子);粒度细于 claude code --append-system-prompt + MCP;transformContext/convertToLlm/beforeToolCall/afterToolCall/shouldStopAfterTurn 程序化钩子 + 6 类事件
├─ N3-pi 运行时形态与调用方式 ✅0.9(高)
│  事实:Node.js ≥22.19.0(ESM/TS);5 种调用(CLI 交互/print/json/rpc + SDK);Bun 编译 standalone binary;无 Go binding;rick 集成路径 = CLI 子进程或 RPC 长连接
├─ N4-pi 会话/流式协议 ✅0.9(高)
│  事实:会话续接 ✅(--session/--continue/--fork,树结构 JSONL);流式 = JSONL over stdio;字段 5 项全不对齐(session_id→sessionId, tool_use→toolCall, tool_result→toolResult, duration_ms 缺失, is_error→isError)
└─ N5-pi 与 rick AgentExecutor 接口语义对齐性 ✅0.9(高)
   事实:接口语义可对齐(Execute 签名兼容,6 方法可填充);NDJSON 解析器需重写;12 处 exec.Command 可泛化(8 低难度 + 4 中难度);pi RPC 长连接优于反复启动子进程
```

## 节点详情

### N1-pi 框架定位与维护活跃度:pi 是什么,谁维护,多活跃
- 置信度:0.9(高)
- 信源验证:代码原文 ✅(package.json/LICENSE/README) | 运行时 ✅(git log 5394 commits, 563/30天) | 文档 ✅(pi.dev/npm/Discord/RFC) | 反事实 N/A
- 调研报告:briefs/research-2-N1-pi框架定位与维护活跃度.md
- 关键事实:
  - 定位:minimal terminal coding harness,"aggressively extensible"
  - 作者:Mario Zechner(badlogicgames)+ earendil-works 组织,多贡献者
  - 年龄:约 1 年(2025-08-09 至 2026-08-04)
  - 活跃度:极度活跃,5394 commits,30 天 563 commits,调研当天有新 commit,版本 v0.83.0
  - License:MIT
  - 不内置:MCP/sub-agents/permission/plan-mode/todos/background-bash(全交 extension)

### N2-pi 扩展点机制:扩展点类型与粒度对比 claude code
- 置信度:0.9(高)
- 信源验证:代码原文 ✅(README/extensions.md/agent README/SDK 120KB+17KB+36KB) | 运行时 ✅(Quick Start 示例 + Philosophy 段对比) | 文档 ✅ | 反事实 N/A
- 调研报告:briefs/research-2-N2-pi扩展点机制.md
- 关键事实:
  - 6 类扩展点:Prompt Templates / Skills / Extensions / Themes / Pi Packages / Agent Core 钩子
  - 粒度显著细于 claude code:transformContext(上下文裁剪)/convertToLlm(消息转换)/beforeToolCall+afterToolCall(工具拦截)/shouldStopAfterTurn(loop 控制)/6 类生命周期事件(可 block)/UI 替换/editor/widget/status/footer/overlay/provider 自定义
  - Extension 是 TypeScript 代码注入(jiti 运行时加载,无需编译)
  - 立场:No MCP(但 extension 可添加)/ No sub-agents(但可写 extension 实现)

### N3-pi 运行时形态与调用方式:运行时语言/调用方式/进程模型
- 置信度:0.9(高)
- 信源验证:代码原文 ✅(package.json engines/exports/bin) | 运行时 ✅(4 种调用方式 + Bun binary) | 文档 ✅(README Programmatic Usage + rpc.md) | 反事实 N/A
- 调研报告:briefs/research-2-N3-pi运行时形态与调用方式.md
- 关键事实:
  - 运行时:Node.js ≥22.19.0(ESM/TS),无 Go/Rust runtime
  - 5 种调用:CLI 交互 / CLI print / CLI json / CLI rpc / SDK 嵌入(Node only)
  - rick(Go)集成路径:CLI 子进程(`pi -p` 或 `pi --mode rpc`),无 Go binding
  - Standalone binary:✅(Bun 编译,`scripts/build-binaries.sh`),分发等同 claude code
  - 进程模型:CLI 单进程;RPC 长连接(一次启动多次 prompt);bash 子进程(cross-spawn)
  - 环境变量契约:PI_SESSION_ID/PI_SESSION_FILE/PI_PROVIDER/PI_MODEL/PI_REASONING_LEVEL

### N4-pi 会话/流式协议:会话续接/流式协议/输出字段
- 置信度:0.9(高)
- 信源验证:代码原文 ✅(session-format.md/json.md/rpc.md Events) | 运行时 ✅(jq 可解析 + Python/Node 客户端示例) | 文档 ✅ | 反事实 N/A
- 调研报告:briefs/research-2-N4-pi会话与流式协议.md
- 关键事实:
  - 会话续接:✅ 完整支持,语义等同或强于 claude code
    - `--session <id>` ≈ claude `--session-id`
    - `--continue`/`-c` ≈ claude `--continue`(rick 现用 `--resume`)
    - `--fork <id>` 强于 claude(原地分支)
    - 树结构 JSONL > claude 线性会话
  - 流式协议:JSONL over stdio(RPC 双向,JSON 单向 stdout),与 claude stream-json 同构
  - 字段对齐:❌ 5 项全不对齐
    - session_id → sessionId(camelCase)
    - tool_use → toolCall(content block type)
    - tool_result → toolResult(role)
    - duration_ms → **缺失**(pi 不输出,需 rick 自计时)
    - is_error → isError
  - pi 独有:steering/followUp 消息队列/compaction 事件/auto_retry 事件/extension_error 事件

### N5-pi 与 rick AgentExecutor 接口语义对齐性:接口对齐 + 12 处重构可行性
- 置信度:0.9(高)
- 信源验证:代码原文 ✅(interface.go/claudecode/executor.go/runner.go/13 处调用点) | 运行时 ✅(对比表 + 重构难度评估) | 文档 ✅ | 反事实 N/A
- 调研报告:briefs/research-2-N5-pi与rick-AgentExecutor语义对齐性.md
- 关键事实:
  - 接口语义:✅ 可对齐(Execute 签名兼容,AgentSession 6 方法可填充)
  - NDJSON 解析器:需重写(字段 schema 全变)
  - 12 处 exec.Command 泛化:✅ 可行(8 低难度 flag 重命名 + 4 中难度接口适配)
  - 收益点:RPC 长连接消除反复启动 + steering/followUp 队列(claude 无)
  - 阻碍点:字段不对齐 + duration_ms 缺失 + 无 --dangerously-skip-permissions 等价 flag
  - rick 现有架构优势:AgentExecutor 接口已抽象 + doing.go 已注入,新增 PiExecutor 是"加法"

## R7 上报项(无法达高置信度的叶节点)

无。本轮 5 个叶节点(N1-N5)置信度均为 0.9(高),无 R7 上报项。

## 整合摘要

总节点数 6(含根) | 高置信度叶节点 5(N1/N2/N3/N4/N5) | R7 上报 0

## Y1/Y3/Y6 三假设事实澄清结论

- **Y1(扩展点粒度)**:✅ 澄清。pi 提供 6 类扩展点,粒度显著细于 claude code 的 `--append-system-prompt` + MCP。关键差异:pi extension 是 TypeScript 代码注入,可访问 AgentState、拦截 tool_call、替换 convertToLlm、自定义 compaction、替换 UI、自定义 provider。claude code 仅有文本注入 + 外部 MCP 工具协议,无程序化钩子。
- **Y3(运行时形态)**:✅ 澄清。pi 是 Node.js ≥22.19.0(ESM/TS),非 Go。无 Go binding。rick(Go)集成路径:CLI 子进程(`pi -p` 或 `pi --mode rpc`),与 rick 现有 `exec.Command("claude", "-p", ...)` 同构。支持 Bun 编译 standalone binary(分发等同 claude code)。RPC 模式为非 Node 集成首选,长连接消除反复启动开销。
- **Y6(会话与流式协议)**:✅ 澄清。pi 完整支持会话续接(`--session`/`--continue`/`--fork`,树结构 JSONL,语义强于 claude code)。流式协议为 JSONL over stdio,与 claude stream-json 同构。但字段 schema 全不对齐(5 项:session_id/tool_use/tool_result/duration_ms/is_error),需重写解析器。pi 不输出 duration_ms(需 rick 自计时)。pi 独有 steering/followUp/compaction/auto_retry 事件。

## 给 human 的 S 阶段三连追问(基于新事实重写)

新事实摘要:pi 是 Node.js 极活跃开源 agent harness(v0.83.0, 5394 commits, 当天有 commit, MIT),6 类扩展点粒度细于 claude code(程序化钩子 + 事件系统 + UI 替换),5 种调用方式(CLI/RPC/SDK),会话续接完整支持,但流式字段 schema 与 claude code 全不对齐(需重写解析器 + 自计时 duration)。

1. **现状补充**:基于 pi 是 Node.js(非 Go)、无 Go binding、需 CLI 子进程集成这一事实,rick 当前 `exec.Command("claude", ...)` 模式与 pi `exec.Command("pi", ...)` 同构——你是否确认"CLI 子进程"是可接受的集成形态(而非要求 Go 原生 binding)?你是否接受 rick 部署时需额外安装 pi(Node.js ≥22.19.0 或 standalone binary)?
2. **期望**:基于 pi 扩展点粒度细于 claude code(程序化钩子 transformContext/convertToLlm/beforeToolCall/shouldStopAfterTurn + 6 类事件 + UI 替换)这一事实,你期望 rick 接入 pi 后**具体深度控制哪些行为**?(如:doing 阶段上下文裁剪?tool_call 拦截做 permission gate?custom compaction 策略?steering 消息队列注入?)——这些期望是否真需 pi 才能实现,还是可通过 rick 端 prompt 模板/doing.md 设计达成?
3. **差距**:基于 pi 流式字段 schema 与 rick 现有 NDJSON 解析器全不对齐(5 项字段 + duration_ms 缺失)这一事实,接入 pi 需新增 `internal/agent/piagent/executor.go` 解析器 + 12 处 exec.Command 重构(8 低难度 + 4 中难度)。你是否接受这个适配成本?若 pi RPC 长连接模式带来收益(消除反复启动 + steering 队列),是否值得将 12 处全部重构为走 AgentExecutor 接口(而非仅 doing.go)?
