# 正方1 反驳（Round 2）

> 立场不变：对「实现 rick 方法」而言，pi 的单一显式触发面 + 简单 agent 定义 + 编排权在 parent，优于 deepseek-harness 的通用低层 seam。
> 反驳对象：反方1（轻量/简洁/可控）、反方2（独立性/可迁移/不绑定）、反方3（deepseek 原生契合/性能）。

---

## 针对反方1「轻量/简洁/可控」的反驳

### R1.1 「一切皆插件=更轻量可控」是概念成本的误读，不是更简单
反方1 论点 1.1 把「Every part of the product is a plugin … no privileged core」当优势。但这句恰恰说明：要承载 rick 方法，rick 必须理解并维护 **Cordis 的 spatiotemporal composability 编程范式**（README 引用的《A Programming Paradigm for Spatiotemporal Composability》）、plugin tree、profile/bundle/`cordis.patch.yml` 分层、以及 llm/tools/subagent 多个 seam。证据：`deepseek-harness/README.md` 首段（"powered by Cordis"）+ 反方1 自引 `docs/architecture.md`「Profiles and bundles」。
- 反方1 说 pi 需要 9 条最佳实践（BP-1~BP-9）是「复杂度」，但那是**操作清单**；dsh 要的是**一门新编程范式 + 插件树心智模型**。「无特权核心」对 rick 不是福音而是负担——rick 要的是一个**稳定的触发协议**，不是一个连 agent-loop 都能被替换、需要自己兜底维护的高自由度框架。

### R1.2 反驳「workflowScript = 模型手写 JS 黑箱」——这是对 pi 触发入口的失实描述
反方1 反驳 2.1 称 pi 要模型「手写 JS 字符串、拼错即失败」。实际 pi 的触发是**固定单行模板** `subagent({workflowScript:"return runs.run('main',{agent:'worker',task:'...'})"})`，模型只需按模板填入 **agent 名 + task** 两个字段，不是自由编写编排逻辑。证据：`research-report-S-bestpractice.md` BP-1 官方示例。
- 反向对照 dsh：模型侧 `tool-subagent` 要求模型处理 `provider` / `run_in_background` / `backgroundMode`(one-shot|continuable) / `maxDepth` / `outputSchema` / `toolFilter` / `persona` 等**更多参数**（正方1 缺点 3 已证，`packages/subagent/tool-subagent/README.md` Config 表）。真正「参数更多、出错面更大」的是 dsh，不是 pi。反方1 若以「schema 静态校验」为 dsh 辩护，那 pi 的「Unknown agent 拒绝」同样是运行时 fail-loud（BP-2），且 pi 的失败面（1 个模板 + 1 个 agent 名）远小于 dsh（1 个 provider 实例 + 1 个 toolName + 多参数）。

### R1.3 「固定工具描述=黑箱」与「约束都是软性」混淆了两类事实，需限定边界
- 反方1 反驳 2.2 说 pi 模型认知来源锁死在 full/compact/custom 三档。但 pi 的**每个自定义 agent 的 systemPrompt 完全由开发者用 frontmatter markdown 控制**（`docs/agents.md` frontmatter reference，`systemPromptMode`/`inheritSkills` 可配），并非「只能选档位不能改」。
- 反方1 反驳 2.3 说 pi 的约束「全是提示词祈使、非源码强制」。这里要纠正一半：**编排权在 parent（BP-8）的底层是运行时工具注入强制**——普通子 agent **不接收 subagent 工具、不接收 pi-subagents skill**（`research-report-S-reasons-agent.md` BP-8），这是工具注入层的不变式，不是写给模型看的软规则；「Unknown agent」拒绝也是运行时强制。「单写者」（BP-6）确实偏约定，我承认这一条是软性。
- 而 dsh 的「运行时强制」（`assertSubagentMaxDepth`、duplicate names fail loud）虽硬，代价是它整体处于 developer preview：`README.md:11`「**THERE WILL BE COMPATIBILITY-BREAKING CHANGES**」、`AGENTS.md:7`「Backends reject old on-disk formats … no compatibility promise」。一个连 on-disk 格式都不承诺兼容的运行时，其「fail loud」的一部分恰恰来自格式不稳定，而非成熟可控。

