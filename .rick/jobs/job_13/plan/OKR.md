# Job OKR: 优化 human-loop，迁移到 prompt 系统内联注入

## 目标 (Objective)
将 human-loop 的三个 sub agent 从 skills 机制迁移到 rick prompt 模板系统，直接嵌入主 prompt，完全解耦 Claude Code CLI，并按 RFC 规范重写所有提示词内容。

## 关键结果 (Key Results)
- KR1: 新建 3 个 sub agent 模板文件，内容按 RFC 规范（含假设追问格式），通过 Go embed 注入主 prompt
- KR2: human_loop.md 主控模板内联三个 sub agent 内容，不再依赖任何斜杠命令
- KR3: skills/ 目录删除，install.sh 中 skills 相关逻辑全部移除
- KR4: human-loop dry-run 输出完整 prompt，编译通过，集成测试通过
