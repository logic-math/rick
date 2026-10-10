# 叶子2：ChatGPT与Claude web UI 交互基准（HITL/信息密度导航/错误恢复）

## ② Human-in-the-loop
- **Canvas 双栏协作+内联编辑**：侧栏独立画布，用户可直接改文字/代码，或选中片段浮动工具条内联提问/格式化，配快捷按钮（调长度/加日志/调试）；顶部箭头回看版本+Show changes 差异比对+恢复旧版。[官方·高](https://help.openai.com/en/articles/9930697-what-is-the-canvas-feature-in-chatgpt)
- **Agent 模式暂停-确认-打断**：执行中可随时打断改指令，agent 按需暂停请求澄清/确认；高危操作需用户确认，敏感站点强制"watch mode"监督；登录时暂停并让用户"Take over browser"接管（接管期不截屏保护密码），交还后从原状态续跑。[官方·高](https://help.openai.com/en/articles/11752874-agent)
- **Claude Artifacts 版本切换**：侧边预览窗+对话并存；每次编辑存新版本，版本下拉可任意切换/编辑/发布指定版本；Markdown 支持选中文字"Edit with Claude"就地改；侧栏 Artifacts 区集中管理。[官方·高](https://support.claude.com/en/articles/9487310-what-are-artifacts-and-how-do-i-use-them)
- **Claude Code 权限分级**：default/acceptEdits/plan 多档权限模式，plan 只读探索后出计划再执行。[官方·高](https://code.claude.com/docs/en/permission-modes)

## ④ 信息密度与导航
- **ChatGPT 侧栏**：Search（Ctrl/Cmd+K）全局搜会话；Projects 聚合相关会话+文件+项目指令与记忆；归档从侧栏移除但保留可恢复，与删除区分。[官方·高](https://help.openai.com/en/articles/10169521-projects-in-chatgpt) [官方·高](https://help.openai.com/en/articles/8809935-deleting-and-archiving-chats-in-chatgpt)
- **Claude 侧栏**：Projects=独立工作区（会话史+知识库+项目指令，右侧知识库面板）；付费版可让 Claude 检索过往对话；新版本演进为"一个项目=一个对话+云端并行线程"汇报等待项。[官方·高](https://support.claude.com/en/articles/9519177-how-can-i-create-and-manage-projects) [官方·高](https://support.claude.com/en/articles/9517075-what-are-projects)
- **上下文管理提示**：Claude 付费新模型 200K–1M token 窗口，帮助中心明确解释长度限制即上下文窗口；API 侧提供自动 compaction（阈值触发摘要旧轮次）。[官方·高](https://support.claude.com/en/articles/11647753-how-do-usage-and-length-limits-work) [官方·高](https://docs.anthropic.com/en/build-with-claude/compaction)

## ⑤ 错误与恢复
- **后台长任务**：ChatGPT agent/云浏览器任务可关机后台续跑，卡住时主动请用户接管或发链接邀请接管；恢复失败时可能重建会话/再次提示。[官方·高](https://help.openai.com/en/articles/20001280-using-cloud-browser-in-chatgpt)
- **Scheduled Tasks**：专用 Scheduled 页统一管理一次性/循环任务，Push+Email 双通道通知，浏览器权限未开时给明确提示。[官方·高](https://help.openai.com/en/articles/10291617-scheduled-tasks-in-chatgpt)
- **Claude Code 检查点**：每条用户消息前自动快照（保留最近100个），/rewind 回滚代码+对话，/resume、--continue 跨会话恢复，/branch、/fork 分叉试错；任务边界用 /compact 摘要、/clear 隔离任务，/context 可视化占用。[官方·高](https://code.claude.com/docs/en/checkpointing) [官方·高](https://code.claude.com/docs/en/commands)
- **上下文超限**：ChatGPT 触顶提示"开新对话"仅文案恢复，无自动续接（三方指出可编辑消息分支绕过）。[三方·低](https://www.folk.com/blog/chatgpt-maximum-length-conversation-fix) 社区报告 Claude Code 触限时 /compact 仍可能报错需 /clear。[社区·低](https://github.com/anthropics/claude-code/issues/23047)

## 结论 3 条
1. HITL 基准=随时打断+分级确认+敏感操作人工接管+自动快照可回滚。
2. 导航基准=项目聚合会话/文件/指令，归档保留可恢复，Cmd+K 全局搜索。
3. 恢复基准=后台续跑+卡住请求接管；上下文超限前主动压缩而非事后报错。
