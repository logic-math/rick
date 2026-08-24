# 调研报告 — skill+loop 抽象 + runtime 转义层架构可行性 — S-R 架构深化

日期：2026-08-14
阶段：S-R（辩证逆转）· research subagent 输出
主题：rick 方法层抽象为「skill + loop」、在 runtime 层加「方法→runtime 转义层」的架构是否可行。

---

## 信源配置

| 信源 | 权重 | 验证方式 | 本次使用 |
|---|---|---|---|
| 代码原文 | 0.4 | Read/Grep 直接读取 | ✅（rick templates/skills + human_loop_prompt.go + tools_init_pi.go；pi-subagents docs/agents.md；dsh packages/skill + workflow + subagent） |
| 运行时行为 | 0.3 | Bash 跑命令 | ⚠️ 部分（grep/目录枚举；未做真实转义层实现） |
| 文档 | 0.2 | Read 官方文档 | ✅（pi agents.md / dsh README） |
| 反事实 | 0.1 | 修改后还原 | ❌ 未使用（S-R 只读） |

置信度 = Σ(信源验证结果 × 权重)，高 ≥ 0.8。计分说明：「文件包含/不包含某字段」由直接 Read/Grep 判定为确定（高），不机械套加权。

---

## 尽调树（快照）

```
根：skill+loop 抽象 + runtime 转义层 可行性
├─ A rick 资产层
│   ├─ A1 wiki 资产 = skills/*.md（方法论知识） ✅高
│   ├─ A2 loop 资产 = 顶层 *.md（workflow + 引用 skill） ✅高
│   ├─ A3 tools 声明 = 缺（当前隐式，无显式工具能力声明） ✅高
│   └─ A4 语义覆盖 = SENSE五阶段/三角色/门禁/反向回流/判断记录 均落 loop+skill ✅高
├─ B pi 转义层
│   ├─ B1 pi agent frontmatter 含 tools/skills/skillPath/systemPromptMode ✅高
│   ├─ B2 pi skills = SKILL.md（项目级 .pi/skills/） ✅高
│   ├─ B3 pi workflowScript = 编排（runs.run/runs.all） ✅高
│   └─ B4 转义层落点 = 生成器（rick 抽象 → pi frontmatter+SKILL.md+workflowScript） ✅高
├─ C dsh 转义层
│   ├─ C1 dsh skill 能力族 = ctx.skills + skill-filesystem + tool-skill ✅高
│   ├─ C2 dsh workflow 能力族 = ctx.workflowEngine + tool-workflow（over subagents） ✅高
│   ├─ C3 dsh subagent = preset/persona/toolFilter 组合 ✅高
│   └─ C4 dsh 转义层落点 = 生成器（rick 抽象 → plugin/preset+skill+workflow） ✅高
└─ D 分层成立性
    ├─ D1 方法层抽象是否 runtime 无关 = 需 tools 显式化+抽象化 ⚠️中
    └─ D2 转义层是否仅重定义不改方法层 = 理论成立，未实测 ⚠️中（R7）
```

树规模：深度 3 | 总节点 15 ≤ 30 ✅

---

## 分支 A：rick 现有资产与「skill + loop」抽象的映射

### A1：wiki 资产 = `skills/*.md`（方法论知识）
- **事实**：`internal/prompt/templates/skills/` 下 20 个 skill 文件，含 `sense.md`（SENSE 五阶段）、`think.md`（推理分析+4维打分+3问）、`research.md`（尽调树+信源加权）、`exporter.md`（RFC 输出）、`doing_loop.md`、`learning_loop.md`、`tdd`、`testing`、`grilling`、`gen-skill`、`gen-loop` 等。均为纯方法论知识（无角色指令、无编排）。→ 对应 human 的「skill = wiki」部分。
- **置信度**：高（直接 Read）。

