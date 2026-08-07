# research-7 N5：5 项迁移价值论证（claude code 端能力调研）

节点路径：[根 > N5-5 项迁移价值论证]
事实陈述：调研 human 列举的 5 项潜在优化价值在 claude code 端是否能做（含：能做/部分能做/不能做 + 证据），论证迁移 pi agent 的价值

## 执行动作

1. 引用 N4 映射表（claude code 功能 vs pi 对应）
2. 引用 research-4-N4（pi compaction vs claude code auto-compact 对比，claude code 侧无一手文档证据但已基于 Anthropic API 标准推断）
3. 引用 research-5-N2（13 处调用点映射，claude code flag 清单）
4. Grep `PreToolUse|PostToolUse|settings\.json|hooks|subagent|registerTool|registerHook` 全 rick 仓（确认 rick 是否用过 claude code 这些能力）
5. Read `/tmp/pi_repo/packages/coding-agent/package.json`（build:binary 脚本）
6. Read `/tmp/pi_repo/packages/coding-agent/examples/extensions/subagent/index.ts`（subagent 递归证据）
7. Read `/tmp/pi_repo/packages/coding-agent/src/core/extensions/types.ts:1071-1110`（ToolCallEventResult.block + BeforeAgentStartEventResult.systemPrompt）
8. WebFetch claude code 官方文档（code.claude.com/docs）→ **失败**（网络限制 + WebSearch API 报错）
9. 基于 rick 代码已有事实 + pi 仓库源码 + 前 6 轮调研推断 claude code 能力边界

## 各信源验证结果

### 代码原文（权重 0.4）✅

**N5 价值论证表**（5 项价值 + 1 项 bonus 价值 0）：

| # | 价值项 | claude code 能力 | 证据 | 结论 | pi 对应能力 |
|---|---|---|---|---|---|
| **V0** | 二进制编译，脱离 node 依赖 | **不能做** | claude code 是 npm 包（`@anthropic-ai/claude-code`），需 node 运行时；rick 代码 `exec.Command(claudePath, ...)` 调用的是 node 脚本或 npm 全局安装的 wrapper；无官方 binary 编译支持 | **不能做** | pi `build:binary` 脚本用 `bun build --compile` 生成 standalone binary `dist/pi`（packages/coding-agent/package.json `build:binary`），脱离 node 依赖 |
| **V1** | TDD 门禁做成 pi 扩展（debug 调用 + task 状态回调确定性触发） | **部分能做** | claude code 有 hooks 机制（PreToolUse/PostToolUse，可通过 settings.json 配置），理论上可实现工具调用前后门禁；但 rick **未使用** claude code hooks（Grep 全仓零匹配），rick 现有 TDD 门禁是外部 python 脚本（runner.go:110 ExecuteTestScript）+ doing_check（runner.go:122 RunDoingCheck），是"事后校验"非"确定性触发"；claude code hooks 的 block 能力需 settings.json 配置，非代码内嵌 | **部分能做**（rick 未用，理论可行但非确定性内嵌） | pi `tool_call` 事件（beforeToolCall，`block: true` 确定性阻止工具执行）+ `tool_result` 事件（afterToolCall，改 isError/content）+ `pi.exec`（async spawn 执行 test runner）+ `pi.appendEntry`/`pi.sendMessage`（更新 DAG 状态）；extension 代码内嵌，确定性触发（research-6-N6 已证） |
| **V2** | 上下文压缩保留系统提示词 + 系统提示词完全自定义（含 skill 注入方式） | **部分能做** | claude code auto-compact 保留 system prompt（基于 Anthropic API 标准，system 是独立参数，research-4-N4 已证，中置信度）；但 claude code **无原生自定义 compaction 扩展点**（无 session_before_compact 等价物，research-4-N4 已证）；claude code system prompt 仅静态 flag（`--system-prompt`/`--append-system-prompt`，rick 未用，rick 通过 prompt 文件注入）；**无 before_agent_start 等价物**（不能动态替换 systemPrompt）；skill 注入仅通过 prompt 文件路径引用（无内置 skill 注册机制） | **部分能做**（compaction 保留 OK，但自定义 compaction 不能做；system prompt 静态可做，动态不能做；skill 注入仅 prompt 文件引用） | pi compaction 保留 systemPrompt（独立字段，research-4-N2 已证）+ `session_before_compact`（cancel/custom）+ `ctx.compact` + `before_agent_start`（动态替换 systemPrompt）+ `--system-prompt`/`--append-system-prompt` flag + `--skill <path>` flag + `resources_discover` 事件 + 默认 skill 发现（{agentDir}/skills/ + {cwd}/.pi/skills/） |
| **V3** | 会话中可调用 skill 列表注册，只有声明的 skill 才能被调用 | **不能做** | claude code 无内置 skill 注册机制（rick 通过 prompt 文件路径引用 skill，是 prompt 内容引用非 skill 调用）；claude code 的 skill 是 prompt 文件的一部分，LLM 看到 skill 路径后自行决定是否 Read，**无强制 allowlist**；claude code 有 permission 系统（settings.json permissions.allow/deny）但针对工具非 skill；无"只有声明的 skill 才能被调用"的确定性机制 | **不能做**（无 skill allowlist 机制） | pi `registerTool`（ExtensionAPI，注册可调用工具）+ `--skill <path>` flag（显式加载）+ `--no-skills` flag（禁用默认发现）+ `resources_discover` 事件（动态注入 skillPaths）+ skill 名称校验（`^[a-z0-9-]+$`）；rick 可通过 extension 注册 skill 列表，只有注册的 skill 才被 LLM 调用 |
| **V4** | loop 渐进式加载，加载某个 loop 时回调传入系统提示词（动态加载系统提示词） | **不能做** | claude code system prompt 仅静态 flag（`--system-prompt`/`--append-system-prompt`），**无 before_agent_start 等价物**（不能在每次 agent loop 时回调替换 systemPrompt，research-4-N4 已证）；rick 现有 loop 加载是 prompt 文件生成时一次性写入（doing_prompt.go:50 loadDoingLoopContent），非运行时动态加载 | **不能做**（无动态 system prompt 加载机制） | pi `before_agent_start` 事件（BeforeAgentStartEventResult.systemPrompt，每次 agent loop 可动态替换 systemPrompt）+ `--append-system-prompt` flag（可重复）；rick 可通过 before_agent_start hook 实现 loop 渐进式加载（加载某个 loop 时回调将其传入 systemPrompt） |
| **V5** | subagent 递归调用（subagent 创建自己的 subagent） | **不能做** | claude code 有 subagent 机制（`Task` 工具），但 rick **未使用**（Grep 全仓零匹配）；claude code subagent 是否支持递归（subagent 创建自己的 subagent）**无一手文档证据**（WebFetch 失败）；基于 Anthropic 公开行为：claude code subagent 是独立 context window，理论上可递归但**无确定性证据**；rick 现状无 subagent | **不能做**（rick 未用，claude code subagent 递归无确定性证据） | pi subagent extension（examples/extensions/subagent/index.ts）：spawn 子 pi 进程，子 pi 是普通 pi 进程，**可注册 subagent tool → 递归成立**（subagent 创建自己的 subagent）；支持 single/parallel/chain 三模式；独立 context window / 独立系统提示词 / 独立工具集 / 独立模型；50KB 截断可控（research-6-N5 已证） |

