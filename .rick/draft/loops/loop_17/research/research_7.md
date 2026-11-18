# research_7 — 哥德尔编译器类比（#28）跨领域深挖：不可自证明层＝抽象层的四个工业实证 — 2026-11-09

> 时间基准 2026-11-09。信源：A = 一手原文（Thompson《Reflections on Trusting Trust》CACM 27(8):761-763, 1984 全文；seL4 SOSP'09 论文+trustworthy.systems 官方 proof statistics；Xen SOSP'03；Turtles OSDI'10；Nizza CollaborateCom'05；Wheeler DDC 官方页+ACSAC'05；Spolsky 抽象泄漏律 2002 原文；Härder & Reuter CSUR 15(4), 1983；Popek & Goldberg CACM 17(7), 1974；本地 superopt 代码原文）；B = 二手可靠（Wikipedia TCB/ACID/Popek-Goldberg 条目、SEP 哥德尔条目、Feferman IAS 稿、Cornell CS5430 课件、MIT Saltzer 站点）。**凡「与 #9/#19/#28 的映射」均为本研究提出的结构同构判断（我的框架非源文）——源文只提供领域事实。**

## 事实性结论

### 一、操作系统：特权环＝不可降级层；hypervisor＝在抽象层上再建抽象
- **x86 四特权环，内核独占 ring 0**：环编号 0（最特权）→3（最不特权）；OS 代码在 ring 0 执行「因为只有 ring 0 能执行特权指令」，应用代码一般用 ring 3；ring 1/2 自 OS/2 后无知名 x86 OS 使用。前提：特权指令（装页表、I/O 等）仅 ring 0 可执行，越权执行被处理器拒绝（静默失败或 fault）。来源：Xen SOSP'03 §3.1（A）。
- **Xen 的环压缩部署**：Xen 占 ring 0、guest OS 改到 ring 1（半虚拟化）——「防止 guest OS 直接执行特权指令、同时与 ring 3 应用安全隔离」，特权指令须经 Xen 验证执行。前提：x86 2005 年前无 VT-x/SVM。来源：Xen SOSP'03（A）。
- **「不可降级层使上层获得自由」的结构论证原文**：kernel＝「在处理器最特权模式执行的系统部分，拥有无限硬件访问权；kernel 的任何实现故障都有可能破坏系统其余部分的正确运行」——因此安全/可靠优先时「标准做法是减少特权代码量、最小化 bug 暴露面」，此乃 security kernel／separation kernel／MILS／microkernel／小 hypervisor 作为最小信任基的共同动机。来源：seL4 SOSP'09 §1（A）。
- **虚拟化形式判据（在抽象层上建抽象的许可证）**：Popek & Goldberg《Formal Requirements for Virtualizable Third Generation Architectures》CACM 17(7):412-421, 1974——VMM 三性质＝等价（VMM 下程序效果与裸机一致，除资源与计时）、资源控制、效率；可虚拟化充分条件＝敏感指令全部特权化（trap）。x86 长期不满足 → VMware 走运行时二进制翻译、Xen 走半虚拟化、Intel/AMD 2005-06 加 VMX/SVM 后硬件化。来源：Popek & Goldberg 1974（A 原文；判据细节经 Wikipedia 条目转述 B）+ Turtles §2（A）。
- **嵌套虚拟化（hypervisor 上再跑 hypervisor）＝一层抽象上再建抽象的工程实证**：x86 只有单层架构虚拟化支持——「无论 trap 发生在哪层（1..n），执行都掉回 level 0」；Turtles（Linux/KVM）让 L0 递归模拟 VMX 给 L1，L1 又可给 L2……多层虚拟化复用在单层硬件支持上（multi-dimensional paging 把 3 层 MMU 翻译表压进硬件仅有的 2 层）。代价：常见负载开销 6-8%（vs 单层虚拟化）；cpuid 裸机 ~100 周期→单层 VM ~2,600→嵌套 ~58,000。来源：Turtles OSDI'10（A）；嵌套虚拟化理论首见 Popek & Goldberg。

