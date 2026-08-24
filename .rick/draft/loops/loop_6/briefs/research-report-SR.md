# 调研报告 — S-R 辩证逆转（对主要矛盾 K4 的逆转尽调）

日期：2026-08-13
阶段：S-R（辩证逆转）· research subagent 输出

---

## 信源配置

| 信源 | 权重 | 验证方式 | 本次使用 |
|---|---|---|---|
| 代码原文 | 0.4 | Read/Grep 直接读取 | ✅（pi-subagents docs/agents.md + configuration.md；rick human_loop_prompt.go + tools_init_pi.go + settings.json + .pi 目录） |
| 运行时行为 | 0.3 | Bash 跑命令 | ✅（`find`/`ls` 核实 rick 的 .pi 与 ~/.rick/pi/agent 目录结构） |
| 文档 | 0.2 | Read 官方文档 | ✅（pi-subagents agents.md/configuration.md 官方规范） |
| 反事实 | 0.1 | 修改后还原 | ❌ 未使用（S-R 只读） |

置信度 = Σ(信源验证结果 × 权重)。高 ≥ 0.8。「文件包含/不包含某文字/某目录是否存在」由直接 Read/find 判定为确定（高）。

---

## 尽调树（快照）

```
根：S-R 辩证逆转（对主要矛盾 K4 做逆转尽调）
├─ 阻碍层（node/edge 定位）
│   ├─ B1 rick node 双重身份（方法层 vs 提示词模板）✅高
│   ├─ B2 E2/E3 边协议不对齐（深度改造的触发点）✅高
│   └─ B3 "深度改造 vs 独立性"零和假设的根源（无稳定外化载体）✅高
├─ 逆转逻辑层（候选，供 human 判）
│   ├─ R1 正向逆转（深度改造必然 → 方法层外化）候选
│   ├─ R2 反向逆转（方法层必然 → 翻译为 pi 原生资产）候选
│   └─ R3 统一解（深度改造 = 翻译方法层，非零和）候选
└─ 替代路径层（实现可选项，每条含事实支撑）
    ├─ P1 自定义 agent（frontmatter 注册 think/research/exporter）✅高
    ├─ P2 pi skills（.pi/skills 注册 sense 方法论）✅高
    ├─ P3 agentOverrides（settings.json 覆盖内置 agent）✅高
    ├─ P4 refinement overlays（.pi/subagents/refinements）✅高
    └─ P5 纯提示词对齐（显式语法 + 内置 agent 名，不注册）✅高
```

树规模：深度 2 | 总节点 12 ≤ 30 ✅

---

## 阻碍层（基于系统论描述符 node/edge）

### B1：rick node 的双重身份（根因所在 node）

- **事实**：rick node 在系统中承担两类资产，二者当前**混在同一批 markdown 文件**里：
  1. **方法层**（先进性来源，human N1「rick 是核心节点」）——SENSE 五阶段框架（`skill_sense.md`）、think pipeline（`skill_think.md`）、research 尽调树（`skill_research.md`）、exporter RFC（`skill_exporter.md`）。这些是 rick 的**方法论知识**，与 pi 无关、可长期存在。
  2. **提示词模板**（触发指令，需对齐 pi）——`sense_loop.md`/`plan.md`/`dream.md` 等，含 243 处自然语言 subagent 触发词、0 处 pi 触发语法（F1/N3.1/N3.2）。
- **落盘位置（关键事实）**：二者都被 `human_loop_prompt.go` 写到**每次会话的临时目录** `loop_N/prompts/*.md`（`WriteSkillFile` + `BuildAndSaveToDir`），模板源在 `internal/prompt/templates/`。即 rick 的方法层**没有稳定、被 pi 发现的外化载体**。
- **置信度**：高（代码原文直接读取）。

### B2：E2/E3 边协议不对齐（深度改造的触发点）

- **事实**：延续 N1 系统论描述符——协议不对齐集中在 E2（rick→main agent 提示词注入，软触发）与 E3（main agent→pi-subagents 工具调用，需 workflowScript + 真实 agent 名）。
- **置信度**：高（沿用 N1 已证实 F1/BP-1/BP-2）。

### B3：「深度改造 vs 独立性」零和假设的根源