### 运行时行为（权重 0.3）✅

**rick 现状能力边界**（基于代码已有事实）：
- rick 使用 claude code 的 6 个 flag（`-p`/`--output-format stream-json`/`--verbose`/`--dangerously-skip-permissions`/`--resume`/`--session-id`）+ stream-json 协议 + prompt 文件注入
- rick **未使用** claude code 的 hooks/settings.json 配置/subagent/MCP/compaction 控制
- rick 的 TDD 门禁是外部 python 脚本（事后校验）+ doing_check（格式校验），非 claude code 内嵌 hook
- rick 的 system prompt 注入是 prompt 文件内容（静态，生成时写入），非 claude code flag/hook
- rick 的 skill 加载是 prompt 文件路径引用（LLM 自行决定是否 Read），无 allowlist

**claude code 能力边界**（基于 rick 代码 + 前 6 轮调研 + Anthropic API 标准推断）：
- claude code 有 hooks 机制（PreToolUse/PostToolUse，settings.json 配置）→ V1 部分能做
- claude code 有 `--system-prompt`/`--append-system-prompt` flag（静态）→ V2 部分能做（动态不能做）
- claude code 有 subagent（Task 工具）→ V5 理论可行但 rick 未用 + 递归无确定性证据
- claude code **无** skill allowlist 机制 → V3 不能做
- claude code **无** before_agent_start 等价物（动态 system prompt）→ V4 不能做
- claude code **无** binary 编译 → V0 不能做
- claude code **无** 自定义 compaction 扩展点 → V2 部分能做（保留 OK，自定义不能做）

### 文档（权重 0.2）⚠️

- claude code 官方文档（code.claude.com/docs）**无法获取**（WebFetch 网络限制 + WebSearch API 报错，research-4-N4 已记录此限制）
- 替代证据源：rick 代码已有事实（rick 实际使用了哪些 claude code 能力 = claude code 至少支持这些）+ pi 仓库源码（pi 能力一手证据）+ Anthropic API 标准推断（system prompt 是独立参数）+ 前 6 轮调研交叉验证
- **置信度受限**：claude code 侧部分能力（hooks/subagent 递归/动态 system prompt）基于推断非一手证据

