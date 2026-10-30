# task56：agent 确定性输出 + session resume + 回合软监控（env_11 反思修复）

# 任务目标
human 三项裁决：
1. **回合预算是软约束**——不截断；超限告警给中控/human
2. **JSON 输出确定性保证**——pi hook 层面：没输出 JSON 或 JSON 不合规 → 直接告诉 agent 重新输出（一次 resume），而非标 invalid 触发全链浪费
3. **失败恢复用 session resume**——agent 执行失败/截断/空回 → 用 pi --session-id 恢复会话继续，避免重跑整个 agent 浪费 token

# 关键结果

## A. 回合软监控（替代硬截断）
- `super/superopt/config.py`：所有 `*_max_turns` 默认改 **0**（不截断）；新增 `turn_soft_warn: 20`（告警阈值）
- `super/superopt/agents.py`：
  - `max_turns` 保留为**紧急硬杀**（0=不限；非零时仅当超 `max_turns * 3` 才真正杀——极端防护）
  - 回合达到 `turn_soft_warn` → AgentResult 增 `turn_warn=True`（事件可观测）
- `super/superopt/loop.py`：`agent_ended` 事件增 `turn_warn` 字段；连续 ≥3 次同角色 turn_warn → 事件 `turn_budget_alert{role, avg_turns, soft_warn}`（中控/human 感知）
- `super/prompts/*.md`：去掉「回合上限」相关表述（如果模板有提及）

## B. JSON 输出确定性 hook（在 PiAgentClient 层面，不是 pi 扩展）
- `super/superopt/agents.py` `run()` 内部改造：
  - 第一次跑完后：解析 final_message 提取 JSON（六字段 + 角色增量）
  - **JSON 缺失/不合规 → 自动 resume 一次**：用 `--session-id` 恢复同一会话 + 发追加消息「你的最终回复缺少必需的 JSON 输出块。请按系统提示词的输出契约，只输出那个 JSON 代码块（```json ... ```），不要任何其它内容」
  - resume 后再解析；仍失败 → 标 valid=False（此时已尽两次，才走旧路径）
  - AgentResult 增 `json_repair: int`（0/1——resume 了几次修 JSON）
- **session-id 管理**：每次 run 分配 `session_id = "superopt-{role}-{iter_uuid}"`；`_build_cmd` 加 `--session-id`；resume 用同 id
- **超时/异常失败恢复**：`run()` 首次因网络/超时/异常终止（exit_code != 0 或 usage.total_tokens==0）→ **resume 一次**（同一 session，追加消息「你刚才被中断了。请继续完成任务，最终回复输出约定 JSON」）；恢复仍失败 → 返回结果（不无限重试）

## C. self_check 语义修正
- `super/superopt/loop.py`：coding/debug 的 `self_check.passed=False` 但 **metrics 为空 {}** → 不触发 debug 循环（说明 agent 没跑完自检，不是实验失败——触发 debug 只会浪费）
- 事件：`self_check_skip_debug{reason: "empty_metrics_no_debug"}` 可观测

## D. API 断连检测
- `super/superopt/loop.py`：连续 N=3 个 agent 空回（usage.total_tokens == 0 且 exit_code == 0 且 wall < 10s）→ 事件 `api_outage_detected` + 暂停调度（等 60s 后重试一次；仍空回 → 暂停并等 human）
- `super/superopt/config.py`：`api_outage_threshold: 3`、`api_outage_retry_s: 60`

## E. 特征空间总量软约束
- `super/skills/gen_env_skill/skill.md`：增「格子数起步建议 ≤20（轴值笛卡尔积）」到特征空间节
- `super/superopt/loop.py`：`_apply_expand_cell` 前检查 `fs.all_cells()` 数量 > `max_cells`（默认 100）→ 拒绝 + 事件 `cell_limit_reached`

## F. new_cell 必须带 description
- `super/superopt/loop.py`：`_apply_proposal` 的 `new_cell` 分支：method 为空 → 拒绝（事件 `new_cell_rejected{reason: "empty_method"}`）+ 写入 description
- `super/prompts/adjust.md`：new_cell 契约强调 method 必填

# 写域
- super/superopt/agents.py
- super/superopt/loop.py
- super/superopt/config.py
- super/prompts/adjust.md
- super/prompts/coding.md
- super/tests/test_agents.py
- super/tests/test_loop.py

# 依赖关系
- 无（L1）

# 测试方法
`python3 -m unittest super.tests.test_agents super.tests.test_loop -v` 全绿：
- JSON 修复：mock 场景第一次无 JSON → resume → 第二次有 → valid=True + json_repair=1
- 失败恢复：mock 场景超时 → resume → 成功 → AgentResult 有效
- 空回检测：3 次空回 → api_outage_detected
- self_check 空 metrics 不触发 debug
- turn_warn 软告警事件
- new_cell 空 method 拒绝

# 禁止事项
- 不改 pi 本体（JSON 修复在 client 层 resume 实现）；不改 gates.py
