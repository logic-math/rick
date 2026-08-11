# 派发：research subagent — N1 矛盾生成（系统论描述符）

E 阶段门禁已 ✅ 通过，进入 N 阶段。N1 = 基于已确认视角，用系统论描述符描述系统，分析稳态，列举多种相互矛盾的状态供 human 选择。

**先读**（如未在上下文）：
- `loop_4/prompts/skill_research.md`、`loop_4/prompts/research.md`（角色与工作流）
- `loop_4/judgment.md`（E 收敛结论）
- `loop_4/briefs/批判门禁-E-r5.md`（核心假设 / rick 价值论 / 架构定位最终形态）
- `loop_2/judgment.md` + `loop_2/briefs/research-report-{3,5,7}.md`（S 阶段已尽调的 rick/pi 现状与迁移事实，复用避免重复）

无 `.rick/config.json`，信源权重默认（代码 0.4/运行时 0.3/文档 0.2/反事实 0.1），高置信 ≥0.8。运行时适配同前：你即调研执行者，可直接用 Read/Grep/Bash/WebFetch/WebSearch，保留尽调树/MECE/加权/R7/落盘/`git restore` 全部约束。

---

## 前序判断（E 阶段已确认视角，N1 基于此描述系统）

- **核心假设**：∃G 外 G′（未见过+无法一次性解决）→ 需迭代/实验探索 → rick 存在。依赖链 A7（有损+非确定）+A5（G′存在）+A15（zero-shot 不选 rick 编排）+A18（单轮不足/多轮改善）全 CONFIRMED。
- **rick 价值论**（D3′重锚）：价值=弥补参数记忆有损+非确定（LLM 内禀，与训练成本正交，刚性）；手段=应对上下文熵增；实现=确定性编排+强制执行+含判断的选择+迭代框架。
- **架构定位**（D3）：rick=引导程序（引导人类正确模式[pi 不可内化]+引导 pi 加载系统提示词）；价值主体=rick；"可完全内嵌 pi"≠"功能不可替代"；rick=方法，pi=实现，遵循 sense 的 pi=rick。
- **S 阶段已确认**：现状 rick+ai_cli+claude code；期望迁移 pi+深度定制（二进制/skill 系统级/自定义 compaction/subagent 扩展）；差距=缺具体实现计划。

## 任务

基于上述视角，用**系统论描述符（5 要素）**描述"rick+pi+LLM+human+外部存储"这一解决 G′ 问题的系统：

| 要素 | 含义 | 需识别 |
|---|---|---|
| node | 系统组件 | human / rick（引导程序·方法）/ pi（agent loop·实现）/ LLM（参数权重·模型）/ 外部存储（doing.md·sense_loop.md·OKR·SPEC·task·debug·skills·loops——确定性信息存储）等 |
| input | 系统输入 | human 的任务需求（G′ problem）/ 上下文（系统提示词·文件）等 |
| output | 系统输出 | 解决 G′ 的产物（code/RFC/决策）/ 学习沉淀（learning/loops）等 |
| inner | 内部协作 input/output | rick→pi（系统提示词注入）/ pi→LLM（上下文·compaction）/ rick→外部存储（确定性加载拼装）/ LLM→外部存储（检索读写）/ human→rick（命令·模式）/ pi→human（交互·简报）等 |
| edge | node 间协作关系 | 承载 inner_input/inner_output 的边 |

### 具体调研

1. **node/input/output/inner/edge 列表 + 图**：基于 rick 源码（`internal/prompt/`、`internal/doing/`、`internal/cli/` 等，复用 loop_2 research-5/7 的调用链与功能枚举）+ pi 文档/源码（loop_2 research-2/3/4 的 pi 扩展点/compaction/subagent）识别 5 要素，画 ASCII 系统图。
2. **稳态分析**：当前稳态 A（rick+ai_cli+claude code）→ 目标稳态 B（rick+pi+深度定制，二进制/skill 系统级/自定义 compaction/subagent）所需控制手段。
3. **多种相互矛盾的状态**（供 human 在 N2 选择主要矛盾）：列举系统演化中相互矛盾的状态对。候选方向（自行补充）：
   - rick 轻量化（仅引导程序）vs 门禁/做事方法内嵌 pi 深度定制
   - rick=方法（可完全内嵌 pi）vs rick=独立引导程序（引导人类，pi 不可内化）
   - 确定性提取/强制执行（rick）vs LLM 参数记忆有损+非确定（内禀）
   - 迭代探索（解决 G′）vs 单次交付期望
   - 训练成本高（G 过去式刚性）vs 价值重锚（与训练成本正交）
   - rick 约束人类正确模式 vs 人类自由度
4. **human 启发性追问**（简报末尾，照 sense_loop N1 格式）：
   - 在这个系统中，你看到哪两股力量在拉扯？
   - 如果系统继续按现状运行，3 年后会发生什么？
   - 系统的哪个节点，如果消失，整个系统会重组？

## 交付标准

按 research.md 的 N1 简报格式（尽调树快照 + 节点详情后追加）：系统论描述符列表+ASCII 图 + 稳态分析（A→B 控制手段）+ 矛盾状态列表 + human 启发性追问。R7 上报无法达高置信度的叶节点。

**禁止**：简报含倾向性（不推荐某矛盾为主要）、替 human 选择矛盾、无事实支撑构建选项、跳过 MECE。

## 产物写入

主报告：`loop_4/briefs/research-report-N1.md`；节点详情按需 `research-N1-{要素}.md`。

## 返回

N1 简报（系统描述符+图+稳态+矛盾状态列表+追问）即为最终输出。
