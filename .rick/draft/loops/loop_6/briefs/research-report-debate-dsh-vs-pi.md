# 调研报告 — pi vs deepseek-harness 用于实现 rick 方法的优劣势（多 agent 辩论）

日期：2026-08-14
阶段：S-R 阶段附带调研（human 临时调研请求）· 主 agent 汇总
方法：6 个 subagent 分正方（pro-pi）/ 反方（pro-deepseek-harness）各 3 视角，Round 1 立论 + Round 2 互读互辩，全部观点落盘 `briefs/debate/`，本文为主 agent 汇总。
源码：`/workdir/sunquan20/AI_CODING/deepseek-harness`（HEAD 47f9438，实读）。

---

## 一、研究对象

- **pi**：rick 当前运行时（已迁移，loop_4 RFC），Earendil Works 的 pi-coding-agent（v0.84.1）+ pi-subagents 扩展（v0.47.1）。
- **deepseek-harness（dsh）**：DeepSeek AI 开源的 agent harness（v0.1.0-rc.5，developer preview），架构「一切皆插件」，基于 Cordis。

---

## 二、正方观点（pi 更适合）

| # | 论点 | 证据 |
|---|---|---|
| 1 | 触发入口唯一且显式 | `subagent({workflowScript:"..."})` + `runs.run(agent:'name')`；think/research/exporter 可注册为 frontmatter 自定义 agent 按名确定性触发 |
| 2 | 版本成熟、契约文档化 | pi v0.84.1 / pi-subagents v0.47.1；9 条最佳实践 BP-1~BP-9 |
| 3 | 工程能力成熟、迁移已完成 | 单写者/async/context/编排权 parent；rick 已完成迁移（job_30，3 项 e2e 通过） |
| 4 | 编排权集中在 parent | 子 agent 默认不持 subagent 工具，契合 rick「main agent=复核层」架构 |

**正方对 dsh 的缺点**：developer preview（明示破坏性变更）、SESSION_FORMAT_VERSION=0 无兼容承诺、触发面分散（多 provider）、子 agent 角色靠组合非一等公民、Node ^22.19 + vendored Cordis 门槛高。

---

## 三、反方观点（deepseek-harness 更适合）

| # | 论点 | 证据 |
|---|---|---|
| 1 | 一切皆插件、无特权核心 | Cordis 架构，model adapter/tool registry/session log/agent loop 全可替换 |
| 2 | 类型化服务 API + strict TS + 声明式配置 | `ctx.subagents.start()`、编译期校验、cordis.yml 可 diff 可审计 |
| 3 | 独立性/可迁移/不绑定单一生态 | MIT + subagent 多 provider（spawn/fork/acp/codex/claude-code）+ `llm-pi-ai` 可复用 pi 模型层而不绑定 pi 运行时 |
| 4 | DeepSeek first-party 原生契合 | `deepseek-official` 专用路由 + reasoning 回传省 token + cache 复用；rick 实际就跑 DeepSeek（deepseek-v4-pro） |

**反方对 pi 的缺点**：workflowScript 是「模型手写 JS 字符串」（无 schema、不可静态校验）、约束靠提示词软约定非运行时强制、目录/扩展/触发入口/生态「四重绑定」、DeepSeek 仅走 openai-completions 通用兼容层。

---

## 四、关键交锋与事实澄清（汇总裁决）

1. **「换 runtime 能解决触发确定性吗？」→ 不能。** 这是本次辩论最重要的收敛点：dsh 的 model-facing 子 agent 触发（`tool-subagent`）同样是模型自主决策，其回报通道明确「guidance, not enforcement」；pi 的触发确定性也依赖「提示词与 tool description 对齐」。**触发概率低的根因是「提示词未对齐协议」（rick 243 处自然语言、0 处语法），换 runtime 不自动消除此根因**——rick 迁到 dsh 仍需对齐 dsh 的 tool schema。

2. **「workflowScript 手写 JS 是黑箱吗？」** 正方澄清：是提示词里预写的固定模板（一次性配置成本），非模型每次现编；但其「无 schema、无静态校验」确为真实弱点（rick 现状 D1 的代价）。反方 1 对此的批评部分成立（对比 dsh 的类型化服务 API + strict TS）。

3. **「绑定 vs 可迁移」** 是价值判断分歧：正方主张「绑定稳定对象（pi 已付迁移成本）是优点，绑定 preview（dsh）是缺点」；反方主张「可迁移是结构性优势，preview 风险被高估（rick 方法资产与 dsh 磁盘格式正交）」。双方各有成立处。

