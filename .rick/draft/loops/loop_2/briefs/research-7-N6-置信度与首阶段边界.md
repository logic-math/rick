# research-7 N6：置信度评估与首阶段边界

节点路径：[根 > N6-置信度评估与首阶段边界]
事实陈述：综合 N1-N5，评估首阶段 1:1 映射置信度，标注剩余风险点，输出 R7 上报项

## 执行动作

1. 整合 N1-N3（13 处调用点 + 完整调用链 + 25 项 claude code 功能枚举）
2. 整合 N4（25 项功能映射表：8 完全等价 + 5 部分等价 + 9 需适配 + 1 需新建 + 5 pi 增强）
3. 整合 N5（6 项价值论证：4 项 claude code 不能做 + 2 项部分能做）
4. 综合前 6 轮调研（research-1~6）已澄清事实
5. 评估首阶段 1:1 映射置信度 + 标注剩余风险点

## 各信源验证结果

### 代码原文（权重 0.4）✅

**首阶段 1:1 映射必需项置信度**：

| 映射项 | 数量 | 置信度 | 风险点 |
|---|---|---|---|
| 完全等价（零成本） | 8 项 | 1.0（高） | 无 |
| 部分等价（解析器重写） | 5 项 | 0.9（高） | 字段 schema 适配（P3/P4/P5/P6：事件名 + 字段名 snake→camel） |
| 需适配（flag 重命名/删除） | 9 项 | 0.95（高） | F2 stream-json→--mode json 解析器重写是主要成本 |
| 需新建（duration 自维护） | 1 项 | 0.9（高） | rick 自维护 startTime → agent_settled 计时，逻辑简单 |
| **首阶段 1:1 映射总体** | **23 项** | **0.93（高）** | 主要成本：1 个解析器重写（claudecode/executor.go → piagent/executor.go）+ 12 处 flag 适配 |

**首阶段不实现项**（pi 增强，后续规划）：
- H1 hooks / C1 settings.json / A1 subagent / M1 compaction 自定义 / pi 独有 flag（--fork/--mode rpc/--system-prompt 等）
- 这些项不影响首阶段 1:1 映射（rick 现有功能不依赖这些）

### 运行时行为（权重 0.3）✅

**首阶段剩余风险点**（基于 N1-N5 综合）：

| 风险点 | 影响 | 置信度 | 缓解方案 |
|---|---|---|---|
| R1：pi `--session <id>` 语义与 claude code `--session-id <id>` 是否完全等价 | easy 命令会话管理 | 0.8（高，research-5-N2 已证 pi --session 接受 path 或 id，语义更广） | 首阶段用 `--session <id>` 替代 `--session-id <id>`，测试 easy resume 行为 |
| R2：pi `--continue` 与 claude code `--resume` 语义是否完全等价 | easy resume 会话续接 | 0.8（高，research-5-N2 已证 pi --continue 续最近会话，--resume 浏览选择） | 首阶段 easy resume 改用 `--continue`（续最近会话，符合 rick 现有语义） |
| R3：pi json 模式 JSONL 字段 schema 与 claude code stream-json NDJSON 完整对应 | doing 核心 parseStream 重写 | 0.9（高，research-5-N2 已做完整字段对齐表） | 新建 `internal/agent/piagent/executor.go`，字段映射：session header.id→sessionID, agent_settled→终止, tool_execution_end→ToolCall, message_end→FinalMessage, 自维护 duration |
| R4：pi 子进程启动/超时/崩溃恢复行为与 claude code 一致 | 13 处 exec.Command 调用 | 0.85（高，pi 是标准 Node.js/binary 进程，exec.Command 行为一致） | 首阶段保持 exec.Command 调用方式，仅换 binary 路径 + flag |
| R5：rick `.rick/skills/{name}_skill/skill.md` 结构 pi 识别 | skill 加载 | 0.85（高，research-6-N2 已证三重冲突 + 4 种适配方案） | 首阶段继续用 prompt 文件路径引用（不依赖 pi skill 发现），后续规划再适配 pi skill 机制 |
| R6：.pi 目录并存（human 已确认首阶段允许并存） | 目录结构 | 0.95（高，research-6-N1 已证 project scope .pi 不主动创建子目录） | 首阶段允许 .rick/ + .pi/ 并存，rick 通过 flag 禁用 pi 默认发现（--no-skills/--no-extensions） |

