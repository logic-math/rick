# 调研报告 — 协议对齐视角下的 rick+pi subagent 触发系统（系统论描述符） — N1 矛盾生成

日期：2026-08-13
阶段：N1（矛盾生成）· research subagent 输出

---

## 信源配置

| 信源 | 权重 | 验证方式 | 本次使用 |
|---|---|---|---|
| 代码原文 | 0.4 | Read/Grep 直接读取 | ✅（rick 模板 + human_loop_prompt.go；引用 S/E 已证实的 pi-subagents 源码） |
| 运行时行为 | 0.3 | Bash 跑命令 | ⚠️ 部分（grep 计数复核；未做 live spawn） |
| 文档 | 0.2 | Read 官方文档 | ✅（引用 S 阶段已核验的 pi-subagents SKILL/docs） |
| 反事实 | 0.1 | 修改后还原 | ❌ 未使用（N1 只读） |

置信度 = Σ(信源验证结果 × 权重)。高 ≥ 0.8 | 中 0.5-0.8 | 低 < 0.5（R7）。「文件包含/不包含某文字」由直接 Grep 判定为确定（高）。

---

## 尽调树（快照）

```
根：协议对齐视角下的 rick+pi subagent 触发系统描述（N1）
├─ F 事实基础层（引用 S/E 已证实 + 本次复核）
│   ├─ F1 rick 模板自然语言触发 243 处 / pi 触发语法 0 处 / 内置 agent 名 0 处 ✅高
│   ├─ F2 pi 最佳实践 BP-1~BP-9 ✅高
│   ├─ F3 pi 自定义 agent 机制 B1~B4 ✅高
│   └─ F4 pi 配置层已排除（扩展已注册、工具可用、无禁用） ✅高
├─ G 系统要素层（5 要素识别，由 F 推导）
│   ├─ G1 node（6 类组件） ✅高
│   ├─ G2 input / output ✅高
│   ├─ G3 inner（内部协作 input/output） ✅高
│   └─ G4 edge（协议不对齐定位在 E2/E3） ✅高
├─ H 稳态分析层（A→B 控制手段，由 F 推导） ✅高
└─ K 矛盾状态层（候选矛盾，供 human 选，构建非可证伪）
    └─ K1~K7（7 个矛盾状态）
```

树规模：深度 3 | 总节点 18 ≤ 30 ✅

---

## F 事实基础层（本次复核 + 引用 S/E 已证实）

### F1：rick 模板的触发语言（本次 grep 复核）

- **自然语言 subagent 术语 243 处**（root 模板 134 处 + `skills/` 子目录 109 处）：「派发 subagent」「每个 subagent 独立启动」「SPAWN Sub Agent」「Main Agent / Sub Agent」「父 Agent / 子 Agent」。
- **pi 触发语法 0 处**：`workflowScript` / `runs.run` / `runs.all` / `agent:'…'` 零出现（延续 N3.1）。
- **内置 agent 名 0 处**：scout/worker/reviewer/researcher/delegate/oracle 零出现（延续 N3.2）。
- **触发要求分布（结构性事实，供矛盾 K6 参考）**：
  - 直接写死触发：`sense_loop.md`（26 处，think/research/exporter）、`plan.md`（8 处，subagent_1~6 六维评审）、`dream.md`（10 处，subagent_1~4 质量验证）。
  - 经 skill 注入触发：`doing.md`/`easy.md` 注入 `doing_loop`（SPAWN Sub Agent、Main/Sub Agent）；`learning.md` 注入 `learning_loop`（父 Agent/子 Agent）。
  - 无触发要求：`ctrl.md`（grep 0 命中）。
- **置信度**：高（直接 grep 判定）。

### F2：pi 最佳实践 BP-1~BP-9（引用 research-report-S-bestpractice.md，高置信）

