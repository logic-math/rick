# N1 软件工程视角 — rick 迁移到 pi agent

## 视角名:编译器三层分离 + IoC 边界 + 插件系统宿主-扩展

## 来源理论(任务派发指定 + 公认知识)

### T1:编译器前端-中间层-后端分离(Compiler Three-Phase Pipeline)
- **前端**(frontend):词法/语法/语义分析,产出 IR(中间表示)。与源语言绑定,与目标机器无关。
- **中间层**(middle-end/opt):基于 IR 做优化(常量折叠/死代码消除/循环优化)。与源语言和目标机器都无关。
- **后端**(backend):IR → 目标机器码(寄存器分配/指令调度)。与目标机器绑定。
- **价值**:任何前端可复用任何后端(LLVM 证明)。新增源语言只改前端,新增目标机只改后端,优化在中间层共享。

### T2:OS 内核系统调用 vs 用户态(System Call Boundary)
- **内核态**(kernel space):特权指令,直接访问硬件。
- **用户态**(user space):受限指令,通过 syscall 请求内核服务。
- **边界**:syscall 是稳定 ABI,内核实现可替换(Linux/macOS/Windows),用户程序不变。
- **价值**:内核替换不影响用户程序,只要 syscall 语义等价。

### T3:框架 vs 库的边界 — 控制反转(Inversion of Control / Hollywood Principle)
- **库**(library):你调用它(`you call it`)。控制权在你。
- **框架**(framework):它调用你(`it calls you`)。控制权在框架。
- **Hollywood Principle**:Don't call us, we'll call you。
- **IoC 容器**:框架定义扩展点(slot/hook),你填入实现,框架在适当时机调用。
- **价值**:框架掌握控制流,扩展点保证可定制性,无需修改框架代码。

### T4:插件系统设计模式(Eclipse/VSCode/Emacs)
- **Eclipse**:扩展点(extension point)+ 扩展(extension),OSGi bundle,同进程。
- **VSCode**:extension host 独立进程,通过 JSON-RPC 通信,进程隔离。
- **Emacs**:hook + advice,lisp 解释器内嵌,同进程动态绑定。
- **核心抽象**:宿主(host)定义扩展点契约,插件(plugin)实现契约,宿主在固定时机调用插件。
- **价值**:插件不修改宿主代码即可扩展行为,宿主升级时插件兼容(若契约稳定)。

---

## 视角本质描述(一句话)

> **rick 迁移到 pi = 把 rick 从"调用 claude code 库"重构为"被 pi 框架调用的扩展"——rick 退化为编译器前端(命令解析 + 系统提示词生成),pi 成为后端(agent loop 执行),两者的边界从 NDJSON stream 协议升级为 pi extension 契约。**

---

## 事实支撑(引用 R1-R8)

### 编译器三层分离映射(T1)
- **前端 = rick**:`internal/cmd/` 解析命令(plan/doing/easy/learning/dream) + `internal/prompt/manager.go` 生成系统提示词(IR)。产出 = 启动 pi 的 flag 集合 + system prompt 文件。
- **中间层 = pi agent core loop**:LLM 调用、tool 执行、compaction、context 管理。pi 的 6 类扩展点 = 中间层优化 hook。
- **后端 = LLM provider**:30+ provider,per-prompt 切换(R3-N3 已验证)。pi 的 provider 抽象 = 后端,与具体模型解耦。
- **R5-N3 事实**:human 已确认 "rick 的职责就是 解析命令,调度任务,构建系统提示词启动 pi agent"——这正是编译器前端的定义。

### OS 内核 syscall 边界映射(T2)
- **用户态 = rick**:9 个命令入口,无特权操作,只产 prompt + 读产物。
- **内核态 = pi**:agent loop 持有 LLM API key、执行 tool、写 session 文件。
- **syscall = pi flag/env 契约**:`--mode json` / `--system-prompt` / `--skill` / `PI_CODING_AGENT_DIR` 等 12 env + 22 flag(R8-N2 已枚举)。
- **R8-N1 事实**:pi 路径硬编码点 17 处,但 env/flag 可覆盖大部分——syscall 边界是稳定的,内核实现细节(.pi 目录)可被 user space 绕过。

### IoC / 框架 vs 库映射(T3)
- **现状(claude code 库)**:rick 主动 `exec.Command("claude")` 13 处(R1-N1),控制权在 rick。claude code 是被调用的库(尽管物理上是独立进程,但语义上 rick 控制流程)。
- **迁移后(pi 框架)**:pi 持有 agent loop,在 `before_agent_start` / `tool_call` / `tool_result` / `session_compact` 等扩展点回调 rick 注入的 extension。控制权反转——pi 调用 rick 的扩展。
- **R2-N4 事实**:pi 6 类扩展点(Prompt Templates / Skills / Extensions / Themes / Pi Packages / Agent Core 钩子)——这就是 IoC 容器的扩展点定义。
- **R4-N3 事实**:pi 5 类 compaction 自定义扩展点(session_before_compact / session_compact / ctx.compact / before_agent_start / transformContext)——这些是 framework 的 hook。
- **human 已确认**(Y11):"所有反馈性的,检查性的门禁都做成扩展内嵌到 pi 中"——这正是 IoC:门禁逻辑成为 pi 在 tool_call 时回调的扩展。

