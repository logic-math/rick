# 正方2 反驳（Round 2）

> 立场：pi 比 deepseek-harness 更适合作为「实现 rick 方法」的长期载体（扩展生态/长期演进视角）。
> 本轮针对反方 1/2/3 的核心论点逐一反驳，标注证据出处，诚实承认对方合理处并限定边界。

---

## 针对反方1「轻量/简洁/可控」的反驳

**对方核心论点**：dsh「一切皆插件 + 类型化服务 API + 声明式配置 + 全源码 MIT」把子 agent 委托收敛为可静态校验的显式契约；pi 的 workflowScript 手写 JS 字符串 + 固定工具描述 + 提示词软约束是「黑箱/过度封装」。

### 反驳 1.1：「简洁可控」以「未定型」为代价，无法支撑长期演进
反方1 的「轻量/简洁/可控」三个优点，全部建立在 dsh 尚处 developer preview、可自由重构的前提上。但「可控/简洁」是**当下代码形态**，「生态稳定」才是 rick 长期演进的必要条件。dsh 官方明示破坏性变更不可避免：
- 证据：`deepseek-harness/README.md:11`——"THERE WILL BE COMPATIBILITY-BREAKING CHANGES"。
- 证据：`deepseek-harness/AGENTS.md`——SESSION_FORMAT_VERSION=0 "no compatibility promise"，"Backends reject old on-disk formats"。
- 证据：`docs/persistence-catalog.md:10`——"the whole format is pinned at SESSION_FORMAT_VERSION = 0 — pre-release, no compatibility implied"。
反方1 大篇幅论证「源码可控」，但回避了：**一个今天可控、明天就破坏性重构的代码库，对需要 3 年稳定演进的 rick 方法层是负资产**。pi 的 v0.84.1 / pi-subagents v0.47.1 是已发布稳定版本（`.../node_modules/@earendil-works/pi-coding-agent/package.json`），这才是「长期演进」视角下的真实权重。

### 反驳 1.2：workflowScript 不是「黑箱」，而是把触发显式化；dsh 的 model-facing 触发同样依赖工具描述
反方1 反驳 2.1 称 pi 的 `workflowScript` 是「模型手写 JS 字符串的黑箱」。这是**层次混淆**：
- pi 的 `workflowScript` 是「唯一公开执行面」的**显式化**——它把「谁触发、怎么触发」从自然语言祈使（rick 现状 243 处软触发词）升格为显式语法。rick 触发概率低的根因是**提示词未对齐该语法**（N3.1：零 workflowScript），不是机制本身不可控。
- dsh 的模型触发同样要靠 model-facing 工具：反方1 自己引证 `dsh-tool-subagent`「registers on ctx.tools」、其 schema 加入 prompt assembly（`docs/architecture.md`「Add a model-facing capability」）。**即 dsh 的模型侧触发与 pi 的 tool description 是同一层次**——都依赖「工具 schema/描述」让模型认知。区别仅在于：pi 把编排脚本放在工具参数字符串里，dsh 把参数放在 schema 里。二者都不是「黑箱 vs 白箱」的对立，而是「动态脚本 vs 静态 schema」的工程取舍。
- 边界承认：`workflowScript` 字符串确无静态校验，这是 pi 的真实弱点（rick 现状 D1 的代价）。但这是「确定性 vs 静态类型」的取舍，不是「黑箱封装」——它可被 9 条文档化最佳实践（BP-1~BP-9）规范化，而 rick 尚未对齐（D1~D7）才是问题所在。

### 反驳 1.3：「9 条最佳实践 = 复杂度高」恰是生态成熟的标志，而非缺点
反方1 反驳 2.4 把「需 9 条最佳实践」说成 pi 的复杂度劣势。反驳：
- 「有 9 条文档化最佳实践」本身是**生态成熟、可学习、可复现**的标志；dsh 的「简洁」很大程度上是因为**它还太新、尚未沉淀出最佳实践**（developer preview）。
- dsh 的「类型化服务 API」（`ctx.subagents.start()`）是**开发者 API**，pi 的 `workflowScript` 是 **model-facing 触发**——两者层次不同，不能直接比「谁更简单」。rick 的目标是「让模型确定地触发子 agent」，这个问题的 model-facing 复杂度在 dsh 里同样存在（tool schema 仍需对齐）。
- 反方1 说 dsh「注册一个 provider + 挂一个 tool consumer」即可，但忽略了 dsh 要「注册 provider + 定义 schema + 配置 profile/bundle/patch + 处理 subagent 多后端（spawn/fork/acp/codex/claude-code）的语义差异」——这些概念面并不比 pi 少，只是散落在 `packages/subagent/*` 十几个包里。

