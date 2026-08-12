# RFC — rick→pi 迁移的价值基础与架构定位 — 2026-08-10

## 主题

固化 sense_loop loop_4 推导结论——rick 为何存在（价值基础）与 rick 是什么（架构定位=引导程序）。本 RFC 严格基于 human 在 `judgment.md` 中的原话与 `briefs/` 中的事实陈述，不补充未确认内容，不替 human 决策 R7 上报项。

## 完成日期

2026-08-10（sense_loop loop_4 全流程完成日：S→E→N→S-R→EC，EC 维持/良质通过）

## 哲学基础

human 在 E 阶段确立两条价值性假设原话：

> 「全知不代表全能，而全知与全能必然存在有限性。智能在某一时刻必然是有极限的。」（YE1，`judgment.md` E-r2）

> 「扩展心智的本质就是提供某种确定性的信息存储方法，以便于确定性的信息提取而非不确定性的信息提取……LLM 的参数权重本质上是一种非确定性的信息压缩，提取时也会具有一定的损失。」（YE3，`judgment.md` E-r2）

要点提炼：

- **智能有极限**：全知≠全能；智能在某一时刻必然有极限（`briefs/批判门禁-E-r5.md` A1，节点 D 侧面支撑——单轮对 G′ 不足即智能极限的工程体现）。
- **扩展心智=确定性存储/提取**：人类工作记忆区有限（无限工作记忆区不产生智能——会引入无关信息使思想无价值），故需外部确定性信息存储；模型亦如此（YE3）。
- **LLM 参数权重=有损+非确定压缩**：A7 CONFIRMED。research 多源印证——Tishby 信息瓶颈（学习=通过瓶颈的有损压缩，泛化=遗忘训练细节）、Delétang（LLM=compressor）、Self-RAG 确认 "sole reliance on parametric knowledge 导致 factual inaccuracies"（来源：`briefs/research-report-EC.md` 节点 C，复用 E-r2 节点 A/B；`briefs/research-report-SR.md` 阻碍识别 X）。

## 核心假设

human 在 E-r4 明确核心假设原话：

> 「最核心的假设在于，存在一个 G 外的问题集合 G′ 它是 LLM 没有见过的问题，也是它无法一次性解决方法问题。因此需要迭代，需要在实验探索中解决问题，故而需要 rick 存在。」（E-r4，`judgment.md`）

**形式化**：∃G 外 G′（LLM 未见过 + 无法一次性解决）→ 需迭代/实验探索 → rick 存在。

**依赖链（全 CONFIRMED）**（来源：`briefs/批判门禁-E-r5.md` "核心假设最终审"；`briefs/research-report-E-r4.md` 节点 A/B/C/D）：

- **A7（CONFIRMED，r3）**：LLM 权重=有损+非确定压缩 → 单次提取不可靠。置信度 0.85。
- **A5（归纳）**：训练数据时效性 → G′ 持续存在。
- **A15（CONFIRMED，r5）**：通用 LLM zero-shot 不选 rick 编排 → rick "确定性选择（含判断）" 是关键差异。research 节点 B（runtime zero-shot 线性单遍不选编排，0.55 R7 真理性强）+ 节点 C（"不可跳过"+确定性拼装=强制执行，1.00 高）。置信度 0.85。
- **A18（CONFIRMED，r5）**：单轮对 G′ 不足，多轮改善 → 需迭代框架。research 节点 D（单轮对 G′ 事实拒答=100% 失败 + 文献四源 "单轮不足多轮改善"，0.90 高）。置信度 0.85。

**边界 nuance**（来源：`briefs/research-report-E-r4.md` 节点 D；`briefs/批判门禁-E-r5.md` A18 边界）：迭代非银弹——"Sample More, Reflect Less"（arxiv 2607.28576）示同等 token 成本下重复采样可超 self-refine/Reflexion；多轮策略有优劣。但 "单轮→多轮改善" 方向性结论稳固（四源 + runtime 印证）。

