# Adjust 决策层运行时级 Review — Bug 报告

> job_56 · review-agent · 2026-11-22 · 范围：adjust 决策层（信息流/行为轨迹/输出消费/调度框架/human 通路/写者竞态）
> 方法：源码走读（loop.py adjust 相关 12 函数）+ super.db 直查（adjusts/humans/events 表）+ pi session 文件（~/.rick/pi/agent/sessions）+ decisions/codes/adjust 产物链
> 工作区：/Users/logic/workdir/MT_INFER/.rick/super/kernelbench-dev

---

## P0 — 严重（数据丢失/功能断路）

### [A-01] P0 · C(输出消费) — Go 侧 SuperEventIn 写 interventions/ 目录但 py 侧只读 humans 表 DB——human 干预通路完全断裂

**证据**：
- Go `internal/handler/super.go:953-982`：`SuperEventIn` 写 `interventions/<ts>-<rand>.json` 文件到目录
- py `super/superopt/intervention.py:77-83`：`InterventionQueue.push` → `self._db.push_intervention()`（写 humans 表）
- py `super/superopt/db.py` `drain_interventions()`：`SELECT * FROM humans WHERE queue_type='intervention' AND applied=0`
- **`rick super event` CLI 产生的干预永远进不了 humans 表**——py 调度器的 `_drain_interventions()` 走 `self.queue.drain()` → DB 查询 → 查不到目录文件
- 实测：工作区 `interventions/` 目录 0 文件、humans 表 intervention 类 0 行——通路从未走通
- **影响**：`rick super event --type stop` 发的 stop 干预**永远不会被调度器消费**（stop_requested 不会被置 true）——human 的停机指令断路

**修复方案**：Go `SuperEventIn` 改为写 humans 表（或双写：目录文件 + sqlite INSERT）；或 py `InterventionQueue.drain()` 兼容读目录文件。

---

### [A-02] P0 · F(写者竞态) — 调度器重启后 `_adjust_thread=None` 导致同一 iter 的 adjust 流水线多次并发——产物互覆、decisions 互写、humans 提案空壳行

**证据**：
- `_maybe_start_adjust` 单飞约束 `if self._adjust_thread is not None: return`（loop.py:2379）——**`_adjust_thread` 是实例属性，调度器重启后新实例的值为 None**，上一进程的在飞 pipeline 信息丢失
- events 表 iter_14 有 **9 次 adjust_dispatched**（seq 1897/1916/1936/1960/1973/2001/2014/2141/2164，13:21-13:42 约 20 分钟内）——每次调度器重启都重新触发同一 iter 的 pipeline
- iter_12 有 4 次 dispatched（seq 1093/1130/1142/1152，11:40-11:50 三次级联重启）
- **产物互覆**：`codes/adjust/iter_14/` 有 9 个 `.superseded-*-race.*` 文件（agent_adjust/contradiction/facts/judgment 各有多个竞态副本）；`decisions/` 有 `iter14.superseded-1433-race.md` 和 `iter14.superseded-1444-race.md`
- **humans 空壳行**：iter_12 的第二次 pipeline 实例（12:44:37）写入了 5 行空壳提案（humans #11-#15：`{"title": "", "kind": "restart", "finding": "", ...}`）——并发实例的 step7 agent 在 JSON 输出被截断/竞争时产生了空字段提案，框架照常入队
- **adjusts 表只记 5 行**（iter_4/7/12/14/18）但 adjust_dispatched 有 21 次——**16 次 dispatch 的 pipeline 没有走到 `_collect_adjust` 的 `db.record_adjust()`**（被重启吞噬）

**修复方案**：①调度器启动恢复时用 env/scheduler.pid 或 DB 里的 adjust 状态标记检测在飞 pipeline → 杀掉/等待收尾；②`_maybe_start_adjust` 加持久化单飞锁（DB 行锁或文件锁）；③push_proposal 拒绝全空 title/finding/proposal 的提案。

---

### [A-03] P0 · C(输出消费) — step7 budget_adjust 全史零落地：EVENT_TYPES 白名单缺失导致事件被 schema 校验静默拒绝

**证据**：
- `budget_adjusted` 事件全史 **0 次**（当前 super.db 直查确认）
- step7 agent 在 iter_18 的 session 输出了完整 `budget_adjust` 五键（`agent_timeout_s 1800 / adjust_step_timeout_s 1500 / adjust_total_timeout_s 7200 / max_tokens 6M / max_parallel_experiments 1`）+ reason
- **根因已确认**（本 review 前的框架修复）：`budget_adjusted` / `budget_adjust_rejected` / `env_fixed` / `env_proposal_rejected` 四事件类型不在 `EVENT_TYPES` 白名单 → EventWriter `validate_event` 抛 ValueError → `_run_adjust_pipeline` 内 try/except 静默吞掉 → 事件不落 DB
- **修复已 commit**（events.py EVENT_TYPES 扩容 + loop.py 合成 data 搬运），但**当前运行中的调度器（pid 33565, 16:58 启动）使用的 events.py 修改时间是 16:49——先于启动**，说明修复代码已被加载
- **遗留风险**：iter_22 的 pipeline 正在跑（step4 已 started 17:27）——它的 step7 将是修复后首次行使 budget_adjust 热调权。如果 step7 输出了 budget_adjust 但事件仍为 0 → 修复不完整

