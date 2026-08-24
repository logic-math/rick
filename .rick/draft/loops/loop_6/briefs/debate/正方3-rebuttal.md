# 正方3 反驳（Round 2）

> 立场（延续 Round 1）：工程能力 / 运行时成熟度视角——pi 以「单一显式触发入口 + 声明式 frontmatter agent + 已完成的迁移地基」在工程能力与运行时成熟度上优于仍处 developer preview、无兼容承诺的 deepseek-harness。本轮回驳对方三个立场的核心论点。

---

## 针对反方1「轻量/简洁/可控」的反驳

**对方核心论点**（反方1.md）：dsh「一切皆插件、类型化服务 API、声明式配置、MIT 全源码」，比 pi「模型手写 workflowScript 字符串 + 固定工具描述 + 提示词软约束」更可控；并声称 pi 的约束是「提示词祈使」，dsh 是「运行时强制」（反方1 反驳 2.3）。

### 反驳 1.1：反方1 混淆了「调用契约的形态」与「触发确定性」——dsh 对 rick 的核心痛点同样没有运行时强制

反方1 反驳 2.3 断言「dsh 在源码层强制触发权/权限边界，pi 是提示词祈使」。此结论对「权限边界」成立，但对「是否触发」不成立——而 rick 的核心痛点正是后者。

- **证据**：`deepseek-harness/packages/subagent/tool-subagent-report/README.md` 第 7 行原文——"The instruction is guidance, **not enforcement**: the mechanism still accepts zero or many calls in one turn, and **no runtime path rejects a child that never reports**." 同一文件的 Known Limitations：「Acceptance is weaker than durable delivery — there is no durable mailbox, idempotency key, delivery receipt, retry protocol, or exactly-once claim.」
- **含义**：dsh 的类型化 API（`ctx.subagents.start()` 等）强制的是「调用了子 agent 之后参数对不对、权限边界到哪」，而「模型这一轮到底要不要调用 subagent 工具」在 dsh 里**同样是模型的自由决策**。子 agent 的「回报」通道明确是 guidance 而非 enforcement。
- 因此反方1 的「dsh 更确定触发」不成立：**换到 dsh 并不会自动解决 rick 的「触发概率低」问题**——这恰是反方1 全篇未回应的核心矛盾。

### 反驳 1.2：「手写 workflowScript 字符串」是反方1 的稻草人——那是 rick 侧的固定模板，不是模型每次自由发挥

反方1 反驳 2.1 把 pi 的触发入口描述为「模型要正确拼 JS 才能触发，拼错即失败」的黑箱。此描述曲解了机制：

- pi 的触发 = 模型调用 **一次** `subagent` 工具，工具名与参数 schema 明确；`workflowScript` 里的 JS 是 **rick 提示词里预写的固定模板**（human 已在 S-R 定方向：把 think/research/exporter 注册为系统级 agent，提示词给出固定触发语法 `runs.run(agent:'think',...)`）。
- 这是**一次性配置成本**（写进提示词模板后恒定复用），不是「每次触发都要模型现编 JS」的动态成本。反方1 把「配置成本」偷换成「触发成本」。
- 证据：`research-report-S-bestpractice.md` BP-1 的官方示例 `subagent({ workflowScript: "return runs.run('main', {agent:'worker', task:'...'})" })`——JS 内容是**模板预写**，模型只需按提示词复述该调用。

### 反驳 1.3：「无特权核心、一切可替换」是「无稳定契约」的同义词，恰是成熟度短板

反方1 论点 1.1 把「agent-loop 本身可替换、无特权核心」当优点。从 rick 的立场（方法层不变、实现层要稳定）看，这是缺点：

