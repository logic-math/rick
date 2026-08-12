# 调研报告 — N1 矛盾生成（系统论描述符） — 2026-08-08

> 派发：`loops/loop_4/dispatch-research-N1.md`（E 门禁 ✅ 通过后进入 N 阶段，N1=基于已确认视角用系统论描述符描述系统+稳态分析+列举矛盾供 N2 选择）
> 工作流：`loops/loop_4/prompts/research.md`（N1 简报格式：系统描述符+矛盾状态列表+human 追问）
> 前序：E 收敛结论（judgment.md + 批判门禁-E-r5.md）—— 核心假设/价值论/架构定位最终形态已确立

## 前序已确认视角（N1 描述系统的基础）

- **核心假设**（A18 CONFIRMED）：∃G 外 G′（未见过+无法一次性解决）→ 需迭代/实验探索 → rick 存在。依赖链 A7（有损+非确定）+A5（G′存在）+A15（zero-shot 不选 rick 编排）+A18（单轮不足/多轮改善）全 CONFIRMED。
- **rick 价值论**（A17/D3′重锚）：价值=弥补参数记忆有损+非确定（LLM 内禀，与训练成本正交，刚性）；手段=应对上下文熵增；实现=确定性编排+强制执行+含判断的选择+迭代框架。
- **架构定位**（A16/D3）：rick=引导程序（引导人类正确模式[pi 不可内化]+引导 pi 加载系统提示词）；价值主体=rick；"可完全内嵌 pi"≠"功能不可替代"；rick=方法，pi=实现，遵循 sense 的 pi=rick。
- **S 阶段已确认**：现状=rick+ai_cli+claude code；期望=迁移 pi+深度定制（二进制/skill 系统级/自定义 compaction/subagent）；差距=缺具体实现计划。

## 信源配置

无 `.rick/config.json`，取默认权重。**N1 是基于已确认事实的综合描述任务**（非新事实验证），信源以"已确认事实的代码原文复用"为主：

| 信源 | 权重 | 本轮来源 |
|---|---|---|
| 代码原文 | 0.4 | 复用 E-r2/E-r4（context.go/doing_prompt.go/doing.md/doing_loop.md/think.md/RFC-001）+ loop_2 research-7-N1N2N3（rick 13 调用点+调用链+claude code 功能枚举）+ research-4-N2/N3（pi compaction/扩展点） |
| 运行时行为 | 0.3 | 复用 E-r2/E-r4 runtime（LLM 非确定采样/G' 拒答/zero-shot 线性单遍） |
| 文档 | 0.2 | 复用 E-r2/E-r4（Delétang/Tishby/RAG/Extended Mind/Self-Refine/Reflexion）+ loop_2 research-2/3（pi 文档） |
| 反事实 | 0.1 | 复用 E-r2 节点 C de facto A/B + E-r4 节点 C zero-shot vs rick 强制对照 |

**加权公式**：置信度 = Σ(信源验证结果 × 信源权重)。N1 的"事实"=已在前序 CONFIRMED 的系统组件/调用链/能力，置信度继承前序（多为 0.9-1.0）；本轮新增的是"系统描述综合"+"矛盾枚举"（矛盾供 human 选，不验证真假）。

---

## 系统论描述符（5 要素）

> 描述对象："rick+pi+LLM+human+外部存储" 这一解决 G′ 问题的系统。稳态 A=rick+ai_cli+claude code（现状）；稳态 B=rick+pi+深度定制（目标）。

### node（系统组件）

