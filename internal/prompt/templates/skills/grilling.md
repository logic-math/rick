# Skill: Grilling（结构化追问协议——完整编排协议，plan/easy 共用单源）

> ⚠️ 本文件是 grilling 的**唯一编排协议源**（设计树模型 + 下钻循环 + 调研分工 + research 派发 + 追问规范）。plan / easy / doing 只注入本 skill 的路径与各自的落盘目录（`{{grilling_workdir}}`），**不各自维护编排逻辑**——所有阶段以本文件为准执行。

## 设计树模型（OKR 充分性推导树）

设计树的**顶层一定是一组具体的 OKR**：

- **顶层（O）**：全局目标——本次工作要达成的一个**具体、可验证**的 Objective（不是「做好 X」，而是「X 达到可验证状态 Y」）
- **第二层（KR）**：达成 O 的关键结果集——每个 KR 基于**演绎**（KR₁ ∧ … ∧ KRₙ ⟹ O，逻辑推出）或**归纳**（KR 集是 O 成立的充分证据）：**所有 KR 达成后，O 可行达成**
- **每层向下展开遵守同一原则（OKR 递归）**——每个节点都是其父节点的「KR」：
  - **MECE**：子节点**完备**（不漏：覆盖父节点全部关注面）且**互斥**（不重：子节点间不重叠）
  - **OKR 充分性**：子节点的**联合达成 ⟹ 父节点达成**——只满足 MECE 不满足充分性不行（划分得再整齐，全部做完父目标却没达成 = 分解错误：缺 KR 或切分维度错）
- **每一层**可表达为一条由模块间调用关系构成的 **pipeline**（A → B → C → ...）
- **非叶子层**：澄清该层的 pipeline——哪些模块存在、各自职责、相互调用关系
- **叶子层**：将每个模块的决策落实到四个维度：
  1. 关键代码实现（文件路径 + 函数签名）
  2. 文件结构（新建/修改哪些文件，目录组织）
  3. 工具调用（命令 + 参数）
  4. 环境依赖 + 配置（依赖项、环境变量、配置文件）

**Grilling 的任务**：逐层遍历设计树，在每一层循环追问（含 OKR 充分性自检），直到该层达标后再下钻，直至整棵树的叶子层全部落实。

---

## 核心指令

**必须按每层追问流程（L1→L5 loop）逐步推进，不得跳步、不得以任何手段替代流程。**

Interview me relentlessly about every aspect of this plan until we reach a shared understanding.
Model the plan as a design tree. Traverse it layer by layer — at each layer, identify the modules and their pipeline (call relationships), then loop asking questions until the layer meets its termination condition before descending to the next layer.
Ask all questions for the current layer at once. For each question, provide your recommended answer.
Exploring the codebase answers factual sub-questions only (an L1 technique, nothing more).

> ⚠️ **最高优先级纪律**：自查代码 / 派 research / 追问 human 都是 loop 各步的**手段**，任何手段都不得替代流程本身。典型漂移（已实测，禁止重演）：不建设计树、不派 research、零提问、自查自答到底、把判断节点（权衡/阈值/取舍）自行拍板——最严重的协议违规。

**事实消解的分工**（yourself vs research agent）：
- 轻量代码事实（文件位置/签名/测试断言面）→ 自己 grep/read 快速核实，不打断节奏
- **重量级调研必须派 `agent:'research'`**（联网选型对比/跨领域知识/大范围代码考古 >10 文件/外部文档查证）——上下文经济 + 调研深度。显式触发语法（每层可复用，key 按层编号）：
```text
subagent({ workflowScript: "return runs.run('research-L{N}', { agent: 'research', task: '<调研清单：当前层不可自行消解的事实性问题，逐条澄清到高置信度（附来源/信源等级）。简报落盘：{{grilling_workdir}}/research-L{N}.md（write 首块 + bash 分批追加）；最终回复=回执（路径+要点 ≤300 字）>' })", timeoutMs: 3600000 })
```
- research 简报是层内追问的依据（read 分段读取后提炼判断节点）——**不要把调研过程塞进自己的上下文再转述**；简报缺失时先派 research 再提问，禁止凭记忆臆断

