# task21：自改进文档层（gen_env skill 进化流程 + super-loop SOP）

## 任务目标
J-i3-4/J-i3-5：《自改进的进化搜索算法》外层循环落进知识资产——四角色 prompt 可进化 + 轨迹→prompt 进化流程。

## 关键结果
- `.rick/skills/gen_env_skill/skill.md` 增「轨迹分析 → 提示词进化」环节（env 升级流程内）：
  1. 取证：跑 super/tools/adjust_trace.py → 决策-效果对照表
  2. 诊断坏模式清单：震荡/过激/迟钝/漏杀/探索不足（expand_cell 从未发生）
  3. 处方：改写 env/prompts/<role>.md（四角色均适用：adjust/builder/coding/debug）
  4. 版本管理约定：**延迟具体化**（默认继承仓内模板；需定制/进化时才落 env/prompts/）；旧版存 env/prompts/_history/<role>.{ts}.md；PROMPTS_LOG.md 记「版本+改动+证据链接」
  5. 下轮验证回环：改善固化/恶化回滚
  6. 模型分配引导：builder=贵档（计划质量优先）/coding=可配轻档（agent_models 示例）
- `.rick/loops/super-loop.md` 增外层自改进 SOP 步骤（run 收尾→轨迹复盘→prompt 迭代决策→重跑），引用 gen_env skill 该环节
- 仓级知识（super-loop/gen_env skill 自身）的进化路径注明：走 rick 既有 learning/dream 循环（两层接通点）

# 写域
- .rick/skills/gen_env_skill/skill.md
- .rick/loops/super-loop.md

# 依赖关系
- task19（机制落地——文档描述与实现一致）；与 task20 并行

# 测试方法
关键词断言：gen_env skill 含「轨迹」「PROMPTS_LOG」「_history」「四角色」；super-loop 含「自改进」「轨迹复盘」

# 禁止事项
- 不动 super/prompts/ 仓内默认模板（task18 域）；不动框架代码
