# human-loop - 人机协作深度思考模块

## 模块职责

`human-loop` 模块是 Rick CLI 的人机协作深度思考工具,通过 SENSE 方法论引导 human 进行结构化深度思考,产出存入 `.rick/draft/rfc/` 目录。

**核心职责**:
- 启动 5 阶段人机协作思考流程(S/E/N/S-R/EC)
- 派发 4 个 subagent(sense_loop/think/research/exporter)
- 嵌入批判门禁到各阶段
- 支持反向回流(后续阶段可重启前序阶段)
- 产出 RFC 文档(基于 human 确认的判断)

**版本**: v2.11.9 (2026-08-04 重构)

---

## 命令

```bash
rick human-loop <topic>
```

- `topic`:思考主题(必传)
- `--dry-run`:只输出 prompt,不调用 Claude
- `-v / --verbose`:显示详细信息

---

## 四文件架构(go embed 内嵌模板)

| 文件 | 角色 | 职责 |
|---|---|---|
| `templates/sense_loop.md` | main agent 协议 | 5 阶段调度+批判门禁嵌入+反向回流 |
| `templates/think.md` | subagent | 推理识别→假设提取→形式化→4 维打分→3 启发性问题 |
| `templates/research.md` | subagent | 尽调树+信源加权+subagent 上下文隔离 |
| `templates/exporter.md` | subagent | RFC 输出(大纲+内容两阶段) |

**对应 skill 文件**(方法论):
- `templates/skills/sense.md` - SENSE 方法论
- `templates/skills/think.md` - 推理过程分析+4 维风险评估方法论
- `templates/skills/research.md` - 尽调树+信源加权方法论
- `templates/skills/exporter.md` - RFC 输出方法论

---

## 5 阶段流程

```
S ⇄ E ⇄ N ⇄ S-R ⇄ EC
↑                    ↑
└── 跃迁/反向回流 ──┘
```

| 阶段 | 名称 | 核心动作 | human 给出 |
|---|---|---|---|
| S | 问题确认 | research 调研现状+对 human 假设追问 | 现状/期望/差距 |
| E | 视角生成 | research 跨领域调研→多视角候选 | **原创视角** |
| N1 | 矛盾生成 | 用系统论描述符描述系统 | 对系统矛盾状态的理解 |
| N2 | 主要矛盾判断 | think 三维打分(根本性/全局性/决定性) | **主要矛盾** |
| S-R | 辩证逆转 | "若 X 必然,实现 Y 应当如何?" | 逆转逻辑判断 |
| EC | 良知批判 | sense_loop 呈现回顾,**human 自判** | 跃迁方向(降维/升维/维持) |

**关键约束**:
- N 阶段必须 N1+N2 双追问,缺一不可
- N2 无主要矛盾 ⇒ 必须触发 S-R(硬约束)
- EC 阶段无 subagent,human 自判良质
- 反向回流上限 `sense_max_backflows`(默认 3 次)

---

## 核心概念

### 1. 尽调树(research)

```
根节点(主题)
├─ 子节点(MECE 划分)
│  ├─ 叶节点(已澄清,置信度≥0.8)
│  └─ 叶节点(无法澄清,R7 上报)
└─ ...
```

- 每层 MECE(不重复不遗漏)
- 节点置信度 < 高且有疑问 → 下钻
- 终止:所有叶节点置信度 = 高

### 2. 信源加权(research)

| 信源 | 默认权重 | 验证方式 |
|---|---|---|
| 代码原文 | 0.4 | Read/Grep |
| 运行时行为 | 0.3 | Bash 跑命令/测试 |
| 文档 | 0.2 | WebFetch/Read |
| 反事实 | 0.1 | 修改代码后还原 |

置信度 = Σ(信源验证结果 × 权重)。高 ≥ 0.8(终止)/ 中 0.5-0.8 / 低 < 0.5(R7 上报)。

### 3. 推理类型识别(think)

| 推理类型 | 形式 |
|---|---|
| 演绎(Deductive) | 前提 A + 前提 B → 结论 C |
| 归纳(Inductive) | 观察 n 案例 → 总结规律 |
| 溯因(Abductive) | 现象 X → 反推解释 Y |

置信度先验:演绎 0.9 / 归纳 0.6 / 溯因 0.4。

### 4. 4 维度打分(think)

