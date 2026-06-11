# Chapter 11: Detecting Anomalies — 事实总结

## 核心概念

- **Anomaly（异常）**：程序行为与大多数其他运行相比不同的地方，可能是感染点的信号
- **Coverage anomaly（覆盖率异常）**：在失败运行中执行但在成功运行中不执行的代码（或反之）
- **Data anomaly（数据异常）**：在失败运行中出现但在成功运行中不出现的变量值组合
- **Invariant（不变量）**：从多次成功运行中自动推断出的、应该始终成立的程序属性
- **TARANTULA**：基于覆盖率对比的 fault localization 工具（Jones et al., 2002）
- **DAIKON**：自动推断程序不变量的工具（Ernst et al., 2001）
- **DIDUCE**：实时不变量监控工具（Hangal and Lam, 2002）
- **Statistical debugging（统计调试）**：通过统计分析大量运行数据定位 bug（Liblit et al., 2003）

## 主要内容

### 11.1 捕获正常行为
**核心思想**：先定义什么是"正常"，再用"正常"作为参照检测异常。
- 正常 = 大量成功运行的共同特征
- 异常 = 失败运行中出现但正常运行中不出现的特征

挑战：需要大量成功运行的数据，且"正常"的定义可能不完整。

### 11.2 比较覆盖率（TARANTULA）

**Jones et al. 2002**，基于代码覆盖率对比：

**TARANTULA 评分公式**：
```
suspiciousness(s) = 
  (failed(s)/totalFailed) / 
  ((passed(s)/totalPassed) + (failed(s)/totalFailed))
```
其中 `failed(s)` = 执行语句 s 的失败测试数，`passed(s)` = 执行语句 s 的通过测试数。

**可视化**：按 suspiciousness 对代码着色（红=高可疑，绿=低可疑）。
**效果**：在 7 个 Siemens 测试集程序中，平均可疑度排名前 5% 包含了 80%+ 的 defect。

### 11.3 统计调试（Liblit et al., 2003）

**场景**：程序已部署给大量用户，收集用户运行时的采样数据。
**CCRYPT 案例**：从 1 万次运行中，通过统计分析定位到加密库中的一个 bug。

**关键特征**：
- 不需要测试用例，只需"通过/失败"标签 + 程序内部采样数据
- 采样点：分支结果（branch taken/not taken）、函数返回值、变量值范围

**统计量**：Fisher's exact test 计算每个采样点与失败的相关性。

### 11.4 现场收集数据（采样策略）
在生产环境中采样的挑战：
- 性能开销必须足够低（< 5% CPU）
- 采样必须代表真实失败（采样率影响检测精度）
- 隐私：采样数据可能包含用户数据

**Liblit 采样策略**：随机采样，每 1000 次执行记录一次，开销 < 1%。

### 11.5 动态不变量（DAIKON）

**Ernst et al., 2001**，Daikon 工具：

**工作原理**：
1. 在程序关键点（函数入口/出口、循环边界）记录变量值
2. 对大量运行的记录数据，枚举候选不变量模板（如 `x > 0`、`x = y + 1`、`a[i] >= 0`）
3. 检验每个候选不变量在所有观察到的运行中是否成立
4. 输出通过验证的不变量

**候选不变量类型**（书中列举）：
- 数值关系：`x > 0`、`x = y * 2`、`x + y = z`
- 数组属性：`a[i] >= 0 for all i`、`a is sorted`
- 指针属性：`p != null`、`p points to valid memory`
- 序列关系：`x increases monotonically`

**DAIKON 的局限**：
- 多项式复杂度：候选不变量数量随变量数指数增长
- 只对**已观察到的运行**成立——不是真正的不变量，只是"观察到的规律"
- 反例：程序可能有未被测试覆盖的路径违反这些"不变量"

### 11.6 实时不变量（DIDUCE）

**Hangal and Lam, 2002**：
- 不是离线分析，而是实时监控
- 在程序运行时动态检测不变量违反
- 一旦检测到违反，立即报告（fail-fast 思想）
- 开销：6-20 倍性能降低

### 11.7 从异常到缺陷
发现异常≠找到缺陷，还需要：
1. 确认异常与失败相关（相关性 ≠ 因果性）
2. 排除环境因素（如操作系统差异导致的覆盖率差异）
3. 结合 Chapter 6 的科学调试法验证因果关系

书中的三个追问框架（Section 11.7）：
- 这个异常是否总是出现在失败时？
- 这个异常是否在成功时从不出现？
- 消除这个异常能否消除失败？

## 对四个追问的回答

**1. 「实验报告」vs「解决问题」**
异常检测是自动化的信息收集过程，产出的是"可疑列表"而非人工报告。书中无明显对立，但强调从异常到缺陷需要人工验证，异常检测结果服务于问题解决。

**2. 「探针不影响系统状态」**
DAIKON 的观测开销：变量值记录会增加 I/O，改变程序时序。DIDUCE 的 6-20 倍性能降低可能影响并发 bug 的出现。书中没有系统讨论这些工具的 Heisenbug 风险，但性能开销本身就是干扰。

**3. 「学习、设计、验证正交」**
本章的异常检测是"学习"工具（识别可疑位置），需要前置的"设计"（选择采样点、候选不变量模板）和后置的"验证"（Section 11.7 的三个追问）。三者是顺序依赖，非正交。

**4. 「理解状态转移是否足以覆盖所有bug」**
书中承认：
- DAIKON 只对已观察运行成立，不保证未覆盖路径
- TARANTULA 的统计相关性不等于因果关系
- 规范完整性不可达（无法枚举所有可能的不变量）
这些承认共同说明：状态转移的理解是必要但不充分的。

## 关键引用

- "Anomalies are differences in behavior between passing and failing runs."
- TARANTULA："the top 5% of suspicious statements contain 80%+ of defects"
- DAIKON："inferred invariants hold for observed runs, but not necessarily all runs"
- DIDUCE："6-20x performance overhead for real-time invariant monitoring"
- Section 11.7："correlation is not causation"

## 与AI Agent调试的关联

- **AI 版覆盖率异常**：对比成功任务和失败任务的工具调用序列，识别只在失败时出现的工具调用模式
- **AI 版不变量推断**：从大量成功任务执行中推断"应该始终成立的属性"（如"search 工具在 read 工具前调用"）
- **统计调试对应**：收集 rick 的大量任务执行数据，通过统计分析识别与失败相关的上下文特征
- **DAIKON 局限的对应**：AI Agent 的"不变量"只对训练时/测试时见过的任务类型成立，新类型任务可能违反所有推断的不变量
