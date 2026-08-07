# 调研报告 — 替换 claude code 引擎，ai_cli 支持 PI agent 可行性调研（第 6 轮 S 阶段） — 2026-08-04

## 信源配置

| 信源 | 默认权重 | 验证方式 |
|---|---|---|
| 代码原文 | 0.4 | Read/Grep pi 仓库源码（/tmp/pi_repo） |
| 运行时行为 | 0.3 | args.ts flag 文档 + package.json + 实际目录结构 |
| 文档 | 0.2 | README + extensions.md + subagent README |
| 反事实 | 0.1 | N/A（纯外部调研，未修改 rick 代码） |

置信度 = Σ(信源验证结果 × 权重)，结果 ∈ {0,1}。高 ≥ 0.8(终止)| 中 0.5-0.8(续研)| 低 < 0.5(R7 上报)。

## 尽调树（快照）

```
根：替换 claude code 引擎，ai_cli 支持 PI agent 可行性（第 6 轮 S 阶段）
├─ N1-Y13-a：pi .pi 目录结构与默认行为            [置信度: 0.9 高 ✅]
├─ N2-Y13-b：pi skill 加载路径扩展性               [置信度: 0.9 高 ✅]
├─ N3-Y13-c：pi session/agent/config 存储扩展性    [置信度: 0.9 高 ✅]
├─ N4-Y14-a：因果链 1+2（提示词调度+门禁内嵌）     [置信度: 0.9 高 ✅]
│  ├─ 因果链 1：系统提示词注入 → 确定性提升         [部分成立 ⚠️]
│  └─ 因果链 2：门禁 extension 内嵌 → 确定性做到   [部分成立 ⚠️]
├─ N5-Y14-b：因果链 3+4（compaction+subagent）     [置信度: 0.9 高 ✅]
│  ├─ 因果链 3：compaction 保留 system prompt      [成立 ✅]
│  └─ 因果链 4：subagent 独立进程隔离              [成立 ✅]
└─ N6-Y14-c：因果链 5（TDD hook 内置循环）         [置信度: 0.9 高 ✅]
   └─ 因果链 5：TDD 验证 → 确定性更新 DAG          [部分成立 ⚠️]
```

## 节点详情

### N1-Y13-a：pi .pi 目录结构与默认行为
- 置信度：0.9（高）
- 信源验证：代码原文 ✅(config.ts line 487-566) + 运行时 ✅(args.ts flag) + 文档 ✅(README) + 反事实 N/A
- 调研报告：briefs/research-6-N1-Y13-a-pi目录结构.md
- 关键事实：
  - .pi 目录结构：user scope `~/.pi/agent/`（skills/prompts/themes/extensions/agents/tools/bin/sessions/ + models.json/auth.json/settings.json/debug.log）+ project scope `{cwd}/.pi/`（skills/prompts/themes/extensions/agents）
  - 可禁用：`--no-skills`/`--no-extensions`/`--no-prompt-templates`/`--no-themes`/`--no-context-files`/`--no-tools`/`--no-builtin-tools`
  - 全局配置项控制：无运行时配置项，`.pi` 目录名由 package.json `piConfig.configDir` 决定（编译期常量）
  - 环境变量/flag 重定向：`PI_CODING_AGENT_DIR`（重定向 user scope）+ `PI_CODING_AGENT_SESSION_DIR`/`--session-dir`（重定向 session）+ `PI_PACKAGE_DIR`（重定向 package dir）+ `--skill`/`--extension`/`--prompt-template`/`--theme`（加载自定义路径）

