# env-layer-bugs — super 进化系统 env 基础设施层运行时 review

> 2026-11-22 · review-agent 派遣（job_56）· 维度 A-G 全扫
> 方法：预期架构 read（db.py/config.py/events.py/env.json/gates/）→ 运行时 DB/sqlite3 直查 → 代码审计

---

## Bug 清单

### BUG-001 [P0] [G] Go 侧 SuperEvent 写 interventions/ 目录，py 侧只读 humans 表——干预通路断裂

**证据**：
- Go `internal/handler/super.go:953-983`：`SuperEventIn` 写 `interventions/<ts>-<rand>.json` 文件
- py `super/superopt/intervention.py:40-80`：`InterventionQueue.drain()` 只调 `self._db.drain_interventions()`（读 humans 表）
- DB 实查：`humans` 表 `queue_type='intervention'` 行 = **0**（Go 从未写入 humans 表）

**影响**：human 通过 `rick super event` 投递的干预**永远不会被 py 调度器摄取**——stop 指令失效、价值判断注入断裂。step1 的 `pending_interventions` 恒为空。

**修复**：Go `SuperEventIn` 改为写 `super.db` humans 表（`INSERT INTO humans (queue_type='intervention', content) ...`），或 py `drain` 兼容读 interventions/ 目录作为 fallback。

---

### BUG-002 [P0] [B] events 表的 experiment_id 列与 experiments 表的 experiment_id 列是**两种不同 ID 体系**，不可 JOIN

**证据**：
- events 表 experiment_id 格式：`problem_40_Matmul_Scaling_ResidualAdd#iter1`（cell_id#iterN）
- experiments 表 experiment_id 格式：`1-202611220044`（workdir 时间戳 ID）
- DB 实查：`SELECT count(*) FROM experiments e WHERE NOT EXISTS (SELECT 1 FROM events ev WHERE ev.experiment_id = e.experiment_id)` = **60**（全部实验无 events 关联）

**影响**：无法从 events 表直接 JOIN experiments 表做归因分析（如「实验 X 的所有事件」需通过 cell_id#iterN 间接查找）。DB 设计文档声称 "cell_id#iterN 是 events 的实验分组语义" 但 experiments 表用的是另一种 ID——两种 ID 并存但没有映射关系存储。

**修复**：events 表加 `workdir_experiment_id` 列（或 experiments 表加 `trace_id` 列），建立映射。短期：events 表的 experiment_id 已含 `cell_id` 信息可间接关联，但需要解析。

---

### BUG-003 [P1] [A] env.json 的 `env_slots=1` 与 super.json 的 `env_slots_per_env=8` 矛盾——取值链覆盖但文档未对齐

**证据**：
- env.json：`"env_slots": 1`
- super.json config：`"env_slots_per_env": 8`
- resources 表实际有 8 个 env_slot

**影响**：不影响运行（conf 覆盖了 env.json 值），但 domain_knowledge.md 写的是 "env_slots=1——benchmark 计时对共享资源敏感"——agent 看到的文档与实际行为不一致，可能误导。

**修复**：domain_knowledge.md 同步更新（或 env.json 的 env_slots 改为实际值）。

---

### BUG-004 [P1] [C] remote_exec.py 的 `push_candidate` 和 `remote_gate_bound` 各自独立选机——不同调用可能选到不同机器

**证据**：
- `push_candidate` (line 80)：调 `run_remote` → 内部调 `_next_h20()` 选机
- `remote_gate_bound` (line 89)：自己调 `_next_h20()` 选机

**影响**：如果外部代码先 `push_candidate`（推到 A 机）然后调 `remote_gate_problem`（在 B 机跑）——候选文件不在 B 机。`remote_gate_bound` 已修复（push+run 同机），但 `push_candidate`+`remote_gate_problem` 组合仍可能跨机。

**修复**：`push_candidate` 返回 `(remote_path, machine)` 元组，`remote_gate_problem` 接受 machine 参数强制绑定。

---

### BUG-005 [P1] [C] SSH 密码硬编码在 remote_exec.py 源码中

**证据**：`remote_exec.py:23`：`H20_PASS = "MiMi@@19950810"`

**影响**：密码暴露在 env 目录下（agent 可读——虽然 gates/ 禁读但 remote_exec.py 在 gates/ 目录内，有 hook 拦截；但 skills/ 薄入口 import 它，路径可能被 agent 看到）。

**修复**：从环境变量 `H20_PASS` 或 `~/.rick/pi/agent/secrets` 读取。

---

