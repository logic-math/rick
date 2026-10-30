# task59：R5 度量口径分离 + 停止判据

# 任务目标
让复盘从「猜」变成「读」：outcome 分类（task57 已做 iteration_ended.outcome）+ 停止判据锚定在「剩余空间 < 测量分辨率」。

# 关键结果
- `super/superopt/loop.py`：
  - **同码双测噪声带标定**：每 `noise_calibration_interval`（默认 10）迭代，框架跑一次**在位精英的重测**（与 gate_check 同协议）→ 计算与档案值的相对差 → 写事件 `noise_calibrated{cell_id, current_perf, archived_perf, noise_pct}` → 存入 `self._noise_band`
  - **停止判据**（`_saturation_signal` 增强）：
    - 原有条件保留（stale_iterations / all_visited / champion_stale）
    - 新增：`noise_band > 0` 且 `top3_spread < noise_band`（头三名差距小于噪声带 → 无法可靠区分）
    - 新增：`identified_headroom < noise_band`（已识别剩余空间合计 < 噪声带）
    - 触发时 saturation 增加 `stop_reason: "headroom_below_noise"` → adjust 可据此提 terminate_recommend
- `super/superopt/config.py`：`noise_calibration_interval: 10`、`ref_drift_threshold: 0.15`
- `super/tools/cost_report.py`：按 `outcome` 分类分列统计（delivered/infra_noop/agent_failed/turn_truncated/no_output 各多少迭代、各耗多少 token）——让成本报告直接显示「真实成本 vs 空转成本」
- 测试：
  - 噪声标定事件 + noise_band 计算
  - top3_spread < noise_band → saturation 触发 + stop_reason
  - cost_report --workdir 按分类输出

# 写域
- super/superopt/loop.py
- super/superopt/config.py
- super/tools/cost_report.py
- super/tests/test_loop.py

# 依赖关系
- task57、task58（loop.py 串行）

# 测试方法
`python3 -m unittest super.tests.test_loop -v` 全绿 + cost_report 冒烟

# 禁止事项
- 不改 prompts
