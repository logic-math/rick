# Chapter 13: Isolating Failure Causes — 事实总结

## 核心概念

- **dd 算法**：ddmin 的广义版本，用于隔离两个配置之间导致失败的最小差异集合
- **Passing configuration (c_pass)**：程序通过的配置
- **Failing configuration (c_fail)**：程序失败的配置
- **Difference (Δ)**：c_fail 与 c_pass 之间的差异集合
- **Failure-inducing difference（失败诱导差异）**：Δ 的子集，足以将 c_pass 变为失败配置
- **Blame-o-meter**：书中对"隔离出导致 bug 的具体代码变更"功能的形象命名

## 主要内容

### 13.1 从 ddmin 到 dd
Chapter 5 的 ddmin 简化单个失败输入，Chapter 13 的 dd 隔离两个配置间的差异：
- 输入：c_pass（通过）+ c_fail（失败）+ 差异列表 Δ = c_fail - c_pass
- 输出：最小的 Δ' ⊆ Δ，使得 c_pass ∪ Δ' 失败

**关键区别**：ddmin 从零开始简化，dd 从已知差异出发隔离。

### 13.2 应用一：程序输入（Mozilla HTML 案例）

**案例**：Mozilla 浏览器在特定 HTML 输入时崩溃。
- c_pass：不崩溃的 HTML 文件
- c_fail：崩溃的 HTML 文件
- Δ：两个文件的字符级差异

**效率对比**：
- ddmin（从零简化）：**48 次测试**定位到 `<SELECT>` 标签
- dd（两文件差异隔离）：**5 次测试**定位到同一结果
- 实际 fuzz 测试数据：节省幅度高达 **100-500 倍**

### 13.3 应用二：线程调度（IBM DEJAVU 案例）

**问题**：并发 bug，只在特定线程调度序列下触发，极难复现。

**工具**：IBM DEJAVU（记录/重放框架）
- **记录阶段**：记录程序实际执行的所有线程切换点（yield points）
- **重放阶段**：按记录的序列精确重放，确定性地复现并发 bug

**dd 应用**：
- c_pass：已知不触发 bug 的线程调度序列
- c_fail：触发 bug 的线程调度序列
- Δ：两个序列的差异（线程切换点的不同）

**结果（书中数据）**：
- 差异总数：**38 亿个**线程切换差异
- 经约 **50 次测试**
- 隔离出 **1 个线程切换**（yield point 59,772,127）
- 准确定位到数据竞争位置：`Scene.java` 第 91 行

### 13.4 应用三：代码变更（GDB blame-o-meter 案例）

**问题**：GDB 4.17 相比 4.16 出现了新 bug，但变更量巨大。

**dd 应用**：
- c_pass：GDB 4.16（通过）
- c_fail：GDB 4.17（失败）
- Δ：8,721 个代码行级变更

**结果（书中数据）**：
- 约 **97 次测试**（约 3 小时编译时间）
- 隔离出 **1 个字符串常量变更**："arguments" → "argument list"
- 这一变更导致某个字符串比较失败，触发错误分支

**Blame-o-meter 概念**：给每个代码变更打分，分数高的变更是 failure cause 的可能性越大。

### 13.5 dd 算法的局限

书中明确指出：
1. **不可离散分解的配置**：图像处理等场景中，输入是连续的（像素值），无法离散分解为子集
2. **非单调性**：某些差异的子集通过，全集失败，但某些中间大小的子集也可能失败（non-monotone）——dd 假设单调性，违反时结果不可靠
3. **最新状态**：书中明确说"Delta debugging on states is a fairly recent technique and not yet fully evaluated"（Section 13.5 相关）

### 13.6 记录/重放分离（处理探针影响）
对并发 bug，dd 通过**记录/重放分离**解决探针影响问题：
- 录制阶段：记录真实执行（有轻微性能开销，但逻辑行为不变）
- 回放阶段：在回放中运行 dd 实验（确定性，可重复）
- 这样探针（调试代码）只在回放阶段运行，不影响原始录制的行为

## Delta Debugging应用（输入/线程/代码变更）

| 应用场景 | c_pass | c_fail | Δ的单位 | 典型效果 |
|---------|--------|--------|---------|---------|
| 程序输入 | 不崩溃输入 | 崩溃输入 | 字符/Token | 5次测试 vs 48次 |
| 线程调度 | 无bug调度 | 有bug调度 | 线程切换点 | 50次测试/38亿差异 |
| 代码变更 | 旧版本 | 新版本 | 代码行变更 | 97次测试/8721变更 |

## 对四个追问的回答

**1. 「实验报告」vs「解决问题」**
书中明确区分"找到 failure cause"与"修正 defect"——前者（dd 的输出）可以自动化，后者仍需人类理解程序内部。dd 的输出是一份精确的"failure cause 报告"，直接服务于问题解决，两者不对立。

**2. 「探针不影响系统状态」**
通过记录/重放分离解决，而非消除探针：录制时探针影响最小化，回放时 dd 实验在确定性环境中运行。书中未声称能完全消除探针影响。

**3. 「学习、设计、验证正交」**
dd 算法内部，学习（通过测试结果更新候选差异集）、设计（选择下一个测试的子集）、验证（执行测试）是耦合的迭代过程，不是正交独立的。

**4. 「理解状态转移是否足以覆盖所有bug」**
书中承认某些配置不可离散分解（图像处理等），且明确说明"Delta debugging on states is a fairly recent technique and not yet fully evaluated"——对于连续状态空间，状态转移的理解无法直接支撑 dd 的自动化。

## 关键引用

- "dd isolates the failure-inducing difference between a passing and a failing configuration."
- GDB blame-o-meter："97 tests, ~3 hours, isolated 1 string constant change"
- DEJAVU 案例："50 tests to isolate 1 thread switch from 3.8 billion differences"
- "Delta debugging on states is a fairly recent technique and not yet fully evaluated."

## 与AI Agent调试的关联

- **Prompt diff 隔离**：对比成功 prompt 和失败 prompt，用 dd 自动隔离导致失败的最小 prompt 差异
- **Tool call 序列 diff**：对比成功任务和失败任务的工具调用序列，隔离导致失败的最小工具调用差异——AI 版的线程调度 dd
- **版本 diff**：对比 rick 不同版本的 prompt 模板，隔离导致任务失败率上升的最小模板变更——AI 版的 blame-o-meter
- **记录/重放**：记录 AI Agent 的完整工具调用序列，支持在确定性回放中运行 dd 实验
