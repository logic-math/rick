# 调研报告 — 替换 claude code 引擎,ai_cli 支持 PI agent 可行性调研(S 阶段三次调研 - pi 深度事实澄清) — 2026-08-04

## 信源配置

| 信源 | 权重 | 验证方式 |
|---|---|---|
| 代码原文 | 0.4 | GitHub API + raw.githubusercontent.com 抓取 pi 仓库源码(build-binaries.sh / package.json / extensions.md / subagent extension / pi-ai README / SDK 文档) |
| 运行时行为 | 0.3 | GitHub API 返回 v0.83.0 release 实际 asset 列表 + 下载次数 + Cross-Provider Handoffs 示例代码 + subagent extension spawn 实现 |
| 文档 | 0.2 | pi.dev README + extensions.md 120KB + subagent README + pi-ai README 79KB + SDK 文档 36KB |
| 反事实 | 0.1 | N/A(本轮纯外部调研,未改 rick 代码) |

置信度 = Σ(信源验证结果 × 权重), 高 ≥ 0.8(终止) | 中 0.5-0.8(续研) | 低 < 0.5(R7)

## 尽调树(快照)

```
根:替换 claude code 引擎,ai_cli 支持 PI agent 可行性调研(S 三次 - pi 深度事实澄清)
├─ N1-standalone binary 部署形态(Y1) ✅0.9(高)
│  事实:6 平台预编译 release artifact(darwin-arm64/x64, linux-x64/arm64, windows-x64/arm64);
│       binary 体积 30-44 MB;Bun 编译内嵌 runtime,真零 Node 依赖;
│       可选 --offline-model-data 内嵌 model catalog;分发形态等同 claude code
├─ N2-skill 系统级注册机制(Y3) ✅0.9(高)
│  事实:registerTool = LLM tool schema 系统级注册(进 system prompt Available tools + Guidelines + function calling schema);
│       registerCommand = CLI 斜杠命令(LLM 不可见);
│       rick skill 是流程描述 markdown,pi tool 是 TS 函数签名,语义不对齐;
│       流程描述型 skill 需重写为 TS extension 才能进 LLM tool schema
├─ N3-provider 接入与任务路由(Y5) ✅0.9(高)
│  事实:30+ 内置 provider + 11 种 API 协议;
│       per-prompt 模型切换原生支持(Cross-Provider Handoffs,同 context 内每次 complete() 可换 model);
│       AgentSession.setModel 运行时动态切换;4 种 auth(env/credential store/ambient/OAuth);
│       per-task 路由最优路径 = CLI flag 或 env var(rick 端决策)
└─ N4-subagent 扩展实现路径(Y6) ✅0.9(高)
   事实:pi 官方提供 subagent extension 范例(examples/extensions/subagent/ 35KB index.ts);
        实现机制 = spawn("pi", ["--mode", "json", ...]) 子进程,每 subagent 独立 pi 进程;
        subagent 是 LLM 可调用 tool(registerTool),支持 single/parallel(max 8)/chain 三模式;
        上下文隔离 = 独立进程 + 独立 system prompt(临时文件)+ 50KB 输出截断;
        rick subagent(prompt 注入同进程)迁移为 pi agents(.pi/agents/*.md)+ subagent tool 调用
```

## 节点详情

### N1-standalone binary 部署形态(Y1):pi release artifact 平台覆盖 + binary 体积 + Node 依赖
- 置信度:0.9(高)
- 信源验证:代码原文 ✅(build-binaries.sh + package.json build:binary) | 运行时 ✅(GitHub API v0.83.0 release 6 asset + 下载次数) | 文档 ✅(README + Bun 官方) | 反事实 N/A
- 调研报告:briefs/research-3-N1-standalone-binary.md
- 关键事实:
  - **6 平台预编译 release 全覆盖**:darwin-arm64/x64、linux-x64/arm64、windows-x64/arm64(v0.83.0 发布于 2026-07-29)
  - **binary 体积 30-44 MB**:darwin-arm64 30.3MB / linux-x64 41.6MB / windows-x64 43.8MB,与 claude code 同量级
  - **真零 Node.js 依赖**:`bun build --compile` 内嵌 Bun runtime(JS runtime,非 Node.js),目标机器无需装 Node.js / Bun / npm
  - **原生 binding 静态链接**:clipboard 6 平台 native 模块编译时静态链接进 binary
  - **可选自包含 model data**:`--offline-model-data` flag 内嵌 model catalog,真正零网络依赖
  - **下载次数证明可用性**:linux-x64 3620 次、windows-x64 2530 次、darwin-arm64 1295 次
  - **rick 自建 binary 可行**:`scripts/build-binaries.sh --platform darwin-arm64 --out ./out` 开源可自建