| node | 角色 | 关键属性 | 证据 |
|---|---|---|---|
| **human** | 任务发起者 + 判断者 | 提出 G′ problem；提供/校验判断；可被引导以正确模式工作 | doing_loop Step 5"人类明确要求停止"；judgment.md human 判断原话 |
| **rick** | 引导程序·方法 | 解析命令+调度任务+构建系统提示词启动 pi；确定性编排+强制执行（doing.md"不可跳过"）+含判断的选择（trigger 匹配+think 打分）；引导人类正确模式（pi 不可内化） | loop_2 research-7（13 调用点）+ E-r4 节点 C（doing.md/doing_loop.md/doing_prompt.go）+ A16 |
| **pi** | agent loop·实现 | 承载 LLM 推理循环；compaction（保留 system prompt）+ before_agent_start（每 turn 重建 system prompt）+ session_before_compact（自定义摘要）+ transformContext（每调用裁剪注入）+ subagent 递归扩展；可深度定制 | loop_2 research-2/3/4 + research-4-N2/N3 |
| **LLM** | 参数权重·模型 | 有损+非确定压缩（A7 CONFIRMED）；zero-shot 默认线性单遍不选 rick 编排（A15）；单轮对 G' 不足，多轮改善（A18） | E-r2 节点 A/B + E-r4 节点 B/D + Delétang/Tishby |
| **外部存储** | 确定性信息存储 | doing.md/sense_loop.md/OKR.md/SPEC.md/task.md/debug.md/skills/loops/domain——文件载体（可版本控制、可校验、compaction-resist 的 system prompt）；LLM 的"Otto 笔记本" | E-r2 节点 C（ContextManager+RFC-001）+ E-r4 节点 C（doing_loop Step 0 Domain 搜索）+ loop_2 research-4-N2（pi compaction 保留 system prompt） |

### input（系统输入）

| input | 来源 | 内容 |
|---|---|---|
| G′ problem | human | 未见过+无法一次性解决的任务需求（human E-r4："最核心假设在于存在 G 外的问题集合 G'"） |
| 上下文·系统提示词 | rick（从外部存储拼装）| doing.md 模板注入的 loops_context/skills_context/debug_context/task_info/doing_loop_content；sense_loop.md/think.md 模板 |
| 判断反馈 | human | judgment.md human 判断原话（Y-E1..Y-E5 + 决策点 D1-D3'） |

### output（系统输出）

| output | 去向 | 内容 |
|---|---|---|
| 解决 G′ 的产物 | human/仓库 | code / RFC / 决策（doing_loop Step 3 COMMIT：git add+commit） |
| 学习沉淀 | 外部存储 | learning 产出的 wiki/skills/tools/SPEC 更新（RFC-001 闭环）；loops 沉淀（sense_loop briefs/judgment） |
| 行为轨迹 | 外部存储 | debug.md/bug{N}.md（doing_loop Step 3 DEBUG Phase 1-6）；prompts/落盘（doing_prompt.go SaveToFile） |

### inner（内部协作 input/output）

| inner_input/output | 流向 | 机制 | 证据 |
|---|---|---|---|
| rick→pi（系统提示词注入）| rick 构建 prompt → pi 启动 | 现状 A：rick 把 system prompt 写入 prompt 文件，claude code 作 positional arg 读取（无 --system-prompt flag）；目标 B：pi before_agent_start 每 turn 重建 systemPromptOptions | research-7-N3（system prompt 注入机制）+ research-4-N3（before_agent_start） |
| pi→LLM（上下文·compaction）| pi agent loop → LLM | 现状 A：claude code 默认 auto-compact（rick 不控制）；目标 B：pi compaction（保留 system prompt + 自定义 summary + firstKeptEntryId）+ transformContext 每调用裁剪 | research-4-N2/N3 |
| rick→外部存储（确定性加载拼装）| rick ContextManager → 文件 | LoadOKRFromFile/LoadSPECFromFile/LoadDebugFromFile + LoadLoopsContext/LoadSkillsContext/loadDoingLoopContent + WriteSkillFile | E-r2 节点 C（context.go/doing_prompt.go） |
| LLM→外部存储（检索读写）| LLM 工具调用 → 文件 | claude code tool_use（Read/Write/Edit/Bash）；doing_loop Step 0.1 Domain 搜索（强制读 domain_dir） | research-7-N3 + E-r4 节点 C（doing_loop Step 0） |
| human→rick（命令·模式）| human CLI → rick | `rick plan/doing/learning/easy/dream/human-loop/ctrl`（13 调用点）；不同模式注入不同系统提示词（A16） | research-7-N1N2（13 调用点） |
| pi→human（交互·简报）| pi → human | 现状 A：claude code interactive（stdin/stdout/stderr 接管）；目标 B：pi 简报 + sense_loop 简报格式 | research-7-N2 + sense_loop.md 简报格式 |
| rick→human（门禁·check）| rick check → human | doing_check/easy_check/plan_check（runAutoFix）；批判门禁简报 | research-7-N1N2（#11/12/13 runAutoFix）+ 批判门禁 briefs |

### edge（node 间协作关系，承载 inner）

