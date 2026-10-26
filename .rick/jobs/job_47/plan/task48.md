# task48：测试与 e2e 适配（五要素/并发/成本前缀）

# 关键结果
- `super/tests/e2e_test.py`：适配五要素断言（archive 格含 elite_code_path/record_path）；并发默认 1 回归不变；`iteration_ended.by_role` 断言；cost_report 在 e2e 产物上冒烟
- `super/tests/mock_pi.py`：如需（并发场景可用同一 ok 场景，按 slot 区分 iteration_id）
- 全量：`bash super/tests/run_all.sh` 绿

# 写域
- super/tests/e2e_test.py
- super/tests/mock_pi.py

# 依赖关系
- task45（并发/前缀）

# 测试方法
run_all.sh 全绿

# 禁止事项
- 不改框架逻辑（发现 bug 报告 parent）
