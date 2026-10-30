# task25：gen_env skill + super-loop（可移植性/饱和/台账/gate→debug 解读）

## 任务目标
把本轮新增的信息层与修复通道写进知识层，让中控会话的自改进循环能用上。

## 关键结果
- `.rick/skills/gen_env_skill/skill.md` 第 7 节增：
  - **可移植性纪律**：env/prompts/*.md 路径必须用占位符（`{{env_dir}}`/`{{perf_profile_cmd}}`/`{{self_check_cmd}}`），禁硬编码绝对路径（env_8 实测教训：复制/迁移即失效）；PROMPTS_LOG 检查项增「可移植性检查：无绝对路径」
  - **可用 ctx 字段清单**（写进第 7 节）：四角色各自可用的新信息（domain_brief/perf_timeline/budget_status/ledger_brief/gate_defs/gate_failure/own_decisions/saturation）——改 prompt 时优先用它们，而不是让 agent 自己 read 文件
  - **饱和/认输解读**：轨迹分析发现长时间无新纪录 → 查 saturation 是否注入；发现 terminate_recommend → 与 human 讨论收手或走 gen_env 轴改版
  - **台账解读**：`ledger.jsonl` 是跨迭代实验档案（idea→实现→门禁→结论），分析坏模式时优先读它
- `.rick/loops/super-loop.md` Step 7 增：饱和信号处置（expand_cell 优先/terminate_recommend 与 human 决策）+ gate→debug 通道说明（框架门禁失败会自动进 debug 修复 1 次）

# 写域
- .rick/skills/gen_env_skill/skill.md
- .rick/loops/super-loop.md

# 依赖关系
- task23/24（机制与模板落地）

# 测试方法
关键词断言：gen_env skill 含「可移植性」「占位符」「terminate_recommend」「ledger」；super-loop 含「饱和」「gate→debug」或「框架门禁失败」

# 禁止事项
- 不改框架/模板代码
