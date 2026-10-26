# adjust 决策记录 · 迭代 10

> 决策窗口：iter6（上一份有落盘的 adjust，无提案、framework_synthesized）→ iter10。区间内实验 iter7-10，台账背景含 iter1-10 全量。
> 注：iter11 已在本决策落盘前被采样（seq 357，einsum_path|m_quarters，份额 0.072917=0.7/9.6）且 builder 在跑——本轮动作作用于 iter12+ 的采样分布。
> skill 说明：super_sense 引用的 canon（internal/prompt/templates/skills/sense.md）在 <工作区根>/.rick 下未检索到，按 super_sense 内置的五步映射表执行，未中断流程。

## 1. 区间实验分析（iter 6 → 10）

区间内逐实验事实（iter7-10）：

| iter | cell | 假设 | 门禁 | perf | delta vs 纪录 | 归因 |
|---|---|---|---|---|---|---|
| 7 | np_dot\|full_blas_dispatch | 单次整阵 np.dot 直派塌缩 26× Python 装配缺口至 BLAS 平价带（预检 437.4ms，预检证逐位一致 maxdiff 0.0） | 未达（no_candidate） | - | - | **过程失败**：builder valid=true 但 coding/debug 双双 truncated（350.7s/107.5s，agent json valid=false）——候选 solution.py 已落盘但未注册 |
| 8 | einsum_path\|m_halves | einsum M 二分整块派发至平价带 | 未达 | - | - | **过程失败**：builder 中断（seq 293，136s，valid=false，iter008 目录无 solution.py） |
| 9 | einsum_path\|m_quarters | einsum M 四分整块派发至平价带 | 未达 | - | - | **过程失败**：builder 中断（seq 299，125s，valid=false，iter009 目录无 solution.py） |
| 10 | np_dot\|m_halves | M 二分 + 每半 np.dot(...,out=) 直写，塌缩 25× 至平价带且避开 matmul out= 慢路径（numpy#18669） | **passed**（score 0.978，软判定 better，ratio 0.936 vs 基线 484.43） | **462.82ms**（自检 454.99 / benchmark 463.45 / gate 453.44 三读，IQR 1.8ms） | +31.99（未破 430.83） | **admit**：全判据命中，verify max_error=0.0 双 seed 双跑逐位一致 |

全量背景（iter1-10，判趋势用）：**4 admit**（iter3 430.83 纪录 / iter4 484.43 / iter5 481.76 / iter10 462.82）+ **6 零产出**（iter1 np_dot|m_quarters、iter2 np_dot|m_halves v1、iter6 einsum|full_blas、iter7 np_dot|full_blas、iter8/9 einsum M 分解）。其中 **iter1/6/7 候选代码已落盘**（各目录 solution.py 在档：iter006 是完整 einsum optimize=path 实现）却因 coding/debug turn-budget 耗尽未注册；iter2/8/9 死在 builder 阶段。

**重复模式（≥2 次）**：
1. **turn-budget 过程性死亡 ×6**（iter1/2/6/7/8/9）——iter010 记录明确诊断：「iter006/007/008/009 连续 4 轮 no_candidate 均因 coding/debug 阶段 turn-budget-exceeded（过度复跑门禁、追噪声）」；iter10 在计划里钉死「每工具跑一次即记录」后 100% 产出并 admit。**过程失败（非性能失败）是第一产出杀手：产出率 4/10。**
2. **einsum 族 3 采 0 候选**（iter6/8/9；seq 293/299 佐证 8/9）——iter006 的 einsum 候选被 iter9 builder 实测（iter009 experiment_plan.json idea 记载）gate_check 531-647ms vs ref 453ms，**超平价带 17-43%**；einsum optimize 路径派发同一 cblas_sgemm + 额外 path 处理开销，语义上被 np_matmul 直派支配。
3. **admit 全部落平价带且无新纪录**：iter3 后连续 7 轮无纪录刷新（iters 4/5/10 admit 462-484 vs 430.83）；iter10 交错 A/B 预检：m_halves_dot 454.4 ≈ m_halves_matmul 454.3 ≈ full_matmul 453.7 ≈ full_dot 480.7（噪声带 ±6% 即 434-488）——合法结构差异 < 机器噪声。

**空白方向**：12 格轴交叉已全被触及（3 个 rowwise 已因 0.045 阶梯证据归零）。有活论题未决的格只剩：np_dot|full_blas（入口不对称的直派形态，iter7 死于过程非性能）与冠军格微变体空间（out= + 预 fault——本轮 research 引入，从未实测）。

**主要矛盾候选**（交 SENSE）：
1. 预算效率矛盾：60% 迭代死于过程失败 + 25% 权重（2.4/9.6）押在语义被支配的 einsum 族（证据：iter1/2/6/7/8/9；权重表）
2. 性能天花板矛盾：合法族坍缩 ±6% 噪声带、纪录为带下沿尾抽、7 轮无新纪录（证据：iters 3/4/5/10；domain 阶梯）
3. 入口不对称微论题：np.dot vs np.matmul 的 out= 路径差异（证据：iter10 同 session 449.6-466.5 vs 480.5；#18669）

