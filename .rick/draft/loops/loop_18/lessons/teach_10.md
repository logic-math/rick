# 第 10 课｜GEAK e2e_workflow：生产级端到端优化＋进化知识库（模块 3 收口）

> 输入：syllabus 模块 3 第 10 课（收口课）｜research/research_5.md（下称 R5；信源：A＝AMD 官方技术文章＋ROCm 文档站＋GEAK repo 代码原文＋AgentX 官方博客；谱系关键发现：arXiv 2507.23194＝v1，e2e_workflow＝v4，引用分开标注——§6）｜zpd.md 第 5-9 课 consolidate＋加速偏好（快收口大步推进；p/d 跨课衰退记号——本课噪声带处轻探一次）｜judgment.md 第 10 课认知强制两答（切入锚）＋裁决 12/13 已接受（立场层引用，不作事实教）｜已会锚直引：Amdahl 边界（第 8 课 f(1−1/s)＞δ）／留存家族（第 3 程序数据库·第 7 骨架库·第 9 Booster Pack）／判真三要素（第 2 课）／#11 深度梯（第 7 课 infra 格）

## 0｜从你的两句话切入（认知强制裁决位）

**a 的裁决位**（优化对象换成整个 serving 端到端，版图有什么不同）：你的原话——「将这些全部联合在一起优化；越是高层次的优化，对端到端加速就越明显；最大的优化空间＝全部维度联合排列组合。」两条直觉都是真的，而且恰好对应 GEAK 的两个真实设计件，本课把你逐条钉到机制上：

- 「层次越高越明显」＝**Amdahl 优先分公式**。你第 8 课的 Amdahl 边界形态是反面（f(1−1/s)＞δ——单点再快也救不了端到端）；本课是它的正面形态：GEAK 给每个候选优化算一个优先分 `pct_gpu_time × achievable_speedup`——**占比大 × 可达加速高才值得动**（R5 面1，A）。你说的「层次越高越明显」翻译过来就是：高杠杆处的 `pct_gpu_time` 大，同样 2× 的 kernel 加速在高占比处乘出来的端到端收益完全不同。这正是「层次直觉」的形式化。
- 「全部维度联合排列组合」＝**cheapest-first 三档**。GEAK 确实联合搜（flag→env→源码→author 四种形态一起排进搜索面），但补一刀：**联合不是无差别枚举**。排列组合爆炸下，GEAK 的答案是「Amdahl 公式先排序，再按成本升序花预算」——flag 最便宜先试，env 次之，源码替换贵，author from scratch 最贵最后（§1 详拆）。GEAK 比你说的「排列组合所有可能性」更省：先算优先分，再决定哪个维度值得枚举。

**b 的裁决位**（agent 反复优化不同批次部署，什么在跨部署留存）：你的原话——「code 执行源码＋冻结的镜像。」你答的是**交付物形态**：GEAK 真实交付物恰好是这东西的工程版——`final_patch.diff`（源码）＋`final_launch.sh`（自包含启动配置）＋可复现环境（模型权重与已装包全程只读＝「冻结」的那半）（R5 面4，A）。你答对了交付物的一半。缺的是**学习维度**：你回答的是「部署 N 发出去什么」，本课补「部署 N+1 的起点为什么比 N 高」——GEAK 的**进化知识库三物件**（§5：静态参考库／蒸馏经验卡／程序库面）。留存机制家族第四次登场（第 7 骨架库、第 9 Booster Pack 之后），GEAK 做到机制最完整一档。

## 1｜版图：优化对象、Amdahl 优先分、三档杠杆

先钉优化对象（R5 面1，A）：**别人的开源引擎栈的 serving 配置＋后端选择＋内核**——输入是磁盘上的模型＋serving backend（sglang|vllm|atom）＋真实工作负载（ISL/OSL/concurrency），**不含引擎编排/调度器本身的改写**。用你第 7 课的 #11 深度梯说：它落的是 infra 格里的「算子＋部署工作流」层，不是「运行时/调度」层——层选择本身就是设计决策（§7 费曼件回收这一点）。profiling 与 benchmark 用同一负载，保证瓶颈判定和收益计量可比。

三档杠杆 cheapest-first（按成本升序，R5 面1 A）：

| 档 | 形态 | 例子 | 成本 |
|---|---|---|---|
| ① launch flags | 启动参数 | --attention-backend、--quantization | 最低 |
| ② env vars | 环境变量 | SGLANG_USE_AITER、VLLM_ROCM_USE_AITER | 低 |
| ③ source | 源码替换/重写 | Triton/FlyDSL/HIP/CK/asm/TileLang 重写＋rebind 调用点 | 高 |
| ④ author | 从头写 | 热 op 无现成实现时全新基线 | 最高 |

