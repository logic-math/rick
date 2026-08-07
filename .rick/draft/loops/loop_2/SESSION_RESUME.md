# SESSION_RESUME.md — loop_2 human-loop 恢复点

> 本文件供下次会话恢复使用。本次 session 于 2026-08-07 中断，human 要求保留全部工作。

---

## 主题
替换 claude code 引擎，ai_cli 支持 PI agent 可行性调研

## 协议路径
- main agent 协议：`/Users/sunquan/ai_coding/CODING/rick/.rick/draft/loops/loop_2/prompts/sense_loop.md`
- research subagent：`prompts/research.md`
- think subagent：`prompts/think.md`
- exporter subagent：`prompts/exporter.md`

## 当前进度

### S 阶段（问题确认）— ✅ 已收敛
- **8 轮 research 完成**（research-report-1 ~ research-report-8 + 各 N 子节点 brief）
- **5 轮批判门禁完成**（批判门禁-S / S-r2 / S-r3 / S-r4 / S-r5，第 5 轮升级 human 介入后突破）
- **human 显式确认收敛**（2026-08-06）：
  - "现状：可以了没有遗漏"
  - "期望: 其实可以算达成了，证明了迁移 pi 是会带来极大收益的"
  - "差距: 却具体的实现计划了"
  - "可以了可以收敛继续了！"

### E 阶段（视角生成）— ⏸ 进行中（中断点）
- **research-E 完成**（5 视角候选：软件工程 / 系统工程 / 认知科学 / 哲学 / 生物生态）
- **human 拒绝全部 5 候选**，提出原创"盒子里的 LLM"思想实验视角
- **批判门禁-E-r1 完成**（重试 1/5）：❌ 未通过，识别 5 个未澄清 Y
- **当前状态**：待 human 选择推进方式

## 恢复点（下次会话从这里继续）

### 待 human 决策的推进方式
1. **直接回答 Y-E2/Y-E3/Y-E5**（价值性/哲学性，最关键；Y-E3 自指悖论需 human 显式回答）+ 并行派 research 调研 Y-E1/Y-E4（事实性）
2. **先 research 调研 Y-E1/Y-E4**（事实性），再回 human 回答 Y-E2/Y-E3/Y-E5
3. **直接回答全部 5 个 Y**（若 human 已有判断）
4. **其他**（human 指定）

### 5 个未澄清 Y（E 阶段批判门禁 r1 产出）

| Y | 类型 | 关键性 | 内容 |
|---|---|---|---|
| **Y-E1** | 事实性/哲学 | 哲学基石 | LLM 是否具备涌现推理能力？"记忆 vs 涌现"边界——human 论断的根基 |
| **Y-E2** | 价值性 | 核心因果链（不可逆 1.0） | 是否存在不随 G' 变化的元认知方法？若存在可训练元方法（CoT/ReAct/TDD），rick 价值前提崩塌 |
| **Y-E3** | 价值性/哲学 | **自指悖论（最关键阻塞）** | rick 方法是 G 内还是 G 外？若 G 内，论断是否自相矛盾？若 G 外，LLM 如何理解执行？ |
| Y-E4 | 事实性 | 前提边界 | 持续学习/在线学习/RAG 场景下，"G 永远过去式"是否仍是刚性前提？ |
| Y-E5 | 价值性 | 架构边界 | rick 与 pi 的不可替代边界？rick 哪些职责 pi 永远无法内化？ |

## 关键产出文件索引

### judgment.md（human 判断原话，逐条）
- `/Users/sunquan/ai_coding/CODING/rick/.rick/draft/loops/loop_2/judgment.md`
- 含 S 阶段全部 human 判断 + E 阶段 human 原创视角原话 + 批判门禁结论

### S 阶段简报（完整收敛总结）
- `briefs/S.md` — S 阶段 8 轮 research 事实总览 + 首阶段边界 + Y15 方案 + 后续价值

### research 主报告（8 轮 + E 阶段）
- `briefs/research-report.md` — R1（rick 侧 ai_cli 现状）
- `briefs/research-report-2.md` — R2（pi 框架定位/扩展点/运行时/协议）
- `briefs/research-report-3.md` — R3（standalone binary/skill 系统级/provider/subagent）
- `briefs/research-report-4.md` — R4（Y7 pi compaction 保留 system prompt）
- `briefs/research-report-5.md` — R5（Y12 交互协议 + Y8 分区验证）
- `briefs/research-report-6.md` — R6（Y13 .pi 目录 + Y14 5 因果链）
- `briefs/research-report-7.md` — R7（完整调用链映射 + 5 项迁移价值论证）
- `briefs/research-report-8.md` — R8（Y15 去掉 .pi 最小可行方法）
- `briefs/research-E.md` — E 阶段 5 视角候选

### research 子节点 brief（R1-R8 + E，共 40+ 文件）
- 命名规则：`briefs/research-{轮次}-{节点路径}.md` / `briefs/research-E-{视角名}.md`

### 批判门禁简报（S 5 轮 + E 1 轮）
- `briefs/批判门禁-S.md` — S 阶段 r1
- `briefs/批判门禁-S-r2.md` ~ `批判门禁-S-r5.md` — S 阶段 r2-r5
- `briefs/批判门禁-E-r1.md` — E 阶段 r1（当前中断点）

