# research-e2e-observe — super 进化系统端到端测试的观察指标体系

> 2026-11-30 · 调研触发：job_56 端到端复杂测试前置（观察什么=证明符合设计目标）
> 方法：本地代码核实（db.py 六表 DDL / events.py 40 种事件类型 / loop.py 产物路径）+ web 调研 3 轮（QD 评估指标 / LLM agent 可靠性基准 / 混沌工程稳态假设）
> 定位：与 research-e2e-task.md（任务设计）配套——本文回答「观察什么、怎么判、怎么查」

---

## 0. 核心结论

**观察体系 = 三层金字塔**（与系统验证金字塔同构）：

1. **稳态不变式层**（混沌工程借鉴）：系统任何时刻必须成立的不变式（单写者/事件序单调/槽位限额/格锁互斥）——违反即偏离设计，无论功能是否跑通
2. **功能链路层**（A-I 九机制）：每个设计机制的**正向证据链**——DB 行/事件序列/文件结构三物证齐全才算该机制「符合设计」
3. **进化质量层**（QD 指标借鉴）：Coverage（格覆盖）/ Optimization（最优分数）/ QD-Score（格分数总和）的时间曲线——证明「进化在有效推进」而非「只是跑完了」

**判定哲学**：正向证据（机制工作）+ 负向信号（无偏离）双清单；每个断言给出可直接执行的 sqlite3 查询或文件检查命令。

---

## 1. 观察指标清单（A-I 每机制的正向证据链）

约定：`WS` = 工作区根（`<super_root>/<name>`）；`$DB` = `sqlite3 $WS/super.db`。判定标准分**必须**（缺即偏离）与**加分**（出现更好）两级。

### A. 7 步 adjust 流水线（answer-isolation）

| 证据 | 观察什么 | 判定标准（符合设计） |
|---|---|---|
| A1. 运行记录 | `adjusts` 表每次 dispatch 一行 | `SELECT count(*) FROM adjusts;` ≥ 调度轮数；每行 `adjust_dir` 指向真实存在的 `codes/adjust/iter_N/` 目录（**必须**） |
| A2. 步骤 session 完整 | `adjusts.sessions` JSON | 7 步 key 齐全（step1..step7）；**step3 为 3 个独立 session**（三视角并行——key 形如 `step3_p1/p2/p3`，每 key 一个 session_id）（**必须**）；session_id 前缀 `superopt-adjust_` 与 pi session 文件名可对上（加分） |
| A3. 事件序列 | events 表步骤事件 | 每个 iter_N：`adjust_dispatched` → `adjust_step_started(step=1..7)` → `adjust_step_ended`；step3 有 **3 对** started/ended（并行证据）；`adjust_decision` 收尾（**必须**）。序列查询见 §3.2 |
| A4. 产物链 | `codes/adjust/iter_N/` 文件 | `facts.md → perspectives.md → contradiction_{1,2,3}.md → judgment.md → reversal.md` 六类产物 + 7 份 `prompt_adjust_*.md`（step3 三份带 `_{1,2,3}` 后缀）（**必须**） |
| A5. answer-isolation | prompt 文件内容 | step2+ 的 prompt 只含**上游产物路径**（facts_path 等）而非上游全文内联——检查 step3 prompt 是否引用 perspectives 路径（加分：验证独立上下文设计） |
| A6. step1 事实注入 | step1 prompt + facts.md | prompt 含 humans 未应用干预（`pending_interventions`）；facts.md 含「human 期望」节——有人工投递干预时应原样出现（**必须**，配合 D2 干预实验） |
| A7. 决策 schema | `decisions/iter_N.md` | markdown 七节骨架（事实/视角/矛盾/判定/逆转/格动作/实验自改进）+ 内嵌 schema JSON 块（primary_contradiction/cell_actions 等）（**必须**） |

### B. sqlite 六表完整性

