---
name: loop-protocol
description: doing/easy 阶段父 agent 执行项目 Loop 的迭代控制协议
---

# Loop 执行协议（父 Agent 迭代控制）

## Step 1：加载 Loop

读取 loops_context，从中选择本次任务对应的 Loop（按 trigger 字段匹配）。

若无匹配 Loop，则按默认节奏执行：分析 → 实现 → 验证 → 收尾。

## Step 2：执行一次迭代

进入 Loop 的一个完整迭代周期：

1. **计划**：明确本轮目标，确认与 Loop trigger/scope 对齐
2. **执行**：完成一个最小可验证单元（不超出 scope）
3. **验证**：运行测试，确认实现符合预期
4. **回顾**：检查本轮产出是否满足 key results，决定继续迭代或退出

## Step 3：退出判断

满足以下任意一条时退出 Loop：

- 所有 key results 均已达成
- 当前任务的测试全部通过
- 人类明确要求停止

## Step 4：收尾

完成最终验证（doing_check 或 easy_check），提交 git commit，更新 tasks.json 状态。

## 分工铁律

- **父 Agent**：控制 Loop 节奏，不直接写代码，只调度子 Agent
- **子 Agent**：执行具体实现（编码、测试、调试），只在当前迭代的 scope 内操作
- **禁止**：子 Agent 跨越 scope；父 Agent 绕过验证直接推进下一轮
