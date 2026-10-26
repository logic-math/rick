# task45：格级互斥 + 并发实验（一个格同时只有一个实验）

# 任务目标
human fix 2：builder/coding/debug 三 agent 必须操作**同一特征格**（一个实验=一次格上的迭代）；**一个格上同时只有一个实验**；**不同格可并行跑实验**。

# 关键结果
- **格级锁**（`super/superopt/loop.py` 或新模块 `cell_lock.py`）：
  - 内存 + 文件双保险：`<ws>/locks/cell_<safe_id>.lock`（O_CREAT|O_EXCL，含 pid/ts；陈旧锁按 mtime > `cell_lock_ttl_s`（默认 3600）清理并记事件 `cell_lock_stale_recovered`）
  - `acquire(cell_id) -> bool` / `release(cell_id)`；采样时跳过已锁格（`sampler` 采样后若锁失败 → 重采样，最多 N 次；全锁则等待/跳过本迭代）
- **并发实验**（`max_parallel_experiments`，默认 1=完全保持现有串行行为）：
  - >1 时主循环以「迭代槽」并行：每个槽 = 独立 iter_dir + 独立 cell 锁 + 独立 worker 线程跑 builder→coding→debug→gate；**主线程独占事件写入与 archive 写入**（worker 通过队列提交结果，主线程落事件——保持单写者纪律）
  - 迭代槽状态机：`pending → running → verify → done/failed`；`iteration_started/ended` 事件由主线程在槽开始/结束时写
  - 预算（wall/token）在槽结束汇总时更新
- **互斥保证**：同一 cell 的两个槽不可能同时 running（锁 + 主线程采样派发时检查）
- config：DEFAULTS 增 `max_parallel_experiments: 1`、`cell_lock_ttl_s: 3600`
- 测试 `test_loop`：①并发 2 槽跑 2 个不同格，事件无交错破坏（seq 连续、单写者）②同一格不被两槽并发（锁拒绝/重采样）③陈旧锁恢复 ④`suspicious_gain`/门禁失败等既有路径在并发下仍工作 ⑤默认 1 时行为与旧完全一致（回归）

# 写域
- super/superopt/loop.py
- super/superopt/cell_lock.py（新建，若选择拆模块）
- super/superopt/config.py
- super/tests/test_loop.py
- super/tests/test_cell_lock.py（新建，若拆模块）

# 依赖关系
- task44（archive 线程安全）、task46（loop 写域串行）

# 测试方法
`python3 -m unittest super.tests.test_loop super.tests.test_cell_lock -v` 全绿 + 全量无回归（默认并发 1 时行为不变）

# 禁止事项
- 不得让 worker 线程直接写 events.jsonl/archive（单写者）；并发默认值必须为 1（保守启用）
