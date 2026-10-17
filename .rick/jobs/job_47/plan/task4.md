# task4：config.py + budget.py + lineage.py + 包骨架

## 任务目标

实现调度器的配置层（super.json 读写 + CLI 合并）、成本账本（时间/token 双轨——token 按 research-L3-pi-token 方案落位）、lineage 重建（从事件流重建谱系，修复 v5 parent_hash 恒 None），以及 super/ 包骨架（__init__/tests 公共设施）。

## 关键结果

- 新建 `super/superopt/__init__.py`（空或版本常量）
- 新建 `super/superopt/config.py`：
  - `DEFAULTS`：{adjust_interval: 5, debug_max_attempts: 3, soft_gate_k: 3, heartbeat_interval_s: 30, heartbeat_timeout_s: 90, term_grace_s: 5, max_wall_time_h: 24.0, max_iterations: 0(=无限制), python_path: "python3", pi_path: ""(空=用 super.json 显式值，无则报错)}——与 ~/.rick/config.json 的 super 段（Go 侧 task11 写默认）一致，但本模块只读 super.json（工作区级），不读 ~/.rick（那是 Go 侧职责）
  - `load(ws: str) -> dict`——读 `<ws>/super.json`；缺键补 DEFAULTS
  - `merge_cli(cfg: dict, args) -> dict`——argparse Namespace 覆盖（仅非 None 项）
  - `init_ws(ws: str, env_path: str) -> dict`——首次初始化 super.json（state=idle/iteration_cursor=0/env=env_path/budget={wall_start, max_wall_time_h}/config 各默认）+ 建目录骨架（events.jsonl 空文件/archive.json 初始/interventions//codes//prompts/ 若缺）
- 新建 `super/superopt/budget.py`：
  - `class CostLedger`：
    - `record_agent(self, role: str, wall_ms: int, tool_calls: int, usage: Optional[dict]) -> None`——usage 结构 {input_tokens, output_tokens, cache_read, cache_write, total, cost_usd}（research-L3-pi-token：聚合 message_end(message.role==assistant).message.usage；兜底 message_update 最后非零；仍零 → total=0 由 summary 降级标注）
    - `record_event(self, type: str, payload: dict)`——**空实现占位**（事件即账：budget 的真相源就是 events.jsonl 的 agent_started/ended 对；CostLedger 只做内存聚合 + 周期性 budget_heartbeat 事件写出由 loop 调用）
    - `summary(self) -> dict`——分 role：{wall_ms, tool_calls, input_tokens, output_tokens, total_tokens, cost_usd} + 总计；零 token 时附 `{"token_degraded": true}`
    - `within_budget(self) -> bool`——wall-clock 预算检查（max_wall_time_h；无迭代上限）
- 新建 `super/superopt/lineage.py`：
  - `rebuild_lineage(events: list, cell_id: str) -> list`——从事件流重建某格精英谱系：elite_admitted/elite_replaced 事件链（payload 含 parents/code_path/gen）→ 返回 [code_path,...] 祖先链（新→旧）
  - `rebuild_all(events: list) -> dict`——全格谱系投影（重放恢复与北极星计算共用）
  - `project_state(events: list) -> dict`——**游标重放投影**：events → {iteration_cursor, state, cells: {visit/weight 恢复}, pending, last_seq}——task8 的 `_recover` 以此为基座（本 task 实现纯函数投影，task8 接线）
- 新建 `super/tests/__init__.py`（空，使 discover 可用）+ `super/tests/test_config.py` + `test_budget.py` + `test_lineage.py`：
  - config：init_ws 建骨架 → load 往返一致；merge_cli 覆盖语义
  - budget：record 聚合正确；summary 分 role；零 token 降级标记
  - lineage：构造 10 条事件（2 格 3 代）→ rebuild_lineage 链序正确；project_state 恢复 visit/weight/cursor

# 写域

- super/superopt/__init__.py（新建）
- super/superopt/config.py（新建）
- super/superopt/budget.py（新建）
- super/superopt/lineage.py（新建）
- super/tests/__init__.py（新建）
- super/tests/test_config.py、test_budget.py、test_lineage.py（新建）

## 依赖关系

- 无（Kahn L1；纯 stdlib 自足。lineage.project_state 消费 task1 的 EVENT_TYPES 字符串常量——**以字符串字面量引用**（"elite_admitted" 等），不 import task1，保持并行无依赖）

## 实现要点

- Python ≥3.9 兼容
- super.json 的 budget 段是**运行时预算状态**（wall_start 进程启动即更新）；config 段是参数——两层分离
- CostLedger 不持久化（内存聚合，事件流是持久真相）

## 测试方法

`python3 -m unittest super.tests.test_config super.tests.test_budget super.tests.test_lineage -v` 全绿

## 禁止事项

- 不得读 ~/.rick（工作区级自治）；不得 import task1/2/3 模块（并行层禁耦合）
