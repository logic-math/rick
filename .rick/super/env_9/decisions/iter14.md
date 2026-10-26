# adjust 决策记录 · 迭代 14

> 决策窗口：iter12（上一份有落盘的 adjust）→ iter14。区间内实验 iter12-14，台账背景含 iter1-14 全量。
> skill 说明：super_experiment_analysis / super_sense / super_cell_management 三个 skill 全部读取并按步执行；
> super_sense 引用的 canon（internal/prompt/templates/skills/sense.md）本轮在工作区内未检索到（iter12 时曾命中，
> 现路径不存在）——按 super_sense 内置的五步映射表执行，未中断流程。
> 预算态势（本次决策的硬背景）：**iterations 14/15（仅剩 1 次）、wall 156.5min/8h、saturation 未触发**。
> 融合门禁（gate_check）本轮未自跑：iter14 debug attempt 200 当日已给 3 次 gate_check + 30 次交错计时
> （正是「精英是否还有潜力」的客观数据），再跑即追噪声，与纪律相悖。

## 1. 区间实验全景（iter 12 → 14）

| iter | cell | 假设 | 门禁 | perf | delta vs 格精英 | 归因 |
|---|---|---|---|---|---|---|
| 12 | einsum_path\|m_quarters | （无新假设落地：coding 于自检前被 900s 进程超时杀死 exit 143，solution.py = iter011 准入版逐字节拷贝；新探索 probe_a-e 停留在探针） | gate passed（soft exempt 无基线可比） | 492.58 | +10.2（vs 482.36） | **reject worse_than_elite——运营性失败**（无新代码入格），读数 = einsum 族噪声带 |
| 13 | np_matmul\|full | V2（每次新建输出 + 写入式预 fault + matmul(out=)，numpy#18669 反转侧）有 >5% 带外收益 | gate passed（band 1.038） | 451.14 | +20.3（vs 430.83） | **reject worse_than_elite——假设否定（平价收口）**；当日 V1「恒慢 14%」（498-523 vs 436，后证为机器慢模式）；**线程收紧恒劣 → 默认 18 线程已最优**；builder 900s 截断由 coding 重建 |
| 14 | np_matmul\|full | P1（per-shape 持久缓冲 + 预 fault + matmul(out=)）有 >5% 带外收益 | gate passed（首跑 band 1.0046 / recheck 中位 band 1.0216） | 440.14（首跑 432.81） | +9.3（vs 430.83） | **reject worse_than_elite——假设否定**；buffer_ab 6 形态（P0/P1/P2/P5/P6/P7）同 floor ~433ms、bitwise P1≡P0、全臂 minflt/call=0；P6 spacer 数据驱动否决；builder 900s 死亡由 coding 恢复 |

**重复失败模式（≥2 次）**：

1. **worse_than_elite 三连拒（iter12/13/14，台账三行 reject）**：三份候选全部 gate passed + band 内平价，却全部因高于格精英被拒。深层原因见 iter14 debug attempt 200：**精英基线 430.83 是跨会话安静窗读数——当日同进程 plain ref 中位 440.96/449.85/452.94/476.48（比精英慢 2.3%-10.6%），当日 warm floor 432.5-433.1，30 次交错计时中低于 409.3ms（0.95×精英）的读数 = 0**。即：合法实现之间 bitwise 一致、同 floor，差异全部来自机器状态漂移与外部争用（尖峰窗 580-590ms）。
2. **900s 进程死亡三连（iter12 coding / iter13 builder / iter14 builder，agents valid=false / exit 143）**：builder/coding 在固定 wall 预算内过度探研深探针；iter14 coding（675s）证明「方案先落码、验证收敛」可活，深探研必死。
3. **机器噪声主导剩余方差**：iter12 同代码 6 读数 ±3% 双侧摆动；iter14 plain 自身 435-476ms 摆动 + 1/5 次外部争用窗 589ms。

**空白方向核查（对照轴定义与 domain brief）**：

