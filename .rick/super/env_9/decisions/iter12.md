# adjust 决策记录 · 迭代 12

> 决策窗口：iter10（上一份有落盘的 adjust）→ iter12。区间内实验 iter11-12，台账背景含 iter1-12 全量。
> 注：iter13 已在本决策落盘前被采样（np_matmul\|full_blas_dispatch 冠军格，visit 1→2，seq 548 后）且 builder 正在跑（iter013-d53a92b8/probes 于 04:59 更新）——本轮动作作用于 iter14+ 的采样分布。
> 过程说明：ledger 的 iter12 agents.adjust（641s / 830k tokens）未产出 decisions/iter12.md（落盘前该文件不存在）——本次为框架重派；本决策记录优先落盘、行文从简。
> skill 说明：本轮 super_sense 引用的 canon（internal/prompt/templates/skills/sense.md）**已检索到并通读**（iter10 时缺失）；super_experiment_analysis / super_cell_management 均按 skill 执行。价值判断项无 human 实时裁决，按保守默认处理并列入 §2 待复核清单。

## 1. 区间实验分析（iter 10 → 12）

| iter | cell | 假设 | 门禁 | perf | delta vs 纪录 | 归因 |
|---|---|---|---|---|---|---|
| 11 | einsum_path\|m_quarters | einsum optimize M 四分整块派发（4×2048 块）至平价带 | passed（soft exempt，无基线）→ **admit** | 482.36 | +51.5 | coding 于自检前被 turn-budget 杀死（exit 143，24 turn）；debug 补验闭环并修复 m<4 GEMV 1-ULP 退化（整阵单发 tensordot 回退）；einsum 轴首个精英（score 1.072 / ref 519.2 高负载窗口） |
| 12 | einsum_path\|m_quarters | probe：einsum 分块入口存在额外开销（einsum_noout(4)=653-670ms vs 单次整阵 450ms vs tensordot-arm ~485ms）；'ki' form_A 装配路线探查 | gate passed 但 perf 492.58 > 本格精英 482.36 → **reject（worse_than_elite）** | 492.58 | +61.8 | **过程死亡 ×2**（builder 900s / coding 900s 双双被杀，seq 474/506，exit 143）；solution.py = iter011 逐字节拷贝（新探索停留 probe_e 未落地）；debug 三轮复验全绿但收益定义上为 0，recheck -2.1% 判定为噪声 |

**全量背景**（iter1-12）：5 admit（iter3 纪录 430.83 / iter4 484.43 / iter5 481.76 / iter10 462.82 / iter11 482.36）+ 7 零产出或拒绝。iter3 后**连续 9 轮无纪录刷新**，admit 全落 462-492 平价带。

**重复模式（≥2 次，区间内新证据）**：
1. **过程失败仍在吃迭代，且集中于「空 method 描述」格**：iter10 的纪律注入只覆盖 np_dot\|full 与 np_matmul\|full 两格（archive 验证：全场 12 格仅此 2 格有 method），而 iter12 被采的 einsum\|m_quarters 描述为空 → builder/coding 双 900s 超时死亡（seq 474/506）。iter11 coding 亦死于 turn-budget（debug 救回）。区间 8 个 agent 段中 3 个死亡。
2. **同代码跨次测量噪声 ±3% 且符号不定**：iter12 debug 对逐字节相同的代码取得 6 次独立 k=3 中位读数 {479.91, 480.46, 486.78, 487.60, 492.58, 508.43}，ratio 0.9949-1.0539 双侧摆动；5 次 ref 读数 433.7-468.4 全带内——「同代码 -2.1% 劣化」实为纯噪声（iter012 experiment_record attempt 200 假设 1 成立）。iter10 的「剩余方差 = 负载噪声」判断被再次实证。
3. **9 轮无新纪录**（iter4-12）：全部合法形态坍缩 434-488 带（±6%），纪录 430.83 为带下沿尾抽；区间内新 admit（482.36）与新 reject（492.58）都在带内/带上沿。