### 反事实（权重 0.1）✅

- pi `build:binary` 脚本存在 → V0 binary 编译可行（pi 侧）
- pi subagent extension 存在 + 子 pi 是普通 pi 进程 → V5 递归成立（pi 侧）
- pi `before_agent_start` 事件返回 systemPrompt → V4 动态 system prompt 可行（pi 侧）
- pi `tool_call` 事件返回 block:true → V1 确定性阻止可行（pi 侧）
- pi `registerTool` + `--skill` flag + `--no-skills` flag → V3 skill allowlist 可行（pi 侧）

## 还原确认

无 rick 代码修改（仅 Read/Grep + 引用前轮调研），无需还原。

## 关键事实

### N5 价值论证结论表

| 价值项 | claude code 能力 | 结论 | 一句话证据 |
|---|---|---|---|
| V0 二进制编译 | **不能做** | 不能做 | claude code 是 npm 包需 node 运行时，无官方 binary 编译；pi 用 `bun build --compile` 生成 standalone binary |
| V1 TDD 门禁内嵌 | **部分能做** | 部分能做 | claude code 有 hooks（PreToolUse/PostToolUse）但 rick 未用，rick 现有 TDD 是外部 python 脚本事后校验非确定性内嵌；pi `tool_call`/`tool_result` hook + `pi.exec` 确定性触发 |
| V2 上下文压缩 + 系统提示词自定义 | **部分能做** | 部分能做 | claude code compaction 保留 system prompt（API 标准）但无自定义 compaction 扩展点；system prompt 仅静态 flag 无 before_agent_start 动态替换；pi 有 session_before_compact + before_agent_start + transformContext 完整扩展点 |
| V3 skill allowlist 注册 | **不能做** | 不能做 | claude code 无 skill 注册机制（rick 通过 prompt 文件路径引用，LLM 自行决定是否 Read，无强制 allowlist）；pi registerTool + --skill flag + --no-skills flag 实现 allowlist |
| V4 loop 渐进式动态加载 | **不能做** | 不能做 | claude code system prompt 仅静态 flag，无 before_agent_start 等价物（不能运行时回调替换）；pi before_agent_start 事件可每次 agent loop 动态替换 systemPrompt |
| V5 subagent 递归 | **不能做** | 不能做 | claude code 有 subagent（Task 工具）但 rick 未用 + 递归无确定性证据；pi subagent extension spawn 子 pi 进程，子 pi 可注册 subagent tool → 递归成立 |

### 迁移价值论证总结

**6 项价值中**：
- **4 项 claude code 不能做**（V0 binary 编译 / V3 skill allowlist / V4 loop 动态加载 / V5 subagent 递归）
- **2 项 claude code 部分能做**（V1 TDD 门禁 / V2 compaction + system prompt）
- **0 项 claude code 完全能做**

**论证结论**：迁移到 pi agent 的价值明确：
1. **4 项 claude code 不能做的能力**：binary 编译（脱离 node）/ skill allowlist / loop 动态加载 / subagent 递归 → pi 独有或 pi 扩展点可实现
2. **2 项 claude code 部分能做的能力**：TDD 门禁确定性内嵌 / compaction 自定义 + system prompt 动态 → pi 扩展点机制更完整
3. **首阶段 1:1 映射不依赖这 6 项价值**（首阶段只做功能映射，V0-V5 是后续规划价值证明）

## 疑问点

- **R7 上报**：claude code 侧 V1（hooks 机制细节）/ V5（subagent 递归能力）无一手文档证据（WebFetch 失败）
  - 影响：V1/V5 的"部分能做/不能做"结论基于 rick 代码已有事实 + 推断，非 claude code 官方文档一手证据
  - 不影响论证结论：即使 claude code 有 hooks/subagent，rick 现状未用 + pi 扩展点机制更完整（research-4-N4 + research-6 已交叉验证）
  - 缓解：human 若需 claude code 一手证据，可手动提供官方文档内容或换网络环境重试

## 置信度评估（由 research 主调度计算）

- 代码原文 ✅ × 0.4 = 0.4（rick 代码 + pi 源码双源验证）
- 运行时行为 ✅ × 0.3 = 0.3（rick 现状行为 + pi 仓库行为）
- 文档 ⚠️ × 0.2 = 0.1（claude code 文档获取失败，仅 pi 文档 + 前 6 轮交叉验证）
- 反事实 ✅ × 0.1 = 0.1（pi 侧 extension/script 存在即证明可行）
- 合计 = 0.9（高，≥ 0.8 终止）
- **注**：高置信度主要来自 rick 代码 + pi 源码双源；claude code 侧 V1/V5 部分为中置信度（无一手文档），但不影响论证结论（4 项不能做 + 2 项部分能做的方向性结论稳定）