| 证据 | 观察什么 | 判定标准 |
|---|---|---|
| B1. 表齐全 | schema | 8 表存在：cells/experiments/adjusts/humans/events/resources（核心六）+ archive_meta/ledger_entries（内部投影）（**必须**） |
| B2. 实验行完整 | experiments 表 | 每个 `codes/N-YYYYMMDDHHMM/` 目录恰好一行；`record_path` 非空且文件存在；**三角色 session 分列**（builder_session_id/coding_session_id/debug_session_id——至少 coding 非空；触发过 debug 的行三列全非空）（**必须**） |
| B3. 格行关联 | cells 表 | `best_experiment_id` 非空的行 → experiments 表存在该行且其 cell_id 一致（双向外键一致性）（**必须**）；`weight` 全部 ≥0（**必须**） |
| B4. 索引一致 | events 表 | `seq` 连续无跳号（AUTOINCREMENT 无删除痕迹）；总事件数 = 各 type 计数和（**必须**） |

### C. 并行实验（二级槽位限额）

| 证据 | 观察什么 | 判定标准 |
|---|---|---|
| C1. 槽位初始化 | resources 表 | `SELECT resource_type, count(*) FROM resources GROUP BY 1;` = 实验槽数 + 环境槽数（实验槽 = max_parallel_experiments；环境槽 = env.json env_slots）（**必须**） |
| C2. 限额生效 | events 时间戳 | 任一时刻在飞实验数 ≤ 实验槽数（`experiment_started` 未配对 `experiment_finished` 的窗口计数）；env_slots=1 时**无重叠窗口**（实验 N 的 finished.ts ≤ 实验 N+1 的 started.ts）（**必须**——这是上轮修复的回归断言） |
| C3. 槽位释放 | resources 表（运行后） | 停机后 holder_experiment_id 全空（无泄漏占用）（**必须**） |

### D. 失败路径与作弊检测

| 证据 | 观察什么 | 判定标准 |
|---|---|---|
| D1. 失败跳过 E2E | events 表 | 出现 `experiment_failed_confirmed`（debug 主动上报）后，同 experiment_id **无** gate_result 事件（跳过端到端验证的证据）（条件性**必须**——任务设计需含死路格；见 task 文档） |
| D2. 干预注入 | humans + events 表 | 人工投递干预（`rick super event`）→ humans 表 intervention 行 applied=0→1；events 表 `intervention` 事件（摄取证据）；下一轮 step1 prompt 含干预原文（**必须**——两轮观察中人工投 1-2 条） |
| D3. 可疑收益复验 | events/experiments | score 涨幅 > 阈值（如 +20%）的实验 → debug session 出现复验行为（experiments 行该实验 debug_session_id 非空 + agent_debug.json 含复验记录）（加分） |
| D4. gates 禁读 | pi session 轨迹 | builder/coding/debug 的 session 工具调用**无 read/bash 命中 `env/gates/` 路径**（hook 拦截生效）；adjust_* 角色可有（**必须**——抽查 2-3 个 session） |
| D5. exploit 全拒 | 门先证红 | 启动时 preverify_red 结果 ok=true（exploits 全拒+金样例零误杀）记录在启动日志/super.json（**必须**） |

### E. 多维目标融合分

| 证据 | 观察什么 | 判定标准 |
|---|---|---|
| E1. gate_check JSON | gate_result 事件 payload | 含 `metrics`（多指标字典）+ `constraints`（每约束 value/target/satisfied）+ 融合 `score`；约束违反 → passed=false 且 score<0（**必须**——任务设计需 ≥2 指标+≥1 约束） |
| E2. 约束拒绝路径 | events | 至少一条 gate_result `passed=false` 且 failure_stage 指向约束类（条件性加分——任务含可违反约束的死路） |

### F. 验证金字塔（L1/L3/E2E 分层）

| 证据 | 观察什么 | 判定标准 |
|---|---|---|
| F1. L1 预检 | builder session 工具调用 | 出现 `verify.py --candidate ... --level L1`（秒级预检——builder 可选行为）（加分） |
| F2. E2E 终验 | gate_result 事件 | 每个 admitted/rejected 实验恰有一条 E2E gate_result（stage=passed/failed；失败跳过路径除外）（**必须**） |
| F3. level 参数透传 | skills 调用 | L1/L3 调用与 E2E 的 benchmark 读数同带（同数量级、无数量级漂移）（加分） |

