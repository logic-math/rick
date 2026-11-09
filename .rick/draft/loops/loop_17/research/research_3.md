# research_3 — z.ai GLM Infra Agent 机制级尽调（loop_17 第 7 课地基） — 2026-11-04

> 时间基准 2026-11-04。信源约定：A = z.ai 官方博客原文（2026-09-17，SPA 抓取全文约 21K 字符）+ GitHub PR/issue API 一手核验；B = 权威技术博客读图转述（explainx.ai）/NeMo 文档/SCMP/TechTimes；C = 一般媒体转述（不作结论依据）。博客为公司立场性叙述：事实层（下文各条）与立场层（文末待裁决候选）分列。承接 research_1 行 15/98/126 与 values.md 待裁决 4（已裁决，不重复裁决）。

## 事实性结论（五个尽调面）

### 面 1｜Infra Agent 组成
- **单 agent 形态**：Infra Agent = "an Infra Agent powered by GLM-5.3"，博客通篇单数，无多 agent/编排器/搜索算法字样；循环 = agent 提假设→改代码→跑实验→按反馈「保留/修改/否决」的单轨迹迭代。前提：仅博客外部叙述，agent 内部结构未载。来源：博客原文（A）
- **工具面 = 工程侧预建 dense feedback 环境**：正确性测试/运行时日志/执行 trace/运行时事件/microbenchmark/端到端指标被组织成可重复、agent 可直接调用的验证工作流；外加「并行策略→kernel 实现」映射，把系统级部署配置转成可逐个验证的 kernel 级任务。dense ≠ 多喂日志 = 局部/廉价及时/客观可验三性质。来源：博客原文（A）
- **知识留存件 = 优化骨架库（optimization skeletons）**：agent 从 SGLang/Flash Linear Attention/DeepGEMM 既有手写 kernel 提取优化技术及适用条件（applicability conditions/转换方法/资源约束/验证证据），经增量+消融实验蒸馏成骨架；新 kernel 从骨架出发，验证过的变更回写库。来源：博客原文（A）

### 面 2｜优化对象（serving 栈的哪些件）
- **落到具体件**：kernel（数值+性能）、Python-C++ 边界并发（GIL）、并行策略（Context Parallel/Layer Split）、量化（W8A8、INT8/FP8/BF16 混合精度 cache）、调度（async scheduling/层级 cache）、EPD 分离架构；正文未提编译器 flag 与路由层。来源：博客原文（A）
- **案例① KDA kernel CP 路径数值 bug（含独立核验）**：分片/非分片执行路径数值对比暴露差异→定位 state merge（M=tl.dot(M_chunk,M)；S_next=tl.dot(M,S)+H）精度→input_precision="tf32x3" 修复。博客称已合入 FLA 上游——实核 PR #1180（fla-org/flash-linear-attention）真实存在、2026-08-27 合并、标题 "[CP] use tf32x3 affine chain in kcp"、含 3 个 CP 测试，与博客描述逐条吻合；但 PR 由人（Clement Chan）提交、维护者（Zhiyuan Li）重做，agent 贡献在公开工件中不可分离。来源：博客+GitHub PR API（A 双源）
- **案例② GIL 持锁阻塞 KV Transfer（含上游旁证+校准注记）**：工程师定验收线 = Prefill+KV Transfer 与 Prefill-only 差 ≤5%；agent 实测 >20%；执行 trace 显示 Python 侧 KV Transfer 与 DeepEP intranode_dispatch/combine 区间从不重叠→根因 = 进入 C++ 不自动释放 GIL、Mooncake Transfer 线程被锁；修复（相关区间放 GIL）后 <1%。上游旁证：DeepEP PR #142（internode 放 GIL，2025-05-08 已合并，正文点名 Mooncake 场景）、PR #555（intranode 放 GIL，2026-01-04 提、至今未合并）。**校准注记**：该 bug 类社区 2025-04 起已知、intranode 变体 2026-01 已被公开提议——agent 属「重新推导已知 bug 类」而非首发现；博客仅称案例①合上游，未称案例②合上游。来源：博客+GitHub PR API（A）
- **案例③ KDA Decode kernel 三段演化**：v0→v1 ReplaySSM 以算力换显存（kernel 变慢）；v1→v2 除法优化 −9.6%；v2→v3 发现 V 维 tiling 使 FP32 归一化/门控重复 4 次→合并 tile 进单 thread block+中间结果寄存器驻留+warp 级归约 = 1.71×（以降并行度消除冗余计算）。来源：博客原文（A）
- **栈生态 = 跨公司开源组合**：SGLang（serving）+ FLA 的 KDA kernel（Kimi Linear 谱系）+ DeepEP（DeepSeek）+ Mooncake KV Transfer（Moonshot）+ 未具名国产加速器 >10 万卡；被服务模型 GLM-5.3-Flash = 320B 总参/18B 激活、45 层 = 34 KDA+11 sparse-MLA（3:1 混合）、mHC 四流残差、1M 上下文原生多模态。前提：架构比例出 B 级二手（NeMo/Raschka/flozi），320B/18B/1M 由多源印证。来源：博客（A）+二手（B）