**修复方案**：已修（EVENT_TYPES + data 搬运），观察 iter_22 的 step7 输出验证。

---

## P1 — 高（功能异常/数据不一致）

### [A-04] P1 · D(调度框架) — adjust_dispatched 的 iteration 在并发下跳跃混乱：iter_8/11/24 出现在事件流但无对应 pipeline 完成

**证据**：
- adjust_dispatched 事件按序：`4, 8, 7, 11, 12×4, 14×9, 18, 24, 18, 22`——iteration 顺序完全错乱
- iter_8 dispatched（seq 682, window_since=4）后 iter_7 dispatched（seq 711, window_since=7）——**iter_8 先于 iter_7 触发**，说明 cursor 跳跃时 `_last_adjust_iter` 被乱序更新
- iter_11 dispatched（seq 972, window_since=4）但没有对应 pipeline 完成或 adjusts 表行——被 iter_12 的 dispatch 覆盖
- iter_24 dispatched（seq 4205, window_since=10）但 iter_18 dispatched（seq 4326, window_since=4）在后——`_last_adjust_iter` 被 iter_24 更新为 24，然后 iter_18 又把它重置为 18

**根因**：`_last_adjust_iter = iteration_id` 在 dispatch 时同步更新（loop.py:2458），但 dispatch 与 collect 是异步的（collect 在下一 tick 才跑）——多次 dispatch 竞争时 `_last_adjust_iter` 被乱序写。加上调度器重启（`_last_adjust_iter` 从 DB 恢复但 cursor 可能已推进很远）→ 区间起点错乱。

**修复方案**：`_last_adjust_iter` 只在 `_collect_adjust` 成功消费后才推进（不在 dispatch 时推进）；或用 DB 持久化。

---

### [A-05] P1 · A(信息流) — step2 视角生成降级时用 3 个占位视角继续，step3 用占位跑 3 个 subagent——浪费 3 次 LLM 调用产出垃圾矛盾

**证据**：
- loop.py:2677-2683：step2 失败时降级 `perspectives = ["（视角降级默认——step2 失败，基于事实直判）视角 A", ...]`
- step3 对这 3 个占位字符串各跑 1 个 subagent（每个 ~60-120s LLM 调用）——**用无意义的占位视角生成矛盾**是纯浪费
- step4 也会拿到这些垃圾矛盾做判定——**判定质量被降级输入污染**

**修复方案**：step2 降级时跳过 step3/4（直接用 facts.md 让 step5 做事实驱动的辩证逆转）；或 step3 检测占位视角时跳过。

---

### [A-06] P1 · E(human通路) — humans 提案队列 21 行全部 applied=0——提案从未被消费/裁决，已沦为不可消费的死库存

**证据**：
- humans 表 21 行提案（id 1-21），`applied` 全部为 0——**没有任何提案被裁决过**
- 没有 `mark_proposal_applied` 的调用方（grep 代码库：只有 `pending_proposals()` 和 `mark_proposal_applied()` 两个方法——后者无人调用）
- 中控会话（super-loop）的提案消费流程：依赖 human 在中控会话里手动查 humans 表再决策——但没有自动通知机制（无 pending_proposal 事件/心跳里的提案计数）
- adjust step7 的 reasoning 已经指出这一点：「重复入队 = 增加不可消费库存（iter_18 判定「再写 = 增加不可消费库存」纪律）」

**修复方案**：①`rick super status` 输出 pending_proposals 计数（让 human 看到待裁决数）；②中控会话启动时注入未裁决提案列表；③humans 表加 `notified_at` 字段标记是否已被中控会话消费过。

---

## P2 — 中（边缘场景/可观测性缺失）

### [A-07] P2 · A(信息流) — step7 的 ctx 缺 `cells_status`——step7 无法看到格权重/visit 状态

**证据**：
- loop.py:2800-2807：step7 的 ctx7 只含 `task/iteration_id/experiment_records/recent_events/domain_knowledge_path/hindsight_evidence/adjust_dir`
- 对比 step6 的 ctx6 含 `cells_status/cell_descriptions/feature_axes` 等
- step7 的预算调节（`max_parallel_experiments`）需要知道当前格权重分布/已访格数——但 ctx 里没有

**修复方案**：ctx7 加 `cells_status`（与 step6 同源）。

---

### [A-08] P2 · B(行为轨迹) — `_hindsight_evidence` 用全量 `self.db.read_events()` 逐条扫描——事件数增长后性能线性退化

