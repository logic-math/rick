优化 human  loop  ，使其能够实现，启发人类原创性思考 的最终目标。
从Users/logic/workdir/AI_CODING/rick/.rick/draft/loops/loop_12/ADR-human-loop-v7.md
这个 ADR 开始
---

## Grilling 澄清结论（2026-10-31，门禁已通过）

**核心目标（human 改述）**：启发人类进行原创思考。

**裁决汇总**（16 问全裁决，详见 doing/grilling/design-tree.md）：
- 架构（Q1）：Go 内嵌模板方案保持不变——v7 = `internal/prompt/templates/sense_loop.md` 原地重写 + 新 skill 模板；`rick human-loop` 命令/渲染管线/目录分配/session/resume 全不动
- Agent 化（Q2）：六 skill 全部升级为独立子代理（zpd-assessor / syllabus-builder / lesson-writer / originality-sparker / recall-review + 保留 research；mission 例外=parent 直接落盘）；teach agent 下线；经 `rickAgents` 清单部署
- 核心流程：①调研与呈现（research agent 调事实 + lesson-writer 按 ZPD 详略呈现，先判图/文再选表示）→ ②费曼学习与记录（human 反讲→zpd-assessor 判对错→zpd.md 学习知识记录=差哪/不差哪→下次详略）→ ③原创性启发（sparker agent 返回部分违背非否定的差异化知识→跨域融合）
- 机制细则：anchors.md 跨 loop 探针（关键词+卡点+未解矛盾）｜三次同类失败即删（信度）｜二次探针误差吸收（线索→换措辞，两轮不过=弱）｜「想不起来=无能力」仅跨 loop｜讲出来=验收动作｜强/中/弱=自然语言 rubric（错误断言+关键概念覆盖）｜冷启动=mission 起点自报+2-3 领域探针｜篇幅=ZPD 驱动详略（无固定 KB 上限）｜文本绘图先判图/文、限 code-fence 简单结构
- SENSE 升级（Q13）：第一步 事实还原→**问题还原**（还原事实信息与价值判断；分清期望与现状）；后四步不动
- 验收（Q12/KR5）：human 自选主题实跑一次完整五步会话，主动判断
- 调研依据：research-L1.md（生产级 LLM 导师操作化：AutoTutor 判级/提示梯度、teach-back 二不过、ALEKS/MemGPT 锚形态、ReConcile 异构共识、ASCII CJK 陷阱）+ ADR 引用的 research_1–4；R7 残留项 doing 开工前网验

**实现范围（叶子层四维度已落实）**：改 sense_loop.md / skills/sense.md（问题还原）；新建 5 个 skill 模板；删 teach 模板×2 + research task 模板；改 agents.go（rickAgents）/check.go/manager.go/human_loop_prompt.go/pibuilder.go；更新 4 组测试；go build+test；部署 update-pi；dry-run 冒烟；human 实跑验收。

## v7 增量裁决（2026-11-01，human 实跑前反馈）

- **② ZPD 生成重写**：不要突兀地问 human 价值判断；流程=research 调研（领域核心概念）→ zpd-assessor 选 3–5 个回忆点/关键概念 → 对每个概念跑费曼演绎（human 讲解→挑战模糊/事实不符/价值判断错误处→逼问修正→最简话语呈现）→ 按讲解与事实符合性推断 ZPD → 生成 **ZPD 档案**（记录会什么/不会什么，随 loop 展开逐次完善）
- **ZPD 与费曼演绎抽为可复用 skill**：feynman=新 agent（费曼技巧执行者：挑战逼问+判定）；zpd-assessor 演化（select 选点 + consolidate 档案管理，去掉原 probe 回忆写出/assess 费曼判断）
- **① 学习目标对齐**：结构化提问即可（固定字段一次成表，非开放式面试）
- **④ 费曼环节**复用 feynman skill 保持一致