## rick 价值论

human 在 E-r4 接受 D3′ 价值重锚原话：

> 「这是最终交付方法价值，应对上下文的熵增这件事是实现这一价值主张的手段。」（E-r4 D3′，`judgment.md`）

**最终价值主张**（D3′ 重锚，来源：`briefs/批判门禁-E-r5.md` "rick 价值论最终形态"）：

- **价值（主体）=弥补参数记忆有损+非确定**：LLM 内禀属性，与训练成本正交，刚性。A7 CONFIRMED 支撑。
- **手段=应对上下文熵增**：compaction-resist system prompt + 文件确定性载体 + 信息网络流。
- **实现机制=确定性编排 + 强制执行 + 含判断的选择 + 迭代框架**：doing.md "不可跳过任何步骤" + doing_loop Step 0-5 全程 "必须/强制/自动触发" + doing_prompt.go 确定性拼装落盘 + Step 0.2 trigger 匹配 + think 假设打分 + sense+doing loop 3 轮上限。A15/A18/节点 C CONFIRMED 支撑（来源：`briefs/research-report-E-r4.md` 节点 C；`briefs/research-report-N1.md` node rick）。

**EC nuance（手段层部分弱化，见 EC nuance 节）**：价值主体锚定 A7+A15+human 判断者；"应对上下文熵增" 作为手段在无限上下文下价值部分下降（来源：`briefs/research-report-EC.md` 节点 D + 整合摘要）。

## 架构定位

human 在 E-r4 D3 与 YE5 确立架构定位原话：

> 「定义成引导程序，他会引导人类以正确的模式工作，以及引导 pi 加载不同的系统提示词。」（E-r4 D3，`judgment.md`）

> 「为了防止命令抽象的冲突，我们选择给 pi 套一个引导的外壳。它的存在就是为了抽象出 rick 的执行工作流，约束用户以 rick 的做事方法与 ai 进行交互。」（YE5，`judgment.md` E-r2）

> 「rick 是解决 G 外问题的方法，pi 是他的实现方式……遵循 sense 做事方法的 pi 就是 rick。」（YE5，`judgment.md` E-r2）

**最终形态**（D3 双职责，来源：`briefs/批判门禁-E-r5.md` "架构定位最终形态"；`briefs/research-report-N1.md` node rick + edge human↔rick）：

- **rick=引导程序（双职责）**：
  - 引导人类以正确模式工作（pi 不可内化：pi 是 agent loop 不面向人类工作流）。
  - 引导 pi 加载不同系统提示词（按模式注入）。
- **价值主体=rick**：引导人类（不可内化）+ 确定性提取架构（pi 原生没有）+ 确定性选择+强制执行（zero-shot→可靠工程的关键跃迁）。
- **"可完全内嵌 pi"≠"功能不可替代"**：rick 可内嵌（pi 调用 rick 的提示词/文件编排层，如 `pi rick-plan`），但提供 pi 原生没有的确定性提取结构 + 引导人类的工作流约束；独立进程存在理由=防止命令抽象冲突（`pi rick-plan` 不优雅，故套引导外壳）。

## 主要矛盾

human 在 N2 选定主要矛盾原话：

> 「核心矛盾就是对模型输入的可控性。」（N2，`judgment.md`）

**选定**：M3-ext——对模型输入的可控性（rick 确定性输入控制 vs LLM 内禀输出非确定+回退/震荡/局部最优）。

**N2 打分事实**（来源：`briefs/批判门禁-N2.md` M1-M8 打分表 + top-N）：think 3 维打分（根本性/全局性/决定性，满分 3.0），M3 / M3-ext / M2 并列 top-1（各 3.0）；human 选定 M3-ext（即 top-1，与 M3/M2 并列）。

