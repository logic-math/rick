# human-loop 模块（internal/handler + templates）

## 职责

`rick human-loop <topic>` 是启发 human 进行原创性思考的入口——SENSE 五阶段思考门禁会话：human 带着主题进入，每阶段经过「orig 概念随机游走调研 → 递归解释呈现 → 概念地图引导表达 → 思考门禁判定」推进，产出思考轨迹（`thought_path.md`，存入 `.rick/draft/loops/loop_N/`，价值维度的个人判断载体）。

## 思考门禁流程（v8.3，job_57 彻底重构）

五阶段（每阶段一个 skill，承担双目标：启发思考内容 + 验证思考能否过门禁）：

| 阶段 | 名称 | 门禁标准（可判定） |
|------|------|--------------------|
| 1 | 问题还原 | 期望/现状/差距三结构化表述说清，价值判断与事实判断分离 |
| 2 | 视角生产 | 描述选择视角的应用价值，能描述其内在因果运动 |
| 3 | 判断矛盾 | 基于视角解释问题，定义决定性/根本性/全局性矛盾及其因果运动 |
| 4 | 辩证逆转 | 符合逻辑地回答：当阻碍 X 是目标 Y 的必要条件时应当如何 |
| 5 | 良质批判 | 回顾思考假设是否可证伪，推导过程逻辑性 |

每阶段循环：读阶段 skill → 派 orig（概念随机游走）→ teacher_polish 润色呈现（递归解释）→ 概念地图收尾引导（human 串联概念、用自己的话表达）→ main agent 内联门禁判定 → 通过推进 / 未过指出缺口重调 orig（无重试上限，推进权在 human）。

## 架构文件（单 agent + 五阶段 skill + 呈现层方法论）

| 文件 | 角色 |
|------|------|
| `sense_loop.md` | main agent 协议（极薄编排层：五阶段门禁循环 + thought_path 写入契约） |
| `skills/sense_reconstruct\|perspective\|contradiction\|reversal\|critique.md` | 五阶段 skill（每阶段：思考目标 + orig 派发契约 + 呈现纪律 + 门禁标准 + 未过行为） |
| `skills/orig.md` | orig agent（概念空间随机游走者；唯一提示词：「给用户返回怎样的内容才能引导他完成本阶段的思考目标？」；跳跃检索连接同中有异/异中有同的概念） |
| `skills/teacher_polish.md` | 呈现层方法论（科普视角；ASD-STE100 短句规范；**递归解释**：零领域专用词汇，每个概念递归以通识内容讲清楚） |
| `skills/sense.md` | SENSE canon（五步权威文本，与 dream 共用） |

orig 经 env 职责 3 落盘为 pi 自定义 agent（`agents/orig.md`）——**单 agent 架构**（v7 的 research/syllabus-builder/lesson-writer/recall-review 四 agent 已删除）。

## 产物结构（v8）

```
.rick/draft/loops/loop_N/
├── thought_path.md   # 唯一产出物：human 原话 + AI 启发的思考轨迹时间线
├── orig/             # orig 调研中间结构（stage{N}_{seq}.md）
├── prompts/          # 启动时渲染快照（模板更新后旧会话不继承，须新 loop 承接）
└── session_id        # --resume loop_N 恢复
```

## 设计要点

- **发散-收敛结构**：概念随机游走 + 递归解释 = 发散（帮 human 建立脑海概念地图）；收尾引导 human 用自己的话表达 = 收敛（过思考门禁）
- **无阈值配置**：门禁无重试上限（推进权在 human）；无 RFC/mission/values 等其他产物
- **v7 教学机制已删除**：ZPD 档案/费曼演绎/教学大纲/课程/recall-review/文件状态机
