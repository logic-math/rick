# task52：测试/e2e 适配 + 真实成本对比验证

# 关键结果
- `super/tests/e2e_test.py`：适配 experiment_id/并行事件（还原断言用 read_experiment）；并发 2 槽 e2e 一例（不同格并行 + 事件按实验可还原）
- `super/tests/mock_pi.py`：如需（debug 不限回合场景用 many_turns 但 MOCK_TURNS 小）
- **真实成本对比**（在 env_9 上跑 2 迭代，max_parallel 默认 2）：记录 by_role tokens 并与迭代9 基线（26.11M/12迭代 ≈ 2.18M/迭代）对比，写进回执与 rsi 证据
- 全量：`bash super/tests/run_all.sh` 绿

# 写域
- super/tests/e2e_test.py
- super/tests/mock_pi.py
- .rick/jobs/job_47/doing/rsi/dev-iterations.md（追加成本对比记录）

# 依赖关系
- task50（并发）

# 测试方法
run_all.sh 全绿 + 成本对比数据落盘

# 禁止事项
- 不改框架逻辑（发现 bug 报告 parent）