- **M3（原形，能力层，3.0）**：确定性提取/强制执行（rick）vs LLM 参数记忆有损+非确定（内禀，A7 CONFIRMED）。结构性矛盾，永久存在（rick 弥补但不消除）。
- **M3-ext（扩展形，3.0，human 选定）**：输入可控+失败模式管理 vs 输出非确定+回退/震荡/局部最优。M3 + D2 失败模式管理扩展——把输出侧的回退/震荡/局部最优纳入矛盾右极。
- **M2（架构层，3.0，并列 top-1）**：rick=方法（可完全内嵌 pi）vs rick=独立引导程序（引导人类，pi 不可内化）。N2 审查发现：M2 与 M3 同量级根本——rick 的本质定义（方法 vs 独立进程）与确定性 vs 非确定同等根本。

## 控制手段

human 在 N1-r1 确认控制手段原话：

> 「主要的控制手段就是治理上下文的熵增。」（N1-r1，`judgment.md`）

**治理上下文熵增**——确定性提取 + 强制执行 + 迭代框架 + 门禁（来源：`briefs/research-report-N1.md` edge + 稳态分析；`briefs/research-report-SR.md` 逆转层 1 rick 现有机制）：

- 确定性提取：ContextManager 从文件加载（LoadOKR/SPEC/Debug/History）+ GenerateDoingPromptFile 注入 + SaveToFile 落盘。
- 强制执行：doing.md "不可跳过任何步骤" + doing_loop Step 0-5 全程 "必须/强制/自动触发"。
- 迭代框架：doing_loop Step 3 Sub Agent per-iteration + 3 轮上限 + DEBUG Phase 1-6。
- 门禁：check 门禁（runAutoFix 循环直到 pass）+ sense 批判门禁（假设打分+top-N 阈值<0.3 不入选）+ human 判断（judgment.md）。

## 收敛机制

human 在 N1-r1 提出，N1-r2 弱化（如实记录弱化轨迹，不粉饰）：

> 「有序的上下文能够使智能在迭代过程中向着最优解的方向有效递进。」（N1-r1，`judgment.md`）

N1-r1 表述为 "向着最优解的方向有效递进"（隐含单调趋优）。N1 门禁 r1 追问后，human 在 N1-r2 主动弱化：

> 「最大化改进……可能存在回退，震荡，局部最优。但这就更加说明，rick 的价值。」（N1-r2，`judgment.md`）

**弱化轨迹**（来源：`briefs/批判门禁-N1-r1.md` + `briefs/批判门禁-N1-r2.md` + `briefs/research-report-N1.md`）：human 从 "向最优解有效递进"（单调趋优）→ "最大化改进"（非单调，承认回退/震荡/局部最优）。收敛机制不保证单调趋优，而是 "有序上下文→最大化改进"，改进非单调，需失败模式管理（回退/震荡/局部最优纳入预期并设停止标准）。

**最终收敛机制**：有序上下文→最大化改进（非单调，含失败模式管理）。失败模式管理=价值扩展（D2）：回退/震荡/局部最优反而强化 rick 价值（human "更加说明 rick 的价值"）。代码证据：doing_loop Step 4 "失败→返回 Step 3 下一轮" + Step 5 "连续 2 轮产出相同错误=无法自动收敛"（回退/震荡/局部最优的代码承认）+ DEBUG Phase 4 上限 3 次后升级人工 + 3 轮上限（防无限震荡）。

## 逆转逻辑（三层）

human 在 S-R 完全接受逆转逻辑原话：

> 「完全接受，非常好。」（S-R，`judgment.md`）

**逆转起点**（来源：`briefs/research-report-SR.md` 阻碍识别）：X（必然前提/阻碍）=LLM 输出有损+非确定（A7 CONFIRMED 内禀，不可消除）；Y（期望）=有限迭代最大化改进/可靠解决 G′。直觉路径 "消除 X→可靠 Y" 阻塞（X 不可消除）→ 必须逆转逻辑。

