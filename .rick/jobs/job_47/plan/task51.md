# task51：成本优化（debug 不截断 + 一次性根因 + builder 相邻格摘要）

# 任务目标
human 裁决：**现在 debug 的浪费就是截断导致的**——要求「一次性根因定位，不要截断」。同时按 cost_report 诊断砍 builder 上下文。

# 关键结果
- `super/superopt/config.py`：
  - `debug_max_turns: 16 → 0`（**0 = 不设回合上限**，由 `agent_timeout_s`（900s）与 wall budget 兜底）
  - `builder_max_turns: 20`（保持）、`coding_max_turns: 24`（保持）
  - 新增 `event_fsync: true`（task49 的 fsync 开关，默认 true 保守；并发时可关）
- `super/prompts/debug.md`：**「一次性根因定位」协议**（替代试错）：
  1. **复现**（≥2 次读数，确认稳定/瞬时）
  2. **三假设**：列 3 个最可能原因，每个写「若成立应观察到什么」+ 一条最小验证命令
  3. **逐一验证**（一次跑完三条，禁止边试边改）
  4. **单次修复**：定位到根因后一次性改对 → 复验（gate_check）→ 出结论（fix / keep_original）
  5. 明确禁止：无假设的试错修改、重复跑同一验证、为了过门而改门禁/容差
- `super/prompts/builder.md`：相邻格注入降本——`neighbor_samples` 改为**摘要形态**（description + elite perf + **实验记录 digest 前 400 字** + code_path，不注入代码全文）；如需看代码用 read 自取（省 prompt 上下文）
- `super/superopt/loop.py`：
  - `neighbor_samples` 组装改为摘要（不再内联 `elite code` 内容——若当前实现是路径引用则保持，若内联则改）
  - debug 派发：`max_turns=0` 时不传上限（agents.run 已支持 None/0=不限）
- 测试：`test_loop`（neighbor_samples 为摘要形态且不含代码正文；debug 在 max_turns=0 时不受回合限制——用 fake client 断言 kwargs max_turns 为 None）

# 写域
- super/superopt/config.py
- super/superopt/loop.py
- super/prompts/debug.md
- super/prompts/builder.md
- super/tests/test_loop.py

# 依赖关系
- task49（loop.py 写域串行）

# 测试方法
`python3 -m unittest super.tests.test_loop super.tests.test_prompts -v` 全绿 + 全量无回归

# 禁止事项
- 不得移除 wall/timeout 兜底（只去掉回合上限）；不改 gates