- **事实性定位（不替 human 判断零和是否成立）**：矛盾 K4 的"不可调和"感来自一个**未明言的前提**——"深度改造"被等价为"放弃 rick 方法层/迁就 pi 语言"，"独立性"被等价为"保留 rick 私有语言"。但 pi 提供的事实机制（见替代路径 P1~P4）显示：**rick 方法层可以被"翻译/外化"为 pi 原生资产**（自定义 agent 或 skills），翻译 ≠ 放弃。故"深度改造 vs 独立性"是否存在非零和解，取决于是否启用外化机制。
- **证据**：agents.md「An agent is a markdown file: YAML frontmatter on top, a system prompt below」+「Custom agents start with a clean system prompt and only the context you intentionally give them」+ Skills 段（`.pi/skills/{name}/SKILL.md` 项目级发现）。
- **置信度**：高（文档原文）。

---

## 逆转逻辑层（候选，供 human 判断，不替 human 选）

> 核心追问：「如果 [深度改造对齐 pi 的 subagent 体系] 是必然发生的前提，要想实现 [触发确定性提升到上限内最高 + rick 方法层先进性不随模型迭代丧失]，我们应当如何？」

### R1 正向逆转：「若深度改造必然，则方法层应当被外化」

- **形式化**：若 [深度改造对齐 pi 是必然前提] 成立，那么 [rick 方法层不应当继续以 rick 私有自然语言存在，而应当被外化为 pi 体系内的原生资产] 成立。
- **事实支撑**：rick 方法层当前无稳定外化载体（B1）；pi 提供自定义 agent / skills 两种外化容器（P1/P2）；「模型不总会遵守软指令，需显式提示强制」（C2）。
- **融贯性**：自洽（外化即深度改造的内容而非代价）；他洽（与 B1/C2/P1/P2 一致）；续洽（预测"方法层外化为 pi agent/skill 后，触发确定性 + 方法层持久性同时提升"）。

### R2 反向逆转：「若 rick 方法层必然（先进性来源），则深度改造应当翻译而非迁就」

- **形式化**：若 [rick 方法层是先进性来源、不可消失是必然前提] 成立，那么 [深度改造对齐 pi 应当采取"把方法层翻译成 pi 语言体系的原生表达"，而非"迁就 pi 而稀释方法层"] 成立。
- **事实支撑**：human N1「rick 是核心节点，消失则退化为平庸 pi agent」「3 年后失去先进性」；B1 方法层与提示词模板可分离。
- **融贯性**：自洽（翻译保留方法层、迁就稀释方法层）；他洽（与 human N1 判断 + B1 一致）；续洽（预测"以翻译方式深度改造，先进性不随模型迭代丧失"）。

### R3 统一解（辩证逆转的更高层）：深度改造 = 把"独立性"翻译进 pi，而非牺牲独立性

- **形式化**：K4 的"深度改造 vs 独立性"在更高系统层次是**对立统一**——深度改造的对象不是"放弃方法层"，而是"把方法层从 rick 私有语言翻译为 pi 原生语言"（agent frontmatter / skill），使 rick 方法层成为 pi 语言体系内的"一等公民"。
- **事实支撑**：pi 的 agent/skill 机制允许"自定义系统提示词 + 自定义技能"（agents.md），即 pi 语言体系**本身为外来方法层预留了承载位**；human E 阶段「两个各自自洽、合作时暴露问题的语言体系」——翻译即让两个语言体系共享同一份"方法层资产"。
- **融贯性**：自洽（翻译统一了"深度改造"与"方法层保留"）；他洽（与 B1/B3、agents.md、human E 类比一致）；续洽（预测"方法层外化为 pi agent/skill 后，K4 矛盾被化解而非被牺牲"）。

> 三者递进：R1/R2 是同一逆转的两个表述方向，R3 是其更高层统一解。**是否成立、选哪条，由 human 判断。**

---

## 替代路径层（实现可选项，每条含事实支撑，不推荐排序）

### P1：自定义 agent（把 think/research/exporter 注册为 pi 自定义 agent）

- **事实**：自定义 agent = frontmatter markdown（`name`/`description`/`tools`/`systemPromptMode`/`inheritSkills`/`defaultContext` 等）。注册作用域：project 级 `.pi/agents/**/*.md`（最高优先级）；user 级（rick 隔离环境）= `~/.rick/pi/agent/agents/**/*.md`。触发用 `runs.run(agent:'think', task:'...')`。
- **rick 现状对照**：rick 零自定义 agent——`~/.rick/pi/agent/` 无 `agents/` 目录、仓库 `.pi/` 无 `agents/`；think/research/exporter 仅为 `loop_N/prompts/*.md` 普通文件（B1/B4）。
- **适用性事实**：`systemPromptMode: replace`（默认，空系统提示词起步）/ `append`（保留 pi 基础提示词）；`inheritSkills: true` 让子 agent 看到 pi skills 目录；`defaultContext: fork`。
- **置信度**：高（agents.md + B1~B4）。