**优先分公式怎么用**：System Architect 拿 Profiler 的 Top-N 表，对每个候选优化算 `pct_gpu_time × achievable_speedup`，排出里程碑计划——先吃「占比大且可达加速高」的果子，便宜的档先走。

留白三（你来推）：**公式里的 `achievable_speedup` 怎么估？**提示一：GEAK 用 roofline 做 advisory 路由——saturated 内核导向「减少字节搬移」而非「提并行度」的改法。提示二：R5 面5 记了一张反例卡——「roofline 预测 attainable=1.0 但实测 +26.9%」，launch-overhead 型收益对 roofline 不可见，这反例本身已入蒸馏卡。你来推：roofline 估的是什么（计算/带宽上限）？为什么 launch-overhead 型收益在它视野外？这个反例入卡意味着 advisory 先验的什么性质？（答案方向在 §5 的 add-only 契约处收。）

数字给满（AMD 性能实验室 2026-07-08，MI300X/MI355X 8 卡，Opus 4.8 单流，R5 面1 A——**全部 AMD 自报，无第三方复现**，§6 判真处再压）：e2e +60% Qwen3.5-27B-FP8（SGLang）、+96% Qwen3-14B-FP8、+42.2% Minimax-M3-MXFP8、+25.4% Kimi K2.6 INT4、+47%/+108% DSV4 Pro FP4。对照第 9 课量级规律：单层编译器搜索只有 2-7%，这里两位数——**高杠杆处的多档联合**正是你 a) 直觉的实证（但注意是自报口径）。

## 2｜控制面 × 判断面：7 专职角色与分形两层

架构主件（R5 面1，A）：**确定性 JS 控制面 × LLM 判断面分离**。JS workflow 拥有预算循环、并行 fan-out、验证、递归、停止条件——一切「不该交给概率性判断」的都钉死在确定性层；LLM agent 只做结构化技术判断（分析/策略/内核 authoring/集成）。结构图：

```
确定性 JS 控制面（预算循环 / 并行 fan-out / 验证 / 递归 / 停止条件）
  └─ 编排 7 个专职 LLM agent（各一段结构化技术判断）
       e2e Director     ─ 隔离评测目录；记录真基线（含噪声带）；终判独立复测仲裁
       System Architect ─ Amdahl 路由；里程碑计划；知识策展
       Profiler         ─ warm-server trace → 标准化 Top-N 表
       Config Tuner     ─ flag / env 档扫荡
       Kernel Extractor ─ 真实 shape＋参考 I/O → 不可变 standalone unittest
       Op Benchmarker   ─ 后端 bake-off
       e2e Integrator   ─ 可逆 overlay 回接＋e2e gate
流水线：Setup/Preflight → Profile → Strategize → ConfigSweep → HeadKernel
         → Milestone loop → Finalize/Report/Validate
（每接受一个改动 → 复合进 carried-forward overlay+config → re-profile，瓶颈移位即重排）
```

三预算模式：fast（仅 HeadKernel）／default（全流程，每接受改动一次 e2e gate）／deep（跨内核×后端 lane 池＋reseeding＋收敛停机）。

**分形两层**（本课最漂亮的一件）：e2e 层把热内核递归交给**原封不动的 kernel_workflow**——传不可变 oracle＋目标语言＋shape/dtype＋regime＋预算，收 verified patch（R5 面1，A）。你第 6 课的 correct-but-slow 传播恐惧在这里有机制答案：kernel 层是**质量不回归**的成熟件（向后兼容、单内核直通），system 层调它时不改它——两层递归不放大质量风险，因为被递归的下层自带 verifier 契约。这与 Harness² 的两级递归（第 14 课见）形似神异：GEAK 分形的是「确定性控制面 × 成熟子工作流」，不是 harness 自改写。

## 3｜verifier 全套：interleaved A/B＋噪声带＋分布 gate（判真第四档）

端到端吞吐怎么测——这是 GEAK 判真配置最厚的一段，逐件拆（R5 面1，代码+方法卡 A）：

