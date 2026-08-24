# 反方1 反驳（Round 2）

> 立场：轻量/简洁/可控。针对正方1/2/3 的核心论点逐一辩驳，证据附出处。

## 针对正方1「subagent 编排/触发机制」的反驳

**1. 驳「pi 触发入口唯一且显式 → 确定性高」**：pi 的"唯一入口"是 `subagent({ workflowScript: "..." })`，即要求模型在工具调用的**字符串参数里手写 JS**（`runs.run(key,{agent,task})`，见正方1 论点1 自引的 BP-1）。入口唯一 ≠ 确定性高——它把编排逻辑塞进一个**无 schema 的自由字符串**，模型拼错 JS 就静默不触发，正是 rick 现状"触发概率低"的直接根因（N3.1：rick 模板 0 处触发语法）。dsh 的触发面是**强类型服务 API** `ctx.subagents.start(name, request)`（`packages/subagent/subagent/README.md` Service API 节 + 测试实调用 `subagent-acp/tests/*.spec.ts`），模型侧经 tool schema 暴露、参数受静态校验。结论：pi 的"唯一且显式"实质是"唯一但脆弱"，dsh 是"可校验的契约"。

**2. 驳「pi 的 fail loud（Unknown agent）保证确定性」**：pi 的 fail loud 只覆盖"agent 名写错"这一种失败；更大的失败面是"模型没写 workflowScript / JS 拼错"，这类失败**不 fail loud、静默不触发**——这正是 human 观察的触发概率低。dsh 在加载期就 fail loud（`AGENTS.md` Conventions：「Misconfiguration fails loud at load」），错误暴露得更早。

**3. 驳「pi 编排权集中 parent 契合 rick 架构」**：架构契合这点我承认，但须限定边界——pi 的"编排权在 parent"是**提示词软约束**（`constraints-and-recipes.md` 约定，非运行时强制），rick 现状 D6"谁触发模糊"正是软约束失效的体现。dsh 在**源码层强制**：`assertSubagentMaxDepth`（深度强制）、delegated policy 把子 agent 审批钉在 `'never'`（`packages/subagent/subagent/README.md`）。同样"编排权"，pi 是祈使，dsh 是保证。

**4. 驳「dsh 触发面分散 / 角色组合 / 参数多 / 递归过度设计」**（正方1 缺点1/2/3/5）："多 provider 共存"是 dsh「无特权核心、一切皆插件」的**必然结果**而非缺陷——`docs/architecture.md:11`「every part is replaceable from configuration」，rick 只需在声明式 cordis.yml 里选定一个 provider，不存在"每次触发都要选"。dsh 的 composition（`applyChildComposition`）是显式、可审计的，而 pi 的自定义 agent 是"空系统提示词起步、需手动拼全部上下文"（N3.3′），未必更简单。参数多但有 schema + 默认值，优于"参数少 + 自由字符串"。递归/continuable 是可关闭能力（`maxDepth` 可设 0 禁委派、`backgroundMode` 可选 one-shot），能力多余是中性甚至优势，不是缺点。

## 针对正方2「扩展生态/长期演进」的反驳

**1. 驳「pi 版本成熟（0.84/0.47）优于 preview」**：版本号高 ≠ 解决 rick 的问题——pi 成熟如斯，rick 在 pi 上**依然触发概率低**（human 观察，S 阶段确认）。反观 dsh 的 preview 是**双刃剑**：破坏性变更风险 vs **在 API 冻结前把 rick 需求铸进去**的机会。rick 做长期决策应看演进方向而非当前版本号；pi 的成熟意味着"协议已定死、rick 只能适应"，dsh 的 preview 意味着"rick 可以影响"。

**2. 驳「pi 有 9 条最佳实践（BP-1~BP-9）是稳定契约」**：为"让模型触发子 agent"这一件本应简单的事需要 **9 条最佳实践 + 手写 JS**，恰恰证明 pi 触发机制**复杂度高、不收敛**（我 Round 1 反驳 2.4）。最佳实践多 = 易错点多，不是优点。dsh 把同一件事收敛为"注册 provider + 挂 tool consumer"。

**3. 驳「pi 扩展生态成熟（agent/skill/agentOverrides/refinement 四种）」**：四种机制并存 = 四种心智负担 + 靠"约定目录 + frontmatter 文本解析"（运行时私有发现逻辑）。dsh 是**统一插件范式**（Cordis）覆盖所有扩展点，扩展点是类型化服务（`ctx.subagents`/`ctx.tools`）而非文本解析。

