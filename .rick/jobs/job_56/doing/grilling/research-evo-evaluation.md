# research-evo-evaluation — 进化搜索/QD 系统的质量评估框架调研

> 2026-11-22 · 调研触发：job_56（super 进化系统需要专门评估框架——传统 ML 指标不适用）
> 方法：web_search 3 轮（Exa，20+ 检索词）+ arXiv/GitHub 直抓核实
> 消费方：super 系统的评估报告设计（report.py）+ super-tech skill 更新

---

## Q1: QD 搜索的标准评估指标

### 1.1 QD survey 2025 的指标体系（A 级）

来源：Quality Diversity Algorithm Design（arXiv:2507.15826, 2025 survey）+ Mouret & Clune 2015（QD 开山）

| 指标 | 定义 | 学术出处 |
|---|---|---|
| **Coverage** | 非空格数 / 总格数 | Mouret & Clune 2015 |
| **QD-Score** | Σ(每格精英的 fitness) | Pugh et al. 2016 |
| **Optimization**（最优目标值） | 全格最高 fitness | Mouret & Clune 2015 |
| **Relative Coverage** | coverage / (uniform random 的 coverage) | QD survey 2025 |
| **Relative Optimization** | opt / (CMA-ES 等单目标优化器的 opt) | QD survey 2025 |
| **Diversity Entropy** | 行为空间的填充密度（Shannon entropy of archive distribution） | arXiv:2403.18872 |
| **Convergence Efficiency** | 达到 X% coverage 所需迭代数 | QD survey 2025 |
| **Archive Quality** | 精英的平均 fitness（不是 max——均值反映整体质量） | QD survey 2025 |

### 1.2 代码生成 QD 系统的指标扩展

来源：FunSearch（Nature 2024）+ AlphaEvolve（arXiv:2506.13131）+ KernelFoundry（arXiv:2610.16281）

| 指标 | 定义 | 使用者 |
|---|---|---|
| **Discovery Rate** | 新解（新格或精英替换）的发现频率——每 N 次 LLM 调用发现多少新精英 | FunSearch / AlphaEvolve |
| **Improvement Magnitude** | 从初始解到最优的提升倍数 | AlphaEvolve |
| **Convergence Curve** | per-iteration 最优值的改善曲线（不只报最终值） | 所有进化系统 |
| **Token Efficiency** | 每 1M token 产生多少有效精英 | KernelFoundry |
| **Search Budget Efficiency** | 在给定 token/GPU 预算下找到多少格 | OpenEvolve |

---

## Q2: FunSearch / AlphaEvolve / OpenEvolve / KernelFoundry 的自评估方式

### 2.1 FunSearch（DeepMind, Nature 2024, A 级）

- **指标**：每个数学问题单独报告——cap set problem（找到更大的 cap set）/ bin packing（更优的启发式）
- **对比基线**：纯随机采样 + LLM one-shot（同预算下）
- **关键设计**：报告了「搜索轮次 vs 解质量」的改善曲线——不只报最终值
- **迁移**：不报告跨问题迁移

### 2.2 AlphaEvolve（DeepMind, arXiv:2506.13131, A 级）

- **指标**：矩阵乘 4×4 复数乘法（11 次 mult → 48 次 mult improvement）/ data center scheduling（0.7% 全局回收率提升）/ kernel optimization（平均 23% 加速）
- **对比基线**：SOTA 算法（Strassen for matmul）+ 生产环境 baseline
- **关键设计**：报告了「进化代数 vs 目标值」曲线——展示了渐进改善过程
- **局限**：不公开题库，不可复现

### 2.3 OpenEvolve（开源, GitHub: codelion/openevolve, B 级）

- **指标**：circle packing（packing density）/ 排序算法（压缩比）
- **对比基线**：初始解（naive）
- **关键设计**：报告了「iteration vs fitness」改善曲线 + MAP-Elites coverage
- **无 GPU kernel 评测**——只有数学/组合优化问题

### 2.4 KernelFoundry（Basis, arXiv:2610.16281, A 级）

- **指标**：KernelBench L1 的 fast_p（与 one-shot LLM 同指标）
- **对比基线**：one-shot LLM（同底座模型）+ KernelBench 已有 baseline
- **关键设计**：报告了「进化代数 vs speedup」曲线——展示进化迭代的增量改善
- **局限**：没有定义「进化系统专属协议」（无 token 成本/无预算公平对比）

