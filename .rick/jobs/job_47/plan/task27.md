# task27：gates.py 三要素门禁（gate_check 优先 + legacy 回退）

## 任务目标
三要素成为 env 基石契约后，框架门禁只调**封装好的 `skills/gate_check.py`**（内部级联 verify→benchmark→软门禁），消灭「框架一套、agent 一套」的语义漂移（env_8 it19 根因）。旧 env（env_7/8/9）走 legacy 回退不迁移。

## 关键结果
- **布局探测**：`<env>/skills/gate_check.py` 存在 → 新布局；否则回退 legacy（`data/test_correctness.py` + `data/bench.py`）
- **新布局 gate_check 契约**（CLI）：
  ```
  <python> skills/gate_check.py --candidate <path> [--baseline <perf_value>] [--env-dir <dir>]
  stdout 单行 JSON:
    {"passed": bool, "failure_stage": "stage2_correctness"|"stage3_perf"|"soft_gate"|null,
     "metrics": {"correctness": {...}, "perf": {"value": num, "score": num, "k": 3, "timings": [...]}, "soft": {...}},
     "details": "..."}
  ```
  gate_check 内部：verify.py（约束）→ benchmark.py（K 次中位）→ 软门禁（与 --baseline 比较，relative_band 从 env.json 读）→ 汇总 JSON
- **gates.py 改造**：`_run_script` 支持新布局；stage2/stage3 合并为「调 gate_check 一次」并解析其 `failure_stage`；超时用 `gates.gate_check_timeout_s`（缺省 600）
- **stage1 环境健康升级**：新布局下检查 `skills/{profile,benchmark,verify,gate_check}.py` 四件齐备 + reference 过 verify；legacy 布局保持原检查（data 两件 + reference）
- **preverify_red**（门先证红）保持可用（走 verify 或 gate_check）
- 测试 `test_gates.py`：新布局 fixture（四件齐备）全链 pass/fail 各 stage；legacy fixture 回退仍绿；缺 profile/benchmark/verify 任一 → stage1 拒；gate_check 超时 → 失败诊断；--baseline 传入时软门禁判定正确（band 内/外）

# 写域
- super/superopt/gates.py
- super/tests/test_gates.py

# 依赖关系
- 无（L1）

# 测试方法
`python3 -m unittest super.tests.test_gates -v` 全绿 + 全量无回归

# 禁止事项
- 不移除 legacy 支持（env_7/8/9 仍需可跑）；不动 loop.py（task28）
