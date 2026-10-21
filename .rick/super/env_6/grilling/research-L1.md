# research-L1：无头浏览器验收本地 web 应用的工具选型（env_6 grilling L1 简报）

- 派发：super env_6 grilling L1（设计树 M4/KR4 前置，设计树 §L1 已记录派发）
- 日期：2026-10-20
- 方法：本机实测（所有命令可复现，见附录）+ 官方文档/issue/commit 溯源 + CDN HEAD 实测下载量
- 信源等级：**A** = 本机实测；**B** = 官方文档 / 一手 issue/commit / npm/PyPI 元数据实测；**C** = 社区二手（仅佐证）
- 置信度：**高** —— 主推荐路径已在本机对目标服务（127.0.0.1:6137）端到端实测通过（含截图与 console 捕获）

---

## TL;DR（结论速览）

1. **推荐：Node 版 Playwright 1.64.1 + `channel:'chrome'`（复用本机已装 Chrome 155.0.8059.40）**
   - `npm i playwright` = 18MB node_modules（playwright-core 13M + 壳 4.9M），**零浏览器下载**，只依赖 npm registry（实测可达）
   - 已实测全链：launch 216ms → goto 45ms → React 渲染（#root 10756 字符，title "rick"）→ 截图 viewport 1440×900（58KB PNG）+ fullPage → console 错误捕获（4× 401 Unauthorized）→ **总计 1.47s**
2. 回退路径（按足迹递增）：`npx playwright install chromium --only-shell`（仅无头壳，98MB zip）→ 完整 `install chromium`（CfT 185MB + 无头壳 98MB + ffmpeg 1MB）
3. **否决 Python 版 playwright**：1.64.0 `requires_python >= 3.10`（PyPI 元数据实测），本机系统 Python 3.9.6，还得另装解释器 + venv —— 对一次性脚本是最重路径
4. **否决 safaridriver**：Safari WebDriver **无 headless 模式**（Selenium issue #5985 已关），需一次性 `sudo safaridriver --enable`，且要另上 Selenium 生态
5. **否决 Chrome CLI（--dump-dom/--screenshot）作为断言引擎**：本机实测 http(s) 目标一律 dump 出 40 字节空 DOM（example.com / 127.0.0.1:6137 / 本地 python http.server 三处复现，exit=2 + "Page load timed out"）；`--virtual-time-budget` 挂死 >90s；`--screenshot` 只出 4.7KB 空白帧（对照 Playwright 同页 58KB）；file:// 对照正常 → **CLI 无头管线在本机不可用**
6. **⚠️ 对任务背景的重要校准（供 J1/J3 裁决）**：背景称「loopback 默认免 token 认证」——**实测不成立**。6137 实例 `/api/sessions`、`/api/events` 均返回 401。token 解析链 = `--token` flag > config `web_token`（真 home 的 `~/.rick/config.json`，实测该键非空）> 自动生成写回并仅打印一次；**只有显式空 token 才完全免认证**（`internal/web/auth.go:31`）。验收脚本必须做 token 注入（见 §4.4）
7. 等待策略：`goto` 用 `domcontentloaded`（实测 45ms）+ 元素断言（自动重试）；`networkidle` 在 1.64.1 已修复 SSE 豁免（发布源码验证 + 实测 586ms 通过），但不作主等待原语

---

## §1 本地事实基线（全部 A 级实测）

| 项目 | 值 | 备注 |
|---|---|---|
| 服务 | `http://127.0.0.1:6137` curl 200 | 进程：`/Users/logic/bin/rick web --listen 0.0.0.0 --port 6137`（无 --token → 用 config token） |
| 鉴权 | `/api/sessions`、`/api/events` 均 **401** | `{"error":{"code":"unauthorized",...use Authorization: Bearer or ?token=}` |
| 前端 | React 18 SPA + Vite；`#root` 挂载；REST `GET /api/sessions` 引导，SSE `GET /api/events?token=` | `web/src/api/client.ts:228`、`web/src/api/sse.ts:105` |
| SSE 行为 | 15s 心跳注释行；断线指数退避重连；单流多路复用；**无 token 时也连接（401 → 报错 → 退避重连）** | `web/src/api/sse.ts`、`web/src/main.tsx:9-13` |
| token 前端契约 | `localStorage["rick-web-token"]` → REST Bearer；SSE `?token=`；TokenGate 表单保存后**整页刷新** | `client.ts:50`、`components/sessions/TokenGate.tsx` |
| 已装浏览器 | Google Chrome **155.0.8059.40**；safaridriver 27.0.1（系统自带） | 无 chromium/firefox；无 playwright/puppeteer 缓存 |
| 运行时 | node v26.11.1 / npm 11.20.0 / python3 **3.9.6**（无 playwright 模块） | web/node_modules 存在，无任何 e2e 依赖 |
| 网络 | npm registry 200；cdn.playwright.dev 可达（HEAD 实测有 Content-Length） | **系统代理 127.0.0.1:7890（HTTP/HTTPS/SOCKS），例外表含 `127.*`/`localhost`** —— loopback 直连不受代理影响（实测 Chrome 正常加载 6137，无需任何代理参数） |

