# 依赖关系
无（基于 job_36 已交付代码的改进任务）

# 写域
web/src/components/chat/
web/src/components/common/Collapse.tsx
web/src/components/monitor/EventStream.tsx

# 任务目标
修复 session 会话「完整行为轨迹」体验：历史事件已回填（entries API 正常）但折叠组不可交互、内容层级不完整。目标：**全部事件（工具调用详情/思考过程/系统提示词/消息全文）都展示在会话时间线中，默认折叠、点击可展开、只默认展开最近活跃项**，方便用户复查 agent 完整行为轨迹。

# 关键结果

1. **可交互折叠组件**：`web/src/components/common/Collapse.tsx`（新建）——受控/非受控折叠容器（标题行 + ▸/▾ 指示 + 内容区动画可选）；键盘可达（button 语义 + aria-expanded）；替换 chat 视图里现有不可点击的「▸思考过程」「▸N 次工具调用」死文本为该组件。
2. **工具调用组完整展示**：折叠组展开后逐个显示工具调用卡（复用/增强 ToolCallCard）——每卡含：工具名、**完整参数**（bash 命令全文/read 文件路径/write 内容摘要/edit diff）、**完整输出**（截断阈值 8KB + 「展开全部」）、isError 红色标记 + 失败摘要；组头显示 `N 次工具调用（M 失败）`+ 工具名序列。
3. **思考过程块**：ThinkingBlock 改用 Collapse 组件——展开显示完整 thinking 文本（markdown 渲染可选，纯文本 pre-wrap 即可）；默认折叠；**正在流式输出的 thinking（当前活跃 turn）默认展开**。
4. **消息全文展示**：assistant 正文（text 块）不可折叠（时间线主干）；user 消息全文；长文本 16KB 截断 + 展开全部。
5. **系统提示词可见**：ChatView 头部新增「会话信息」折叠区（默认折叠）——展示 session 元信息：type/params/pi_session_id/创建时间；**系统提示词**：从 entries 的首条 user 消息之前的元数据 + 会话的 prompt 来源说明（plan/easy 等类型的 prompt 文件路径提示「<job>/plan/prompts/plan_prompt.md」——不内嵌文件内容，显示路径即可，文件内容经 job files API 另行查看）；若 entries 含 model_change/thinking_level_change 也在此区显示。
6. **默认展开策略**：页面加载后只展开**最后一个活跃片段**（最近的 assistant 正文之后的 thinking/tool 组）；其余全部折叠；提供「全部展开/全部折叠」切换按钮（时间线顶部小按钮）；折叠状态本地记忆不要求（session 内 useState 即可）。
7. **历史回填合并去重**：确认 viewModel 的 buildHistoryItems 与 SSE 实时事件的合并逻辑在「先回填后实时」场景无重复（已有的去重逻辑复核，发现问题修复——比如历史 message_end 与实时 envelope 的 id 对齐）。
8. **EventStream（monitor 视图）同步增强**：编排事件流的工具调用行同样可点击展开（复用 Collapse + 简版详情）；gate 结果横幅保持。
9. `npx tsc --noEmit` + `npm run build` 通过；手动 playwright 冒烟（parent 会跑）。

# 测试方法
- tsc + build 全绿
- 自测：npm run dev 打开一个有历史的会话（6137 上有真实 plan 会话 aed8d899-766d-4139-8840-0ea5432a495b），验证：折叠组可点击展开/收起、24 次工具调用展开后有完整命令与输出、thinking 展开有全文、默认只展开最近活跃段

# 上下文提示
- 入口：`web/src/components/chat/ChatView.tsx`（125 行附近 getEntries 回填 + buildHistoryItems）
- viewModel：`web/src/components/chat/viewModel.ts`（事件→时间线 item 映射，history + live 两路）
- 后端数据已验证完整：GET /api/sessions/{id}/entries 返回 51 entries（message/user、message/assistant 含 thinking+text+tool_use blocks、message/toolResult 含完整 content、model_change、thinking_level_change）——**纯前端任务，后端无需改**
- entries 结构：`{type:"message", id, parentId, timestamp, message:{role, content:[{type:"text"|"thinking"|"tool_use"|"tool_result", ...}]}}`；tool_use 含 name/input；toolResult 消息的 content 为结果块（含 isError）
- 现有 ToolCallCard/ThinkingBlock 已有样式基础，重点是接上 Collapse 交互 + 历史数据字段
- 主题：R&M 深色 token（--rm-* / tailwind 语义色），折叠交互 hover 用 portal 绿点缀
