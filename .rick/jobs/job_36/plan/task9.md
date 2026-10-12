# 依赖关系
task7

# 写域
web/src/components/chat/
web/src/components/extui/

# 任务目标
会话聊天窗组件族：事件流渲染分层（delta 直渲/工具折叠卡/thinking 折叠/错误二分）+ 输入交互 + extension_ui 弹窗桥。

# 关键结果
1. `web/src/components/chat/ChatView.tsx`：会话主视图 props{sessionId}——订阅 events store 该 session 的事件流；顶部会话头（类型徽标+title+状态点：active=传送门绿呼吸/running=飞碟悬停/closed/error）；主体 MessageList 滚动容器（新消息自动贴底+用户上滚取消贴底）；底部 ChatInput
2. `MessageList.tsx` + `MessageBubble.tsx`：user/assistant 气泡（assistant 用 react-markdown+remark-gfm+rehype-highlight 渲染；流式 delta 时直接拼接渲染当前气泡——StreamText.tsx 光标闪烁；message_end 后落定）；错误二分：连接/服务错误=顶部 ErrorBanner，agent 内容错误=内联卡片
3. `ToolCallCard.tsx`：工具调用折叠卡（工具名+参数摘要单行；展开看全参数与结果；bash=ansi-to-react 渲染输出；read/write/edit=代码块；edit=@git-diff-view/react 渲染 diff；≥2 个连续工具调用自动折叠为一组「N 次工具调用」）；`ThinkingBlock.tsx`：thinking 内容默认折叠（「思考过程」可展开）
4. `ChatInput.tsx`：多行输入（Enter 发送/Shift+Enter 换行；软键盘 VisualViewport 适配——fixed 底部+visualViewport resize 监听）；agent streaming 时输入框变 steer 条（发送即 steer）；命令斜杠 `/`（/abort /close /compact 提示——仅 /abort /close 实装调 API，其余提示暂不可用）；附件按钮占位（disabled+tooltip「v1 暂不支持」）
5. `SteerBar.tsx`：streaming 中的操作条——Abort 按钮（调 abort API）+ steer 输入（与 ChatInput 合一可选）；closed 会话显示 Resume 按钮
6. `web/src/components/extui/ExtensionUIDialog.tsx`：**自包含**（不 import 同层 task10 的 components/common/——并行中间态缺模块 tsc 红；样式 token 直接用 theme.css 变量）——按 extension_ui_request 的 method 渲染——select（选项列表单选）/confirm（确认卡）/input（单行）/editor（多行 textarea 预填）；「取消」按钮发 cancelled；提交发 uiResponse API（request_id 对应）；同 session 多个 pending 请求排队展示
7. 视觉：R&M 主题 token（chat 气泡用 surface 色+传送门绿 accent；错误用 Morty 黄警示/红）；组件全部函数式+props 驱动（无业务 fetch——全走 stores/api）
8. `npx tsc --noEmit` 通过（npm run build 留给 gate3 层完成后统一执行——同层 task10 并行 build 会踩 dist 竞争）

# 测试方法
tsc+build 全绿；手工 smoke：npm run dev 起占位数据（stores 手动注入 fake 事件序列）逐组件目检（流式光标/折叠卡/弹窗四态）

# 上下文提示
- 渲染分层模式=OpenHands/LibreChat 共识（research-L1-r2-leaf-4）：delta 直渲+批处理防卡顿（events store task7 已做缓冲——组件订阅批量 flush 后的视图模型）
- Markdown XSS：dompurify 清洗（react-markdown rehype-raw 慎用——默认不启用 raw，图片外链禁）
- 组件参考 pi-web-ui ChatInput/ToolCallBlock/ThinkingBlock 结构（research-L2.md，MIT）
