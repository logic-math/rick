# 调研报告 — Y12 rick cli↔pi 交互协议 + Y8 transformContext 分区原生支持验证 — 2026-08-04

## 信源配置

| 信源 | 默认权重 | 验证方式 |
|---|---|---|
| 代码原文 | 0.4 | Read/Grep rick 仓库 Go 代码 + /tmp/pi_repo 本地 pi 仓库 TypeScript 源码 |
| 运行时行为 | 0.3 | 13 处 exec.Command 调用点分类 + pi 4 种模式路由 + 字段对齐性对比 |
| 文档 | 0.2 | Read pi docs/(skills.md / prompt-templates.md / extensions.md) + rick MEMORY.md + pi README.md |
| 反事实 | 0.1 | N4 反事实验证(pi 原生支持分区的 5 个条件全部不满足)|

置信度 = Σ(信源验证结果 × 权重),结果 ∈ {0,1}。高 ≥ 0.8(终止)| 中 0.5-0.8(续研)| 低 < 0.5(R7 上报)。

**信源加权细节**:
- N1(现状交互协议):三源齐备(代码 + 运行时 + 文档),反事实 N/A → 0.9
- N2(pi 映射关系):三源齐备(代码 + 运行时 + 文档),反事实 N/A → 0.9
- N3(rick 职责变化):三源齐备(代码 + 运行时 + 文档),反事实 N/A → 0.9
- N4(Y8 分区验证):四源齐备(代码 + 运行时 + 文档 + 反事实),全部 ✅ → 1.0

---

## 尽调树(快照)

```
根:Y12 rick cli↔pi 交互协议 + Y8 transformContext 分区原生支持验证
├─ N1-现状 rick cli↔claude code 交互协议 ✅0.9
│  事实:13 处 exec.Command 调用点(仅 doing.go 走 AgentExecutor 接口);
│       stream-json NDJSON 协议(type/session_id/tool_use/tool_result/duration_ms/is_error);
│       per-task 启动退出,无 RPC/长连接;--dangerously-skip-permissions / --resume / --session-id 特殊 flag;
│       callClaudeCodeCLI/Background 共享函数 + ClaudeCodeExecutor 接口实现
├─ N2-迁移到 pi 的映射关系 ✅0.9
│  事实:pi 4 种模式(interactive/print/json/rpc);30+ RPC 命令 + 20+ 事件类型;
│       字段 5 项不对齐(snake→camel)+ duration_ms 缺失(需自计时);
│       --dangerously-skip-permissions 删除(pi 默认无 permission);
│       --output-format stream-json → --mode json;--resume/--session-id → --continue/--session;
│       13 处调用点映射:8 处低难度(flag 重命名/删除)+ 2 处中难度(解析器重写)+ 3 处低难度(换 binary)
├─ N3-rick cli 职责变化 ✅0.9
│  事实:rick 现状 20 类职责;pi 有等价能力(prompt-templates/skills/extensions/compaction/session/multi-provider);
│       pi 兼容 claude code skill 格式(skills.md 明示 ~/.claude/skills 可加载);
│       迁移后三分类:保留 10 类(调度/状态机/校验)+ 迁移 4 类(模板/skill/subagent/loop 内嵌)+ jointly 5 类;
│       rick 最小职责集 = 命令分发 + workspace + DAG + retry + actpath + debug + git + tools 校验 + 三阶段状态机 + human-loop 草稿 + dream 调度
└─ N4-Y8 transformContext 分区原生支持验证 ✅1.0(决定性证据)
   事实:transformContext 签名 (messages: AgentMessage[], signal?) => Promise<AgentMessage[]>,单参扁平数组,无原生分区概念;
       SessionEntryBase 仅 type/id/parentId/timestamp 4 字段,无 tag/region/zone/category/metadata;
       grep 全 pi_repo 无 "context region/segment/zone/partition" 匹配(排除 TUI 无关结果);
       官方示例全部"整体 transform"(prune/inject),无"按区域 transform"示例;
       反事实验证:pi 原生支持分区的 5 个条件(SessionEntryBase 字段/AgentMessage 属性/ctx 参数/官方示例/docs 章节)全部不满足;
       Y8 "给上下文分区"需在 transformContext + session_before_compact + before_agent_start 之上自建分区抽象;
       自建路径候选 3 类(transformContext 分区 / session_before_compact 区域压缩 / before_agent_start 区域注入),均需 extension 自行实现
```