### BUG-006 [P1] [B] experiments 表和 cells 表无索引（除 PK 外）——大数据量查询性能

**证据**：
- `sqlite3 $WS/super.db ".indexes"` 只返回 `idx_events_experiment`（events 表）
- experiments 表有 `iteration`、`cell_id`、`status` 列但无索引
- cells 表的 `weight`、`best_experiment_id` 无索引

**影响**：当前 60 实验量级影响可忽略，但 100+ 迭代后 `SELECT ... WHERE cell_id=?` 和 `WHERE status=?` 会全表扫描。

**修复**：`CREATE INDEX idx_experiments_cell ON experiments(cell_id)` + `idx_experiments_status ON experiments(status)` + `idx_cells_weight ON cells(weight)`。

---

### BUG-007 [P1] [D] Go 侧 `goSuperDefaults()` 的 `max_parallel_experiments: 1` 与 py `DEFAULTS["max_parallel_experiments"]: 1` 一致，但 Go daemon 启动时用 `initSuperWorkspace` 重建 super.json config——**运行中改值会被 daemon 覆盖**

**证据**：
- Go `internal/handler/super.go:207`：`"max_parallel_experiments": 1`
- Go `SuperStart` → `initSuperWorkspace` → `writeSuperJSONAtomic`——每 daemon 启动写 super.json
- 实测：手动改 super.json 的 `max_parallel_experiments` 从 2→4，daemon 重启后被覆盖回 2（需 stop→改→start）

**影响**：运行中 step7 的 `budget_adjust` 热调 `max_parallel_experiments` 只改 py 内存 conf，不改 super.json——下次 daemon 重启时 Go 侧从 super.json 读旧值，**热调丢失**。

**修复**：py 侧热调 `max_parallel_experiments` 时同步写回 super.json（或 Go daemon 读 conf 前先从 py 侧心跳获取实际运行值）。

---

### BUG-008 [P2] [A] super.json config 的 `agent_timeout_s=1200` 与 step7 荐值 1800 不一致——budget_adjust 已修但值未更新

**证据**：
- super.json：`"agent_timeout_s": 1200`
- iter_18 step7 输出：`"agent_timeout_s": 1800`（budget_adjust 首次生效将改变）
- iter_14/iter_12 step7 均荐 1800（三轮一致推荐）

**影响**：修复 budget_adjust bug 前 step7 无法热调（bug 已修）；修复后首轮 adjust 才会生效。当前 1200s 导致 coding agent 超时被实杀 ≥6 例。

**修复**：已修 budget_adjust 事件类型；下一轮 adjust 触发时 step7 将自动应用 1800。

---

### BUG-009 [P2] [B] adjusts 表的 `iter_8` 记录存在但 `decisions/` 目录无 iter8.md（decisions 文件缺失）

**证据**：
- adjusts 表有：iter_4, iter_7, iter_12, iter_14, iter_18
- decisions/ 目录有：iter4.md, iter7.md, iter11.md, iter12.md, iter14.md, iter18.md（+ 3 个 superseded）
- **iter11.md 存在但 adjusts 表无 iter_11**（可能因中断/竞态未写入 adjusts）

**影响**：adjusts 表与 decisions 文件不对称（decisions 有 iter11 但 adjusts 表没有——iter11 可能是被中断的 adjust 运行，产物落盘了但 adjusts 行未写）。

**修复**：`record_adjust` 应在流水线开始时（而非结束时）写入（status=started），结束时更新。或增加 `adjust_dir` 扫描兜底。

---

### BUG-010 [P2] [E] 四台 GPU 机的 /tmp/super-kb/ cand/ 目录堆积（704+ 文件）——无清理机制

**证据**：
- 33.229.1.97：`/tmp/super-kb/cand/` 有 704 个文件
- 33.229.1.85：1 个，33.229.29.190：18 个，33.229.60.179：0 个

**影响**：长时间运行后磁盘可能占满（每候选 ~1-5KB，704 个 ≈ 1-3MB——当前可忽略，但 100+ 迭代 × 8 并发 = 800+ 候选/GPU）。

**修复**：remote_gate_bound 执行完后 `rm` 推送的候选文件（一次 SSH 链内完成）。

---

### BUG-011 [P2] [F] benchmark 计时窗口的 flock 锁在**远程** benchmark_impl.py 内实现——但本地的 benchmark 薄壳直接透传远程结果，锁在远程生效

**证据**：
- 远程 benchmark_impl.py：`BENCH_LOCK = "/tmp/super-kb/.bench_lock"` + `fcntl.flock`
- 本地 gates/benchmark_impl.py（薄壳）：透传远程结果，无本地锁