- 线程维度：已测死（iter13 thread_sweep 收紧恒劣，18 线程最优）——且属 env 层非代码层。
- 输出缓冲/预 fault 维度：已测死（iter13 V2 + iter14 P1/P2/P5/P6/P7 全形态同 floor、零缺页）。
- K 求和序改变族：死域（A 级证据，超容差 14-29×，毫无侥幸）。
- einsum 族：入口本征劣化（无 out= 字面入口 653-670ms vs tensordot 臂 ~485ms，iter12 探针），族精英 482-492 距 floor +11%。
- M 分割族：结构代价 +26-50ms（iter004 484.4 / iter005 481.8 / iter010 462.8 vs 直派 433-437），不可能接近 430.83。
- **结论：合法解空间无剩余空白方向。**

**主要矛盾候选（交 SENSE）**：

1. 性能矛盾已终结——剩余方差是机器状态而非实现（iter14 debug attempt 200 测量侧归因 + floor 探针）。
2. 采样分布与空间状态错配：seq 626 显示 48.98% 采样份额投向微变体已穷尽的冠军格。
3. 预算只剩 1 次迭代——末次采样的价值只在「低风险收档」而非「探索」。

## 2. SENSE：主要矛盾与辩证逆转

- **主要矛盾（一句话）**：合法解空间已在 vecLib sgemm floor（~433ms）上 bitwise 收敛（全部合法形态逐位一致、同 floor、零缺页），而剩余性能方差全部来自跨会话机器状态漂移与外部争用——**系统继续采样已不是在探索实现，而是在对机器噪声重复抽签，且预算只剩 1 次迭代**。
  证据：iter14 debug attempt 200（plain ref 440-476 全部高于精英 430.83；30 计时零个 <409.3；floor 433 = sgemm 本体）；iter14 buffer_ab（6 形态同 floor、P1≡P0 bitwise、minflt/call=0）；iter13（V2 平价收口 + 线程恒劣）；iter12（同代码 ±3% 摆动）；台账 iter12/13/14 三连 worse_than_elite。
- **排除的候选**：①「装配层残余可优化」——iter003 起 profile matmul self 100%、包装层为零（iter13 记录 §1）；②「out=/预 fault 反转还有收益」——iter13 V2 + iter14 P1 双证伪，本机 macOS 分配行为下 numpy#18669（Linux/OpenBLAS 情境）无可省成本（iter14 根因归因）。
- **辩证逆转**：
  - 阻碍 X = floor 即 ceiling（BLAS 平价天花板）；期望 Y = 更低 runtime。
  - **方向 1（收口逆转）**：既然「达到平价」是任务完成判据，就把天花板从阻碍反转为验收凭证——性能目标已实现（430.83 ≤ floor 433），正确动作是收口归档而非再抽签。→ 对应动作：`terminate_recommend` + 冠军格收权。
  - **方向 2（末次采样价值重定义）**：把「预算枯竭」反转为「档案完整性手段」——最后 1 次迭代导向 method 完备的格（np_dot|full 补族精英缺口——iter007 留洞至今；或冠军基线复认），不导向空描述格（过程死亡风险）。→ 对应动作：`soft_weight` × 5 + `modify_cell` × 2。
  - **方向 3（已考虑并否决）**：「机器噪声既可掩盖收益也可造就纪录」→ 追安静窗纪录。否决：iter12 纪律已证同代码 ±3% 双侧摆动 = 噪声；iter14 debug 30 计时零个 <409.3；噪声纪录不可复现，违背「不追噪声」纪律（该纪律写在本 env 多格 method 中）。
- **良质批判**：置信度 **高**。残余假设： floor 测量代表性——iter13/14 两轮 gate + 探针互证，风险低； 精英 430.83 为有效纪录——接受其为纪录的同时承认其「安静窗读数」性质，这恰是「不可再抽签改善」的对称证据； 合法空间封闭性——依赖 domain brief A 级死族清单 + 本轮 research（§3）未发现新合法族，成立。需补证：无（预算不允许且无必要）。
- **待人工复核的价值判断**：是否值得为「复现 430.83 安静窗」追加投入（换机器条件/预约安静窗/扩迭代数）——本轮按保守默认（不再抽签）处理，交 `terminate_recommend` 由人裁决。

## 3. research（按 research_brief：arXiv/OpenReview，创新点与趋势）