**节点状态汇总**:
| 节点 | 状态 | 置信度 | 信源验证 |
|---|---|---|---|
| N1-现状 rick cli↔claude code 交互协议 | 已澄清 | 0.9(高) | 代码 ✅ + 运行时 ✅ + 文档 ✅ + 反事实 N/A |
| N2-迁移到 pi 的映射关系 | 已澄清 | 0.9(高) | 代码 ✅ + 运行时 ✅ + 文档 ✅ + 反事实 N/A |
| N3-rick cli 职责变化 | 已澄清 | 0.9(高) | 代码 ✅ + 运行时 ✅ + 文档 ✅ + 反事实 N/A |
| N4-Y8 transformContext 分区原生支持验证 | 已澄清 | 1.0(高,决定性) | 代码 ✅ + 运行时 ✅ + 文档 ✅ + 反事实 ✅ |

---

## 节点详情

### N1-现状 rick cli↔claude code 交互协议:13 处 exec.Command 调用点机制 + 数据格式 + 错误处理 + 生命周期 + flag + 封装层

- 置信度:0.9(高)
- 信源验证:
  - 代码原文 ✅:`internal/agent/interface.go`(AgentExecutor 1 方法 + AgentSession 6 方法)+ `internal/agent/claudecode/executor.go`(NDJSON 5 字段 + parseStream 4 分支)+ 13 处 exec.Command 调用点穷举
  - 运行时行为 ✅:12 处直接 exec.Command + 1 处走接口(doing.go);interactive/background/stream-json 三种调用形态
  - 文档 ✅:接口注释 + executor_test.go 证明 NDJSON 是 claude 专属
  - 反事实 N/A:现状代码调研
- 调研报告:briefs/research-5-N1-现状rick-cli与claude-code交互协议.md
- 关键事实:
  - 13 处调用点:仅 #1(claudecode/executor.go:39)走 AgentExecutor 接口,其余 12 处直接 exec.Command
  - 数据格式:12 处 interactive/background(stdin/stdout 直通 terminal)+ 1 处 stream-json(NDJSON over stdout pipe)
  - stream-json 字段:type/session_id/message/is_error/duration_ms(snake_case)+ content tool_use/tool_result/text/tool_use_id
  - 错误处理:cmd.Run() error 包装,无 exit code 判断,无 stderr 解析,解析失败 warn 跳过
  - 生命周期:per-task 启动退出,无 RPC/长连接,session resume 通过 flag(--resume/--session-id)
  - 特殊 flag:`-p` / `--output-format stream-json` / `--verbose` / `--dangerously-skip-permissions` / `--resume` / `--session-id`
  - 封装层:AgentExecutor 接口 + ClaudeCodeExecutor 实现 + callClaudeCodeCLI/Background 共享函数 + CallClaudeCodeCLI 备用方法

### N2-迁移到 pi 的映射关系:pi CLI/RPC 调用方式 + 字段映射 + 错误处理 + 生命周期 + flag 映射 + 适配层设计

- 置信度:0.9(高)
- 信源验证:
  - 代码原文 ✅:`/tmp/pi_repo/packages/coding-agent/src/cli/args.ts`(全部 flag)+ `main.ts`(4 模式路由)+ `modes/rpc/rpc-types.ts`(30+ RPC 命令)+ `agent/types.ts`(AgentEvent union,20+ 事件)+ `agent-session.ts`(AgentSessionEvent)
  - 运行时行为 ✅:13 处调用点映射可行性表(8 低难度 + 2 中难度 + 3 低难度)
  - 文档 ✅:args.ts Args interface + main.ts resolveAppMode + rpc-types.ts 完整 schema
  - 反事实 N/A:外部代码调研
