# Chapter 12: Causes and Effects — 事实总结

## 核心概念

- **Failure cause（失败原因）**：导致程序失败的最小变化——如果没有这个变化，失败就不会发生
- **Counterfactual model（反事实模型）**：来自 Hume (1748)，经 Lewis (1973) 精炼的因果性定义框架
- **Actual cause（实际原因）**：真实世界与"最近可能世界"（effect 不发生）之间的最小差异
- **Common context（公共上下文）**：在所有测试中都存在的条件——这些条件虽然必要，但不是 failure cause（无法被隔离）
- **Alternate world（替代世界）**：假设某个变化不存在时程序的行为
- **Post hoc ergo propter hoc**：时间上在前≠因果上是原因，是要避免的逻辑谬误
- **Three classes of failure causes（三类失败原因）**：Input / State / Code

## 主要内容

### 12.1 因果性的哲学基础

**Hume (1748)**：因果性的定义——"如果 C 事件没有发生，E 事件也不会发生"（反事实条件句）。

**Lewis (1973)** 的精炼：
- "最近可能世界"（nearest possible world）：与真实世界最相似但 C 不发生的世界
- Actual cause = 真实世界与最近可能世界之间的最小差异
- "最近"由 Ockham 剃刀原则决定：假设尽量少的差异

**书中对调试的应用**：
- 真实世界 = 程序以输入 c_fail 运行，产生 failure
- 替代世界 = 程序以输入 c_pass 运行，不产生 failure
- Actual cause = c_fail 与 c_pass 之间的最小差异集合

### 12.2 验证因果的唯一方法是实验

**书中明确指出**（Section 12.2，书页 273）：
- 演绎推理（代码分析）不足以确认因果
- 必须在"替代世界"（alternate world）中实际运行程序并观察
- **"Post hoc ergo propter hoc"是要避免的谬误**：时间上在前不等于因果上是原因

**具体操作**：
1. 找到一个失败输入 c_fail 和一个成功输入 c_pass
2. 提出假设：差异 D ⊆ (c_fail - c_pass) 是 failure cause
3. 实验：使用 c_pass + D 运行程序
4. 观察：程序是否失败？
   - 失败 → D 可能是 cause（进一步最小化）
   - 通过 → D 不足以导致 failure

### 12.3 Common Context 问题
**定义**：在 c_fail 和 c_pass 中都存在的元素是 common context，不能作为 failure cause 被隔离。

**问题**：Java 虚拟机、操作系统等基础设施是所有测试的 common context——它们可能是 failure 的必要条件，但不是 failure cause（因为无法通过实验改变它们）。

**书中的处理**：Common context 是调试的前提条件，接受其存在，专注于可变化的部分。

### 12.4 三类 Failure Cause（书页 278）

**1. Input Cause（输入原因）**：
- 程序输入的某个部分导致 failure
- 实验：删除/替换输入的某部分，观察 failure 是否消失
- 对应工具：Delta debugging（Chapter 5/13）

**2. State Cause（状态原因）**：
- 程序运行时某个状态变量的值导致 failure
- 实验：在执行中间替换该变量的值（注入正确值），观察 failure 是否消失
- 需要调试器支持

**3. Code Cause（代码原因）**：
- 程序代码的某个部分（变更）导致 failure
- 实验：撤销某个代码变更，观察 failure 是否消失
- 对应工具：`git bisect`、Delta debugging on code changes

### 12.5 验证实验的结构
每类 failure cause 的验证需要**两次实验**：
1. **有效果实验（Effect present）**：使用 c_fail + 假设的 cause → 应该产生 failure
2. **无效果实验（Effect absent）**：使用 c_fail - 假设的 cause → 应该不产生 failure

两次实验都成功 → cause 得到验证。

### 12.6 Failure Cause 与 Defect 的关系
- Failure cause 是可以被实验验证的、最小的导致 failure 的差异
- Defect 是代码中需要被修复的位置
- Failure cause 建议 fix（workaround：移除 cause 使 failure 消失）
- 但 failure cause 不一定是真正的 defect 位置（后续 Chapter 14/15 处理）

## Failure Cause的形式化定义

**基于反事实模型（Lewis, 1973）**：

设 c_fail 是失败配置，c_pass 是成功配置，failure() 是布尔函数。
- `c` 是 failure cause 当且仅当：
  1. `failure(c_fail) = true`（c_fail 确实失败）
  2. `c ⊆ c_fail`（c 是 c_fail 的子集）
  3. `failure(c_pass ∪ c) = true`（在成功配置中加入 c，程序失败）
  4. `failure(c_pass) = false`（c_pass 不失败，即不加 c 时程序通过）

**Actual cause = 满足上述条件的最小 c**（由 Ockham 剃刀原则）。

## 对四个追问的回答

**1. 「实验报告」vs「解决问题」**
书中诊断报告（实验记录）直接服务于问题解决，两者不对立。每次实验（有效果/无效果）的结果是下一步行动的依据。书中无明显对立，记录是解决问题的工具。

**2. 「探针不影响系统状态」**
Chapter 12 以"替换变量值"（state cause 实验）这类主动改变状态的方式验证因果。书中将此归类为"实验"而非"观察"——改变状态是实验的一部分，不是问题，而是设计。

**3. 「学习、设计、验证正交」**
本章的因果验证流程：假设（学习）→ 设计两次实验（设计）→ 执行实验（验证）→ 分析结果（反馈到学习）。三者是迭代循环，不是正交独立的。

**4. 「理解状态转移是否足以覆盖所有bug」**
书中没有直接声明，但通过 Common Context 问题隐含了局限：某些 failure 的原因在 common context 中，无法通过实验隔离。**停机问题的不可判定性**在 Chapter 7（演绎极限）和 Chapter 16 中有明确陈述。

## 关键引用

- Hume (1748) / Lewis (1973)：反事实因果性定义
- "Post hoc ergo propter hoc" is a fallacy in debugging.
- "The only way to verify a cause is through experimentation." （书页 273/275）
- "Common context cannot be isolated as a failure cause."
- 三类 failure cause：Input / State / Code（书页 278）

## 与AI Agent调试的关联

- **AI 版反事实实验**：对 AI Agent 失败，实验"如果去掉 prompt 中的某段话，AI 还会失败吗？"——这是 AI 版的 input cause 实验
- **State cause 对应**：在 AI Agent 执行中间替换某个工具调用的返回值（注入正确结果），观察后续步骤是否恢复正常——这是 AI 版的 state cause 实验
- **Code cause 对应**：撤销某次 prompt 模板修改，观察失败是否消失——对应 code cause 实验
- **Common context 问题**：Claude 模型版本、系统 prompt 等是 AI Agent 的 common context，无法通过实验隔离，但可能是失败的根本原因
