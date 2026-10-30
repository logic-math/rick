# task18：adjust.md 重写（权重语义 + expand_cell）

## 任务目标
J-i3-1/J-i3-2：adjust 成为唯一采样决策者——prompt 契约对齐新语义。

## 关键结果
- `super/prompts/adjust.md` 重写：
  - **权重=相对采样份额**：概率∝权重；可设 >1（如 2.0=平均份额两倍）；weight=0=功能性灭绝
  - 决策纪律条款：每次权重变动须引用 ≥1 条事件证据（seq 号）；同类连续降权须间隔（防震荡）；新增格默认权重 1.0 由框架处理
  - **expand_cell 提案 schema**：`{type: "expand_cell", action: "add_axis_value" | "add_cell", axis, value | descriptor, rationale}`（走既有裁决链——硬动作）；何时值得加格的判断启发（数据饱和/新方向证据）
  - 输出契约 JSON 块声明保持（proposals 数组）
  - 移除 UCB 相关表述（若有）
- 提示词中告知：采样器已改为纯加权随机（你调权重=直接调采样分布）

# 写域
- super/prompts/adjust.md

# 依赖关系
- 无（L1 首层并行）

# 测试方法
`grep -c "采样份额\|expand_cell" super/prompts/adjust.md` ≥2；无 ucb 残留表述

# 禁止事项
- 不动其它三个模板；不动框架代码