| edge | 承载的 inner | 稳态 A（现状）| 稳态 B（目标）| 控制手段 A→B |
|---|---|---|---|---|
| human↔rick | human→rick 命令；rick→human 门禁 | rick CLI（13 调用点） | rick CLI（不变，引导人类职责保留） | 引导人类不可内化（A16），edge 保留 |
| rick↔pi | rick→pi 系统提示词注入 | ai_cli=ClaudeCodeExecutor exec.Command(claude, -p, stream-json, promptFile)；prompt 文件内容注入 | pi agent loop + before_agent_start 程序化注入 systemPromptOptions | 用 pi before_agent_start/transformContext 替代 prompt 文件内容注入（动态+compaction-resist） |
| rick↔外部存储 | rick→存储 确定性加载拼装 | ContextManager + LoadLoopsContext/SkillsContext | 不变（rick 价值主体=确定性提取架构，A17） | edge 保留（rick 不可替代价值） |
| pi↔LLM | pi→LLM 上下文·compaction | claude code 默认 auto-compact（rick 不控制） | pi 自定义 compaction（session_before_compact + customInstructions + firstKeptEntryId）+ transformContext 分区 | 用 pi compaction 扩展点替代默认 auto-compact（保留 system prompt + 标记流程/方法不可压缩） |
| LLM↔外部存储 | LLM→存储 检索读写 | claude code tool_use（Read/Write/Edit/Bash） | pi tool_use + skill 系统级注册 | skill 从 prompt 文件路径引用 → pi skill 系统级注册（loop_2 research-3-N2） |
| pi↔human | pi→human 交互·简报 | claude code interactive（stdin/stdout/stderr） | pi 简报 + sense_loop 简报 | 用 pi 简报格式 + sense_loop N1/S 简报格式替代 claude interactive |

### ASCII 系统图（稳态 B 目标形态）

```
                       ┌─────────────────────────────────────────┐
                       │            human（判断者/发起者）          │
                       └──────┬──────────────────────────┬────────┘
                              │ 命令·模式                │ 门禁·简报
                              ▼                          │
   ┌─────────────────────────────────┐        ┌──────────┴──────────┐
   │   rick（引导程序·方法）           │        │  check/sense 简报     │
   │   - 解析命令+调度                 │        │  批判门禁             │
   │   - 确定性编排+强制执行(不可跳过)  │        └─────────────────────┘
   │   - 含判断的选择(trigger+think)   │
   │   - 引导人类正确模式[pi不可内化]    │
   └──────┬───────────────────────┬──┘
          │ 系统提示词注入         │ 确定性加载拼装
          │ (before_agent_start)   │ (ContextManager)
          ▼                        ▼
   ┌─────────────────────┐   ┌──────────────────────────────┐
   │  pi（agent loop·实现）│   │  外部存储（确定性信息存储）   │
   │  - compaction 保留    │◄──┤  doing.md/sense_loop.md      │
   │    system prompt     │   │  OKR/SPEC/task/debug         │
   │  - transformContext   │   │  skills/loops/domain         │
   │  - subagent 递归      │──►│  (文件载体=Otto 笔记本)       │
   └──────────┬──────────┘   └──────────┬───────────────────┘
              │ 上下文·compaction             │ 检索读写
              │ (每 turn 重建)                │ (tool_use)
              ▼                               │
   ┌─────────────────────┐                   │
   │  LLM（参数权重·模型） │◄──────────────────┘
   │  有损+非确定压缩(A7) │
   │  zero-shot 不选编排  │
   │  单轮不足/多轮改善   │
   └─────────────────────┘
```

**稳态 A（现状）关键差异**：rick↔pi edge = ai_cli(ClaudeCodeExecutor exec.Command claude -p stream-json promptFile)；prompt 文件内容注入（非 before_agent_start）；pi↔LLM edge = claude code 默认 auto-compact（rick 不控制）；无 subagent/skill 系统级/自定义 compaction。

---

## 稳态分析 A→B（控制手段）

