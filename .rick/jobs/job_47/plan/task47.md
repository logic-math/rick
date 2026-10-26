# task47：文档更新（五要素/并发约束/成本 skill）

# 关键结果
- `.rick/skills/gen_env_skill/skill.md`：
  - 特征空间节增「五要素」说明（name/method/weight/score/**code_path**——当前最佳精英代码，由 admit 同步；关联实验记录）
  - 增「并发约束」段：一个格同时只有一个实验；不同格可并行（`max_parallel_experiments`）；格锁 TTL
  - 增「成本监控」段：run 收尾用 `super_cost_analysis` skill + `super/tools/cost_report.py` 复盘；`iteration_ended.by_role` 是归因口径
- `.rick/loops/super-loop.md`：Step 7 增「成本复盘」（cost_report + 浪费模式清单 → 优化处方）；Step 3 监控增 token/cost 分布
- `.rick/skills/super_cost_analysis/skill.md` 已在 task46（本 task 只交叉引用）

# 写域
- .rick/skills/gen_env_skill/skill.md
- .rick/loops/super-loop.md

# 依赖关系
- task44/45/46（机制落地）

# 测试方法
grep 含「五要素」「code_path」「并发」「cost_report」

# 禁止事项
- 不改代码
