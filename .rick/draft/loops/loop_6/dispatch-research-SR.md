# 派发：research subagent — S-R 辩证逆转（对主要矛盾 K4 做逆转尽调）

**先读**：
- `/workdir/sunquan20/AI_CODING/rick/.rick/draft/loops/loop_6/prompts/research.md` + `skill_research.md`
- `/workdir/sunquan20/AI_CODING/rick/.rick/draft/loops/loop_6/judgment.md`（S/E/N1/N2 收敛判断）
- `/workdir/sunquan20/AI_CODING/rick/.rick/draft/loops/loop_6/briefs/research-report-N1.md`（系统论描述符 + 稳态 + K1~K7）
- `/workdir/sunquan20/AI_CODING/rick/.rick/draft/loops/loop_6/briefs/think-N2.md`（三维打分 + K4 层次评估）
- `/workdir/sunquan20/AI_CODING/rick/.rick/draft/loops/loop_6/briefs/research-report-S-reasons-agent.md`（B1~B4 自定义 agent 机制）

运行时适配同前：你即调研执行者，直接 Read/Grep/Bash，不再递归派发 subagent，保留 MECE/信源加权/R7/落盘约束。

---

## 五派发要素

**阶段**：S-R 辩证逆转

**主题**：subagent 在 pi runtime 下触发概率低，优化提示词（使触发确定性提升到上限内最高）。

**草稿路径**：`.../draft` | rfc：`.../draft/rfc` | 会话：`.../draft/loops/loop_6`

**前序判断**（S/E/N 收敛，human 原话要点）：
- 主要矛盾 = **K4 改造深度 vs 框架独立性**（human 选定，三维打分 3.0 满分）。
- 控制手段 = 理解 pi 触发语言 → 改造 prompt。
- human N1 补充：3 年后失去先进性（模型迭代使 harness 效果变差）；rick 是核心节点（消失则退化为平庸 pi agent）；立场「完全偏向深度改造，不考虑独立性」。

**核心追问（对选中主要矛盾的辩证逆转）**：

> **如果 [深度改造对齐 pi 的 subagent 体系] 是必然发生的前提，要想实现 [触发确定性提升到上限内最高 + rick 方法层先进性不随模型迭代丧失]，我们应当如何？**

（等价反向：如果 [rick 方法层是先进性来源、不可消失] 是必然前提，要想实现 [深度改造对齐 pi]，我们应当如何？——两种表述均请覆盖。）

**任务派发**：对上述逆转逻辑做尽调，为 human 给出可选项：
1. **阻碍**：基于系统论描述符的 node/edge，识别矛盾不可调和的根源（rick 方法层 node 与 pi 协议 node 在 E2/E3 边的不对齐，以及"深度改造牺牲方法层"的风险）。
2. **逆转逻辑**：探索"若 [阻碍方 X] 是 [期望方 Y] 的前提，则 Y 应当 ___"的可能解（例如：把 rick 方法层外化/注册为 pi 系统级 agent，使"深度对齐"与"方法层保留"统一——此仅为方向示例，请尽调后给出有事实支撑的可选项）。
3. **替代路径**：调研可行的实现路径（基于已证实事实 B1~B4：自定义 agent = frontmatter markdown、注册作用域 builtin/package/user/project、触发 runs.run(agent:'name')；以及 BP-7 官方编排 recipe）。

**结果核验**：阻碍（node/edge 定位）+ 逆转逻辑候选 + 替代路径可选项（每条含事实支撑）+ R7。

## 产出与落盘

主报告：`/workdir/sunquan20/AI_CODING/rick/.rick/draft/loops/loop_6/briefs/research-report-SR.md`

**禁止**：简报含倾向性（不替 human 选逆转逻辑）、替 human 判断、无事实支撑构建选项、跳过 MECE。

## 返回

S-R 简报全文（阻碍 + 逆转逻辑候选 + 替代路径可选项 + R7 + human 启发性追问 3 问）作为最终输出返回 sense_loop。追问照 sense_loop S-R 格式：

① 如果 [深度改造对齐 pi] 是不可避免的前提，实现 [触发确定性 + rick 先进性] 的最意想不到的路径是什么？
② 什么看似阻碍的力量，其实可以转化为推动力？
③ 在 [深度改造] 必然的前提下，[触发确定性 + rick 先进性] 实现的"逆向工程"是什么？
