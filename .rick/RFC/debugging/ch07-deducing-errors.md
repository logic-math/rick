# Chapter 7: Deducing Errors — 事实总结

## 核心概念

- **Deduction（演绎）**：从抽象的程序代码推导到具体程序运行中的可能错误，无需实际执行程序
- **Data dependence（数据依赖）**：变量 B 的值依赖于变量 A 的赋值
- **Control dependence（控制依赖）**：语句 B 的执行与否依赖于控制语句 A（如 if/while）
- **Program Dependence Graph (PDG)**：表示程序中所有数据依赖和控制依赖的有向图
- **Program slice（程序切片）**：给定切片准则（变量+位置），所有可能影响该点该变量的语句集合
- **Slicing criterion（切片准则）**：`<行号, 变量名>` 的二元组，指定关注的程序点
- **Backward slice（后向切片）**：影响某点某变量的所有语句（最常用于调试）
- **Forward slice（前向切片）**：某点某变量可能影响的所有后续语句
- **Chop（切割）**：两个准则之间的交集切片
- **Code smell（代码异味）**：静态分析可识别的可能导致 bug 的代码模式

## 主要内容

### 7.1 隔离值来源
调试中的核心问题：给定失败点的一个感染变量，它的值从哪里来？
- 静态方法：分析代码，找到所有可能赋值该变量的语句
- 动态方法：在实际执行中追踪该变量的实际赋值历史（Chapter 9）

### 7.2 控制流图（CFG）
基础数据结构：
- 节点：程序语句
- 边：执行流（包括条件分支、循环、函数调用/返回）
- 用于后续依赖分析的基础

### 7.3 追踪依赖

**数据依赖**：
- Use-Def 链：变量的每次使用（use）追溯到其定义（def）
- 可通过静态分析自动构建

**控制依赖**：
- 语句 B 控制依赖于语句 A：A 的执行结果决定 B 是否执行
- 后支配树（Post-dominator tree）算法计算控制依赖

### 7.4 Program Slicing

**后向切片（Backward Slice）**——最重要的调试工具：
- 准则：`<失败点行号, 失败变量名>`
- 输出：所有可能影响该变量在该点取值的语句集合
- 实际效果：将需要检查的代码量减少 60-95%

**前向切片（Forward Slice）**：
- 准则：`<某赋值语句, 被赋值变量>`
- 输出：该变量可能影响的所有后续语句
- 用途：评估修改某变量的影响范围

**Chop（切割）**：
- 两个准则之间的路径切片
- 用于找到从"某已知感染点"到"失败点"的最短代码路径

**Backbone/Dice**：
- 多个切片的交集/差集，用于进一步缩小范围

**精度对比（静态 vs 动态，书中数据）**：
- 静态切片：平均保留 30% 的程序语句
- 动态切片：平均保留 5% 的程序语句（精确 6 倍）

### 7.5 演绎代码异味（Code Smells）
静态分析可自动检测的常见 bug 模式：
- **未初始化变量**：变量在定义前被使用
- **空指针解引用**：指针可能为 null 时被解引用
- **类型不匹配**：赋值或比较时类型不兼容
- **不可达代码**：永远不会被执行的语句（如 `if(false)`）
- **无限循环**：循环条件永远为真的情况

工具：Lint（C）、FindBugs（Java）、Pylint（Python）、Clang Static Analyzer

### 7.6 静态分析的极限
**书中明确指出**（Section 7.6）：
- 静态分析是停机问题的实例，理论上不可判定
- 原文："any type of deduction is limited by the halting problem"
- 所有静态分析工具必须采用保守近似（conservative approximation）：宁可报告假阳性（false positives），不能漏报真阳性
- 这导致静态分析工具通常产生大量误报

## 调试方法/技术（重点：Program Slicing）

**使用后向切片调试的完整流程**：
1. 识别失败点（failure 发生的行号）
2. 确定失败变量（哪个变量取了错误的值）
3. 构建切片准则 `<行号, 变量名>`
4. 计算后向切片（工具自动完成）
5. 在切片范围内（而非全部代码）进行人工检查
6. 结合 Chapter 6 的科学调试法，在切片内提出假设

**工具**：
- Frama-C（C 语言静态切片）
- Joana（Java 切片）
- WHYLINE（动态切片，Chapter 9 详述）

## 对四个追问的回答

**1. 「实验报告」vs「解决问题」**
Chapter 7 是零次运行的演绎层——演绎层无实验记录。记录属于更高层（归纳/实验，见 Chapter 6）。书中没有对立，而是嵌套关系：演绎缩小范围，实验验证假设，记录支撑迭代。

**2. 「探针不影响系统状态」**
Chapter 7 是静态分析，天然规避此问题——完全不运行程序，因此探针影响为零。Section 7.5.3 提及：调试时插入的断言语句可能成为不可达代码（影响代码结构），指向 Chapter 8 的详细讨论。

**3. 「学习、设计、验证正交」**
书中用层次图（Figure 6.5）表达嵌套关系而非正交关系。演绎（Chapter 7）是学习阶段的工具，用于缩小假设空间；设计和验证在 Chapter 6 的科学调试循环中完成。

**4. 「理解状态转移是否足以覆盖所有bug」**
Section 7.6 明确指出这是停机问题的实例，不可判定，任何演绎必须采用保守近似。**书中原话："any type of deduction is limited by the halting problem."** 状态转移的完整理解在理论上不可达。

## 关键引用

- "any type of deduction is limited by the halting problem" （Section 7.6）
- "Program slicing reduces the code to be inspected by 60-95%."
- "Static slices are on average 30% of the program; dynamic slices are 5%."
- "Conservative approximation: report false positives rather than miss true positives."

## 与AI Agent调试的关联

- **AI 版"程序切片"**：对 AI Agent 的失败，可以分析哪些 prompt 片段、哪些工具调用结果、哪些中间输出对最终失败有影响——这是 AI 版的后向切片
- **停机问题的对应**：AI Agent 的行为理论上也不可完全静态预测（对应停机问题），所有 AI 调试工具也必须采用保守近似
- **代码异味 vs AI 异味**：AI Agent 有类似代码异味的模式——如"连续三次使用同一工具失败后不换策略"、"输出包含'I cannot'但继续执行"——可以静态检测