### A2：loop 资产 = 顶层 `*.md`（workflow + 引用 skill）
- **事实**：`internal/prompt/templates/` 顶层 `sense_loop.md`（五阶段推进）、`think.md`/`research.md`/`exporter.md`（三角色工作流）、`plan.md`/`dream.md`/`doing.md`/`learning.md`/`ctrl.md`。每个 loop 通过 `**先读**：{{xxx_skill_path}}` 引用对应 skill，并在正文展开工作流步骤。→ 对应 human 的「loop = workflow + skill」部分（loop 引用 skill 已是既有事实）。
- **证据**：`think.md:5`「**先读**：`{{think_skill_path}}`」；`dream.md` 引用 `{{sense_skill_path}}`/`{{evolve_skills_skill_path}}`/`{{gen_skill_path}}`；`plan.md` 引用 `skill:tdd`/`skill:grilling`/`skill:testing-anti-patterns`。
- **置信度**：高（直接 Read）。

### A3：tools 声明 = 缺失（当前隐式）
- **事实**：rick 的 skill/loop 文件中**无任何显式「工具能力声明」字段**。角色的工具能力（read/grep/bash/write/WebSearch）是**隐式**的——靠角色正文自然语言描述（如 research 的信源表写「Read/Grep」「Bash 跑命令」）。`human_loop_prompt.go` 也只写 skill 文件与角色文件，无 tools 元数据。
- **含义（事实性）**：human 的「skill = wiki + tools」中，「wiki」已存在，「tools」是**待补充的缺失层**——需为每个 skill/角色显式声明其工具能力（才能由转义层翻译为 runtime 的 tools allowlist）。
- **置信度**：高（grep 无 tools 字段命中）。

### A4：语义覆盖判断（SENSE 特有语义能否装进 skill+loop）
- **事实**：SENSE 五阶段（sense_loop.md）、三角色（think/research/exporter.md）、批判门禁（sense_loop.md「批判门禁」节 + think 的「门禁结论」格式）、反向回流（sense_loop.md「反向回流机制」）、判断记录（judgment.md 写入规则）均已作为**流程语义**写在 loop 文件内、作为**方法论语义**写在 skill 文件内。
- **判断（非建议）**：这些语义均为「workflow 步骤 + 方法论规则」两类，可分别落入 loop 与 skill，**无需新的第三种载体**——即「skill + loop」能覆盖 rick 方法层全部语义，前提是补上 A3 的 tools 声明。
- **置信度**：高。

---

## 分支 B：pi 侧「转义层」的可行性

### B1：pi 自定义 agent frontmatter 含 tools/skills/skillPath/systemPromptMode
- **事实**：pi 自定义 agent = markdown（YAML frontmatter + system prompt），frontmatter 支持 `name`/`description`/`tools`/`model`/`systemPromptMode`(replace/append)/`inheritProjectContext`/`inheritSkills`/`defaultContext`(fresh/fork)/`skills`/`skillPath`/`memory` 等字段。其中 **`tools` 是显式工具 allowlist**（如 `tools: read, grep, find, ls, bash, mcp:chrome-devtools`），**`skills`/`skillPath` 是显式技能选择**，`systemPromptMode: replace` 默认空系统提示词起步。
- **证据**：`pi-subagents/docs/agents.md` 第 3/7-9/114-129/149/169-174/242-261 行。
- **含义**：pi 的「frontmatter tools 字段」可直接承载 human 的「skill = wiki + tools」中的 tools 层；「system prompt = wiki」+「skills/skillPath」承载 wiki 与技能。
- **置信度**：高。

