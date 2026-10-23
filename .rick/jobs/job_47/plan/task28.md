# task28：loop.py 三要素注入 + profile 历史 + builder 决策落盘

## 任务目标
三要素（profile/benchmark/verify/gate_check）注入 b/c/d 三 agent；每迭代自动采集 profile 形成历史轨迹并注入 builder；builder 决策（论文/灵感/推理/计划）落盘供中控会话读。

## 关键结果
- **三要素命令注入**（新 ctx 键，全部角色）：
  - `profile_cmd`（`<python> <env>/skills/profile.py --candidate <path>`）、`benchmark_cmd`、`verify_cmd`、`gate_check_cmd`（含 `--baseline` 用法说明）
  - 注入到 **builder / coding / debug**（adjust 可选给 gate_check_cmd 供其判断空间是否榨干）
  - legacy env（无 skills/ 四件）→ 命令为空串，模板侧「未提供则跳过」兜底
- **profile 自动采集与归档**：
  - 每迭代 builder 之前：对**父本候选**（或 env/code/naive_start.py）跑 `profile_cmd` → 存 `<ws>/profile_history/<cell_id>/iter{N}.json`（payload: `{ts, iteration_id, cell_id, candidate, profile}`）；失败静默（profile 是辅助事实，不阻断迭代）
  - `_profile_history(cell_id, n=3)`：读某格最近 n 条 → 注入 builder ctx 的 `profile_history`
  - builder ctx 增：`target_profile_history`（目标格）、`parent_profile_history`（父格）、`inspiration_profile_history`（近邻灵感格各取最近 1-2 条）
- **builder 决策落盘**：
  - 解析 builder 输出新增字段：`papers: [{arxiv_id, title, insight, applicability}]`、`inspirations: [{source_cell, what}]`、`reasoning: str`
  - 落盘 `<ws>/designs/iter{N}_{cell_id}.json`：`{iteration_id, cell_id, ts, profile_facts(摘要), papers, inspirations, reasoning, experiment_plan}`
  - ctx 增 `designs_dir`（告知落盘目录）+ `recent_designs`（近 3 条决策摘要，供避免重复方向）
- **测试**（test_loop.py 扩展）：①三要素命令注入到 builder/coding/debug 的 prompt（fixture env 造 skills/ 四件）②legacy env 命令为空 + 模板不炸 ③profile 采集归档（iter_N.json 存在且字段齐）④profile_history 注入 target/parent/inspiration ⑤designs/iter{N}_{cell}.json 落盘含 papers/inspirations/reasoning ⑥recent_designs 注入

# 写域
- super/superopt/loop.py
- super/tests/test_loop.py

# 依赖关系
- task27（gate_check 契约；loop 侧仅传路径，弱依赖）、task30（三要素契约定义）

# 测试方法
`python3 -m unittest super.tests.test_loop -v` 全绿 + 全量无回归

# 禁止事项
- 不动 gates.py（task27）；不动 super/prompts（task29）