### 二、形式验证：TCB＝显式声明的不可证明层；seL4＝最小化+全量证明的代价函数
- **TCB 定义**：对系统安全至关重要的全部硬件/固件/软件组件集合——TCB 内的 bug 可能危害整个系统的安全性质；TCB 外组件的 misbehave 不得泄漏超出授权的权限。来源：Wikipedia TCB（B）+ Nizza'05 操作化（A）。
- **TCB 最小化原则原文**：「TCB 应最小化：只有本质的 security-sensitive 功能才属于 TCB——即省去就会破坏功能与安全的那部分组件」；依据＝错误数与代码量正相关、易受攻击性随复杂度上升——「宁要大量小组件，不要少量大组件」。实证：Linux 2.4 内核 70% 是驱动、错误率高 7 倍（Chou et al. SOSP'01）；VPN 安全敏感功能 <5%——抽出 IPSec 独立域后 TCB ~55k 行（IPSec 域仅 5k）；邮件签名 crypto 域 TCB ~105k 行 vs Linux+X+Mozilla+JVM 栈 ~1000 万行。来源：Nizza CollaborateCom'05（A）。
- **1975 根**：Saltzer & Schroeder《The Protection of Information in Computer Systems》（Proc IEEE 63(9), 1975）设计原则族——Economy of Mechanism（机制越简单越小越好：易理解/构造/分析，故不易藏未知漏洞）+ Least Privilege 等。来源：论文（A；原文未直读，经 MIT Saltzer 站点+Cornell CS5430 课件转述 B）。
- **seL4 数字（原文口径）**：8,700 行 C + 600 行 ARM 汇编；已验证 7,500 行 C（汇编+引导代码 1,200 行未验证）；证明 200,000 行 Isabelle 脚本（≈3,500 页 A4）；设计+文档+实现+验证合计 25 人年（重做估 <10 人年）；全程修 ~460 bug（C 代码 160 个：16 个验证前+144 个验证期发现）；对照 Common Criteria EAL6 经验 $1,000/行 → 传统评估 ~$8.7M 且保证更弱。来源：trustworthy.systems 官方 proof statistics + SOSP'09 slides（A）。
- **seL4 的不可证明层＝显式假设清单**：「We assume the correctness of the compiler, assembly code, boot code, management of caches, and the hardware; we prove everything else」——编译器恰在不可验证清单上（与 Thompson 攻击面精确重合，此注=我的框架非源文）。来源：seL4 SOSP'09 摘要（A）。
- **TCB 边界可移动但信任不可移除**：把内存分配 policy 移到 userland「不改变该模块属于 TCB 的事实，但使其可被单独验证、并可依赖已验证内核性质」。来源：seL4 SOSP'09 §3.2（A）。

### 三、编译器：Thompson 1984＝#9 自指性边界的编译器经典版；DDC＝可信自举的解法
- **三段论证（图灵奖演讲全文，A）**：Stage I 自复制程序（quine：可携带任意 excess baggage 且能被另一程序生成）；Stage II 编译器「学习」（\v 例子——自指定义教一次即自延续，「as close to a 'learning' program as I have seen」）；Stage III 双木马：①编译 login 时插入后门（接受特定密码）；②是 Stage I 程序，在编译器编译自身时注入两个木马。「训练」完成后删净源码 bug，新二进制每次重编译自动再注入——「the login command will remain bugged with no trace in source anywhere」。来源：CACM 27(8):761-763, 1984-08（A）。
- **道德律原文**：「You can't trust code that you did not totally create yourself… No amount of source-level verification or scrutiny will protect you from using untrusted code」；攻击面不限于编译器——assembler/loader/microcode 皆可，「层级越低越难检测；藏好的微码木马几乎不可检测」；首次读到该攻击是在空军对 Multics 的安全评估（原文标注「Unknown Air Force Document」；通认 Karger & Schell 1974，无一手确认）。来源：同上（A）。
- **解法：Diverse Double-Compiling（Wheeler）**：用受信编译器 cT 编译被测编译器的父本源码得 cP，再以 cP 编译被测源码，与被测二进制逐位比对——一致即「源码忠实代表二进制」；攻击者须同时攻陷原编译器与全部受信编译器（受信者由防御方事后自选）；不要求两个编译器产出相同二进制（常见误解），只要求父本确定性。载体：ACSAC'05 论文 + 2009 GMU 博士论文（形式证明+GCC/恶意编译器实验）；真实应用=GNU Mes。边界（作者自认）：DDC 通过≠无恶意（例：VS 2015 Update 2 默认植入遥测，测不出）；立场=「trust, but verify」——问题不是信任本身，而是缺独立验证过程。来源：Wheeler 官方页（A）。
- **与 #9 对应（我的框架非源文）**：木马②＝「验证器不能验证自己」的编译器版；解法不是更好的自审而是异源外部验证者——与 superopt 外置裁决（提案-裁决分离）同构。