### B2：pi skills = SKILL.md（项目级发现）
- **事实**：pi 技能为 `SKILL.md`，项目级发现路径 `.pi/skills/{name}/SKILL.md`（标准 Pi），另有 `skillPath`（agent 定义文件相对路径的私有技能目录）与 `skills`（显式选择哪些技能）。`inheritSkills: false` + 显式 `skills` + `skillPath` 可让子 agent 只收选定私有技能。
- **证据**：`agents.md`「Skills」节（第 283-305 行）与「skillPath」字段说明（第 327 行）。
- **含义**：rick 的 skills/*.md 可落为 pi 的 SKILL.md（或经 skillPath 私有注入），loop 引用 skill 的关系可映射为 pi 的 `skills` 字段。
- **置信度**：高。

### B3：pi workflowScript = 编排（runs.run/runs.all）
- **事实**：pi 触发/编排的唯一执行面是 `subagent({ workflowScript: "..." })`，脚本内 `runs.run(key,{agent,task})`（单子）/`runs.all([...])`（并行）。→ 对应 human 的「loop = workflow」的 runtime 编排承载。
- **证据**：前序 `research-report-S-bestpractice.md` BP-1；pi-subagents tool-description.ts。
- **置信度**：高。

### B4：转义层在 pi 上的具体落点（生成器）
- **事实性结论（非建议）**：转义层可落为一个「方法→pi」生成器，输入 rick 方法层抽象（skill=wiki+tools、loop=workflow+skill），输出：
  1. 每角色一个 agent frontmatter（`name: think/research/exporter`、`description`、`tools:` 从 skill 的 tools 声明翻译、system prompt = wiki 内容、`skills`/`skillPath` 指向 SKILL.md）；
  2. 每 skill 一个 `SKILL.md`（落 `.pi/skills/` 或 skillPath 目录）；
  3. 每个 loop 一个 workflowScript 模板（用 `runs.run(agent:'think',...)` 显式触发）。
- **关键约束核实**：① 空系统提示词起步（systemPromptMode: replace 默认）→ 转义层必须把 wiki 显式注入为 system prompt；② skillPath 相对 agent 定义文件解析 → 转义层需处理相对路径；③ workflowScript 无静态校验 → 转义层生成**固定模板**（一次性配置成本，非模型每次现编），规避此弱点。
- **置信度**：高（机制够用，无硬性缺口）。

---

## 分支 C：dsh 侧「转义层」的可行性（验证"重定义转义层即可适配其他 runtime"）

### C1/C2：dsh 也有原生 skill 与 workflow 能力族（关键发现）
- **事实**：dsh `packages/skill/` 提供技能发现与加载（`ctx.skills`、`skill-filesystem` 从本地文件系统发现技能、`tool-skill` 模型侧加载器）；`packages/workflow/` 提供动态工作流（`ctx.workflowEngine`、`tool-workflow`「runs model-authored orchestration workflows **over subagents**」、`tool-ralph` 固定 fresh-agent 工作流）。
- **证据**：`packages/skill/README.md`（「discovers reusable agent instructions… provider-neutral catalog」）；`packages/workflow/README.md`（「runs model-authored orchestration workflows over subagents」）。
- **含义（事实性，重要）**：dsh 与 pi **都有一等公民的「skill」与「workflow」概念**，只是命名/形态不同（pi=SKILL.md+workflowScript；dsh=skill-filesystem+tool-skill+workflowEngine+tool-workflow）。这直接佐证 human 的「skill + loop」抽象是**两个 runtime 共有的能力边界**，而非 pi 特有——转义层架构有跨 runtime 的事实基础。
- **置信度**：高。

### C3：dsh subagent = preset/persona/toolFilter 组合
- **事实**：dsh 子 agent 由 `applyChildComposition(childCtx, parent, composition)` 组合父级 preset + 子 persona + toolFilter；`start(name, request)` 类型化服务；支持 `depthLimit`(maxDepth)/`toolFilter`/`persona`/`outputSchema`。
- **证据**：`packages/subagent/subagent/README.md`（第 5/15-19/36-43 行）。
- **含义**：dsh 侧转义层可把 rick 的 skill（wiki+tools）→ preset（wiki）+ persona（角色）+ toolFilter（tools），loop（workflow）→ dsh 的 workflow 能力族 + subagent 编排。
- **置信度**：高。

### C4：dsh 侧转义层落点（生成器）
- **事实性结论（非建议）**：dsh 侧转义层是**另一个生成器**（同一 rick 抽象，不同输出形态）：skill→dsh skill（经 skill-filesystem 发现）+ preset/persona/toolFilter；loop→dsh workflow（tool-workflow / Ralph）+ subagent provider。**rick 方法层抽象无需改动，只换生成器的目标格式**。
- **置信度**：高（机制够用）。

---

## 分支 D：分层成立性

### D1：方法层抽象是否足够 runtime 无关
- **事实**：rick 现有 wiki（skills）与 workflow（loops）**本身是 runtime 无关的 markdown**（无 pi 语法、无 dsh 语法——这正是 S 阶段 N3.1/N3.2 的「缺口」，换个角度也是「独立性」的证据）。**唯一未 runtime 无关的部分是「tools」**：当前 tools 隐式、且 pi 的 tools 名（read/grep/bash/mcp:...）与 dsh 的 toolFilter 名不同。
- **判断（非建议）**：分层成立的前提是 rick 方法层**显式声明 tools 且用抽象工具名**，由转义层做「抽象工具名 → runtime 具体工具名」的映射。此为 D1 的 ⚠️ 中置信（需设计验证抽象粒度）。
- **置信度**：中（映射可行性高，但抽象粒度需设计实测）。

### D2：转义层是否「仅重定义、不改方法层」
- **判断（非建议）**：基于 B4/C4，两个 runtime 的生成器共享同一 rick 抽象输入、仅输出格式不同，理论成立。但**尚未实际实现两个转义层并验证切换成本**，故不能高置信断言。
- **置信度**：中 → R7。

---

## 可行性结论

**⚠️ 有条件可行**（架构方向成立，需满足下列条件）——理由：

1. **抽象有事实基础**：human 的「skill + loop」恰好对应 rick 现有资产的既有划分（skills/*.md=wiki、顶层 *.md=workflow+skill 引用），且 pi 与 dsh **都有原生 skill + workflow 能力族**（B1~B3 与 C1~C2），抽象不是臆造、而是两个 runtime 共有的能力边界。
2. **转义层机制够用**：pi 的 frontmatter（tools/skills/skillPath/systemPromptMode）+ workflowScript、dsh 的 preset/persona/toolFilter + skill/workflow 能力族，均能承载「同一 rick 抽象 → 不同 runtime 翻译」。
3. **但存在 5 项缺口/条件**（见下），其中「tools 显式化 + 抽象化」是分层的必要前提。

## 缺口清单（条件）

1. **tools 缺失层**：rick 当前无显式 tools 声明（A3），需为每个 skill/角色补「工具能力声明」才能翻译为 runtime allowlist。
2. **tools 名跨 runtime 不统一**：pi 的 `tools:`（read/grep/bash/mcp:...）≠ dsh 的 `toolFilter` 命名 → 转义层需维护「抽象工具名 → 具体工具名」映射表。
3. **转义层尚不存在**：rick 当前无转义层代码（`tools_init_pi.go` 仅注册 pi-subagents/pi-web-access 扩展，无「方法→runtime 翻译」逻辑），需新建。
4. **pi workflowScript 无静态校验**：转义层须生成固定模板（一次性配置）规避，而非让模型动态拼 JS。
5. **SENSE 特有语义需显式落入 loop**：门禁/反向回流/判断记录已写在 loop 文件（A4），转义层须保证翻译后这些语义不丢失（尤其「编排权在 parent」「单写者」等约束）。

## R7 上报项（无法达高置信的叶节点）

1. **「转义层架构是否真正降低 runtime 切换成本」**——需实际实现 pi + dsh 两个转义层并实测切换（写→切→跑通），当前为理论可行，非实测可证。
2. **「tools 抽象到何种粒度才能跨 runtime 通用」**——需设计并验证抽象工具名集合（如 read/write/execute/search/web 的最小集合），当前无设计定案。
3. **「同一 rick 抽象在 pi 与 dsh 上产出等价行为」**——需双 runtime 实测触发确定性/产出等价性，当前未做。

## 整合摘要

分支 A：4 项（A1/A2/A4 高、A3 高=缺失）｜分支 B：4 项（全高）｜分支 C：4 项（全高）｜分支 D：2 项（中/R7）｜R7 上报 3 项。

**关键结论（事实性，非建议）**：human 的「skill + loop」抽象映射到 rick 既有资产（wiki=skills/*.md、workflow=顶层 *.md 引用 skill），且 pi（frontmatter tools/skills + workflowScript）与 dsh（skill 能力族 + workflow 能力族 + preset/persona/toolFilter）均原生提供可承载该抽象的机制；「方法→runtime 转义层」的分层在机制上成立，但需先补齐 rick 的「tools 显式化+抽象化」缺失层，并以「实现两个转义层并实测切换」验证其真实收益。