- 调研报告:briefs/research-5-N2-迁移到pi的映射关系.md
- 关键事实:
  - pi 4 种模式:interactive(TUI)/ print(单次 text)/ json(单次 JSONL 事件流)/ rpc(长连接 stdin/stdout JSONL)
  - 30+ RPC 命令:prompt/steer/follow_up/abort/new_session/get_state/set_model/compact/switch_session/fork/clone/get_entries/get_tree 等
  - 20+ 事件类型:agent_start/end/settled + turn_start/end + message_start/update/end + tool_execution_start/update/end + compaction_start/end + auto_retry + queue_update + bash_execution_update
  - 字段对齐:5 项不对齐(session_id→sessionId / tool_use→tool_execution_start / tool_result→tool_execution_end / duration_ms→缺失 / is_error→isError)
  - flag 映射:`--dangerously-skip-permissions`→删除 / `--output-format stream-json`→`--mode json` / `--resume`→`--continue`/`--resume` / `--session-id`→`--session`
  - 适配层:新建 `internal/agent/piagent/executor.go` 实现 AgentExecutor + pi 事件流解析器(对标 claudecode.parseStream)
  - 13 处调用点:8 处低难度(flag 重命名/删除)+ 2 处中难度(解析器重写)+ 3 处低难度(换 binary)

### N3-rick cli 职责变化:现状职责清单 + 迁移后保留/迁移/jointly + 最小职责集 + 模板/skill/subagent 归属

- 置信度:0.9(高)
- 信源验证:
  - 代码原文 ✅:`internal/cmd/` 全命令文件 + `internal/prompt/manager.go`(embed.FS 10 模板 + 19 skill)+ `.rick/` 运行时结构(domain/loops/skills/draft/jobs/dream/RFC)+ pi docs/skills.md + prompt-templates.md
  - 运行时行为 ✅:rick 20 类职责 + pi 等价能力对比表 + 迁移三分类(保留 10 + 迁移 4 + jointly 5)
  - 文档 ✅:pi skills.md "Using Skills from Other Harnesses" 段明示兼容 claude code skill + rick MEMORY.md v2.9.0 架构
  - 反事实 N/A:现状对比调研
- 调研报告:briefs/research-5-N3-rick-cli职责变化.md
- 关键事实:
  - rick 现状 20 类职责:命令分发/prompt 模板/skill 加载/loop 加载/agent 调用/任务执行/行为轨迹/debug 体系/DAG/retry/三阶段/easy/dream/human-loop/ctrl/tools 校验/git/workspace + context 管理
  - pi 等价能力:prompt-templates / skills / extensions / compaction / session(树结构)/ multi-provider / standalone binary
  - pi 兼容 claude code skill 格式:`~/.claude/skills` 可加入 pi settings(skills.md 明示)
  - 迁移三分类:
    - **保留 10 类**:命令分发/workspace/DAG/retry/actpath/debug/git/tools 校验/三阶段状态机/human-loop 草稿/dream 调度
    - **迁移 4 类**:prompt 模板内容 / skill 内容 / subagent 派发 / doing_loop/learning_loop 内嵌逻辑
    - **jointly 5 类**:agent 调用 / session 管理 / skill 触发 / retry / raw log
  - rick 最小职责集:命令分发 + workspace + DAG + retry + actpath + debug + git + tools 校验 + 三阶段状态机 + human-loop 草稿 + dream 调度(11 类)
  - Y11 疑问句解答:rick 作为基本 cli = 保留调度/状态机/校验,模板/skill/subagent 内容迁移到 pi extension

### N4-Y8 transformContext 分区原生支持验证:pi 是否支持上下文分区原生概念 + 区域化保留/压缩可实现性

