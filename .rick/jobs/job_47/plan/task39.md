# task39：loop 接线（adjust 四步 / builder 相邻格+调研 / coding TDD 上下文 / debug 三类触发 / 实验记录注入）

# 任务目标
把新 loop 全部接进调度器（依赖 task38 的 archive/experiment_record/feature_space 扩展）。

# 关键结果
- **adjust 四步上下文**（`_maybe_start_adjust` ctx 扩展）：
  - `experiment_records`：`experiment_record.recent(ws, since_iter=上次 adjust 迭代, until_iter=当前)`（区间内全部实验摘要）
  - `experiment_sessions`：区间内 agent pi session 路径清单（`~/.rick/pi/agent/sessions/` 下按 cwd 匹配 `codes/iter{N}` 的目录，best-effort；找不到则空）
  - `decision_history`：近 3 份 `decisions/*.md` 摘要
  - `cell_descriptions`：archive 各格 description（name/method/source_refs）
  - `decisions_path`：`<ws>/decisions/iter{N}.md`
  - 记录 `self._last_adjust_iter = cursor`
- **决策记录**（`_collect_adjust`）：agent 若未写 decisions 文件 → 框架按 proposals+evidence 合成一份最小决策记录（标注 `framework_synthesized: true`）；事件 `adjust_decision` 增 `decision_path`
- **格动作应用扩展**（`_apply_proposal`）：
  - `new_cell`（action=create）：走 feature_space 热重载链加格（复用 expand_cell 的 add_cell 路径）+ archive.set_description + `cell_created` 事件
  - `modify_cell`（action=improve_description）：archive.set_description + `cell_modified` 事件（含 method/source_refs/rationale）
- **builder ctx 扩展**（`_run_builder`）：`target_record`（该格精英 code_path 对应迭代目录的实验记录 digest）、`neighbor_samples`（`fs.sample_neighbors(cell_id, 2)` → 每格 {cell_id, description, elite_code_path, record_digest}）、`research_report_path`、`experiment_record_path`；builder 完成后检查两个文件是否存在（缺失→事件标注 `research_report: false` / `record_updated: false`）
- **coding ctx 扩展**（`_run_coding`）：`research_report`（内容截断 3000）、`experiment_record`（内容）、`coding_plan_path`、`test_path`；完成后校验 `test_solution.py` 是否存在（TDD 痕迹，事件 `tdd_trace: true/false`）并检查实验记录 mtime 更新
- **debug 三类触发**（`_verify_and_admit` 后）：
  - ①门禁失败（已有）
  - ②`gr.passed and perf_gain_pct >= debug_policy.gain_threshold_pct`（`perf_gain_pct` 相对该格精英或全局最优计算）→ debug 复验（attempt=100，reason=`suspicious_gain`）
  - ③`gr.tolerance_band and debug_policy.recheck_tolerance_band` → debug 复验（reason=`tolerance_band_recheck`）
  - debug ctx 增 `experiment_record`、`perf_gain_pct`、`recheck_reason`
  - debug 结论 `keep_original` → 原候选仍可入格（记录 debug 结论事件）
- **实验记录注入**：coding/debug 的 ctx 注入 `experiment_record`（读现有内容）
- **config**：DEFAULTS 增 `debug_gain_threshold_pct: 20`、`debug_recheck_tolerance_band: true`（env.json `debug_policy` 覆盖）
- 测试 test_loop：①adjust ctx 含 experiment_records/decision_history/cell_descriptions ②决策记录合成回退 ③new_cell 建格含 description + 热重载 ④modify_cell 改描述 ⑤builder 相邻格 2 个 + 两文件校验 ⑥coding TDD 痕迹校验 ⑦debug 三类触发（suspicious_gain/tolerance_band_recheck/门禁失败）与 keep_original ⑧实验记录注入 debug/coding

# 写域
- super/superopt/loop.py
- super/superopt/config.py
- super/tests/test_loop.py

# 依赖关系
- task38（archive/experiment_record/feature_space 扩展）

# 测试方法
`python3 -m unittest super.tests.test_loop -v` 全绿 + 全量无回归

# 禁止事项
- 不动 gates.py；不动 prompts（task40）；不动 gating 契约