| 件 | 机制 | 对应你第 2 课判真三要素 |
|---|---|---|
| 基准 harness | bench_e2e.sh，backend 无关，Director/Profiler/Tuner/Integrator 共用同一测量；server 生命周期＋serving-GPU 互斥锁＋untimed warmup＋N timed rounds 取 median+spread | 环境控制（隔离/互斥/预热） |
| 测量生命周期 | warm_server（默认）／isolated_server（每腿全新冷启，spread 界 boot-to-boot 方差）／legacy——**禁止跨生命周期取比值**（实测 warm 826.9 vs isolated 833.3 tok/s，方向不可预测） | 同口径（可比性纪律） |
| interleaved A/B | REF,CAND,REF,CAND… 单 GPU＋固定 port，≥5-7 repeats/leg；drift-control 腿（最后一条腿须为 REF 且属最低，防 warming 伪影）；同 session 重测分母——22h 旧基线曾把 +15.72% 报成 +0.86% | 重复测量 |
| 接受 gate 三条件 | engagement proof（改动真在 live server 跑）＋超噪声带（固定 shape 0.5%、AgentX trace replay 4% floor）＋输出 parity（greedy byte parity；量化/lossy 改动改走 gsm8k accuracy gate，cand ≥ baseline − tol） | 显著性/正确性 |
| 非重叠分布 gate | cand_min > ref_max 才算赢（sub-band 且重叠＝NULL） | 显著性 |
| Director 终判 | 独立重测并仲裁官方数字 | 独立复测 |

留白二（你来推）：**噪声带为什么分两档——固定 shape 0.5%、真实 trace replay 4% floor？**提示：固定 shape 的负载完全确定，噪声只来自计时与频率；AgentX trace replay 用的是 393 个真实 Claude Code session（median 142K input tokens）。你来推：真实 replay 的 4% floor 多出来的噪声源是什么？（方向：到达序列/长度分布的随机性——你的主场，排队论语言可以直接用。）

**判真第四档**：模块 3 四个 exemplar 的判真成熟度谱系——z.ai 零记载／FlashInfer-Bench 半配／CompileIQ 满配（Welch t p<0.05＋Cohen's d>0.2 双闸）／**GEAK＝另一种取向**：不做参数化统计检验，用**工程级简化**——噪声带 floor＋非重叠分布 gate（cand_min>ref_max，两个样本分布完全不重叠才判赢）。轻探一次（p/d 跨课衰退项）：CompileIQ 的双闸里 p 管「是不是碰巧」、d 管「值不值得」——GEAK 的非重叠 gate 同时把这两问合成了一问（不重叠 ⇒ 既非碰巧也够大），代价是**更保守**（分布有重叠的真改进会被拒）。这是统计哲学的取景差：参数化检验用分布假设换灵敏度，GEAK 用保守门槛换免假设、可工程直查。两个都成立，判真档位无高下——但对自报数字的约束力不同（§6）。

## 4｜优化物三形态与交付物：可逆 overlay＋上游化

优化物按成本递增三形态（R5 面4，A）：①config-only（flags/env/launch script）——**config-only win 是真 win**；②**可逆 overlay 源码替换**——PYTHONPATH overlay＋sitecustomize.py rebind dispatch attr（如重绑 aiter:gemm_a8w8_blockscale），**never editing installed packages**，可整体回退；③author from scratch（热 op 无现成实现时，kernel 层 author mode 写全新基线再 rebind）。

留白一（你来推）：**为什么「never editing installed packages」是硬纪律而不是风格偏好？**提示：交付物要发给别人、别的批次部署上跑。你来推：如果直接改 site-packages 里的包，交付物会变成什么形态（依赖当前机器状态的 diff）？overlay 机制下，交付物是什么形态（独立目录＋PYTHONPATH 前置）？

交付物（run 落盘 exp/e2e_<model>_<ts>/，R5 面4 A）：final bundle＝overlay＋final_patch.diff＋final_launch.sh（替代原启动命令即用）＋final_report.md＋各阶段工件；模型权重与已装包全程只读。——你 b) 答的「code 源码＋冻结镜像」逐字落在这里。

**上游化持久路径**（与「越用越聪明」并列的第二条持久路线）：focused PR 进 SGLang/vLLM 主干——带测速证据/ablation/CI review/硬件 guard，探索变体不进 final patch。实证形态＝AgentX 博客（R5 面3 A）：AMD 团队用 HyperLoom 辅助内核优化（fused indexer 91.24→35.56µs 即 2.57×）→**17+ 个 SGLang 合入 PR**→InferenceX 公开基准验证→公开 versioned recipe。注意边界：HyperLoom 是相关系统非 GEAK e2e_workflow 本体（§6 判真再压一次）。

## 5｜进化知识库三物件：不是一个库，是三件东西（留存家族第四次登场）