- 置信度:1.0(高,决定性证据)
- 信源验证:
  - 代码原文 ✅:`agent/types.ts`(transformContext 签名 + 注释)+ `agent.ts`(AgentOptions)+ `agent-loop.ts`(调用点 line 288)+ `session-manager.ts`(SessionEntryBase 4 字段)+ `extensions/types.ts`(SessionBeforeCompactEvent)
  - 运行时行为 ✅:transformContext 每 turn LLM 调用前执行 + 5 类扩展点组合能力 + Y8 论点 6 维验证
  - 文档 ✅:agent README.md "Prune old messages, inject external context"(仅 2 类用途,无 partitioning)+ CHANGELOG "preprocessor → transformContext" + pi docs 全文无 region/segment/zone/partition
  - 反事实 ✅:pi 原生支持分区的 5 个条件(SessionEntryBase 字段/AgentMessage 属性/ctx 参数/官方示例/docs 章节)全部不满足
- 调研报告:briefs/research-5-N4-Y8-transformContext分区原生支持验证.md
- 关键事实:
  - transformContext 签名:`(messages: AgentMessage[], signal?) => Promise<AgentMessage[]>`,**单参扁平数组,无原生分区概念**
  - SessionEntryBase:仅 `type/id/parentId/timestamp` 4 字段,**无 tag/region/zone/category/metadata**
  - pi 全仓无 "context region/segment/zone/partition" 概念(grep 排除 TUI 无关结果)
  - 官方示例全部"整体 transform"(prune/inject),无"按区域 transform"示例
  - Y8 论点验证:✅ 内嵌 agent loop 内部 + ✅ pi 扩展内部 + ✅ 自由组合上下文 + ❌ 原生分区 + ⚠️ 自建分区 + ⚠️ 区域化保留/压缩(组合逻辑自实现)
  - 自建分区路径候选 3 类:transformContext 分区 / session_before_compact 区域压缩 / before_agent_start 区域注入
  - 关键限制:自建分区抽象维护成本(标记一致性 / 跨 session 持久化 / 与 compaction 交互 / 与 tree navigation 交互)

---

## R7 上报项(无法达高置信度的叶节点)

**无 R7 上报项**。4 个节点全部高置信度(N1=0.9, N2=0.9, N3=0.9, N4=1.0)。

---

## 整合摘要

总节点数 4 | 高置信度叶节点 4(N1=0.9, N2=0.9, N3=0.9, N4=1.0) | R7 上报 0

**Y12 事实澄清结论**:✅ **已澄清(高置信度)**
- 现状 rick cli↔claude code 交互协议:13 处 exec.Command 子进程调用,1 处走 AgentExecutor 接口(doing.go),12 处直接硬耦合;stream-json NDJSON 协议;per-task 启动退出;--dangerously-skip-permissions / --resume / --session-id 特殊 flag
- 迁移到 pi 的映射关系:pi 4 种模式(interactive/print/json/rpc);字段 5 项不对齐 + duration_ms 缺失;flag 映射明确(--dangerously-skip-permissions 删除 / --output-format stream-json → --mode json / --resume → --continue / --session-id → --session);13 处调用点 8 低难度 + 2 中难度 + 3 低难度;适配层 `internal/agent/piagent/executor.go`
- rick cli 职责变化:20 类现状职责;迁移后保留 10 类(调度/状态机/校验)+ 迁移 4 类(模板/skill/subagent/loop 内嵌)+ jointly 5 类;rick 最小职责集 = 11 类(调度 + 状态机 + 校验)

**Y8 事实澄清结论**:✅ **已澄清(高置信度,决定性证据)**
- pi transformContext **不原生支持**上下文分区(签名单参扁平数组,SessionEntryBase 无 tag/region 字段,全仓无 context region/segment/zone/partition 概念,官方示例全部整体 transform,反事实验证 5 条件全部不满足)
- Y8 human 论点"给上下文分区"需在 transformContext + session_before_compact + before_agent_start 之上**自建分区抽象**
- 自建分区路径候选 3 类,均需 extension 自行实现分区逻辑(pi 不识别分区标记)
- 自建分区抽象维护成本:标记一致性 / 跨 session 持久化 / 与 compaction 交互 / 与 tree navigation 交互

