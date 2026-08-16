# human-loop 模块（internal/handler + templates）

## 职责

`rick human-loop <topic>` 是人类介入深度思考的入口，用 SENSE 方法论引导对复杂问题进行结构化分析，产出存入 `.rick/draft/`（价值维度的个人判断载体）。

## SENSE 方法论（5 阶段 + 反向回流）

| 阶段 | 名称 | 核心动作 |
|------|------|----------|
| S | 问题确认 | research 调研现状 + 对 human 假设追问 |
| E | 视角生成 | research 跨领域调研 → 多视角候选 → human 给原创视角 |
| N1 | 矛盾生成 | 用系统论描述符（node/input/output/inner/edge）描述系统 |
| N2 | 主要矛盾判断 | think 三维打分（根本性/全局性/决定性）+ human 选定 |
| S-R | 辩证逆转 | "若 X 必然，实现 Y 应当如何？" |
| EC | 良知批判 | sense_loop 呈现回顾，human 自判（无 subagent） |

允许反向回流（后续阶段发现关键事实可重启前序阶段，上限 `sense_max_backflows` 默认 3）。

## 四文件架构（think/research/exporter 为 pi 自定义 agent）

| 文件 | 角色 |
|------|------|
| `sense_loop.md` | main agent 协议（5 阶段调度 + 批判门禁嵌入 + 反向回流） |
| `think.md` | 推理识别 + 假设提取 + 4 维打分 + 3 启发性问题 |
| `research.md` | 尽调树 + 信源加权 + subagent 上下文隔离 |
| `exporter.md` | RFC 输出（大纲 + 内容两阶段） |

think/research/exporter 经 env 职责 3 落盘为 pi 自定义 agent（`agents/{name}.md`）。

## 配置项

`.rick/config.json` 的 `human_loop` 嵌套字段：`max_retries`（默认 5）、`sense_max_backflows`（默认 3）、`think_top_n`、`think_min_assumptions`（默认 5）、`research_source_weights`。

## 目录结构

```
.rick/draft/loops/loop_N/
├── prompts/           # sense_loop / think / research / exporter + skills
└── briefs/            # 每轮子 agent 产出简报
```
