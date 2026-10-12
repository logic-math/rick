# task5：SENSE 范式重构（human-loop v6）——sense canon + 分工执行 + 教学单元写作要求 + 动态大纲 + 考核终止

## 任务目标

把 `sense.md` 从「五阶段方法论」整体重写为**思考范式 canon（全局统一：human-loop 的 research/teach + dream 共用）**，并让 human-loop 按新范式运转：AI 用 SENSE 完成一次思考产出教学大纲；教学单元以**写作要求**（非固定模板）还原原创作者的 SENSE 过程并把学生自己的判断逼出来；大纲支持动态变更；以**考核演练**（human 当老师、teach 当教练）收敛终止。

**权威文本**：`sense.md` 的内容以 human 原话为准（下方「五步权威文本」逐字写入 skill，允许补「如何判定」的可操作定义，禁止二次抽象/改写其要义）。

## 关键结果

- **KR1 `skills/sense.md` 整体重写**（125 行旧文全删，含系统论描述符/三维打分/阶段/回流）：写入五步范式 + 明示「**这是思考范式，不是流程；内容呈现形式自由，只要回答这些问题**」+ 三个可操作定义（见「范式补充定义」）。
- **KR2a research 接第 1 步**（`templates/research.md` + `skills/research.md`）：任务模式 `facts`——**递归**调研所有相关事实，**直到所有事实都来自常识性置信信息源**（终止三层定义见下）；递归中**识别所有价值性信息**并落盘 `{{values_path}}`（每条：原文/出处 + 为什么是价值判断 + 待 human 裁决）；无法收敛者入 `## Gaps`。保留尽调树 + 信源等级 A/B/C 作为**方法**。
- **KR2b teach 接第 2-5 步**（`templates/teach.md` + `skills/teach.md`）：任务模式 `syllabus`——消费 research 事实 + human 价值裁决（`{{values_path}}`），执行第 2/3/4/5 步（构建视角→洞察矛盾→辩证逆转→良质批判），产出 `{{syllabus_path}}` = **一次完整 SENSE 思考的留痕**（五步各一段：知识领域强/弱相关、主要矛盾多备选、逆转多方案、良质批判三问）+ 教学序列（模块序：先学什么、再学什么、每块为什么在这个位置）。**矛盾与逆转必须给多个备选供 human 判断**。
- **KR3 教学单元（`lesson` 模式）= 写作要求（非模板）**：正文必须回答三类问题（形式自由、不设固定小节、大纲随内容变化）：
  1. **还原原创作者的 SENSE 过程**：他的原始问题是什么 → 当时做了哪些事实调研 → 持有哪些价值判断 → 调研了哪些知识领域 → 最终构建了何种视角 → 怎么定位到主要矛盾 → 如何完成辩证逆转；
  2. **教学良质自检**（teach 自反思）：是否结合了学生自身的知识体系？有没有无法理解的信息？是否遵循教学良质？（**依据仅限 mission 起点 + judgment 已表达判断，禁止凭空假设学生已知什么**；允许以脚注/边注等轻形式呈现，勿打断叙事）；
  3. **追问回到学生自己**：启发 human 把知识回归应用在自己的问题中，构建**自己的**事实判断/价值判断/视角/矛盾/逆转方法。
  保留 v5 已有：正文全文入会话（最终回复=正文）+ 落盘 `lessons/teach_{N}.md` + 引注事实 + primary source。
- **KR4a `syllabus-update` 模式**：human 下钻/扩展知识点时，teach 调整大纲（**允许插入/重排**），在 `{{syllabus_path}}` 末尾留 **changelog**（含触发它的 human 提问原话）；parent 呈现变更片段 → human 确认 → 再教学。
- **KR4b 考核演练（`assess` 模式，单一形式）**：**human 当老师、teach 当学生**——human 讲出自己的 SENSE 思考，teach 作为**教练**指出其中的问题环节（哪一步缺证据/哪一步价值未裁决/哪一步矛盾未定/逆转是否成立/假设是否够置信）。通过 → 终止（收尾不产 RFC）。
- **KR5 冲突清理与兼容**：`internal/builder/pibuilder.go` 的 human-loop SOP 文本（现为「SENSE 方法（5 阶段）…允许反向回流」）改为描述新范式；dream 继续共用 `skills/sense.md`（`{{sense_skill_path}}`），需保证 dream prompt 渲染与测试不破。
- **KR6 变量与版本**：`human_loop_prompt.go` 注入 `values_path`（`{{loop_dir}}/values.md`）与 `sense_skill_path`（真正被 teach/research 的 prompt 引用，消除 v4/v5 死写入）；`cmd/rick/main.go` VERSION → **5.0.15**。

## 五步权威文本（逐字写入 `skills/sense.md`，作为唯一权威）

> **第一步 事实还原**：递归的调研所有涉及到的事实信息，直到所有事实信息都来自于常识性的置信信息源。递归过程中识别所有的价值性信息，都必须上报给 human，等待 human 做出价值判断。
> **第二步 构建视角**：从第一性原理出发讨论当前涉及到的事实所属及相关的知识领域，将这些知识领域作为解释问题现象的思考框架，分别给出强相关与弱相关的知识领域，强相关则是为了能够高效的解决问题，弱相关则是为了探索创造性的解决方案，强相关是收敛性思维，弱相关则是发散性思维。
> **第三步 洞察矛盾**：选择思考框架后，使用本知识领域的术语来解构问题的成因；结合当前的具体问题，确定造成问题的演化过程中，哪种要素是全局性、根本性、决定性的成因。可以给出多个备选，供 human 做出判断。
> **第四步 辩证逆转**：当确认主要矛盾及矛盾的主要方面后，例如两个方面 X 和 Y，其中 X 是阻碍而 Y 是期望状态，那么就要引导思考：当 X 是 Y 必然成立的前提时，应当如何做才能实现 Y？将阻碍变成达成目标的手段，这就是辩证逆转，给出多个可行的解决方案供 human 选择判断。
> **第五步 良质批判**：反思整个思考过程，使思考有效的各种假设在当前的具体问题上，是否足够置信？能否由整个思考过程必然得出目标可达成的结论？如果不能那差在哪？如果可以是否有被忽略的关键假设？