**逆转的三层结构**（来源：`briefs/research-report-SR.md` 逆转逻辑；尽调置信度 0.90 高）：

- **层 1 — 可控性转移（输出侧→输入侧）**：输出侧（LLM→output）A7 内禀不可控→放弃追求输出确定性；输入侧（rick→LLM，经 pi→LLM edge）可控→rick 确定性提取（ContextManager 从文件加载）+ 确定性拼装（GenerateDoingPromptFile + SaveToFile 落盘）+ 强制执行（doing.md "不可跳过任何步骤"）。逆转：把 "可控" 全部押在输入；输入确定+输出非确定→输出方差被确定的输入 "锚定"。
- **层 2 — 非确定吸收（迭代+失败模式管理）**：用迭代吸收方差——doing_loop Step 3 Sub Agent per-iteration + Step 4 失败→返回 Step 3 + 3 轮上限（防无限震荡）。回退/震荡/局部最优管理：DEBUG Phase 1-6（遇红强制触发，Phase 4 上限 3 次后升级人工）+ check 门禁（runAutoFix 循环直到 pass）+ sense 批判门禁 + human 判断。逆转：迭代非 "重复直到对"，而是 "用确定的框架管理非确定的输出"。
- **层 3 — 非确定转化（阻碍→推动力）**：输出非确定=多样性源→多次采样+选择=改进机制（呼应 "Sample More, Reflect Less" arxiv 2607.28576）。非确定→探索 G′ 的多个可能解→check 门禁+sense 批判+human 判断筛选→收敛到 "最大化改进"。逆转：非确定非 "要消除的噪声"，而是 "探索 G′（未见过问题）所必需的变异源"——G′ 本质未见过，确定性地输出反而可能错；非确定地探索+确定性门禁筛选，比 "强行确定" 更适配 G′ 的开放性。

**rick 现有机制填补逆转**（来源：`briefs/research-report-SR.md` "rick 现有机制如何填补逆转"，置信度 0.9-1.0）：层 1 确定性提取+强制执行（1.0）/ 层 2 迭代框架+失败模式管理（1.0/0.9）/ 层 3 compaction 抗熵增+探索性采样（1.0/0.9）。

## 替代路径

human 在 S-R 完全接受替代路径 P1–P6（含利弊并陈，不推荐，供 human 选）（来源：`briefs/research-report-SR.md` 替代路径可选项）：

- **P1 compaction 策略**（治理上下文熵增，承载 pi↔LLM edge）：P1a 自定义 compaction（session_before_compact + customInstructions + firstKeptEntryId + 保留 system prompt，作 compaction-resist 载体）/ P1b 默认 auto-compact（rick 不控制）。
- **P2 迭代框架**（吸收输出非确定，承载 doing_loop+pi→LLM edge）：P2a rick sense+doing loop（确定性编排+失败模式管理+门禁+停止标准）/ P2b Self-Refine（arxiv 2303.17651，通用但无外部门禁）/ P2c Reflexion（arxiv 2303.11366，多轮反思但无确定性提取）/ P2d 重复采样（arxiv 2607.28576，最简但无结构化收敛）。
- **P3 RAG vs 上下文工程**（确定性提取的不同实现路径，承载 rick↔外部存储+LLM↔外部存储 edge）：P3a RAG（arxiv 2005.11401，通用但检索非确定）/ P3b rick 上下文工程（文件载体+ContextManager 确定性加载+强制注入+compaction-resist）。
- **P4 skill 系统级注册**（提升确定性触发概率，承载 LLM↔外部存储 edge）：P4a skill 系统级注册（系统级触发，非依赖 LLM 选择）/ P4b prompt 文件路径引用（现状，触发概率低）。
- **P5 subagent 递归**（分层迭代，承载 doing_loop Step 3 Main→Sub）：P5a pi subagent 递归（每轮独立 Sub Agent，上下文隔离）/ P5b 单 agent 无递归（现状，无上下文隔离）。
- **P6 二进制部署**（控制手段的部署形态，承载 rick↔pi edge 运行时形态）：P6a pi 二进制编译（自包含部署，脱离 node）/ P6b 依赖 node（现状）。