先纠一个直觉：「GEAK 的知识库」不是单一物件，是**三个不同机制的组合**（R5 面2，A）：

```
perf_knowledge/  静态参考库（~586 docs、54 算子×后端笛卡尔矩阵、SOTA registry）
  · 消费契约：只 seed/定位候选，永不降实测基线、永不跳验证
  · README 原文自认：「may be stale, incomplete, or wrong…not decisions」
knowledge/learned/  蒸馏经验卡（kernel 层与 e2e 层各一份）
  · ≤40 张上限，一卡一原则 ~12-20 行，reuse key＝kernel_class·gfx
  · 置信度三级：★noise／★★单次非重叠或 ≥2 一致／★★★≥2 非重叠或 verified e2e
  · 读取纪律：先自成 profile 计划、后读卡交叉（防卡先入为主框死搜索）
kb/  程序库面（identity/store_local/store_remote/retract/attest/curate）
  + e2e_store.py warm-start 三梯级回写（全 benchmark 点/TP 级/precision 级）
```

**蒸馏卡的 add-only 三铁律**（本课机制核心，R5 面2 A）：卡只能加候选，**不能删候选、不能剪 bake-off、不能跳测量**；卡与测量冲突→信机器、修卡；无卡可封路——「caution 是『也验证 X』，永不是『别做 Y』」。留白一答案收口：§1 留白三的 roofline 反例卡（预测 attainable=1.0 实测 +26.9%）入卡＝advisory 先验自带被证伪记录，但它**只加注不封路**——roofline 仍可路由，例外继续记录。你来推（留白三后半）：为什么这套契约叫「反笼子」（anti-cage）？README 自述「cold runs（KB 出现前）也能到 ~20% e2e；KB 唯一职责是帮收敛更快/走得更远，绝不能把能干的 run 关进笼子」——**KB 是加速器不是能力源**。假如允许卡剪枝，一张过时/特化的卡能把搜索关进什么笼子？

**kb/ 库面的淘汰机制**：store 无 DELETE——retraction＝**墓碑重写**（retained:false＋retired_reason＋排序标量清零沉底＋champion 指针重指；「假旗若只打标不清标量仍霸榜」）＋**attestation 真机台账**（recalls/validations/failures/not_reproduced——「validated 标志只是写者自评，三个月后读者问的是有没有别人拉出来跑成过」；单次失败是证据不是裁决，accumulated 台账才触发回撤）＋curate 按 direction 折叠（一 idea 一 rank，runner-up 作 alternates 免占槽）。warm-start：e2e 记录形态＝CONFIG＋值得 overlay 的 kernel 清单（patch 可选）——「store 有可绑 overlay artifact 时重放 +85.9%」。

**对照 AlphaEvolve 程序数据库（第 3 课锚，部件语言直引）**——留存机制家族对照表（第四次登场）：

| 维度 | AlphaEvolve DB | z.ai 骨架库（第 7 课） | CompileIQ Booster Pack（第 9 课） | GEAK KB（本课） |
|---|---|---|---|---|
| 库的身份 | **搜索种群本身**（分数驱动保留淘汰） | 蒸馏件（优化骨架沉淀） | 预调工件目录（注入搜索起点） | **搜索旁 advisory 先验库** |
| 回写什么 | 程序+多维分数+执行输出，最优回浮 | 优化骨架 | ACF 目录 | 蒸馏卡（≤40）+e2e 记录+attestation 台账 |
| 淘汰 | 分数驱动（岛/MAP-Elites） | 未见记载 | 无 | 墓碑回撤+champion 重指（无 DELETE） |
| 共同家族特征 | 搜索从资产开始／验证过的写回／适用条件随行 | 同 | 同 | 同（reuse key＝kernel_class·gfx 即「适用条件随行」） |

关键异同一句话：AlphaEvolve 的 DB **就是**种群——淘汰即进化压力；GEAK 的 KB **在搜索旁**——种群是 run 内 worktree 候选，KB 只 seed 候选不参与淘汰，测量永远裁决（R5 面2 A）。谱系连接：GEAK v2 曾有 GEAK-OpenEvolve（Triton-to-Triton QD/MAP-Elites，AlphaEvolve 谱系直接分支），v4 把进化搜索换成确定性 workflow＋蒸馏 KB——**同一家族内的一次范式切换**：从「分数驱动种群」换到「advisory 先验＋确定性控制面」。

## 6｜谱系拆正与生产先例措辞（信源纪律示范＋判真收口）

**谱系拆正（硬纪律）**：R5 R7 上报项——research_1 行 13/15 把两代混标了。正确口径（R5，A）：