### G. 权重生态

| 证据 | 观察什么 | 判定标准 |
|---|---|---|
| G1. 权重事件 | events 表 weight_changed | 每次 cells.weight 变化恰有一条 weight_changed（old/new/reason/rule_id，source=agent:adjust）（**必须**）；无事件突变即偏离（见 §2 负向） |
| G2. 采样分布生效 | events 表 cell_sampled | weight 调整后，后续 cell_sampled 的 `weight_share` 与新权重比例一致（升权格采样份额上升）（**必须**） |
| G3. 防震荡 | weight_changed 序列 | 同一格连续降权之间有新证据间隔（reason 引用不同 iter/seq）；无「降了立刻升回」震荡对（**必须**） |
| G4. 灭绝与复活 | cells+events | 若出现 weight=0（灭绝）：`invalidation`/后续无采样；复活路径（resurrection 事件）可选（条件性加分——任务需有死路格） |

### H. 饱和与矛盾转移

| 证据 | 观察什么 | 判定标准 |
|---|---|---|
| H1. 饱和信号 | step1 prompt 饱和注入 | 连续无新纪录轮后 step1 prompt 的 saturation 非空（S1/S2/S3 计算证据）（条件性**必须**——长跑任务后半段应出现） |
| H2. 加格决策 | events/proposals | 饱和后 adjust proposals 含 expand_cell/new_cell（走裁决链：proposal → adjudicated → cell_created）；或 terminate_recommend（收手建议落 humans 提案队列）（条件性加分） |
| H3. 格数增长 | cells 表 | 新格行出现 + `cell_created` 事件 + feature_space.json 轴值更新（热重载 `parallel_degraded`/重载无异常）（条件性加分） |

### I. 实验自改进（step7）

| 证据 | 观察什么 | 判定标准 |
|---|---|---|
| I1. 领域知识沉淀 | env/domain_knowledge.md | 出现 step7 追加节（`## adjust 实验知识沉淀 · iter_N`）；只追加不修改既有内容（**加分**——bug 解法类发现） |
| I2. 提案队列 | humans 表 proposal 行 | step7 的 skill/verifier 反思进 proposal 队列（applied=0 待 human 裁决）+ `env_proposal_queued` 事件（**加分**） |
| I3. 直写边界 | domain_knowledge diff | step7 直写仅限追加节；gates/verify 逻辑无改动（验证器不被被测方修改——D4 同源）（**必须**） |

---

## 2. 异常信号（负向检查清单——出现即偏离设计）

| # | 异常信号 | 检测方法 | 指向的设计偏离 |
|---|---|---|---|
| N1 | **同格无锁并发** | 同一 cell_id 的 experiment_started 窗口重叠（两实验同格同飞） | 格锁失效（C 层） |
| N2 | **幽灵实验** | codes/ 有目录但 experiments 表无行（或反之） | 实验注册链断裂 / 槽位泄漏复发（上轮 bug4 回归） |
| N3 | **目录轰炸** | `ls $WS/codes/ \| wc -l` ≫ experiments 行数 | 槽满先注册后拿槽的顺序缺陷回归（上轮 bug4） |
| N4 | **权重突变无事件** | cells.weight 与最后一条 weight_changed.new_weight 不一致；或有权重差但无事件 | 单写者纪律被绕过（G1） |
| N5 | **session 缺失** | adjust 步骤 event 有 started 无 ended；adjusts.sessions 缺某 step key | 流水线断裂未降级记账（A3/降级协议：应有 adjust_step_degraded 配对） |
| N6 | **步骤乱序** | events 按 seq 排序后 step 序列非 1→7（step4 先于 step3 ended 等） | 流水线串行约束被破坏（A 层） |
| N7 | **实验三 session 缺失** | experiments 行 builder/coding session_id 全空 | agent session 记账链断（B2） |
| N8 | **槽位泄漏** | 停机后 resources.holder 非空 | release 链断裂（C3） |
| N9 | **events 跳号/回退** | seq 不连续或 ts 与 seq 大面积乱序 | 单写者被破坏（多调度器并发——进程锁失效，上轮 bug6 相关） |
| N10 | **gate 旁路** | admitted 实验无 E2E gate_result 且无 experiment_failed_confirmed 兜底 | 验证跳过协议被误用（D1/F2） |
| N11 | **验证器被改** | git diff env/gates/ 非空（实验期间） | step7 直写边界失守 / 作弊通道（I3/D4） |
| N12 | **humans 双消费** | 同一 humans 行被两次干预事件引用 | 干预摄取幂等性破坏 |
| N13 | **step3 并行缺失** | contradiction 三份产物时间戳同线程串行（started/ended 交错无重叠） | 三视角并行退化为串行（性能回归非正确性，降级为观察项） |
| N14 | **权重震荡** | 同格 N 轮内 weight_changed 方向交替 ≥2 次且 reason 无新证据 | 防震荡纪律失效（G3） |