**对验收脚本的关键含义**：① 鉴权开启 → 脚本需注入 token（§4.4）；② SSE 常开连接 → 等待原语必须元素级（§4.1-4.2）；③ 代理存在但 loopback 豁免 → 无需处理。

---

## §2 四方案能力对比（调研清单 1）

验收需求映射：打开页面 / 元素断言（sessions 列表含某会话、状态字段）/ 等待动态内容（SSE 驱动）/ 截图留档 / console 错误捕获。

| 能力 | Playwright 1.64.1 | Puppeteer 25.14 | 直调 CDP | safaridriver |
|---|---|---|---|---|
| 元素断言 | ✅ locator + `expect()` 自动重试断言（toBeVisible/toHaveText） | ⚠️ waitForSelector/waitForFunction，无内建断言库（需手写轮询） | ❌ 全手写 Runtime.evaluate 轮询 | ⚠️ Selenium 显式等待，粒度粗 |
| 截图 | ✅ `page.screenshot({fullPage})`（本机实测 58KB PNG） | ✅ 同源能力 | ✅ `Page.captureScreenshot` | ✅ WebDriver screenshot |
| 等待动态内容 | ✅ 自动等待 + waitForFunction | ✅ waitForFunction | ❌ 手写 | ⚠️ 仅显式等待 |
| SSE 页面等待策略 | ✅ goto `domcontentloaded`（实测 45ms）；networkidle 已修复 EventSource 豁免（见下） | ⚠️ networkidle0 会永不触发（SSE 计入 inflight）；`networkidle2` 可容纳单条流（B 级文档 + C 级推断） | ⚠️ 自管（可监听 Network 事件自行豁免） | ❌ 无网络级等待概念 |
| console 错误捕获 | ✅ `page.on('console')`/`page.on('pageerror')`（实测捕获 4× 401） | ✅ 同名事件 | ✅ Log/Runtime 域 | ⚠️ 仅 logs 端点，弱 |
| 无头模式 | ✅（channel chrome 实测通过） | ✅ | ✅ | ❌ **Safari 无 headless**（Selenium #5985） |
| 安装对系统侵入 | 低（可零浏览器下载） | 中（默认自动下载浏览器） | 最低但要手写一切 | 需 `sudo safaridriver --enable` 一次性授权 |
| 一次性脚本工作量 | 小 | 中 | 大 | 大（+Selenium 栈） |

### §2.1 networkidle × SSE 的完整溯源（本简报最关键的技术事实）

