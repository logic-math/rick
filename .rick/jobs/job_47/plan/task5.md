# task5：agents.py pi 池客户端 + mock_pi + 单元测试

## 任务目标

实现 py 调度器与 pi agent 的桥：subprocess 直调 `pi --mode json`（J4 裁决：每迭代一进程，env 热更新友好），NDJSON 事件流解析，**token 计量**（research-L3-pi-token 方案：聚合 message_end(message.role==assistant).message.usage + 兜底链），产出 AgentResult（六字段基座 + 角色增量 + usage）。

## 关键结果

- 新建 `super/superopt/agents.py`（stdlib-only）：
  - `@dataclass AgentResult`：六字段基座 valid/score/perf_value/perf_type/perf_desc/summary + `usage: dict`（input_tokens/output_tokens/cache_read/cache_write/total_tokens/cost_usd）+ `role` + `final_message: str` + `tool_calls: list` + `wall_ms: int` + `exit_code: int`
  - `class PiAgentClient`：
    - `__init__(self, pi_path: str, agent_dir: Optional[str] = None)`——pi_path 必填（如 `/Users/logic/.rick/pi/agent/runtime/node_modules/.bin/pi`）；agent_dir 设 PI_CODING_AGENT_DIR 环境变量（继承托管配置，缺省不设）
    - `run(self, role: str, prompt_path: str, workdir: str, timeout_s: int = 1800) -> AgentResult`——命令：`pi --mode json --append-system-prompt <prompt_file> -p <bootstrap> --cwd <workdir>`；bootstrap 用最小指令（「按系统提示词完成任务，最终回复输出约定 JSON」）；NDJSON 逐行解析（subprocess.PIPE，select/stream 读取防死锁）：
      - 终止判定：流自然结束（agent_settled 事件或 EOF）或超时（SIGTERM→5s→SIGKILL）
      - **token 计量**：聚合所有 `message_end` 且 `message.role=="assistant"` 的 `message.usage`（input/output/cacheRead/cacheWrite 字段名按 research-L3-pi-token 实测：usage.input/output/cacheRead/cacheWrite/totalTokens 或近似——**以 research 简报的字段映射为准**，缺字段=0）；兜底：全零时取最后一次非零 `message_update.usage`；仍零 → usage 全 0（CostLedger 侧降级标注）
      - final_message：最后一条 assistant message_end 的文本
      - tool_calls：tool_execution_end 计数（name 去重列表）
      - **六字段提取**：final_message 内嵌 JSON 块（```json ... ``` 或裸 JSON）解析出 valid/score/perf_*/summary；解析失败 → valid=false, summary=final_message 前 200 字（不炸循环）
- 新建 `super/tests/mock_pi.py`：fake pi 脚本（照 rick tests/mock_agent 范式 + bugs.md fake 脚本坑位：shebang 恢复 PATH、无 bashism）——`MOCK_SCENARIO` 环境变量驱动多场景：
  - `ok_builder`：输出合法 NDJSON（2 条 message_end 含 usage 非零 + assistant final 含六字段 JSON）
  - `ok_coding`：含 tool_execution_end ×3 + final JSON
  - `zero_usage`：message_end 无 usage → 兜底 message_update 有非零 usage → 计量仍非零
  - `all_zero`：全程零 usage → AgentResult.usage 全 0（降级路径）
  - `bad_json`：final 是废话无 JSON → valid=false 优雅降级
  - `timeout_sleep`：sleep 无限 → 触发客户端超时 kill → exit_code 非 0
- 新建 `super/tests/test_agents.py`（unittest）：mock_pi 六场景全覆盖断言（usage 聚合数值精确断言：两 message_end 的 input 相加等；超时用 2s/1s 短参）
- `super/tests/__main__` 不需要——测试直接 `unittest super.tests.test_agents`

# 写域

- super/superopt/agents.py（新建）
- super/tests/mock_pi.py（新建）
- super/tests/test_agents.py（新建）

## 依赖关系

- task1/task2/task3/task4（Kahn L2）——实际弱依赖：AgentResult dataclass 自足；usage 结构与 task4 CostLedger 对齐（字段名契约，无 import）

## 实现要点

- NDJSON 字段名大小写防御（pi 事件 schema 的 usage 键名——research-L3-pi-token 已实测：按简报落，遇缺键取 0 不 raise）
- subprocess 读取用线程或 select（macOS 无 poll timeout 问题，用线程读 stdout/stderr 最稳）
- prompt 文件路径绝对化；workdir 必须存在（codes/{uuid}/ 由调用方建好）
- timeout 梯与 Go 侧 supervisor 语义一致（TERM→grace→KILL）

## 测试方法

`python3 -m unittest super.tests.test_agents -v` 全绿（mock_pi 可执行位 chmod +x；测试以 `python3 super/tests/mock_pi.py` 直接调——PiAgentClient 支持 `pi_path` 指向 `python3 mock_pi.py` 包装：**取 exec 形式**：pi_path 可执行文件，mock_pi.py 加 shebang）

## 禁止事项

- 不得 import task1-4 模块（六字段与 usage 是 dataclass 契约）；不得在客户端做门禁判断（gates 是 task7）；不得用 pi RPC（subprocess 直调是 J4 终裁）