**证据**：
- loop.py:2570：`evs = self.db.read_events()` 读**全部事件**（当前 2718 条非心跳）
- 每次调用 `_hindsight_evidence`（step1 和 step7 各调一次 = 每次 adjust 调 2 次）都全量读
- 7 天长跑场景（预估 10 万+ 事件）→ 每次 adjust 花数秒读全量事件

**修复方案**：改为 SQL 查询（只取 `adjust_decision` 之后的事件 + 只取涉及 proposals target 的 cell 事件）。

---

### [A-09] P2 · D(调度框架) — `_apply_budget_adjust` 的 `max_parallel_experiments` 热调不生效——主循环不重读该值

**证据**：
- `_apply_budget_adjust`（loop.py:2940-2944）将 `self.conf["max_parallel_experiments"] = new_value`
- 但 `_loop_parallel` 在函数入口时读取：`max_parallel_experiments` 被 `__init__` 固化为 `self.max_parallel_experiments`（loop.py:385-386）
- **`self.conf[k] = v2` 只改 dict，不改 `self.max_parallel_experiments` 属性**——主循环继续用旧值
- 对比 `adjust_interval` 在主循环每 tick 从 conf 读取（loop.py:629）——这个是热调生效的

**修复方案**：`_apply_budget_adjust` 里对 `max_parallel_experiments` 特殊处理（同步更新 `self.max_parallel_experiments` 属性）。

---

### [A-10] P2 · F(写者竞态) — decisions/iter{N}.md 的 `_ensure_decision_file` 用「不存在则写」语义——并发 pipeline 实例竞态时首写胜出，后写丢弃

**证据**：
- loop.py:3864+：`if os.path.isfile(path): return path`——先到的实例写入，后到的跳过
- iter_14 有 3 个 decisions 文件（iter14.md + 2 个 superseded）——说明 step7 agent 手动做了 superseded 重命名保全，但框架层没有这个逻辑
- **幸存的 iter14.md 可能不是最终完整的版本**（取决于哪个实例先写）

**修复方案**：decisions 文件写入用时间戳后缀 + 最新链接（或 DB 记录权威版本号）；`_ensure_decision_file` 改为追加模式（同 iter 多次决策拼接而非首写固定）。

---

### [A-11] P2 · A(信息流) — `experiment_records` 的区间范围（`_last_adjust_iter` → `iteration_cursor`）在多次 dispatch 竞态下范围错乱

**证据**：
- iter_14 被 dispatch 9 次，每次 `_last_adjust_iter` 被更新为 14 → 下一次 dispatch 的 experiment_records 范围= 14→cursor
- 但中间有多次 dispatch 被重启吞噬（pipeline 未完成）→ 实际已完成的 adjust 只有 iter_4/7/12/14/18 五次
- **中间的 dispatch（iter_8/11/12×3/24）消耗了 `_last_adjust_iter` 的推进但没产生决策**——它们的区间内的实验记录在下次 adjust 时被跳过

**修复方案**：同 A-04——`_last_adjust_iter` 只在 collect 成功后推进。

---

## Bug 清单总表

| # | 严重度 | 维度 | 一句话摘要 | 修复优先级 |
|---|---|---|---|---|
| A-01 | **P0** | E(human通路) | Go 写 interventions/ 目录 vs py 读 humans 表——human 干预通路完全断裂（stop 指令断路） | 🔴 立即修 |
| A-02 | **P0** | F(写者竞态) | 调度器重启后 `_adjust_thread=None` → 同 iter pipeline 多次并发（iter_14 有 9 次 dispatch）→ 产物互覆 + humans 空壳行 | 🔴 立即修 |
| A-03 | **P0** | C(输出消费) | budget_adjusted 事件被 EVENT_TYPES schema 校验静默拒绝（已修复，待 iter_22 验证） | ✅ 已修 |
| A-04 | P1 | D(调度框架) | `_last_adjust_iter` 在 dispatch 时同步更新——多次 dispatch 竞争时区间起点乱序 | 🟡 高 |
| A-05 | P1 | A(信息流) | step2 降级时 3 个占位视角继续跑 step3×3 subagent——浪费 LLM 调用产出垃圾矛盾 | 🟡 高 |
| A-06 | P1 | E(human通路) | humans 提案队列 21 行全 applied=0——无消费机制，已沦为死库存 | 🟡 高 |
| A-07 | P2 | A(信息流) | step7 ctx 缺 cells_status——预算调节缺格权重上下文 | 🟢 中 |
| A-08 | P2 | B(行为轨迹) | hindsight_evidence 全量读 events——10万+事件时线性退化 | 🟢 中 |
| A-09 | P2 | D(调度框架) | max_parallel_experiments 热调不生效——self 属性不随 conf dict 更新 | 🟢 中 |
| A-10 | P2 | F(写者竞态) | decisions/iter{N}.md 首写胜出——并发实例时后写丢弃 | 🟢 中 |
| A-11 | P2 | A(信息流) | experiment_records 区间在 dispatch 竞态下范围错乱（中间 dispatch 消耗区间但无产出） | 🟢 中（同 A-04 修复） |
