# 派发：think subagent — N2 主要矛盾判断（三维打分 K1~K7）

**先读**：
- `/workdir/sunquan20/AI_CODING/rick/.rick/draft/loops/loop_6/prompts/think.md` + `skill_think.md`
- `/workdir/sunquan20/AI_CODING/rick/.rick/draft/loops/loop_6/briefs/research-report-N1.md`（K1~K7 矛盾状态 + 系统描述符 + 稳态 A→B）
- `/workdir/sunquan20/AI_CODING/rick/.rick/draft/loops/loop_6/briefs/批判门禁-N1.md`（N1 门禁结论 + human 已指向 K4）
- `/workdir/sunquan20/AI_CODING/rick/.rick/draft/loops/loop_6/judgment.md`（S/E/N1 收敛判断）

运行时适配同前：你即 think 执行者，直接执行工作流，不递归派发 subagent。top-N 默认 3。

---

## 阶段：N2 主要矛盾判断

**主题**：subagent 在 pi runtime 下触发概率低，优化提示词（使触发确定性提升到上限内最高）。

**前序判断**：
- S/E/N1 收敛：协议对齐视角（V5）；动作=高度定制化改造 rick 提示词；系统论描述符已建（E2/E3 边协议不对齐）；稳态 A→B 控制手段 5 项。
- human N1 已指向：本质矛盾 = K4 改造深度 vs 独立性，立场「完全偏向深度改造，不考虑独立性」。

**任务**：对 N1 的 K1~K7 每个矛盾状态做**三维打分**，输出 top-N：

- **根本性**：1.0 触及根本问题 / 0.5 边缘问题
- **全局性**：1.0 影响全局 / 0.5 影响局部
- **决定性**：1.0 系统从 A→B 必经 / 0.5 仅影响部分
- 总分 = 根本性 + 全局性 + 决定性（满分 3.0）

**特别标注**：human 已指向 K4。对 K4 单独打分并标注「human 已指向」。评估 K4 是否即 top-1，还是有更根本的矛盾。

**「看似次要实则根本」审查**：是否有 K 打分偏低、但实际是更根本的矛盾（如 K3 框架独立性、K5 确定性 vs 能力上限）？是否在 human 指向的 K4 之外构成更根本矛盾？

**N2 human 启发性追问**（简报末尾，照 sense_loop N2 格式）：
- 系统从 A→B 的关键转化点在哪里？为什么是这点而非别处？
- 如果你只能控制一个变量，你会控制哪个？这个变量对应的矛盾是什么？
- 主要矛盾之外，有没有"看似次要实则根本"的矛盾？

## 交付标准

按 think.md 简报格式：矛盾状态打分表（K1~K7，三维 + 总分 + 排序）+ top-N + human 指向标注 + 「看似次要实则根本」审查 + N2 启发性追问。

落盘 `.../loop_6/briefs/think-N2.md`。

**禁止**：简报含倾向性（不替 human 选定主要矛盾、不推荐某 K 为 top）、替 human 决策、无事实支撑构建选项。

## 返回

N2 简报全文（打分表 + top-N + 追问）作为最终输出返回 sense_loop。
