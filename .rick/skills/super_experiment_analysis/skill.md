# skill:super-experiment-analysis（adjust 第一步：区间实验全景分析）

> 服务对象：**adjust agent**（super 进化调度器的解空间调整者）
> 加载方式：adjust 的系统提示词声明本文件路径 + 触发条件；执行到本步骤时用 `read` 读取（不内联）

## 触发场景

每轮 adjust 开始时加载（调度器在 `adjust_interval` 迭代边界派发 adjust agent）。信号：

- ctx 中有 `experiment_records`（上次 adjust 到本次之间**全部**实验记录摘要）与 `decision_history`（历史决策）
- 你需要先回答「这段时间系统发生了什么」才能做任何决策——**不允许跳过分析直接调权**

## 预期效果

产出一份**区间实验全景摘要**，作为后续 SENSE 的输入：

- 逐实验一行：`iter / cell / 假设 / 门禁结果(failure_stage) / perf / delta / 失败归因`
- 归纳出 **1-3 个主要矛盾候选**（全局性、根本性、决定性的性能成因），每个候选附证据（iter 号 / seq 号）
- 标出「高风险重复模式」（同一失败原因出现 ≥2 次）与「空白方向」（现有格未覆盖的维度）

## 核心内容

### 读什么（按优先级）

1. **`experiment_records`**（ctx 注入的区间摘要）——先看它建立全景。
2. **实验记录原文**：摘要被截断或某实验「记录说失败但没说清为什么」时，直接 `read codes/iter{N}-{uuid}/experiment_record.md` 全文；优先读**最近的**与**失败的**实验。
3. **agent session（按需）**：ctx 的 `experiment_sessions` 给出区间内 agent 的 pi session 路径清单（pi runtime 的 sessions 目录按 cwd 匹配 `codes/iter*`）。**只在实验记录不足时读**（例：记录写「测试红了」但没有失败文本、builder 调研结论与实现不一致）。读 session 时关注 assistant 的关键判断与 toolResult 的真实读数，不要逐条通读。
4. **`decision_history`**：上次及以前 adjust 的决策记录——用来避免「重复同一处方」（同一降权/加格动作连续多轮无效 = 信号）。
5. **`ledger.jsonl`**（区间台账）：`verdict` / `perf_delta` / `reject_reason` 是最快的横截面。

### 怎么分析

- **分三类**：
  - 探索性成功：入格（`admit`/`replace`）——记录 perf 与相对上一最优的 delta
  - 无效实验：自检/门禁失败——按 `failure_stage` 归类（正确性 vs 性能 vs 软门禁 vs 环境）
  - 劣化实验：门禁通过但 perf 退化（soft_gate 拒 / 容差带内）——记录退化的量级
- **找重复模式**：同一 `failure_stage` 或同一归因关键词出现 ≥2 次 → 系统性矛盾候选（例：「三个不同格都在装配阶段劣化」→ 装配范式是共性瓶颈）
- **找空白**：对照 `cell_descriptions` 与特征轴定义，列出「有理论潜力但没有任何格覆盖」的维度（例：全部格都在 BLAS 派发族内，无人试非 BLAS 的数值方法）
- **判趋势**：结合 `perf_timeline`（若 ctx 提供）判断「最优是否长期停滞」（连续多轮无新纪录 = 探索枯竭信号）

### 输出（写给 SENSE 的输入）

在决策记录（ctx 的 `decisions_path`）中写「## 一、区间实验全景」一节：

```markdown
## 一、区间实验全景（iter A → B）
| iter | cell | 假设 | 门禁 | perf | delta | 归因 |
|---|---|---|---|---|---|---|
| 12 | np_dot\|m_quarters | ... | pass | 484ms | +54 | 入格 |
| 13 | einsum\|m_halves | ... | soft_gate | 512ms | +82 | 装配税 |

**重复模式**：<同一失败原因的 ≥2 次出现>
**空白方向**：<未覆盖维度>
**主要矛盾候选**（交 SENSE 裁决）：
1. <候选一>（证据：iter12 seq…）
2. <候选二>（证据：…）
```

### 纪律

- 只用记录里**真实存在**的数据；缺数据就写「无数据」，禁止编造读数
- 本步骤只做**分析**，不做动作（调权/加格/改描述都是后续 `super_cell_management` 的职责）——提前下结论会绕过 SENSE 的矛盾判定
- 读 session 有 token 成本：每轮最多读 2-3 个最关键 session 的关键片段
