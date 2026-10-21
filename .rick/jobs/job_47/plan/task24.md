# task24：三模板升级（占位符 + 饱和/认输动作）

## 任务目标
ctx 键已由 task22/23 提供 → 模板侧消灭硬编码、把饱和信号与认输动作写进纪律。ctx 契约（task23 提供）：`env_dir`、`perf_profile_cmd`、`saturation`。

## 关键结果
- `super/prompts/builder.md`：
  - 「可用工具」段改为**占位符**：`{{perf_profile_cmd}}`、`{{self_check_cmd}}` 若为空则不展示；领域知识用 `{{env_dir}}/domain_knowledge.md`——**禁止硬编码绝对路径**（写明纪律：模板必须 env 可移植）
  - 「输入材料」增：饱和信号 `{{saturation}}`——**收到非空饱和信号时**：优先设计「跳出当前方向」的实验（跨族组合/换轴灵感/引用 domain_knowledge 的未覆盖做法），而不是继续在当前族内微调；并在 idea 里写明「已知饱和，本实验试图打破」
- `super/prompts/adjust.md`：
  - 输入材料增 `saturation`
  - **饱和处置纪律**：收到非空 saturation → 首选 `expand_cell`（按 schema 提新方向）；若判断整个特征空间已榨干（冠军 perf 已触及 domain 天花板/所有存活格停滞）→ 提 `terminate_recommend`（`{type:"terminate_recommend", evidence, rationale}`，供人类决策收手/换轴）
  - 动作清单增 `terminate_recommend`（建议性，非执行动作，不进裁决自动执行）
- `super/prompts/debug.md`：工具命令占位化（`{{perf_profile_cmd}}` 可选段落）

# 写域
- super/prompts/builder.md
- super/prompts/adjust.md
- super/prompts/debug.md

# 依赖关系
- task22（键名契约已定；本 task 只写模板）

# 测试方法
`grep -c "/Users/" super/prompts/*.md` 为 0（无硬编码绝对路径）；`grep "saturation\|terminate_recommend" super/prompts/{builder,adjust}.md` 命中

# 禁止事项
- 不改 env_8 的 env/prompts（用户 env 资产）；不动框架代码