### 四、数据库：ACID 原子性＝BEGIN/COMMIT 之间中间态不可见
- **ACID 铸词**：Härder & Reuter《Principles of Transaction-Oriented Database Recovery》CSUR 15(4):287-317, 1983 正式铸造 ACID 缩写，建于 Gray 更早工作（命名 atomicity/consistency/durability，未含 isolation）之上。来源：论文（A）+ Wikipedia ACID（B）。
- **原子性原文**：事务＝「$BEGIN_TRANSACTION 与 $COMMIT_TRANSACTION 之间的全部数据库交互」，要求「所有动作不可分地执行：要么全部正确反映在数据库中，要么什么都没发生」；COMMIT 前任何时点出错/回滚→数据库无任何变化——中间态对外不可见。来源：同上（A）。
- **superopt 对应物（本地代码 A；映射=我的框架非源文）**：实验落盘 `<ws>/codes/iter{N}-{uuid}/` 隔离目录（experiment_record.py:9/214）＝每实验独立工作区；映射：隔离目录=事务工作区，gates+裁决通过=COMMIT，未过=ROLLBACK（中间态不入种群/精英库）；差异：DB 的 commit 判定在事务系统内部（WAL），superopt 的 commit 判定外置（gates+adjudicator）。

### 五、语言设计：抽象泄漏律＝「完备抽象性」的反面；哥德尔表述的准确化
- **抽象泄漏律原文**：「All non-trivial abstractions, to some degree, are leaky」——TCP 试图完全抽象不可靠的 IP，但网线被咬断/劣质集线器时网络漏进抽象层（抽象的承诺是「正常运行时不让你看见」，不是「永不让你看见」）；推论：抽象省工作时间、不省学习时间（漏点出现仍须懂底层）；例证：页错误（内存抽象漏）、SQL 查询计划（声明式漏）、NFS（远程文件抽象漏）。来源：Spolsky《The Law of Leaky Abstractions》2002-11-11 原文（A）。
- **哥德尔第一不完备定理准确表述**：「在任何一致的、其中能承载一定量算术的形式系统 F 中，存在 F 语言的命题，在 F 内既不能证明也不能否证」；原始论文 1931《论 PM 及相关系统的形式不可判定命题》（核心为 Proposition VI）。来源：SEP（B）+ Feferman IAS 演讲稿（B）。
- **三个防 folk 误用点**：①限定缺一不可——一致性（不一致系统任何命题皆可证）+ 足够算术（纯一阶谓词演算是完备的，Gödel 1929 完备性定理）；②「不可证」相对 F——Gödel 句可在更强系统内证明，但更强系统有自己的新不可判定句（第二定理：F 不能证明自身一致性）→不完备性随层迁移而非消失；③只约束形式系统——外推到「人类认知/科学有永远证不出的真理」是 folk 版。来源：SEP/Feferman（B）。
- **对 #28/#19 的准确化（我的框架非源文）**：「上一层能处理下一层不可证的命题」对应「Gödel 句可证于更强系统」（数学上成立）；「完备抽象性」应读作工程完备（正常运行域内屏蔽底层，Spolsky 意义），泄漏律禁止「绝对完备」的措辞；层可无穷向上=每层都有新边界=第二定理的嵌套回归，#19「实用主义截断」=工程停止条件——与 Turtles 每层 6-8% 税同构：不是不能叠层，是每层有代价，实用主义决定叠几层。

