# task3：导航实时性（前端路由层）

## 任务目标

左侧导航点击后立即实时切换：切会话立即显示目标会话内容、路由切换重置滚动、工作区行点击即跳转。

## 关键结果

- KR2a：`SessionPage.tsx` 给 ChatView/MonitorView 加 `key={session.id}` 强制重挂——修复主根因（ChatView epoch 守卫不含 sessionId 导致切 A→B 时 B 的历史永不拉取、界面保留 A 的消息）
- KR2b：`App.tsx` 路由切换重置主滚动容器 scrollTop（location.pathname 变化时归零）
- KR2c：`WorkspaceListNode.tsx` 行点击改为**直接导航**到该工作区 sessions（useNavigate）；展开/折叠改由箭头/chevron 图标承担（stopPropagation）
- MonitorView 本身无此 bug（按 sessionId 依赖正常），key 重挂对其无害

# 写域

- web/src/routes/SessionPage.tsx
- web/src/App.tsx
- web/src/components/layout/

# 依赖关系

无

## 实现要点

- 主根因证据：`SessionPage.tsx:59-62` 无 key；`ChatView.tsx:319` epoch 守卫 `if (historyEpochRef.current === epoch) return`，epoch 不含 sessionId——切会话时 effect 重跑但 epoch 相同提前 return
- `key={session.id}` 一行修复：epoch 守卫、history state、refs、滚动全部随组件重挂重置
- 滚动容器是 `<main>`（`App.tsx:310` 一带）——用 `useEffect` on `location.pathname` 把 `mainRef.current.scrollTop = 0`（或 scrollIntoView）；React Router v6 `useLocation`
- 工作区行现状：`WorkspaceListNode.tsx:100-135` 点击=展开/折叠；改后行点击 navigate(`/ws/${wsId}/sessions`)，chevron 按钮 onClick stopPropagation
- 注意保持现有默认折叠行为与键盘可达性（chevron 加 aria-label）

## 测试方法

- `cd web && npm run typecheck && npm run build` 绿
- 代码级断言（gate1）：SessionPage 含 key={session.id}；App.tsx 含 pathname 滚动重置；WorkspaceListNode 含 navigate( 行点击 + chevron stopPropagation
- 手动路径（dev UI 8414）：会话 A ↔ B 切换内容立即正确；长页切短页滚动归零；侧栏点工作区行直接跳转该工作区 sessions

## 禁止事项

- 只改写域 3 处；不碰 starfield/ChatView/events（task4 域）；不执行任何 git 操作；一切改动只在 dev 树