## S 阶段关键事实速查（供下次会话快速回忆）

### pi 框架
- 开源轻量级 agent（github earendil-works/pi），MIT，v0.83.0，5394 commits，30 天 563 commits
- Node.js ≥22.19.0（非 Go），无 Go binding
- 支持 Bun 编译 standalone binary（30-44 MB，6 平台预编译，真零 Node 依赖）
- 显式 No MCP/No sub-agents/No permission/No plan-mode
- 4 种模式：interactive / print / json / rpc

### pi 扩展能力
- 6 类扩展点：Prompt Templates / Skills / Extensions / Themes / Pi Packages / Agent Core 钩子
- 5 类 compaction 自定义扩展点
- 30+ provider，per-prompt 模型切换原生支持
- 官方 subagent extension 范例（spawn 子 pi 进程，独立 context）
- compaction 保留 system prompt（systemPrompt 是 Context 独立字段）

### rick 现状
- ai_cli = `internal/agent/`（AgentExecutor 接口 + ClaudeCodeExecutor 实现）
- 13 处 exec.Command 调用点（仅 doing.go:204 走接口）
- stream-json NDJSON 协议
- 9 个命令入口

### 功能映射表（25 项）
- 完全等价：8 项
- 部分等价：5 项
- 需适配：9 项
- 需新建：1 项（duration_ms 自计时）
- pi 增强（rick 未用）：5 项

### 首阶段边界（human 已确认 4 决策项）
1. JSONL 替代 NDJSON（解析器重写）
2. .rick/ + .pi/ 并存（首阶段；后续用方案 A 去掉 .pi）
3. skill 继续用 prompt 文件路径引用
4. duration 自维护（rick startTime → agent_settled 计时）

### Y15 去掉 .pi 方案（方案 A，human 已选）
pi 从不主动创建 .pi 目录（源码级验证）。最小可行方法 = 纯 env/flag：
```
PI_CODING_AGENT_DIR=.rick/.pi-agent
PI_CODING_AGENT_SESSION_DIR=.rick/sessions
--no-skills --no-extensions --no-prompt-templates --no-themes --no-context-files
--skill .rick/skills/ --extension .rick/extensions/
```

### 后续规划价值（V0-V5，论证迁移 pi 的价值）
| 价值项 | claude code 能力 |
|---|---|
| V0 二进制编译 | 不能做 |
| V1 TDD 门禁内嵌 | 部分能做 |
| V2 compaction + system prompt 自定义 | 部分能做 |
| V3 skill allowlist 注册 | 不能做 |
| V4 loop 渐进式动态加载 | 不能做 |
| V5 subagent 递归 | 不能做 |

## E 阶段 human 原创视角核心论点（供下次会话回忆）

### 盒子里的 LLM 思想实验
- LLM = 被关在盒子里的全知天才（被动，无自主性）
- LLM 效果 = f(input)，相同 LLM 下效果取决于 input
- G = 训练数据集（过去式），G' = 人类面临的更新问题集
- LLM 智能提升时，若 G 不变则 input 有效性要求下降；若 G 也迁移则 input 必须弥补 G' 迁移

### human 核心论断
- LLM 本质 = 知识压缩 + 可能性拟合 + 记忆（推理也需 input 前提）
- LLM 只给可能性，结果由调用方负责
- 全知全能 LLM 是悖论（不能创造自己不知道的事）
- 解决 G' 的做事方法无法被 LLM 训练进去（方法随 G' 变化）
- **rick 价值**：解决 G' 问题域下无法被 LLM 训练覆盖的行为轨迹 → 抽象为做事方法 → 通过上下文工程实现
- **深度定制 pi**：为 G' 问题留下探索空间
- **架构定位**：rick = 引导程序，核心在 pi 之中

### 批判门禁 r1 识别的关键风险
- **Y-E2（不可逆 1.0）**：若存在可训练元方法（CoT/ReAct/TDD），rick 价值前提崩塌
- **Y-E3（自指悖论）**：rick 方法是 G 内还是 G 外？human 自身论断可能自相矛盾

## 下次会话恢复指令

1. 读取本文件了解进度
2. 读取 `judgment.md` 了解 human 全部判断原话
3. 读取 `briefs/批判门禁-E-r1.md` 了解未通过详情
4. 向 human 确认推进方式（1/2/3/4）
5. 继续 E 阶段 → N 阶段 → S-R 阶段 → EC 阶段 → exporter

## 协议约束提醒
- 每阶段重试上限 5 次（S 阶段已用完，E 阶段当前 r1）
- 反向回流上限 3 次
- N 阶段双追问强制（N1 矛盾生成 + N2 主要矛盾判断）
- N2 无主要矛盾 ⇒ 必须触发 S-R
- EC 阶段 sense_loop 只呈现回顾，不替 human 提议跃迁方向

## 信源限制
本次 session 中 WebSearch/WebFetch 多次因网络策略阻断不可用（尤其 claude code 官方文档 code.claude.com）。pi 侧调研基于 github 仓库 raw 源码 + 官方文档，置信度高。若下次会话网络恢复，可补 claude code 一手证据。

---

**最后更新**：2026-08-07
**session 状态**：中断保留，等待新会话恢复