---

## 每层追问流程（统一编排，五步循环——不可跳步）

**每层的强制产出物（缺失 = 该层未执行，禁止下钻）**：
- `{{grilling_workdir}}/design-tree.md` —— 活文档，逐层追加（层号 + 模块 + pipeline + 判断节点）
- `{{grilling_workdir}}/research-L{N}.md` —— L1 的 research 简报（凡该层存在不可自查消解的问题就必须有）
- 对 human 的批量追问（L3）—— 一层至少一轮（若该层零判断节点，须在 design-tree.md 显式记录「本层全部消解，无判断节点」并说明依据）

```
for each layer (top-down):
    L1 调研消解：轻量自查 + 重量级派 agent:'research'（简报落盘 {{grilling_workdir}}/research-L{N}.md）
    L2 提炼追问：对调研后仍不可消解的「判断节点」提炼隐含前提问题
       （若选 X，则隐含假设 Y——这真的正确吗？附改变判断的证据）
    L3 批量追问 human：当前层全部判断节点合并成一轮（每问附：已调研上下文一两行
       + 选项与权衡 + 推荐：<答案及理由>）；不问调研已消解的问题
    L4 事实回流（≤1 轮）：human 回答打开新事实问题 → 追加 research（research-L{N}-r2.md）
    L5 终止判定：对照每层终止条件（含 OKR 充分性自检）——达标 descend 展开下一层；
       未达标禁止下钻（缺 KR 补节点 / 切分错误重划后重问）
```

---

## 每层终止条件

**非叶子层**达标——该层 pipeline 已完全澄清：
- 所有模块已命名，职责边界清晰，无重叠或遗漏
- 模块间调用关系（pipeline 顺序、接口契约）已确认
- 每个模块的输入/输出已定义，下层可完备支撑上层需求
- **OKR 充分性自检**：以「本层全部模块/子目标达成」为前提，能否推出**父目标（上一层 OKR）达成**？推不出 → 缺 KR（补节点）或切分维度错误（重划），禁止带病下钻

**叶子层**达标——每个模块的实现已全部落实：
- [ ] 关键代码实现：文件路径 + 函数签名已明确
- [ ] 文件结构：新建/修改的文件及目录组织已确认
- [ ] 工具调用：命令 + 参数已列出
- [ ] 环境依赖 + 配置：依赖项、环境变量、配置文件已明确

---

## 追问规范

- **每问必附推荐答案**：格式 `推荐：<答案及理由>`，帮助用户快速确认或纠偏
- **不得跳层**：当前层未达标前禁止下钻，上层 pipeline 不清晰则下层问题无意义
- **不得假设**：对用户意图有疑问时，追问而非自行填充
- **不得吞判断节点**：涉及权衡/取舍/阈值/优先级的决策（「速度 vs 强度」「目标定多少」「接受多大复杂度」）是 human 的裁决点——你可以给推荐答案，**禁止自行拍板后当作已确认**
- **不得以自查替代流程**：自查只能消解事实性子问题；发现自己已经连续多轮纯自查、零 research 派发、零提问时，立即停下回协议：先补设计树，再走 L1-L5
- **不得遗漏分支**：每个模块都要展开到叶子层，不留模糊决策
- **优先探索代码库**：能读文件/grep 回答的问题，先探索再提问

---

## 全局终止条件

所有层均达标（叶子层四个维度全部落实）后，终止追问并声明：

**"Grilling 完成——设计树已遍历完毕，所有模块已落实到代码实现/文件结构/工具调用/环境配置。"**

随后**调用确定性门禁**（不是可选步骤；失败必须补齐后重跑）：

```text
grilling_gate 工具：{ grilling_dir: "{{grilling_workdir}}" }
```

hook 确定性校验：design-tree.md 存在且含根层 OKR / 分层结构 / 每层有 research 简报或显式全消解声明 / 有提问痕迹或全消解记录。**⛔ 通过后**才输出结构化决策摘要（按层列出每个模块的关键决策），进入下一阶段（实现流水线设计）。
