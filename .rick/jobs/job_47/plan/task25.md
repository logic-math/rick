# task25：gen_env skill + super-loop（自改进可移植性与饱和解读）

## 任务目标
把本轮机制写进知识层，使自改进循环不退化成「硬编码补丁」。

## 关键结果
- `.rick/skills/gen_env_skill/skill.md` 第 7 节（轨迹分析→提示词进化）增：
  - **可移植性纪律**：env/prompts/*.md 中的路径必须用占位符（`{{env_dir}}`/`{{perf_profile_cmd}}`/`{{self_check_cmd}}`），**禁止硬编码绝对路径**（env 复制/迁移即失效；env_8 实测教训）
  - **PROMPTS_LOG 检查项**增「可移植性检查：无绝对路径」
  - **饱和/认输解读**：轨迹分析发现「长时间无新纪录 + adjust 未提 expand_cell」→ 检查 saturation 信号是否正常注入；发现 `terminate_recommend` 提案 → 与 human 讨论收手/换轴
- `.rick/loops/super-loop.md` Step 7 增：饱和信号与 terminate_recommend 的处置指引（人收到建议后如何决策：换轴走 gen_env 轴改版 / 收手归档）

# 写域
- .rick/skills/gen_env_skill/skill.md
- .rick/loops/super-loop.md

# 依赖关系
- task23/24（机制与模板落地）

# 测试方法
关键词断言：gen_env skill 含「可移植性」「占位符」「terminate_recommend」；super-loop Step 7 含「饱和」

# 禁止事项
- 不改框架/模板代码
