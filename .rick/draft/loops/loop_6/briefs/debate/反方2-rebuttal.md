# 反方2 反驳（Round 2）

> 立场（延续 Round 1）：deepseek-harness 以 MIT + 插件化 seam（LLM/subagent/组合均可替换）为 rick 方法提供「实现可迁移、不锁定单一生态」的载体。本轮针对正方 1/2/3 的论点逐条反驳，证据均实读自 `deepseek-harness` 源码与 rick 侧已调研事实。

## 针对正方1「subagent 编排/触发机制」的反驳

### 反驳 1-1：正方1 把「入口唯一」偷换为「模型会触发」，而 rick 现状恰好反证唯一入口≠触发确定性
- 正方1 论点1 主张「pi 触发入口唯一且显式 → rick 三角色可 1:1 确定性触发」。但 rick 侧已证实事实（`research-report-S.md` N3.1/N3.2）是：rick 提示词 243 处自然语言触发词、0 处 `workflowScript`/`runs.run`、0 处内置 agent 名，导致触发概率低。**入口唯一是 pi 的设计事实，但 rick 若未对齐该入口，模型照样不触发**——唯一入口并不自动带来确定性，确定性来自「提示词是否与该入口对齐」，而这与运行时选谁无关。
- 因此正方1 用「入口唯一」论证「pi 更利于触发确定性」是不充分的：dsh 同样有显式模型侧工具 `tool-subagent`（`packages/subagent/tool-subagent/`），触发入口同样是「模型显式调用工具」，二者在同构层面，只是 dsh 的入口按 provider 实例化。

### 反驳 1-2：正方1 的「碎片化缺点」其实是 seam 的能力面，可在配置层收敛为单入口
- 正方1 缺点1/缺点3 称 dsh「触发面分散、参数多、出错面大」。但 dsh 的多 provider 是「一个 seam，多个实现并存」的能力暴露面，rick 可在配置层**固定一个 provider（如 spawn-in-process）**并把触发封装成与 pi 等价的单一调用，参数多寡是「默认暴露」而非「必须承受」。
- 反观 pi 的「唯一入口」本质是「只有一个厂商实现」——这是绑定，不是优势。dsh 的 seam 让 rick 方法层**自己定义子 agent 后端**（`packages/subagent/README.md`：「Multiple named providers may coexist in one context」），可迁移性正是 rick 方法层需要的（human S-R：rick=方法、pi=实现，方法应可与实现解耦）。

### 反驳 1-3：正方1 缺点5 的「递归委派是过度设计」站不住——dsh 可配置 maxDepth=0 禁止委派
- 正方1 缺点5 称 dsh「递归委派 + continuable 多轮对 rick 单层派发是过度设计」。但 dsh 的 `maxDepth` 默认 3，**`0` 即禁止委派**（`packages/subagent/tool-subagent/README.md:28`：「Absolute delegation-depth cap, default `3` (`0` forbids delegation)」）。
- 即「编排权集中在 parent」在 dsh 上是**一个配置项**（maxDepth=0），与 pi 的硬规则（BP-8）等价可达；正方1 把「dsh 默认开放递归」当作「dsh 无法约束递归」，是逻辑滑坡。能力更多≠必须使用，反方2 立场下的 rick 完全可只开单层。

### 承认并限定：正方1 缺点4（preview 破坏性变更）属实，但属「当前时点成熟度」而非「结构劣势」
- dsh 的 developer preview 与「THERE WILL BE COMPATIBILITY-BREAKING CHANGES」是事实（`README.md:11`），反方2 Round 1 已诚实保留。此点削弱的是「今天即可稳定上线」，不影响「方法层长期可迁移、不锁定」的结构性判断；需把「成熟度」与「可迁移性」分开评估。

## 针对正方2「扩展生态/长期演进」的反驳

### 反驳 2-1：正方2 用「厂商生态内的稳定」偷换「长期先进性」——绑定单一生态恰是 human 所判最大风险
- 正方2 P1/P2/P3 用 pi 的版本号（0.84/0.47）、九条最佳实践、四种外化机制论证「长期演进更优」。但这些「稳定」全部是 **pi 生态内部约定**：rick 方法层一旦用 pi 的 agent/skills/agentOverrides/refinement 承载，就与 pi 的目录（`~/.pi/agent/agents/**`）、扩展（`requiredExtensions=["pi-subagents"]`）、触发入口（`workflowScript`）深度耦合（反方2 Round 1 观点1/2 已证）。
- human N1 已判「3 年后失去先进性是最大风险」。绑定单一厂商的「生态内稳定」，若该厂商演进方向与 rick 方法层冲突，反而成为先进性枷锁。dsh 的 profile/bundle/patch（`docs/architecture.md`「Profiles and bundles」节）把方法层打包为**可分发、可 patch 覆盖**的单元，跨运行时可迁移性更强。

