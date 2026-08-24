# 派发：think subagent — 批判门禁（S-R 最终辩证判断）

**先读**：
- `/workdir/sunquan20/AI_CODING/rick/.rick/draft/loops/loop_6/prompts/think.md` + `skill_think.md`
- `/workdir/sunquan20/AI_CODING/rick/.rick/draft/loops/loop_6/judgment.md`（S/E/N/S-R 全部收敛 + 本次最终判断）
- `/workdir/sunquan20/AI_CODING/rick/.rick/draft/loops/loop_6/briefs/research-report-SR-architecture.md`（架构可行性 + 5 缺口）
- `/workdir/sunquan20/AI_CODING/rick/.rick/draft/loops/loop_6/briefs/research-report-debate-dsh-vs-pi.md`（pi vs dsh 辩论）

运行时适配同前：你即 think 执行者，直接执行 6 步工作流，不递归派发 subagent。min_assumptions=5，top-N=3。

---

## 阶段：批判门禁（S-R 最终辩证判断）

**主题**：subagent 在 pi runtime 下触发概率低，优化提示词（使触发确定性提升到上限内最高）。

**待审材料（human 本次回答原话）**：

> tools 是隐含在 skill 中的，可以忽略。
> 我认为，这里有一个关键的判断，就是深度定制与独立性这对矛盾，我判断他不是主要矛盾。 是因为我们应该相信未来的 coding agent 的能力足够强，这就意味着原先更换底层runtime 的复杂度 在今天看来并不一定很难。
> 因此我们无需在设计上保留独立，只需要深刻的改进rick，使其不断的提升效果即可。 未来，只需要将这些工作映射迁移到 dsh 上即可。 本质上是因为 coding 让实现这件事变得不再值钱，之前的是方法。
> 我们深刻的，正确的，清晰的描述 rick 的方法，就可以随意切换任意实现。
> 当然，方法与实现是辩证的，在未来的优化中我们会入侵到运行时内部实现方法的更新，方法本身也会包含足够的实现细节，这些细节会指导我们落地具体的设计。
> 说白了，就是要不断的细化我们的方法设计，工程化的描述rick 方法。这份工程化的方法描述，才是核心。
> 我们应该做到，通过这份方法描述，可以被转化为预期行为完全一致的开发计划，并完成开发任务。 等待效果等价的软件实现。
> 这是 rick 要追求的，方法与实现的隔离。 方法以自然语言描述，相信ai coding agent 通过逻辑清晰的自然语言，可以生成等价一致的开发计划 。
> 这是我们所做的假设，只要自然语言无歧义的描述正确的验收标准，就可以实现这一点。

**已知事实（供置信度更新）**：
- N2 曾选 K4（深度定制 vs 独立性）为主要矛盾；human 本轮显式否定之。
- S-R 架构调研：skill+loop 抽象有事实基础，转义层机制够用，但 tools 显式化是缺口（human 本轮否认可忽略）。
- 辩论结论：换 runtime 不自动解决触发确定性；dsh 与 pi 都有原生 skill/workflow 能力族。

**任务**：识别推理（演绎/归纳/溯因）→ 提取假设 → 形式化（"如果 X 那么 Y"）→ 4 维打分 → top-N。若 top-N 中有未澄清的 Y，上报需 human 决策点。**重点审视「自然语言无歧义验收标准 ⇒ 等价一致开发计划」这一核心假设及其隐含前提。**

## 交付标准

按 think.md 批判门禁简报格式：假设列表 + top-N 3 问 + 门禁结论（✅/❌）。落盘 `.../loop_6/briefs/批判门禁-SR.md`。

**禁止**：简报含倾向性、替 human 决策 Y、生成价值性假设（只从已有推理提取）、少于 min_assumptions、只 1 种推理类型、每假设只 1 问、确认性句式。

## 返回

批判门禁简报全文作为最终输出返回 sense_loop。
