# task23：loop.py ctx 全面增强 + gate 失败接 debug + ledger 接线

## 任务目标
env_8 信息审计暴露的全部盲点一次修完（8 项）。依赖 task22（agents.run 的 max_turns）与 task26（Ledger API）。

## 关键结果

### A. adjust：recent_events 给真值（P0）
`_recent_events_summary(n=15)` 重写——按事件类型提取**关键字段值**（不再是 payload_keys）：
- `gate_result`: `{iteration_id, cell_id, passed, failure_stage, stage, perf_value, soft_verdict}`
- `agent_ended`: `{iteration_id, role, wall_s, total_tokens, truncated, valid}`
- `cell_sampled`: `{iteration_id, cell_id, weight_share, cold_start, visit}`
- `elite_admitted`/`elite_replaced`: `{iteration_id, cell_id, perf_value, gen, score}`
- `weight_changed`: `{cell_id, old_weight, new_weight, rule_id}`
- `candidate_produced`: `{iteration_id, cell_id, self_check_passed, diff_path}`
- `adjust_decision`: `{iteration, proposals 摘要}`
- `intervention`: `{raw}`；`resurrection`/`invalidation`: `{cell_id, reason}`
- `heartbeat` 过滤（噪音）

### B. 新增 ctx 组装方法（供四角色）
- `_perf_timeline(n=10)`：委托 Ledger.perf_timeline
- `_budget_status()`：`{iterations_used, max_iterations, wall_minutes, max_wall_h, tokens_used, max_tokens, tokens_pct}`
- `_gate_defs()`：读 env.json 提取 `{tolerance, bench:{size,k,timeout_s}, gates:{score_max,runtime_ms_min,ref_runtime_band_ms}}`
- `_domain_brief(limit=2000)`：读 env/domain_knowledge.md 截断（优先保留「## 性能分析 SOP」「## 性能调优 SOP」「死族/天花板」相关行）
- `_self_check_and_profile_cmds()`：返回 `(self_check_cmd, perf_profile_cmd)`（venv python 优先；perf_profile 不存在则空串）
- `_workdir_files(iter_dir)`：工作区文件名列表（父本+solution）
- `_recent_interventions(3)`：intervention 事件原文

### C. 各角色 ctx 扩展
- **builder**：`env_dir`、`perf_profile_cmd`、`self_check_cmd`、`saturation`、`domain_brief`、`perf_timeline`、`budget_status`、`ledger_brief`、`gate_defs`
- **coding**：`gate_defs`、`perf_baseline`（该格精英 perf 或空）、`workdir_files`、`ledger_brief`、`saturation`
- **debug**：`perf_profile_cmd`、`gate_defs`、`ledger_brief`、`saturation`、`gate_failure`（框架门禁失败详情，见 D）
- **adjust**：`saturation`、`perf_timeline`、`budget_status`、`domain_brief`、`ledger_brief`、`own_decisions`（Ledger.own_decisions("adjust")）

### D. gate 失败 → debug 接管（P0）
`_step` 中 `_verify_and_admit` 后：若 gate 失败且 `failure_stage ∈ {stage2_correctness, stage3_perf, soft_gate, stage1_env?}`（stage1 环境失败不修）→ 调 `_run_debug`（ctx 含 `gate_failure={failure_stage, metrics, details}`，attempt 从 100 起编号以免与自检 debug 混淆）**最多 1 次**框架修复机会 → debug 产出候选后**再跑一次** `_verify_and_admit`；仍失败则按原逻辑记失败。
- 需防死循环：每迭代最多 1 次 gate→debug

### E. saturation 信号
`_saturation_signal()`：`stale_iterations ≥3`（自上次全局最优刷新）+ 全部 weight>0 格 `visit ≥2` + 冠军格近 3 次采样无刷新 → 返回 `{stale_iterations, all_visited, champion_cell, best_perf, alive_cells}`；否则 None。`self._best_perf`/`self._stale_iters` 在 `_verify_and_admit` 与 `_step` 尾维护。

### F. terminate_recommend
`_apply_proposal` 识别 `type=="terminate_recommend"` → 写 `proposals/{pid}.json`（`needs_human: true`）+ `proposal` 事件；**不进裁决自动执行**。

### G. ledger 接线
`_step` 收尾组装一条 Ledger 条目（A 段 schema）→ `self.ledger.append(...)`（Ledger 在 `__init__` 构造，懒建目录）。

### H. max_turns 传参
`_dispatch_agent` 与 adjust 线程：`max_turns=int(conf.get("<role>_max_turns") or 0) or None`，透传 `client.run(..., max_turns=...)`。

### I. 测试（test_loop.py 扩展）
①recent_events 含 gate_result 的 failure_stage 真值（造失败事件）②perf_timeline/budget_status/gate_defs/domain_brief 非空且结构正确 ③builder/coding/debug/adjust 的 prompt 文件各含新 ctx 键渲染值（如 gate_defs 的 relative_band、saturation）④gate 失败→debug→重验成功的全链（假 gate 先拒两次后过）⑤gate 失败→debug 仍失败 → 记失败不崩 ⑥terminate_recommend 落 proposals + 事件 + archive 不变 ⑦ledger 条目写入（含 verdict/perf_delta/reject_reason）⑧max_turns 透传（fake client 记 kwargs）⑨saturation 触发/不触发两态

# 写域
- super/superopt/loop.py
- super/tests/test_loop.py

# 依赖关系
- task22（client.run max_turns 签名）、task26（Ledger API）

# 测试方法
`python3 -m unittest super.tests.test_loop -v` 全绿 + `python3 -m unittest discover -s super/tests -p "test_*.py"` 全量无回归

# 禁止事项
- 不动 agents.py/config.py/ledger.py；不动 super/prompts/*.md（task24）