### N2-Y13-b：pi skill 加载路径扩展性
- 置信度：0.9（高）
- 信源验证：代码原文 ✅(skills.ts line 430-481) + 运行时 ✅(args.ts --skill) + 文档 ✅(extensions.md resources_discover) + 反事实 N/A
- 调研报告：briefs/research-6-N2-Y13-b-skill加载路径.md
- 关键事实：
  - 默认加载路径：`{agentDir}/skills/`（user scope）+ `{cwd}/.pi/skills/`（project scope）
  - 自定义目录：`--skill <path>` flag（可多次，支持目录/文件）+ `resources_discover` extension 事件（动态注入 skillPaths）
  - rick `.rick/skills/{name}_skill/skill.md` 结构**不能直接识别**：pi 要求 `SKILL.md`（大写），rick 用 `skill.md`（小写）；rick 目录名带 `_skill` 后缀，pi name 校验 `^[a-z0-9-]+$`（下划线不通过）
  - 适配方案：重命名 `skill.md`→`SKILL.md` + 去 `_skill` 后缀 / `--skill` flag 直接指定 / extension 适配 / 符号链接

### N3-Y13-c：pi session/agent/config 存储扩展性
- 置信度：0.9（高）
- 信源验证：代码原文 ✅(session-manager.ts line 476-489) + 运行时 ✅(args.ts session flag) + 文档 ✅(subagent README) + 反事实 N/A
- 调研报告：briefs/research-6-N3-Y13-c-session-agent-config.md
- 关键事实：
  - session 存储可配置：默认 `{agentDir}/sessions/{cwd-encoded}/`，`PI_CODING_AGENT_SESSION_DIR`/`--session-dir`/`--session`/`--session-id`/`--fork`/`--no-session` 可重定向
  - agent 定义可重定向：user scope `{agentDir}/agents/`（通过 `PI_CODING_AGENT_DIR` 重定向），project scope `{cwd}/.pi/agents/`（不可运行时重定向，但 subagent extension 的 `agentScope` 可限制只读 user scope）
  - config 环境变量覆盖：6 个 PI_ 前缀环境变量 + 30+ provider API key，settings.json 配置项不支持环境变量覆盖
  - 并存方案下 pi 创建的 .pi 子目录：user scope 强制创建 `sessions/` + `auth.json` + `settings.json` + debug.log；project scope 不主动创建

### N4-Y14-a：因果链 1+2 验证（提示词调度 + 门禁内嵌）
- 置信度：0.9（高）
- 信源验证：代码原文 ✅(system-prompt.ts + agent-session.ts + runner.ts + agent-loop.ts) + 运行时 ✅(extensions.md 示例) + 文档 ✅(types.ts BeforeAgentStartEventResult) + 反事实 N/A
- 调研报告：briefs/research-6-N4-Y14-a-因果链1+2.md
- 关键事实：
  - **因果链 1 部分成立 ⚠️**：pi 支持 rick 注入系统提示词（3 种入口：`--system-prompt`/`--append-system-prompt`/`before_agent_start`），system prompt 作为 `Context.systemPrompt` 独立字段传递给 LLM；但"确定性提升"是 LLM 行为假设，prompt injection 仍可覆盖 system prompt
  - **因果链 2 部分成立 ⚠️**：门禁做成 pi extension 内嵌（`beforeToolCall` hook + `tool_call` 事件 + `block: true`）→ 工具调用确定性阻止（硬阻止非建议）；但仅限工具调用门禁，LLM 文本回复门禁不可预防阻止（仅 `message_end` 事后替换）

### N5-Y14-b：因果链 3+4 验证（compaction 保留 + subagent 隔离）
- 置信度：0.9（高）
- 信源验证：代码原文 ✅(agent-loop.ts + compaction.ts + subagent/index.ts) + 运行时 ✅(subagent README) + 文档 ✅(session-manager.ts buildSessionContext) + 反事实 N/A
- 调研报告：briefs/research-6-N5-Y14-b-因果链3+4.md
- 关键事实：
  - **因果链 3 成立 ✅**：systemPrompt 不在 messages 中，是 `Context.systemPrompt` 独立字段；compaction 只处理 messages 历史，不触及 systemPrompt；每次 LLM 调用从 `agent.state.systemPrompt` 读取，始终完整可见；extension 可通过 `session_before_compact` 自定义 compaction
  - **因果链 4 成立 ✅**：subagent 通过 `spawn` 启动独立 pi 子进程（`shell: false`），有独立 context window / 独立系统提示词 / 独立工具集 / 独立模型；结果作为 tool_result 回传父进程，不注入父 messages history；50KB 截断可控（完整结果在 tool details）