- 「可替换」意味着 dsh **不给任何稳定契约承诺**：`AGENTS.md` 第 7 行——"Remove this section at the first tagged release. With no external consumers, prefer the correct foundation over compatibility shims… Backends reject old on-disk formats. SQLite uses monotonic `SCHEMA_VERSION`; `dsh-session` keeps `SESSION_FORMAT_VERSION` at 0 **with no compatibility promise**."
- 对 rick 而言，「运行时私有、有明确契约」的 pi 是**稳定黑箱**（loop_4 已支付迁移成本），而「全开源可替换」的 dsh 意味着 rick 需**自己锁定并维护一个实现**，把「用运行时」变成「养运行时」。

### 反驳 1.4：「MIT + strict TS + 显式约定」是代码质量优点，不是运行时成熟度判据

反方1 论点 1.4/1.5 用代码质量指标（MIT、strict:true、显式优于隐式）论证「可控」。这些指标是「代码可读性」，与「运行时成熟度」是两回事：

- 成熟度判据是**版本稳定性 + 兼容性承诺**：dsh 为 `0.1.0-rc.5`（pre-release，`package.json` 第 3 行）+ README「THERE WILL BE COMPATIBILITY-BREAKING CHANGES」+ `engines: node ^22.19.0 || >=24.0.0`（第 8-9 行，部署门槛激进）。
- 反方1 全篇**没有回应** preview / 破坏性变更 / 无磁盘格式兼容承诺这三个工程成熟度硬伤。

---

## 针对反方2「独立性/可迁移/不绑定」的反驳

**对方核心论点**（反方2.md）：dsh 以 MIT + 插件化 seam「不锁定单一生态」，pi 以目录/扩展/触发入口/生态「四重绑定」深度耦合。

### 反驳 2.1：「不绑定」在 dsh 当前阶段 = 「必须频繁跟着破坏性变更迁移」

反方2 把「绑定」本身当作缺点，这是价值判断错误。工程成熟度视角下，关键是**绑定的是不是稳定对象**：

- 反方2 自己在「保留」节承认：dsh「处于 developer preview…THERE WILL BE COMPATIBILITY-BREAKING CHANGES」「Backends reject old on-disk formats…no compatibility promise」。
- 即 dsh 的「可迁移/不绑定」代价是：rick 沉淀的方法层/会话产物若落其存储层，**跨版本无迁移保障**，必须跟着每次破坏性变更重写。相比之下，pi 的「绑定」是绑定一个**已支付迁移成本、契约明确、且 rick 已跑通 3 项 e2e** 的运行时（loop_4 RFC + job_30）。
- **绑定稳定对象（pi）是优点；绑定不稳定对象（dsh preview）才是缺点。**反方2 的论证把二者混为一谈。

### 反驳 2.2：反方2 的「llm-pi-ai 可复用 pi 模型层」是自伤的论据——它反证了 pi 生态的成熟

反方2 论点 2 称「dsh 可挂载 Earendil 的 pi-ai 作为 provider，复用 pi 的模型能力而不绑定 pi 运行时」。

- 这恰证明：**pi 的模型层（pi-ai）是通用、成熟、值得第三方 harness 复用的组件**。反方2 一边否定 pi 生态，一边又把 pi 生态的模型层当作 dsh 的优点来引用，逻辑自相矛盾。
- 而且「多 provider 并存」正是碎片化来源（见下条）。

### 反驳 2.3：「subagent 多后端并存」是碎片化，不是自由度——各后端行为不一致

反方2 论点 3 把「spawn/fork/acp/codex/claude-code/dsh-sdk 多 provider 并存」当「不绑定」的优点。但实读源码发现各后端**能力不一致**：

- 证据：`packages/subagent/subagent-acp/README.md`「rejects requests for persona, tool filtering, depth enforcement, or structured output instead of silently omitting them」；`subagent-claude-code/README.md`「output schemas, child personas, tool filtering, and harness depth enforcement are rejected by the shared service for this provider」；`subagent-codex/README.md` 同款「No optional shared capabilities」。
- 含义：同样的「触发子 agent」动作，走不同后端时**得到的能力集不同**（有的拒绝 persona、有的拒绝 depth 强制、有的拒绝 structured output）。这是「行为不一致」，对 rick 这种需要**确定性行为**的方法论载体是负担。pi 的单一 `workflowScript` 执行面反而保证**跨场景一致行为**（BP-1 单一入口 = 单一故障面，工程可精确控制）。