### 反驳 2-2：正方2 的「迁移成本已支付」是沉没成本论证，且高估了换 dsh 的成本
- 正方2 P4 称「rick 已迁 pi、迁移成本已支付，换 dsh 推倒重来」。但 human S-R 已判「rick=方法、pi=实现」——方法本应与实现解耦。若 pi 的绑定与「方法/实现解耦」目标相悖，已支付成本不应阻碍为方法层寻找更可迁移载体。
- 且 dsh 可挂载 `llm-pi-ai` provider（`packages/llm/llm-pi-ai/README.md`：「Generic multi-provider adapter ... backed by `@earendil-works/pi-ai`」），即**迁到 dsh 不必重写模型层**——pi 的模型能力可被 dsh 复用，迁移成本被正方2 高估。

### 反驳 2-3：正方2 D4「vendored Cordis = 生态依赖深」结论反了
- 正方2 D4 称 dsh「vendored Cordis、生态依赖深、迁移成本高」。但 vendored 恰恰是**把框架源码内嵌、自维护分支**，第三方集成 dsh 时无需额外处理 Cordis 依赖，是「自包含」而非「生态依赖深」。
- 真正的「能力不在自己手里」是 pi 侧：rick 的 subagent 能力依赖 Earendil 的 npm 扩展（`tools_init_pi.go` 硬编码 `ensureNpmExtension("pi-subagents", ...)`），扩展失效/演进即 rick 能力受制（反方2 Round 1 观点2）。

### 承认并限定：正方2 D1/D2/D3（preview + SESSION_FORMAT_VERSION=0 无兼容承诺）属实
- dsh 的会话格式 `SESSION_FORMAT_VERSION=0`、无兼容承诺（`AGENTS.md:7`、`docs/persistence-catalog.md:10`）是事实。这确实是 dsh 作为「长期稳定载体」的当前硬伤，反方2 Round 1 已保留。边界：它约束的是「把 rick 产物沉淀进 dsh 存储层」的稳定性，不否定「方法层作为插件可迁移」的结构性优势。

## 针对正方3「工程能力/运行时成熟度」的反驳

### 反驳 3-1（事实纠错）：正方3 反驳6「dsh LLM 绑定 DeepSeek」是错误结论
- 正方3 反驳6 称「dsh LLM 能力绑定 DeepSeek 提供方、厂商锁定」。但实读 `packages/llm/` 目录，除 `llm-deepseek` 外还有 **`llm-pi-ai`** 包，其 README 明写「Generic multi-provider adapter ... backed by `@earendil-works/pi-ai`」，可挂 OpenAI-compatible gateway、自建服务、任意新 provider（「configuration rather than a code change」）。
- 正方3 仅依据 `AGENTS.md` 目录注释「llm/ ... DeepSeek providers」就下结论，遗漏了 `llm-pi-ai` 包——**dsh 非但不绑定 DeepSeek，反而能复用 pi 的模型层**。这一事实错误削弱了正方3 的「厂商锁定」反驳。

### 反驳 3-2：正方3 反驳4 混淆「回报通道」与「触发入口」
- 正方3 反驳4 引 `tool-subagent-report/README.md`「The instruction is guidance, not enforcement ... no runtime path rejects a child that never reports」，断言 dsh「并未解决触发确定性」。
- 但该段针对的是子 agent 的**回报（report）收尾环节**，而非**触发（start）环节**。dsh 的触发入口是模型侧显式 `tool-subagent` 工具，与 pi 的 `subagent` 工具同构。pi 的「fail loud（Unknown agent 拒绝）」只保证「名字错会报错」，同样不保证「模型会主动触发」——正方3 把「回报软约束」等同于「触发软约束」，且默认 pi 在触发确定性上更强，未给出对照证据。

### 反驳 3-3：正方3 把「单点入口」当优点，恰是 rick 方法层要避免的结构性风险
- 正方3 论点1 称 pi「单一入口 = 单一故障面，工程可控」。但「单一入口」的代价是**单一厂商实现、不可替换子 agent 后端**。dsh 的 subagent seam 可挂 `claude-code`/`codex`/`acp` 等多种后端（`packages/subagent/` 目录实列），rick 的子 agent 后端不被锁死。
- 对 rick 方法层的长期可迁移性（反方2 立场核心），「单点」是结构性风险，不是优点；正方3 把「可控性」与「可迁移性」对立，默认选择了前者而回避了后者。

### 反驳 3-4：正方3 论点3 的「迁移已支付」同 2-2，属沉没成本论证
- 同反驳 2-2：human S-R 已判「rick=方法、pi=实现」，方法应可与实现解耦；且 dsh 可复用 `pi-ai` 模型层，换 dsh 的迁移成本被正方3 高估。

### 承认并限定：正方3 反驳5 的「投递缺陷」属实，且是 dsh 诚实标注的已知限制
- dsh `tool-subagent-report` 的 Known Limitations「Acceptance is weaker than durable delivery — no durable mailbox, idempotency key, delivery receipt, retry protocol, or exactly-once claim」属实。反方2 承认这是 dsh 当前的投递可靠性短板；但正方3 未提供 pi 在此维度更强的对照证据，不能默认 pi 更优，只能判为「dsh 已披露、待补齐」。

## 本轮结论（一句话）

正方三方以「成熟稳定」论证 pi 更优，但都默认了「绑定单一生态 = 长期优势」，且正方3 出现「dsh 绑定 DeepSeek」的事实错误；反方2 坚持：成熟度是当前时点优势，可迁移/不绑定才是 rick 方法层长期先进性（human N1 的 3 年风险）所需的结构性优势。