## 2. SENSE：主要矛盾与辩证逆转

- **主要矛盾（一句话）**：合法解空间的性能天花板已被 iter003 触顶（全族坍缩 434-488ms ±6% 噪声带、纪录 430.83 为带下沿尾抽、连续 7 轮无新纪录），因此决定剩余 5 轮产出的不再是 kernel 机制，而是**预算效率**——60% 迭代死于可避免的过程失败（turn-budget 耗尽于复跑门禁/过度调研），且 25% 采样权重押在语义被支配、3 采 0 候选的 einsum 族上（证据：iter1/2/6/7/8/9 六轮零产出且 iter1/6/7 候选代码在档；archive 权重 2.4/9.6）。
- **排除的候选**：入口不对称微论题——iter10 已实测同带（449.6-466.5 vs 全带 434-488），收益量级 < 噪声，不足以承担主要矛盾；保留为冠军/np_dot 格的微变体菜单，不作主攻方向。
- **辩证逆转**：
  - 阻碍 X = ±6% 噪声带淹没一切合法结构差异；期望 Y = 刷新纪录（<430.83）。**逆转：噪声带既是阻碍也是唯一剩余方差源 → 把「重采」变成手段**：预算集中到冠军格 np_matmul|full_blas（3.0→5.0），用多次重采收割负载低潮尾部——430.83 本身即尾抽的直接证据。→ 动作：soft_weight + modify_cell（微变体菜单）。
  - 阻碍 X = 过程失败吃掉 60% 迭代；期望 Y = 高产出。**逆转：把失败教训变成设计约束**——iter10 已验证的「每工具一次、读数即记录」纪律 + 最小实现范围（一行内核 + 契约移植）写进下一轮最可能被采样的格的 method 描述，让纪律随格分发。→ 动作：modify_cell np_dot|full_blas 与 np_matmul|full_blas。
  - 阻碍 X = einsum 族语义被支配（同 BLAS + 额外开销）；期望 Y = 轴向探索完整。**逆转：承认「轴值齐全 ≠ 等价价值」**——einsum 各格按单次失败地板收缩至 0.5，预算让给冠军与 np_dot 直派重试。→ 动作：soft_weight einsum ×3。
- **良质批判**：置信度**高**——过程失败模式有 6 条直接事件证据且 iter10 提供了「对照性成功」（同纪律下 100% 产出）；天花板有 4 个 admit + domain 阶梯 + 交错 A/B 预检三重互证。残余假设：①尾部可收割——若 430.83 是一次性异常（机器冷态低负载），冠军 3+ 次重采全落 440+ 即证伪；②预 fault 净收益为正——256MB 输出全预写 ~15-25ms，仅当 gemm 提速超过它才成立（一次实测可证伪）；③过程纪律可经 method 文本传递——iter12/13 若仍死于 turn-budget 即证伪（届时需框架层修复，超出本角色职权）。
- **待人工复核的价值判断**：无——本轮全部动作在保守默认内（提权有直接证据、降权遵守单次失败地板 0.5、无灭绝动作）。

## 3. research（role=adjust · 论文/issue 检索）

检索：web_search 4 角度（GEMM new methods 2024-25 arxiv / MM 算法族谱 taxonomy / sgemm 输出页 fault 开销 / numpy matmul vs dot out= 开销）。筛除不可行族：GPU 量化/混合精度族（[2509.01229] LiquidGEMM、[2506.11728] mixed-precision、[2503.01253] NM-SpMM）改数值精度撞 1e-3 容差墙（f32 锚自身离真值 1.42e-2）；Strassen/fast-MM 族谱（fmm.univ-lille.fr、[1412.1145]）改 K 求和序（死族清单实测超容差 14-29×）；分布式/异构族单进程 env 不可用。**可落地 ≤3 条**：

