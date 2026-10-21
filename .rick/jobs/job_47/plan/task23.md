# task23：loop 输入完整性 + saturation 信号 + terminate_recommend + 预算传参

## 任务目标
修 env_8 实测暴露的三个缺口：①builder ctx 缺 env_dir（导致 env/prompts 里硬编码绝对路径）②数据饱和信号未注入（18 轮无新纪录，adjust 仍只调权重、expand_cell 0 次）③缺「认输/换轴」表达。

## 关键结果
- **ctx 完整性**：`_run_builder` ctx 增 `env_dir`（= self.env_dir）、`perf_profile_cmd`（若 `<env>/skills/perf_profile.py` 存在则 `<venv python|python3> <该路径> --candidate <path>` 形式，否则空串）、`saturation`；`_run_debug` ctx 增 `perf_profile_cmd`、`saturation`
- **饱和信号** `_saturation_signal() -> dict | None`：满足全部条件时返回 dict（否则 None）——
  - `stale_iterations`：自上次全局最优（archive 全精英最优 perf）刷新以来的完成迭代数 ≥ 3
  - `all_visited`：所有 weight>0 的格 visit ≥ 2
  - `champion_stale`：当前最优格近 3 次采样未刷新精英
  返回 `{"stale_iterations": N, "all_visited": true, "champion_cell": cid, "best_perf": v, "alive_cells": K}`
  实现：`self._best_perf` 在 `_verify_and_admit` 入格时更新（更优则刷新计数），`_stale_iters` 在 `_step` 尾部递增/清零；注入到 builder 与 adjust 的 ctx 键 `saturation`（None 时渲染为空/缺省）
- **adjust ctx**：`_maybe_start_adjust` 的 ctx 增 `saturation`
- **terminate_recommend 支持**：`_apply_proposal` 识别 `type == "terminate_recommend"` → 写 `proposals/{pid}.json`（`needs_human: true`）+ `proposal` 事件（不进裁决队列自动执行——它是建议，供中控会话/人读）；**不调用**裁决器（adjudicator 收到也忽略）
- **预算传参**：`_dispatch_agent` 与 adjust 线程调 `client.run(..., max_turns=int(conf.get("<role>_max_turns") or 0) or None)`；agents.py 签名见 task22（`run(role, prompt_path, workdir, timeout_s, model=None, max_turns=None)`）
- 测试 `super/tests/test_loop.py`：①builder prompt 含 env_dir 与 perf_profile 路径（fixture 造 env/skills/perf_profile.py）②饱和信号：构造 stale 场景（多次迭代 perf 不刷新）断言 ctx 里 saturation 非空且字段正确；不饱和时为空 ③terminate_recommend：adjust 提案 → proposals/ 文件含 needs_human + proposal 事件，archive 无变化 ④max_turns 传参（fake client 记录 kwargs）

# 写域
- super/superopt/loop.py
- super/tests/test_loop.py

# 依赖关系
- task22（client.run 的 max_turns 签名）

# 测试方法
`python3 -m unittest super.tests.test_loop -v` 全绿 + 全量无回归

# 禁止事项
- 不动 agents.py/config.py（task22 域）；不动 super/prompts/*.md（task24 域）
