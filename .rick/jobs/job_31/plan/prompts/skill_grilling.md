# Skill: Grilling（结构化追问协议）

## 设计树模型

Coding plan 的设计阶段是一棵**设计树**：

- **每一层**代表一个设计层次，由若干**模块**组成
- **每个模块**是下一层的根节点，向下展开为更细粒度的子模块
- **每一层**可表达为一条由模块间调用关系构成的 **pipeline**（A → B → C → ...）
- **非叶子层**：澄清该层的 pipeline——哪些模块存在、各自职责、相互调用关系
- **叶子层**：将每个模块的决策落实到四个维度：
  1. 关键代码实现（文件路径 + 函数签名）
  2. 文件结构（新建/修改哪些文件，目录组织）
  3. 工具调用（命令 + 参数）
  4. 环境依赖 + 配置（依赖项、环境变量、配置文件）

**Grilling 的任务**：逐层遍历设计树，在每一层循环追问，直到该层达标后再下钻，直至整棵树的叶子层全部落实。

---

## 核心指令

Interview me relentlessly about every aspect of this plan until we reach a shared understanding.
Model the plan as a design tree. Traverse it layer by layer — at each layer, identify the modules and their pipeline (call relationships), then loop asking questions until the layer meets its termination condition before descending to the next layer.
Ask all questions for the current layer at once. For each question, provide your recommended answer.
If a question can be answered by exploring the codebase, explore the codebase instead of asking.

---

## 每层追问流程

```
for each layer (top-down):
    while 当前层未达标:
        1. 识别该层的模块组成和 pipeline（调用关系）
        2. 批量列出当前层所有待澄清问题（含推荐答案）
        3. 等待用户回答
        4. 重新评估当前层终止条件
    descend: 以本层每个模块为根节点，展开下一层
```

---

## 每层终止条件

**非叶子层**达标——该层 pipeline 已完全澄清：
- 所有模块已命名，职责边界清晰，无重叠或遗漏
- 模块间调用关系（pipeline 顺序、接口契约）已确认
- 每个模块的输入/输出已定义，下层可完备支撑上层需求

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
- **不得遗漏分支**：每个模块都要展开到叶子层，不留模糊决策
- **优先探索代码库**：能读文件/grep 回答的问题，先探索再提问

---

## 全局终止条件

所有层均达标（叶子层四个维度全部落实）后，终止追问并声明：

**"Grilling 完成——设计树已遍历完毕，所有模块已落实到代码实现/文件结构/工具调用/环境配置。"**

随后输出结构化决策摘要，按层列出每个模块的关键决策。
