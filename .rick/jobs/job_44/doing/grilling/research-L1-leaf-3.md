# 调研：Devin / Cursor / Copilot Workspace 的 agent 交互模式基准与教训（2025–2026）

面向 rick（Go+React 编排工具）交互升级：任务看板/事件流/门禁可对标下述模式。

## ① 长任务运行感知
1. **Devin 进度直播**：任务启动即给出计划；Progress 页统一呈现终端/IDE/浏览器状态，可实时旁观、随时接管，会话支持事后回放（replay）。官方=高 https://docs.devin.ai/work-with-devin/devin-session-tools
2. **Devin 成本可见**：按 ACU 计量，任意用户可在 Session Insights 查单会话用量与会后效率分析（官方案例：42→12 ACU 优化）；Managed Devins 可给子会话设 ACU 上限并监控消耗。官方=高 https://docs.devin.ai/product-guides/session-insights https://docs.devin.ai/work-with-devin/advanced-capabilities
3. **Devin 复用与并行**：Playbook 把成功会话固化为可分享复用流程；2.0 多 Devin 并行（各配云 IDE）；"Manage Devins" 为管理者+worker 模式（各独占 VM、自跑测试验证后合并 PR）；2.1 加 🟢🟡🔴 完成置信度；2.2（2026-02）可自测、自验证、自修复。官方=高 https://docs.devin.ai/product-guides/creating-playbooks https://cognition.com/blog/devin-2 https://cognition.com/blog/devin-can-now-manage-devins https://cognition.com/blog/devin-2-1 https://cognition.com/blog/introducing-devin-2-2
4. **Cursor 后台任务**：Cloud Agents（原 Background Agents）在专用 VM 长时运行，并行数量不限、不依赖本地在线；Slack 集成触发+完成通知，Web/移动端可管理；Agents Window 集中管理跨仓库多 agent，本地/云一键互转。官方=高 https://cursor.com/docs/background-agent https://cursor.com/docs/agent/agents-window
5. **Cursor 并行与排队**：2.0 单提示最多 8 个 agent 并行（git worktree 隔离防冲突）；2.2（2025-12）自动评判多 agent 结果并推荐最优；queued messages 支持排队追加。官方=高 https://cursor.com/changelog/2-0 https://cursor.com/changelog/2-2 https://cursor.com/changelog/1-2

## ② Human-in-the-loop
6. **Cursor diff 审查**：逐文件查看、按风险决定接受/继续调整/回退；Review→Find Issues 让 agent 逐行自查；Agent Review 可设为每次 agent 任务后自动跑代码审查。官方=高 https://cursor.com/learn/reviewing-testing https://cursor.com/docs/agent/agent-review
7. **Cursor 打断与追加**：见偏可 Stop（Cmd+Shift+Backspace）立即取消改向；消息可排队（拖拽排序、当前任务完成后顺序执行），或即时发送在下一工具调用处"转向"，Cmd+Enter 立即中断。官方=高 https://cursor.com/docs/agent/overview https://cursor.com/changelog/1-4
8. **Cursor 计划先行与回滚**：Plan Mode 先研究代码库、提澄清问题、产出可编辑 Markdown 计划，确认后才写码；Checkpoints 在每次重大改动前自动快照，聊天时间线任一点可预览并整组恢复。官方=高 https://cursor.com/docs/agent/plan-mode https://cursor.com/blog/plan-mode https://cursor.com/docs/agent/overview
9. **Devin HITL**：会话中可随时追加消息（含 API）；/btw 侧聊可在不打断主任务下提问。官方=高 https://docs.devin.ai/api-reference/v1/sessions/send-a-message-to-an-existing-devin-session https://docs.devin.ai/work-with-devin/devin-session-tools

## ⑤ 错误恢复 + Copilot Workspace 教训
10. **Cursor 错误恢复**：Await 工具让 agent 等待后台命令/子 agent 完成或监听特定输出（如 "Ready"/"Error"）；3.0 起 agent 更善于监控长任务。官方=高 https://cursor.com/changelog/3-0
11. **Copilot Workspace 终局**：2024-04 技术预览，始终未 GA，2025-05-30 关闭；GitHub 同期转向 issue→PR 的 Copilot coding agent 与 VS Code Agent Mode。近官方=高 https://gist.github.com/idan/15f101b8ff4be8bebe17147c08995363 官方=高 https://github.com/newsroom/press-releases/coding-agent-for-github-copilot
12. **教训（内部工程视角）**：纯 Web 端无法构建/测试生成代码、缺验证闭环是最大技术败因；大 diff 审查负担重、单次预览难迭代、信任难建立。工程博客=中 https://dsyme.net/2025/01/25/copilot-workspace-and-the-birth-of-task-oriented-programming 三方=低 https://chagdev.com/github-copilot-workspace-six-months-in-what-actually-happened/

## 结论 3 条
1. 长任务感知三要素：计划先行、进度可旁观/可回放、ACU 成本可见。
2. HITL 三件套：细粒度 diff 审查、可打断可排队追加、checkpoint 回滚。
3. Workspace 教训：无验证闭环的大 diff 压垮审查与信任；须小步交付+自验证。