触发入口唯一且显式（BP-1）、agent 名显式（BP-2）、任务=compact contract（BP-3）、async 默认（BP-4）、context 明确（BP-5）、单写者（BP-6）、编排 recipe（BP-7）、编排权在 parent（BP-8）、模型认知来源=tool description（BP-9）。

### F3：pi 自定义 agent 机制 B1~B4（引用 research-report-S-reasons-agent.md，高置信）

自定义 agent = frontmatter markdown（B1）；注册作用域 builtin/package/user/project（B2）；触发用 `runs.run(agent:'name')`（B3）；rick 现状零自定义 agent、think/research/exporter 仅为普通 markdown 文件（B4）。

### F4：pi 配置层已排除（引用 research-report-S-reasons-agent.md A2，高置信）

扩展已注册、subagent 工具可用、tool description 默认 full、无任何禁用项 → 配置层不是触发概率低的因素。

---

## G 系统要素层（系统论描述符，基于协议对齐视角）

### G1：node（系统组件，6 类）

| node | 说明（协议对齐视角） |
|---|---|
| **human** | 发出 rick 命令，做 sense 各阶段判断（决策权）。 |
| **rick（命令+提示词+方法层·引导程序）** | 生成/维护提示词模板（sense_loop/plan/dream/doing_loop/learning_loop），定义 think/research/exporter 等角色 prompt。当前用**自然语言**描述 subagent 触发（F1）。 |
| **main agent（pi 下 LLM 会话）** | 上下文 = rick 提示词 + pi system prompt + tool description。是「是否/如何触发 subagent」的实际决策与执行者。 |
| **pi-subagents 运行时** | subagent 工具 + tool description（触发语法/内置 agent 名）+ agent 注册表。唯一执行面 = workflowScript（BP-1）。 |
| **子 agent** | think/research/exporter 或 pi 内置/自定义 agent。被触发后执行调研/思考/输出。 |
| **外部存储** | rick 模板文件、agent 定义 markdown、pi 配置 settings.json（确定性信息存储）。 |

### G2：input / output

- **input**：
  - human 的任务需求/命令（human-loop / plan / doing / easy / learning / dream / ctrl）。
  - pi 协议规范（tool description 定义的触发语法 + 内置 agent 名，BP-9）。
- **output**：
  - 子 agent 被触发并产出结果（research 报告 / think 打分 / RFC）。
  - rick 命令交付物（RFC / plan / doing 产物 / learning 沉淀）。

### G3：inner（系统内部协作的 input/output）

| inner | 方向 | 内容 | 协议状态 |
|---|---|---|---|
| i1 提示词注入 | rick → main agent | 自然语言 subagent 触发指令（243 处，零语法/零 agent 名） | **不对齐** |
| i2 协议认知 | tool description → main agent | 触发语法 + 内置 agent 名（唯一执行面 workflowScript） | 正常（但未在 rick 提示词中显式引用） |
| i3 工具调用 | main agent → subagent 工具 | workflowScript + runs.run（若触发） | **当前概率低** |
| i4 符号表 | agent 定义 → 运行时 | 内置 7 名；rick 零自定义 agent（B4） | **缺失（自造名无映射）** |
| i5 结果回流 | 子 agent → main agent | 调研/打分/产出摘要 | 正常 |

### G4：edge（协作关系，承载 inner）

| edge | 承载 | 协议不对齐？ |
|---|---|---|
| E1 human ↔ rick | 命令 → 提示词生成 | 否 |
| **E2 rick ↔ main agent** | i1 提示词注入（软触发） | **★ 不对齐**：rick 用自然语言，main agent 需要的是 pi 触发语法 |
| **E3 main agent ↔ pi-subagents 运行时** | i3 工具调用（需 workflowScript + 真实 agent 名） | **★ 不对齐**：无语法/无 agent 名 → 触发不确定 |
| E4 运行时 ↔ 子 agent | i4 agent 启动（符号表解析，名字不在表 → "Unknown agent"） | 是（自造名未注册，D2） |
| E5 子 agent ↔ main agent | i5 结果回流 | 否 |
| E6 rick/pi ↔ 外部存储 | 模板读写 / agent 定义 / 配置 | 部分（rick 未向 agent 发现目录写 agent 定义，B4） |

