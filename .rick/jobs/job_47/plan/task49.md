# task49：事件日志并行化（experiment_id 维度）

# 任务目标
human 裁决的事件序方案：**「格子 + 迭代号」= 一次实验的唯一标记**；**支持并行写入事件日志**；通过 grep 不同格子及迭代号还原某次实验的事件流。

# 关键结果
- `super/superopt/events.py`：
  - 事件行 schema 增 `experiment_id`（`"<cell_safe>#iter<N>"`，其中 cell_safe = cell_id 中非字母数字替换为 `_`）与 `experiment_seq`（**实验内**单调递增序号）
  - `EventWriter.append(type, payload, source, experiment_id=None)`：experiment_id 给定时维护每实验的计数（内存 dict + 启动时扫描 events 恢复各实验计数）；未给定时 `experiment_id=None, experiment_seq=None`（全局运维事件如 heartbeat/session_started）
  - **并行写安全**：`append` 保持单次 `write` 一行（O_APPEND 下 POSIX 单行 <4KB 写入原子），fsync 可配置关闭（`fsync=False` 用于高频实验事件，退出时统一 flush）；seq 字段改为**不保证全局单调**（文档注明：定位用 `(experiment_id, experiment_seq)`，全局时序用 ts）
  - `read_experiment(experiment_id) -> list`：读全部事件按 experiment_seq 排序还原单实验流
  - `validate_event` 适配新字段（experiment_seq 可为 None）
- `super/superopt/lineage.py` `project_state`：
  - 兼容两种事件（有/无 experiment_id）：**已完成迭代**判定 = 存在对应 `iteration_ended`（按 (cell,iter) 或旧 iteration_id）；cursor = max(已完成 iteration_id)
  - **中断实验识别**：有 `iteration_started`/`cell_sampled` 但无 `iteration_ended` 的 experiment_id → 返回 `interrupted_experiments: [...]`（崩溃恢复时可用于清理格锁 + 记事件）
- `super/superopt/loop.py`：所有迭代内事件 append 传 experiment_id（iteration_started/cell_sampled/agent_started/ended/candidate_produced/gate_result/elite_*/iteration_ended）；运维事件（session/heartbeat/intervention/adjust_*/proposal/weight_changed）不带（或带 adjust 专用 id `adjust#iter<N>`）
- 测试：`test_events`（experiment_id/experiment_seq 计数/read_experiment 排序/并行 append 100 线程无行损坏/fsync=False 可读）、`test_lineage`（新旧混排投影/cursor 取已完成 max/interrupted_experiments 识别）、`test_loop`（事件含 experiment_id；grep 还原单实验可行）

# 写域
- super/superopt/events.py
- super/superopt/lineage.py
- super/superopt/loop.py
- super/tests/test_events.py
- super/tests/test_lineage.py
- super/tests/test_loop.py

# 依赖关系
- 无（L1）

# 测试方法
`python3 -m unittest super.tests.test_events super.tests.test_lineage super.tests.test_loop -v` 全绿 + 全量无回归

# 禁止事项
- 不破坏既有事件消费方（gates/adjust_trace/cost_report 读 payload 不受影响）；seq 语义变更必须在文档与 docstring 显式说明（旧消费者若依赖全局 seq 单调需改）
