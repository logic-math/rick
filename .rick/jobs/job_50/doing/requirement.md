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

## v7.2 增量裁决（2026-11-01，human 实跑前反馈二）

- **费曼演绎精简 3 步**（parent 会话内联执行，不再 agent 往返）：① 引导用户用自己的话讲出来 ② 清晰化引导：识别事实模糊或价值模糊的表达并追问 ③ 用户无法回答→补充答案重新教学；并记录 ZPD——模糊点标记薄弱知识领域
- **探索流 skill**（替代 originality-sparker；费曼完成后可选进入，human 决定）：始终识别 human 回答中的原创性信息点并扩展——先引导到某个跨领域差异性概念（内联检索），讲解概念并关联当前讨论主题，形成原创性表达，引导持续回答；澄清纪律：价值清晰、符合事实（无法验证部分清晰识别为假设）；结束标准=询问用户是否可推进下一课，结束权在 human
- **交互协议（费曼/ZPD 档案/探索流）= parent 内联 skill（方法论文件）**；重生型 agent 收缩为 4 个：research / syllabus-builder / lesson-writer / recall-review（移除 zpd-assessor、feynman、originality-sparker 三个 agent）
- **推进权归 human**（探索流结束询问）；ZPD 档案记录薄弱点供下轮 lesson-writer 详略

## v7.4 增量裁决（2026-11-03，human 完整体验 loop_17 后反馈）

- **课程内容=SENSE 还原教学原则**：返回内容只展示作者的 SENSE 思考过程——还原作者掌握的信息（结构化呈现：事实块/证据/价值判断，不散文化）→ 基于这些信息经 SENSE 各步得出结论的过程；细节程度由不断丰富的 ZPD 档案确定；以思考过程引导学生完成思考；过程中捕获学生原创观点并记录（judgment.md）与鼓励
- **探索流重定位**：移出每课子循环；全部课程学习完后，统一通过收集的原创信息点扩展，进行**一次统一探索流**（⑤ 收口阶段）
- **调研**：混沌学园（human 称「混动学园」）讲过「探索流」——基于其方法研究升级方向以激发原创性思考（research-L2）

### v7.4 探索流方法体升级（research-L2 依据：混沌学园探索流）

- 方法内核：三动作「表达→呼应→看见」（MP 神经网络隐喻：呼应=隐藏层，输出>输入=原创性来源）+ 三规则（不戴帽子/零建议零反馈/只做呼应）+ 关键词级呼应点火 + Y 轴可能性纪律 + 话轮平等 + 当下生发（挑此刻仍点燃的点，祥林嫂式回放卡死流）+ 输出>输入收敛判据（连续 2 轮无新火花即停该点）+ 跨域概念=平行面相（替代物理小组）
- 澄清纪律/输出纪律/结束权保留

## v7.5 增量裁决（2026-11-08，human 体验 loop_18 探索流后反馈）

- **废掉混沌学园三动作探索流方法**（回应无法带来启发思考）
- **探索流 = SENSE 状态机驱动的思考推进循环（loop）**：
  1. 检查探索流维护的 SENSE 状态（问题还原确定了吗？视角确定了吗？）→ 判定学生当前 SENSE 思考阶段 → 推进思考进程向前，直到完成一次完整 SENSE 思考
  2. 过程中不断派 research subagent 调研本领域或新领域概念，为学生 SENSE 思考提供正确信息输入；判断学生思考是否达到该阶段标准 → 达标推进
  3. 探索流职责 = 提供**相似异质性概念的详细描述**（启发思考）+ 通过学生回答判定其 SENSE 阶段 + 按阶段选择呈现不同概念，帮助完成该阶段思考
  4. 原创性想法依旧识别并记录（judgment.md，作为后续内容使用）

## v7.6 增量裁决（2026-11-08，human 体验 loop_19 v7.5 探索流后反馈）

- v7.5 仍不行（阶段对但信息泛）——**探索流采用方案 A：靶向卡点调研**
- 循环四步：①进程判定（SENSE 阶段 + 该阶段的具体卡点=可调研的精确问题）②靶向调研（卡点+思考摘要派 research）③张力呈现（最小充分集 1-3 块，每块标注与学生的张力：证实/证伪/扩展他说的哪句话，末尾留引导性问题——产出必须学生自己完成）④推进判定（达标推进/未达标重判卡点再来）
- 按阶段呈现信息类型：问题还原=事实+价值张力对｜构建视角=视角候选｜洞察矛盾=矛盾结构先例｜辩证逆转=逆转先例｜良质批判=批判标准
- 启发机制假设：启发的本质 = 信息与学生自己思考的张力（v7.4 薄/v7.5 泛的失败归因）