---

## 3. 观察方法（可直接执行的检查模板）

### 3.1 sqlite 查询模板（六表直查）

```bash
DB="sqlite3 $WS/super.db"

# ① 进化质量三指标（QD 借鉴——时间序列观察）
$DB "SELECT count(*) FROM cells WHERE best_experiment_id IS NOT NULL;"          # Coverage（有精英格数）
$DB "SELECT count(*) FROM cells;"                                               # 总格数（Coverage 分母）
$DB "SELECT max(score) FROM experiments WHERE status='admitted';"               # Optimization（全局最优）
$DB "SELECT sum(score) FROM experiments e WHERE status='admitted' AND experiment_id IN (SELECT best_experiment_id FROM cells WHERE best_experiment_id IS NOT NULL);"  # QD-Score 近似（格精英分总和）

# ② 流水线完整性（A 层）
$DB "SELECT adjust_id, iteration, json_extract(sessions,'$.step1') IS NOT NULL AS s1, length(sessions) FROM adjusts;"
$DB "SELECT iteration, group_concat(step) FROM events WHERE event_type='adjust_step_started' GROUP BY iteration;"

# ③ 事件时间线（按类型过滤）
$DB "SELECT seq, ts, event_type, substr(payload,1,80) FROM events WHERE event_type NOT IN ('heartbeat') ORDER BY seq;"

# ④ 槽位限额（C 层——env_slots=1 时无重叠）
$DB "SELECT e1.experiment_id, e2.experiment_id FROM events e1, events e2 WHERE e1.event_type='experiment_started' AND e2.event_type='experiment_started' AND e1.seq < e2.seq AND EXISTS (SELECT 1 FROM events f WHERE f.event_type='experiment_finished' AND f.experiment_id=e1.experiment_id AND f.seq > e2.seq);"

# ⑤ 权重生态（G 层）
$DB "SELECT c.cell_id, c.weight, (SELECT new_weight FROM events w WHERE w.event_type='weight_changed' AND json_extract(w.payload,'$.cell_id')=c.cell_id ORDER BY w.seq DESC LIMIT 1) AS last_event_w FROM cells c;"

# ⑥ 失败路径（D 层）
$DB "SELECT experiment_id, status, score FROM experiments WHERE status NOT IN ('admitted');"
$DB "SELECT e.experiment_id FROM events e WHERE e.event_type='experiment_failed_confirmed' AND NOT EXISTS (SELECT 1 FROM events g WHERE g.event_type='gate_result' AND g.experiment_id=e.experiment_id);"

# ⑦ 三角色 session 完整性（B2）
$DB "SELECT experiment_id, builder_session_id IS NOT NULL, coding_session_id IS NOT NULL, debug_session_id IS NOT NULL FROM experiments;"
```

### 3.2 事件流序列断言（关键序列）

1. **实验链序列**（每实验）：`experiment_started → iteration_started → cell_sampled → agent_started(builder) → agent_ended → agent_started(coding) → agent_ended → candidate_produced → [agent_started(debug) → agent_ended] → gate_result → [elite_admitted|experiment_failed_confirmed] → iteration_ended → experiment_finished`
2. **adjust 流水线序列**（每轮）：`adjust_dispatched → (adjust_step_started(1) → adjust_step_ended(1)) → ... → step3×3 并行 → ... → adjust_decision`；降级时配对 `adjust_step_degraded`
3. **硬动作裁决链**（若触发）：`proposal → adjudicated → cell_created|cell_modified|weight_changed`