**路径关系**（来源：`briefs/research-report-SR.md` 路径关系说明）：P1/P2/P3 直接对应逆转三层；P4/P5/P6 是支撑性控制手段。各路径的 a 选项多为稳态 B（目标），b 选项多为稳态 A（现状）。

## EC nuance

human 在 EC 自判颠覆性假设 + 良质通过原话：

> 「我认为最不安的假设是 如果模型的上下文足够长，模型智能足够强。是否就不需要管理上下文了？……如果存在一个无限度上下文，自我可以不断迭代直到任务完成。那还需要 rick 吗？」（EC 自判，`judgment.md`）

> 「保持现状，良质判断通过」（EC，`judgment.md`）

**research 结论**（来源：`briefs/research-report-EC.md` 整合摘要 + 对 human 请求的直接回答）：rick 仍需要，核心价值不 collapse（4 价值全 HOLD）；仅 A17 手段层部分弱化 + A18 边界细化。跃迁方向=维持。不建议触发反向回流。

- **COLLAPSE（部分，手段层）**：A17 的 "手段=应对上下文熵增" 部分弱化——context 趋向 effectively very long（2M+）+ auto-compaction（Codex 服务端）自动化熵管理→rick 的 compaction-resist system prompt 手段价值相对下降。这是手段层弱化，非价值主体弱化。
- **PRESERVE（核心，价值主体层）——4 价值全 HOLD**：
  1. **A7 参数级弥补（价值主体）HOLD**：A7 是参数级（权重）有损+非确定；上下文是非参数级——上下文长度不修复权重有损。与上下文长度正交，不 collapse。
  2. **A15 确定性编排/强制执行 HOLD**：zero-shot LLM 线性单遍不选 rick 编排，无限 context 不改变 "不选"；doing.md "不可跳过" 强制执行仍需。与上下文长度正交。
  3. **human 判断者不可替代 HOLD**：G′ "完成"/"最大化改进" 由 human 认定；LLM 无法自证 G′ 解决（不知道自己不知道；无限 context 不注入不存在的 G′ 知识）。
  4. **失败模式管理（SR 层 2）HOLD**：回退/震荡/局部最优在无限上下文下仍存（不收敛性 + Lost in the Middle arxiv 2307.03172 长上下文退化）。

**关键辨析**（来源：`briefs/research-report-EC.md` 节点 C）：human "压缩=遗忘" 洞察对非参数级（context）成立——compaction/pruning 是 lossy。但无限上下文不避免损失：(1) 非参数级——避免压缩级遗忘但不可达；长 context 有检索/注意力级损失（Lost in the Middle）；遗忘移位未消除。(2) 参数级（A7）——完全不被上下文长度触及，权重有损+采样非确定持续。遗忘从压缩级移到检索/注意力级，参数级损失不动——总损失未消除。

**建议的 refinement（非回流）**：A17 手段重锚——将 rick 价值主张更牢固地锚定在 A7（参数级有损，内禀）+ A15（确定性选择/强制执行）+ human 判断者，而非 "应对上下文熵增"。A18 边界细化——对真正 G′（未见过）单轮不足 HOLD；对 bounded G′（语料内+有界）部分弱化。

## 派生修订需求

human 在各步确认的修订点（来源：`briefs/批判门禁-E-r5.md` 收敛结论；`briefs/批判门禁-N1-r{1,2}.md`；`briefs/research-report-EC.md` refinement）：

