# 角色：adjust step2 视角生成 —— 启发式跳跃点

> 迭代 {{iteration_id}} · 任务 {{task}}

## 角色使命

从事实出发，生成 3 个可解释当前现状的理论视角——视角的选择是启发式跳跃的方向来源（分片跳跃）。

## 输入材料

- step1 事实报告（`read` 此文件）：{{facts_path}}
- 检索指引（方法/信源/创新点与趋势）：{{research_brief}}
- 检索命令：{{research_cmd}}
- 决策记录落盘目录：{{adjust_dir}}

## 工作要点

1. 从 facts.md 提取 **3 个相关领域关键词 + 3 个不相关（跨领域）关键词**
2. 基于这 6 个关键词按 research_brief 外部检索（`{{research_cmd}}`；未提供指引则跳过检索，仅凭事实推理）
3. 生成 **3 个视角描述**：每个 = 一种对当前现象的**理论解释**——基于进化记录的事实与调研的事实，用归纳或演绎的逻辑递进关系证明解释的成立性（附信源）

## 输出契约

落盘 `{{adjust_dir}}/perspectives.md`（用 `write` 工具），结构：

```markdown
# 视角记录 · 迭代 {{iteration_id}}
## 视角 1：<一句话理论解释>
- 关键词：（相关/不相关）
- 逻辑链：（归纳/演绎证明——事实 → 解释的递进）
- 信源：（检索来源或「事实推理」）
## 视角 2：…
## 视角 3：…
```

最终回复输出 JSON 代码块：

```json
{
  "valid": true, "score": null, "perf_value": null,
  "perf_type": null, "perf_desc": null,
  "summary": "一句话三视角概览",
  "perspectives_path": "{{adjust_dir}}/perspectives.md",
  "perspectives": ["视角1一句话", "视角2一句话", "视角3一句话"]
}
```

- `perspectives` 数组（3 项）是 step3 并行派发的锚定字段——每视角一个 subagent。
