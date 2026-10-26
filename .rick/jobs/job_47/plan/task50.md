# task50：并发槽池（异格并行实验）

# 任务目标
在 task49 的并行事件 + task44 的 archive RLock + task45 的格锁基础上，实现**真正的异格并行实验**（一个格同时只有一个实验；不同格并行）。

# 关键结果
- `super/superopt/loop.py`：
  - `max_parallel_experiments`（默认 **2**）生效：主线程调度 N 个迭代槽
  - 每槽流程：主线程采样（跳过已锁格）→ acquire 格锁 → 起 worker 线程执行 `_step_locked(cell_id)`（builder/coding/debug/gate/admit，内部写事件带 experiment_id）→ 完成后主线程/worker 释放锁
  - **事件并行写**依赖 task49（O_APPEND + experiment_id）；archive 写入已有 RLock
  - 槽池生命周期：`while 未达预算/未达 max_iterations: 补满在飞槽；收割完成的槽`；`stop_requested` → 等待在飞槽收尾（grace）后退出
  - 事件：`experiment_started/experiment_finished`（带 experiment_id, slot）便于按实验 grep；`slot_locked_skip` 保留
  - 预算检查在槽收割后更新（wall/token）
- 测试 `test_loop`：①2 槽并行跑 2 个不同格（时间重叠断言：两实验的 ts 区间有交叠）②同一格不被两槽并发（锁拒绝/重采样）③并行下事件按 experiment_id 还原互不污染（read_experiment 每条流完整）④archive 入档无丢失（并发 2 槽各入一格）⑤stop 时在飞槽优雅收尾 ⑥默认并发 2 但只有 1 格时退化为串行

# 写域
- super/superopt/loop.py
- super/tests/test_loop.py

# 依赖关系
- task49（并行事件）、task51（同 loop.py 串行）

# 测试方法
`python3 -m unittest super.tests.test_loop -v` 全绿 + 全量无回归

# 禁止事项
- 不得绕过格锁；不得让两槽写同一 archive 格；并发默认值保守（2）
