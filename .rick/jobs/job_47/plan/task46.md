# task46：token 成本监控 skill + cost_report 工具

# 任务目标
human fix 3+4：token 成本监控作为**一等 skill** 集成进 super loop；通过分析 pi session 确认成本分布；先保证**可解释性与计量**（优化后续）。

# 关键结果
- 新建 `.rick/skills/super_cost_analysis/skill.md`（三段式）：
  - 触发场景：run 收尾 / adjust 决策前（判断是否值得为该格继续烧预算）/ human 要求成本复盘 / 迭代成本异常（单迭代 tokens > 2× 中位）
  - 预期效果：产出成本分布报告（按 role/迭代/cell），识别浪费模式，给优化建议
  - 核心内容：
    1. **数据源**：`events.jsonl` 的 `agent_ended.usage`（input/output/cache_read/cache_write/total/cost_usd）+ `iteration_ended.tokens` + pi session 文件（`<agent_dir>/sessions/<cwd-slug>/*.jsonl` 的 usage 事件，可看单轮 input 膨胀曲线）
    2. **分析维度**：按 role（builder/coding/debug/adjust）× 迭代 × cell 聚合；上下文膨胀（input_tokens 随回合增长）；检索/工具调用占比（tool_calls 数 vs token）；cache 命中率（cache_read/input）
    3. **浪费模式清单**：重复 benchmark 交叉验证（coding 在同一候选上跑多轮 gate_check）、超长上下文（单 agent input > 200k）、无效检索（papers 引用未被 plan 使用）、debug 烧在可自愈的瞬时波动上
    4. **优化处方**：限长（调研报告/相邻格只读摘要）/限次（benchmark 复验上限）/分工（builder 贵档只在关键格、coding 轻档）
- 新建 `super/tools/cost_report.py`（stdlib-only CLI）：
  - `--workdir <ws> [--json]`：读 events.jsonl → 输出 markdown 表：`role × 迭代` 的 tokens/cost/wall、单迭代合计、Top-5 昂贵迭代、按 role 的 token 占比、上下文膨胀曲线（每 agent 的 input_tokens 序列）
  - 若 `<agent_dir>/sessions` 可定位（读 super.json 的 pi_path 上级推 agent_dir 或 env `PI_CODING_AGENT_DIR`），补充 session 级明细；定位失败则仅用事件流（降级）
- `super/superopt/loop.py`：`iteration_ended` payload 增 `by_role`（本迭代各 role 的 {tokens, wall_ms, tool_calls} 聚合）——**成本可解释性**（每迭代可归因到角色）
- 测试：`test_loop`（iteration_ended.by_role 结构）；cost_report 冒烟（fixture 事件 → markdown 表 + --json 输出）

# 写域
- .rick/skills/super_cost_analysis/skill.md（新建）
- super/tools/cost_report.py（新建）
- super/superopt/loop.py（仅 iteration_ended payload 增 by_role）
- super/tests/test_loop.py

# 依赖关系
- 无（L1；与 task44 写域冲突点=loop.py/test_loop.py —— **冲突**！改：本 task 的 loop 改动（by_role）与 task44（admit record_path）都在 loop.py）

# 依赖关系（修正）
- 与 task44 **串行**（同为 loop.py 写域）：改为 L2，依赖 task44

# 测试方法
`python3 -m unittest super.tests.test_loop -v` 全绿 + `python3 super/tools/cost_report.py --workdir <fixture>` 输出表

# 禁止事项
- 不改 gates/prompts；不做成本优化（只计量与解释）
