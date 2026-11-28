彻底重构 human-loop
使其能够启发原创思考，创造性的解决问题。
---

## Grilling 澄清结论（2026-11-25，完整过程见 grilling/design-tree.md）

**核心目标（O）**：human-loop 彻底重写为极简的「原创性学习-思考-表达会话」——现有全部逻辑（教学机制/SENSE 内嵌/文件状态机/历史兼容）删除；human 通过学习提升自己。

**v8 范式（human 裁决定型）**：
- 基于 SENSE 五阶段（问题还原→视角生产→判断矛盾→辩证逆转→良质批判）的思考门禁制 loop：每阶段 = 一个 skill（启发 + 验证双目标），通过门禁才进下一阶段
- 每个阶段 skill 由 main agent 内联执行；门禁判断由 main agent 负责（门禁标准 = sense.md 可操作定义）
- 一个 orig agent（唯一提示词：「给用户返回怎样的内容才能引导他完成本阶段的思考目标？」）基于思考进度调研、返回启发内容（不是答案）；中间结构落盘 `{loop_dir}/orig/stage{N}_{seq}.md`
- 未过门禁：AI 一句话指出缺口 + 重调 orig 再启发，无硬性重试上限，推进权在 human（自主性最重要）
- 唯一产出物：思考轨迹 `loop_N/thought_path.md`（human 原话 + AI 启发的时间线追加式）

**删除清单**：教学六 skills（zpd/feynman/exploration_stream/syllabus_builder/lesson_writer/recall_review）+ research skill + 四旧 agent（research/syllabus-builder/lesson-writer/recall-review）+ sense_loop.md 旧内容；保留 sense.md（dream 共用 + v8 骨架）；历史 loop 目录不删

**新建清单**：五个阶段 skill（sense_reconstruct/perspective/contradiction/reversal/critique.md）+ orig.md + sense_loop.md 极薄编排层重写；agents.go 单 orig agent；代码改动 handler/builder/测试

**执行约束**：dev 分支实现 → human 实跑测试 → 通过后合并

**调研依据**：grilling/research-L1.md（24 条结论：认知机制/AI 先例/CPS 映射）+ research-L1-r2.md（24 条结论：skill 生态先例/writing-to-think/生成性学习）+ values.md（v1-v7 价值裁决记录）