**协议不对齐定位**：主要发生在 **E2**（提示词注入边）与 **E3**（工具调用边）——rick 的「语言体系」与 pi 的「语言体系」在边界处不共享协议（对应 human E 阶段「两个各自自洽、合作时暴露问题的语言体系」类比）。

### ASCII 系统图

```
         ┌───────────┐
         │   human   │  input: 任务需求/命令 + sense 判断
         └─────┬─────┘
               │ 命令 (E1)
   ┌───────────▼───────────────────────────────┐
   │ rick（命令+提示词模板+方法层）               │
   │  sense_loop / plan / dream / doing_loop /  │
   │  learning_loop / ctrl                      │
   │  · 自然语言 subagent 触发 243 处            │
   │  · pi 触发语法 0 处 · 内置 agent 名 0 处     │
   └───────────┬───────────────────────────────┘
               │ E2 提示词注入（软触发）★协议不对齐
   ┌───────────▼───────────────────────────────┐
   │ main agent（pi 下 LLM 会话）                │
   │  上下文 = rick 提示词 + pi system prompt    │
   │         + tool description（i2 协议认知）   │
   └───────────┬───────────────────────────────┘
               │ E3 subagent 工具调用 ★协议不对齐
               │   （唯一执行面 workflowScript，
               │    需真实 agent 名；无 → 触发概率低）
   ┌───────────▼───────────────────────────────┐
   │ pi-subagents 运行时                        │
   │  · subagent 工具 + tool description        │
   │  · agent 注册表（内置 7 名）                │
   │  · rick 零自定义 agent（B4）                │
   └───────────┬───────────────────────────────┘
               │ E4 agent 启动（符号表解析，需真实名，否则 "Unknown agent"）
   ┌───────────▼───────────────────────────────┐
   │ 子 agent（think/research/exporter          │
   │  或内置 scout/worker/reviewer/…）           │
   └───────────┬───────────────────────────────┘
               │ E5 结果回流
               ▼
         main agent ──► human（展示简报）

   外部存储（E6 读写边）：rick 模板 / agent 定义 markdown / pi settings.json
```

---

## H 稳态分析（当前稳态 A → 目标稳态 B）

### 当前稳态 A（协议未对齐）

- rick 提示词用自然语言软触发（243 处），零 pi 触发语法、零内置 agent 名（F1）。
- main agent 收到的触发指令（i1）与 pi 的 subagent 工具协议（i2 tool description）**不对齐**。
- agent 注册表无 rick 自造名（B4），即使 main agent 尝试触发，自造名也无法解析（"Unknown agent"，BP-2）。
- **结果**：main agent 触发 subagent 的概率低（human 观察，S 阶段已确认），系统稳定停留在「协议未对齐、触发不确定」状态。

### 目标稳态 B（协议对齐）

- rick 提示词按 pi 协议改造：显式触发语法（workflowScript + runs.run）+ 真实 agent 名。
- 把 think/research/exporter（及 plan/dream 的子 agent 角色）注册为 pi 自定义 agent（frontmatter markdown 落盘到 agent 发现目录）。
- 触发权归属明确（parent 触发，子 agent 不递归触发，BP-8）。
- **结果**：main agent 收到与 tool description 对齐的触发指令，能确定性调用 subagent 工具，触发确定性提升到上限内最高（残余上限 = 模型 tool-calling 能力，R7 待实测）。

### A→B 所需控制手段（协议对齐视角，事实性枚举，非建议排序）