### 3.3 文件检查点

| 路径 | 结构断言 |
|---|---|
| `codes/N-*/experiment_record.md` | 四节结构（实验设计/实现与过程/排障记录/结论与门禁）+ frontmatter（cell_id/hypothesis/status/score） |
| `codes/adjust/iter_N/` | 六类产物 + 7 份 prompt（step3 三份带后缀） |
| `decisions/iter_N.md` | 七节骨架 + 内嵌 schema JSON 块 |
| `env/domain_knowledge.md` | step7 只追加节；`git diff` 验证 gates/ 无改动 |
| `env/feature_space.json` | expand 后轴值新增（热重载一致性） |
| pi sessions（`~/.rick/pi/agent/sessions/*codes-N-*/`） | 三角色 jsonl 齐全；抽查 builder/coding 无 gates/ 路径调用 |

---

## 4. 行业先例映射（可借鉴的观察框架）

| 先例 | 核心思想 | 借鉴到 super |
|---|---|---|
| **QD 评估三指标**（Pugh 2016 / Cully 2018 / QD survey, A 级） | Coverage（非空格数）+ Optimization（最优目标值）+ QD-Score（格分数总和）——三维刻画「多样性×质量」 | §3.1① 进化质量三指标：cells 有精英格数 / max(score) / 格精英分总和——**时间序列观察**（曲线应上升，Coverage 应向全格推进） |
| **ReliabilityBench**（arXiv:2601.06112，B 级） | 一致性（k-trial pass）/鲁棒性（扰动级）/容错（故障注入）三维 | 失败路径 D 层=容错维度；同格重复采样的一致性（同码重测 score 波动带）=一致性维度可观察 |
| **混沌工程稳态假设**（principlesofchaos.org + AWS，A 级） | 定义稳态→假设扰动下保持→引入故障→观察偏离 | §2 负向清单=「稳态不变式」：单写者/事件序/槽位限额/格锁——任何一条违反即系统偏离设计的强信号（与功能是否跑通无关） |
| **SRE 四金信号**（Google SRE，B 级） | 延迟/流量/错误/饱和度 | super 对应物：agent wall_ms（延迟）/事件速率（流量）/agent_ended.valid=false 率（错误）/槽位占用率（饱和度）——cost_report.py 已承载大部分 |
| **pass@k 使命级阈值**（企业多维框架，B 级） | 单次成功率掩盖脆弱性——连续 k 次成功才算可靠 | 长跑任务同格多访的 admit 率、同码重测稳定性（noise_calibrated 事件的噪声带） |

---

## 5. R7 上报项（无法收敛到强证据的部分）

1. **「矛盾转移」的客观判定**：H 层的饱和→矛盾转移是语义事件（step4 判定），无独立数值锚——只能观察「饱和注入→加格/收手决策」的行为链，无法验证判定本身的正确性（需对照实验，超出单次 e2e 观察）
2. **answer-isolation 效果**：A5 只能验证「产物经文件传递」的机制形态，「隔离是否真的防了污染」需要 A/B 对照（有/无隔离的决策质量 diff）——单次观察只能给机制证据
3. **step3 并行的正确性**（N13 降级为观察项）：三线程并行的事件交错可证明并行发生，但「并行结果与串行等价」无单次观察法

## 6. Gaps

- **观察成本权衡**：全量负向检查（§2）若逐条自动化需专用观察脚本；单次 e2e 建议人工抽查 N1-N5/N8/N10/N11（高危项）+ SQL 模板批量跑正向（§3.1）
- **pi session 抽查的代表性**：D4/A5 的 session 轨迹检查是抽样性的——全量扫描工具（adjust_trace.py 扩展）可作为后续工程项
- **进化质量基线**：QD 三指标无「应达到多少」的绝对标准——用时间序列单调性（非下降）+ 与任务设计预期（task 文档）的相对对照判定