---

## 针对反方2「独立性/可迁移/不绑定」的反驳

**对方核心论点**：dsh MIT + 插件化 seam（LLM/subagent/composition 均可替换，可复用 pi-ai 而不绑定）提供「实现可迁移」；pi 以目录/扩展/触发入口/生态四重绑定深度耦合 rick。

### 反驳 2.1：「可迁移」是理论上的，实际代价是「自己维护一个 developer preview fork」
反方2 把「MIT + 一切皆插件」解读为「可迁移、不锁定」。但可迁移的前提是**上游稳定、fork 成本低**。dsh 的反例恰在其自己的「保留」段已诚实标注：developer preview + SESSION_FORMAT_VERSION=0 无兼容承诺。含义是：**rick 若迁到 dsh，实际上是要 fork 一个存储格式无兼容承诺、API 会破坏性变更的项目，自己维护分支**——这把「绑定 pi 生态」换成了「绑定 dsh 源码 + 自维护 fork」，迁移/维护成本并不更低。
- 证据：反方2 自身「保留」段引用 `README.md`「THERE WILL BE COMPATIBILITY-BREAKING CHANGES」、`AGENTS.md`「no compatibility promise」。

### 反驳 2.2：「复用 pi-ai 而不绑定」忽略了 dsh 同样依赖 Earendil npm 生态
反方2 用 `llm-pi-ai` provider（`packages/llm/llm-pi-ai/README.md`「backed by `@earendil-works/pi-ai`」）论证 dsh「不绑定 pi 运行时」。但这条证据反而说明：**dsh 的模型层同样依赖 Earendil 的 npm 包（pi-ai）**。反方2 反驳 pi 的「npm 扩展绑定」（pi-subagents），却对 dsh 的「llm-pi-ai 依赖 pi-ai」视而不见——两者是同构的依赖关系。所谓「不绑定」是选择性的：dsh 用 pi-ai 做模型层、pi 用 pi-subagents 做子 agent 层，**二者都站在 Earendil 的包生态上**。rick 迁到 dsh，只是把「依赖 pi-subagents」换成「依赖 llm-pi-ai + dsh 自身」，并未减少对第三方生态的依赖。

### 反驳 2.3：「四重绑定」多为 rick 主动选择或确定性设计，非「锁定」
- 「目录绑定」（PI_CODING_AGENT_DIR 隔离）：是 rick 主动用隔离环境管理自身配置（`tools_init_pi.go` 注入 `piagent.AgentEnv()`），pi 本身支持 user 级（`~/.pi/agent/agents/`）与 project 级（`.pi/agents/`）多作用域。隔离 ≠ 锁定。
- 「触发入口绑定」（workflowScript 唯一执行面）：是**确定性设计**——唯一执行面让触发语义可预期、可对齐（BP-1）。反方2 赞 dsh 的「subagent 多 provider 后端」（spawn/fork/acp/codex/claude-code），但**多后端 = 触发语义不统一**，对 rick「提升触发确定性」的核心目标反而是负担：同一「think」角色在 dsh 里走进程内还是走 Claude Code 子 agent，语义和确定性都不同。
- 边界承认：pi 的「生态绑定」是真实的长期风险，human N1 已言「3 年后失去先进性」。但化解该风险的正确路径是 human 已确认的 S-R 逆转逻辑「rick=方法、pi=实现，把方法翻译进 pi」，而非换一个 developer preview 的 dsh。

---

## 针对反方3「deepseek 原生契合/性能」的反驳

