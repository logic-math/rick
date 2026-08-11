# research-E-r4 节点 A — rick 的方法是否在 LLM 训练数据 G 中

节点路径：[根 > E-r4-zero-shot对比 > A-rick方法是否在G中]
事实陈述：rick 的方法（doing/learning/dream、sense 5 阶段、plan-do-learn、debug 循环、act-path）是否在 LLM 训练数据 G 中？（即这些方法是否被通用工程实践/公开语料隐式覆盖）

## 执行动作

1. 代码原文：Read rick `internal/prompt/templates/doing.md`（doing 模板，注入 doing_loop_content/loops_context/skills_context/debug_context）+ `internal/prompt/templates/skills/doing_loop.md`（Doing Loop Step 0-5 完整协议）+ `internal/prompt/templates/think.md`（think 6 步假设分析）
   - 注：rick 源码在 `/workdir/sunquan20/AI_CODING/rick`（带下划线，git repo）；本 brief 写入 `/workdir/sunquan20/AICODING/rick`（无下划线，dispatch 所在）
2. 运行时 demo：claude CLI 问"plan-do-learn cycle / sense S/E/N/S-R/EC / doing-learning-dream 三阶段"是否为已知命名方法论（无工具，从记忆答）
3. 文档：arxiv API 检索 "plan-do-check-act" / "plan-do-learn" → 0 篇 ML 文献命中（结果为 carbon/satellite/green-AI 无关项）；arxiv 检索 self-refine/Reflexion/in-context-learning-limits/Plan-and-Solve（节点 D 共用）
4. 复用 E-r2：`briefs/research-E-r2-C.md`（rick GenerateDoingPromptFile 确定性注入 + RFC-001 信息网络流）
5. 信源权重默认：代码 0.4 / 运行时 0.3 / 文档 0.2 / 反事实 0.1

## 信源验证结果

### 代码原文（权重 0.4）✅（rick 方法是独创编排，非通用组合）

**证据 1 — doing_loop.md 是 rick 原创编排**（`internal/prompt/templates/skills/doing_loop.md`，Step 0-5）：
- Step 0：Domain 搜索（强制读 `domain_dir`）+ Loop 匹配（trigger 字段匹配，有匹配执行项目 Loop，无匹配执行默认 Loop）
- Step 1：Main Agent 确认全局目标（task.md 目标 + Key Results + 成功标准）
- Step 2：Main Agent 压缩上下文（bug\*.md frontmatter summary 提取 + 跨轮核心事实）
- Step 3：**Main Agent 启动 Sub Agent**（每轮迭代独立 Sub Agent，执行 `[ANALYZE]→[RED]→[GREEN]→[REFACTOR]→[COMMIT]`，遇红 `[DEBUG]`）
  - ANALYZE：声明 "I will use skill:sense."，按 **S→E→N**（Symptoms/Evidence/Next）分析
  - RED：声明 "I will use skill:tdd."，先写失败测试，必须确认 FAIL
  - GREEN：最小实现，失败→DEBUG
  - DEBUG：触发条件（测试 FAIL/编译报错/行为异常），优先搜 `bugs.md`，创建 `bug{N}.md` Phase 1-6，Phase 4 上限 3 次
  - REFACTOR：测试全绿后改善命名/结构/去重，回归失败→DEBUG
  - COMMIT：git add+commit + check 命令循环直到 pass
- Step 4：Main Agent 产出评估（check pass/测试全通过/Key Results 达成，失败→返回 Step 3 下一轮）
- Step 5：停止标准（成功退出/优雅退出：3 轮上限/连续 2 轮同错/人工停）

→ 这是 **TDD（RED/GREEN/REFACTOR）+ sensemaking（S/E/N）+ debug-skill（Phase 1-6）+ Main/Sub Agent 递归 + 3 轮迭代上限 + check 门禁** 的**特定编排**。各组件是通用工程实践，但**这套精确的步骤序列、阶段命名、Sub Agent-per-iteration、3 轮上限、check 门禁**是 rick 原创编排，非任何单一已知方法论的复刻。

**证据 2 — think.md 是 rick 原创假设分析协议**（`internal/prompt/templates/think.md`，6 步）：
推理识别（演绎/归纳/溯因）→ 假设提取（多视角强制：3 类推理各≥1 + 交叉≥1）→ 数量保障（min_assumptions，反事实/边缘/隐含补强 2 轮）→ 形式化"如果 X 那么 Y"+3 启发性问题（信念/前提/反例）→ 4 维打分（影响范围/不可逆性/影响程度/置信度）→ 期望值公式 + top-N（浮动+阈值）。
→ 形式化的"假设清单 + 启发性 3 问 + 期望值打分"协议是 rick 原创，非标准批判性思维框架的复刻。

**证据 3 — doing.md 模板强制注入**（`internal/prompt/templates/doing.md`）：
```
{{loops_context}}  {{skills_context}}  {{debug_context}}  {{task_info_section}}
{{doing_loop_content}}   ← Doing Loop Step 0-5 注入
**你需要一步步执行以下操作，不可跳过任何步骤。**   ← 强制
```
→ rick 通过模板把上述编排**注入** prompt，非建议。

### 运行时行为（权重 0.3）✅（决定性——LLM 自证不认识 rick 特定方法）