### 2.5 Eureka（NVIDIA, arXiv:2310.12931, A 级）

- **指标**：RL reward 改进幅度（Isaac Gym 环境）
- **对比基线**：人工设计的 reward 函数
- **不适用**：非代码优化系统（是 reward 函数搜索）

---

## Q3: 搜索效率指标（搜索过程的质量）

来源：QD survey 2025 + CompoundingKernel（arXiv:2611.19230）+ OpenEvolve 实测

| 指标 | 定义 | 计算方式 | 我们的 DB 对应 |
|---|---|---|---|
| **Sample Efficiency** | 每 N 次 LLM 调用产生多少 admitted 精英 | admitted_count / total_agent_calls | `SELECT count(*) FROM experiments WHERE status='admitted'` / `SELECT count(*) FROM events WHERE event_type='agent_started'` |
| **Improvement Rate** | 每轮迭代的精英分数改善幅度 | Δ(格精英分) / iteration | `SELECT cell_id, score, iteration FROM experiments WHERE status='admitted' ORDER BY iteration` → 按 iteration 分组计算均值改善 |
| **Novelty Rate** | 新格首次被占入的频率 | new_cell_admits / total_experiments | `SELECT iteration, count(DISTINCT cell_id) FROM experiments WHERE status='admitted' GROUP BY iteration` → 新 cell 数 |
| **Elite Replacement Rate** | 精英被替换的频率（格内竞争强度） | elite_replaced / total_admits | `SELECT count(*) FROM events WHERE event_type='elite_replaced'` |
| **Exploration-Exploitation Ratio** | 新格探索 vs 精英精化的实验比例 | new_cell_experiments / revisit_experiments | `SELECT cell_id, count(*) FROM experiments GROUP BY cell_id` → count=1（首访） vs count>1（重访） |
| **Convergence Curve** | 最优分数随迭代的变化 | per-iteration max(score) | `SELECT iteration, max(score) FROM experiments WHERE status='admitted' GROUP BY iteration` |

---

## Q4: 自适应/自改进能力的评估

### 4.1 学术框架

来源：RRSI（Recursive Self-Improvement, arXiv:2609.24972）+ SIM（Self-Improving Memory, arXiv:2611.18856）

| 维度 | 定义 | 评估方式 |
|---|---|---|
| **Adaptive Sampling Quality** | adjust 权重调节是否改善了后续采样效率 | 对比：调节前 vs 调节后的 sample efficiency（同格比较） |
| **Budget Utilization** | 总预算中有多少产生了有效改进 | admitted_token / total_token |
| **Self-Correction Rate** | 系统自我纠错的成功率 | budget_adjusted 次数 + 每次的效果（改善/恶化） |
| **Decision Accuracy** | adjust 的格动作是否被后续证据支持 | 被升权格的后续产出 / 被降权格的事后引用（hindsight） |
| **Knowledge Accumulation** | domain_knowledge 增长 + 被引用率 | domain_knowledge 字节数增长曲线 + agent prompt 中的引用次数 |

### 4.2 我们 super.db 可直接计算的指标

| 指标 | SQL |
|---|---|
| adjust 决策准确率 | `SELECT count(*) FROM events WHERE event_type='weight_changed' AND json_extract(payload,'$.new_weight') > json_extract(payload,'$.old_weight')` vs 后续 admitted |
| budget 调节效果 | `SELECT * FROM events WHERE event_type='budget_adjusted'` → 对比调节前后的 agent timeout 率 |
| 知识沉淀率 | `SELECT length(content) FROM humans WHERE queue_type='proposal'` 增长曲线 |
| 提案消费率 | `SELECT count(*) FROM humans WHERE applied=1 / (SELECT count(*) FROM humans WHERE queue_type='proposal')` |

---

## Q5: 长跑/无人值守稳定性指标

来源：SRE 四金信号（Google SRE）+ chaos engineering + CompoundingKernel 的长跑实证