---

## 针对反方3「deepseek 原生契合/性能」的反驳

**对方核心论点**（反方3.md）：dsh 作为 DeepSeek AI 的一等公民 harness，对 DeepSeek 思考/工具调用/缓存的 first-party 原生适配优于 pi 的通用 provider 兼容层。

### 反驳 3.1：反方3 偷换了辩论维度——从「工程成熟度」换到「模型适配深度」

正方3 的立场维度是**工程能力 / 运行时成熟度**（preview、破坏性变更、兼容承诺、迁移成本）。反方3 全篇谈的是**模型适配深度 / 性能**（reasoning 回传省 token、cache 复用），**没有回应** dsh 的 preview 状态、破坏性变更、无磁盘格式兼容承诺这三个工程成熟度硬伤。

- 换句话说：反方3 证明了「dsh 在 DeepSeek 模型上适配更深」，但这**不构成**「dsh 工程成熟度更高」，二者是正交维度。

### 反驳 3.2：「first-party 血统」≠ 稳定，且「first-party 专属优化」= 厂商锁定

反方3 论点 1 用「DeepSeek AI 自研」论证契合度。但：

- first-party ≠ 稳定：dsh 自研却仍处 developer preview（README「THERE WILL BE COMPATIBILITY-BREAKING CHANGES」）。
- 且「first-party 专属优化」= **厂商锁定**。rick 的需求是「运行在 pi 上可用任意模型」（正方3 Round 1 反驳 6：pi 通用 provider 可换 deepseek/claude/gemini/gpt）。反方3 的「DeepSeek 专属优化」在「多模型可用」需求下反而是缺点——把 rick 的模型层锁死在 DeepSeek。

### 反驳 3.3：「reasoning 回传省 token / cache 复用」是性能维度，与 rick 的核心痛点（触发确定性）无关

反方3 论点 3/4 的优化（reasoning_content 回传省 token、cache 复用）是**性能/成本**维度。rick 本次要解决的核心问题是**触发确定性**（subagent 触发概率低）：

- 反方3 全篇没有证明 dsh 在「触发确定性」上优于 pi。事实上 dsh 的 model-facing subagent 触发同样是模型决策（tool schema 暴露给模型、模型决定是否调用），其「回报」通道明确是 "guidance, not enforcement"（`tool-subagent-report/README.md` 第 7 行）。
- 因此反方3 的「性能更优」不回应、也不改变「触发确定性」这个本次主题的结论。

### 反驳 3.4：反方3 承认了 pi 是「通用 provider」，这恰是 pi 的架构优点

反方3 反驳 1 说 pi 用「通用 provider 兼容层」承载 DeepSeek，把「通用」当缺点。但「通用 + 明确契约 + 可换模型」正是 rick 需要的：

- rick 期望「触发确定性提升到上限内最高」且「不随模型迭代失去先进性」（human N1）。通用 provider 层让 rick **不被单一模型厂商锁定**，模型迭代时可换。
- dsh 的「专用 deepseek-official 路由」是「窄而深」，pi 的「通用 provider」是「宽而稳」——对「方法层长期稳定」的目标，宽而稳优于窄而深。

---

## 本轮结论（一句话）

反方1/2/3 分别以「可控」「可迁移」「性能」三个维度立论，但均未回应 dsh 的 developer preview + 破坏性变更 + 无磁盘格式兼容承诺这一工程成熟度硬伤，且其「dsh 更确定触发」的共同隐含前提被 dsh 自己的 "guidance, not enforcement" 文档直接证伪，因此 pi 在「承载 rick 方法」的工程能力与运行时成熟度上仍胜出。
