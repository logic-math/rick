# skill:super-experiment-design（builder：把灵感变成可复现实验）

> 服务对象：**builder agent**
> 加载方式：builder 提示词声明路径 + 触发条件；research 调研报告完成后 `read` 本文件

## 触发场景

builder 已完成 research（按 `research_brief` 的 method/sources/focus 检索，产出 `research_report.md`），手上有了事实与灵感，此时需要把「想法」定形成**一次可复现实验**：

- 信号：ctx 有 `target_record`（目标格精英实验记录）/`neighbor_samples`（两个随机相邻格的描述+精英代码+实验记录）/`research_report_path`/`experiment_record_path`
- 反例：如果你还没有调研报告（或报告里只有泛泛结论），先回 research 步骤——没有事实支撑的实验设计 = 浪费一轮编码

## 预期效果

产出两份对齐的产物：

1. **实验记录**（`experiment_record.md`，六段）——给 coding/debug/adjust 与下一轮复现用的**完整上下文**
2. **实验计划**（`experiment_plan.json`）——结构化字段（idea/target_cell/parents/plan_steps/expected_behavior + hypothesis/validation_logic）

判据：coding 拿到它可以直接开工；框架拿到它知道验证什么；下一轮 adjust 读到它知道「这个实验假设什么、验证了没有」。

## 核心内容

### 实验记录六段模板（写入 ctx 的 `experiment_record_path`）

```markdown
# 实验记录 iter{N} · {cell_id}

## 1. 背景与价值
<现状读数（profile/精英 perf）+ 缺口（主要矛盾）+ 论文/调研依据（research_report 的关键结论）>
<为什么这个实验值得占用一轮预算>

## 2. 实验假设
<一句话、可证伪、带方向与量级：如「改用 F 序装配后 @8192 中位 < 480ms（当前 512ms）」>

## 3. 验证逻辑
- 支持假设：<观察 A；量级 B>
- 否定假设：<观察 C；量级 D>
- 对照（如做 A/B）：<对照项是什么>

## 4. 具体实验步骤
1. <改哪个文件/哪一段（父本 → 目标改动）>
2. <实现要点/采用的机制（引用调研报告中的落地细节）>
3. <怎么跑：自检口径/规模/次数>

## 5. 观测现象
<预期看到什么指标变化；出现什么信号说明机制生效/失效；预期量级>

## 6. 执行过程与结论
<!-- 占位：coding 追加实现要点与测试红绿；debug 追加卡点/根因/解决；门禁结果 -->
```

### 设计纪律

- **一迭代一实验**（单变量优先）：多变更必须拆成 A/B 对照并在第 3 段写明；否则归因失效（无法判断哪个变更起了作用）
- **假设可证伪**：必须有明确的证伪线（数字/现象）；「性能会更好」不是假设
- **指标可观测**：用 env 的 `benchmark`/`profile` 口径（ctx 的 `gate_defs` 给出容差/规模/K）——不要发明无法测量的指标
- **步骤可复现**：写清文件/函数/参数，下一篇 agent（coding）不需要再猜
- **对齐调研**：第 1、4 段必须能追溯到 `research_report.md` 的具体结论（哪篇论文的哪个机制、哪个代码样例的哪个写法）——**不允许凭空设计**

### 与上下游的交接

- **上游**（目标格 + 两个随机相邻格，即 ctx 的 `neighbor_samples`）：相邻格的描述/精英代码/实验记录是**跨格灵感源**——把它们与目标格的差异写进第 1 段（「邻格 X 用了转置融合，本格未用」）
- **下游**（coding）：按 rick TDD canon（`internal/prompt/templates/skills/tdd-zh.md`）实现——先写测试看红再实现；coding 会补充本记录第 6 段
- **下游**（debug）：失败时用本记录的第 2/3 段判断「假设被否定还是实现有误」
- **下游**（adjust）：下一轮全局分析会读本记录——假设/验证逻辑写得清，adjust 才能判断该方向是否值得继续

### 输出实验计划（experiment_plan.json）

```json
{
  "idea": "<一句话>",
  "target_cell": "<cell_id>",
  "parents": ["codes/iter{N}/solution.py"],
  "plan_steps": ["<3-6 步，coding 的执行清单>"],
  "expected_behavior": "<与记录第 5 段对齐>",
  "hypothesis": "<与记录第 2 段对齐>",
  "validation_logic": "<与记录第 3 段对齐>"
}
```

### 纪律（防跑偏）

- 不得把「多个方向的合集」塞进一次实验（拆到后续迭代）
- 不得修改 env 定义（门禁/容差/特征空间）——那是 adjust 与 gen_env 的职责
- 若调研结论显示「该方向已被 env 现有证据否定」（axis_perf/ledger 里有记录）→ 换方向，别重复已知死路
