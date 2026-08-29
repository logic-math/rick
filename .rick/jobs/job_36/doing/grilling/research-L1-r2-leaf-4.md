# 调研：主流 Agent Web UI 的 Session 管理交互模式（rick web ui 参考）

## ① OpenHands（Agent Canvas）
- **a) 新建**：启动表单先选 repo + agent profile（会话绑定 profile/plugins），填完才进聊天流。数据模型=会话内事件流（消息/工具调用/文件变更均存 event）。
- **b) resume**：侧栏 Conversation Panel 分页列出会话（时间/工作区分组、状态点），点击进入后通过 WebSocket 回放历史事件重建 UI（isLoadingHistory 标记）。
- **c) 流**：WebSocket 双连接；StreamingDeltaEvent 增量渲染文本；Zustand useEventStore 批处理防高频卡顿；handleEventForUI 归一化并把最终消息与流式 delta 合并去重；工具调用为可折叠分组卡片；错误两类——可展示错误走 banner，AgentErrorEvent 内联聊天。
- **d) 移动端**：1024px 断点切 desktop/mobile 布局（已知 bug：跨断点 resize 触发长会话 30s+ 重载）；移动端用 mobile panel page + 顶栏图标按钮 + 侧栏折叠成状态点。

## ② claude-code-webui（sugyan）
- **a) 新建**：进页面即聊天（ChatPage），无前置类型选择；设置（provider/model/权限模式）在 SettingsModal、localStorage 持久化；会话=session id，直接复用 Claude CLI 的 ~/.claude/projects 下 JSONL 存储。
- **b) resume**：HistoryView 按项目分组列出会话（时间戳+摘要+重复过滤），选中后以 --resume 恢复并回读 JSONL 重建消息历史。
- **c) 流**：SSE + UnifiedMessageProcessor 管道：文本 delta 流式渲染；工具调用渲染为带权限确认的卡片（default/plan/acceptEdits 三模式）；错误经 onPermissionError 等回调处理；session id 从 init 消息提取。
- **d) 移动端**：README 官方截图即含 Mobile Experience，响应式布局，输入区常驻底部。

## ③ goose（block/goose，desktop 与 web）
- **a) 新建**：桌面端"+"新会话，底部目录切换器选 working dir，可挂 recipe/extension；无类型前置选择。数据模型：1.10.0 起 SQLite（sessions.db），含消息历史+工作目录+extension 状态。
- **b) resume**：侧栏 Chat 区列最近 10 个活跃会话，点击即续聊；ACP 断连（睡眠/掉线）后带上限退避+抖动重连并恢复活跃会话；支持 goose://resume 深链。
- **c) 流**：goosed agent server 用 SSE（reply 端点 `data:` JSON 流）；桌面 React 组件分层：消息显示/用户输入/工具执行可视化/实时流更新。
- **d) 移动端**：Electron 桌面为主（多窗口）；新增 ui/web（React18+Vite）与桌面功能对齐，共用同一 TypeScript SDK。

## ④ LibreChat
- **a) 新建**：新对话按钮→空白聊天流，首条消息时才落库；参数（endpoint/model/temperature 等）作为 conversationPreset 存进 IConversation 文档，随会话持久化。
- **b) resume**：侧栏会话列表（搜索/分组）→ GET /api/messages/:conversationId 分页拉取消息树重建；支持消息树分支/编辑再生成。
- **c) 流**：可恢复 SSE——POST 发起与 GET EventSource 订阅分离，断线指数退避重连，导航离开不中断生成（仅 stop 按钮终止）；工具调用 2+ 连续自动折叠成分组卡（类型图标、JSON 解析、复制、错误态、完成态判断）；Thinking 独立折叠组件。
- **d) 移动端**：768px JS 断点（useMediaQuery max-width:768px）驱动侧栏 drawer 覆盖态；统一图标条侧栏保证桌面/移动一致；Tailwind md: 类需与 JS 断点对齐（曾有不对齐 bug）。

## 【结论】
1. 主流均"先进聊天后补配置"，参数随会话持久化；cmd 类型宜作会话属性而非前置表单门槛。
2. resume 统一为"列表→按 id 拉事件/消息重放重建"；SSE 可恢复流+断线重连是成熟标配。
3. 渲染分层共识：文本 delta 直渲＋工具调用折叠卡＋thinking 折叠＋错误内联；移动端 768/1024 断点抽屉侧栏。

---
主要来源：deepwiki（OpenHands 7.x/8.2、claude-code-webui 4.x、block/goose 3.x、LibreChat 6.1）；GitHub 源码（OpenHands frontend、sugyan/claude-code-webui、block/goose ui/desktop、danny-avila/LibreChat）；block-goose.mintlify.app、librechat.ai/docs、docs.openhands.dev。