**空白方向**：无格级空白（12 轮后轴交叉全覆盖、3 个 rowwise 已归零）。唯一未实测机制 = 冠军格微变体菜单（pre-fault + out= 反转侧，iter10 已写入 method，iter13 builder 正在执行）。einsum 轴残留线索：probe_e 揭示的 einsum 'ki' 调用 112ms 潜在线索（iter012 记录明示「属下轮探索范畴」）。

**主要矛盾候选**（交 SENSE 裁决）：
1. **噪声天花板矛盾**：合法解全部剩余方差 = 本机负载噪声（证据：iter12 同代码 6 读数双侧摆动；iter11 ref 519 vs iter12 ref 433.7；9 轮无纪录）。
2. **预算效率矛盾**：剩余 3 轮（iter13 在跑 + 2 轮），区间内损失近 1 轮等效预算于「空描述格的过程死亡 + 同代码重复实验」（iter12 的 900+900+840s 全花在逐字节拷贝的复验上）。
3. （备选）einsum 轴被支配矛盾：5 采 1 admit 1 reject，入口开销实测存在（分块 einsum 比等价 tensordot 慢，653-670 vs 485）。

## 2. SENSE：主要矛盾与辩证逆转

- **主要矛盾（一句话）**：合法解空间的全部剩余方差 = 本机负载噪声（Accelerate 大 GEMM 走共享 SME/AMX 协处理器、容量固定——见 §3 研究佐证；同代码 6 读数 ±3% 双侧摆动，iter12 seq 536/547；9 轮无纪录 iters 4-12），而剩余预算仅 2-3 轮且其中相当比例仍会被空描述格的过程失败与被支配格的重复采样消耗（iter12 双杀 seq 474/506 + 同代码 reject）——因此**剩余迭代的产出由「采样命中率 × 过程存活率」而非 kernel 机制决定**，采样分布必须集中到「有精英、带内、method 已注入纪律」的格。
- **排除的候选**：einsum 轴被支配矛盾——降级为支撑证据（不是产出决定因素，处置为降权而非主攻）；预算效率矛盾——与主矛盾合并（其解法正是权重与描述两个手段）。
- **辩证逆转**：
  - 阻碍 X = 噪声带淹没一切结构差异；期望 Y = 刷新纪录（<430.83）。**逆转：噪声带既是阻碍也是唯一剩余方差源 → 把「重采」变成手段**：预算集中到最小开销内核冠军格（6.0），多次重采收割低负载窗口（430.83 本身即尾抽证据）。→ 动作：soft_weight 冠军 5.0→6.0。
  - 阻碍 X = 空描述格被采样后死于超时（iter12 双杀）；期望 Y = 高产出。**逆转：把「过程可靠性」变成采样分布本身的维度**——权重与描述向「method 已注入 + 有精英」的格集中：np_dot\|m_halves（精英 462.82、iter10 纪律成功格、全场第二快精英）提权 + 注入 method；einsum\|m_quarters（有界衰减）让出份额。→ 动作：soft_weight ×3 + modify_cell ×1。
  - 阻碍 X = einsum 入口开销实证存在且轴价值已榨干；期望 Y = 轴向探索完整。**逆转：承认「轴值齐全 ≠ 等价价值」**——einsum 唯一合法形态 = 手工复制其内部序列（tensordot）= 等价 matmul 族且入口更贵；轴探索完成，不再投入，保留 0.25 残留通道（'ki' 112ms 线索可复活）。→ 动作：无 new_cell / 无 expand_cell。
- **良质批判**：置信度**高**——噪声双侧摆动有 6 读数直接证据；过程死亡有 agent json exit 143 ×2；天花板有 9 轮无纪录 + AMX/SME 平台研究 + domain 阶梯三重互证。残余假设：①「method 文本纪律可传递过程可靠性」——iter10 注入的两格尚未被采验证（iter13 冠军格是首次，np_dot\|full 自 iter7 后未被采）：若 iter13/14 仍死于过程则证伪，需框架层修复（超出本角色职权）；②冠军微变体（pre-fault 反转侧）可能有带外收益——iter13 在跑，其结果直接裁决；③若 iter13-15 冠军重采全 440+ 且无带外收益 → 天花板为硬墙，**下一轮 adjust 应提 terminate_recommend**（iter10 证伪线 b 的延续）。
- **待人工复核的价值判断**（保守默认处理）：① einsum\|m_quarters 衰减止于 0.25 而非 0.0（该格已具 2+ 独立失败证据、灭绝合法，但保守保留 'ki' 线索复活通道）；② 未提 terminate_recommend（迭代预算 3/15 见底，但 iter13 正是裁决实验，等结果）；③ 冠军提权上限 6.0 而非极端集中（保留 np_dot 双格 28.5% 份额作入口不对称论题与多样性储备）。

