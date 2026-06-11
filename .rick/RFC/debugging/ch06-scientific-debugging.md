# Chapter 6: Scientific Debugging — 事实总结

## 核心概念

- **Scientific method（科学方法）**：提出假设 → 设计实验 → 执行实验 → 观察结果 → 修正假设的迭代过程
- **Hypothesis（假设）**：关于 defect 或 infection 位置的可证伪命题
- **Prediction（预测）**：基于假设对程序行为的具体预期
- **Observation（观察）**：实际执行程序后记录到的事实
- **Refutation（证伪）**：实验结果与预测不符，假设被推翻
- **Confirmation（确认）**：实验结果与预测符合，假设得到支持（但未被证明）
- **Logbook（调试日志）**：记录假设、实验、观察、结论的文档

## 主要内容

### 6.1 调试作为科学实验
书中将调试定义为应用科学方法的过程。引用 Mastermind 游戏作为类比：
- Mastermind：猜测颜色序列，得到反馈，更新猜测
- 调试：提出假设，设计实验，观察结果，更新假设

**核心论点**："If you don't write down what you tried, you'll be trying the same things again in the middle of the night."（不记录尝试过的内容，凌晨还会重复相同尝试）

### 6.2 假设的要求
好的调试假设必须满足：
1. **可证伪**：存在能够推翻它的实验
2. **具体**：指向特定的代码位置或变量
3. **与已知事实一致**：不能与已观察到的现象矛盾

### 6.3 设计实验
实验设计要求：
- **控制变量**：每次只改变一个因素
- **可重复**：相同实验应产生相同结果
- **最小化**：实验尽量小，以快速执行

### 6.4 执行实验
执行时记录：
- 实验的具体操作（精确到命令行）
- 实际观察到的输出
- 与预测的对比

### 6.5 分析结果
四种可能结果：
1. **预测正确**：假设得到支持，可进一步细化
2. **预测错误**：假设被证伪，必须修改
3. **UNRESOLVED**：实验无法得出结论，需重新设计
4. **意外发现**：观察到与假设无关但有价值的信息

### 6.6 科学调试的完整循环
```
观察 Failure
    ↓
形成 Hypothesis
    ↓
推导 Prediction
    ↓
设计 Experiment
    ↓
执行 + 记录
    ↓
分析 Observation
    ↓
确认或证伪 Hypothesis
    ↓（证伪则返回"形成 Hypothesis"）
缩小范围，重复
    ↓
找到 Defect
```

### 6.7 调试大型程序的困难
书中在 Section 6.7 明确承认大型命令式程序中状态空间不可枚举：
- GCC 编译器有 44,000 个变量
- 原话（近似）："Are these 4 megabytes of executable code correct? There is simply no algorithmic way to answer this question in general."
- 算法调试在此情况下不可扩展（unscalable）

### 6.8 调试日志（Logbook）的价值
书中明确规定调试日志应记录：
1. 当前假设
2. 为验证假设设计的实验
3. 实验执行的确切步骤
4. 观察到的结果
5. 结论：假设是否得到支持

**理由**：不记录会导致重复劳动，且无法向他人解释调试过程。

### 6.9 观察过程的缺陷
Section 6.9 承认"观察过程可能有缺陷"（flawed）：
- 观察工具（调试器、日志）可能引入误差
- 主动修改状态的操作被归类为"实验"而非"观察"
- 该章未展开处理 Heisenbug，留到 Chapter 8

## 科学调试法的完整步骤

| 步骤 | 活动 | 输出 |
|------|------|------|
| 1. 观察 | 复现失败，收集症状 | 失败的精确描述 |
| 2. 假设 | 基于症状提出候选原因 | 可证伪的假设命题 |
| 3. 预测 | 从假设推导程序行为 | "如果假设成立，则 X 会发生" |
| 4. 实验 | 设计能验证预测的测试 | 实验方案 |
| 5. 执行 | 运行实验，记录结果 | 观察数据 |
| 6. 分析 | 对比预测与观察 | 假设确认/证伪 |
| 7. 迭代 | 缩小范围或修改假设 | 更精确的下一轮假设 |

## 假设-验证循环的具体操作

**形成假设的来源**（书中列举）：
- 代码检查（从语法/语义角度）
- 程序切片（Chapter 7）
- 异常检测（Chapter 11）
- 经验（"这类 bug 通常在 X 处"）

**证伪策略**：
- 替换怀疑的变量值，观察 failure 是否消失
- 注释掉怀疑的代码段
- 插入断言，观察何时触发

## 对四个追问的回答

**1. 「实验报告」vs「解决问题」**
书中将记录（Logbook）视为**解决问题的前提**，而非副产品。Mastermind 类比明确说明不记录会导致"凌晨彻夜重复相同调试"。但书中同时指出：记录的目的是支持后续实验（工具性价值），而不是记录本身有价值。**两者不对立，记录是解决问题的必要手段。**

**2. 「探针不影响系统状态」**
书中在 Section 6.9 承认"观察过程可能有缺陷"，但第 6 章未展开处理探针效应。主动修改状态的操作被归类为实验而非观察。Chapter 8 详细处理此问题。

**3. 「学习、设计、验证正交」**
书中的流程将学习（推导假设）、设计（设计实验）、验证（观察+结论）分离为不同活动，但它们是**串行依赖**而非正交独立的。每一步都以前一步的结果为输入。书中用 Figure 6.5 的层次图（非正交图）表达三者关系。

**4. 「理解状态转移是否足以覆盖所有bug」**
书中在 Section 6.7 **明确承认大型程序中状态空间不可枚举**，并指出算法调试在此情况下不可扩展。这是对"理解状态转移足以覆盖所有 bug"的直接否定。

## 关键引用

- "If you don't write down what you tried, you'll be trying the same things again in the middle of the night."
- "A hypothesis must be falsifiable."
- "Debugging is the process of applying the scientific method to find the cause of a failure."
- "Are these 4 megabytes of executable code correct? There is simply no algorithmic way to answer this question in general."（Section 6.7）

## 与AI Agent调试的关联

- **Logbook 对应 debug.md**：rick 的 debug.md 实现了书中 Logbook 的功能，但可以更结构化（假设/预测/实验/观察的四字段格式）
- **假设-验证循环**：rick 的重试机制应体现科学调试法：每次重试不是"再试一次"，而是基于上次失败信息形成新假设
- **预测字段**：rick 在执行任务前可要求 AI 给出预测（"如果这个方案正确，测试应该输出 X"），增加可证伪性
- **证伪 vs 确认**：rick 的测试通过只能"支持"任务完成假设，不能"证明"——这对应书中"确认不等于证明"的原则
