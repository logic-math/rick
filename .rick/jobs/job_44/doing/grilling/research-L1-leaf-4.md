# Research: 2024–2026 AI Agent 产品交互模式有效性证据清单(开发者向)

## Summary
官方 changelog 与工程博客一致验证了流式+可打断、diff-first 审查、plan mode、后台任务、checkpoint 回滚与键盘优先等模式;而向导式/重 review 界面(Copilot Workspace)与逐轮权限弹窗已被官方弃用或证伪。开发者核心诉求按证据强度排序:可打断的透明执行流 > 可审查的 diff > 可回滚 > 低打扰的后台并行。

## Findings

### A. 被验证有效的模式

1. **流式渲染+随时可打断(Esc 打断,保留已完成工作)** — Claude Code 官方 keyboard shortcuts 文档将 `Esc` 定义为 "Interrupt Claude... Stop the current response or tool call mid-turn so you can redirect. Claude keeps the work done so far";Cursor 1.4 将"边工作边插话"作为 steerability 核心改进:消息在下一个 tool call 后立即生效,⌘+Enter 可打断注入。信源:官方文档/changelog(高)。[Claude Code interactive-mode](https://code.claude.com/docs/en/interactive-mode) · [Cursor 1.4](https://cursor.com/changelog/1-4)
2. **Diff-first 审查** — Claude Code `/diff`、`/code-review`(后改为后台 subagent 以免污染对话流)持续迭代;Cursor 以 inline diff + Accept/Reject 为核心交互。官方 changelog(高)。[Claude Code changelog](https://code.claude.com/docs/en/changelog) · [Cursor changelog](https://cursor.com/changelog/1-0)
3. **Plan mode 计划先行** — Claude Code 将 `plan` 设为 Shift+Tab 权限模式之一;后续 changelog 多次修复 plan mode 误放行 file-modifying/browser 调用,说明是重度使用的正式功能。官方文档(高)。[Interactive mode](https://code.claude.com/docs/en/interactive-mode)
4. **后台任务+完成通知** — Cursor 0.50 引入 Background Agent(early preview),1.0 因"early signals have been positive"扩展至所有用户 GA;Claude Code 将 /code-review 改为后台 subagent。官方 changelog(高)。[Cursor 0.50](https://cursor.com/changelog/0-50) · [Cursor 1.0](https://cursor.com/changelog/1-0) · [Claude Code changelog](https://code.claude.com/docs/en/changelog)
5. **Checkpoint 回滚** — Claude Code Checkpointing:每个 user prompt 前自动快照,可 rewind 文件+对话到任意状态(Esc Esc / `/rewind`)。官方文档(高)。[Checkpointing](https://code.claude.com/docs/en/checkpointing)
6. **可展开的工具调用透明度** — Claude Code/Agent SDK 展示每步 tool call 及参数;社区与工程文章均把"进度流+可介入"列为 agent UI 必备七要素之一(进度流/确认门/错误恢复)。官方文档+工程实践(中)。[Brainy agent UI patterns](https://brainy.ink/paper/ai-agent-ui-design-patterns)
7. **键盘优先操作** — Claude Code 提供 vim 编辑模式、Shift+Tab 循环权限模式、Esc 打断/关闭弹窗、Ctrl+G 等快捷键体系,是终端工具的默认交互层。官方文档(高)。[Interactive mode](https://code.claude.com/docs/en/interactive-mode)

### B. 被弃用/证伪的模式

1. **Copilot Workspace 式重 review 界面(向导式多步:任务→计划→实现→审 PR)** — 官方 2025-05-30 结束 technical preview 并下线,转向 VS Code Agent Mode;主因是 Web 端无法 build/test 验证代码、"low latency autocomplete 的技术不适用于 repo-scale 建议"。官方 sunset 说明(高)。[Copilot Workspace Sunset](https://gist.github.com/idan/15f101b8ff4be8bebe17147c08995363) · [dsyme 分析](https://dsyme.net/2025/01/25/copilot-workspace-and-the-birth-of-task-oriented-programming/)
2. **逐轮权限弹窗 / 轻信"人审"兜底** — Anthropic 工程博客披露遥测:用户批准约 93% 的权限提示,导致审查疲劳失效,因此用 auto mode 后台安全检查替代"每 turn 询问"。官方工程博客(高)。[How we contain Claude](https://www.anthropic.com/engineering/how-we-contain-claude)
3. **自动全量接受(yolo/bypass 作为默认)** — 同上:Anthropic 明确批评逐 turn 批准形同虚设;Cursor/Cladue 均把 acceptEdits/bypass 设为需显式切换的非默认模式。官方(高)。

### C. 开发者体验要素排序(证据强度)

1. **准确性与可验证性 > 一切** — 2025 Stack Overflow 调查(49k 人):46% 不信任 AI 输出准确性 vs 33% 信任;87% 担心 agent 准确性 → diff/测试可验证是第一诉求。可信调查(中高)。[SO Survey 2025 AI](https://survey.stackoverflow.co/2025/ai)
2. **可打断/可纠偏(steerability)** — Cursor 专门 changelog 优化"工作中插话";Claude Code Esc 语义保留已完成工作。官方(高)。
3. **低打扰的后台并行与可见性** — Background Agent GA + Claude Code 自动模式说明"打断用户"本身是负体验。官方(高)。
4. **可回滚的安全感** — checkpoint/rewind 成为 Claude Code 一级文档特性。官方(高)。
5. **透明度(工具调用可见、成本/上下文用量可见)** — Cursor 在 chat 中显示 context window 使用量;SO 调查 81% 关心数据安全。官方+调查(中高)。[Cursor 1.3](https://cursor.com/changelog/1-3)

## Sources
- Kept: Claude Code changelog / interactive-mode / checkpointing (code.claude.com) — 官方一手证据
- Kept: Cursor changelog 0.50 / 1.0 / 1.3 / 1.4 (cursor.com) — 官方一手证据
- Kept: Copilot Workspace Sunset gist (idan, GitHub Next 总监) — 官方弃用声明
- Kept: Anthropic engineering "How we contain Claude" — 93% 批准率遥测数据
- Kept: Stack Overflow Developer Survey 2025 — 49k 样本开发者信任数据
- Dropped: hackernoon / zejiantan.com / junpingyi.com 个人博客 — 观点性、无一手数据;dsyme 保留作为 Copilot Workspace 技术失败佐证

## Gaps
- 缺少对"一次性大段输出不可中断"被正式弃用的直接官方声明(该模式因竞品全面流式化而被市场淘汰,属间接证伪);
- Cursor/Claude 未公开 plan mode、checkpoint 的使用率数据;HN 讨论未系统采样,排序主要依赖官方行为信号+大样本调查。

## 结论 3 条
1. 流式+Esc 随时打断、diff 审查、plan mode、checkpoint 回滚已被官方反复投入验证为有效。
2. 重 review 向导界面与逐轮权限弹窗均被官方弃用(93% 盲批准证明人审兜底失效)。
3. 开发者首要诉求是可验证准确性与可纠偏,其次是后台并行低打扰与用量透明。
