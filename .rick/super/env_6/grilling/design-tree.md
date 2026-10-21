# env_6 设计树（grilling 活文档）

> 使命锚点：`requirement.md` = 「优化 512×512 矩阵乘性能，从纯 Python 三重循环起点进化（无头浏览器全链验收）」
> 协议：逐层追加；每层 = 模块 + pipeline + 判断节点；L1 调研记录随层落盘。

---

## 第 0 层（根）：OKR

**O（Objective）**：交付 env_6 super 进化使命的**端到端全链**：matmul 512×512 进化从纯 Python 三重循环起点真实跑通（grilling → gen_env → 门先证红 → 后台进化入格 → 干预 → 干净停止），并以**无头浏览器自动化验收**全链产出，验收资产落盘可复验。

**KR 集（演绎：KR1 ∧ KR2 ∧ KR3 ∧ KR4 ⟹ O）**：
- **KR1 需求理清**：grilling 设计树落盘并过 grilling_gate——「无头浏览器全链验收」语义、进化预算、验收断言面、门禁定制全部裁决。
- **KR2 env 合格**：gen_env 产出 env（任务定义/门禁/exploit 注册表/特征空间/.venv）过验收自检四项（参考过正确性门 / bench 出分 / exploit 全拒识 / naive 过门）。
- **KR3 进化跑通**：`rick super start`（含门先证红 100% 拒识 + 金样例零误杀）→ 后台进化达标（state=running、心跳新鲜、iteration_cursor 递增、精英入格、北极星可读）→ 干预经 `rick super event` 投递且事件流可见 → `rick super stop` 干净收尾（stopped + session_ended + 无孤儿进程）。
- **KR4 无头浏览器全链验收**：无头浏览器自动化验收覆盖全链可观测面（web UI 面 + 后端真相源面），验收脚本与报告落盘 env_6 工作区，可复验。

**OKR 充分性自检**：需求理清（KR1）是其余三步的前提；env 合格（KR2）是进化启动的硬前置（无合格 env 不启动）；进化跑通（KR3）是使命主体；无头浏览器验收（KR4）是本次使命区别于 env_2~5 的增量要求，也是「交付完成」的客观判据。四者联合 ⟹ 端到端全链交付。✅

---

## 第 1 层：使命 pipeline（四大模块）

```
[M1 grilling 需求追问] → [M2 gen_env 环境建设] → [M3 进化运行监督] → [M4 无头浏览器全链验收]
     （现在）              （KR2）                （KR3）              （KR4，收尾）
```

| 模块 | 职责 | 输入 | 输出 | 边界 |
|---|---|---|---|---|
| M1 grilling | 设计树追问循环，消解判断节点 | requirement + 仓库事实 | design-tree.md（过 grilling_gate） | 不建设 env、不启动进化 |
| M2 gen_env | 访谈式 env 建设（任务定义/门禁/exploit/特征空间/.venv） | KR1 裁决 + matmul 模板 | `env_6/env/` + `env_6/gates/`，验收自检四项全过 | 门禁阈值属 human 决策，skill 只引导 |
| M3 进化运行监督 | start（门先证红）→ status 监控 → event 干预 → stop | KR2 的 env | events.jsonl（唯一真相源，只读）+ archive 精英 + 干预事件 + 干净停止 | 中控不执行进化逻辑；不直写 events/archive |
| M4 无头浏览器验收 | 自动化断言全链可观测面，报告落盘 | M3 的运行痕迹 + web UI | 验收脚本 + 验收报告（env_6 下） | 范围待 KR1-Q1 裁决 |

**层内 pipeline 依赖**：M1 → M2 → M3 → M4 严格串行（后者的输入是前者的产出）；M4 的断言面设计在 M1 就要裁决（影响 M3 观察哪些痕迹留档）。

### L1 调研记录（轻量自查 + research 派发）

