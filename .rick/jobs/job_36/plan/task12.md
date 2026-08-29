# 依赖关系
task9, task10

# 写域
web/src/routes/
web/src/components/sessions/
web/src/App.tsx
web/src/main.tsx
web/src/styles/
web/public/
web/vite.config.ts

# 任务目标
前端组装：路由/页面/会话创建流（cmd 参数弹窗）/Settings（token+Customize/Reset）/PWA 激活/响应式收口。

# 关键结果
1. `web/src/App.tsx`+`main.tsx`：react-router 路由树——`/`（Sessions 默认页）`/jobs` `/knowledge` `/settings` `/session/:id`（ChatView 或 MonitorView 按 type 分派）；TopNav（三功能区+Settings 齿轮；连接状态点=SSE 状态：绿=连接/黄=重连中/红=断开）；Sidebar（工作区列表+每工作区会话列表+「添加工作区」「新建会话」按钮；<768px 折叠 drawer 汉堡）
2. `web/src/components/sessions/NewSessionModal.tsx`：新建会话弹窗——选工作区（下拉）→ 选 cmd 类型（卡片组：plan/easy/doing/ctrl/human-loop/learning/dream，各带图标+一句话说明）→ 动态参数表单（按 type：plan=requirement 多行+job 可选；easy=requirement+ctx_path 可选；doing/ctrl/learning=job 下拉（该工作区 jobs 列表拉取填充）+human-loop=topic+dream=job_num 数字+mode 单选 interactive/background）→ 提交 createSession API→ 跳转 /session/:id；参数校验前置（必填缺失禁用提交）
3. `web/src/routes/Sessions.tsx`：工作区+会话总览页（Sidebar 主内容化：每工作区区块含会话卡片列表+新建按钮；closed 会话卡显示 Resume）；`Jobs.tsx`：工作区选择器+JobsList/JobDetail；`Knowledge.tsx`：工作区选择器+KnowledgeBrowser；`Settings.tsx`：token 设置（输入+保存 localStorage+测试连接按钮调 /api/config）、Customize UI 按钮（调 POST /api/web/customize——task13 契约补充项：返回 {scaffolded: bool}；提示「已就绪，可在任意会话中让 agent 修改 ~/.rick/web/src 后构建」）、Reset to baseline 按钮（确认弹窗→POST /api/web/reset）、服务器信息展示（version/rick_version）
4. PWA 激活：`web/vite.config.ts` 启用 vite-plugin-pwa（manifest：name "rick web"、theme_color #0b0e14、background_color #0b0e14、icons=public/ 下 192/512 png——传送门绿飞碟简图标（SVG 转 png 或占位纯色）；registerType autoUpdate；workbox runtimeCaching 只缓存 /assets/）；`web/public/manifest.webmanifest` 若插件生成则不手写
5. 响应式收口：768/1024 两档断点全页面过一遍（Tailwind md:/lg: 与 JS useMediaQuery 对齐——工具函数放 stores/ui.ts）；移动端：Sidebar drawer/聊天输入 fixed 底部+VisualViewport/表格横向滚动
6. 首次体验：无 token 时全屏锁页（Portal 旋涡+token 输入框——api client 的 authRequired 信号）；无工作区时 EmptyState 引导添加
7. `npx tsc --noEmit` + `npm run build` 全绿；`npm run build && ls web/dist/` 产出完整（index.html+assets+manifest+sw.js）

# 测试方法
tsc+build 全绿；npm run dev 手工全流程：锁页→输 token→添加工作区→建 plan 会话（无后端时 mock stores）→各页面移动视口（devtools 375px/768px/1440px）目检

# 上下文提示
- api-contract.md 有一处本 task 新增依赖：POST /api/web/customize 与 /api/web/reset（Settings 按钮）——已在 task13 的写域内（routes.go 挂载），先在此声明契约（响应 {ok:true, scaffolded?:true}）
- 图标：不引图标库，SVG 手写/内联（保持轻量）；R&M 元素复用 task3 的 Portal/Saucer