| 维度 | 稳态 A（现状）| 稳态 B（目标）| 控制手段 |
|---|---|---|---|
| agent 实现 | ai_cli = ClaudeCodeExecutor（exec.Command claude 二进制） | pi agent loop（extension 机制） | 替换 internal/agent/claudecode → pi extension（before_agent_start/transformContext/session_before_compact） |
| 系统提示词注入 | prompt 文件内容（positional arg，无 --system-prompt flag） | pi systemPromptOptions + before_agent_start 程序化注入 | 从"文件内容注入"→"before_agent_start 每 turn 重建"（动态+compaction-resist） |
| compaction | claude code 默认 auto-compact（rick 不控制） | pi 自定义 compaction（session_before_compact + customInstructions + firstKeptEntryId） | 用 pi compaction 扩展点替代默认；标记"流程/方法"为不可压缩（system prompt 注入 / 自定义 summary 保留） |
| skill 加载 | prompt 文件路径引用（WriteSkillFile） | pi skill 系统级注册 | skill 系统级注册（提升触发概率，loop_2 research-3-N2） |
| subagent | 未使用（claude code subagent rick 不调用） | pi subagent 递归扩展（doing_loop Step 3 Main→Sub Agent per-iteration） | pi subagent 扩展（loop_2 research-3-N4） |
| 部署 | 依赖 node（claude code CLI） | 自包含二进制（pi 编译） | pi 二进制编译脱离 node（V0 价值） |
| rick 职责 | 命令解析+调度+系统提示词构建（不变） | 不变（引导程序双职责保留） | rick 轻量化但引导人类+确定性提取架构保留（A16/A17） |

**A→B 不变量**（rick 价值主体，edge 保留）：rick↔human（引导人类）、rick↔外部存储（确定性加载拼装）。**A→B 变量**（实现层）：rick↔pi（注入方式）、pi↔LLM（compaction）、pi↔human（简报）、LLM↔外部存储（skill 注册）。

---

## 矛盾状态列表（供 human 在 N2 选择主要矛盾）

> 矛盾按 MECE 维度枚举（架构层/能力层/过程层/价值层/人机层/身份层），每个矛盾=系统演化中两股拉扯力量的状态对。**不替 human 选择**，仅列举供 N2 判断。

### M1（架构层）：rick 轻量化（仅引导程序）vs 门禁/做事方法内嵌 pi 深度定制
- **左极**：rick 极薄——仅命令解析+系统提示词注入，所有做事方法/门禁内嵌 pi extension（doing_loop/sense/think 作 pi skill）
- **右极**：rick 持续承载做事方法+门禁（doing.md/doing_loop.md/sense_loop.md 作 rick 模板，pi 仅作执行器）
- **拉扯**：A16"引导人类 pi 不可内化"（右极支撑）vs A10"独立存在=工程便利/可内嵌 pi"（左极支撑）；D3"可完全内嵌 pi≠功能不可替代"居中
- **后果**：左极→rick 退化为命名壳（A8m）；右极→rick 与 pi 边界固化，定制成本高

### M2（架构层）：rick=方法（可完全内嵌 pi）vs rick=独立引导程序（引导人类，pi 不可内化）
- **左极**：rick=sense 方法，"遵循 sense 的 pi 就是 rick"（YE5 原话），可完全内嵌 pi（pi rick-plan 命令）
- **右极**：rick=独立引导程序，引导人类正确模式（pi 是 agent loop 不面向人类工作流，不可内化）
- **拉扯**：YE5"一个是方法一个是实现"（左极）vs A16"引导人类 pi 不可内化"（右极）；D3 双职责居中
- **后果**：左极→rick 进程消失，降为 pi 命名前缀；右极→rick 进程必需，但与"可内嵌"张力

### M3（能力层）：确定性提取/强制执行（rick）vs LLM 参数记忆有损+非确定（内禀）
- **左极**：rick 用文件载体+强制执行实现确定性提取（ContextManager+doing.md"不可跳过"+compaction-resist system prompt）
- **右极**：LLM 参数权重有损+非确定是内禀属性（A7 CONFIRMED），不可消除，每次提取必有损/方差
- **拉扯**：rick 的确定性（外部存储+强制）vs LLM 的非确定（权重采样）；两者共存——rick 弥补 LLM 的非确定，但不消除
- **后果**：这是 rick 价值的存在理由（A17：弥补有损+非确定），非"解决"而是"弥补"——矛盾是结构性的，永久存在

### M4（过程层）：迭代探索（解决 G′）vs 单次交付期望
- **左极**：G′ 需迭代/实验探索（doing_loop Step 3-5：Sub Agent per-iteration + 3 轮上限 + DEBUG + 停止标准；sense 多阶段）
- **右极**：human/用户对"快速单次交付"的期望（单轮直接给答案）
- **拉扯**：A18"单轮不足，多轮改善"（左极）vs 用户体验/效率期望（右极）；doing_loop 3 轮上限是妥协点
- **后果**：迭代过深→交付慢/成本高；迭代不足→G′ 解决不可靠；3 轮上限是当前妥协，但是否最优？

