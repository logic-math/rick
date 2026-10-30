# task54：adjust 并发饥饿修复（门槛放宽 + 跨边界触发）

# 任务目标
真缺陷（红黑树 env 实跑实证：21 迭代 0 次 adjust）：并发槽池下 `_loop_parallel` 把 adjust 触发放在 `if not slots:` 内，而槽池每 tick 自动补满 → 该门几乎永不打开 → **adjust 永久饥饿**；且 `cursor % adjust_every == 0` 在并发 cursor 跳跃时会漏判。

# 关键结果
- `super/superopt/loop.py` `_loop_parallel`：
  - **adjust 触发移出 `if not slots`**：每 tick 独立检查（adjust 本就是后台线程：`_maybe_start_adjust` 起线程、`_collect_adjust` 主线程应用提案；`_adjust_lock` + `_adjust_thread is None` 保证单飞；读 cells_status 走 archive RLock 快照、读 events 是只读 → 与 worker 并行安全）
  - **跨边界触发**：改用 `self._last_adjust_iter` 判断 `iteration_cursor - _last_adjust_iter >= adjust_every`（并发下 cursor 可能跳跃，等值判断会漏；首轮 _last_adjust_iter=0）
  - 其余全局态操作（feature_space 热重载 / 裁决消费 / epoch 复评）**保留在 `if not slots`**（它们改全局面貌，与 worker 交叠有风险）——docstring 说明这种职责划分
- `_loop`（串行）同步改为跨边界判断（两路径一致，避免语义分叉）
- **adjust 在飞期间不重复触发**（已有 `_adjust_thread is None` 保护，确认并加测试）
- 事件：`adjust_dispatched{iteration, window_since}`（可观测：何时触发、距上次多少迭代）
- 测试 `test_loop`：
  ①**并发 2 槽 6 迭代 → adjust 至少触发 2 次**（回归断言，直指本 bug）
  ②串行路径 adjust 次数与旧行为一致
  ③adjust 在飞时边界再次到达不重复派发
  ④跨边界：cursor 跳跃（如 3→7）仍触发（>= 判断）
  ⑤adjust 与 worker 并行时 events 可还原（experiment_id 流完整）、archive 无损坏

# 写域
- super/superopt/loop.py
- super/tests/test_loop.py

# 依赖关系
- 无（L1）

# 测试方法
`python3 -m unittest super.tests.test_loop -v` 全绿（含并发 adjust 触发断言）+ 全量无回归

# 禁止事项
- 不把热重载/裁决/epoch 移出 `if not slots`；不改 adjust 的决策逻辑