### 六、核心问题：五领域怎么处理「不可自证明层的最小化与审计」（归纳+映射=我的框架非源文）
- **共同结构三件套（跨四领域归纳）**：①最小化——不可自证明层做小（内核小化/TCB 最小化/假设清单显式化——seL4 把「不可证明」写成显式假设清单而非假装不存在）；②隔离——该层与被服务层之间放硬边界（特权环/protection domain/事务边界），越界即 trap；③外部化验证——该层自身正确性不由被服务层证明，由异源外部验证（Isabelle 证内核、DDC 异源互证、human 裁决外置）。
- **TCB→「人类裁决层」映射**：TCB＝「必须信任但无法用系统内手段验证的组件集」。RSI 中 harness（gates/verifier/采样/门禁/adjudicator 规则）满足 TCB 定义——superopt 的「内核」；且 #9/Thompson 论证其不可由被 gate 的 loop 自审。superopt 实例（本地代码 A）：改门/删格/改格定义等硬动作只产提案（`<ws>/proposals/`），裁决在调度器进程之外完成（adjudicator.py 提案-裁决分离）+human 可改 decision 文件覆盖——与 seL4「内核外单独验证」同构。
- **harness 是不是 TSB：是**——且 superopt 已按 TCB 工程规范拆解它：gates.py「框架零阈值假设」把全部判定值（阈值/容差/评分函数）外移到 env 侧，而 env=gen env 与 human 对话生成（judgment.md human 原话：硬验证逻辑「与 human 对话而生成的，case by case」）＝机制/policy 分离；机制层小而稳定可审计，值层可溯源到 human。
- **「人类裁决层最小化」的准确表述**：不是 human 参与点越少越好，而是最小化「必须信任而不能独立验证的组件总量」；human＝TCB 中不可再降级的最终成员——如同 ring 0 不能被 ring 3 进程降级，价值判断不能被系统内裁决替代（human 终稿表述「人类是唯一可以做价值判断的主体」=TCB 的内核态地位）。
- **「每加一层抽象 TCB 变大」的工程证据**：seL4 比例 7.5k 行 C→200k 行证明→25 人年（证明/代码≈27:1，成本超线性）；Nizza 警告「不把最小 TCB 膨胀成大型模拟框架的 VMM 尚未实现」——每加一层不可独立验证的自动化，TCB 按代码量线性变大（错误数∝代码量，Chou 实证）。RSI 推论：harness 靠堆 agent/堆脚本加固＝TCB 单调膨胀；正解＝加固逻辑溯源到最小可审计核（gen env 溯源 human+外置裁决）+对 harness 自身做 seL4 式最小化或 DDC 式异源验证。

## R7 上报项
- [Intel 一手]：SDM Vol.3 特权环原文未读（网络受限）——ring 事实以 Xen SOSP'03 同行评审原文替代，缺官方手册页码。
- [TSB 术语]：parent 用「TSB」非标准术语；标准=TCB（trusted computing base）。本研究按「TCB 的 RSI 化转写」处理，是否独立命名由 human 定。
- [Thompson 引 4]：空军 Multics 评估文献 Thompson 本人无法定位（原文「Unknown Air Force Document」）；通认 Karger & Schell 1974 但无一手确认。
- [Saltzer-Schroeder 全文]：1975 原文未直读，原则措辞以 MIT 站点+Cornell 课件转述为准。
- [seL4 时点]：官方 proof statistics 为 2009 首版口径（2015 全部证明已 ~500k 行）；现状未深挖。
- [DDC 部署面]：除 GNU Mes 外的工业部署面未尽调。

## Gaps
- 「人类裁决层最小化/Thompson 道德律的 RSI 化用」是否采纳为设计原则＝价值判断——已上报 values.md 待裁决 14/15，待 human 裁决。
- TCB 定义移植 RSI 时「安全性质」的本体对应（正确性？价值对齐？精英解有效性？）未定义，需 human 拍板。
- 「每层抽象有税」在虚拟化有数字（6-8%/层），在 RSI/agent 栈无对应实证——需要的话另开调研。