**4. 驳「rick 已在 pi 迁移完成（P4）」**：迁移成本是**沉没成本**，不能作为"pi 更好"的证据（沉没成本谬误）。且这次迁移恰恰**暴露**了 pi 的触发问题（触发概率低），说明"已迁移"≠"迁移得好"。评估应看 human N1 的"未来 3 年先进性"，而非"已投入"。

**5. 驳「dsh 会话格式无兼容承诺（SESSION_FORMAT_VERSION=0）/ 可改名重组（D2/D3）」**：这被夸大了。rick 的方法层资产（SENSE 五阶段 + judgment/doing/learning 沉淀）是 **rick 自己管理的确定性文件**，与 dsh 的 session 磁盘格式**正交**，不落在其存储层。rick 集成的是 dsh 的**插件 seam 抽象**（Cordis 是已发表范式，有论文），而非具体包名；且 pi 的 pi-subagents 同样是第三方，rick 也不控制其演进。SESSION_FORMAT_VERSION=0 对 rick 影响有限。

**6. 驳「dsh 工程门槛高（Node ^22.19 + vendored Cordis，D4）」**：Node 是新增运行时依赖，这点部分承认；但 **vendored Cordis 是可控性的体现**——源码就在仓库 `vendor/` 里，rick 可读/改/审计；而 pi 的核心 agent-loop 是运行时私有实现，rick 不可改。门槛高换来源码级可控。

## 针对正方3「工程能力/运行时成熟度」的反驳

**1. 驳「pi subagent 触发唯一且显式 / frontmatter 声明式（论点1/2）」**：同正方1 论点1、正方2 反驳3——"唯一入口"是"模型手写 JS 字符串"的包装；frontmatter markdown 是"声明式文本"，但其发现/加载/校验是运行时私有实现，而 dsh 是 **strict TS 编译期校验**（`AGENTS.md`：「Everything compiles under strict: true with noImplicitAny」）。同为声明式，dsh 有类型系统兜底，pi 靠文本解析（frontmatter 拼错可能静默/运行时才报）。

**2. 驳「pi 成熟编排原语 + 已迁移（论点3）」**：pi 的"编排原语"（单写者/async/context/编排权）是 `constraints-and-recipes.md` 的**提示词约定**，非运行时强制（同上方针对正方1 第3条）。"3 项 e2e 测试通过"验证的是 JSONL 解析/tool call 通路，**不是 subagent 触发确定性**——后者恰是本次待解决的 R7 未验证项。用"迁移已完成"论证"pi 更稳"是循环论证。

**3. 驳「dsh 触发是软性 guidance 非 enforcement（反驳4）」**：正方3 引用的 `tool-subagent-report` 软点，只在"**子 agent 回报父 agent**"这个环节（README:7「The instruction is guidance, not enforcement」）；而"**父 agent 触发子 agent**"环节 dsh 是 `ctx.subagents.start()`（类型化服务）+ `assertSubagentMaxDepth`（运行时强制）。pi 的"编排权在 parent"同样是提示词软约束。正方3 拿 dsh 的次要环节对比 pi 的核心环节，**不对等**。

**4. 驳「dsh subagent 碎片化 + 投递缺陷（反驳5）」**："multiple providers 并存"是"无特权核心、可替换"的体现（`docs/architecture.md:11`），不是碎片化。dsh 对"Acceptance weaker than durable delivery"是**诚实自述**（透明）；pi 对同类投递可靠性**没有明确承诺文档**——"未披露的缺陷"风险高于"已披露的缺陷"。

**5. 驳「dsh 绑定 DeepSeek 提供方（反驳6）」**：与 dsh 架构自述**矛盾**。`AGENTS.md:17` 的 `llm/ ... + DeepSeek providers` 说的是"**内置** provider 是 DeepSeek"；但 `docs/architecture.md:11` 明言「Every part of the product is a plugin, **including the model adapter** ... so every part is replaceable from configuration」。即 model adapter 是可替换插件，dsh **不锁定** DeepSeek；正方3 的"厂商锁定"结论不成立。

## 本轮结论（一句话）

正方以"成熟/唯一入口/已迁移"论证 pi 更稳，但逐条看：pi 的"唯一入口"实为"模型手写 JS 字符串"（脆弱非确定）、其约束靠提示词软约定、其"成熟"未解决 rick 实际的触发问题；dsh 以"一切皆插件 + 类型化服务 + strict TS + 源码级强制 + model adapter 可替换"在轻量/简洁/可控上更胜，preview 风险被正方高估（rick 方法资产与 dsh 磁盘格式正交、且 preview 意味着可影响 API 设计）。
