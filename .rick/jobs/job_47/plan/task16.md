# task16：budget token 上限

## 任务目标
对抗评审发现的真缺陷修复：budget.py 只有墙钟无 token 上限。

## 关键结果
- `super/superopt/budget.py`：CostLedger 增加 total_tokens 累计查询；`within_budget()` 增加检查 `max_total_tokens`（>0 时生效；CostLedger 需持有该配置或由构造传入）；超限返回 False
- `super/superopt/config.py`：DEFAULTS 增 `"max_total_tokens": 0`（0=不限制，注释示例 50_000_000）
- loop.py 的预算退出路径**不改**（已走 within_budget → break → session_ended）——但 session_ended payload 建议补 `budget_exhausted: true` 标注（若 loop 侧需要小改，**留给 task19**；本 task 保证 within_budget 语义正确即可，loop 不动）
- `super/tests/test_budget.py`：token 上限触发/不触发用例

# 写域
- super/superopt/budget.py
- super/superopt/config.py
- super/tests/test_budget.py

# 依赖关系
- 无（L1 首层并行）

# 测试方法
`python3 -m unittest super.tests.test_budget -v` 全绿

# 禁止事项
- 不动 loop.py（归 task19）
