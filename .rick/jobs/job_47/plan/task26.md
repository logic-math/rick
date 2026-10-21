# task26：ledger.py 实验台账模块

## 任务目标
env_8 实证：coding agent 自发在迭代目录写 `ab_notes.md` 补系统缺失的「实验档案」层。本任务把这一层做成系统能力——跨迭代可查的实验台账，供四角色消费。

## 关键结果
- 新建 `super/superopt/ledger.py`（stdlib-only，py3.9）：
  - `class Ledger`：`__init__(ws)`；落盘 `<ws>/ledger.jsonl`（append-only，单行 JSON）
  - `append(entry: dict) -> None`：容错（非法字段不炸）；自动补 `ts`
  - `read_all() -> list`（坏行跳过）
  - `brief(n: int = 8) -> list`：最近 n 条精简摘要 `[{iteration_id, cell_id, idea(截 120), verdict, perf_value, perf_delta, reject_reason}]`
  - `perf_timeline(n: int = 10) -> list`：`[{iteration_id, cell_id, perf_value, is_record: bool}]`（从台账推，is_record=当时全局最优刷新）
  - `own_decisions(role: str, n: int = 5) -> list`：某角色最近 n 条决策摘要（adjust 用：自己的历史提案+效果）
- 条目 schema（约定，loop 侧填充）：`{iteration_id, cell_id, idea, parents, plan_steps_count, gate: {passed, failure_stage, perf_value}, verdict: "admit"|"replace"|"reject"|"no_candidate", perf_delta, reject_reason, agents: {role: {wall_ms, tokens, truncated}}}`
- `super/tests/test_ledger.py`：append/read_all/brief/perf_timeline(is_record 正确)/own_decisions/坏行容错/空文件

# 写域
- super/superopt/ledger.py（新建）
- super/tests/test_ledger.py（新建）

# 依赖关系
- 无（与 task22 并行，写域互斥）

# 测试方法
`python3 -m unittest super.tests.test_ledger -v` 全绿

# 禁止事项
- 不碰 loop.py（task23 接线）；不依赖第三方库