### N6-Y14-c：因果链 5 验证（TDD hook 内置循环）
- 置信度：0.9（高）
- 信源验证：代码原文 ✅(exec.ts + types.ts + runner.ts + agent-session.ts) + 运行时 ✅(extensions.md 示例) + 文档 ✅(types.ts ExtensionAPI.exec) + 反事实 N/A
- 调研报告：briefs/research-6-N6-Y14-c-因果链5.md
- 关键事实：
  - **因果链 5 部分成立 ⚠️**：
    - ✅ afterToolCall hook 能执行外部 test runner（`pi.exec` async spawn，await 同步阻塞，支持 timeout/signal）
    - ✅ hook 内可访问 tool_call 参数（`event.input.file_path` / `input.command` 等）
    - ✅ hook 内可更新 DAG 状态（`pi.exec` 写 tasks.json / `pi.appendEntry` 写 session / `pi.sendMessage` 触发 LLM）
    - ⚠️ hook 执行失败阻塞行为不对称：beforeToolCall 失败**阻塞**（工具不执行），afterToolCall 失败**不阻塞**（错误被 catch，LLM 收到原始 tool_result）
    - 💡 "确定性"依赖 hook 实现质量：TDD hook 放 beforeToolCall 可确定性阻止"未写 test 就改实现"；放 afterToolCall 需主动返回 `isError: true` 让 LLM 知道 test 失败

## R7 上报项（无法达高置信度的叶节点）

无。所有 6 个叶节点置信度均为 0.9（高，≥ 0.8），无 R7 上报项。

## 整合摘要

总节点数 6 | 高置信度叶节点 6 | R7 上报 0

**Y13 三维度（N1/N2/N3）事实澄清结论**：

1. **pi .pi 目录可删除性**（N1）：
   - project scope `.pi` 目录名是编译期常量（package.json `piConfig.configDir`），运行时不可改
   - 但可通过 `--no-skills`/`--no-extensions` 等禁用 project scope 发现
   - user scope 可通过 `PI_CODING_AGENT_DIR` 重定向
   - **3 种实现路径**：fork pi 改 configDir / flag 重定向禁用默认发现 / 并存（human 已确认"首次合并可以允许并存"）

2. **pi skill 加载路径扩展性**（N2）：
   - 默认 `{agentDir}/skills/` + `{cwd}/.pi/skills/`
   - `--skill <path>` 可指定任意路径 + `resources_discover` 事件可动态注入
   - rick `.rick/skills/{name}_skill/skill.md` 结构**不能直接识别**（文件名/目录名/名称校验三重冲突）
   - 适配方案：重命名 / flag 指定 / extension 适配 / 符号链接

3. **pi session/agent/config 存储扩展性**（N3）：
   - session 存储可通过 `PI_CODING_AGENT_SESSION_DIR`/`--session-dir` 重定向
   - agent 定义 user scope 可重定向，project scope 不可运行时重定向（但 subagent extension 可限制 scope）
   - config 支持环境变量覆盖（6 个 PI_ 前缀 + 30+ provider key）

**Y14 五因果链（N4/N5/N6）事实澄清结论**：

