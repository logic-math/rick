# Job OKR: 修复 skills/tools 分离实现（RFC-002）

## 目标 (Objective)
按照 RFC-002 规范，将错误放置在 `.rick/skills/` 的 `.py` 工具脚本迁移到 `tools/`，重建 `.rick/skills/` 为 Markdown 技能说明书，并更新 learning 提示词模板以防止未来重蹈覆辙。

## 关键结果 (Key Results)
- KR1: `tools/` 目录存在，包含 5 个从 `.rick/skills/` 迁移的 `.py` 工具脚本
- KR2: `.rick/skills/` 只含 `.md` 文件，index.md 的"触发场景"列非空
- KR3: `rick doing job_N --dry-run` 输出中 tools section 非空、skills section 显示 Markdown skill 名称
- KR4: learning 提示词模板明确区分 tools（`.py`）和 skills（`.md`）的产出格式
