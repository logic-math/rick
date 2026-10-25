# task36：仓级 super_* skill 资产五件

# 任务目标
四 agent loop 所需的 skill 资产（KR7）：各自三段式（触发场景/预期效果/核心内容），方法论主体**引用 rick canon**（sense / debug-skill / tdd-zh），只写 super 场景适配。

# 关键结果
新建五件（`.rick/skills/<name>/skill.md`，格式对齐既有 skills 如 fake_binary_script_skill）：
1. `super_experiment_analysis`（adjust）：读区间内全部 `experiment_record.md` + 按需读 agent session（`~/.rick/pi/agent/sessions/...` 或 dev-home 路径提示）→ 输出「区间实验全景摘要」（每实验的假设/结果/perf delta/失败阶段）+ 主要矛盾候选
2. `super_sense`（adjust）：引用 `.rick` 内 rick sense canon（`internal/prompt/templates/skills/sense.md` 的方法论要点摘要 + 指向），场景适配=「从实验全景中判定**主要矛盾**（当前性能瓶颈的全局性/根本性成因）→ 辩证逆转得到突破方向 → 良质批判自检」
3. `super_cell_management`（adjust）：格动作手续——`new_cell`（必须给 name/method/source_refs/正交性论证：新维度未被现有格覆盖）、`modify_cell`（改 method 描述，可引用论文/调研地址）、权重调控（0=删除语义）；并说明与 feature_space 热重载/archive 的落盘路径
4. `super_experiment_design`（builder）：从调研报告设计**可复现实验**，产出实验记录六段（背景与价值/实验假设/验证逻辑/具体步骤/观测现象/执行记录占位）；纪律=一迭代一实验、可证伪、指标可观测
5. `super_debug_protocol`（debug）：复现 → research（该域关键组件**用户手册**：确保工具正确使用/参数最佳）→ 提出**三个最可能失败原因**逐一验证（各自明确「若成立应观察到什么」）→ 修复或维持；引用 rick debug-skill canon（无根因不修）

# 写域
- .rick/skills/super_experiment_analysis/skill.md（新建）
- .rick/skills/super_sense/skill.md（新建）
- .rick/skills/super_cell_management/skill.md（新建）
- .rick/skills/super_experiment_design/skill.md（新建）
- .rick/skills/super_debug_protocol/skill.md（新建）

# 依赖关系
- 无（L1，与 task37 并行）

# 测试方法
五件存在；每件含「触发场景/预期效果/核心内容」三段；`grep` 含关键语义（主要矛盾/正交性/实验假设/三个最可能/六段）

# 禁止事项
- 不复制 rick canon 全文（引用路径 + 要点摘要）；不改代码