- **GEAK v1**＝arXiv 2507.23194（2025-07）：Triton kernel 生成 agent＋评测基准；实证＝benchmark 级（TritonBench-modified 184 内核＋ROCm 30 内核，correctness 63%/speedup 2.59×）。论文**无 production 声明**——「AMD 生产级」标签源自微信文章⑨二手转述（C），非论文口径
- **GEAK v4**＝e2e_workflow＋进化知识库（2026-07-17 发版）：**无 arXiv/会议版本**，一手来源＝ROCm 官方文档＋repo 代码（A）；全部数字 AMD 自报，无第三方复现

本课引用纪律：e2e 机制/知识库/数字全部标 v4（文档+代码 A）；2507.23194 只在 v1 语境出现。**两代混标是本课反面教材**——二手综述把「论文实证＋系统工程」焊成一个叙事时，源语言会整段漂移；你综述时的对应动作＝每个声明回溯到具体版本的具体信源层。

**生产先例的准确读法**（判真收口）：「生产级」的四层实证形态（R5 面3 A）——①官方产品化（AMD-AGI 开源、ROCm 文档站收录、v4.0-4.2 版本化）；②AMD 内部使用声明（model bring-up 场景，无规模数字）；③**持续运行实证＝两 tier CI**（GPU-free 静态＋每 enrolled model 真机端到端跑、每模型独立 verdict）——是持续自动化 e2e 优化，**但形态是 CI，不是生产流量**；④上游生产路径＝17+ SGLang 合入 PR＋InferenceX 公开基准（versioned recipe；峰值 per-GPU 1.72×、P90 ITV 2.46× 为两个 deployed configuration 对比，作者自标「非受控 A/B」）。判真结论：**未见「GEAK 持续跑在客户生产流量」的一手声明**——「生产级」的准确读法＝真实 serving 工程管线在运行（CI＋评测＋上游合入），生产流量级证据为零。对照你的 b) 直觉：交付物（diff+launch.sh+冻结环境）你答对了，但**发出去 ≠ 跑在生产流量上**——这与你第 7 课 z.ai 判定（自报无复测）是同一把尺子的两次使用。

## 7｜关键概念清单（你当老师前的自检表）

1. 优先分公式 `pct_gpu_time × achievable_speedup`＝Amdahl 边界的正面形态（占比大×可达加速才值得动）
2. cheapest-first 三档杠杆：flag→env→源码/author，按成本升序花预算（联合搜索但预算有梯）
3. 确定性 JS 控制面×LLM 判断面分离＋分形两层（system 层调原封不动的 kernel_workflow，质量不回归）
4. verifier 全套：warm-server interleaved A/B＋噪声带（0.5%/4%）＋非重叠分布 gate＋parity/accuracy gate＋Director 独立终判
5. 优化物三形态（config-only／可逆 overlay／author from scratch）＋交付物（overlay+patch.diff+launch.sh，权重与包只读）
6. 知识库三物件（perf_knowledge 静态参考／learned 蒸馏卡 add-only／kb 墓碑回撤+attestation 台账）——advisory 先验，非种群

## 8｜费曼检验五件＋原创接口

1. **e2e 机制复述**：用你自己的话（「层次越高越明显」「联合排列组合」）把优先分公式和三档杠杆讲清——两者怎么接起来（先排序再花预算）
2. **知识库三物件＋对照**：AlphaEvolve DB 为什么是种群、GEAK KB 为什么是 advisory——同一家族特征三件，异同各一句
3. **谱系拆正一句**：v1 是什么（arXiv、benchmark 级）、v4 是什么（文档+代码 A、无论文）
4. **挂图定位＋判语**：路线 3 第四 exemplar、模块 3 收口；#11 深度梯 infra 格「算子+部署工作流」层；判真谱系第四档（噪声带哲学 vs Welch t 哲学——保守换免假设）
5. **生产先例准确读法＋选层判据**：CI+评测+上游 PR≠生产流量；美团版选层清单（kernel/编译器 flag/调度配置/编排）里 Amdahl 公式怎么当判据

原创接口（判断权在你，第 21 课整合件）：① 美团 serving RSI 的优化对象选层——用 Amdahl 优先分给四个层排序，哪些层占比大且可达加速高（层选择即设计决策）；② 知识库形态选型——advisory（GEAK）vs 种群（AlphaEvolve）vs 蒸馏件（z.ai），哪个配你的确定性控制面偏好。

现在你当老师，把这一课讲给我听。
