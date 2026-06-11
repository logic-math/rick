# Chapter 8: Observing Facts — 事实总结

## 核心概念

- **Observation（观察）**：收集关于实际程序执行的事实，不（理论上）改变程序行为
- **Do not interfere（不干扰原则）**：观察手段应尽量不改变程序行为，这是观察的首要原则
- **Logging（日志记录）**：在程序中插入输出语句记录运行时状态
- **Interactive debugger（交互式调试器）**：允许程序员在运行时暂停、检查、修改程序状态的工具
- **Postmortem debugging（事后调试）**：程序崩溃后，通过 core dump 分析崩溃时状态
- **Core dump**：程序崩溃时操作系统保存的进程内存镜像
- **Watchpoint**：在变量值改变时触发断点的调试器功能
- **COCA**：统一代码/数据查询语言（书中介绍）

## 主要内容

### 8.1 观察的首要原则：不干扰
书中将"**Do not interfere**"列为观察的首要原则，并用一张完整的表格梳理每种技术各自引入的干扰（见 8.6 节汇总）。

核心张力：观察需要收集信息，但收集信息必然有代价（性能、内存、执行路径改变）。

**Heisenbug 概念**（书中明确使用此术语）：观测行为本身改变 bug 的出现——是"观测不可避免影响系统"的最极端承认。

### 8.2 Logging（日志记录）

**层次一：printf debugging**
- 直接在代码中插入 `printf`/`print` 语句
- 优点：简单直接
- 缺点：需要修改代码，难以批量关闭，多线程下输出混乱

**层次二：日志框架（LOG4J/SLF4J 等）**
- 分级日志：TRACE < DEBUG < INFO < WARN < ERROR < FATAL
- 运行时可配置级别
- 不同 appender（控制台/文件/网络）
- 日志格式：时间戳 + 线程 + 类名 + 消息

**层次三：AOP 切面日志（ASPECTJ）**
- 不修改业务代码，通过切面在指定连接点插入日志
- 编译期或运行期织入
- 完全隔离日志代码与业务代码

**层次四：二进制插桩（PIN）**
- 在机器码级别插入观察代码
- 无需源代码
- 可观察任意内存访问和指令执行

**干扰程度**：printf < LOG4J（生产关闭）< ASPECTJ < PIN

### 8.3 Interactive Debuggers（交互式调试器）

**GDB 完整命令集**：
- `break <location>`：设置断点
- `run [args]`：启动程序
- `next`/`step`：单步（不进入/进入函数）
- `continue`：继续执行到下一断点
- `print <expr>`：打印表达式值
- `display <expr>`：每次停止时自动显示
- `watch <var>`：设置监视点（变量改变时停止）
- `backtrace`/`bt`：显示调用栈
- `frame <n>`：切换到第 n 个栈帧
- `list`：显示当前位置源代码
- `info locals`：显示当前函数局部变量
- `set variable <var>=<val>`：修改变量值

**Watchpoint（监视点）**：
- 当指定内存地址或变量的值改变时触发暂停
- **性能代价**：引入 1,000 倍以上性能降低（书中数据）
- 是所有调试手段中**干扰最大**的

**COCA 统一查询语言**（书中 Section 8.3 介绍）：
- 统一语法查询代码结构和运行时数据
- 示例：`select * from methods where name matches "get*" and calls "toString"`

### 8.4 Postmortem Debugging（事后调试）

**Core dump 生成**：
- Linux：`ulimit -c unlimited` → 崩溃生成 `core` 文件
- Windows：Dr. Watson 自动生成 minidump
- 内容：所有寄存器、堆栈、堆内存的快照

**GDB 加载 core dump**：
```bash
gdb ./program core
(gdb) bt    # 查看崩溃时的调用栈
(gdb) info registers    # 查看寄存器状态
(gdb) frame 3    # 切换到第3个栈帧
(gdb) info locals    # 查看局部变量
```

**核心地位**：backtrace（回溯调用栈）是 postmortem debugging 的最重要工具。

**理论上零干扰**：事后调试是唯一对原始运行**理论上零干扰**的技术——因为在原始崩溃时没有任何额外代码运行（core dump 由 OS 保存）。

**最佳实践（书中）**：在调试器内重复运行（replay within debugger）——加载 core dump 后，在调试器内重新执行以观察崩溃前的状态序列。

### 8.5 各技术干扰程度对比

| 技术 | 干扰程度 | 原因 |
|------|---------|------|
| Postmortem（core dump）| 零（理论）| 原始运行无额外代码 |
| 日志宏（编译期关闭）| 零（实践）| 编译后不存在 |
| printf debugging | 低-中 | 改变 I/O 时序 |
| LOG4J（运行时）| 中 | CPU/内存开销 |
| 断点调试 | 高 | 暂停整个程序 |
| 监视点（watchpoint）| 极高 | 1000x 性能降低 |

## 调试方法/技术（logging/debugger/postmortem详述）

**调试选择流程**：
1. 首先尝试 postmortem（零干扰，但只有崩溃信息）
2. 其次尝试 logging（低干扰，但需要预先插入）
3. 再次尝试 interactive debugger（高干扰，但最灵活）
4. 最后使用 watchpoint（极高干扰，用于定位内存写入位置）

## 对四个追问的回答

**1. 「实验报告」vs「解决问题」**
书中将观察记录（日志输出、调试器输出）定位为科学调试法（Chapter 6）中"观察"步骤的产出。记录是解决问题的中间产物，不是最终目标。两者不对立。

**2. 「探针不影响系统状态」（本章最直接回答）**
书中将"**Do not interfere**"列为观察的**首要原则**，并系统梳理了每种技术的干扰程度：
- 事后调试理论上零干扰
- 日志宏编译期关闭时零干扰但同时丧失观察能力
- 监视点干扰最大（1000x 性能降低）
- **Heisenbug**（观测导致 bug 消失）是书中对"观测不可避免影响系统"的最极端承认
- **结论**：完全不干扰是理想，实践中所有观察都有代价，需要权衡

**3. 「学习、设计、验证正交」**
本章未直接讨论此框架。观察（本章主题）对应学习阶段的信息收集，是验证阶段的前置条件。三者不是正交的。

**4. 「理解状态转移是否足以覆盖所有bug」**
本章未直接讨论此问题。但 Heisenbug 的存在本身说明：状态转移在某些情况下因为观测本身的影响而无法被准确捕获。

## 关键引用

- "Do not interfere." （Section 8.1，观察的首要原则）
- "Watchpoints introduce a performance slowdown of factor 1,000 or more."
- "Postmortem debugging is the only technique that introduces zero interference in the original run."
- "Heisenbugs: the act of observation changes the observed behavior."

## 与AI Agent调试的关联

- **探针不透明性**：向 AI Agent 的 prompt 中添加调试指令（"请记录你的每个步骤"）会改变 AI 的行为——这是 AI 版的 Heisenbug，无法消除
- **Postmortem 对应**：记录 AI Agent 每次工具调用的完整输入/输出（不在执行时干预）是 AI 版的 core dump，干扰最小
- **日志级别**：rick 可以设置"调试模式"（记录所有中间步骤）和"生产模式"（只记录关键事件），对应 LOG4J 的分级
- **Watch 对应**：对关键变量（如任务状态、测试结果）设置条件触发记录，对应 watchpoint
