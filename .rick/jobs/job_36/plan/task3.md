# 依赖关系
无

# 写域
web/
Makefile

# 任务目标
前端骨架 + Rick & Morty 深色主题体系 + embed 接入：React SPA 的可构建基座（Vite 构建链走通 + dist 占位可 embed）。

# 关键结果
1. `web/package.json`：deps=react18/react-dom/zustand/react-router-dom/markdown 渲染栈（react-markdown remark-gfm rehype-highlight ansi-to-react @git-diff-view/react dompurify——按 api-contract 同级的 design-tree M6 清单）；devDeps=vite/@vitejs/plugin-react/typescript/tailwindcss@4/@tailwindcss/vite/vite-plugin-pwa（PWA 配置本 task 只装依赖不激活，task12 激活）
2. `web/vite.config.ts`：plugins = `react() + tailwindcss()`（@tailwindcss/vite）+ `build.outDir=dist` + dev proxy（/api → http://127.0.0.1:6137）+ manifest 输出（.vite/manifest.json 供后端 asset 映射——参考 ygncode 方案）
3. `web/tsconfig.json`、`web/index.html`（`<div id="root">`）、`web/src/main.tsx`、`web/src/App.tsx`（临时占位路由：三个空页 Sessions/Jobs/Knowledge + 左侧栏/顶部导航布局壳）
4. **R&M 主题体系** `web/src/styles/theme.css`：CSS 变量 token——传送门绿 `--rm-portal:#39ff88`、深空底 `--rm-space:#0b0e14`/`--rm-space-2:#1a1c2e`、星云紫 `--rm-nebula:#5d3fd3`、Rick 蓝灰 `--rm-rick:#a6c8dd`、Morty 黄 `--rm-morty:#ffd54a` + 语义 token（bg/surface/border/text 主次/成功/失败映射到上述）；Tailwind 4 通过 `@theme` 引用同名变量
5. **动态星空背景** `web/src/components/starfield/StarfieldBackground.tsx`：canvas 实现静态星点（数量~120、大小/亮度随机、视口 resize 适配）+ 闪烁动画（requestAnimationFrame 或 CSS opacity 交替）+ 偶发流星（每 6-15s 一颗，斜线拖尾渐隐）；`Meteor.tsx` 可选拆分；`prefers-reduced-motion: reduce` 时降级为静态星点无动画；性能：单 canvas、动画帧率限 ~30fps、页面不可见时暂停（visibilitychange）
6. `web/src/components/starfield/Portal.tsx`：绿色传送门组件（CSS/SVG 同心圆旋涡 + `@keyframes` 旋转），props: size/loading——本 task 作为布局壳 logo/加载指示占位；`web/src/components/starfield/Saucer.tsx`：飞碟 SVG 组件（占位，task10 用作 job 运行指示）
7. `web/embed.go`：`package web` + 多 directive 累积导出（本机实证可编译可读）：
   ```go
   //go:embed all:src
   //go:embed package.json vite.config.ts tsconfig.json index.html
   var SrcFS embed.FS
   //go:embed all:dist
   var DistFS embed.FS
   ```
   （根配置文件必须在 SrcFS——customize 抽取后要能 npm build）；`web/dist/index.html` 占位文件（保证空 dist 可 embed 不报错）；exported as fs.FS via `fs.Sub` 变量或直接 embed.FS
8. `Makefile` **新建**（仓库现无 Makefile）：`web-dist` 目标（cd web && npm ci && npm run build）与 `web-dev`（npm run dev）；README 不动（task15 统一写文档）
10. `web/.gitignore`：内容 `node_modules/`（node_modules 绝不入库——否则层提交 git add -A 吞数百 MB；dist 与 package-lock.json 正常入库供 npm ci）；验证 `git check-ignore web/node_modules` 命中
9. 验证：`cd web && npm install && npm run build` 成功产出 `web/dist/index.html` + `web/dist/assets/`；`go build ./...` 成功（embed 生效）；`go vet ./web/` 无告警

# 测试方法
npm run build 成功 + go build ./... 成功 + 人工浏览器预览（npm run dev 看星空动效/主题色/布局壳）；星场 reduced-motion 降级可用 devtools 模拟验证

# 上下文提示
- node ≥22.19 已是环境依赖（pi 要求），npm 可用；npm install 用国内 registry 不需要（默认源可用）
- dist 提交仓库是 rick 哲学（node 不进 rick 构建链）——本 task 产出的 dist 后续随源码提交
- 主题灵感：Rick and Morty（飞碟/传送门/星空），深色默认；不要引入组件库（antd/mui 等）——组件全自建，保持轻量
- ⚠️ embed 语法：`//go:embed all:dist`（all: 前缀包含下划线/点开头文件）；embed 的 src 排除 node_modules（src/ 内不会有）