### N2-skill 系统级注册机制(Y3):registerTool/registerCommand 语义 + rick skill 映射
- 置信度:0.9(高)
- 信源验证:代码原文 ✅(extensions.md registerTool/registerCommand 全文 + rick skill 目录) | 运行时 ✅(extensions.md 1881 行 + pi Skills 体系) | 文档 ✅ | 反事实 N/A
- 调研报告:briefs/research-3-N2-skill系统级注册.md
- 关键事实:
  - **registerTool = LLM tool schema 系统级注册**:进 system prompt `Available tools` 段(一句话描述)+ `Guidelines` 段(使用时机)+ LLM function calling schema(OpenAI/Anthropic tools)。LLM 推理时可见、可主动调用。**这是"系统级注册"**。
  - **registerCommand = CLI 斜杠命令**:仅交互式 TUI 可用(`/stats`),LLM 推理时不可见、不可调用。**非系统级注册**。
  - **registerTool 动态生效**:运行时注册,无需重启 session,LLM 立即可见
  - **setActiveTools 控制 tool 启用**:可按任务阶段动态启停(doing 启用 research、learning 禁用)
  - **rick skill 是流程描述 markdown**:触发场景 + 预期效果 + 核心内容(分步骤 + bash 命令)。无函数签名,无参数 schema,无返回值
  - **pi tool 是 TypeScript 函数签名**:name + description + parameters(typebox schema)+ execute 函数
  - **语义不对齐**:rick skill(协议文档)≠ pi tool(可执行函数)。流程描述型 skill 不能直接映射为 pi tool,需重写为 TS extension + execute 函数
  - **pi 原生支持 Agent Skills standard**:`~/.pi/agent/skills/` + `.pi/skills/` 自动加载,`/skill:name` 调用(斜杠命令,非 LLM 自动触发)
  - **触发概率提升机制成立**:registerTool 把 skill 从"文件需主动读"提升为"schema 直接进 LLM 推理空间",但仅适用于可映射为 tool 的 skill(函数签名型)

### N3-provider 接入与任务路由(Y5):provider 列表 + 切换粒度 + per-task 路由
- 置信度:0.9(高)
- 信源验证:代码原文 ✅(pi-ai providers 目录 30+ 文件 + API 目录 11 协议 + extensions.md registerProvider + SDK setModel) | 运行时 ✅(Cross-Provider Handoffs 示例 + 30+ env var 表) | 文档 ✅ | 反事实 N/A
- 调研报告:briefs/research-3-N3-provider任务路由.md
- 关键事实:
  - **30+ 内置 provider**:OpenAI/Anthropic/Google/AWS Bedrock/Azure + OpenRouter/Together/Fireworks/HF/Groq/Cerebras + 国内(Kimi/MiniMax/Moonshot/Qwen/Xiaomi/ZAI)+ 本地(llama.cpp via registerProvider)
  - **11 种 API 协议**:anthropic-messages / openai-completions / openai-responses / google-generative-ai / google-vertex / azure-openai-responses / bedrock-converse-stream / mistral-conversations / cloudflare / openrouter-images / pi-messages
  - **per-prompt 模型切换原生支持**:Cross-Provider Handoffs,同一 context 内每次 `complete()` 可传不同 model,自动转换消息格式(thinking → text、tool calls 保留)
  - **AgentSession.setModel**:运行时动态切换模型(异步生效),返回 false 表示无 API key
  - **4 种 auth 方式**:环境变量(30+ provider 全支持)+ Credential Store(pi 管理)+ Ambient(AWS/gcloud/Azure CLI)+ OAuth(Anthropic/Google/Copilot/Vertex/自定义)
  - **registerProvider 动态注册**:支持代理(baseUrl 重写)+ 本地(llama.cpp refreshModels)+ 企业(OAuth SSO),apiKey 支持 env var 引用(`$VAR`)、命令引用(`!cmd`)、字面量
  - **PI_PROVIDER/PI_MODEL 环境变量注入 bash 子进程**:子进程内嵌 pi 调用可继承模型
  - **per-task 路由最优路径 = CLI flag 或 env var**:rick 端根据 task 类型(doing/dream/research)选 model 传给 `pi -p --model <model>`,最低改造量