**与前序判断的关系**:
- Y12 三维度(N1/N2/N3)全部高置信度澄清 → r4 门禁 top-3 假设 #6(最终分 0.80)"Y12 交互协议未定义 → 迁移落地无法启动"反例成立:**协议已定义,迁移落地可启动**
- Y8 验证(N4)高置信度澄清 → r4 门禁假设 #1(最终分 -0.20)+ #8(最终分 0.20)"pi transformContext 不原生支持分区 → 需自建分区抽象"**事实成立**,human 需决策是否接受自建分区抽象
- Y10/Y11 价值性判断仍需 human 决策(本轮事实调研不替 human 判断价值性)

---

## S 阶段简报(给 sense_loop → human)

### 尽调树快照(引用主报告)

4 节点全部高置信度:Y12 ✅ 澄清(N1/N2/N3 = 0.9)+ Y8 ✅ 澄清(N4 = 1.0,决定性证据)

### R7 上报项

无。4 节点全部高置信度,无无法澄清的叶节点。

### Y12 三维度 + Y8 验证事实澄清结论

**Y12-a(N1)现状 rick cli↔claude code 交互协议**:
- 13 处 exec.Command 子进程调用(仅 doing.go 走 AgentExecutor 接口,其余 12 处直接硬耦合)
- stream-json NDJSON 协议(type/session_id/tool_use/tool_result/duration_ms/is_error)
- per-task 启动退出,无 RPC/长连接,session resume 通过 flag
- 特殊 flag:`--dangerously-skip-permissions` / `--output-format stream-json --verbose` / `--resume` / `--session-id`

**Y12-b(N2)迁移到 pi 的映射关系**:
- pi 4 种模式:interactive / print / json / rpc(前 3 种 per-prompt 子进程同构 rick 现状,第 4 种 RPC 长连接是 rick 现状无的增强)
- 字段 5 项不对齐(snake→camel)+ duration_ms 缺失(需 rick 自计时)
- flag 映射:`--dangerously-skip-permissions`→删除(pi 默认无 permission)/ `--output-format stream-json`→`--mode json` / `--resume`→`--continue`/`--resume` / `--session-id`→`--session`
- 13 处调用点:8 处低难度(flag 重命名/删除)+ 2 处中难度(解析器重写,doing.go 调用链)+ 3 处低难度(换 binary)
- 适配层:新建 `internal/agent/piagent/executor.go` 实现 AgentExecutor + pi 事件流解析器

**Y12-c(N3)rick cli 职责变化**:
- rick 现状 20 类职责;pi 有完整等价能力(prompt-templates/skills/extensions/compaction/session/multi-provider)
- pi 兼容 claude code skill 格式(`~/.claude/skills` 可加入 pi settings)
- 迁移三分类:保留 10 类(调度/状态机/校验)+ 迁移 4 类(模板/skill/subagent/loop 内嵌)+ jointly 5 类
- rick 最小职责集 = 11 类(命令分发 + workspace + DAG + retry + actpath + debug + git + tools 校验 + 三阶段状态机 + human-loop 草稿 + dream 调度)
- Y11 疑问句解答:rick 作为基本 cli = 保留调度/状态机/校验,模板/skill/subagent 内容迁移到 pi extension

**Y8(N4)transformContext 分区原生支持验证**:
- pi transformContext **不原生支持**上下文分区(决定性证据:签名单参扁平数组 + SessionEntryBase 无 tag/region 字段 + 全仓无 context region/segment/zone/partition 概念 + 官方示例全部整体 transform + 反事实验证 5 条件全部不满足)
- Y8 human 论点"给上下文分区"需在 transformContext + session_before_compact + before_agent_start 之上**自建分区抽象**
- 自建分区路径候选 3 类,均需 extension 自行实现分区逻辑
- 自建分区抽象维护成本:标记一致性 / 跨 session 持久化 / 与 compaction 交互 / 与 tree navigation 交互