**对方核心论点**：dsh 是 DeepSeek first-party harness，专用 `deepseek-official` 路由 + reasoning 回传省 token + cache 复用，优于 pi 的通用 provider 兼容层。

### 反驳 3.1：这是「模型层」优势，不是「承载 rick 方法层」的优势
反方3 的全部论点（deepseek-official 路由、thinking_mode/tool_calls 一等公民、reasoning 省 token、cache 复用）都落在**模型 wire protocol 适配**上。但 rick 方法层的载体是 **subagent 编排/触发机制**（think/research/exporter 的角色分工、SENSE 五阶段推进），不是模型的 thinking_mode 序列化。反方3 对「实现 rick 方法」这个核心问题的贡献有限——它论证的是「用 DeepSeek 模型时谁省 token」，不是「谁更能确定性承载 rick 的 subagent 编排」。
- 边界承认：若 rick 长期只用 DeepSeek 模型，反方3 的「first-party 原生适配」是 dsh 的真实优势（限定在模型层）。但这不构成「换实现体系」的理由，因为模型层优势可以通过其他方式获得（pi 的 provider 层本身也在迭代），而方法层承载才是 rick 的核心矛盾。

### 反驳 3.2：first-party 契合 = 另一种「绑定」（模型绑定）
反方2 指责 pi 绑定「生态」，反方3 却在论证 dsh 绑定「DeepSeek 模型」。`deepseek-official` 路由「deliberately distinct from pi-ai's catalog name `deepseek`」（`packages/llm/llm-deepseek/README.md`）恰恰说明 dsh 的深度优化是**专用**的：换模型就要新增路由。而 pi 的通用 provider 层（反方3 引证 `deepseek.json` `"api":"openai-completions"`）一条 openai-completions 路由服务数十个模型，**换模型零改动 rick 方法层**。对 human「3 年后随模型迭代不失去先进性」的关切，通用 provider 层的「模型可替换性」比 first-party 专用路由的「单模型深度优化」更契合。
- 证据：反方3 引证 `pi-ai/dist/providers/deepseek.models.js`「auto-generated」+ `deepseek.json` `openai-completions`——这正说明 pi 的 DeepSeek 承载是「可自动生成的通用适配」，模型目录随 catalog 迭代自动更新。

### 反驳 3.3：性能/缓存优化与 rick 核心矛盾正交
rick 的当前痛点是 **subagent 触发概率低（协议不对齐，K1/K2/K4）**，不是 token 浪费或推理缓存。反方3 的「reasoning 回传省 token / cache 复用」是**工程性能边际**，与「触发确定性」正交。即便 dsh 在 DeepSeek 上省 10% token，若其 subagent 触发语义（多 provider 后端）不统一，rick 的触发确定性目标仍无解。反方3 未回应 rick 的核心矛盾，属于「用性能维度绕开承载维度」。

---

## 本轮结论（一句话）

反方三视角分别用「简洁可控（代码形态）」「可迁移（理论 fork 能力）」「模型原生契合（模型层性能）」论证 dsh 优势，但三者均未正面回应 rick 的核心矛盾——**「触发确定性的长期稳定承载」**——而 dsh 的 developer preview + 无兼容承诺（SESSION_FORMAT_VERSION=0）恰在这一维度是负资产，pi 的成熟版本 + 文档化触发契约（BP-1~BP-9）+ rick 现成迁移基础仍是更优的长期载体。

---

## Acceptance Report

- changed-files: `/workdir/sunquan20/AI_CODING/rick/.rick/draft/loops/loop_6/briefs/debate/正方2-rebuttal.md`
- commands-run: 实读 debate/正方2.md（己方初始观点）、反方1.md、反方2.md、反方3.md（对方观点）；引用 deepseek-harness 源码（README.md/AGENTS.md/docs/persistence-catalog.md/docs/architecture.md/packages/llm/llm-deepseek/README.md/packages/llm/llm-pi-ai/README.md）与 pi/pi-subagents 版本事实。
- residual-risks: 未重新跑 dsh 的 pnpm install/test（反驳轮无需）；pi-ai 的 deepseek.models.js/data 引用自反方3 自身引证（未重复全文核验该 dist 文件）。
- no-staged-files: true
