# research-L3 简报：前端架构层调研（汇总索引）

元信息：L3 前端架构层 | 本简报为**汇总索引**——第 3 层（前端架构）的全部事实性问题已在前两轮调研中消解，本文件按层索引归档，供门禁与后续实现流水线引用。

## 事实性结论（来源：research-L1-r2.md + 叶子文件 + research-L2.md，均同目录在案）

1. **交互模式共识**（来源 research-L1-r2-leaf-4，置信 0.75）：新建会话「参数作为会话属性」优于前置表单门槛；resume=列表→按 id 重放（与 pi get_entries(since) 游标天然对齐）；渲染分层=文本 delta 直渲（Zustand 批处理防卡顿）+工具折叠卡+thinking 折叠+错误二分+流式与终稿合并去重；传输=可恢复 SSE（POST 发起+GET EventSource，断线指数退避）
2. **移动端适配**（同上）：断点共识 768px/1024px；移动端侧栏折叠 drawer；输入区常驻底部（fixed+VisualViewport 处理软键盘）；Tailwind 断点须与 JS 断点对齐
3. **组件复用源**（来源 research-L2，置信 0.85）：ygncode/pi-web（Svelte，Go 侧为主力复用）；pi-web-ui（React 18 组件族：ChatInput/ToolCallBlock/ThinkingBlock/BashBlock/MessageList，MIT，换 SSE 客户端即可移植）；pi-agent-dashboard（React 19 chat-embed 子路径，需 git vendor+锁版本 ~24 依赖）——React 路线组件供给面最广（human 已裁决 React）
4. **渲染栈推荐**（来源 research-L2，置信 0.85）：react-markdown + remark-gfm + rehype-highlight(highlight.js) + ansi-to-react + @git-diff-view/react + dompurify；暂不引 CodeMirror/Monaco（无编辑需求）
5. **embed 构建方案**（来源 research-L2，置信 0.9）：ygncode 的 Vite manifest + go:embed 方案（web/dist + .vite/manifest.json + assets.go）可直接移植
6. **PWA**（来源 research-L2 ygncode 先例）：vite-plugin-pwa 成本低；service worker 需 localhost/HTTPS 安全上下文，LAN http 降级普通网页
7. **主题实现载体**：CSS 变量 token + keyframes 动效（星空/流星/传送门旋涡）；prefers-reduced-motion 降级为静态背景（W3C/MDN 惯例）——R&M 画风为 human 设计指令（Q19），非调研事实

## 层内判断节点（均已 human 裁决，见 design-tree.md L3 终判）

- UI 信息架构=A（侧栏+功能区）；主题=R&M 深色画风（动态星空/传送门/飞碟）；PWA=做；技术栈=Zustand+react-router+Vite+Tailwind 4；并发互锁=不做；分发=C 混合+自迭代

## 结论

第 3 层全部事实性问题已消解（调研在案：research-L1-r2-leaf-4 / research-L2.md 及其叶子），判断节点全部上呈 human 并裁决完毕。
