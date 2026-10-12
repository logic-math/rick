# Job OKR: grilling 机制融合到 easy 和 plan 命令

## 目标 (Objective)
在 `rick easy` 和 `rick plan` 的同一 Claude 会话中嵌入结构化 grilling 追问机制，通过逐问追问 + 给出推荐答案，将需求澄清到具体可落实的代码路径或工具调用级别，从而提升 plan 产出质量。

## 关键结果 (Key Results)
- KR1: 新增独立 `skills/grilling.md` skill 文件，可通过 embed.FS 加载，内容包含追问协议 + 终止条件
- KR2: `plan.md` 模板移除 sense S→E→N 追问部分，新增 grilling 步骤（grilling_skill_path 变量注入），SOP 步骤连续无断号
- KR3: `easy.md` 模板新增 grilling 步骤，grilling 结束后追加澄清内容到 requirement.md（保留原始问题）
- KR4: `./bin/rick plan --dry-run` 和 easy 生成的 prompt 文件均包含 grilling skill 路径，不含未替换的 `{{grilling_skill_path}}` 占位符