- **B**：GitHub issue [microsoft/playwright#41513](https://github.com/microsoft/playwright/issues/41513)：页面保持 `EventSource`（SSE）长连接时 `networkidle` 挂起直到导航超时。
- **B**：修复 commit [c370acf](https://github.com/microsoft/playwright/commit/c370acfe5ab176187182eeeed526adb54c221d30)（PR #41514，2026-07-01 合入）："fix(networkidle): do not wait for open EventSource (SSE) connections"。
- **A（源码级验证）**：下载 playwright-core **1.64.1 发布 tarball**，`lib/coreBundle.js` 中确认修复已随 stable 发布：

```js
_isExcludedFromNetworkIdle(request2) {
  if (request2._isFavicon) return true;
  if (request2.resourceType() === "eventsource") return true;  // ← SSE 豁免已上线
  return false;
}
```

- **A（行为实测）**：对 6137（SSE 连接中）`waitUntil:'networkidle'` **586ms 正常 resolve**。
- **B**：**WebSocket 仍不豁免**（[issue #37226](https://github.com/microsoft/playwright/issues/37226)）——本应用用 EventSource 不受影响，但若未来换 WS 会踩坑。
- **结论**：即便已修复，官方与社区一致建议 networkidle 只作辅助；**主等待原语 = 元素级断言**（见 §4.1）。

### §2.2 Puppeteer 的 networkidle 语义（B 级）

`networkidle0` = 0 个 inflight 连接持续 500ms；`networkidle2` = ≤2 个。SSE 流的 CDP `loadingFinished` 永不触发 → 计入 inflight → **networkidle0 在 SSE 页面永不触发**（文档语义 + 推断，未实测）；单条 SSE 流可用 `networkidle2` 容纳，但规范做法同样是元素级等待。

### §2.3 safaridriver 细节（B 级）

- Selenium issue [#5985](https://github.com/SeleniumHQ/selenium/issues/5985)（已关）：Safari 无 `.headless()` 等价物；Apple WebDriver 文档亦无 headless 能力项。
- 需一次性 `safaridriver --enable`（sudo）；Playwright 根本不驱动 safaridriver（其 WebKit 是自带构建 = 又要下载）。
- 判定：四个维度（无头/等待/console/SSE）全不满足 → **排除**。

### §2.4 直调 CDP（B 级）

`chrome --remote-debugging-port` + WebSocket 直驱：能力上限最高（Page.captureScreenshot / Runtime.evaluate / Log.entryAdded 全可及），但选择器等待、断言重试、报告全部手写；连 chrome-remote-interface 也是 npm 依赖。Playwright 本身就是「带电池的 CDP 客户端」→ 一次性脚本无必要 → **排除**。

---

## §3 安装足迹（调研清单 2，macOS arm64 实测）

| 路径 | 网络下载 | 磁盘占用 | 评价 |
|---|---|---|---|
| **Node playwright + `channel:'chrome'`**（推荐） | npm registry ~18MB（playwright 4.9M + playwright-core 13M，实测 `npm i` 1s，缓存命中） | **18MB** node_modules | **零浏览器下载**；复用已装 Chrome 155 |
| Node playwright + `install chromium`（默认全量） | chrome-mac-arm64.zip **185MB**（194,831,853B，HEAD 实测）+ headless-shell **98MB** + ffmpeg 1MB | 解压后数倍（官方「few hundreds of MB」） | hermetic 但最重 |
| Node playwright + `--only-shell` | 仅 headless-shell **98MB** | ~98MB×2 量级 | 无头专用最省的 hermetic 回退（B 级：官方文档） |
| **Puppeteer 25.14（npm i 默认）** | 包壳 0.04M + **自动下载** CfT ~170MB(macOS) + chrome-headless-shell（[pptr.dev 安装指南](https://pptr.dev/guides/installation)） | >500MB | npm i 时**静默下载浏览器**是惊喜行为 |
| puppeteer-core + executablePath | ~6MB | ~6MB | 零浏览器下载但 channel/路径全手配，且无自动断言 |
| **Python playwright 1.64.0** | wheel 41MB（macosx arm64，PyPI 实测）+ 同一套浏览器 zip | 41MB + 浏览器 | **`requires_python >= 3.10`，本机 3.9.6 直接不符**（PyPI 元数据实测）→ 需另装解释器 + venv，对一次性脚本最重 |
| Chrome CLI | 0 | 0 | 但本机实测不可用（§5） |
| safaridriver | 0（系统自带） | 0 | 但无 headless（§2.3） |

补充实测细节（A 级）：
- `npx playwright install --dry-run chromium` 输出：CfT **156.0.8078.4**（playwright build v1248），落盘 `~/Library/Caches/ms-playwright/`（本机 HOME 即 `rick-dev-home`）；默认 `install chromium` 会同时装 full chromium + headless shell + ffmpeg 三个包。
- `--only-shell` 的正确用法：**不指定 channel 时**无头跑的就是 headless shell（官方文档）；指定 `channel:'chrome'` 则根本不需要任何下载。
- npm registry 与 cdn.playwright.dev 实测均可达；系统代理 7890 存在但 `127.*` 在例外表 → loopback 验收流量不走代理，实测无需任何代理配置。

**一次性验收脚本最省路径排序**：channel:'chrome'（18MB，全实测）> --only-shell（98MB）> puppeteer-core（6MB 但手写多）> Python（被 3.9.6 否决）。

---

## §4 实践模式（调研清单 3）：SSE 驱动页面的断言脚本范式

### §4.1 等待原语决策（按优先级）

1. **goto 等待**：`waitUntil: 'domcontentloaded'`（实测 45ms resolve；不要用 networkidle 作主等待——历史 bug 见 §2.1，且对 WS 页面仍会挂）
2. **动态内容主等待**：locator 断言（自动重试到 timeout）：
   ```js
   await expect(page.locator('[data-session-id="env_6"] .status')).toHaveText(/running|idle/, { timeout: 15_000 });
   ```
   （若组件无稳定 testid，先用 `getByText/角色选择器`；建议后续给关键节点补 `data-testid` ——属 env 验收面设计，交 J3 裁决）
3. **REST 引导等待**：`await page.waitForResponse(r => r.url().includes('/api/sessions'))` —— 断言「列表已从后端拉回」
4. **SSE 更新等待**：直接对 SSE 驱动的 DOM 节点断言（toBeVisible/toHaveText）；兜底 `page.waitForFunction(() => document.querySelector(sel)?.textContent === 'expected')`
5. **禁止**：`waitForTimeout` 硬等待；networkidle 作唯一等待

### §4.2 已实测的骨架（本机对 6137 跑通，1.47s）

```js
import { chromium } from 'playwright';
const browser = await chromium.launch({ channel: 'chrome', headless: true });   // 复用系统 Chrome，零下载
const page = await browser.newPage({ viewport: { width: 1440, height: 900 } });
const consoleErrors = [], pageErrors = [];
page.on('console', m => { if (m.type() === 'error') consoleErrors.push(m.text()); });
page.on('pageerror', e => pageErrors.push(String(e)));
await page.goto('http://127.0.0.1:6137/', { waitUntil: 'domcontentloaded', timeout: 20_000 });
await page.waitForFunction(() => (document.querySelector('#root')?.innerHTML || '').length > 50, null, { timeout: 15_000 });
await page.screenshot({ path: 'acceptance/viewport.png' });
await page.screenshot({ path: 'acceptance/full.png', fullPage: true });
// 实测结果：#root 10756 字符、title "rick"、consoleErrors 4×401、截图 58KB/1440×900
```

### §4.3 截图与留档规范（建议，交 J3）

- 每断言组一张 viewport PNG + 收尾一张 fullPage PNG；命名 `NN-场景-时间戳.png`，落 `env_6` 验收资产目录；PNG 实测单张 ~58KB，长期留档无压力
- console 错误与 pageerror 全量落 JSON/MD 报告（本机实测可捕获 401 类资源错误）
- 报告记录：断言清单（通过/失败）、截图索引、console/pageerror 摘要、耗时 —— 满足 KR4「可复验」

### §4.4 token 注入（本简报新增的必要步骤）

当前 6137 实例鉴权开启（§1），验收脚本三选一：

1. **localStorage 预置**（推荐）：goto 前对空白页 `page.addInitScript(t => localStorage.setItem('rick-web-token', t), TOKEN)` → REST 自动带 Bearer，SSE 自动带 `?token=`（前端契约，client.ts:50/sse.ts:381）；token 从服务器实例的 config 读取（`~/.rick/config.json` 的 `web_token`，实测该键存在且非空）——**脚本从环境变量取，不落盘明文**
2. TokenGate 表单路径：`fill` + 保存（整页刷新，多一步等待）
3. 专用验收实例：验收时另起 `rick web --port <P> --token <已知>` —— 干净但多进程管理（交 J1 裁决执行者/端口策略）

---

## §5 轻量替代边界（调研清单 4）：Chrome CLI 实测

官方 CLI 能力面（B 级，[headless-cli 参考](https://developer.chrome.com/docs/automation-and-testing/headless-cli)）：`--dump-dom`（执行 JS 后序列化 DOM）、`--screenshot[=path]`、`--window-size`、`--timeout=<ms>`（到点强制捕获）、`--virtual-time-budget=<ms>`（快进定时器）、`--print-to-pdf`。

**本机实测矩阵（A 级，Chrome 155.0.8059.40 / macOS arm64）**：

| 测试 | 结果 |
|---|---|
| `--dump-dom file:///…/t.html` | ✅ 66 字节正确 DOM（对照组成立） |
| `--dump-dom --timeout=8000 http://127.0.0.1:6137/` | ❌ 40 字节空 `<html><head></head><body></body></html>`，exit=2，stderr "Page load timed out" |
| `--dump-dom https://example.com`（含 `--no-proxy-server` 变体） | ❌ 同样空 DOM / 无 --timeout 时挂死 >120s |
| `--dump-dom http://127.0.0.1:8977/`（本地 python http.server，排除应用因素） | ❌ 空 DOM → **与应用/SSE 无关，是 CLI 管线本身** |
| `--virtual-time-budget=8000`（对 6137） | ❌ 挂死 >90s |
| `--screenshot --window-size=1280,800`（对 6137） | ⚠️ 产出 1280×800 PNG 但仅 **4.7KB**（对比 Playwright 同页 58KB）→ 近乎空白帧 |

文献佐证（C 级，同类报告）：[SO #71418104](https://stackoverflow.com/questions/71418104/chrome-headless-dump-dom-flag-prints-nothing-to-stdout)（dump-dom 无输出）、chromium headless-dev 邮件列表（持续打开的网络请求导致 dump 永不打印）、Selenium headless 空 pageSource 报告。

**能力边界结论**：即便 CLI 在本机可用，它也只有「时间定界的单发捕获」——无元素级等待、无 console 捕获、无交互、断言只能靠 grep dump 输出；而**本机实测它连基本 http(s) dump 都不成立**（file:// 正常 → 网络/导航提交环节的问题）。**判定：CLI 路径在本机不可用，不作为任何一级验收手段**（连冒烟级都不给）。

---

## §6 推荐选型与理由（调研清单 5）

### 推荐：Node Playwright 1.64.1 + `channel:'chrome'`，独立目录安装

**决策链**（对照验收要求）：

1. **断言能力足够**：locator 自动重试断言 + waitForResponse + waitForFunction，覆盖「sessions 列表含某会话 / 状态字段 / SSE 驱动的动态内容」三类断言（§4.1）
2. **外网依赖最低**：仅需 npm registry（~18MB，实测可达且大概率 npm 缓存命中 1s 装完）；零浏览器下载（复用 Chrome 155）；loopback 流量不走系统代理（例外表实测）
3. **一次性脚本成本最低**：单文件 .mjs + node 直跑（node 26 已就绪）；无需 test runner/配置；Python 路径被 3.9.6 硬否决；Puppeteer 相比之下无自动断言且默认静默下载 >500MB
4. **已端到端实测**：launch→goto→渲染→截图→console 捕获全链 1.47s 通过（§4.2）——这是四方案中唯一做过全链实测的
5. **安装卫生**：在 `env_6` 验收资产目录内 `npm i playwright`（独立 package.json），**不污染 web/package.json / web/node_modules**；用完可整目录归档或删除

**风险与回退**：

| 风险 | 概率 | 缓解 |
|---|---|---|
| channel:'chrome' 依赖系统 Chrome（自动升级漂移；官方称 bundled chromium 最佳） | 低 | 回退①：`npx playwright install chromium --only-shell`（98MB）切 hermetic；回退②：全量 chromium |
| Chrome 155 CLI 管线损坏是否暗示底层异常会波及 CDP 驱动 | 已排除 | Playwright 走 CDP 全链实测通过（§4.2）；CLI 损坏仅限其 dump/screenshot 管线 |
| SSE 页面 networkidle 陷阱 | 已消解 | 1.64.1 源码级修复 + 实测；主等待本就不用 networkidle |
| 系统代理环境变化 | 低 | loopback 豁免；必要时 launch 加 `--no-proxy-server` 或 proxy bypass |
| 鉴权 token 获取 | 需裁决 | §4.4 三方案，倾向 addInitScript + 环境变量注入（token 不落盘） |

**交给 J1/J3 的输入**（本调研产生的决策项）：
- J1：验收执行者用哪种 token 通道（§4.4）；是否给关键 DOM 节点补 `data-testid`（提升断言稳健性，属 UI 侧小改）
- J3：截图/报告留档形态采纳 §4.3 规范与否；验收脚本资产目录位置（建议 `env_6` 下独立目录）

---

## §7 信源清单

| # | 信源 | 等级 | 要点 |
|---|---|---|---|
| 1 | 本机实测命令（附录） | A | 服务/鉴权/安装尺寸/Playwright 全链/CLI 失败矩阵 |
| 2 | playwright-core 1.64.1 发布 tarball 源码（coreBundle.js `_isExcludedFromNetworkIdle`） | A | EventSource 豁免已随 stable 发布 |
| 3 | [playwright#41513](https://github.com/microsoft/playwright/issues/41513) + [fix commit c370acf](https://github.com/microsoft/playwright/commit/c370acfe5ab176187182eeeed526adb54c221d30) | B | networkidle×SSE bug 与修复 |
| 4 | [playwright#37226](https://github.com/microsoft/playwright/issues/37226) | B | WebSocket 仍不豁免 |
| 5 | [playwright.dev/docs/browsers](https://playwright.dev/docs/browsers) + [class-browsertype](https://playwright.dev/docs/api/class-browsertype) | B | channel:'chrome'、--only-shell 语义 |
| 6 | [pptr.dev/guides/installation](https://pptr.dev/guides/installation) | B | npm i 自动下载 CfT ~170MB(macOS)+headless-shell；~/.cache/puppeteer |
| 7 | PyPI `playwright` 1.64.0 JSON 元数据（本机 curl） | A/B | requires_python >=3.10；wheel 41MB |
| 8 | npm 元数据（本机 npm view/pack 实测） | A | playwright 4.8M/core 13M/puppeteer-core 5.8M unpacked |
| 9 | [developer.chrome.com headless-cli 参考](https://developer.chrome.com/docs/automation-and-testing/headless-cli) | B | --dump-dom/--screenshot/--timeout/--virtual-time-budget 官方语义 |
| 10 | [Selenium#5985](https://github.com/SeleniumHQ/selenium/issues/5985) + [selenium.dev Safari 文档](https://www.selenium.dev/documentation/webdriver/browsers/safari/) | B | Safari 无 headless；需 safaridriver --enable |
| 11 | cdn.playwright.dev HEAD 实测（本机 curl） | A | chrome zip 185MB / headless-shell 98MB / ffmpeg 1MB |
| 12 | SO #71418104、chromium headless-dev 邮件列表、Selenium 空 pageSource 报告 | C | CLI dump 空输出的同类社区佐证 |

---

## 附录：关键复现命令（全部在本机执行过）

```bash
# 基线
curl -s -o /dev/null -w "%{http_code}" http://127.0.0.1:6137/            # → 200
curl -s "http://127.0.0.1:6137/api/sessions?workspace=super" | head -c 120  # → 401 unauthorized
ps aux | grep "rick web"                                                  # → --listen 0.0.0.0 --port 6137（无 --token）
"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome" --version  # → 155.0.8059.40

# Playwright 选型验证（独立目录，不碰仓库）
mkdir -p /tmp/pwtest && cd /tmp/pwtest && npm init -y && npm i playwright   # 1s，18MB，无浏览器下载
#   verify.mjs 见 §4.2；node verify.mjs → 全链 1.47s 通过（脚本留存 /tmp/pwtest/verify.mjs）

# 下载量实测
npx playwright install --dry-run chromium                                # CfT 156.0.8078.4 + 三包清单
curl -sIL <zip-url> | grep -i content-length                             # chrome 185MB / shell 98MB / ffmpeg 1MB
npm view playwright dist.unpackedSize; npm view puppeteer version        # 4.8MB / 25.14.0

# Python 路径否决依据
curl -s https://pypi.org/pypi/playwright/json | python3 -c "import json,sys; d=json.load(sys.stdin); print(d['info']['requires_python'])"  # >=3.10（本机 3.9.6）

# CLI 失败复现（对照 file:// 成功）
"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome" --headless --no-first-run \
  --user-data-dir=$(mktemp -d) --dump-dom --timeout=8000 http://127.0.0.1:6137/   # → 40 字节空 DOM, exit=2
```

*简报完。调研产物：本文件 + /tmp/pwtest/{verify.mjs,results.json,shot-viewport.png,shot-full.png}（临时区，可随时清理/归档）。*