### N4-subagent 扩展实现路径(Y6):pi Extension 实现 rick 风格 subagent
- 置信度:0.9(高)
- 信源验证:代码原文 ✅(subagent extension README + index.ts 35KB + registerTool + spawn 实现 + rick sense_loop.md) | 运行时 ✅(subagent 3 模式 + 安全模型 + 50KB 截断 + SDK 第 11 行) | 文档 ✅ | 反事实 N/A
- 调研报告:briefs/research-3-N4-subagent扩展.md
- 关键事实:
  - **pi 官方提供 subagent extension 范例**:`packages/coding-agent/examples/extensions/subagent/`(README + index.ts 35KB + agents/ + prompts/)
  - **实现机制 = spawn 子进程**:`spawn("pi", ["--mode", "json", "--append-system-prompt", tmpFile, "Task: ..."])`,每个 subagent 独立 pi 进程
  - **subagent 是 LLM 可调用 tool**:通过 `pi.registerTool({ name: "subagent", ... })` 注册,LLM 推理时主动调用,支持 3 种模式(single / parallel max 8 / chain with `{previous}`)
  - **subagent 上下文隔离**:独立 pi 进程(独立 context window)+ 独立 system prompt(临时文件)+ 独立 tool/model 配置 + 输出 50KB 截断
  - **subagent 安全模型**:project-local agent(`.pi/agents/*.md`)需用户确认(交互式 TUI),防 repo 注入恶意 prompt
  - **AgentSession 无内置 spawnSubagent**:subagent 必须通过 extension 自己 spawn 子进程实现(pi 不提供"虚拟 subagent")
  - **rick 现有 subagent 模式 = prompt 路径注入**:main agent 通过 `{{think_agent_path}}` 读取 subagent prompt 文件,角色切换执行,**非进程隔离,非 context 隔离**
  - **迁移路径**:rick subagent(think/research/exporter)映射为 pi agents(`.pi/agents/think.md` 等),main agent 调用 `subagent` tool 派发,每个 subagent 独立 pi 进程(强隔离)
  - **路径 B(pi 内部 subagent extension)优于路径 A(rick 端多次 spawn)**:trace 继承 + subagent 作为 LLM tool 可被 LLM 主动调用

## R7 上报项(无法达高置信度的叶节点)

无。本轮 4 个叶节点(N1/N2/N3/N4)置信度均为 0.9(高),无 R7 上报项。

**次要疑问(非阻塞,标注供后续参考)**:
- pi binary 在 Linux musl(alpine)上的兼容性未实测(Bun 编译产物通常依赖 glibc)— rick 部署环境为 macOS/Linux 主流发行版,非 alpine,非阻塞
- per-prompt 跨 provider 切换时 thinking → text 转换的语义损失未实测 — rick 主要 doing/dream 用同 provider,跨 provider 场景少,非阻塞
- subagent 输出 50KB 截断对 rick exporter RFC(可达 50KB)是否够用未实测 — 临界,需后续 doing 阶段验证,非阻塞

## 整合摘要

总节点数 5(含根) | 高置信度叶节点 4(N1/N2/N3/N4) | R7 上报 0

## Y1/Y3/Y5/Y6 四假设事实澄清结论

- **Y1(standalone binary 自包含)**:✅ 澄清。pi v0.83.0 release 提供 6 平台预编译 binary(darwin-arm64/x64, linux-x64/arm64, windows-x64/arm64),体积 30-44 MB,通过 `bun build --compile` 内嵌 Bun runtime,真零 Node.js 依赖。可选 `--offline-model-data` 内嵌 model catalog。分发形态等同 claude code(单 tar.gz/zip + SHA256SUMS + install.sh)。rick 亦可自建 binary(`scripts/build-binaries.sh` 开源)。下载次数证明真实可用(linux-x64 3620 次)。
- **Y3(skill 系统级注册 → 触发提升)**:✅ 澄清(部分)。pi `registerTool` 是 LLM tool schema 系统级注册(进 system prompt `Available tools` + `Guidelines` + function calling schema),LLM 推理时可见可调用——这是"系统级注册"。`registerCommand` 仅 CLI 斜杠命令,LLM 不可见。**但**:rick skill 是"流程描述 markdown"(触发场景 + 分步骤 + bash 命令),pi tool 是"TypeScript 函数签名"(name + parameters + execute),语义不对齐。流程描述型 skill 不能直接映射为 pi tool,需重写为 TS extension。仅含 .py 脚本的 skill(verify_go_changes 等)可包装为 pi tool(但需 rick 端写 TS shim 调用 .py)。**触发概率提升机制成立(registerTool 把 skill 从"文件需主动读"提升为"schema 进 LLM 推理空间"),但仅适用于可映射为 tool 的 skill**。
- **Y5(多 provider → 任务路由)**:✅ 澄清。pi 内置 30+ provider(OpenAI/Anthropic/Google/AWS/Azure + OpenRouter/Together/Fireworks/HF/Groq/Cerebras + 国内 Kimi/MiniMax/Moonshot/Qwen/Xiaomi/ZAI + 本地 llama.cpp via registerProvider),11 种 API 协议。**per-prompt 模型切换原生支持**(Cross-Provider Handoffs:同 context 内每次 `complete()` 可传不同 model,自动转换消息格式)。AgentSession.setModel 运行时动态切换。4 种 auth(env / credential store / ambient / OAuth)。**per-task 路由最优路径 = CLI flag 或 env var**(rick 端根据 task 类型选 model 传给 `pi -p --model <model>`),无需 rick 维护多个 pi 子进程。
- **Y6(subagent 默认扩展)**:✅ 澄清。pi 显式 No sub-agents(内置),但**官方提供 subagent extension 范例**(`examples/extensions/subagent/` 35KB index.ts)。实现机制 = `spawn("pi", ["--mode", "json", "--append-system-prompt", tmpFile, ...])` 子进程,每 subagent 独立 pi 进程(独立 context window + 独立 system prompt + 50KB 输出截断)。subagent 是 LLM 可调用 tool(registerTool),支持 single/parallel(max 8)/chain 三模式。rick 现有 subagent(think/research/exporter)是 prompt 路径注入同进程(非隔离),迁移为 pi agents(`.pi/agents/*.md`)+ subagent tool 调用后变为强进程隔离。**路径 B(pi 内部 subagent extension)优于路径 A(rick 端多次 spawn)**:trace 继承 + LLM 可主动调用 subagent。

