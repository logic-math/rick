# research-L5-leaf-1 —— dev 构建与依赖约束（实测）

红线遵守：未改仓库被跟踪文件（实验全在 /tmp 拷贝）；未动 8413 生产进程；未写 ~/.rick；无 git 写操作。

环境：node v24.16.0 / npm 11.13.0；PATH 上的 `go` 是 **go1.22.4**，仓库 go.mod 要求 `go 1.25.0`（GOTOOLCHAIN=auto 自动切 1.25.0，工具链 214MB 已在 `~/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.25.0.linux-amd64`）；GOPROXY=`https://proxy.golang.org,direct`；npm registry=`https://registry.npmjs.org/`（npm ping PONG 533ms）；仓库无 `vendor/`。

## 实验表

| # | 实验 | 命令（原文摘） | 实测 | 结论 |
|---|---|---|---|---|
| 1 | embed 缺 dist 的编译行为 | `/tmp/embedtest` 最小模块（复刻 web/embed.go 的 4 条 directive）；有 dist 时 `go build ./...`；`rm -rf web/dist` 后重跑 | 有 dist：exit 0（0.045s）。无 dist：`web/embed.go:10:12: pattern all:dist: no matching files found`，**exit 1** | `//go:embed all:dist` 要求 dist 目录至少 1 个文件；缺失是**编译期硬失败**，无占位/降级逻辑 |
| 2 | 仓库 dist 是否入库 | `git ls-files web/dist \| wc -l`；`find web/dist -name '*placeholder*' -o -name '.gitkeep'` | 12 个产物文件入库（index.html/sw.js/assets/*）；无占位文件 | 新 clone 可直接 `go build`（dist 已提交）；一旦删除或加进 .gitignore 立刻编译失败 |
| 3 | go build 计时 | `time go build -o /tmp/rick-timing ./cmd/rick`（main 在 cmd/rick） | 热（默认 GOCACHE）**0.44s**；冷（新 GOCACHE + `GOPROXY=off`）**6.70s**；冷后二次 0.41s | 重建成本秒级，可纳入 dev 循环；离线（GOPROXY=off）在缓存热时可行 |
| 4 | npm run build 计时/产物 | `cp -a` web 的 src/public/configs → /tmp/webcopy，`ln -s` 仓库 node_modules，`time npm run build` | 3.7s（vite built in 2.36s）；产出 dist/ 8 个根文件 + assets 2 个（js 1,700,427B / css 75,219B）；引用 `assets/index-B0FGmfnL.js`+`index-DHv2zk82.css` | vite build 秒级；**与仓库 dist hash 不一致**（见 #5） |
| 5 | 追查 hash 差异（重要） | 把仓库 `web/dist` 放进 /tmp/webcopy 构建树后重跑两次 | 两次都复现仓库 hash `index-C3ifqTyg.js`+`index-BWKqsxDY.css`（与 ~/.rick/web/dist 的 md5 一致）；唯一规则数 976（含 dist）vs 901（无 dist），`only-in-with-dist` **80 条**（`.align-middle`/`.basis-\[50\%\]` 等），JS 体积两者均 1,700,427B | 根因：`web/dist` 入库且**未被 web/.gitignore 忽略**，Tailwind v4 自动源扫描把旧 dist 里的类名也算进 CSS；CSS hash 变→Vite 把 CSS 依赖名内联进 entry JS→JS hash 也变。**必须就地构建（web/ 内、dist 在场）才复现提交/生产产物**；干净副本构建=功能等价但 hash 不同 |
| 6 | npm ci 是否要网 | /tmp/webcopy2：`npm ci`（热 cache）→ 2.74s；`npm ci --cache /tmp/npmcache-cold`（冷 cache，联网）→ **8.71s**，node_modules 171MB / cache 37MB；`npm ci --offline --cache <空>` → `ENOTCACHED ... zwitch-2.0.4.tgz` exit 1；`npm ci --offline --cache <上文冷cache>` → 2.2s exit 0 | 要网（或预热 cache）；本环境冷装 8.7s 可接受；无网环境只能复用已有 `web/node_modules`（**187MB / 353 包**） |
| 7 | vite build 自身是否要网 | 上述构建全程只用软链 node_modules，PWA 插件本地生成 sw.js/workbox-9c191d2f.js | 无下载、无报错 | node_modules 就位后 vite build 完全离线可跑 |
| 8 | Go 工具链/cold 机器 | `env GOMODCACHE=/tmp/emptymod GOPROXY=off go build ./cmd/rick`；`... GOTOOLCHAIN=local`（go1.22.4） | 前者：`go: download go1.25.0 for linux/amd64: toolchain not available` exit 1；后者：`go.mod requires go >= 1.25.0 (running go 1.22.4)` | 真冷机（无网）无法构建：需 214MB 工具链 + cobra/goldmark 模块，且无 vendor |

## 结论（3 条）

1. dist 入库是构建硬前提：删/忽略 `web/dist` → `go build` 直接编译失败（非运行期降级），且无占位文件。
2. 前端务必**就地**在 `web/` 内构建（dist 在场）：否则 hash 改变、CSS 少 80 条工具类（Tailwind 会扫旧 dist）。
3. go/npm 都能离线，但前提缓存已热：工具链 1.25.0 + 模块 + npm cache；本环境冷 `npm ci` 8.7s、冷 go build 6.7s。

## 待验证点（confidence）

- Tailwind 扫描范围是否还包含其它未忽略目录（如 `web/dist.old/` 已被 .gitignore，实测未验证扫不扫）——confidence 0.6。
- 「删 dist 也能编译」的设想需新增占位文件（如 `web/dist/.gitkeep`）并改 embed 指令，属代码改动，未做——confidence 1.0（现状不可行）。
- 冷 go build 6.7s 的计时前提是工具链/模块已缓存；真冷机需再加工具链下载（本机代理带宽很高，未隔离测量）——confidence 0.5。
- npm ci 8.7s 依赖本环境镜像带宽，公网/慢网可能显著更慢——confidence 0.4。
