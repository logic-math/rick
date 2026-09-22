# research-L4-leaf-4：前端 overlay 与 vite 开发链隔离（Q6）

只读考察；未触碰生产（172761/8413/prod HOME）；本叶子无 /tmp 残留。

## 1 overlay 解析链
- StateDir=`web.WebStateDir()`=`$HOME/.rick/web`：internal/cmd/web.go:122、internal/web/web.go:25-31
- overlay=`StateDir/dist`：server.go:66-68（装配）、server.go:154-158（watcher）
- 注入：server.go:73；生效判定=`Lstat(dir/index.html)`：static.go:67-72；all-or-nothing（缺 asset→404，非 asset→SPA）：static.go:74-87、95-112、117-142
- embed 兜底=编译期 DistFS：cmd/web.go:121 `webembed.DistWeb()`、web/embed.go:31-41

⇒ overlay 只由 HOME 决定，**dev 独立 HOME 即自然隔离**；共享 HOME 则两实例服务同一目录，dev 可改写生产 UI。
风险：overlay 半拷贝（缺 assets/）=新 index.html + 资源 404 → 白屏（static.go:74-87）。

## 2 scaffold（env）
- DeployWebScaffold→`$HOME/.rick/web`（硬编码）env/web.go:32-38,51-56；源=SrcFS 且**不含 dist** env/web.go:68、embed.go:26-29；marker `.rick-managed` 幂等 env/web.go:57-62,72-74
- ResetWebCustomization 只保留 sessions.json env/web.go:92-105
- 实测 `~/.rick/web/`：仅 archived.json/dist/job-names.json/sessions.json，**无 src、无 marker**；overlay dist 与仓库 web/dist sha256 相同（index.html `d03b90a1…`，assets 同名 `index-C3ifqTyg.js`）

⇒ 现网 dist 由 `~/.rick/deploy-web.sh`（npm build+cp）产出，customize 从未跑。
**发现(medium)**：reset 会连带删 `archived.json`/`job-names.json`，与文档 cmd/web.go:186-189 不符。

## 3 vite 链
- proxy 硬编码 `http://127.0.0.1:6137`：vite.config.ts:57-60（6137=flag 默认 cmd/web.go:65，**≠生产 8413**）
- outDir="dist"、manifest:true：:53-56；PWA devOptions.enabled=false：:48-50；**无 base**（默认 /，各端口根路径不冲突）
- 前端无硬编码后端：client.ts:88 `window.location.origin`、sse.ts:105,381-382 相对 `/api/events` ⇒ 同源
- package.json：dev=vite/build=vite build/typecheck；vite ^6.3.5、React18、无 engines；node v24.16.0 / npm 11.13.0 满足

⇒ 唯一改动点：`"/api": process.env.RICK_DEV_API ?? "http://127.0.0.1:6137"`；dev 态 `RICK_DEV_API=http://127.0.0.1:<devport> npm run dev`（5173，HMR 免构建免 overlay）。

## 4 frontend_reload
- watcher 轮询**本实例** distDir（2s、debounce500ms、>5s 强制 flush）：watcher.go:50-99（publish 76,95）；distDir=自身 cfg.StateDir/dist server.go:154-158；事件 sse.go:18,409-413；消费 sse.ts:422-424 `location.reload()`，seq 去重 sse.ts:53-54,335-342

⇒ 不同 HOME：互不触发；共享 HOME：dev 构建 → 两实例各自广播 → **生产全部浏览器重载**（后端 job 不受影响）。

## 5 入库与生效
- `git ls-files web/src` = **66**（已跟踪）→ dev worktree 可直接改源码
- `git ls-files web/dist` = **12**（已跟踪，embed.go:31-32 `//go:embed all:dist`）→ dev 内 `npm run build` 会让 worktree 变脏
- 改 src 后必须 build（生效只读 dist）：①重新 `go build` ②拷 dist 到 overlay（static.go:74-77）

## 6 结论：最小改动清单
- **无需改**：overlay 路径、embed 兜底、缓存策略、SSE 同源、PWA
- **需改**：仅 vite.config.ts:57-60 proxy target 环境变量化（或只在 dev worktree 手改，不入库）
- **dev 启动**：`HOME=$DEV_HOME ./bin/rick_dev web --port <devport> --listen 127.0.0.1`；UI 迭代用 `npm run dev`
- **残留泄漏点**：①共享 HOME 时 overlay 无锁无实例标记，dev 一构建即刻改生产 UI ②overlay 半拷贝→404 白屏 ③在生产 HOME 跑 `rick web reset` 会删 archived/job-names ④proxy 硬编码 6137 既不等于生产 8413，默认使用会指向无实例端口
- **跨叶子**：dist 入库使 dev worktree 的 `npm run build` 产生 git 脏状态（Q3/leaf-1 需覆盖）