检索 3 组角度（fast MM 算法族分类 / 超越 vendor BLAS 的新方法 / 求和序与可复现性权衡）。**结论：2024-2025 文献中不存在「保持 BLAS K 求和序且快于 vendor sgemm」的新方法族**；三簇新方向全部无法映射为合法格：

| 来源 | title / 方向 | insight（机制） | applicability（本空间） |
|---|---|---|---|
| arXiv:2506.19405（Dumas/Pernet/Sedoglavic 2025）+ fmm.univ-lille.fr（5426 算法集）+ arXiv:2410.20538（Alman-Yu） | fast MM 算法族自动生成 / 领头常数改进 | 递归/低秩乘法算法族以改变乘法与求和结构换乘法次数 | **不可用**：任何改变 K 求和序的实现超容差 14-29×（死族清单同型，A 级证据）——不新建格 |
| arXiv:2509.01229（LiquidGEMM）/ arXiv:2512.04226（tritonBLAS）/ ACM TACO 2025（FlashGEMM）/ arXiv:2504.08009（Ozaki Scheme II） | 超越 vendor BLAS 的内核工作 | W4A8 量化内核（GPU/LLM）；Triton 解析参数选择（GPU）；**序列** matmul 数据复用（CPU）；整数模运算精确仿真 FP | **不可用**：量化改变数值（非法）；Triton 是 GPU 路线（本任务 CPU）；FlashGEMM 需 matmul 序列（本任务单次整乘）；Ozaki 是精度工具非速度工具且舍入序与 BLAS 不同 |
| SIAM Reproducible BLAS / ExBLAS（hal-01539180 等） | 可复现求和序 | 固定求和序换可复现性，性能有代价 | **不可用（反向）**：本任务正确性 ≡ 匹配 vendor BLAS 序，已天然满足；固定序不产生加速 |

→ **不提 new_cell**（无可落地的合法正交方向；「宁缺毋滥」）。本轮 research 的实际作用：为 `terminate_recommend` 提供外部佐证——天花板 = vendor BLAS 平价在文献侧同样成立。

## 4. 格管理动作（单轮有界：soft_weight × 5 ≤ 5，modify_cell × 2 ≤ 2，new_cell 0）

| # | 类型 | 目标 | 动作 | 依据（iter/seq） | rule_id |
|---|---|---|---|---|---|
| 1 | soft_weight | np_matmul\|full_blas_dispatch | weight=2.0（6.0→2.0） | iter13（451.14 拒）+ iter14（440.14 拒，seq 685 gate passed band）+ iter14 buffer_ab 6 形态同 floor ~433 ≈ 精英 430.83 + 该格 method 自身纪律「不复跑、不追噪声」——尾部收割已实测完成，继续采样 = 噪声抽签 | tail_harvest_complete_decay |
| 2 | soft_weight | np_dot\|m_halves | weight=1.0（1.5→1.0） | seq 622 提权依据 = #18669 预 fault 反转「从未实测」；iter13 V2 + iter14 P5/P2 探针已在同族形态证伪（同 floor、minflt/call=0）——提权前提部分坍塌，部分回撤（非全撤，格保留 method 与精英） | tail_premise_falsified_pullback |
| 3 | soft_weight | np_dot\|m_quarters | weight=0.25（0.5→0.25） | visit=1/elite=null（台账无 admit 记录，iter9 窗口）+ 空描述格自举死亡模式（iter12 coding 900s 死亡、iter13/14 builder 900s valid=false）——末次迭代风险控制 | unproductive_cell_risk_decay |
| 4 | soft_weight | einsum_path\|m_halves | weight=0.25（0.5→0.25） | 同上：visit=1/elite=null + einsum 族本征 +11%（iter12 探针 653-670 vs 485）+ 空描述死亡模式 | unproductive_cell_risk_decay |
| 5 | soft_weight | einsum_path\|full_blas_dispatch | weight=0.25（0.5→0.25） | 自身 iter6 no_candidate（台账）+ 空描述死亡模式 | unproductive_cell_risk_decay |
| 6 | modify_cell | np_matmul\|full_blas_dispatch | method 更新为「收口态」：已证伪清单（V2/P1/线程/spacer）+ 基线单跑协议 + 430.83 的安静窗性质说明 | iter13/14 两轮 builder 900s 死亡的直接诱因 = method 仍列出「未测变体」诱导深探研；iter14 debug 已给出全部收口数据 | description_sharpen（收口） |
| 7 | modify_cell | np_dot\|m_halves | method 更新：② 预 fault 变体降级为「确认性单跑（预期带内、不收割）」，主交付 = 基线复测 + ③ m<4 回退鲁棒性；标注本格 floor ~463 不可能接近全局纪录 | iter13/14 探针证伪 #18669 反转侧（P5 452.9-482.4 与 P0 同 floor）；iter010 462.8 vs iter007 预检 437.4 → 结构代价 +26ms | description_sharpen（降级死变体） |
| 8 | terminate_recommend | 全局 | 建议收口 | 预算 14/15（仅剩 1 迭代）；iter11-14 连续 4 轮无新纪录（perf_timeline）；iter12/13/14 三轮 worse_than_elite 同因拒绝；iter14 debug 证精英 430.83 当日不可复现（plain 440-476ms）、30 计时零个 <409.3、floor 433 = sgemm 本体；domain brief A 级死族清单封死 K 序族；本轮 research 无新合法族 | ceiling_reached_and_budget_exhausted |

