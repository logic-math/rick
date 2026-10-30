# task44：特征格五要素（+ elite code_path / record_path）+ archive 线程安全

# 任务目标
human fix 1：特征格从四要素升为**五要素**——`{name, method(description), weight, score, code_path}`，其中 `code_path` 标记**当前最佳精英代码**并关联其**实验记录**。
另：为并发实验（task45）准备 archive 线程安全。

# 关键结果
- `super/superopt/archive.py`：
  - 格 schema 增顶层 `elite_code_path`（当前最佳精英的 code_path，admit 时同步）与 `elite_record_path`（该精英对应迭代目录的 experiment_record.md，admit 时由调用方传入）
  - `admit(cell_id, entry)` 接收并写入两字段；`get_record_path(cell_id)` 访问器
  - `cells()` 快照包含五要素字段
  - 旧档兼容：缺字段返回 None/空
  - **线程安全**：`__init__` 增 `threading.RLock`；`admit`/`set_weight`/`set_visit`/`set_description`/`get_elite`/`cells`/`save`/`load` 全部加锁（并发实验时多 worker 写档案）
- `super/superopt/loop.py`：`_verify_and_admit` 调 `admit` 时传 `record_path`（该迭代 experiment_record.md 路径）
- 测试：`test_archive`（五要素往返/elite_code_path 同步/record_path/admit 更新/旧档兼容/并发写 20 线程不炸）；`test_loop`（入格后 archive 格含 elite_code_path + elite_record_path）

# 写域
- super/superopt/archive.py
- super/superopt/loop.py（仅 _verify_and_admit 的 admit 调用一行 + record_path 计算）
- super/tests/test_archive.py
- super/tests/test_loop.py

# 依赖关系
- 无（L1；loop.py 改动仅一行，与 task45 的并发改动错开——task45 在 L2，届时以本 task 的 archive 线程安全为基础）

# 测试方法
`python3 -m unittest super.tests.test_archive super.tests.test_loop -v` 全绿

# 禁止事项
- 不改 gates/prompts；不改 scheduler 并发结构（task45）
