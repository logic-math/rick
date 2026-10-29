# task57：R1+R2 基础设施熔断 + 回合×协议失配修复

# 任务目标
env_11 分析发现的最高优先级两项：①68% 迭代是 API 配额空转（调度器无熔断）②回合预算与完成协议结构性失配（final_message=None → JSON 修复不触发 → debug 全量代偿）。

# 关键结果

## R1 基础设施熔断
- `super/superopt/agents.py`：
  - `_needs_failure_resume` 增强：识别**配额报错**（summary 含「额度」「credit-hub」「quota」「exceeded」关键词 且 tok=0）→ 标 `infra_failure=True`
  - AgentResult 增 `infra_failure: bool = False`
  - 配额报错时 **不再 resume**（resume 也只会失败）→ 直接返回带 infra_failure 标记的结果
- `super/superopt/loop.py`：
  - **三态失败分类**：`iteration_ended` payload 增 `outcome` 字段：
    - `delivered`（有候选入格/延迟入库）
    - `agent_failed`（agent 真实失败：self_check 有内容但 fail）
    - `turn_truncated`（truncated=True）
    - `infra_noop`（infra_failure 或 tok=0 或 wall<5s 且无产出）
    - `no_output`（其它无产出）
  - **熔断**：连续 `api_outage_threshold`（默认 3）次 `infra_failure` → 写 `api_outage_detected` 事件 + **暂停循环**（不推进 cursor、不建 codes/iterNNN 目录、不写空壳 decisions）
  - **定时探活恢复**：暂停后每 `api_outage_retry_s`（默认 60s）发一次 probe（跑一个极小 agent「回复 OK」）→ 成功则恢复循环（事件 `api_outage_recovered`）
  - `infra_failure_concluded` 加入 EVENT_TYPES + EVENT_SCHEMAS + 测试
- `super/superopt/config.py`：`api_outage_threshold: 3`、`api_outage_retry_s: 60`、`api_outage_probe_timeout_s: 30`
- `super/superopt/events.py`：EVENT_TYPES + EVENT_SCHEMAS 增 `api_outage_recovered`

## R2 产物落盘协议 + 截断可抢救
- `super/superopt/loop.py`：
  - **agent 完成后的产物提取**：`_run_coding` 和 `_run_debug` 在 agent_ended 后，如果 `result.valid=False` 但 **workdir 有产物**（solution.cpp/solution.py 存在且非空）→ **不触发 debug 重做**，而是从产物提取：
    1. 检查 `workdir/solution.*` 的 mtime 是否在本轮 agent 启动之后（= agent 写了但没输出 JSON）
    2. 是 → 把产物交给框架 gate_check 终验（与正常 candidate 同一路径）
    3. gate 通过 → 入格（绕过 agent JSON 契约的缺位——「做完了但没说」不再等于「没做」）
  - 事件：`artifact_rescued{code_path, reason: "agent_completed_but_no_json"}`
  - **截断可抢救**：`result.truncated=True` 时同样检查产物 → 有产物走上述路径
- `super/superopt/agents.py`：
  - `json_repair` 触发条件修正：`not result.valid and (not result.data)` （去掉 `result.final_message` 非空的要求——截断时 final_message=None 但产物可能在 workdir）
  - AgentResult 增 `artifact_path: str = ""`（如果 client 侧能定位产物文件）

## 测试
- `test_loop`：
  - ①熔断：3 次 infra_failure → api_outage_detected → cursor 不推进 → 不建 iterNNN 目录
  - ②恢复：暂停后 probe 成功 → api_outage_recovered → 循环继续
  - ③产物抢救：agent valid=False 但 workdir 有 solution.cpp → gate_check 通过 → 入格 + artifact_rescued 事件
  - ④截断抢救：truncated=True 但产物在 → 同上
  - ⑤ outcome 分类：delivered/agent_failed/turn_truncated/infra_noop/no_output 各至少一例
- `test_agents`：配额报错识别 → infra_failure=True → 不 resume
- `test_events`：新事件类型齐备

# 写域
- super/superopt/agents.py
- super/superopt/loop.py
- super/superopt/config.py
- super/superopt/events.py
- super/tests/test_loop.py
- super/tests/test_agents.py
- super/tests/test_events.py

# 依赖关系
- 无（L1）

# 测试方法
`python3 -m unittest super.tests.test_loop super.tests.test_agents super.tests.test_events -v` 全绿 + 全量无回归

# 禁止事项
- 不改 gates.py；不改 prompts
