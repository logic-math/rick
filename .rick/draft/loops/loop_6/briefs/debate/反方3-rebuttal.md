## 反方3 反驳（Round 2）

立场延续：在「用 DeepSeek 模型实现 rick 方法」这一具体场景下，deepseek-harness 的 first-party 原生契合/性能适配，优于 pi 的通用 provider 兼容层。以下逐条反驳正方三篇，证据均实读源码/文档核实。

---

### 针对正方1「subagent 编排/触发机制」的反驳

**反驳 1：正方1 的「单一触发面=高确定性」停在协议层，回避了「确定性上限由模型契合度决定」这一层。**
- 正方1 论点1/2 主张 pi 的「单一显式入口（subagent 工具 + workflowScript）+ fail loud（Unknown agent）」触发确定性高。但 pi 的触发确定性靠的是「提示词与 tool description 对齐」，最终仍是**模型自主决定是否调用工具**——这是通用机制，不针对 DeepSeek 模型的 tool-calling 遵循度做任何 first-party 调优。
- 而 rick **实际跑的就是 DeepSeek**：rick 运行产物 `rick/.pi/subagents/artifacts/*.jsonl` 中 `"provider":"deepseek","model":"deepseek-v4-pro"`。触发确定性的上限恰恰由「DeepSeek 模型 + harness 对 DeepSeek 官方协议（thinking_mode/tool_calls）的适配深度」决定。deepseek-harness 的 `llm-deepseek` 以官方 API 文档为真源（`packages/llm/llm-deepseek/README.md:5`「source of truth: the API docs — guides/thinking_mode, guides/tool_calls」），把 tool_calls 作为一等公民实现；pi 却把 DeepSeek 当 openai-completions 兼容层承载（`pi-ai/dist/providers/data/deepseek.json` `"api":"openai-completions"`）。正方1 的「协议对齐」论点未触及「模型协议契合」这一决定性层次。

**反驳 2：正方1 缺点2 把「组合（composition）」当缺点，实为表达力优势。**
- 正方1 引 `applyChildComposition` 称 dsh 子角色「靠父 preset 组合 + persona + toolFilter 叠加，角色边界被稀释」。但组合正是 dsh「一切皆插件（Cordis）」架构的优势：rick 方法层是「sense 复核层 + think/research/exporter 三角色 + 各阶段门禁」的**有层级、有组合**结构，dsh 的 `preset + persona + toolFilter` 能逐层表达「父级能力 + 子角色人格 + 工具边界」，比 pi「一个 frontmatter markdown 平铺」更能承载 human 逆转逻辑「把 rick 方法翻译进实现体系」的语义。
- 且 `applyChildComposition` 的「join 父级 preset」是**为防止缺陷**而设计：`packages/subagent/subagent/README.md:43`「a child that joined nothing would reach the model with an empty tool registry」——这是工程严谨性，不是正方1 说的「边界被稀释」。

**反驳 3：正方1 缺点3 的「参数多=出错面大」把可选能力当必填负担。**
- dsh `tool-subagent` 的 `outputSchema/maxDepth/toolFilter/persona` 是**可选能力**，不是必填。其中 `outputSchema`（结构化输出校验）恰恰能把 rick 三角色产出的验收从 pi 侧 BP-3 的「软性 contract（output 形状描述）」升级为「硬性 schema 校验」，是提升 think/research/exporter 产出确定性的加分项，正方1 未正面回应。

**反驳 4：正方1 缺点5 的「递归委派过度设计」是可关掉的。**
- `maxDepth` 默认 3 但 `0` 即禁止委派（正方1 自己引用「maxDepth 默认 3，0 禁止委派」）。rick 设 `maxDepth=0` 即可复现 pi 的「单层派发」约束。正方1 把「可选能力」当成「必付成本」，逻辑不成立。

---

### 针对正方2「扩展生态/长期演进」的反驳

**反驳 1：正方2 的「成熟度/稳定性」论据事实正确，但「稳定 ≠ 适合」。**
- 承认 dsh 确处 developer preview、`SESSION_FORMAT_VERSION=0` 无兼容承诺（`AGENTS.md:7`、`docs/persistence-catalog.md:10`）——这是事实。但限定边界：dsh 是 DeepSeek AI **官方** harness，其「迭代快、敢破坏兼容」的另一面是「紧跟 DeepSeek 模型协议演进」。rick 要「3 年后不失去先进性」（human N1 判断），先进性的关键来源是**与底层模型协议同步演进**；DeepSeek 思考模型协议（reasoning/tool_calls）在快速迭代，first-party harness 最先跟进——`llm-deepseek` 已实现 `reasoning_effort: off|high|max` 三档 wire 映射 + reasoning 回传省 token + cache 复用（`packages/llm/llm-deepseek/README.md:71-93`）。pi 的「稳定」是**通用层的稳定**，DeepSeek 协议变更时 pi 只能靠通用 provider 层被动适配，而非 first-party 主动演进。

