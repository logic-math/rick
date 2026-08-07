# research-4-N4-PI agent 概念

节点路径:[根 > N4-PI agent 概念]
事实陈述:PI agent 在 rick 仓库中是否出现，若出现记录上下文；若不出现标记 R7

## 执行动作
- Grep "PI agent|PIAgent|pi_agent|piAgent|PI-agent|PI\s+agent" 全仓库（Go + MD）
- Grep "PI" 全仓库 Go 代码（确认是否有 PI 缩写定义）

## 各信源验证结果

### 代码原文 0.4 ❌
- "PI agent" / "PIAgent" / "pi_agent" / "piAgent" / "PI-agent" 在整个 rick 仓库（Go 代码 + .rick 文档）**零匹配**
- 唯一匹配：`.rick/draft/loops/loop_2/prompts/sense_loop.md`（本次会话协议文件）+ 本次调研报告自身
- Go 代码中无 "PI" 作为类型/常量/变量定义

### 运行时行为 0.3 ❌
- 无法验证（代码中无 PI agent 实现）

### 文档 0.2 ❌
- MEMORY.md 无 PI agent 记录
- .rick/loops/ .rick/skills/ .rick/domain/ 无 PI agent 定义

### 反事实 0.1 ❌
- 无代码可修改

## 置信度计算
0.4×0 + 0.3×0 + 0.2×0 + 0.1×0 = **0.0（低）**

## 还原确认
未修改代码，无需还原。

## 疑问点
**R7 上报**：PI agent 在 rick 仓库中无任何定义性引用。需 human 澄清：
1. PI agent 的全称/来源（是某个外部框架？美团内部 agent？还是 human 自创概念？）
2. PI agent 的协议规格（输入输出格式、调用方式、与 claude code 的差异点）
3. PI agent 的实现位置（是否已有可执行文件？还是需 rick 实现？）

无法通过代码/文档调研达高置信度，标记 R7。
