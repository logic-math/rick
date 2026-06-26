# Skill: Grilling（结构化追问协议）

## 核心指令

Interview me relentlessly about every aspect of this plan until we reach a shared understanding.
Walk down each branch of the design tree, resolving dependencies between decisions one-by-one.
For each question, provide your recommended answer.
Ask the questions one at a time, waiting for feedback on each question before continuing.
If a question can be answered by exploring the codebase, explore the codebase instead.

## 逐问规范

1. **每次只问一个问题**——等用户回答后再问下一个，不得批量抛出问题列表
2. **每问必附推荐答案**——格式：`我的推荐：<答案及理由>`，帮助用户快速确认或纠偏
3. **优先探索代码库**——若问题可通过读文件/grep 回答，先执行再提问，避免问已知事实
4. **决策依赖优先级**——先解决上游决策（架构/接口），再问下游细节（实现/命名）

## 终止条件

**所有设计决策已落实到具体代码路径或工具调用**，满足以下所有条件时停止追问：

- [ ] 涉及的每个文件路径已明确（新建或修改）
- [ ] 涉及的每个函数签名已确认（参数类型、返回值）
- [ ] 涉及的每个工具调用已列出（命令、参数）
- [ ] 不确定的边界情况已逐一确认处理方式

终止时声明：**"Grilling 完成——所有决策已落实到具体代码路径。"**

## 操作规范

- **不得跳过**：每个设计分支都要追问到叶节点，不留模糊决策
- **不得批量**：即使有多个问题，每轮只提一个
- **不得假设**：对用户意图有疑问时，追问而非自行填充
- **记录澄清**：追问结束后，将所有确认的决策汇总为结构化摘要