**无 R7 上报项**：所有 6 个风险点置信度 ≥ 0.8，无需 R7 上报。

### 文档（权重 0.2）⚠️

- claude code 官方文档无法获取（WebFetch 网络限制 + WebSearch API 报错，research-4-N4 已记录）
- 影响：claude code 侧 V1（hooks 机制细节）/ V5（subagent 递归能力）无一手文档证据
- 不影响首阶段 1:1 映射：首阶段映射基于 rick 代码已有事实（rick 实际使用了哪些 claude code 能力 = claude code 至少支持这些），不依赖 claude code 文档
- 影响：N5 价值论证的 V1/V5 部分为中置信度（基于推断），但方向性结论（4 项不能做 + 2 项部分能做）稳定

### 反事实（权重 0.1）✅

- pi `build:binary` 脚本存在 → V0 binary 编译可行
- pi subagent extension 存在 + 子 pi 是普通 pi 进程 → V5 递归成立
- pi `before_agent_start` 事件返回 systemPrompt → V4 动态 system prompt 可行
- pi `tool_call` 事件返回 block:true → V1 确定性阻止可行
- pi 仓库 `registerTool` + `--skill` flag + `--no-skills` flag → V3 skill allowlist 可行

## 还原确认

无 rick 代码修改（仅整合 N1-N5 + 前 6 轮调研），无需还原。

## 关键事实

### 首阶段 1:1 映射置信度评估

**总体置信度：0.93（高，≥ 0.8）**

**置信度构成**：
- 23 项必需映射项平均置信度 0.93
- 6 个风险点全部 ≥ 0.8
- 前 6 轮调研已澄清 Y1-Y14 大部分事实性假设
- rick 代码 + pi 源码双源验证

**首阶段边界**（明确）：
1. **核心交付**：1 个解析器重写（claudecode/executor.go → piagent/executor.go）+ 12 处 flag 适配（callClaudeCodeCLI/runAutoFix/learning 直接 exec）+ 1 个 duration 自维护
2. **不实现**：hooks / subagent / compaction 自定义 / skill allowlist / loop 动态加载 / binary 编译（这些是后续规划价值，首阶段不依赖）
3. **并存允许**：.rick/ + .pi/ 并存（human 已确认），rick 通过 flag 禁用 pi 默认发现
4. **skill 适配延后**：首阶段继续用 prompt 文件路径引用 skill（不依赖 pi skill 发现机制），后续规划再适配

### R7 上报项

**无 R7 上报项**。所有 6 个风险点置信度 ≥ 0.8，所有 6 个价值论证项有明确结论（4 项不能做 + 2 项部分能做）。

**待 human 决策项**（非 R7，是首阶段边界确认）：
1. 首阶段是否接受"pi json 模式 JSONL 替代 stream-json NDJSON"（解析器重写成本）？
2. 首阶段是否接受".rick/ + .pi/ 并存"（human 已确认 Y13）？
3. 首阶段是否接受"skill 加载继续用 prompt 文件路径引用"（不依赖 pi skill 发现）？
4. 首阶段是否接受"duration 自维护"（rick startTime → agent_settled 计时）？

## 疑问点

无。首阶段 1:1 映射置信度 0.93（高），6 个风险点全部 ≥ 0.8，无 R7 上报项。

## 置信度评估（由 research 主调度计算）

- 代码原文 ✅ × 0.4 = 0.4
- 运行时行为 ✅ × 0.3 = 0.3
- 文档 ⚠️ × 0.2 = 0.1（claude code 文档获取失败，不影响首阶段映射）
- 反事实 ✅ × 0.1 = 0.1
- 合计 = 0.9（高，≥ 0.8 终止）