- **D1（E-r4）核心价值重述**：有限迭代**最大化改进**（非单调，含回退/震荡/局部最优，需失败模式管理）。统一存储/推理维度（推理极限=工作记忆区局限延伸，research 信息瓶颈支撑）。
- **D2（E-r4）失败模式管理=价值扩展**：回退/震荡/局部最优纳入矛盾右极（M3-ext）；"更加说明 rick 的价值"。
- **D3（E-r4）架构定位**：rick=引导程序双职责（引导人类正确模式[pi 不可内化]+引导 pi 加载系统提示词）；价值主体=rick。
- **D3′（E-r4）价值重锚**：价值主体锚定 A7（弥补参数记忆有损+非确定，与训练成本正交，刚性）；手段=应对上下文熵增。
- **EC A17 手段重锚**：价值主体锚定 A7+A15+human 判断者（非 "应对上下文熵增"，后者手段层弱化）。
- **EC A18 边界细化**：对真正 G′（未见过）单轮不足；对 bounded G′（语料内+有界）承认弱化。

## 遗留逻辑漏洞（R7）

以下均为**待 human 决策**（不替 human 决策；来源：`briefs/批判门禁-N2.md` 审查；`briefs/research-report-EC.md` R7；`briefs/research-report-E-r4.md` 节点 B；`briefs/research-report-SR.md` R7）：

1. **M8 迭代策略优劣（R7/未决）**：rick sense+doing loop 迭代框架 vs Self-Refine/Reflexion/重复采样的最优性未 benchmark 验证（A18-Q2 未验证）。"Sample More, Reflect Less"（arxiv 2607.28576）示重复采样可超结构化反思，rick 迭代框架的相对有效性是开放问题。不影响逆转逻辑方向性（单轮→多轮改善稳固），但影响 P2a vs P2b/c/d 的选择。（来源：`briefs/批判门禁-N2.md` 审查 1；`briefs/research-report-SR.md` R7 上报项）
2. **A7 节点 A/B R7（LLM 内部机制源码不可访问）**：LLM 权重有损+非确定的多源交叉印证（Tishby/Delétang/Self-RAG/runtime 采样非确定），但 LLM 内部机制源码不可访问，置信度受信源可达性上限约束（0.85 而非 1.0）。（来源：`briefs/research-report-E-r4.md` R7 共性说明）
3. **A15 节点 B R7（LLM 决策源码不可访问）**：zero-shot 是否选 rick 方法——runtime demo 线性单遍不选编排 + 三 arxiv 主源印证，真理性强，但 LLM 决策源码不可访问 + paired rick-injected demo 不可运行，置信度 0.55（受信源可达性上限约束，非结论存疑）。（来源：`briefs/research-report-E-r4.md` 节点 B R7）
4. **EC 节点 A/B R7（无限上下文+强模型）**：节点 A "无限上下文可行/可达" 0.20 低；节点 B "能否消除上下文管理需求" 0.55 中。LLM 注意力/检索/淘汰源码不可访问，置信度受信源可达性上限约束；真理由 docs（多 arxiv 主源：Lost in the Middle/RULER/Self-RAG/context pruning）+ runtime 复用已充分确立。（来源：`briefs/research-report-EC.md` R7 上报项）
5. **human 判断者不可替代性（N2 提示，未纳入主要矛盾）**：think N2 审查 3 提示 "human 判断者的不可替代性"（系统若无 human 判断，G′ 的 "最大化改进" 由谁认定）是未被 M1–M8 覆盖的潜在根本矛盾。human 选定 M3 时未将其纳入为新矛盾。EC 验证其 HOLD（4 价值之一），但未纳入主要矛盾表述——待 human 决策是否补入。（来源：`briefs/批判门禁-N2.md` 审查 3 + 启发性追问 3）
6. **M7 rick 长期形式（长期开放变量）**：M7 "可完全内嵌 pi vs 功能不可替代" 概念上 D3 已化解（形式 vs 价值），但长期演化未关闭——若 pi 社区未来原生实现 rick 式确定性提取结构（ContextManager+doing.md 强制注入+compaction-resist），rick 进程的 "形式"（独立进程）失去最后一项存在理由（防命名冲突），M7 左极完全实现，rick 进程消失。当前不阻塞 A→B，但是 rick 长期存在形式的开放变量。待 human 决策长期形式定位。（来源：`briefs/批判门禁-N2.md` 审查 2）