### M5（价值层）：训练成本高（G 过去式刚性）vs 价值重锚（与训练成本正交）
- **左极**：rick 价值前提"训练成本高→rick 必需"（权重级训练不可实时，A4 边界刚性）
- **右极**：D3′ 重锚"rick 价值=弥补有损+非确定，与训练成本正交"（rick 操作上下文级，实时可行）
- **拉扯**：A4"训练可降至实时则 rick 不再需要"（左极）vs A17"有损+非确定是内禀，与训练成本正交"（右极）；E-r4 节点 D 化解（权重级不可实时，但 rick 在上下文级）
- **后果**：左极→rick 价值依赖"训练贵"为前提（脆弱）；右极→rick 价值稳固（内禀），但需重述价值主张

### M6（人机层）：rick 约束人类正确模式 vs 人类自由度
- **左极**：rick 引导人类以 sense/doing 正确模式工作（A16：引导人类；doing.md"不可跳过"强制）
- **右极**：human 自由度——熟练用户可能不需引导，自发遵循 plan-do-learn（A16 Q3 反例：熟练用户非必需）
- **拉扯**：rick 的强制/约束（左极）vs 用户体验自由/熟练用户便利（右极）；引导人类是 rick 不可内化价值，但对熟练用户是便利非必需
- **后果**：左极→约束性强，新手友好但熟练用户受限；右极→自由度高，但 zero-shot 默认不选 rick 编排（A15），失去确定性

### M7（身份层）："可完全内嵌 pi" vs "功能不可替代"
- **左极**：rick 功能可完全内嵌 pi（pi 调用 rick 的提示词/文件编排层；pi rick-plan 命令）
- **右极**：rick 功能不可替代（引导人类+确定性提取架构，pi 原生没有）
- **拉扯**：D3"可完全内嵌 pi≠功能不可替代"——技术上可内嵌，但价值上不可替代（A16+A17）
- **后果**：左极→rick 进程可消失；右极→rick 价值保留但形式可变；这是"形式 vs 价值"的张力

### M8（过程层·补充）：迭代策略本身有优劣（rick sense+doing loop vs 重复采样/反思）
- **左极**：rick 的迭代框架（sense S-E-N-S-R-EC + doing loop 3 轮上限 + DEBUG Phase 1-6）是确定性编排
- **右极**：research 节点 D 边界 nuance——"Sample More, Reflect Less"示同等 token 成本下重复采样可超 self-refine/Reflexion；多轮策略有优劣
- **拉扯**：A18"需迭代→rick 存在"（方向性）vs"rick 迭代框架是否最优"（策略性，A18-Q2 未验证）
- **后果**：若 rick 迭代框架非最优，rick 存在理由从"迭代框架"收缩到"确定性选择+确定性提取"（A15+A7）

> **MECE 自检**：M1/M2/M7 覆盖架构层（rick-pi 边界/形式）；M3 覆盖能力层（确定性 vs 非确定）；M4/M8 覆盖过程层（迭代 vs 单次 / 迭代策略优劣）；M5 覆盖价值层（训练成本 vs 重锚）；M6 覆盖人机层（约束 vs 自由）。6 维度 8 矛盾，完备+互斥。

---

## 尽调树快照（N1 综合）

```
根：N1 — rick+pi+LLM+human+外部存储 系统描述+矛盾枚举
├─ 系统描述（5 要素）                  [1.00 高] 继承前序 CONFIRMED 事实
│   ├─ node（5 组件）✅ 代码 0.4+runtime 0.3+文档 0.2+反事实 0.1
│   ├─ input/output ✅（同上）
│   ├─ inner（7 内部协作）✅ research-7 调用链+E-r2/E-r4 代码
│   └─ edge（6 边+稳态 A→B 控制手段）✅ research-7-N1N2N3+E-r4 节点 C
├─ 稳态分析 A→B ✅ [1.00 高]（loop_2 research-3/4/7 迁移价值+扩展点已尽调）
└─ 矛盾状态枚举（M1-M8）              [N/A]  供 human 选，不验证真假
    └─ 8 矛盾按 MECE 6 维度（架构/能力/过程/价值/人机/身份）
```