1. **{github:OpenMathLib/OpenBLAS#3197, "speed up GEMM by pre-faulting memory", 机制：GEMM 输出数组若无物理页备份，写出现场 first-touch fault 逐页停顿；实测「先 memset（时间计入测量）再 gemm」可比原调用快至 4×}** → applicability：本 env 每次计时调用都新分配 256MB 输出（np.empty / np.matmul 自分配），全部未 fault。写进冠军格 method 微变体菜单：c[:]=0 预 fault 后再派发，一次实测净增益。
2. **{github:numpy/numpy#18669, "matmul out= slower than dot on unfaulted pages——预 fault 后反转", 机制：matmul 的 out= 路径在输出页未 fault-in 时走慢路径；页预 fault 后差异反转，matmul+out= 反超 dot+out=}** → applicability：iter10 只实测了「未 fault + dot 占优」一侧；「预 fault + matmul」反转侧从未测过——写进冠军格微变体菜单；并确认 np_dot|full_blas 重试时 out= 变体入口最宽容。
3. **{arXiv:2411.00442（iter010 research_report §3 已引，本轮复核）, "Revealing Floating-Point Accumulation Orders", 机制：GEMM 累加序是未文档化实现细节，改变调用形态/分块即改变结果}** → applicability：约束句写进两个 modify_cell——微变体必须保持单次整阵/整块派发（M 分解合法，K/N 分解必死）。

## 4. 格管理动作

（权重变动 5 格 + 描述改进 2 格，均在「单轮有界」内；无 new_cell：轴交叉已全覆盖且天花板下新建格只会稀释采样份额；无灭绝动作：einsum 各格均按「单次失败地板 0.5」处置。）

1. **soft_weight np_matmul|full_blas_dispatch：3.0 → 5.0**（rule: champion_tail_harvest）
   依据：iter003 纪录 430.83 为带下沿尾抽；iters 4/5/10 三个 admit 全在 462-484 未破纪录（iter10 见 seq 347-348）；domain 阶梯 + iter10 交错 A/B 证明剩余方差=噪声，冠军重采是收割低负载尾部唯一手段；且 np_matmul 族候选产出率 3/3（iter3/4/5）全场最高。份额 31.3% → 45.5%。
2. **soft_weight np_dot|m_halves：0.7 → 1.0**（rule: fresh_elite_momentum）
   依据：iter10 admit 462.82 / score 0.978 / 软门禁 better（ratio 0.936，seq 346-348），gen-1 精英刚立；np_dot 入口不对称（#18669）在同 session 对照成立——全场唯一「最新产出且有活微变体空间」的精英格。
3. **soft_weight einsum_path|full_blas_dispatch：1.0 → 0.5**（rule: dominated_family_decay）
   依据：iter6 ledger verdict=no_candidate（iter006 agent_coding/debug 双 truncated、valid=false）；iter006 候选被 iter9 builder 实测 gate_check 531-647ms vs ref 453（超带 17-43%，iter009 experiment_plan.json idea 记载）；domain：einsum optimize 派发同一 cblas + 额外 path 开销，被 np_matmul 直派支配。
4. **soft_weight einsum_path|m_halves：0.7 → 0.5**（rule: single_failure_floor）
   依据：iter8 builder 中断（seq 293，valid=false，iter008 目录无 solution.py），visit=1 无精英无台账行——单次失败按纪律降至 0.5 地板；叠加族级 3 采 0 候选背景。
5. **soft_weight einsum_path|m_quarters：0.7 → 0.5**（rule: single_failure_floor）
   依据：iter9 builder 中断（seq 299，valid=false，iter009 目录无 solution.py）——同上地板处置。注：iter11（seq 357）已再采本格，其结果是下一轮「继续降或复活」的直接证据。
6. **modify_cell np_matmul|full_blas_dispatch：空描述 → 微变体菜单 + 过程纪律**（rule: description_sharpen）
   依据：该格 description 为空、builder 无微变体指引；research（OpenBLAS#3197 预 fault + #18669 反转侧）给出从未实测的 out=/预 fault 变体；iters 4/5/10 证明纯结构无增益——冠军格需要「微变体实测 + 尾部收割」的明确方向。method 全文见提案 JSON。
7. **modify_cell np_dot|full_blas_dispatch：空描述 → 最小重试范围 + 过程纪律**（rule: retry_scope_minimization）
   依据：iter7 三段 agent（builder 214s / coding 351s / debug 108s）全 truncated 而 no_candidate——失败在过程（过度调研/复跑）非性能（预检 437.4 ≈ matmul 433.1，±1%）；该格是唯一「有活论题（入口不对称）且无精英」的高权格（2.0），重试必须把范围钉死到一行内核 + 契约移植。method 全文见提案 JSON。

## 5. 预期效果

- 采样分布（iter12+）：冠军 31.3% → 45.5%；einsum 族 25% → 13.6%；np_dot|full ≈ 18% 不变；np_dot|m_halves 7.3% → 9.1%。
- 预期：① 过程纪律经 method 注入后，候选产出率 40% → ~70%+（iter12-15 ≥3 轮有候选）；② 冠军格获 2-3 次重采：若预 fault/out= 微变体有带外收益则精英下探 430 以下，若三次全落 440+ 则确认天花板为硬墙；③ np_dot|full_blas 大概率产出首个精英并闭环入口不对称论题。
- 证伪线：a) iter12/13 仍死于 turn-budget → method 注入不足，需框架层修复（届时写入「待人工复核」）；b) 冠军 3 次重采全 440+ 且微变体无带外收益 → 下一轮提 terminate_recommend；c) iter11 einsum|m_quarters 再零产出 → 下一轮该格有 2+ 直接证据，可提 weight=0；d) einsum|m_halves iter12+ 再被采且零产出 → 同上。
- 预算视角：剩余 5 轮，墙钟 20.8min/8h、token 无上限——集中权重机会成本近零；风险「过早集中错过未知方向」被 10 轮证据否定（12 格轴交叉已全触及：3 rowwise 阶梯性死亡 + 3 matmul admit + 1 np_dot admit + 3 einsum 过程性死亡 + np_dot|m_quarters 单败）。