## SENSE 流程记录

sense_loop loop_4 全流程客观记录（来源：`judgment.md` 各阶段门禁结果）：

- **S（loop_2 已收敛）**：现状=rick+ai_cli+claude code；期望=迁移 pi+深度定制（二进制/skill 系统级/自定义 compaction/subagent 递归）；差距=缺具体实现计划。S 阶段在 loop_2 收敛，本会话从 E 恢复点继续。
- **E（批判门禁 r1→r5，r5 ✅ 通过，重试 5/5）**：5 个未澄清 Y（Y-E1 涌现推理/智能极限 / Y-E2 元方法稳定性 / Y-E3 rick 方法 G 内外 / Y-E4 G 过去式刚性 / Y-E5 rick/pi 不可替代边界）全部澄清转正。并行 research 两轮：E-r2（验证 "LLM 权重有损压缩+下游推论"，CONFIRMED）、E-r4（zero-shot 对比，A15/A18 CONFIRMED）。核心假设稳固；rick 价值论（D3′ 重锚）+ 架构定位（D3 双职责）最终形态确立。分析见 `briefs/批判门禁-E-r{4,5}.md`、`briefs/research-report-E-r4.md`。
- **N1（批判门禁 r1→r2，r2 ✅ 通过，重试 2/5）**：4 项澄清——核心价值重述（有限迭代最大化改进，非单调）/ 主要矛盾选定方向（输入可控+失败模式管理 vs 输出非确定+回退/震荡/局部最优，M3 方向扩展）/ 控制手段（治理上下文熵增）/ 收敛机制（有序上下文→最大化改进，非单调，含失败模式管理）。残余非阻塞：N10（失败模式管理=价值扩展，溯因低置信，机制 confirmed 但 "比无序更能管理" 未 research 验证）。分析见 `briefs/批判门禁-N1-r{1,2}.md`、`briefs/research-report-N1.md`。
- **N2（think 3 维打分 M1–M8，human 选定 M3-ext）**：top-N=3（M3/M3-ext/M2 并列 3.0）。"看似次要实则根本" 审查：M8（underpins D2 失败模式管理价值扩展）/ M7（长期形式未关闭）/ M5（尾部风险）+ 提示 "human 判断者不可替代性" 未覆盖。human 选定 M3-ext，未纳入 M2 或新矛盾。分析见 `briefs/批判门禁-N2.md`。
- **S-R（human "完全接受，非常好"，跳过门禁）**：research 逆转逻辑尽调（三层结构，置信度 0.90）+ 替代路径 P1–P6（含利弊，不推荐）。human 完全接受三层逆转逻辑 + 替代路径。门禁：跳过（human 为纯确认性语句，按协议跳过门禁）。分析见 `briefs/research-report-SR.md`。
- **EC（并行 research 颠覆性假设调查，4 价值全 HOLD；human 自判良质通过，跃迁方向=维持）**：human 自判最不安假设（无限上下文+强模型→rick 是否还需要），显式要求 "详细调查+证实与反驳观点"。research 结论：rick 仍需要，核心价值不 collapse（A7 参数级弥补/A15 确定性编排/human 判断者不可替代/失败模式管理 全 HOLD）；仅 A17 手段层部分弱化 + A18 边界细化。不建议反向回流。human 自判良质通过，跃迁方向=维持。分析见 `briefs/research-report-EC.md`。

**全流程完成**：S（loop_2 已收敛）→ E（r1→r5 ✅）→ N1（r1→r2 ✅）→ N2（选定主要矛盾）→ S-R（human 完全接受，跳过门禁）→ EC（维持，良质通过）→ exporter 阶段一（大纲 human 确认）→ exporter 阶段二（本 RFC）。