### P2：pi skills（把 sense/think/research/exporter 方法论注册为 pi 技能）

- **事实**：pi skills = `SKILL.md` 文件，项目级发现路径 `.pi/skills/{name}/SKILL.md`，user 级（rick 隔离环境）= `~/.rick/pi/agent/skills/{name}/SKILL.md`。parent 触发时用 `skill: "name"` 参数注入，或靠技能元数据（name/description/location）让模型按需 `read` 加载。
- **rick 现状对照**：rick 已有 `skill_sense.md/skill_think.md/skill_research.md/skill_exporter.md` 四份方法论文件，但**命名/目录结构不符合 pi skills 约定**（pi 要求 `{name}/SKILL.md`），且落在临时 `loop_N/prompts/` 而非 `.pi/skills/`；`~/.rick/pi/agent/` 无 `skills/` 目录。
- **适用性事实**：`agents.md` Skills 段「Project config skills/{name}/SKILL.md (.pi/skills/...)」为项目级发现；`inheritSkills: false` + 显式 `skills` 可控制子 agent 只收选定技能。
- **置信度**：高（agents.md Skills 段 + B1）。

### P3：agentOverrides（settings.json 覆盖内置 agent 的 systemPrompt/description 等）

- **事实**：`settings.json` 的 `subagents.agentOverrides.<name>` 可覆盖内置 agent 的 `systemPrompt`/`description`/`model`/`inheritProjectContext` 等字段，无需复制整个 agent。
- **rick 现状对照**：rick 的 `~/.rick/pi/agent/settings.json` 目前**无 `subagents` 配置块**（仅 hideThinkingBlock/packages/theme）。
- **置信度**：高（agents.md「Overriding builtins」+ settings.json 实测）。

### P4：refinement overlays（项目级给内置/自定义 agent 叠加 guidance）

- **事实**：`.pi/subagents/refinements/<agent>.md` 以 `<pi-subagents-refinement>` 块注入该 agent 的 child system prompt，不动 agent 定义本体；`{action:"refine"}` 从近期 run 证据自动起草。
- **rick 现状对照**：仓库 `.pi/subagents/` 现只有 `artifacts/`、`missions/`（运行产物），无 `refinements/`。
- **置信度**：高（agents.md「Refinement overlays」+ .pi 目录实测）。

### P5：纯提示词对齐（不注册任何 agent/skill，仅改触发措辞）

- **事实**：把 rick 模板中的自然语言触发词改为显式 `subagent({workflowScript:"return runs.run('main',{agent:'worker'|'scout'|'reviewer'|...,task:'...'})"})` + 内置 agent 名，用 pi 内置 7 个 agent 直接承载 think/research/exporter 的角色，不改动 rick 方法层文件。
- **适用性事实**：BP-1/BP-2（显式语法 + 内置名）；BP-2 内置名 = scout/worker/reviewer/researcher/delegate/oracle。
- **置信度**：高（BP-1/BP-2 已证实）。

---

## R7 上报项（无法达高置信的叶节点）

1. **各替代路径的触发确定性提升幅度**——需真实 pi 会话实测（延续 S 阶段 R7），只读阶段无法量化。
2. **"方法层外化"是否能保持先进性（模型迭代后不退化）**——属 human N1 的 3 年预测 + 因果假设，无法只读证实。
3. **R1/R2/R3 逆转逻辑的成立性**——属 human 判断，research 只摆候选与事实支撑，不替 human 判。
4. **P1~P5 之间"唯一最优 vs 组合"关系**——属 human 决策，research 不替选。

---

## 整合摘要

阻碍 3 项（B1~B3，高置信）| 逆转逻辑候选 3 条（R1~R3，供 human 判）| 替代路径 5 条（P1~P5，均高置信事实支撑）| R7 上报 4 项。

**关键事实（非建议）**：pi 语言体系**本身为外来方法层预留了承载位**——自定义 agent（frontmatter）、skills（SKILL.md）、agentOverrides、refinement overlays 四种机制都可承载 rick 方法层；而 rick 现状是四种机制**均未启用**（零自定义 agent、零 pi skills、settings 无 subagents 块、无 refinements）。这为「深度改造 = 把方法层翻译进 pi 而非牺牲独立性」的逆转逻辑提供了事实基础。

---

## human 启发性追问（照 sense_loop S-R 格式）

① 如果 [深度改造对齐 pi] 是不可避免的前提，实现 [触发确定性 + rick 先进性] 的最意想不到的路径是什么？

② 什么看似阻碍的力量，其实可以转化为推动力？

③ 在 [深度改造] 必然的前提下，[触发确定性 + rick 先进性] 实现的"逆向工程"是什么？