## 节点详情（置信度继承前序）

### 系统 5 要素描述 — 置信度 1.00（高）
- 代码 ✅ 0.4（E-r2/E-r4 + loop_2 research-7/4，多源交叉）
- 运行时 ✅ 0.3（E-r2/E-r4 runtime 印证）
- 文档 ✅ 0.2（E-r2/E-r4 + loop_2 research-2/3）
- 反事实 ✅ 0.1（E-r2 节点 C de facto A/B + E-r4 节点 C 对照）
- 合计 1.0（高）—— 5 要素均基于前序 CONFIRMED 事实，本轮综合不引入新未决事实

### 矛盾状态枚举 — 置信度 N/A
- 矛盾是"供 human 在 N2 选择"的候选，不验证真假
- 每个矛盾的两极均有前序假设支撑（A4/A7/A10/A15/A16/A17/A18 + D3/D3'）
- MECE 自检通过（6 维度 8 矛盾，完备+互斥）

## R7 上报项

- **无 R7**。N1 的系统描述基于前序 CONFIRMED 事实（置信度继承 0.9-1.0）；矛盾枚举不验证真假（供 human 选）。无叶节点置信度 < 高且无疑问点的情况。

## 整合摘要

- **总节点数**：3（系统描述/稳态分析/矛盾枚举）+ 1 根 = 4
- **高置信度叶节点**：2（系统描述 1.0 / 稳态分析 1.0）
- **R7 上报**：0
- **矛盾枚举**：8 个（M1-M8），按 MECE 6 维度，供 human 在 N2 选主要矛盾

---

## human 启发性追问（照 sense_loop N1 格式）

### 1. 在这个系统中，你看到哪两股力量在拉扯？
（提示：M1-M8 中哪对矛盾是你感到最强烈的拉扯？是架构层的"rick 轻量化 vs 深度定制"（M1/M2），还是能力层的"确定性 vs 非确定"（M3），还是过程层的"迭代 vs 单次交付"（M4）？或你认为有未列举的更根本矛盾？）

### 2. 如果系统继续按现状（稳态 A：rick+ai_cli+claude code）运行，3 年后会发生什么？
（提示：稳态 A 的限制——claude code 默认 auto-compact 不可控、无 subagent 递归、skill 非 system 级、依赖 node——持续 3 年会导致什么？是 rick 被 claude code 的演进吞并（A8m 命名壳），还是 rick 因确定性提取价值（A17）保留但形式停滞？或 LLM 演进（确定性解码/无损权重）使 M3 的"非确定"右极削弱？）

### 3. 系统的哪个节点，如果消失，整个系统会重组？
（提示：若 rick 消失 → 确定性提取+引导人类消失，系统降为 pi+LLM+存储的 zero-shot 模式（A15 不选编排）；若 pi 消失 → rick 退回 ai_cli+claude code（稳态 A 停滞）；若 LLM 消失 → 整个系统无意义（解决 G' 的智能引擎没了）；若外部存储消失 → LLM 退回纯参数记忆（A7 有损无弥补）；若 human 消失 → 系统无任务源+无判断者。哪个节点的消失最致命？）

---

## 安全约束确认

- 无 rick 代码修改 → 无需 `git restore` 还原
- 所有 Read/Grep/curl 只读
- 复用 loop_2/E-r2/E-r4 brief 已标注来源
- 矛盾枚举不含倾向性（不推荐某矛盾为主要，等 human 在 N2 选）

## 信源清单（复用，本轮无新抓取）

- rick 代码（`/workdir/sunquan20/AI_CODING/rick`）：context.go/doing_prompt.go/doing.md/doing_loop.md/think.md/RFC-001 + internal/agent/claudecode（research-7-N1N2N3）
- loop_2 briefs：research-7-N1N2N3（13 调用点+调用链）/ research-4-N2/N3（pi compaction/扩展点）/ research-3-N2/N4（skill 系统级/subagent）/ research-2-N2（pi 扩展点机制）
- E-r2/E-r4 briefs：节点 A/B/C/D（LLM 有损/非确定/确定性提取/zero-shot/单轮 vs 多轮）
- 文档：Delétang/Tishby/RAG/Extended Mind/Self-Refine/Reflexion（E-r2/E-r4 已抓取）