4. **事实纠错**：正方 3 早期称「dsh 绑定 DeepSeek」被反方 2/3 纠正——dsh 有 `llm-pi-ai` 包（multi-provider adapter，backed by pi-ai），模型层不锁定 DeepSeek。**但** dsh 的「深度优化」确为 DeepSeek 专用（deepseek-official 路由）。

5. **「DeepSeek 原生契合」是 dsh 的最强点，但其作用域是「模型层」而非「方法层承载」**。rick 的核心矛盾（触发确定性）落在「subagent 编排/触发」层，dsh 在此层与 pi 同构（都靠模型调用工具），并无结构性优势。

---

## 五、优劣势对比总表

| 维度 | pi | deepseek-harness |
|---|---|---|
| 版本/稳定性 | ✅ 成熟（0.84/0.47），契约文档化 | ❌ developer preview，破坏性变更，SESSION_FORMAT_VERSION=0 无兼容承诺 |
| 触发确定性 | 同构：靠模型对齐 tool description；但机制已调研清楚（BP-1~BP-9） | 同构：靠模型对齐 tool schema；「回报」明确 guidance 非 enforcement |
| 方法层承载 | ✅ frontmatter agent/skills 声明式注册，rick 已迁移 | ⚠️ 插件化 seam 可承载，但需自维护 fork |
| 可迁移/不绑定 | ❌ 目录/扩展/触发入口/生态四重绑定（长期风险，human N1「3 年失去先进性」） | ✅ MIT + 多 provider + llm-pi-ai 复用模型层 |
| 模型适配 | ⚠️ DeepSeek 走 openai-completions 通用层 | ✅ DeepSeek first-party（reasoning/cache 优化） |
| 工程可控性 | ⚠️ workflowScript 无 schema 字符串；约束靠提示词软约定 | ✅ strict TS + 类型化服务 API + 编译期校验 |
| 迁移成本 | ✅ 已支付（3 项 e2e） | ❌ 需重新适配（但可复用 pi-ai 模型层） |

---

## 六、核心观点

1. **本次问题的本质是「协议对齐」，不是「运行时选型」。** 触发概率低的根因是 rick 提示词未对齐当前运行时的触发协议（243 处自然语言、0 处触发语法/agent 名）；换到 dsh 同样要「对齐 dsh 的 tool schema」，且 dsh 的触发同样是「guidance 非 enforcement」。**换 runtime 不解决触发确定性。**

2. **pi 是当前更稳妥的实现载体。** 成熟版本 + 文档化触发契约 + rick 已完成迁移，且其「把 think/research/exporter 注册为系统级 agent」的机制与 human 已确认的逆转逻辑（rick=方法、pi=实现，把方法翻译进 pi）直接吻合。

3. **dsh 的两个真实优势是「可迁移/不绑定」和「DeepSeek 原生契合」，但均不构成「现在换」的理由**：前者被其 preview 无兼容承诺抵消（换来的是「自己维护一个无承诺的 fork」）；后者是模型层优势（rick 实际用 DeepSeek 确为加分项），但可通过 pi 的 provider 层迭代部分获得，且与「方法层承载」正交。

4. **长期风险提示（供 human 决策）**：human N1 自判「3 年后失去先进性」的担忧，在 pi 路径上体现为「生态绑定」风险，在 dsh 路径上体现为「preview 破坏性变更」风险。化解方向应是 human 已确认的 S-R 逆转逻辑——**把 rick 方法层「翻译」为可迁移资产（agent 定义/skills，而非深绑 pi 私有目录），保持「方法/实现解耦」**；同时把 dsh 作为「成熟后（首个 tagged release）再评估」的候选，而非当前切换目标。

**一句话核心观点**：pi 是 rick 方法当前正确的实现载体（稳定性 + 迁移已完成 + 触发问题可通过协议对齐解决）；deepseek-harness 的插件化「方法/实现解耦」理念与 rick 的长期诉求精神一致、且对 DeepSeek 原生契合，但它的 developer preview 状态使其暂不具备替代 pi 的条件——长期应在「保持方法层可迁移」的前提下继续用 pi，并在 dsh 成熟后重新评估。

---

## 附：12 份辩论原始文件

`briefs/debate/` 下：正方1/2/3.md、反方1/2/3.md（Round 1 立论）+ 正方1/2/3-rebuttal.md、反方1/2/3-rebuttal.md（Round 2 反驳）。