| 指标 | 定义 | 我们的 DB 对应 |
|---|---|---|
| **Monotonic Improvement** | 后期最优 ≥ 前期最优（不退化） | `SELECT iteration, max(score) FROM experiments WHERE status='admitted' GROUP BY iteration` → 单调不减检查 |
| **Budget Utilization** | 有效实验 / 总实验 | `SELECT count(*) FROM experiments WHERE status='admitted' / SELECT count(*) FROM experiments` |
| **Infra Failure Rate** | 基础设施失败（remote_link/OOM/timeout）占实验失败的比例 | `SELECT count(*) FROM events WHERE event_type='gate_result' AND json_extract(payload,'$.failure_stage') IN ('remote_link','environment_oom')` |
| **Recovery Rate** | 中断后恢复正常运行的能力 | interrupted 后的下一个 admitted 实验出现的时间间隔 |
| **Knowledge Growth** | domain_knowledge.md 的字节数增长 + 被引用率 | `wc -c env/domain_knowledge.md` + `grep -c "domain_knowledge" codes/*/prompt_*.md` |
| **Token Efficiency Curve** | per-iteration token 消耗是否稳定或有改善趋势 | `SELECT iteration, sum(json_extract(payload,'$.total_tokens')) FROM events WHERE event_type='agent_ended' GROUP BY iteration` |
| **Adjust Decision Quality** | adjust 的格动作是否有效（升权格后续改善/降权格无后续产出） | weight_changed 事件 → 对比后续 admitted |

---

## Q6: super 进化系统评估报告模板（四维度）

### 维度 1: 搜索过程质量（Search Quality）

```
指标                  SQL（super.db 直查）
────────────────────────────────────────
Coverage              SELECT count(*) FROM cells WHERE best_experiment_id IS NOT NULL
                      / SELECT count(*) FROM cells
QD-Score              Σ(每格精英的 speedup)
Optimization          max(score) FROM experiments WHERE status='admitted'
Archive Mean Quality  avg(score) FROM experiments WHERE status='admitted'
Sample Efficiency     admitted / total_experiments
Novelty Rate          新格首次占入频率（per-iteration）
Elite Replacement     elite_replaced 事件计数
Convergence Curve     per-iteration max(score) 曲线
```

### 维度 2: 自适应能力（Adaptivity）

```
指标                  数据来源
────────────────────────────────────────
Adjust 决策次数       adjusts 表行数
格动作数              weight_changed / cell_modified 事件
Budget 调节次数       budget_adjusted 事件
决策准确率            升权格的后续 admitted 率
自纠错率              升权被证伪后回调的次数
hindsight 准确率      _hindsight_evidence 的 evidence 质量
```

### 维度 3: 知识积累（Knowledge）

```
指标                  数据来源
────────────────────────────────────────
Domain Knowledge      env/domain_knowledge.md 字节数 + 节标题数
知识引用率            agent prompt 中引用 domain_knowledge 的次数
提案产出数            humans WHERE queue_type='proposal'
提案消费率            humans WHERE applied=1 / total proposals
跨格迁移              同一 domain_knowledge 条目被不同格的 agent 引用
```

### 维度 4: 长跑稳定性（Stability）

```
指标                  SQL / 文件
────────────────────────────────────────
Monotonic Improvement per-iteration max(score) 单调不减
Budget Utilization    admitted_token / total_token
Infra Failure Rate    remote_link + OOM 占比
Recovery Rate         interrupted → 下一个 admitted 的时间间隔
Token Efficiency      per-iteration token 消耗趋势
Adjust Interval       adjusts 表相邻 iteration 的间隔
```

---

## R7 上报项

1. KernelFoundry 的完整评测数据表不在 GitHub 公开——需要 arXiv 全文精读
2. AlphaEvolve 的评测协议不可复现（不公开题库）——只能参考其报告格式
3. QD survey 2025 的相对覆盖/相对最优的具体归一化方法需要全文精读
4. 「搜索效率」的统一定义在学术界无共识——不同系统用不同定义（token/调用/时间）
5. 自适应能力的评估框架是**空白**——RRSI/SIM 只讨论了风险（过拟合/记忆退化），没有正面评估框架

## Gaps

- **没有统一的「进化系统质量评估」协议**——QD survey 关注搜索算法指标，FunSearch/AlphaEvolve 关注任务结果，KernelFoundry 关注 KernelBench 分数——没有系统性的四维度评估
- **自适应能力的评估是空白**——学术上 RRSI 讨论了递归自改进的风险但没有正面评估框架；我们的 adjust 层的自适应能力（权重调节/预算热调/知识沉淀）是**原创设计**，没有先例评估协议
- **知识积累的量化**——domain_knowledge 的「被引用率」在代码搜索系统中没有学术先例（CompoundingKernel 有 knowledge transfer 的定性分析但没有量化指标）
- **我们的评估报告如果开源——填补进化系统评估协议的空白**（与 agent kernel benchmark 一样是原创贡献机会）