1. **协议对齐改造**：rick 模板中的自然语言触发词 → 显式 pi 触发语法 + 真实 agent 名（消除 E2/E3 边的不对齐）。
2. **系统级 agent 注册**：把自造角色名注册为 pi 自定义 agent（建立符号表项，消除 E4 边的 "Unknown agent"）。
3. **触发权归属约束**：提示词明确 parent（main agent）是唯一触发者，子 agent 不递归触发（对应 BP-8/D6）。
4. **任务形态改造**：长 procedural 脚本 → compact contract（对应 D3，降低长会话 summarize 丢失 subagent 段的风险）。
5. **实测验证**：改正后实测触发率，验证模型能力残余上限（R7 项，human 已明确承接）。

---

## K 矛盾状态层（候选矛盾，供 human 在 N2 选择主要矛盾）

> 每个矛盾 = 两股相反力量的拉扯，均基于已证实事实（F/G/H）。**不替 human 选择**。

| # | 矛盾状态 | 两股力量 |
|---|---|---|
| **K1 触发语言** | rick 自然语言软触发（自洽但不确定，表达方法论意图）vs pi 显式强制触发语法（确定但需改造提示词） | F1 vs BP-1 |
| **K2 角色命名** | rick 自造角色名 think/research/exporter/subagent_1~6/父·子 Agent（表达 rick 语义分工）vs pi 真实 agent 名（协议约束，需注册/映射） | N3.2 vs BP-2/B1~B4 |
| **K3 框架独立性** | rick 方法层独立性（引导程序 / sense 方法论自成体系）vs pi 协议要求（必须按 pi 触发语法/agent 注册改造，长期跟随 pi） | human E 动作 vs rick 自身方法层 |
| **K4 改造深度** | 高度定制化改造（长期跟随 pi 语言体系）vs rick 框架稳定性/独立性（保持方法层可迁移性） | human E 判断 vs 框架演化风险 |
| **K5 确定性 vs 能力上限** | 协议对齐可解决触发确定性（可控制）vs 模型 tool-calling 能力上限（不可控残余，R7 待实测） | H 控制手段 1~4 vs A1/R7 |
| **K6 统一 vs 特殊** | 统一协议对齐改造覆盖各命令 vs 各命令特殊场景（ctrl 无 subagent 要求；dream 四子 agent 轮询；doing/learning 经 skill 注入；human-loop 三 subagent 会话级持久文件） | human 确认"同一根因" vs F1 触发分布差异 |
| **K7 触发权归属** | 编排权集中 parent（单一触发点，BP-8）vs 提示词自然语言下"谁触发"模糊（模型可能让 parent 或子 agent 各自理解） | BP-8 vs D6 |

---

## R7 上报项（无法达高置信的叶节点）

1. **模型 tool-calling 能力差异的量化证据**（A1）——本地无 benchmark 信源，无法验证，需 WebSearch/实测。
2. **「触发概率低」的量化/复现证据**（N3.3）——human 观察，只读阶段无法复现真实 pi 会话触发率。
3. **「提示词缺口 ⇒ 触发概率低」的因果归属**（N3.4）——结构性缺口已证实，因果归属属假设，待实测收敛。
4. **「各命令特殊场景的触发失败点是否完全一致」**（对应 K6）——只能通过改正后逐命令实测验证，当前无法高置信。

---

## 整合摘要

事实基础 4 项（F1~F4，高置信）| 系统要素 5 项（node/input/output/inner/edge，由事实推导）| 稳态 A→B 控制手段 5 项 | 矛盾状态 7 个（K1~K7）| R7 上报 4 项。

**关键提示（事实性，非建议）**：协议不对齐集中定位在 **E2（提示词注入边）与 E3（工具调用边）**，即 rick 语言体系与 pi 语言体系的交界处——与 human E 阶段「两个各自自洽、合作时暴露问题的语言体系」判断一致；7 个矛盾状态是该不对齐在不同切面上的拉扯。

---

## human 启发性追问（照 sense_loop N1 格式）

① 在这个系统中，你看到哪两股力量在拉扯？
② 如果系统继续按现状运行，3 年后会发生什么？
③ 系统的哪个节点，如果消失，整个系统会重组？
