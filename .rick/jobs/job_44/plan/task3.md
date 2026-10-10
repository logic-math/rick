# task3：humans 页面前端 + 草稿持久化（书写体验）

## 任务目标

①web 新增 humans 工作区页面（浏览 human-loop 学习材料与进度，纯文件树 + 内容查看，**不要**阶段进展/判断卡片）；②修复书写体验——未发送草稿不再因页面刷新 / `frontend_reload` / 路由切换 / 会话中断而丢失。

## 关键结果

- **KR3a humans 页面**：
  - 路由 `/ws/:wsId/humans`（`App.tsx` 的 `WorkspaceRoutePage` kinds 增加 `"humans"`）+ 侧栏导航项「Humans」
  - `web/src/routes/Humans.tsx`：页面骨架（标题 + 说明「human-loop 学习材料与进度」），挂载 `HumansBrowser`
  - `web/src/components/humans/HumansBrowser.tsx`：左文件树 + 右内容双栏（交互镜像 `KnowledgeBrowser`），数据源 `GET /api/workspaces/{ws}/humans/tree|file`；markdown 用既有 `MarkdownView`
  - 空态提示：无 human-loop 时显示「尚无学习记录——在会话中以 human-loop 模式开始一次学习」
- **KR3b 草稿持久化**（`web/src/components/chat/ChatInput.tsx` + `ChatView.tsx` 传参）：
  - 挂载时从 `localStorage["rick-web-draft:" + sessionId]` 恢复草稿
  - 输入防抖（~500ms）写入；发送成功后清除；会话切换各存各的
  - `localStorage` 不可用（隐私模式/配额）→ try/catch 静默降级为纯内存（不报错、不影响发送）
  - sessionId 经 props 从 `ChatView` 传入（`ChatInput` 现无 sessionId）
- **KR3c 构建与类型**：`cd web && npm run typecheck && npm run build` 绿

# 写域

- web/src/routes/Humans.tsx（新建）
- web/src/components/humans/HumansBrowser.tsx（新建）
- web/src/App.tsx（路由 + 导航）
- web/src/components/chat/ChatInput.tsx
- web/src/components/chat/ChatView.tsx（传 sessionId/props）
- web/src/api/client.ts（如需新增 humans api 方法）
- web/src/components/layout/WorkspaceListNode.tsx（**仅一行**：WS_VIEWS 加 Humans 导航项；parent 批准的范围例外，2026-10-05）
- web/src/components/chat/SteerBar.tsx（**仅机械透传**：SteerBarProps 加 sessionId?: string 并传给 ChatInput；不改 steer/abort/closed 逻辑；parent 批准，2026-10-05）

# 依赖关系

- 与 task1/task2 无依赖（同层并行）；后端契约已在 plan 冻结（`/humans/tree|file` 与 knowledge 同构）

## 实现要点

- **`KnowledgeBrowser` 是交互镜像模板**（`web/src/components/knowledge/KnowledgeBrowser.tsx`：`api.tree`/`api.file` + 左树右内容 + `MarkdownView`）——新建 `HumansBrowser` 适配 humans API，**不修改** `KnowledgeBrowser`（避免回归）
- **草稿键**：`rick-web-draft:<sessionId>`（前缀常量定义在 ChatInput 或 lib）；发送成功路径已有多处 `setText("")`（ChatInput.tsx:126/136/166/198）——统一在清空处同步 `localStorage.removeItem`
- 已有 localStorage 先例可参照：`web/src/api/sse.ts`（游标）、`routes/Settings.tsx`（token/starfield）
- 移动端：humans 页面双栏在小屏退化为树/内容切换或上下布局（与 KnowledgeBrowser 现状一致即可）
- `frontend_reload`（`sse.ts:427` `location.reload()`）导致的重载由 localStorage 恢复覆盖，无需改 sse

## 测试方法

- `cd web && npm run typecheck && npm run build` 绿
- 断言存在：`Humans.tsx`、`HumansBrowser.tsx`、`App.tsx` 含 `humans`、`ChatInput.tsx` 含 `rick-web-draft` 与 `localStorage`
- 手工（dev 实例 8414）：①Humans 页面能列出 `loop_N`（mission/research/lessons/judgment）与 rfc 历史并打开阅读；②会话输入框输入文字 → 刷新页面 → 文字仍在；③发送成功后刷新 → 草稿已清除

## 禁止事项

- 不得实现阶段进展条 / 判断点卡片（human 明确不要）
- 不得修改知识库（KnowledgeBrowser）、jobs 页面既有行为
- 不得引入新依赖（复用现有组件库/样式变量）
- 不得修改后端（`internal/**`）——write 域仅前端
