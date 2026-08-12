如何让 pi 把 think 折叠不显示，感觉无效信息比较多。其次是
---

## Grilling 澄清结论（2026-XX-XX，job_33）

**原始需求**：如何让 pi 把 think 折叠不显示（无效信息多）+ 需求被截断的"其次"。

**Q1（主题）**：当前 tokyo-night-dark 主题不够好，希望有更多主题可选。
→ 结论：提供 `rick tools theme` 命令（列出可选主题 + 切换），可选集合：内置 dark/light、tokyo-night-dark/light（@wishx127/pi-tokyo-night）、nightowl（mitsupi 包，可选装）。

**Q2（范围）**：对 rick 所有 pi 调用默认生效；并把 README"未来演进"中的规划直接落地——在 `~/.rick/pi` 创建配置目录，管理 pi 的全部配置（settings/扩展/主题），rick 自闭环。
→ 结论：实现配置目录隔离：rick 所有 pi 子进程注入 `PI_CODING_AGENT_DIR=~/.rick/pi/agent`；`rick tools init-pi` 在隔离目录下建 settings.json（含 hideThinkingBlock）、装扩展、管主题；与用户 `~/.pi` 完全隔离。

**Q3（隐藏方式）**：用 `hideThinkingBlock: true`，直接不展示思考过程（不是 --thinking off，思考仍生成但不显示）。
→ 结论：rick 托管的 settings.json 写入 `"hideThinkingBlock": true`，对 easy/plan/ctrl/human-loop 交互 TUI 全部生效。

**Q4（噪声位置）**：rick easy 中产生大量思考内容冲淡关键信息。
→ 结论：由 Q3 的 hideThinkingBlock 解决 TUI 展示；doing 的 --mode json 路径解析器本就不取 thinking 内容（只取 type=="text"），无需改动。

**实现要点**：
1. 新增 `internal/agent/piagent/agentdir.go`：AgentDir()（~/.rick/pi/agent，支持测试覆盖）、EnsureAgentDir()、AgentEnv()（注入 PI_CODING_AGENT_DIR）、SettingsPath()
2. `cli.go` CallCLI + `executor.go` Execute：cmd.Env = AgentEnv()（覆盖 plan/easy/ctrl/human-loop/doing 全部入口）
3. `tools_init_pi.go`：pi install/list/version 全部带 AgentEnv()；settings.json 路径改为隔离目录；从 ~/.pi/agent/settings.json 迁移（theme+packages）；确保 hideThinkingBlock: true
4. 新增 `rick tools theme`：list/set 主题
5. 更新 README「演进方向」段落（已实现）
