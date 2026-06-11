# Chapter 2: Tracking Problems — 事实总结

## 核心概念

- **Problem report（问题报告）**：记录软件问题的结构化文档，包含复现步骤、症状、环境信息
- **Problem database（问题数据库）**：集中存储和管理问题报告的系统（如 Bugzilla）
- **Problem life cycle（问题生命周期）**：从问题发现到关闭的五个阶段
- **Severity（严重程度）**：问题对用户影响的程度（Bugzilla 7级：blocker/critical/major/normal/minor/trivial/enhancement）
- **Priority（优先级）**：解决问题的紧迫程度
- **WORKSFORME**：无法复现的问题状态
- **WONTFIX**：已知但决定不修复的问题

## 主要内容

### 2.1 软件问题的五阶段生命周期
1. **Open**：问题被报告，尚未处理
2. **Assigned**：已分配给开发者
3. **In Progress**：正在处理
4. **Resolved**：已处理（修复/关闭/推迟）
5. **Verified/Closed**：验证修复有效，问题关闭

### 2.2 问题报告的构成
基于 Bettenburg et al. 2008 年对 156 名开发者的调查（Saarland 大学），开发者最需要的报告元素：
1. **Stack trace**（最重要）
2. **Steps to reproduce**
3. **Test case**
4. **Screenshots**
5. **Version information**

### 2.3–2.4 问题管理与分类
BUGZILLA 模型：
- 7个 severity 级别
- 字段：Summary、Description、Component、Version、OS、Hardware
- 关键字段：Steps to Reproduce、Expected Results、Actual Results

### 2.5–2.6 问题处理状态机
状态转移：NEW → ASSIGNED → RESOLVED（FIXED/INVALID/WONTFIX/DUPLICATE/WORKSFORME/INCOMPLETE）→ VERIFIED → CLOSED

可重新打开（REOPENED）。

### 2.7 需求作为问题
需求变更和功能请求（enhancement）也用同一系统追踪，统一管理软件演化。

### 2.8 重复问题管理
DUPLICATE 状态：标记与已有问题相同；关联原始问题；修复原始时自动通知所有重复报告人。

### 2.9 问题与版本控制的双向关联
- commit message 中引用 bug ID：`fixes #1234`
- 从 bug 可以找到修复的 commit
- 从 commit 可以找到相关 bug
- 支持"这个版本修复了哪些 bug"的查询

### 2.10 问题与测试的关联
**关键结论**："Test cases make problem reports obsolete"（测试用例使问题报告冗余）。
- 一个可执行的测试用例比文字描述的问题报告更精确
- 问题关闭时应附上能复现该问题的测试用例
- 测试用例通过后表示问题已解决

## 调试方法/技术

- **结构化问题报告**：标准化字段确保信息完整性
- **状态机管理**：避免问题丢失或重复处理
- **双向关联**：bug↔commit 关联实现变更可追溯性
- **测试用例替代**：用可执行测试代替文字描述

## 对四个追问的回答

**1. 「实验报告」vs「解决问题」**
书中将记录（problem report）视为解决问题的信息前提，两者是顺序依赖而非对立关系。Section 2.10 进一步指出测试用例可使问题报告冗余——更精确的记录形式（可执行测试）比文字记录更有价值。

**2. 「探针不影响系统状态」**
Chapter 2 仅涉及"事后采集"（log、stack trace、core dump），未讨论探针影响系统状态的问题。该话题安排在 Chapter 8。

**3. 「学习、设计、验证正交」**
本章流程是顺序流水线（报告→分配→修复→验证→关闭），不支持三者完全正交。Section 2.10 隐含了测试（验证）与追踪（记录）的分离，但两者仍有顺序依赖。

**4. 「理解状态转移是否足以覆盖所有bug」**
本章未讨论此问题。最接近的是 WORKSFORME（环境不可达，无法复现）和 WONTFIX，这两个状态承认了某些问题的不可解性，但深层讨论推迟到 Chapter 6。

## 关键引用

- "Test cases make problem reports obsolete."
- "The problem database is the memory of the debugging process."
- BUGZILLA 的 WORKSFORME 状态定义："The problem described is not reproducible on our end."

## 与AI Agent调试的关联

- **rick 的 debug.md** 对应书中的 problem report，但缺乏结构化字段（severity、steps to reproduce、stack trace）
- **测试脚本** 对应书中"测试用例替代文字报告"的思想——rick 已有此机制
- **版本关联**：rick 用 git commit 关联任务完成，与书中 bug↔commit 双向关联思想一致
- **WORKSFORME 问题**：AI Agent 的非确定性输出导致大量"无法复现"场景，是 rick 调试能力的核心挑战之一
