# S 阶段简报 — 问题确认 — 2026-08-04 ~ 2026-08-06

## 主题
替换 claude code 引擎，ai_cli 支持 PI agent 可行性调研

## 8 轮 research 澄清事实总览

| 轮次 | 调研焦点 | 关键结论 |
|---|---|---|
| R1 | rick 侧 ai_cli 现状 + claude code 耦合点 + PI agent 概念 | ai_cli = internal/agent/ 抽象层；13 处调用点（仅 1 处走接口）；PI agent 在仓库零匹配（R7 上报） |
| R2 | pi 框架定位 + 扩展点 + 运行时 + 协议 | pi 6 类扩展点；Node.js（非 Go）；standalone binary；字段 5 项不对齐 + duration_ms 缺失；No MCP/sub-agents/permission/plan-mode |
| R3 | standalone binary + skill 系统级 + provider + subagent | Y1/Y3/Y5/Y6 全部澄清；pi 6 平台预编译 30-44 MB；registerTool 真系统级；30+ provider per-prompt 切换；官方 subagent extension 范例 |
| R4 | Y7 pi compaction 是否保留 system prompt | Y7 ✅ compaction 保留 system prompt（N2=1.0 决定性证据）+ 5 类自定义扩展点 |
| R5 | Y12 交互协议 + Y8 分区验证 | 13 处调用点映射 + flag 映射 + rick 职责三分类；Y8 pi 不原生支持分区需自建抽象 |
| R6 | Y13 .pi 目录 + Y14 5 因果链 | .pi 目录编译期常量；skill 加载可重定向；5 因果链 2 成立 + 3 部分成立 |
| R7 | 完整调用链映射 + 5 项迁移价值论证 | 25 项 claude code 功能映射（8 完全等价 + 5 部分等价 + 9 需适配 + 1 需新建 + 5 pi 增强）；6 项价值 4 项不能做 + 2 项部分能做 |
| R8 | Y15 去掉 .pi 最小可行方法 | 方案 A 纯 env/flag（0 行 pi 代码）为最小可行方法 |

## 关键事实清单

### pi 框架
- 开源轻量级 agent（github earendil-works/pi），MIT，v0.83.0，5394 commits，30 天 563 commits
- Node.js ≥22.19.0（非 Go），无 Go binding
- 支持 Bun 编译 standalone binary（30-44 MB，6 平台预编译，真零 Node 依赖）
- 显式 No MCP/No sub-agents/No permission/No plan-mode（设计哲学）
- 4 种模式：interactive / print / json / rpc（前 3 种同构 rick 现状，rpc 长连接增强）

### pi 扩展能力
- 6 类扩展点：Prompt Templates / Skills / Extensions / Themes / Pi Packages / Agent Core 钩子
- 5 类 compaction 自定义扩展点：session_before_compact / session_compact / ctx.compact / before_agent_start / transformContext
- 30+ provider + 11 种 API 协议，per-prompt 模型切换原生支持（Cross-Provider Handoffs）
- 官方 subagent extension 范例（spawn 子 pi 进程，独立 context，支持 single/parallel/chain）
- compaction 保留 system prompt（systemPrompt 是 Context 独立字段，compaction 只处理 messages）

### rick 现状
- ai_cli = `internal/agent/` 抽象层（AgentExecutor 接口 + ClaudeCodeExecutor 实现）
- 13 处 exec.Command 调用点（仅 doing.go:204 走接口，其余 12 处直接硬耦合）
- stream-json NDJSON 协议（type/session_id/tool_use/tool_result/duration_ms/is_error）
- per-task 启动退出无 RPC
- 9 个命令入口（plan/doing/easy/learning/dream/human-loop/ctrl/tools/tools-plan-check）

### 功能映射表摘要（25 项）
- 完全等价：8 项（-p / --verbose / --resume / interactive / prompt 文件 / stdin / session 持久化 / 权限跳过 / MCP）
- 部分等价：5 项（assistant/tool_use/text/tool_result 字段映射）
- 需适配：9 项（stream-json→--mode json / flag 映射 / system prompt 注入 / skill 路径）
- 需新建：1 项（duration_ms 自计时）
- pi 增强（rick 未用）：5 项

## 首阶段边界（human 已确认）

### 目标
- pi 二进制安装（不依赖 node）
- 与 claude code 保持相同功能
- rick 现有功能不变性

### 4 决策项
1. JSONL 替代 NDJSON（解析器重写）
2. .rick/ + .pi/ 并存（首阶段；后续用方案 A 去掉 .pi）
3. skill 继续用 prompt 文件路径引用
4. duration 自维护（rick startTime → agent_settled 计时）

### 核心成本
- 1 个解析器重写（claudecode/executor.go → piagent/executor.go）
- 12 处 flag 适配（8 低难度 + 2 中难度 + 2 低难度）
- 1 个 duration 自维护

## Y15 去掉 .pi 方案（方案 A）

pi 从不主动创建 .pi 目录（源码级验证：0 处 mkdirSync）。最小可行方法 = 纯 env/flag：

```
PI_CODING_AGENT_DIR=.rick/.pi-agent
PI_CODING_AGENT_SESSION_DIR=.rick/sessions
--no-skills --no-extensions --no-prompt-templates --no-themes --no-context-files
--skill .rick/skills/ --extension .rick/extensions/
```

遗留 3 处硬编码（project scope settings.json/npm/git）在 rick 不创建对应文件的情况下不影响 pi 运行。

## 后续规划价值（V0-V5）

| 价值项 | claude code 能力 | 论证结论 |
|---|---|---|
| V0 二进制编译 | 不能做 | claude code 是 npm 包需 node；pi 用 bun build --compile |
| V1 TDD 门禁内嵌 | 部分能做 | claude code 有 hooks 但 rick 未用；pi tool_call/tool_result hook 确定性触发 |
| V2 compaction + system prompt 自定义 | 部分能做 | claude code compaction 保留 OK 但无自定义扩展点；system prompt 仅静态 flag |
| V3 skill allowlist 注册 | 不能做 | claude code 无 skill 注册机制；pi registerTool + --skill/--no-skills |
| V4 loop 渐进式动态加载 | 不能做 | claude code system prompt 仅静态 flag；pi before_agent_start 可动态替换 |
| V5 subagent 递归 | 不能做 | claude code 有 subagent 但 rick 未用；pi subagent spawn 子 pi 可递归 |

## S 阶段通过条件

human 显式确认：
- "现状：可以了没有遗漏"
- "期望: 其实可以算达成了，证明了迁移 pi 是会带来极大收益的"
- "差距: 却具体的实现计划了"
- "可以了可以收敛继续了！"

## 后续阶段

- E 阶段（视角生成）：基于 R1-R8 事实，跨领域调研多视角候选
- N 阶段（矛盾判断）：系统论描述符 + 主要矛盾三维打分
- S-R 阶段（辩证逆转）：若 N2 无主要矛盾则强制触发
- EC 阶段（良知批判）：human 自判跃迁方向