**影响**：锁语义正确（远程锁保证同机计时互斥），但**跨机不互斥**——A 机和 B 机可以同时计时（各自有独立的 .bench_lock 文件）。这是设计意图（每机独占计时），但文档未明确说明。

**修复**：无需修复（设计意图正确），但 domain_knowledge.md 应说明「flock 是 per-machine 锁，跨机不互斥」。

---

### BUG-012 [P2] [A] config.py DEFAULTS 的 `adjust_step_timeout_s=1800` 和 `adjust_total_timeout_s=10800` 存在于 DEFAULTS 但 super.json 实际值是 900 和 5400

**证据**：
- py DEFAULTS：`"adjust_step_timeout_s": 1800, "adjust_total_timeout_s": 10800`
- super.json config：无此二键（`config.load` 补 DEFAULTS 后实际 1800/10800）
- 但 step7 iter_18 荐值 1500/7200（介于旧 900/5400 和新 DEFAULTS 1800/10800 之间）

**影响**：无直接影响（DEFAULTS 补全语义正确），但 step7 的 budget_adjust 可能在 adjust 运行中调节 adjust_step_timeout_s——导致同一流水线内不同步骤用不同超时。

**修复**：在流水线启动时快照 timeout（而非每步实时读 conf），确保同一轮 adjust 内超时一致。

---

## Bug 清单总表

| # | 严重度 | 维度 | 一句话摘要 | 修复优先级 |
|---|---|---|---|---|
| BUG-001 | **P0** | G | Go SuperEvent 写 interventions/ 目录但 py 只读 humans 表——干预通路完全断裂 | **立即修** |
| BUG-002 | **P0** | B | events.experiment_id 与 experiments.experiment_id 是两种 ID 体系——不可 JOIN | **立即修** |
| BUG-003 | P1 | A | env.json env_slots=1 vs super.json env_slots_per_env=8——文档与实际不一致 | 高 |
| BUG-004 | P1 | C | push_candidate 和 remote_gate_bound 各自选机——跨机文件不可见风险 | 高 |
| BUG-005 | P1 | C | SSH 密码硬编码在 remote_exec.py 源码 | 高 |
| BUG-006 | P1 | B | experiments/cells 表无二级索引——大数据量查询性能 | 高 |
| BUG-007 | P1 | D | Go daemon 重启覆盖 py 热调的 max_parallel_experiments——热调丢失 | 高 |
| BUG-008 | P2 | A | agent_timeout_s=1200 vs step7 荐 1800——budget_adjust 已修待生效 | 中 |
| BUG-009 | P2 | B | adjusts 表 iter_11 缺失（decisions 有但 adjusts 无）——流水线竞态 | 中 |
| BUG-010 | P2 | E | 四台 GPU cand/ 目录堆积无清理（704 文件/机） | 低 |
| BUG-011 | P2 | F | flock 是 per-machine 锁——跨机不互斥（设计意图但未文档化） | 低 |
| BUG-012 | P2 | A | 流水线内 timeout 应快照而非实时读（防 step7 调节影响当前轮） | 低 |

---

## 未发现 Bug 的维度（通过验证）

| 维度 | 验证项 | 结果 |
|---|---|---|
| E. 题库部署 | 四台 GPU 的 verify_impl.py/benchmark_impl.py md5 一致 | ✅ 全一致 |
| E. 题库部署 | 四台 GPU 的 problems/level2/ 文件数一致 | ✅ 106 个文件（含 pycache） |
| F. 门禁 | preverify_red（exploits 全拒+金样例零误杀） | ✅ 通过（运行中验证） |
| F. 门禁 | 四分布实现（normal×3/uniform/heavytail/outlier/boundary） | ✅ 远程代码正确 |
| F. 门禁 | TF32 baseline 全局设置 | ✅ benchmark_impl.py 正确设置 |
| F. 门禁 | 权重同步（load_state_dict strict=False） | ✅ verify/benchmark 均有 |
| B. WAL 模式 | PRAGMA journal_mode=WAL | ✅ 已设置 |
| B. 外键 | cells.best_experiment_id → experiments.experiment_id | ✅ 全部有对应行 |
| B. 幽灵实验 | codes/ 目录数 vs experiments 表行数 | ✅ 一致（61 vs 61） |
| G. 提案队列 | humans 表 proposal 队列（21 条待裁决） | ✅ 正常运作 |
| G. 提案权限 | env_proposal_rejected（非 restart/verifier 类拒绝） | ✅ 事件类型已加入 |
