# task22：agents 回合预算 + config 默认值

## 任务目标
env_8 实测：builder 单次中位 478s/646k token、极值 1575s/2101k（23 回合）——成本失控主因是回合数（每回合重算完整 input）。pi 无 max-turns flag → 框架侧在 NDJSON 流中计数 assistant 回合，超限优雅终止。

## 关键结果
- `super/superopt/agents.py`：`PiAgentClient.run(role, prompt_path, workdir, timeout_s, model=None, max_turns=None)`；流解析中统计 assistant `message_end` 次数；达到 max_turns → 走现有 kill 梯（SIGTERM→grace→SIGKILL）终止；`AgentResult` 增 `truncated: bool`（默认 False）；截断时 summary 前缀标注 `[turn-budget-exceeded: N] `，final_message 保留已产出的最后一条（便于上层 fallback 解析）
- `super/superopt/config.py`：DEFAULTS 增 `builder_max_turns: 12`、`coding_max_turns: 16`、`debug_max_turns: 10`、`adjust_max_turns: 8`（0/None = 不限；注释写明 env_8 实测依据：中位 3-7 回合、极值 23）
- `super/tests/test_agents.py`：新增用例——mock_pi 场景产生 >max_turns 回合 → 断言 truncated=True、进程被终止（wall 时间受限）、summary 标注；max_turns=None 不受限

# 写域
- super/superopt/agents.py
- super/superopt/config.py
- super/tests/test_agents.py

# 依赖关系
- 无（L1）

# 测试方法
`python3 -m unittest super.tests.test_agents -v` 全绿 + 全量无回归

# 禁止事项
- 不动 loop.py（task23 消费本 API）
