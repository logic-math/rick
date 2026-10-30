# task58：R3+R4 幻影 elite 拒绝 + method_stale 自动标记

# 任务目标
env_11 分析发现的台账可信度问题：①阴性结果（keep_original）被 gate 入账为新纪录 ②method 基座比实际精英落后 2 代。

# 关键结果

## R3 幻影 elite 拒绝（同码重测不入账）
- `super/superopt/loop.py` `_verify_and_admit`：
  - 入格前**代码等价检查**：计算候选代码文件的 md5（或 content hash）
  - 与当前格精英的 `elite_code_hash`（archive 新字段）比较
  - **等价 → 拒绝 elite_replaced**，只刷新 `perf_value`（测量值更新），不刷新 `gen`/位次/方法基座
  - 事件：`elite_measurement_refresh{cell_id, code_hash, new_perf, old_perf}`（vs elite_replaced）
  - archive：格 schema 增 `elite_code_hash: str`（admit 时计算并存储）
- `super/superopt/archive.py`：
  - `admit()` 接受 `code_hash` 参数；`get_code_hash(cell_id)` 访问器
  - 替换时写入新 hash
- **ref 漂移防线**：`gate_check` 结果中 `ref_runtime_ms` 与该格精英的 `ref_runtime_ms` 偏差 > `ref_drift_threshold`（默认 0.15）→ 事件 `ref_drift_warning{old_ref, new_ref, drift_pct}` + gate_result 标注 `measurement_degraded: true`（按 env 的 tolerance 处理，不硬拒）

## R4 elite_replaced 时自动标记 method_stale
- `super/superopt/loop.py`：
  - `_verify_and_admit` 中 elite_replaced 发生时 → `archive.set_description` 的 method 前插入 `⚠ method_stale: elite gen{N} > method 基座 gen{M}，需要 update`（或者独立的 `stale: bool` 字段）
  - archive 格 schema 增 `method_stale: bool`（默认 False；elite_replaced → True；adjust 的 modify_cell 清除）
  - **builder ctx 注入**：`cell_descriptions` 中 stale=True 的格标注 `[STALE]`，builder prompt 强制以 `elite_code_path` 实际代码为第一事实源
- `super/superopt/archive.py`：`set_stale(cell_id, stale)` / `is_stale(cell_id)`
- `super/prompts/adjust.md`：modify_cell 语义增「清除 stale 标记」

## 测试
- `test_loop`：
  - ①同码入格：md5 相同 → elite_measurement_refresh（非 elite_replaced），gen 不变
  - ②不同码入格 → elite_replaced + method_stale=True
  - ③method_stale 格 → builder prompt 含 [STALE] 标注
  - ④ref 漂移超阈值 → ref_drift_warning
  - ⑤adjust modify_cell → method_stale 清除

# 写域
- super/superopt/loop.py
- super/superopt/archive.py
- super/prompts/adjust.md
- super/tests/test_loop.py
- super/tests/test_archive.py

# 依赖关系
- task57（loop.py 写域串行）

# 测试方法
`python3 -m unittest super.tests.test_loop super.tests.test_archive -v` 全绿

# 禁止事项
- 不改 gates.py（md5 检查在 loop 层）
