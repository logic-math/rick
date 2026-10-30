# task41：super-loop 更新（四 agent loop + skill 加载规范 + 实验记录恒常上下文）

# 关键结果
`.rick/loops/super-loop.md`：
- 架构段更新：四 agent loop 图（adjust 四步 / builder 调研+设计 / coding TDD / debug 复验）+ 触发时机表
- 新增「skill 加载规范」：agent 提示词声明 skill 路径+触发条件、按需 read（不内联/不自动）；列出 super_* 五件 + 复用 rick canon（sense/tdd/debug-skill）
- 新增「实验记录 = 恒常上下文」：codes/iter{N}/experiment_record.md 由 builder 建、coding/debug 补；adjust 全局分析读区间全部
- 新增「决策记录」：decisions/iter{N}.md（adjust 思考全过程；中控会话轨迹复盘必读）
- Step 3/7 证据源更新（decisions/ + experiment_record 全文 + research_report）

# 写域
- .rick/loops/super-loop.md

# 依赖关系
- task39/40（机制与模板落地）

# 测试方法
grep 含「四 agent loop」「skill 加载规范」「实验记录」「决策记录」「TDD」

# 禁止事项
- 不改代码/模板
