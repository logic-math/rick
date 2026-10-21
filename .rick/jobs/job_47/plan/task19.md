# task19：loop 块（expand_cell 链 + 热重载 + 决策快照 + 事件）

## 任务目标
J-i3-2 + 可观测性地基：expand_cell 应用链（提案→裁决→生效）、feature_space 热重载、adjust 决策快照按迭代归档、adjust_decision 结构化事件。

## 关键结果
- `super/superopt/loop.py`：
  1. `_apply_proposal` 支持 `expand_cell` 类型：不直改 feature_space——写 proposals/ 队列文件（走既有硬动作裁决链，proposal 事件）
  2. **裁决决策消费**（_drain_decisions 或等价处）：读到 approve 的 expand_cell decision → 应用到 `ws/env/feature_space.json`（add_axis_value：向 axes[].values 追加值并落盘）
  3. **feature_space 热重载**：每迭代边界（_loop 头部）重读 feature_space.json 的 mtime 变化则重建 self.fs（validate_orthogonal）并刷新 sampler 的 fs 引用（Sampler 构造注入的 fs 需更新——或 sampler 改为持 getter；以最小改动为准）；新格自然进入分殖优先
  4. **adjust 决策快照按迭代归档**：`codes/adjust/iter_{N}/`（prompt_adjust.md + agent_adjust.json 各一份，不再覆盖写）
  5. **adjust_decision 事件**：_collect_adjust 消费时写一条结构化事件（payload：iteration、proposals 摘要、cells_status 快照摘要）
  6. `cell_sampled` 事件字段适配：`ucb_score` → `weight_share`（与 task15 对齐）
  7. session_ended payload 补 `budget_exhausted` 标注（配合 task16 的 within_budget）
- `super/superopt/adjudicator.py`：expand_cell 裁决规则（add_axis_value：轴存在性/值不重复校验→approve；带 regression 证据要求与否按既有硬动作风格定，宽松于修门——加格是扩张性动作风险低）
- 测试适配：`super/tests/test_loop.py`（expand_cell 链：提案→decision 文件→热重载→新格被分殖采到；快照 iter_N 目录存在；adjust_decision 事件断言；weight_share 字段）+ `super/tests/test_adjudicator.py`（expand_cell 裁决用例）
- Python 3.9 兼容

# 写域
- super/superopt/loop.py
- super/superopt/adjudicator.py
- super/tests/test_loop.py
- super/tests/test_adjudicator.py

# 依赖关系
- task15（采样语义与字段名）、task18（提案 schema）

# 测试方法
`python3 -m unittest super.tests.test_loop super.tests.test_adjudicator -v` 全绿 + `python3 -m unittest discover -s super/tests -p "test_*.py"` 全量无回归

# 禁止事项
- 不动 sampler.py/budget.py；不动 e2e_test.py（归 task20）