### R1.4 「源码 MIT 全开放=可控」对 rick 是维护负担而非资产
反方1 论点 1.4 把「MIT + strict TS + 全源码」当可控性。但 rick 的 S-R 目标是把方法「翻译」进实现体系（human 判断），不是 fork 一套 harness 改 TypeScript 源码。dsh 的「可控」要求 rick 读/改/维护实现源码；pi 的「可控」是**声明式 markdown 落盘注册 agent**（P1/P2 已证）。对 rick 这种「方法层 + 少量配置」的诉求，声明式落盘远优于「fork 源码」。MIT 许可不等于 rick 应该 fork；dsh 当前 `SESSION_FORMAT_VERSION=0`、无兼容承诺（`AGENTS.md:7`），fork 后要持续跟随破坏性迭代，这正是反方2 自己「保留」节诚实标注的风险。

---

## 针对反方2「独立性/可迁移/不绑定」的反驳

### R2.1 「MIT + 无特权核心=可迁移」是纸面优势，实际被 developer preview 抵消
反方2 论点 1 把 MIT + 无特权核心当「可迁移/不绑定」。但可迁移的前提是**稳定**。反方2 自己「保留」节已诚实标注：dsh 处于 developer preview（`README.md:11` 兼容性破坏）+「Backends reject old on-disk formats … no compatibility promise」（`AGENTS.md:7`）。一个连 `SESSION_FORMAT_VERSION` 都停在 `0` 且明示不兼容的运行时，「可迁移」的成本是**迁移后持续跟随它的破坏性迭代**；而 pi 已经是 rick **已完成迁移的稳定态**（loop_4 RFC + job_30 落地）。「纸面可迁移」与「实际稳定可用」之间，rick 的诉求是后者。

### R2.2 「llm-pi-ai 可复用 pi 模型层」这一论据自我消解，反而证明 pi 生态是共享基础设施
反方2 论点 2 用 `packages/llm/llm-pi-ai` 证明「pi 模型层可被 dsh 挂载，不必绑定 pi 运行时」。这条证据恰恰说明：**pi 的模型层是开放的共享 seam，其生态是基础设施**。但由此推出「用 dsh 更好」是逻辑跳跃——若 pi 模型层本就开放可复用，rick 直接留在 pi 就同时获得模型层 + 已迁移的 subagent 扩展 + 稳定态；迁到 dsh 则要重新搭建 subagent 后端（6 个 provider 里选）并承担 developer preview 风险。反方2 的论据支持「模型层解耦」，不支持「运行时应换成 dsh」。

### R2.3 「目录/扩展绑定=强耦合」需限定：这是一次性、已完成的适配，且与 dsh 的声明式是同类机制
反方2 反驳 1/2 说 pi 的 agent 发现目录（`~/.rick/pi/agent/agents`）与 `requiredExtensions=["pi-subagents"]` 是「绑定」。但：
- 该适配是**一次性的、已落地**的成本（`tools_init_pi.go` + `PI_CODING_AGENT_DIR` 隔离，job_30 已迁移），不是每轮付出的持续成本。
- pi 的 agent 注册本质就是「声明式 markdown 落盘到发现目录」，与 dsh 的「声明式 cordis.yml/patch」是**同一类机制**（声明式配置 + 可 diff 可审计）。反方2 一边夸 dsh 的声明式 patch、一边贬 pi 的声明式落盘，是双重标准。
- dsh 的「多 subagent 后端并存」（`packages/subagent/` 下 spawn/fork/acp/codex/claude-code/dsh-sdk 6 个 provider）对 rick 是**选择过载**：rick 只需「单一确定触发」，多后端并存增加决策面与失控面（正方1 缺点 1），不构成「更自由」的优势。