### 面 3｜3.2× 吞吐测量口径（判真角度）
- **口径**：端到端 serving 吞吐；基线 = 新硬件首次成功运行时的 W8A8 初始状态（T+0=1.00×）；13 天至 3.22×；正文文字仅 "roughly 3×"/"tripling end-to-end throughput relative to the initial baseline"，3.22×/13 天/逐日轨迹出 Figure 1（Layer Split+Context Parallel 两步 1.41×→2.49× 贡献主体增益；2.67× 平台期 3 天）。前提：本会话模型不支持读图、无 OCR 工具，逐日表经 explainx.ai 单源读图转述。来源：博客正文 3×（A）+读图转述 3.22×（B）
- **判真要素零记载**：全部自报、无独立复测（10 万卡声明 CNBC/TechTimes 亦无法独立核实）；无重复测量/方差/显著性/环境控制（锁频/预热/隔离）任何记载；基线 = 未调优首跑（弱基线，正撞 values 待裁决 7 的 naive 基线立场）；agent 与工程师贡献无量化拆分（仅 "Much of the work"）。来源：博客未载即缺（A）+TechTimes（B）

### 面 4｜"the model optimizes the system; the system runs the model" 机制对应
- **族级自指、非实例级**：优化者 = GLM-5.3（agent 动力模型），被优化系统承载 GLM-5.3-Flash 全部生产推理（"All production inference for GLM-5.3-Flash runs on this system"）——优化者与被服务者是同家族不同变体；原句 "The model thus participated in optimizing its own inference system"；博客未称 agent 自身推理跑在被优化栈上；跨部署留存 = "the experience accumulated with each deployment reduced the engineering effort required for the next round of optimization"。来源：博客原文（A）

### 面 5｜人类边界与迭代形态
- **边界（原文口径）**：工程师三职责 = 定优化目标与系统约束／搭 agent 可直接用的反馈环境／审查涉数值语义-并发行为-生产风险的关键变更；"Choosing objectives, setting boundaries, and assessing risk remain human responsibilities"（自认未达 RSI）。来源：博客原文（A）
- **迭代形态**：13 天项目内单轨迹持续闭环（反馈循环贯穿全程）+ 骨架库回写；无 population/多样性/岛模型/多模型混合等 AlphaEvolve 部件——部件对照：proposer=agent ✓、verifier=分层反馈+人审 ✓、database=骨架库（弱形态：无分数驱动回浮、无淘汰、库的检索组织机制未载）；第二轮（系统优化自己刚改过的部分）未发表。来源：博客原文（A）
- **时间线交叉自洽**：Ox-Alpha 匿名上线 2026-08-20 → 官方揭晓 8-26/27（各源差一天）→ FLA PR #1180 合并 8-27（与揭晓日重合）→ 博客 9-17 回溯叙述；「13 天」精确日历区间未载。来源：博客+多源（A/B）

## 立场层（新断言候选——本次仅落盘本文件，未追加 values.md，请 parent 按三栏格式转录为待裁决 8/9，human 裁决栏留空）
- **待裁决 8 候选·NVIDIA 可比性**："Both hardware utilization efficiency and per-token cost reached levels comparable to mainstream NVIDIA GPUs."——无数字支撑、未定义「主流 GPU」口径、无独立复测；竞争性等价声明（官方一手 A，但为立场句）。
- **待裁决 9 候选·进度不可挡+长期守线**："But the numbers, two weeks, threefold throughput, and 100,000 accelerators, tell us that progress at this boundary will not slow down simply because we want it to."（前句 "We believe humans should continue to hold that line for a long time to come."）——n=1 自报样本外推 + 规范性主张（官方一手 A，但为立场句）。

## R7 上报项
- [Figure 1 读图]：3.22×/13 天/逐日轨迹与各日命名优化，本会话无法目验（模型不支持图像、无 OCR）——缺：human 目视核验 Figure 1，或第二家可信读图转述交叉（现仅 explainx.ai 单源 B）。
- [agent 宿主]：Infra Agent（GLM-5.3）自身推理是否跑在被优化栈上——博客未载——缺：官方口径。
- [骨架库机制]：库的检索/组织/淘汰/规模——博客未载——缺：官方技术细节或后续论文。
- [硬件与成本]：加速器型号；「NVIDIA 可比」的利用率/per-token 成本数字——官方未具名未发布，CNBC 无法核实——缺：独立审计或官方数据。
- [贡献拆分]：agent 与工程师工作量占比——无量化——缺：官方拆分口径。

## Gaps
- 测量判真协议全缺（重复测量/方差/显著性 Welch t·Cohen's d/锁频预热隔离零记载）——第 7 课只能教「自报+弱基线+无公开协议」，无法教其测量方法（对照 CompileIQ 博客完整协议，反差极大）。
- Infra Agent 内部结构（工具调用接口形态、上下文管理、有无 sub-agent/自动编排、单次实验 token 成本）未载——只知外部循环形态。
- 3.22× 的负载口径（并发/请求分布/上下文长度/是否含 1M 与多模态负载）未载。
- 发布后状态未发表：agent 是否持续在生产系统上闭环优化（持续 RSI vs 一次性项目）；explainx 亦点名「第二轮才是 RSI 真判据，Z.ai 未发表」。
- 与 AlphaEvolve 对照缺件：多样性/岛模型/多模型混合/分数驱动回浮全无——教学按「退化形态程序数据库」处理骨架库，防止讲成 MAP-Elites。
