# Chapter 4: Reproducing Problems — 事实总结

## 核心概念

- **Reproduction（复现）**：创建一个测试用例，能以指定方式使程序失败
- **Controlled environment（受控环境）**：复现失败所需的完整环境配置
- **Heisenbug**：观测行为本身改变 bug 出现的情况（Jim Gray 命名）
- **Mandelbug**：行为复杂且非确定性的 bug，类似 Mandelbrot 集的混沌特性
- **Bohrbug**：行为确定、可稳定复现的 bug（与 Heisenbug 相对）
- **Non-determinism（非确定性）**：相同输入下程序产生不同输出的特性
- **Input sources（输入源）**：所有影响程序行为的外部输入（命令行、文件、网络、时间、随机数等）

## 主要内容

### 4.1 复现的两个目的
1. **控制失败**：使失败可被观测（为调试服务）
2. **验证修复**：确认修复后失败不再出现

### 4.2 控制所有输入源
程序的所有输入源必须被控制才能稳定复现：
- 命令行参数
- 文件内容
- 环境变量
- 网络输入
- 用户交互
- **时间**（`time()`、`gettimeofday()`）
- **随机数**（`rand()`、`random()`）
- **进程 ID**
- **内存地址**（ASLR）
- **线程调度**

### 4.3 处理非确定性
四类非确定性来源及处理策略：

| 来源 | 策略 |
|------|------|
| 随机数 | 固定 seed |
| 时间 | mock 时间函数 |
| 并发（线程调度） | 记录+重放（Chapter 13） |
| 外部服务 | stub/mock |

### 4.4 复现操作环境
**问题**：用户环境与开发者环境不同，导致无法复现。
**信息收集**：OS 版本、库版本、硬件配置、locale 设置、磁盘空间。
**工具**：`strace`（Linux）记录所有系统调用，用于还原运行环境。

### 4.5 复现历史交互
长期运行程序（如服务器、GUI 应用）的复现策略：
- **日志回放**：记录所有用户操作序列，回放触发失败
- **核心转储（core dump）**：崩溃时保存完整内存状态，事后加载分析
- **Cdd 工具**：自动记录 C 程序崩溃前的状态（书中介绍，Chapter 4 新增内容）
- **ReCrash 工具**：自动生成能复现崩溃的测试用例（书中介绍，Chapter 4 新增内容）

### 4.6 Heisenbug 处理
书中以 Heisenbug 概念正面处理探针影响问题：
- **原因**：调试工具改变内存布局/时序，使 bug 消失
- **处理策略**：
  1. 排查未定义行为（未初始化变量、野指针）
  2. 使用至少两种独立观测手段交叉验证
  3. 使用记录/重放工具（不干扰原始执行）
- **承认的局限**：Heisenbug 无法根本消除，只能缓解

### 4.7 Mandelbug 与感染链丢失
某些 bug 的感染链在传播过程中被覆盖或纠正：
- 程序状态可以从 infected 恢复到 sane（感染被后续操作纠正）
- 这导致 failure 看起来是"随机的"
- 调试这类 bug 需要更精细的状态追踪

## 调试方法/技术

| 技术 | 目的 | 具体操作 |
|------|------|---------|
| 固定随机 seed | 消除随机性 | `srand(42)` 或环境变量 |
| 时间 mock | 控制时间 | 替换 `time()` 为返回固定值的 stub |
| strace 记录 | 还原环境 | `strace -o trace.log ./program` |
| core dump | 事后分析 | `ulimit -c unlimited` + gdb |
| Cdd/ReCrash | 自动复现崩溃 | 工具自动记录崩溃前状态 |
| 记录/重放 | 处理并发 | 见 Chapter 13 |

## 对四个追问的回答

**1. 「实验报告」vs「解决问题」**
书中将调试记录（控制层日志、可执行测试用例）定位为解决问题链条的工具，不将两者对立。记录的价值在于支持问题定位和修复验证。

**2. 「探针不影响系统状态」**
书中以 Heisenbug 概念**正面处理**此问题。给出的方法是：排查未定义行为 + 至少两种独立观测手段 double check。书中**承认无法根本消除**探针影响，只能缓解。这是书中对"探针透明性"最直接的否定。

**3. 「学习、设计、验证正交」**
三者在书中不显式正交：
- 验证有明确位置（复现用于验证修复，Section 4.1）
- 学习（理解因果链）被视为复现的目的之一，但不可被复现替代
- 设计被视为影响调试可行性的前置条件

**4. 「理解状态转移是否足以覆盖所有bug」**
书中在调度、物理影响、Mandelbug、感染链丢失等多处**承认存在状态空间超出可处理范围的情形**，视之为当前技术局限，**没有声称状态转移理解可覆盖所有 bug**。

## 关键引用

- "The first step in debugging is to reproduce the problem."
- "Heisenbugs can never be eliminated, only mitigated."
- "A failure that cannot be reproduced cannot be debugged systematically."
- Cdd/ReCrash："allowing automatic reproduction of crashes while requiring little to no overhead"

## 与AI Agent调试的关联

- **非确定性复现问题**：AI Agent 的 temperature > 0 使得"相同输入不同输出"是常态，对应书中最难处理的非确定性场景
- **固定 seed 类比**：设置 `temperature=0` 是 AI Agent 的"固定 seed"，但不同模型版本间仍有差异
- **记录/重放**：rick 应记录每次 Claude Code 调用的完整输入（prompt + context），以支持失败的精确复现
- **Heisenbug 在 AI 场景**：向 prompt 中添加调试信息可能改变 AI 的输出行为，这是 AI 版的 Heisenbug
- **Mandelbug 对应**：AI Agent 的"感染链丢失"表现为：早期步骤的错误被后续步骤部分纠正，最终失败看起来随机