### R2.4 「唯一触发入口=锁定」是因果倒置：唯一入口正是确定性优势
反方2 反驳 3 把 pi 的「唯一 workflowScript 执行面 + 固定内置名」当锁定。但**唯一入口恰恰是触发确定性的来源**（正方1 论点 1：模型无需在多 provider/多 toolName 间选择）。「把 rick 自造名翻译成 pi agent 名」是 human 在 S-R 已明确的「rick=方法、pi=实现」的**必然落地动作**，不是「锁定」，而是「方法落到唯一稳定实现上」。反方2 用「多后端可替换」换来的自由，正是 rick 明确不要的（human N2：完全偏向深度改造、不考虑独立性）。

---

## 针对反方3「deepseek 原生契合/性能」的反驳

### R3.1 「first-party 血统=更契合」是品牌叙事，不是可靠论据；且 dsh 自身也在快速破坏性迭代
反方3 论点 1 用「developed by DeepSeek AI」当契合度论据。但 first-party 不等于稳定：同一 README 明示 developer preview + `THERE WILL BE COMPATIBILITY-BREAKING CHANGES`（`README.md:11`）。DeepSeek 自己都声明未来破坏兼容，「模型迭代最先跟进」的反面是「API/格式频繁变更、跟随成本高」。对 rick 的「3 年后不失去先进性」（human N1）诉求，一个明示破坏性迭代的 harness 是显性风险，而非保障。

### R3.2 「pi 是通用兼容层、无 deepseek 优化」的论据不充分，且把「性能优化」与「触发确定性」混为一谈
反方3 反驳 1 的证据只有两条：pi 的 `deepseek.models.js` 是 auto-generated、`deepseek.json` 里 `api=openai-completions`。这两条只能说明「pi 的 deepseek 模型目录是生成式元数据 + 走 OpenAI 兼容协议」，**不能证明 pi 对 DeepSeek 的 reasoning/tool-calls 无适配**。更重要的是：本次 loop 的主题是**subagent 触发确定性**（D1/D2/D7 已证：提示词未对齐 pi 触发机制），不是「DeepSeek 思考模式的 token 节省」。反方3 把「cache 复用 / reasoning 回传省 token」这类**推理性能优化**，偷换为「实现 rick 方法的载体优劣」，是论题偏移。rick 要的是「确定性触发子 agent」，不是「deepseek 推理性能最优的 harness」。

### R3.3 「rick 用通用层跑 deepseek = pi 劣势」是因果错位
反方3 反驳 3 用 rick 运行产物里 `"provider":"deepseek","model":"deepseek-v4-pro"` 证明 pi 路径上 deepseek 只是「一个 provider」。但该证据只能说明 rick 用 DeepSeek 模型，**不能说明触发概率低与 DeepSeek 适配有关**。触发概率低的根因是 rick 提示词 243 处自然语言触发词、0 处 pi 触发语法/agent 名（N3.1/N3.2 已证），与「deepseek 在 pi 上的适配深度」无关。反方3 把两个独立变量强行挂钩，属因果错位。

### R3.4 「deepseek-official 专用路由」恰恰反证 pi 已覆盖 deepseek，dsh 需额外差异化
反方3 论点 2 引用 `llm-deepseek` 的 `deepseek-official` 路由「deliberately distinct from pi-ai's catalog name `deepseek`」。这反向说明：**pi-ai 的 catalog 里已经存在 `deepseek` provider**，dsh 为了差异化才额外维护一条 `deepseek-official` 路由。即「deepseek 模型接入」在 pi 侧是既有能力，dsh 的「专用适配」是增量优化而非从无到有。该增量（reasoning 回传省 token、cache 复用）是**性能层**收益，不改变「pi 的确定性触发 + 稳定态」在承载 rick 方法上的根本优势。

---

## 本轮结论（一句话）

反方把「通用性 / 多后端 / 源码可改 / deepseek 推理性能优化」包装成优势，但对 rick「单一确定触发 + 稳定长期载体 + 已完成迁移」的诉求而言，这些要么是过度能力（多 provider/多参数）、要么是纸面优势（MIT 可迁移 vs developer preview 无兼容承诺）、要么是论题偏移（性能优化 ≠ 触发确定性）；pi 的确定性触发 + 运行时强制的编排权 + 方法层可声明式注册为 agent，才是 rick 方法翻译落地的正解。
