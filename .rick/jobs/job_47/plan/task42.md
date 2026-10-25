# task42：测试与 e2e 适配（新 ctx / 实验记录 / 三类 debug 触发）

# 关键结果
- `super/tests/e2e_test.py` 适配：新 ctx 键（experiment_records/neighbor_samples/target_record 等）不破坏断言；mock 场景补「builder 写实验记录 + coding 写测试文件」的最小产物（使 record/tdd 校验为 true）；debug 三类触发在 mock 下各走一次
- `super/tests/mock_pi.py`：场景扩展（builder 场景写 experiment_record.md/research_report.md + experiment_plan 含 hypothesis；coding 场景写 test_solution.py；debug 场景写 fix_report + three_hypotheses）
- 全量回归：`python3 -m unittest discover -s super/tests -p 'test_*.py'` 全绿；`bash super/tests/run_all.sh` 绿

# 写域
- super/tests/e2e_test.py
- super/tests/mock_pi.py

# 依赖关系
- task39（新 ctx/触发）

# 测试方法
run_all.sh 全绿（含 e2e 双回归）

# 禁止事项
- 不改框架逻辑（发现框架 bug → 报告 parent）
