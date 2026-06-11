# Chapter 3: Making Programs Fail — 事实总结

## 核心概念

- **Test case（测试用例）**：一组输入+预期输出，用于确定性地触发特定程序行为
- **Test driver（测试驱动）**：执行测试用例的框架代码
- **Test oracle（测试断言）**：判断程序输出是否正确的机制
- **Unit test（单元测试）**：针对单个函数/模块的最小粒度测试
- **Regression test（回归测试）**：防止已修复问题重新出现的测试集
- **Test isolation（测试隔离）**：确保测试之间互不干扰
- **setUp/tearDown**：测试框架中的初始化和清理钩子

## 主要内容

### 3.1 测试的目的（为调试服务）
书中明确区分两种测试目的：
- **Validation testing**：证明程序符合规格（找不到 bug）
- **Debugging testing**：让程序失败，以便定位 defect

调试阶段的测试目标是"让程序以可控方式失败"，而非证明其正确。

### 3.2–3.3 三个层次的测试
1. **展示层（Presentation layer）**：UI 测试，脆弱，难以自动化
2. **功能层（Functional layer）**：API/接口测试，推荐的调试入口
3. **单元层（Unit layer）**：函数级别，最精确，最容易隔离

以 MOZILLA 崩溃问题（BUGZILLA #24735）为贯穿全章的具体示例。

### 3.4 自动化测试框架
JUNIT（Java）/ UNITTEST（Python）框架结构：
- `setUp()`：每个测试前的初始化
- `tearDown()`：每个测试后的清理（**关键**：即使测试失败也执行，确保环境复原）
- `assert*()`：断言方法族

### 3.5 测试隔离
**问题**：测试之间共享状态会导致一个测试的副作用影响另一个。
**解决**：依赖反转（Dependency Injection）——将外部依赖（数据库、文件系统、网络）替换为可控的 mock 或 stub。

`AutomatedPresentation` 模式：用程序化接口代替 UI 操作，实现测试自动化。

### 3.6 先写测试再修复
**书中明确建议**：收到 bug 报告时，第一步是写一个能复现该 bug 的测试用例，而不是先看代码。
原因：
1. 强迫精确描述失败条件
2. 修复后可立即验证
3. 防止回归

### 3.7–3.8 随机测试与变异测试
- **Random testing**：自动生成随机输入，触发意外崩溃
- **Mutation testing**：自动修改代码（变异），检查测试套件是否能检测到变异

### 3.9 Quality assurance 的极限
书中直接陈述：**"Quality assurance can never reach perfection."**
软件模型检查（model checking）仅适用于可建模为有限状态机的系统。

## 调试方法/技术

| 技术 | 层次 | 自动化程度 | 适用场景 |
|------|------|-----------|---------|
| 手动测试 | 展示层 | 低 | UI bug |
| 功能测试 | 功能层 | 中 | API bug |
| 单元测试 | 单元层 | 高 | 逻辑 bug |
| 随机测试 | 任意层 | 高 | 崩溃/安全 bug |
| 变异测试 | 代码层 | 高 | 测试质量评估 |

## 对四个追问的回答

**1. 「实验报告」vs「解决问题」**
书中明确建议"问题发生时优先写测试用例而非写问题报告"——测试用例是问题记录的更精确、可执行形式。两者不对立，测试用例是更高质量的记录。

**2. 「探针不影响系统状态」**
书中通过 `tearDown` 机制和依赖反转（`AutomatedPresentation`）处理此问题——测试执行后清理环境，使探针（测试代码）不永久改变系统状态。没有系统性讨论 Heisenbug 问题，该话题留到 Chapter 8。

**3. 「学习、设计、验证正交」**
书中支持三者尽量独立，但明确指出**设计质量是验证独立性的前提**——循环依赖（测试依赖被测模块的内部状态）会使测试无法隔离进行。

**4. 「理解状态转移是否足以覆盖所有bug」**
书中直接陈述：**"Quality assurance can never reach perfection."** 软件模型检查仅适用于可建模为有限状态机的系统。随机测试的存在本身承认了确定性枚举的不充分性。

## 关键引用

- "Quality assurance can never reach perfection."
- "Before fixing a bug, write a test case that exposes it."
- "Testing can show the presence of bugs, but never their absence." (Dijkstra，转引)

## 与AI Agent调试的关联

- **先写测试原则**：rick 的测试脚本生成在 doing 阶段的实现对应此原则，但测试脚本是在任务执行前生成，而非在失败后写
- **tearDown 对应**：rick 的任务执行应确保每次重试前环境复原，否则前次失败状态会污染下次执行
- **随机测试在 AI 场景**：AI Agent 本身具有随机性（temperature > 0），每次执行都是一次"随机测试"，rick 需要利用这个特性
- **测试隔离问题**：AI Agent 调用外部工具（文件系统、shell）时，工具副作用会跨测试积累，对应 Chapter 3 的测试隔离问题