不变动：np_dot|full（2.0，method 完备，iter007 族精英缺口的最佳补档格——与收权后的冠军并列最高份额）、np_matmul|m_halves/m_quarters（0.5，有精英有产出史）、einsum|m_quarters（0.25，seq 623 已降，无新证据不再动——防震荡）、三个 rowwise 格（0.0 灭绝态保留谱系）。

## 5. 预期效果与证伪

**动作后采样分布（iter15，若照常执行）**：总权重 12.25 → 7.0；np_dot|full 28.6%（原 16.3%）、np_matmul|full 28.6%（原 49.0%）、np_dot|m_halves 14.3%（原 12.2%）；无精英无 method 的四格合计 14.3%（原 14.3%）；冠军格份额近半 → 两格并列，末次采样从「噪声抽签」转为「收档导向」。

**预期变化**：

1. iter15 落 np_dot|full（~29%）：一行内核 + 契约移植，预期产出 435-460ms 候选、补上 np_dot 直派族的精英缺口（iter007 no_candidate 留洞 7 轮）；小概率安静窗 <430.83 创真纪录。
2. iter15 落冠军格（~29%）：method 已更新为「基线单跑」——预期不再出现 900s builder 死亡（探研预算被已证伪清单截断），产出带内读数如实记录。
3. terminate_recommend 触发人工裁决：接受 → 收口归档（最终答案 = iter003 430.83ms，BLAS 平价）；拒绝并加预算 → 需先解决「430.83 为跨会话幸运读数」的测量问题，否则增量预算仍是噪声抽签（已在建议中写明）。

**证伪条件（下轮 adjust / 人工复盘时检验）**：

- 若 iter15 在冠军格产出 <430.83 → 「不可再抽签改善」判断被证伪（安静窗再现）——但只改纪录数字，不改「无合法超越机制」结论；
- 若 iter15 在 np_dot|full 产出 no_candidate 或 900s 死亡 → method 收口不足，需补 modify（现 method 已极简，概率低）；
- 若任一 M-split / einsum 格产出新纪录 → 主要矛盾判定错误（现证据 482-492 vs floor 433 下概率可忽略）。

## 附：过程纪律自检

- 不删格：✓（全部动作为权重/描述级，三灭绝格为历史决策且谱系保留）。
- 证据引用：✓（每个动作 ≥1 条 iter/seq 证据）。
- 防震荡：✓（einsum|m_quarters seq 623 已降、无新证据不再动；np_dot|m_halves 部分回撤基于新证据 #18669 反转侧证伪；冠军格 6.0→2.0 基于三轮新证据且非灭绝）。
- 单轮有界：✓（权重 5 格 ≤5、modify 2 ≤2、new_cell 0、terminate 1）。
- 软硬分界：✓（soft_weight 自主生效；modify_cell / terminate_recommend 走裁决）。