**反驳 2：正方2 P4「迁移成本已支付」是沉没成本论证。**
- 「已经在 pi 上迁移了」只能说明「换 dsh 有切换成本」，不能构成「继续选 pi」的充分理由（沉没成本谬误）。且「3 项 e2e 测试通过」只证明「pi 能跑」，不证明「pi 是最优实现载体」。

**反驳 3：正方2 的 BP-1~BP-9 是通用 subagent 契约，维度错位。**
- BP-1~BP-9（触发语法/agent 名/async/单写者/编排权）是**通用编排契约**，完全不覆盖「DeepSeek 思考模型深度适配」维度（reasoning 回传省 token、cache 复用报告、thinking_mode 三档）。正方2 用「通用契约完备性」论证「rick 用 DeepSeek 场景的适合性」，恰好落在反方3 视角（DeepSeek 原生契合/性能）的证据覆盖范围之外——正方2 未触及反方3 的核心论点。

---

### 针对正方3「工程能力/运行时成熟度」的反驳

**反驳 1：正方3 反驳6 的「多模型需求」是未经 human 确认的偷换前提。**
- 正方3 称「dsh LLM 绑定 DeepSeek，与 rick 用任意模型的需求不一致」。但 human 全程未说「rick 必须用任意模型」；human 说的是「深度跟 pi 的 subagent 体系结合」「理解 pi 的触发语言」，且 rick **实际运行就是 DeepSeek**（`.pi/subagents/artifacts/*.jsonl` `"provider":"deepseek","model":"deepseek-v4-pro"`）。若 rick 实际即「用 DeepSeek 模型」，dsh 的 DeepSeek 绑定从「缺点」翻转为「优点」。正方3 用未证前提贬低 dsh，逻辑不成立。

**反驳 2：正方3 反驳4 的「软性/guidance」批评对 pi 同样成立（双重标准）。**
- 正方3 引 `tool-subagent-report/README.md:7`「guidance, not enforcement」称 dsh 触发同样软性。但 pi 的触发确定性**同样**依赖模型自主调用 subagent 工具——pi 侧 BP-9「模型认知来源 = tool description」也是 guidance，rick 侧调研从未证明 pi 有「硬性强制触发」。正方3 用「软性」打 dsh、却回避 pi 同病，是双重标准。

**反驳 3：正方3 反驳5 的「无 durable mailbox」把「诚实披露」当缺点。**
- dsh 自披露「Acceptance is weaker than durable delivery — no durable mailbox/idempotency key/exactly-once」（`tool-subagent-report/README.md:62`）是工程诚实。但 pi 的 subagent **同样没有** durable mailbox/exactly-once 投递保证（pi 靠 parent 编排 + 单写者 BP-6，无投递幂等）。正方3 把 dsh 自披露的局限当 pi 的优势，而 pi 同样缺失该能力。

**反驳 4：正方3 论点2 的「frontmatter 声明式简单」表达力受限。**
- 承认 frontmatter markdown 对「一个角色一个文件」的扁平分工简单。但限定边界：rick 方法层是「sense 复核层 + 三角色 + 阶段门禁」的**有层级、有组合**结构，dsh 的 `preset + persona + toolFilter` 组合式表达比 pi 平铺 frontmatter 更能承载这种层级语义。正方3 把「简单」当无条件优点，忽略了「简单 = 表达力受限」的另一面。

---

### 本轮结论（一句话）

正方三篇的「稳定性/成熟度/迁移地基」在通用 harness 层面成立，但系统性回避了「rick 实际用 DeepSeek 模型」这一事实——pi 以 openai-completions 兼容层承载 DeepSeek，deepseek-harness 以 first-party 官方协议（`deepseek-official` 路由 + reasoning 回传省 token + cache 复用）原生契合 DeepSeek，在「用 DeepSeek 实现 rick 方法」的触发确定性上限与性能维度上，反方3 的论点未被实质驳倒。