## 范式补充定义（可操作化，仅补"如何判定"，不改要义）

1. **递归终止三层**：①来源收敛到 primary source / 经典教科书 / 规范 / 权威综述；②每一步都能回答「我从哪知道的？再往上呢？」直到**答案不再需要引用**（常识层）；③**回溯不到常识层的条目一律不视为事实**，标记为价值/假设并转入 `values.md` 上报。不设深度硬上限——**收益递减即停**（继续递归不再改变结论）；无法收敛显式入 `## Gaps`。
2. **价值信息上报**：阶段末**批量**呈报（非逐条打断）；human 逐条裁决写回 `values.md`；**裁决完成前不进入第 2 步**。
3. **范式声明**：SENSE 是思考范式，**不是流程清单**；模型在此范式下自由发挥（内容形式、顺序、篇幅自由，只要回答范式要求的这些问题）。

## 写域

- internal/prompt/templates/skills/sense.md（整体重写）
- internal/prompt/templates/skills/research.md
- internal/prompt/templates/research.md
- internal/prompt/templates/teach.md
- internal/prompt/templates/skills/teach.md
- internal/prompt/templates/sense_loop.md
- internal/prompt/human_loop_prompt.go
- internal/prompt/human_loop_prompt_test.go
- internal/builder/pibuilder.go（仅 human-loop SOP 文本）
- cmd/rick/main.go（仅 VERSION 一行）
- internal/env/agents.go（**仅一行**：teach agent description 与新范式对齐——parent 补记的范围例外，task5 实现时遗漏于写域，2026-10-12）

# 依赖关系

- task4（前置层；本任务为 job_44 后续层）

## 实现要点

- **`sense_loop.md`（parent）状态机改造**：新增三个分支——①`values.md` 缺失或未裁决 → **价值裁决回合**（把 research 的价值清单原文呈现 → 请 human 逐条判断 → 裁决写回 → 才继续）；②`syllabus.md` 缺失 → 派 research(`facts`) → 派 teach(`syllabus`) → 原文呈现 → human 确认（可修订，含请求 teach 改）→ 落盘；③**大纲动态变更回合**（human 下钻/扩展 → 派 teach(`syllabus-update`) → 呈现变更 + changelog → human 确认）；④终止回合：派 teach(`assess`)（human 讲、teach 当学生+教练反馈）→ 通过则收尾。
- **呈现层保留 v5**：teach 正文原文作为 parent 回复（不改写/不摘要）。
- **`sense_skill_path` 真正被引用**：teach/research 的 prompt 里加「**先读** `{{sense_skill_path}}`（SENSE 思考范式 canon）」，并在 `human_loop_prompt.go` 注入该变量（现在 skill_sense.md 被写入但无人引用 = 死写入）。
- **dream 兼容**：`dream.md` 已引用 `{{sense_skill_path}}`（dream_prompt.go 注入），本次不改 dream 侧代码；仅确保 `skills/sense.md` 重写后 dream 的「读取后深度反思」仍成立（新 canon 即反思范式）。
- **不做**：不新增 agent；不改 humans 页面/草稿持久化；不改 web 前端。

## 测试方法

- `go build ./...` 绿
- `go test ./internal/prompt/ ./internal/env/ ./internal/builder/ ./internal/handler/` 绿（含新变量注入断言）
- 渲染冒烟：`go build -o /tmp/rick-v6 ./cmd/rick && tmp=$(mktemp -d) && mkdir -p $tmp/.rick && (cd $tmp && HOME=$tmp /tmp/rick-v6 human-loop --dry-run '自检主题')`：
  - 含 `values.md`、`syllabus.md`、`sense_skill_path`/`skill_sense.md`
  - 含「事实还原」「构建视角」「洞察矛盾」「辩证逆转」「良质批判」五步名
  - 不含「反向回流」「系统论描述符」「三维打分」「node/input/output/inner/edge」
- `skills/sense.md` 断言：含五步权威文本关键语义（递归至常识性置信信息源 / 价值性信息上报 / 强相关与弱相关 / 收敛性思维与发散性思维 / 全局性/根本性/决定性 / 阻碍变手段 / 假设置信与必然性）；**不含**旧阶段结构（`## S —`/`## E —`/`N1`）与旧方法论；含「范式」声明与不固定形式表述

## 禁止事项

- 不得把 SENSE 写成流程清单/固定模板（human 明确：只要遵循思考流程，形式自由）；不得给 lesson 设固定小节模板
- 不得改写 human 五步文本的要义（可补"如何判定"，不可换概念/弱化"必须上报 human"）
- 不得改动 dream 侧代码/doc（共用 canon 即可）、不得动 web/src/**、internal/web/**、内网 humans 页面与草稿持久化
- 不得保留任何旧 sense 方法论痕迹（系统论描述符/三维打分/阶段/回流）
- 不得 git 操作；不得修改 plan/ 下任何文件（含 gate4.py）