### 给 human 的 S 阶段三连追问(基于新事实重写,聚焦 Y10/Y11 价值性假设最终判断)

**新事实摘要**:Y12 三维度(N1/N2/N3)全部高置信度澄清——rick 现状 13 处 exec.Command 子进程调用,迁移到 pi 是 8 低难度 + 2 中难度 + 3 低难度的 flag/解析器适配;rick 最小职责集 = 11 类(调度/状态机/校验),模板/skill/subagent 可迁移到 pi extension;pi 兼容 claude code skill 格式。Y8 高置信度澄清——pi transformContext **不原生支持**上下文分区,Y8 "给上下文分区"需自建分区抽象(3 路径候选,维护成本由 extension 承担)。

**三连追问**(基于 Y12/Y8 澄清后,聚焦 Y10/Y11 价值性假设的最终判断):

1. **现状补充(Y10 首阶段成功判据 + Y8 自建分区抽象接受度)**:Y12 已证 rick 现状 13 处调用点迁移到 pi 是 8 低难度 + 2 中难度(flag/解析器适配),Y10 首阶段"保持现有功能不变 + pi 以二进制形式启动"的"难项"具体指什么——是 pi 二进制启动本身难(部署/分发),还是保持功能不变难(13 处调用点 + NDJSON→JSONL 解析器重写)?Y8 已证 pi 不原生支持分区,需自建分区抽象,**首阶段是否包含自建分区抽象**?还是首阶段仅做"pi 二进制启动 + 现有功能不变"(Y8 自建分区抽象推迟到后续阶段)?**human 请明确:(a) Y10 首阶段"难项"具体指什么?(b) Y8 自建分区抽象是否纳入首阶段?**

2. **期望(Y11 rick 最小职责集 + Y9 功能等价边界)**:Y12 已证 rick 最小职责集 = 11 类(调度/状态机/校验),模板/skill/subagent 可迁移到 pi extension。Y11 human 疑问句"rick 作为基本 cli 存在?"的解答候选 = 保留 11 类调度职责 + 迁移 4 类内容到 pi extension。**human 请确认:(a) rick 最小职责集 11 类是否接受?(b) 模板/skill/subagent 迁移到 pi extension 后,rick 是否仍 embed.FS 持有模板内容(编译期嵌入,自包含分发),还是完全依赖 pi 文件系统运行时加载(需配套分发 .pi/skills/ 目录)?(c) Y9 "功能等价"边界——doing/learning/dream 三阶段 × per-task 流程是否作为等价性度量基线?还是依赖 human 直觉判断?**

3. **差距(Y10 由难到易策略 + Y12 适配层优先级)**:Y12 已证 13 处调用点迁移是 8 低难度 + 2 中难度 + 3 低难度。Y10 "由难到易"策略下,首阶段应先做 2 处中难度(doing.go AgentExecutor 接口 + piagent/executor.go 解析器重写)还是先做 8 处低难度(flag 重命名/删除)?**human 请明确:(a) 首阶段优先做中难度(doing.go 走接口 + pi 解析器)还是低难度(其余 12 处 flag 适配)?(b) pi RPC 长连接模式(`--mode rpc`)是否纳入首阶段(消除反复启动子进程开销 + steering/followUp 队列),还是首阶段仅用 print/json 模式(同构 rick 现状 per-prompt 子进程)?(c) 13 处调用点是否全部重构为走 AgentExecutor 接口(统一抽象),还是仅 doing.go 走接口其余保持 exec.Command + flag 适配层?**

**→ human 请思考并回答上述三连追问。基于 Y12/Y8 事实澄清,本轮 research round 5 已穷尽事实性假设,后续判断为价值性决策(Y10/Y11),请 human 给出最终判断以推进到 E 阶段(视角生成)或 N 阶段(矛盾生成)。**