| 维度 | 取值 |
|---|---|
| 影响范围 | (决定性+根本性+全局性)/3,每个 1.0/0.5 |
| 不可逆性(下行) | 1.0 不可恢复 / 0.5 可修复 |
| 影响程度(上行) | 1.0 高赔率 / 0.5 低赔率 |
| 置信度 | 推理类型先验,可被 research 更新 |

**期望值公式**:
```
期望分 = (影响程度 × 置信度) - (不可逆性 × (1 - 置信度))
最终分 = 期望分 × 影响范围分
```

### 5. 假设数量保障 + 3 启发性问题(think)

- **最低假设数**:`think_min_assumptions`(默认 5)
- **多视角强制**:3 类推理各至少 1 个 + 交叉类 1 个
- **补强流程**:低于 5 则反事实/边缘/隐含假设迭代 2 轮

**每假设 3 启发性问题**:

| 问 | 提问 | 启发维度 |
|---|---|---|
| Q1 信念 | 关于 [Y],你内心最确信的是什么?最不确定的是什么? | 自省内心状态 |
| Q2 前提 | [Y] 成立需要什么前提?这些前提你确认过吗? | 追溯隐含前提 |
| Q3 反例 | 什么证据会让你改变对 [Y] 的判断? | 反向思考 |

总提问数 ≥ 5 × 3 = 15 问题(默认配置)。

### 6. 系统论描述符(N1 阶段)

| 要素 | 含义 |
|---|---|
| node | 系统组件 |
| input | 系统输入(符号,物质/信息/能量) |
| output | 系统输出(同上) |
| inner | 系统内部协作的 input/output |
| edge | node 之间的协作关系,承载 inner_input/inner_output |

### 7. 批判门禁嵌入

- think 不再是独立步骤,而是嵌入各阶段
- 每阶段 human 实质性回答后触发(纯确认语句跳过)
- 门禁未通过达 `max_retries`(默认 5)次升级 human 介入

---

## 配置项

`.rick/config.json` 的 `human_loop` 字段:

```json
{
  "human_loop": {
    "max_retries": 5,
    "sense_max_backflows": 3,
    "think_top_n": 3,
    "think_min_assumptions": 5,
    "research_source_weights": {
      "代码原文": 0.4,
      "运行时行为": 0.3,
      "文档": 0.2,
      "反事实": 0.1
    }
  }
}
```

| 字段 | 默认值 | 含义 |
|---|---|---|
| `max_retries` | 5 | 批判门禁未通过最大重试次数 |
| `sense_max_backflows` | 3 | 反向回流上限 |
| `think_top_n` | 3 | top-N 假设数 |
| `think_min_assumptions` | 5 | 假设最低数量 |
| `research_source_weights` | 0.4/0.3/0.2/0.1 | 信源权重(和应=1.0) |

---

## 产物

| 路径 | 内容 |
|---|---|
| `{{loop_dir}}/prompts/` | 4 个 prompt + 4 个 skill 文件 |
| `{{loop_dir}}/briefs/` | 各阶段简报 + research-report.md + research-{N}-{节点}.md |
| `{{loop_dir}}/judgment.md` | human 判断原话(禁止 AI 推理) |
| `{{rfc_dir}}/rfc-[主题]-[日期].md` | 最终 RFC 输出 |

---

## Go 代码结构

| 文件 | 职责 |
|---|---|
| `internal/cmd/human_loop.go` | 命令入口,调用 GenerateHumanLoopPromptFile |
| `internal/prompt/human_loop_prompt.go` | 生成 4 个 prompt 文件,注入变量 |
| `internal/prompt/manager.go` | embed.FS 加载模板 |
| `internal/config/config.go` | HumanLoopConfig 结构体 |
| `internal/config/loader.go` | GetDefaultConfig 设默认值 |

---

## 关键设计决策

1. **5 阶段非线性**:替代 v2 7 步线性,允许反向回流
2. **批判门禁嵌入**:think 不再独立步骤,嵌入各阶段
3. **EC human 自判**:不替 human 提议跃迁方向
4. **简化产物**:2 产物(briefs+judgment),删除 loops.md/progress.md
5. **配置化**:所有阈值可配置(重试/回流/top-N/min/权重)
6. **启发性提问**:禁止确认性句式,引导 human 深度思考
7. **假设数量保障**:min_assumptions + 多视角强制 + 补强流程

---

## 相关文档

- [architecture.md](../architecture.md) - 整体架构
- [modules/cmd.md](cmd.md) - 命令处理器
- [modules/prompt.md](prompt.md) - 提示词管理
- [modules/config.md](config.md) - 配置系统
