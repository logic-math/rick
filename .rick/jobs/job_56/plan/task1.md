# task1 — M1 DB 地基：sqlite 全量 DB 化 + 进程锁 + 槽位资源调度

> job_56 · DAG L1（与 task2 并行，写域互斥）· 无依赖
> 源设计：grilling/design-tree.md KR1+KR2（含 Q1/Q2/Q7 裁决）

# 任务目标

将 super mod 的进化元信息存储从 json/jsonl 全量迁移到 sqlite 单库（`<ws>/super.db`），落地五张核心表 + py 调度器进程锁 + 槽位资源调度（实验槽/环境槽二级限额），迁移 py 侧全部消费方与 Go 侧读方（supervisor/web/status），一次切净、旧工作区保留不迁移。

# 关键结果

1. **KR-A【db.py 五表】**：`super/superopt/db.py` 新建——sqlite3 封装（stdlib-only，连接管理+线程安全），五张表：
   - `cells`（特征格表）：`cell_id`（PK）/ `name`（名称）/ `method_desc`（优化方法描述，TEXT 详细）/ `best_experiment_id`（当前最优实验 ID）/ `weight`（REAL，权重值）
   - `experiments`（实验表）：`experiment_id`（PK，格式 `<iteration>-<YYYYMMDDHHMM>` 迭代号-年月日时分时间戳）/ `iteration`（INTEGER）/ `workdir`（工作目录名=experiment_id）/ `record_path`（实验记录 md 路径）/ `session_id`（pi agent sessionID，行为轨迹还原）/ `cell_id`（FK）/ `score`（REAL，可空）/ `created_at`
   - `humans`（人类价值判断表，两类队列）：`id`（PK 自增）/ `queue_type`（TEXT：`intervention`=干预队列[human 提交] / `proposal`=提案队列[runtime 产出待裁决]）/ `content`（内容描述）/ `applied`（INTEGER 0/1，是否已被应用）/ `created_at` / `applied_at`
   - `events`（系统级事件日志表）：`seq`（PK 自增）/ `ts` / `event_type` / `source` / `experiment_id`（可空索引）/ `payload`（JSON TEXT）——查询此表即感知系统状态变化
   - `resources`（资源表）：`resource_type`（TEXT：`experiment_slot`=实验槽 / `env_slot`=环境槽，联合 PK 一部分）/ `resource_id`（TEXT，如 `exp-slot-0` / `env-slot-default-0`）/ `description` / `quota`（INTEGER 额度数）——一条记录代表一个资源位
2. **KR-B【进程锁】**：py 调度器启动时在 env 下写 PID 锁文件（`<ws>/env/scheduler.pid`）：已存在且 `os.kill(pid, 0)` 存活 → 启动失败退出（exit 非 0+明确报错）；正常退出/被杀时不清也行（下次启动校验进程死活即可，不依赖清理）。一个 super 会话对一个 env 唯一调度器。
3. **KR-C【槽位资源调度】**：`db.py` 提供资源分配 API（`acquire_resource(resource_type, experiment_id)` / `release_resource(...)`，事务性：查空闲位→占用→提交）；调度主循环改为：总实验槽满→等在飞实验完成释放；该实验目标 env 的环境槽满→等环境槽释放。`resources` 表按 config 初始化（`max_parallel_experiments` 个实验槽 + `env_slots_per_env`（默认 1）个环境槽/env）。**调度器仅管资源分配**——两实验同 env 并发时分时复用是 ENV 设计责任（gen_env 规则，task2 落文档）。
4. **KR-D【py 侧消费方迁移】**：`events.py`（EventsLog→events 表写+游标查询）/ `archive.py`（格状态+精英+权重→cells 表+实验目录引用；内存快照接口保持兼容）/ `ledger.py`（jsonl→experiments+events 查询）/ `intervention.py`（InterventionQueue→humans 表：intervention 类 push/drain，drain 后标 applied=1 不删除）/ `adjudicator.py`（读 events/archive→改 DB 查询）/ `sampler.py`（权重源→cells 表）/ `feature_space.py`（格定义加载，状态从 DB）/ `budget.py`、`lineage.py`（消费方切换）/ `config.py`（db 路径+槽位参数）/ `__main__.py`（CLI 适配）。**loop.py 仅改存储调用行**（构造点/恢复点/ctx 组装的读调用切 DB API），**不得改动 adjust 流水线与实验链的流程逻辑**（task3/task4 的职责）。
5. **KR-E【实验 ID 编号】**：迭代号-年月日时分（`<iter>-<YYYYMMDDHHMM>`），同时是工作目录名（`codes/<experiment_id>/`）与实验表 PK；迭代号继续用 `iteration_cursor`。
6. **KR-F【Go 侧迁移】**：`internal/handler/super.go`（events 聚合/status/北极星→SQL 查询 super.db；heartbeat 机制保留——Go 写 heartbeat.json 或迁 DB 均可，选侵入最小）/ `internal/web/sessions.go`（同）/ supervisor 进程回收配合进程锁（启动 py 前先杀旧进程的既有杀梯保留）。Go sqlite 驱动用 `modernc.org/sqlite`（纯 Go 无 cgo）；`go.mod`/`go.sum` 更新。
7. **KR-G【tools 迁移】**：`super/tools/adjust_trace.py`、`cost_report.py` 改 SQL 查询（CLI 参数不变）。
8. **KR-H【测试迁移】**：受影响的现有测试（test_events/test_archive/test_ledger/test_intervention/test_sampler/test_feature_space/test_adjudicator/test_config/test_loop/e2e_test/super_session_test 等）全量迁移到 DB 断言；新增 `super/tests/test_db.py`（五表 CRUD/进程锁/资源分配阻塞语义/事务性）。

