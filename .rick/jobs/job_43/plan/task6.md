# task6：移动端性能二轮（动画降级 + blur 补漏 + 节流降频）

## 任务目标

消除 5.0.12 后残留的「会话活跃期」发烫源：空态飞船常驻动画全局静态化；工具运行期动画/状态点脉冲/秒表 tick 移动端降级；补漏第 5 处 blur。

## 关键结果

- KR7a：**空会话空态飞船动画全局关闭**（MessageList.tsx:246 `<Saucer flying>` → 静态，flying=false 或等价）——新开会话停在空态不再持续烧
- KR7b：**工具运行期 Saucer 动画移动端关闭**（SteerBar.tsx:54 / MessageList.tsx:54 / MonitorView.tsx:176 的 flying 动画，桌面保留）——推荐手法：theme.css 加 `@media (max-width:767px)` 下 rm-saucer-hover/rm-beam/SMIL 等价降级（animation-duration≈0），或组件级窄屏判定条件渲染 `<animate>`
- KR7c：**第 5 处 blur 补漏**：MessageList.tsx:284 展开工具条 `bg-surface-raised/90 backdrop-blur` 加移动端不透明+md: 恢复（同首轮手法）；App.tsx:114 断线横幅与 FileReaderPanel.tsx:92 移动端阅读器同样处理
- KR7d：**状态点脉冲移动端降级**：rm-pulse（ChatView.tsx:83/93 活跃/流式呼吸、SessionBadge.tsx:90、WorkspaceListNode.tsx:125、StatusDot.tsx:55、StreamText.tsx:23 流式光标）窄屏不跑（静态色点仍显示状态）——推荐统一 CSS 媒体查询降级
- KR7e：**ToolElapsed 移动端降频**（ToolCallCard.tsx:138-147 useTick）：窄屏 1s → 5s（保持功能：耗时数字仍更新）
- 桌面视觉零回归（桌面动画全保留）

# 写域

- web/src/components/chat/MessageList.tsx
- web/src/components/chat/SteerBar.tsx
- web/src/components/chat/ToolCallCard.tsx
- web/src/components/chat/StreamText.tsx
- web/src/components/chat/ChatView.tsx
- web/src/components/monitor/MonitorView.tsx
- web/src/components/sessions/SessionBadge.tsx
- web/src/components/layout/WorkspaceListNode.tsx
- web/src/components/common/StatusDot.tsx
- web/src/components/starfield/Saucer.tsx
- web/src/components/files/FileReaderPanel.tsx
- web/src/styles/theme.css
- web/src/App.tsx

# 依赖关系

无

## 实现要点

- 优先**CSS 媒体查询统一降级**（theme.css 已有 prefers-reduced-motion 同款块可参照）：`@media (max-width: 767px)` 下关闭 rm-saucer-hover/rm-beam/rm-pulse 等装饰动画（animation: none / duration 0.01ms）——组件改动最小化；SMIL `<animate>` 不受 CSS 管，需 Saucer.tsx 条件渲染（窄屏判定已有先例：ChatInput 用 matchMedia，starfieldMode.ts 有 NARROW 判定模式）
- 空态飞船是**全局**关闭（不限移动端——human 裁决：空态本来就静，视觉损失≈0）
- 断线横幅（App.tsx:114）只在 reconnecting/closed 显示——blur 降级同 md: 手法即可
- rm-chat-in（0.15s 单次入场动画）无需处理（非循环）
- 各 rm-* 动画清单见 theme.css:110-153

## 测试方法

- `cd web && npm run typecheck && npm run build` 绿
- gate3 静态断言：空态 Saucer 非飞行动画、theme.css 有 767px 媒体查询降级块、MessageList 展开条 md: blur、ToolElapsed 窄屏降频
- 手动路径（dev UI 8414 移动视口）：新会话空态不烧（无动画）；跑工具时飞船静止；展开条不透明；桌面视口动画全保留

## 禁止事项

- 不做虚拟化（task7 域）；不动 ChatInput.tsx（task5 域）；不做任何 git 操作；只在 dev 树