**已自查消解的事实**（信源：仓库文件直读，等级 A）：
1. env_1~5 全部空跑（state=idle、events.jsonl 0 行、archive 空）——「验收复验」×2 只验了 web 入口链路（job_48 的 super 会话 v2），**从未有一次跑完进化全链**。
2. web super 会话链路已建成：`rick web` 单实例服务 http://127.0.0.1:6137（loopback 默认免 token）→ NewSessionModal super 表单（requirement 必填）→ `scanNextEnvID` 自动编号 env_N → `BuildSuperPrompt`（严格加载 super-loop + gen_env skill）→ spawn。
3. 仓库**无任何 Playwright/Puppeteer/chromedp 基建**（web devDeps 无 e2e 框架；PortalArt.tsx:64 注释提到过一次人工「无头实测」canvas 性能，无脚本留存）。
4. matmul 模板完整可用：env.json（512 float32 / 容差 1e-3 / K=3 中位数 / score=ref/cand 加速比）、feature_space 2 轴（impl_strategy × memory_layout）、test_correctness.py（64×64 固定 seed=42）、bench.py（预热 1 + K=3）、domain_knowledge.md（分析 SOP + 调优 SOP + 作弊模式清单）。
5. 本机 Apple M5 Max（arm64）；系统 Python 3.9.6 无 numpy → env 需建 `.venv`；pi 0.84.2 可用（`rick-dev-home/.rick/pi/agent/runtime/node_modules/.bin/pi`）。
6. rick 二进制在 `/Users/logic/workdir/rick-dev/bin/rick`（不在 PATH）。
7. super.json 默认：max_wall_time 24h、adjust_interval 5、max_iterations 0（不限）、preverify_red=true。

**research 派发**（重量级，联网选型对比）：research-L1 **已完成**（简报 22.4KB，7 章 + 复现命令，信源分级 A/B/C，主路径本机端到端实测）。核心结论：

1. **选型落定**：Node Playwright 1.64.1 + `channel:'chrome'`——复用本机 Chrome 155.0.8059.40，npm 仅 18MB 零浏览器下载；已对 127.0.0.1:6137 全链实测通过（launch→goto→React 渲染 #root 10756 字符→截图 58KB→console 捕获 4×401，总计 1.47s）。安装卫生：env_6 验收资产目录内独立 package.json，不污染 web/。
2. **⚠️ 校准（推翻自查第 2 条的一半）**：loopback **不免 token**——运行中的 6137 实例鉴权开启（`/api/sessions`/`/api/events` 均 401；进程 `rick web --listen 0.0.0.0 --port 6137`，无 `--token` → 用 config `web_token`，实测非空）。验收脚本必须 token 注入：推荐 `page.addInitScript` 预置 `localStorage['rick-web-token']`（token 从环境变量取，不落盘明文）。
3. **否决路径**（全部实测依据）：Python playwright（requires ≥3.10 vs 本机 3.9.6）；safaridriver（Safari 无 headless）；Chrome CLI `--dump-dom/--screenshot`（本机对 http 目标一律 40 字节空 DOM / 4.7KB 空白帧，file:// 对照正常——CLI 无头管线在本机损坏）。
4. **SSE 等待策略**：Playwright 1.64.1 已修复 EventSource 对 networkidle 的豁免（源码级验证 + 实测 586ms）；主等待原语仍用 `domcontentloaded` + 元素级断言（locator 自动重试），禁 waitForTimeout 硬等待。
5. **回退链**：channel:'chrome'（18MB）→ `install chromium --only-shell`（98MB）→ 全量 chromium（284MB）。

简报全文：`research-L1.md`。

### L1 判断节点（待 human 裁决 → 见批量追问）

- **J1**：「无头浏览器全链验收」的**链路范围与执行者**——谁跑无头浏览器、断言哪些面（web UI 面 / 后端真相源面 / 两者）。
- **J2**：进化运行**预算与达标判据**——跑多久 / 多少迭代 / 入格目标，什么算「跑通」。
- **J3**：**验收通过标准**——断言面清单是否认可、验收资产落盘形态。
- **J4**：**门禁与 exploit 定制程度**——照模板 or 收紧；exploit 样本构造方向。
- J5（附带）：grilling 交互模式——本会话正常交互追问（human 在场），验收脚本只断言产物存在性，不重放问答。（此项无明显分歧，默认此方案）

**research-L1 回流新增的子决策**（并入 Q1/Q3 裁决，不单独开问）：
- J1a：token 通道——推荐 `addInitScript` 预置 localStorage + 环境变量取值（不落盘明文）；替代方案是验收时另起 `rick web --token <已知>` 专用实例（多进程管理成本）。
- J3a：验收资产目录——推荐 `env_6/acceptance/`（脚本 + 截图 + 报告整目录，npm 依赖装在此不污染仓库 web/）。
- J3b：是否给关键 DOM 节点补 `data-testid`——**推荐不补**（避免为验收改产品代码；用 getByText/角色选择器即可，research §4.1 已给出无 testid 的断言范式）。

---