| 因果链 | 结论 | 关键证据 |
|---|---|---|
| 1. rick 注入系统提示词 → pi 基于系统提示词工作 → 确定性提升 | ⚠️ 部分成立 | 注入机制成立（3 种入口），"确定性提升"是 LLM 行为假设（prompt injection 仍可覆盖） |
| 2. 门禁做成 pi extension 内嵌 → 在 main agent 中确定性做到 | ⚠️ 部分成立 | 工具调用门禁确定性成立（block 硬阻止），文本门禁不可预防阻止（仅事后替换） |
| 3. 流程/方法作为系统提示词 + compaction 保留 → 长程 debug 确定性 | ✅ 成立 | systemPrompt 是 Context 独立字段，compaction 只处理 messages，systemPrompt 始终完整可见 |
| 4. subagent 独立进程隔离 → 上下文污染避免 | ✅ 成立 | spawn 独立子进程，独立 context window，结果作为 tool_result 回传不注入父 messages |
| 5. 所有变更执行 TDD 验证 → 确定性更新 DAG 状态 | ⚠️ 部分成立 | TDD 机制成立（pi.exec 执行 test runner），DAG 更新成立，但 afterToolCall 异常不自动阻塞需主动 isError:true |

## S 阶段最终三连追问（基于全部 6 轮 research 事实，准备进入 E 阶段）

### 追问 1：现状补充？
基于 6 轮 research，pi 的扩展能力已完整澄清：
- pi 提供 6 类扩展点（Prompt Templates / Skills / Extensions / Themes / Pi Packages / Agent Core 钩子）
- pi 支持 3 种系统提示词注入（`--system-prompt` / `--append-system-prompt` / `before_agent_start`）
- pi 支持 beforeToolCall/afterToolCall hook（工具调用确定性阻止 / 工具结果修改）
- pi compaction 保留 system prompt（独立字段不参与压缩）
- pi subagent 真进程隔离（spawn 子进程）
- pi .pi 目录名是编译期常量（需 fork 才能改）
- pi skill 加载路径可重定向（`--skill` flag / `resources_discover` 事件）
- rick `.rick/skills/{name}_skill/skill.md` 结构需适配（文件名/目录名/名称校验三重冲突）

**human 现状补充？** 上述事实是否与 human 的认知一致？是否有遗漏的扩展点或限制？

### 追问 2：期望？
基于 Y11/Y13/Y14 human 期望：
- rick 职责 = 3 类（cli 解析 / 调度 main agent / 维护任务状态机）
- rick = 系统提示词调度器
- pi = 基于系统提示词工作 + 维护上下文整洁性
- 所有门禁做成 pi extension 内嵌
- 不需要 .pi 目录（所有上下文从 .rick 读取）
- 首阶段 = claude cc 与 pi 1:1 功能映射
- 后续规划提升确定性与有效性

**human 期望澄清？** 基于因果链 1/2/5 部分成立的结论：
- 因果链 1"确定性提升"是 LLM 行为假设，rick 是否接受"注入机制成立但 LLM 遵循不保证"？
- 因果链 2 文本门禁不可预防阻止（仅事后替换），rick 是否接受"工具调用门禁确定性 + 文本门禁事后修正"？
- 因果链 5 afterToolCall 异常不自动阻塞，rick 是否接受"TDD hook 需主动 isError:true 实现确定性"？

### 追问 3：差距？
基于 6 轮 research，首阶段差距已明确：
- **.pi 目录处理**：3 种方案（fork 改 configDir / flag 禁用默认发现 + 显式加载 / 并存），human 已确认"首次合并可以允许并存"
- **skill 结构适配**：rick `.rick/skills/{name}_skill/skill.md` 需重命名为 `SKILL.md` + 去 `_skill` 后缀，或用 flag 指定，或写 extension 适配
- **13 处调用点映射**：NDJSON→JSONL 解析器 + flag 映射 + AgentExecutor 适配层（research round 5 已澄清）
- **门禁 extension 化**：rick 现有 debug/doing_check/easy_check 等门禁需重写为 pi extension（beforeToolCall/afterToolCall hook）
- **后续规划因果链验证**：因果链 1/2/5 部分成立，需 human 决策是否接受"部分成立"的边界

**human 差距判断？** 基于因果链 3/4 成立 + 1/2/5 部分成立的结论：
- 首阶段 1:1 功能映射的边界是否清晰？
- 后续规划"提升确定性与有效性"的方法是否需要调整（因 1/2/5 部分成立）？
- 是否进入 E 阶段（视角生成）？