# 写域

（文件级清单，与 task2 互斥）

- super/superopt/db.py（新建）
- super/superopt/events.py
- super/superopt/archive.py
- super/superopt/ledger.py
- super/superopt/intervention.py
- super/superopt/adjudicator.py
- super/superopt/sampler.py
- super/superopt/feature_space.py
- super/superopt/budget.py
- super/superopt/lineage.py
- super/superopt/config.py
- super/superopt/__main__.py
- super/superopt/loop.py（仅存储调用行，禁改 adjust/实验链流程逻辑）
- super/superopt/cell_lock.py（仅当需适配；默认不动）
- super/tests/test_db.py（新建）
- super/tests/test_events.py
- super/tests/test_archive.py
- super/tests/test_ledger.py
- super/tests/test_intervention.py
- super/tests/test_sampler.py
- super/tests/test_feature_space.py
- super/tests/test_adjudicator.py
- super/tests/test_config.py
- super/tests/test_loop.py
- super/tests/test_budget.py
- super/tests/test_lineage.py
- super/tests/e2e_test.py
- super/tests/super_session_test.py
- super/tools/adjust_trace.py
- super/tools/cost_report.py
- internal/handler/super.go
- internal/handler/super_test.go
- internal/web/sessions.go
- internal/web/sessions_test.go
- internal/runtime/supervisor.go（仅进程锁配合点）
- internal/runtime/supervisor_test.go（仅进程锁配合点）
- go.mod
- go.sum

**loop.py 边界**：只改「存储访问调用行」（EventsLog/Archive/Ledger/InterventionQueue/snapshot 的构造与读取），**禁改** `_maybe_start_adjust/_collect_adjust/_join_adjust`（adjust 逻辑）、`_step_locked/_slot_worker/_dispatch_agent/_apply_proposal`（实验链逻辑）的流程语义（存储行切换允许，逻辑重构禁止）。

# 实现要点

1. **单写者不变**：py 调度器是 super.db 唯一写者（Go 侧只读——读连接 `?mode=ro` 或普通连接只执行 SELECT）；events 表 seq 由 py 单调递增，Go/tools 全走 SELECT。
2. **sqlite 并发**：调度器内多线程（槽 worker+adjust 线程）共享一个连接 `check_same_thread=False`+一把互斥锁（写少读多，锁粒度足够）；WAL 模式（`PRAGMA journal_mode=WAL`）提高 Go 侧读并发。
3. **archive 兼容**：`_cells_snapshot` 等内存接口签名不变（调用方零改动），实现改读 DB；精英代码/diff 仍在实验目录（codes/），cells 表存引用不存正文。
4. **humans 表语义**：drain（消费）= `SELECT ... WHERE queue_type='intervention' AND applied=0` + 更新 `applied=1`；提案队列入队 API 给 task3 的 step7 用（`push_proposal`）。**本 task 只落表+API，消费方接入由 task3 做**。
5. **资源 API 事务性**：acquire 用 `BEGIN IMMEDIATE` 防两线程同抢一位；占用记录一张 `resource_allocations`（resource_id/experiment_id/acquired_at）内部表（或 resources 表加 `holder_experiment_id` 列——选一实现，语义等价）。
6. **进程锁**：`superopt/__main__.py` 启动最前部检查（先于任何 DB 写）；报错文案含存活 PID 与建议（「先 rick super stop 回收旧进程」）。
7. **旧工作区**：不迁移、不兼容读取；`super.json` 自检发现旧布局（存在 events.jsonl 且无 super.db）→ 明确报错指引新建 env。
8. **Go 侧 modernc.org/sqlite**：`go get modernc.org/sqlite`；查询集中在 internal/handler/super.go 的 events 聚合函数与 internal/web/sessions.go 的状态组装；**保持对外 JSON 输出字段不变**（web API 兼容）。
9. **stdlib-only 纪律**：py 侧只用标准库 sqlite3（super 仓现状约束）。

# 测试方法（自测，过程性）

- RED：先写 `super/tests/test_db.py` 断言（五表 DDL/CRUD/进程锁存活检测/资源分配-阻塞-释放/事务抢位），先跑红
- GREEN：实现 db.py 至全绿；再逐模块迁移（每迁一个消费方跑对应测试文件）
- 回归：`python3 -m unittest discover -s super/tests -p 'test_*.py' -v` 全绿（e2e 用 mock_pi 路径可跑）
- Go 侧：`go build ./... && go test ./internal/... ` 全绿
- 自测代码写在 super/tests/（产品测试，随交付）——不额外落盘一次性脚本

# 验收（gate1.py 承担）

`python3 .rick/jobs/job_56/plan/gates/gate1.py`：test_db+test_events+test_archive+test_intervention+test_gates+test_sampler+test_feature_space+test_adjudicator 全绿 + go build+go test（internal）+ 结构断言（五表 DDL 关键字段/进程锁/资源表类型/旧 json 路径废弃）。