## 给 human 的 S 阶段三连追问(基于新事实重写,聚焦 Y2 价值性假设的判断准备)

新事实摘要:pi 部署形态等同 claude code(6 平台预编译 binary 30-44 MB,零 Node 依赖,可自建);registerTool 是真"系统级注册"(进 LLM tool schema)但 rick skill 多为流程描述需重写为 TS extension;pi 30+ provider + per-prompt 切换原生支持(Cross-Provider Handoffs);pi 官方 subagent extension 范例存在(spawn 子进程,LLM 可调用 tool,3 模式)。

**前两轮已识别未澄清的 Y2(价值性假设)**:深度控制 → 更好效果(未确认瓶颈根因是"控制不足"而非"prompt 设计/任务分解")。本轮 Y1/Y3/Y5/Y6 四事实性假设全部澄清,现在需基于新事实准备 Y2 价值性判断。

1. **现状补充**:基于 Y3 澄清(registerTool 是真系统级注册,但 rick skill 多为流程描述需重写为 TS extension)这一事实,你当前 doing 阶段效果瓶颈的**具体表现**是什么?是(a) skill 触发率低(LLM 不主动读 skill.md)?(b) 长程任务 context 丢失(compaction 丢关键决策)?(c) task 完成率低(doing.md 单 task 承载过多)?(d) subagent 上下文污染(同进程角色切换串味)?——**请给具体案例 + 度量数据**(如:最近 N 个 job 中 skill 触发率 X%、长程任务失败率 Y%),以判断瓶颈根因是否真为"控制不足"。若根因是 prompt 设计/任务分解,则 pi 深度控制非解药。
2. **期望**:基于 Y1/Y3/Y5/Y6 全部澄清(pi 部署等同 claude code / registerTool 系统级但需重写 / per-prompt 模型切换原生 / subagent extension 范例存在)这一事实,你期望接入 pi 后**具体深度控制哪些行为**?请从以下新事实支持的扩展点中选 top-3:(a) `transformContext` 自定义 compaction(保留 rick act-path.md / debug/ 历史);(b) `registerTool` 把 rick skill 重写为 LLM tool schema(提升触发率);(c) `subagent` extension 把 think/research/exporter 拆为独立 pi 进程(强隔离);(d) `registerProvider` + per-prompt setModel(doing 用 Sonnet / dream 用 Haiku);(e) `beforeToolCall` hook 做 permission gate(替代 --dangerously-skip-permissions);(f) `pi.events` 事件系统做行为轨迹捕获(等价验收)。——这些期望中,哪些真需 pi 才能实现,哪些可通过 rick 端 prompt 模板/doing.md 设计达成?
3. **差距**:基于 Y3 澄清(rick skill 流程描述 ≠ pi tool 函数签名,需重写为 TS extension)这一事实,接入 pi 的**适配成本**包括:(a) 12 处 exec.Command 重构(8 低 + 4 中,前两轮已确认);(b) NDJSON 解析器重写(5 项字段不对齐 + duration_ms 自计时);(c) rick skill 重写为 TS extension(12 个 skill,流程描述型需重新设计为函数签名);(d) rick subagent prompt 重写为 pi agents `.pi/agents/*.md`(4 类:think/research/exporter/critic);(e) pi subagent extension 集成(安装 + agents/ 配置 + 工作流 prompts/)。**你是否接受这个适配成本?**其中(c) skill 重写为 TS extension 是最大工作量,你是否确认"行为轨迹捕获等价"的验收标准(指 NDJSON 字段对齐 + duration_ms 自计时 + tool_call 拦截时机 + turn 边界 + compaction 事件),还是要求语义级等价(指 subagent 上下文隔离强度 + skill 触发机制 + provider 切换语义)?