**Demo — LLM 从记忆识别 rick 方法名**（claude CLI，无工具）：
prompt："Describe from your own knowledge: (1) plan-do-learn cycle; (2) sense dialectical method S/E/N/S-R/EC; (3) doing/learning/dream three-phase loop. For each, say whether you recognize it as established or project-specific."

LLM 输出（原文要点）：
- **plan-do-learn**："I do **not** recognize 'Plan-Do-Learn' as an established, named software engineering methodology. It is clearly patterned on PDCA/PDSA (Deming/Shewhart) + Build-Measure-Learn (Lean Startup). **plausible project-specific or organizational naming variant** rather than a standardized methodology."
- **sense S/E/N/S-R/EC**："I do **not** recognize this as an established named methodology. The abbreviation set doesn't map onto any dialectical or systems-thinking framework I know... doesn't match OODA / SECI / Kolb's... **project-specific**."
- **doing/learning/dream**："I do **not** recognize this as an established named methodology either. The 'dream' phase is **atypical for SE**... **project-specific**."

→ 三重结论：
1. **通用组件在 G 中**：LLM 能按名回忆 PDCA/PDSA、Build-Measure-Learn、OODA、SECI、Kolb、thesis-antithesis-synthesis → 这些是公开语料常识
2. **rick 特定编排不在 G 中**：plan-do-learn / sense S-E-N-S-R-EC / doing-learning-dream 均被判定为 "project-specific"，LLM "inferring, not recalling"
3. **human 假设精确印证**："rick 的方法可能包含在了训练数据中"（组件包含，✅）"但作为通用的模型它并不会选择用这个方法"（特定编排不被识别/不被选，✅）——见节点 B

### 文档（权重 0.2）✅（通用方法论在公开语料；rick 编排不在 ML 文献）

**证据 1 — 通用方法论是公开语料常识**（LLM runtime demo 旁证 + 学术常识）：
- PDCA/PDSA（Deming/Shewhart cycle）、Build-Measure-Learn（Eric Ries Lean Startup）、OODA（Boyd）、Kolb experiential cycle、TDD（Beck）、thesis-antithesis-synthesis（Hegel）——均为有标准引用的公开方法论，必然在 LLM 训练语料中
- arxiv API 检索 "plan-do-check-act" OR "plan-do-learn"：0 篇 ML/NLP 文献命中（结果为 carbon-aware satellite / green-AI / image anonymization 无关项）→ "plan-do-learn" 非学术文献术语

**证据 2 — 自我反思/迭代方法论是学术热点（但非 rick 编排）**：self-refine（arxiv 2303.17651）、Reflexion（2303.11366）、Plan-and-Solve（2305.04091）——这些是**通用**"迭代精炼/反思"框架，非 rick 的 sense+TDD+debug+subagent+3轮 精确编排。rick 编排是这些思想的**特定工程化组合**，组合本身不在文献中。

### 反事实（权重 0.1）❌

- 未修改代码做反事实（Node A 是事实性"在/不在 G"问题，反事实不直接适用）
- runtime demo 已构成事实判定（在 G→应识别；不识别→不在 G），但该判定属 runtime 信源，不重复计入反事实

## 还原确认

无 rick 代码修改，无需还原。Read/curl/claude CLI 均只读。

## 置信度评估（由 research 主调度计算）

- 代码原文 ✅ × 0.4 = 0.4（doing_loop.md Step 0-5 + think.md 6 步 + doing.md 强制注入，rick 原创编排三源一致）
- 运行时 ✅ × 0.3 = 0.3（LLM 自证不认识 rick 三方法名，判定 project-specific）
- 文档 ✅ × 0.2 = 0.2（通用方法论在语料 + rick 编排不在 ML 文献）
- 反事实 ❌ × 0.1 = 0.0
- **合计 = 0.9（高，≥0.8 终止）**

## 关键事实

1. **✅ rick 方法"部分在 G"**（nuanced 回答 human）：
   - **组件在 G**：TDD、PDCA、sensemaking、debug、迭代、subagent——均为公开语料常识（LLM 按名可回忆 PDCA/OODA/Kolb/TDD）
   - **特定编排不在 G**：plan-do-learn / sense S-E-N-S-R-EC / doing-learning-dream / Doing Loop Step 0-5（Sub Agent-per-iteration + 3 轮上限 + check 门禁 + S→E→N + RED/GREEN/REFACTOR/DEBUG/COMMIT 精确序列）——LLM 判定 "project-specific"，非任何已知方法论复刻
2. **human 论断精确成立**："方法可能包含在训练数据中"（组件包含 ✅）+ "通用模型不会选择用这个方法"（特定编排不被识别/选，见节点 B ✅）
3. **关键辨析**：rick 的价值不在"发明了新组件"（组件都是常识），而在"**确定性地编排**了一套特定做事方法"（见节点 C）——这与 human "rick 确定性地选择此方法（含判断）"论断一致

## 疑问点

- 无疑问点阻断。节点 A 置信度 0.9 达高，终止。
- 边界：本节点证"组件在 G / 编排不在 G"，不证"通用模型是否选此方法"（节点 B）或"确定性选择是否关键差异"（节点 C）

## R7 上报

- 无。节点 A 置信度 0.9（高），终止。
