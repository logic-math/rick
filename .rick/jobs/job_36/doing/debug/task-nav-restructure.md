# 依赖关系
无

# 写域
web/src/App.tsx
web/src/routes/Sessions.tsx
web/src/routes/Jobs.tsx
web/src/routes/Knowledge.tsx（改名 Dreams.tsx）
web/src/routes/Dreams.tsx
web/src/index.html
web/public/favicon.svg
web/src/components/layout/（新建，若拆组件）

# 任务目标
导航重构 + 品牌改名 + favicon：
1. 侧边栏统一为「工作区树」：每个工作区节点展开其子项——Sessions（对话视图）、Jobs（job 视图）、Dreams（原 Knowledge 改名）——三页都挂在工作区下（路由 /ws/:wsId/sessions、/ws/:wsId/jobs、/ws/:wsId/dreams）；移除顶部 Sessions/Jobs/Knowledge 三个全局 NAV；保留 Settings 齿轮（顶栏）。
2. 路由结构：/ 重定向到第一个工作区的 sessions（无工作区则引导添加）；/ws/:wsId/sessions|jobs|dreams 三个视图；/session/:id 不变（跨工作区直达）。
3. Dreams 视图 = 原 Knowledge 内容（domain/loops/skills 文件树浏览）改名 + **新增 dream 日志浏览**（.rick/dream/ 下 dream_run_*_log.md 文件列表与内容——后端接口如已有则复用，否则走 knowledge 根扩展或 job 文件接口；若后端无 dream 浏览接口，前端先展示 domain/loops/skills + 标注 dream 日志待后端补）。
4. 品牌：左上角 logo 文本 "rick web" → "rick"；顶栏中间文本 "rick web" → "rick"；index.html <title> → "rick"。
5. favicon：web/public/favicon.svg（绿色传送门 SVG——同心椭圆旋涡 + 高光，SMIL animateTransform 旋转动画，静态兜底同样美观）；index.html <link rel="icon" type="image/svg+xml" href="/favicon.svg">。
6. 工作区树的交互：展开/折叠（默认展开）；工作区项点击展开子项而非跳转（或整体作为链接）；当前激活的子项高亮（portal 绿）。
7. 移动端：drawer 保持；工作区树同样适用。
8. npx tsc --noEmit + npm run build 通过；playwright 冒烟（可在 6173 验证实例上跑——HOME=/tmp/rick-e2e-home 有工作区 test）。

# 测试方法
tsc + build 全绿；playwright 验证：侧边栏显示工作区树（test 工作区下有 Sessions/Jobs/Dreams 三子项）、点击切换路由、Dreams 页显示知识树、品牌名 rick、favicon 请求 200。

# 上下文提示
- 现路由：App.tsx 294-308 行（/ /jobs /knowledge /settings /session/:id）；NAV_ITEMS 常量 40 行
- Knowledge.tsx 现为「工作区选择器 + KnowledgeBrowser」——重构后由路由提供工作区上下文（/ws/:wsId/dreams），页面直接渲染该工作区的知识树，去掉内部选择器
- Jobs.tsx 同款改造（去掉内部选择器，用路由 wsId）
- 后端改动在并行 task-B（写域 internal/web/），本 task 不碰后端——若需要 dream 日志或 prompt 接口，先按契约存在调用（不存在则功能留空标注）
- R&M 主题 token 在 web/src/styles/theme.css（--rm-portal 等）