### 插件系统映射(T4)
- **宿主 = pi**:定义扩展点契约(extension events)。
- **插件 = rick 的 .rick/skills/ + .rick/extensions/**:doing_check / easy_check / debug skill 等。
- **VSCode 式进程隔离 vs Emacs 式同进程**:pi 的 extension 是同进程(Node.js require),而非 VSCode 式 extension host 进程隔离。这意味着 rick extension 崩溃会影响 pi 主进程——这是风险。
- **R8-N3 事实**:resources_discover 是追加式(非拦截),30+ 事件 0 个能拦截路径——pi 的扩展点是"追加式"而非"拦截式",类似 Eclipse 的 extension point(追加贡献)而非 Emacs 的 advice(可拦截)。

---

## 融贯性评估

### 自洽(视角内部逻辑一致)
- ✅ T1/T2/T3/T4 四个理论在"宿主-扩展边界"这一核心抽象上自洽:都描述"前端/用户态/库/插件"如何被"后端/内核态/框架/宿主"调用或调用后者。
- ✅ rick 退化前端 + pi 成为后端,与 human Y11 确认的 "rick = 系统提示词调度器,pi = 真正工作" 完全一致。

### 他洽(与其他视角不冲突)
- ✅ 与 N3 认知科学视角(工作记忆分区)他洽:前端=rick 产生意图,后端=pi 维护工作记忆,两者通过 system prompt(IR)传递。
- ✅ 与 N5 生物学视角(共生)他洽:宿主(pi)+ 共生体(rick extension)的隐喻不冲突。
- ⚠️ 与 N4 哲学视角(现象学 ready-to-hand)有张力:工具"消失"意味着控制流不可见,但 IoC 强调"框架调用你"——扩展点调用是显式的,不是 ready-to-hand。这个张力是 productive 的(见启发性问题 3)。

### 续洽(可延展到未来)
- ✅ V0-V5 迁移价值可被这个视角解释:V1 TDD 门禁 = pi 在 tool_call hook 回调 rick 扩展;V3 skill allowlist = pi 的 --skill/--no-skills flag;V4 loop 渐进式加载 = before_agent_start hook 动态替换 system prompt;V5 subagent 递归 = pi spawn 子 pi(类似编译器嵌套调用)。
- ✅ 后续 rick 功能"逐渐深入到 agent 内部"(human Y11)= rick 从前端向中间层渗透,但仍保持 IoC 边界。

---

## 独特偏见(此视角看不到什么)

1. **偏见:过度结构化**。把 rick/pi 关系框定为"前端-后端 + 宿主-扩展",可能掩盖 agent 执行过程中的**涌现性与不确定性**(N2 系统工程视角、N5 生态视角能补)。
2. **偏见:静态边界假设**。编译器三层分离假设边界稳定,但 human 说 "后续 rick 的功能将逐渐深入到 agent 内部"——边界是动态演化的,软件工程视角难以刻画这种"渗透"。
3. **偏见:忽视认知负荷**。IoC 把控制权交给框架,但人类开发者理解控制流的成本上升(调试时不知道框架何时回调)——这是 N3 认知科学视角能补的盲区。
4. **偏见:忽视权力关系**。框架-扩展是权力不对等关系(框架定义契约,扩展只能遵守)——N4 哲学视角能揭示这种"框架霸权"。

---

## 4 维打分初评(think 阶段会正式打分,research 初评)

| 维度 | 评分(1-10) | 说明 |
|---|---|---|
| 覆盖广度(覆盖迁移多少方面) | 8 | 覆盖架构边界/控制流/扩展机制/未来演进,但未覆盖认知/生态/哲学维度 |
| 解释深度(能否解释迁移本质) | 9 | IoC + 编译器三层 + 插件系统三重理论叠加,解释力强 |
| 跨领域原创性(与软件工程常识的距离) | 6 | 软件工程视角是"内行视角",原创性较低(但解释力高) |
| 行动启发性(能否指导决策) | 8 | 明确了"rick 退化为前端 + pi 成为框架"的边界,指导 extension 设计 |

---

## 3 启发性问题(信念/前提/反例)

### Q1(信念追问)
**这个视角的核心信念是"控制权应从 rick 反转到 pi"(IoC)。但如果 rick 仍然需要主动调度(如 plan/doing/learning 三阶段工作流,每阶段启动一次 pi),那么"控制权反转"是否只是**局部反转**(单次 agent loop 内反转,而非全局工作流反转)?rick 仍然是"工作流编排者",pi 是"单步执行者"——这与"库 vs 框架"的二元划分是否冲突?**

### Q2(前提追问)
**这个视角的前提是"pi 的 extension 契约足够稳定,可类比 syscall ABI"。但 R8-N4 事实显示 pi 20 天发布 9 个版本(5394 commits/1.5 年)——extension 契约是否真的稳定?如果 pi 在版本迭代中频繁变更 extension event 签名,那么"pi 是稳定内核、rick 是用户程序"的类比是否成立?还是说 rick 实际上绑定了一个**高速演化的内核**?**

### Q3(反例追问)
**反例:Emacs 的 hook+advice 模式允许扩展**拦截**核心函数(通过 advice-around),而 R8-N3 事实显示 pi 的 resources_discover 是**追加式非拦截**。如果 rick 需要在未来"拦截" pi 的某些行为(例如禁止 pi 加载某个 skill,或修改 pi 的 compaction 决策),pi 的扩展点是否足够?这个视角下,pi 是"Emacs 式可拦截"还是"Eclipse 式仅追加"?如果是后者,迁移是否锁死了未来的拦截需求?**