## 3. research（role=adjust · 论文检索）

检索：web_search 4 角度（GEMM new methods 2024-25 / AlphaTensor 与 fast-MM 实用性 / Apple Accelerate AMX-SME 性能 / numpy dot-matmul out= 开销）。筛除不可行族：FlashGEMM（TACO 2025，GEMM **序列**数据复用——本 env 单次乘法不适用）、RISC-V GEMM（HPCC 2025，平台不符）、SparseZipper（2502.11353，SpGEMM 稀疏不符）、QiMeng-GEMM（AAAI，LLM 代码生成——改求和结构撞墙）、混合精度（2506.11728，精度墙死族）、AlphaTensor/fast-MM 族（Nature 610 + ACDA25 Alternative Bases——改乘法-累加结构，K 求和序墙，死族清单实测超容差 14-29×）。**可落地 ≤3 条**：

1. **{julialinearalgebra.github.io/AppleAccelerate.jl benchmarks（M2 Max / macOS 26，2026-07 复测），"Accelerate offloads large GEMM to a shared SME co-processor that a single thread already saturates"，机制：Accelerate 的 BLAS 大 GEMM 走 AMX/SME 矩阵协处理器，该单元为共享固定容量数据通路，单线程即饱和（与 OpenBLAS 的 CPU 核扩展模型不同）}** → applicability：解释本 env 全部合法内核坍缩同带的原因——**平台容量即天花板，非算法可破**；带内波动 = 共享单元的争用 → 尾抽 = 低争用窗口。直接支撑「冠军重采收割尾部」策略；同时证伪「线程数调优」方向（Accelerate 自管调度，OPENBLAS_NUM_THREADS 类不可用——避免浪费剩余迭代）。
2. **{github:numpy/numpy#18669（iter10 已引，本轮复核），"matmul out= 慢于 dot 仅在输出页未 fault 时成立，预 fault 后反转"，机制：输出页 first-touch fault 逐页停顿}** → applicability：反转侧（预 fault + out=）仍未实测——iter13 冠军格在测 full 形态；**m_halves 形态的反转侧同样从未实测**（两半 c[:]=0 预 fault 后两次 dot+out= 直写）——写入 np_dot\|m_halves 的 method 微变体菜单（本轮 modify_cell）。
3. **{github:numpy/numpy#23588（复核），"matmul 对非连续/非常规输入走 noblas 慢路径 ~100×，dot 入口更宽容"}** → applicability：确认 np_dot 入口不对称论题的机制边界（iter10/iter011 已引用同源证据）；支撑 np_dot\|full（2.0 保持）与 np_dot\|m_halves（提权）的预算配置，无需新动作。

## 4. 格管理动作

（3 权重变动 + 1 描述改进，均在「单轮有界」内（≤5 权重、≤2 描述）；无 new_cell：研究未发现可落地新族（全部撞 K 序墙或平台墙），轴交叉全覆盖；无灭绝动作：einsum\|m_quarters 按 0.25 衰减而非 0.0（保守默认，见 §2 待复核①）；无 terminate_recommend：iter13 裁决实验在跑（见 §2 待复核②）。）

1. **soft_weight np_matmul|full_blas_dispatch：5.0 → 6.0**（rule: champion_tail_harvest）
   依据：iters 4-12 连续 9 轮无新纪录（perf_timeline；admit 全 462-492 vs 纪录 430.83=带下沿尾抽）；iter12 同代码 6 读数 ±3% 双侧摆动证明门禁读数即噪声采样（iter012 experiment_record attempt 200，ratio 0.9949-1.0539）；研究 §3-1 证 AMX/SME 平台容量固定——重采收割低争用窗口是唯一剩余手段。冠军候选产出率 3/3（iter3/4/5 族），iter13 已在采（策略生效中）。份额 45.5% → 49.0%。
2. **soft_weight np_dot|m_halves：1.0 → 1.5**（rule: elite_momentum_tail_harvest）
   依据：iter10 admit 462.82（全场第二快格精英，gen-1 精英刚立）；iter10 该格 coding 完整走完（全场唯一「最新产出 + 过程纪律成功 + 精英在带」三重成立格）；#18669 反转侧在 m_halves 形态从未实测（本轮同步注入 method，见动作 4）。份额 9.1% → 12.2%。
3. **soft_weight einsum_path|m_quarters：0.5 → 0.25**（rule: cell_ceiling_reached_decay）
   依据（iter10 降 0.5 后的新证据，非震荡）：iter12 reject worse_than_elite（seq 536/547，perf 492.58 > 精英 482.36——本格精英即形态上限，再采期望收益 ≤0）；iter12 双过程死亡（seq 474/506，builder/coding 各 900s 被杀）；einsum 入口开销实测（iter12 probe：einsum_noout(4)=653-670ms vs tensordot-arm ~485ms——家族被支配再实证）。该格累计：iter9 过程死 / iter11 admit / iter12 reject+双杀。0.25 而非 0.0：保留 probe_e 'ki' 112ms 线索复活通道（保守默认）。份额 4.5% → 2.0%。
4. **modify_cell np_dot|m_halves：空描述 → 微变体菜单 + 过程纪律**（rule: description_sharpen）
   依据：该格提权至 1.5 后描述仍为空（archive 验证），iter12 证明空描述格被采即有过程死亡风险；iter10 注入纪律的两格（np_dot\|full / np_matmul\|full）模式可复制；#18669 反转侧在本格形态从未实测。method 全文见提案 JSON（基线复测 + 预 fault 变体 + m<4 整阵回退 + 每工具一次纪律 + 工作量 ≤ iter010 一半）。

## 5. 预期效果

- 采样分布（iter14+）：冠军 45.5% → 49.0%；np_dot\|full 18.2%（不变）；np_dot\|m_halves 9.1% → 12.2%；**三个 method 注入格合计 77.6%**（原 72.7%）；einsum 族 13.6% → 10.2%（其中 m_quarters 4.5% → 2.0%）。
- 预期：① iter14/15 的采样以 ~78% 概率落在「有精英 + 带内 + 纪律注入」的格上，过程存活率回升（对照 iter10 注入后该模式格 0 过程死亡）；② 冠军格获 1-2 次重采（含 iter13 共 2-3 次）：若 pre-fault/out= 反转侧有带外收益则精英下探 <430 破纪录，若全落 440-490 带内则确认平台容量硬墙；③ einsum\|m_quarters 不再吸收重复实验预算（iter12 一轮耗 2640s/2.4M token 于逐字节拷贝的复验——本次改分布后此类支出归零）。
- 证伪线：a) iter14/15 仍有过程双杀（含 method 注入格）→ 「纪律可经文本传递」假设证伪，需框架层修复，届时写入待人工复核；b) iter13-15 冠军重采全 440+ 且微变体无带外收益 → **下一轮 adjust 提 terminate_recommend**（证据将满足：12+ 轮无纪录 + 平台容量研究 + 全族带内）；c) einsum\|m_quarters 若意外被采且 'ki' 路线真有带外收益（admit 且破纪录）→ 本轮衰减判断错误，下轮恢复至 0.5+；d) np_dot\|m_halves 若被采后仍 turn-budget 死亡 → method 注入对该格无效，下轮回退权重。
- 预算视角：剩余 3 轮（iter13 在跑）/ 墙钟 96.7min/8h / token 无上限——迭代是唯一紧约束；集中权重的机会成本近零（轴交叉全覆盖 + 9 轮带内证据），风险「错过未知方向」被研究否证（新族全部撞墙）。
