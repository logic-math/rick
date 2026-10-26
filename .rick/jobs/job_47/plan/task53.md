# task53：文档更新（并行事件/并发实验/成本策略）

# 关键结果
- `.rick/loops/super-loop.md`：事件段更新——**实验唯一标记 (cell_id, iteration_id) → experiment_id**；事件流支持并行写（O_APPEND）；还原单实验用 `grep '"experiment_id":"<cell>#iter<N>"'`；Step 3 监控增 `read_experiment` 用法；并发实验说明（max_parallel_experiments 默认 2、格锁、异格并行）
- `.rick/skills/super_cost_analysis/skill.md`：增「debug 不截断后的一次性根因」作为成本优化处方之一（截断→重试是浪费主因；一次性定位优先）；`event_fsync` 开关说明
- `.rick/skills/super_debug_protocol/skill.md`：增「一次性根因定位」协议摘要（复现→三假设→一次验证→单次修复）
- `.rick/skills/gen_env_skill/skill.md`：并发约束段更新（默认 2、事件按实验标记）

# 写域
- .rick/loops/super-loop.md
- .rick/skills/super_cost_analysis/skill.md
- .rick/skills/super_debug_protocol/skill.md
- .rick/skills/gen_env_skill/skill.md

# 依赖关系
- task49/50/51（机制落地）

# 测试方法
grep 含「experiment_id」「并行」「一次性根因」「read_experiment」

# 禁止事项
- 不改代码
