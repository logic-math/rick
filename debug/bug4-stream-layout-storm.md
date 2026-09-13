---
summary: "web 端流式比 CLI 慢——根因不是 pi/服务端，而是浏览器主线程被「每条历史 assistant 消息一个无限 SVG 旋转动画（Portal 头像）」压垮：长会话（240 条消息 ≈ 2.5 万 DOM 节点）下每帧全文档重排（dirtyObjects=7224/26395、partialLayout=false）+ ~65 次/秒样式重算。修复：Portal 旋转只保留给正在流式的消息 + 星空 canvas 与贴底滚动不再逐帧读布局（clientWidth/scrollHeight）；帧间隔 38→17ms（60fps）、DOM 延迟 p50 52→21ms、长任务 70→0、同量流式 drain 87→15s。附带修掉星空 canvas 逐帧强制重排与 MessageList 贴底 scrollHeight 读取。"
status: "✅ 已解决"
---

# bug4：web 端流式推进慢（长会话掉帧 / 事件积压）

## Phase 1: 构建反馈回路

**用户报告**：「底层 pi 返回消息的速度那么慢？使用 CLI 的时候为什么没有？」
——web 端流式文字推进明显慢/卡，CLI 无此现象。

**既有证据（父会话已采集，作为起点）**：真实会话 74s 窗口内 SSE 送达 1969 条
`message_update`，相邻 delta 间隔中位 **8ms**（p90 75ms）——**服务端/pi 不慢**。

**反馈回路（新建，可重复运行）**：
- 工具：`/tmp/ui-verify/measure_perf.mjs`（Playwright + CDP Performance 指标）
- 场景：mock 历史（N 条 assistant 消息，每条 6 段文本 + 1 段 thinking）
  + mock SSE 以 8ms/帧 注入 `text_delta`（标记 `D{i}|`，逐帧写入 DOM）
- 指标：DOM 延迟（事件发出 → 标记文本出现在 DOM，p50/p95）、帧间隔、
  长任务、drain（最后一帧被渲染的总耗时）、CDP 窗口内
  Task/Script/Layout/RecalcStyle 时长与次数
- 运行：`RICK_TOKEN=$(cat /tmp/lat/token.txt) HIST=240 FRAMES=1200 INTERVAL=8 KIND=text \
  LABEL=x node /tmp/ui-verify/measure_perf.mjs`（6173 隔离实例 + 生产构建）

**基线（HIST=240 ≈ 25214 DOM 节点，1200 帧）**：

| 指标 | hist0（无历史） | hist240（长历史） |
| --- | --- | --- |
| DOM 延迟 p50 / p95 | 21 / 24 ms | **52 / 63 ms** |
| 帧间隔 p50 / p95 | 17 / 17 ms | **38 / 48 ms**（≈26fps） |
| 长任务数（max） | 0 | **70（max 1489ms）** |
| drain（1200 帧×8ms） | 10.3s | **87.3s（8.4×）** |
| CDP Task / Layout / StyleRecalc（600 帧窗口） | — | 41471 / 11313 / 8836 ms |

## Phase 2: 复现最小化

- 变量隔离：只改历史消息条数（0 / 30 / 240），其它不变 → 延迟与帧间隔随条数单调恶化
  （hist30 已出现 1 次 251ms 长任务；hist240 帧间隔 38ms）。**成本随消息数线性增长**。
- 最小复现单元：单个 mock 会话 + 240 条历史 + 8ms/帧 delta 注入（6 秒内可重复）。
- 基线对照：`hist0`（无历史）= 21ms / 17ms / 0 长任务 —— 证明应用本身在短会话下是健康的。

## Phase 3: 可证伪假设

### 假设 1（高）：每帧强制同步重排（forced synchronous layout）
- **观察证据**：`StarfieldBackground` 的每个绘制函数逐帧读 `canvas.clientWidth/clientHeight`
  （14 处）；`MessageList.jumpToBottom` 与 `onScroll` 每次读 `scrollHeight`。
- **推断**：DOM 变更后读布局属性 → 浏览器同步重排全文档（2.5 万节点 ≈ 1.3ms/次，每帧 ~10 次）。
- **验证方法**：instrument `Element.prototype` 的 scrollHeight/clientWidth getter
  （计数 + 耗时 + 慢读取调用栈）。

### 假设 2（高）：历史消息中的无限 CSS 动画数量随消息数增长
- **观察证据**：`Portal`（每条 assistant 消息的头像）内联 `rm-portal-spin ... infinite`
  作用在 SVG `<g>` 上（不可合成）。
- **推断**：N 条消息 = N 个无限动画 → 主线程每帧为每个动画元素重算样式/布局。
- **验证方法**：CDP trace 统计 `animationiteration` 事件数；注入 CSS
  `[role="log"] svg g{animation:none}` 做消融对比。

### 假设 3（低）：`buildChatViewModel` 每帧全量重放是瓶颈
- **验证方法**：esbuild 单独打包 viewModel → Node 微基准（不同 envelope 数）。

### 假设 4（低）：服务端转发/SSE 链路慢
- **验证方法**：既有实测（delta 间隔中位 8ms）已排除；CDP Script 时长占比亦可交叉验证。

## Phase 4: 插桩观察

**H3 证伪**（微基准，`/tmp/lat/bench_vm.mjs`）：全量重放耗时
`0.116ms p50 @2020 envelopes`、`0.269ms @4020` —— 可忽略；CDP 窗口 Script 合计 1257ms/41.5s（3%）。

**H1 部分成立（但对帧时间不足）**：
- 布局读取插桩：**6002 次读取累计 8371ms**；慢读取调用栈聚合指向
  **星空 canvas 逐帧 `clientWidth/clientHeight`（8042ms）** 与
  **MessageList 贴底 `scrollHeight`（905ms）**。
- 修复读取后（读取→0，rAF 回调累计 8440ms→242ms）：帧间隔仍 **38-40ms** ——
  说明 H1 是共犯，不是主因。

**H2 决定性证据**：
1. CDP trace：`EventDispatch` 中 **`animationiteration` 2926 次 / ~1.5s**（≈1000+ 个无限动画元素）。
2. `Layout` 事件参数：**`dirtyObjects=7224 / totalObjects=26395`、`partialLayout=false`、
   layoutRoot=`#document`** → 每帧全文档布局（~2 次/帧）。
3. 消融：注入 `[role="log"] svg g{animation:none!important}`（同场景 400 帧）：

| 指标 | 基线 | 停掉消息区 SVG 动画 | 倍数 |
| --- | --- | --- | --- |
| Task | 28276 ms | **2824 ms** | 10.0× |
| Layout | 7558 ms | **305 ms** | 24.8× |
| StyleRecalc | 6198 ms | **55 ms** | 112× |
| 帧间隔 p50 | 39 ms | **17 ms** | 2.3× |
| DOM 延迟 p50 | 51 ms | **19 ms** | 2.7× |
| drain | 28.2 s | **4.2 s** | 6.7× |

**根因**：`AssistantBubble` 为**每条**（含全部历史）消息渲染 `<Portal>`，其 SVG `<g>`
带 `rm-portal-spin … infinite` 内联动画。SVG 内部 transform 动画不可合成，必须由主线程
逐帧更新；数百~上千个并发无限动画使主线程饱和 → 每帧全文档样式重算 + 重排 →
帧间隔 38-40ms、流式事件积压 → 用户感知「web 比 CLI 慢」（CLI 无此渲染管线）。
（H4 早前已被 SSE 实测排除。）

## Phase 5: 修复回归

三处修复（均在前端，`web/src/**`）：

1. **`components/starfield/Portal.tsx`**：新增 `spin?: boolean`（默认 **false**，静态）；
   `loading || spin` 才注入 `rm-portal-spin`。`components/chat/MessageBubble.tsx` 的
   `AssistantBubble` 传 `spin={item.streaming}` —— **只有正在流式的那条消息旋转头像**。
   装饰性固定位置的 Portal（`App.tsx` 侧栏 logo、`ChatView.tsx` 头部、`Settings.tsx` 页脚，
   共 3 个）显式 `spin` 保留设计动效。
2. **`components/starfield/StarfieldBackground.tsx`**：尺寸只在 `resize()`/`ResizeObserver`
   读取并缓存 `cssW/cssH`，所有绘制函数改用缓存值 → 消除每帧 5 次强制全文档重排。
3. **`components/chat/MessageList.tsx`**：贴底/滚动判定不再读 `scrollHeight`——
   用 `ResizeObserver` 维护内容/视口高度缓存（回调时布局已干净），贴底改为写哨兵值
   `BOTTOM_SENTINEL` 由浏览器 clamp 到底。修正 effect 依赖（列表首现后再挂观察器，
   避免首帧 refs 为 null 导致高度缓存恒 0 → 贴底失效——回归验证时发现并修掉）。

**修复后（同场景，生产构建）**：

| 指标 | 修复前 | 修复后 | 对照 hist0 |
| --- | --- | --- | --- |
| DOM 延迟 p50 / p95 / max | 52 / 63 / 126 ms | **21 / 26 / 41 ms** | 21 / 24 / 25 ms |
| 帧间隔 p50 / p95 | 38 / 48 ms | **17 / 21 ms（60fps）** | 17 / 17 ms |
| 长任务数（max） | 70（1489ms） | **0** | 0 |
| drain（1200 帧） | 87.3 s | **15.3 s** | 10.3 s |
| CDP Task/帧 | 69 ms | **9 ms**（7.7×） | — |
| CDP Layout/帧 | 18.9 ms | **1.5 ms**（12.6×） | — |
| CDP StyleRecalc/帧 | 14.7 ms | **0.17 ms**（87×） | — |

其它场景（修复后）：thinking 展开（hist240，800 帧）21/26ms、17ms、0 长任务；
iPhone 模拟 19/25ms、17ms、0 长任务；真实长会话（8412，`aed8d899`，673 entries，
真实历史 + 注入流式 800 帧）20/26ms、17ms、2 次长任务（≤52ms）。

**回归验证**：
- `repro12_think_regression.mjs`（bug2/3 不变量：落定文本抗裁剪 / live key 就地更新 /
  settle 无重挂 / 展开 override 保持）桌面 + iPhone → **全部通过**（替换 0 / 消失 0 /
  override 保持 / 无 JS 错误）
- `repro13_multicycle.mjs`（3 段 × 240 帧，跨裁剪多轮）→ **重挂 0、文本头完整、override 保持**
- 滚动功能核对（新增 `verify_scroll.mjs`）：初始贴底 ✓ / 流式持续跟随 ✓ /
  上滚出现「回到底部」且不再强制拉底 ✓ / 点击回到底部 ✓
- 头像动画核对（`verify_spin.mjs`）：历史 120 条 → 动画元素 0；流式中 → **1**（仅 live）；落定后 → 0
- 星空渲染 sanity：canvas 有绘制、resize 后跟随缩放 ✓
- `npx tsc --noEmit` / `npm run build` / `go test ./internal/web/ ./internal/runtime/` 全绿

## Phase 6: 清理事后分析

- **为什么之前没发现**：bug1/2/3 的复现脚本都用**短历史**（0-2 条消息），历史消息量
  这一维度未被覆盖 → 「每消息一个动画」的线性成本被隐藏。教训：流式性能回归脚本必须
  带**长历史**变量（本次已固化 `measure_perf.mjs` + `HIST` 参数）。
- **测量方法学教训**：
  ① 我的测量轮询器自身曾把 `textContent` 扫描误算成应用开销——所有插桩指标先做
  空跑基线（`hist0`）对照；
  ② trace 聚合必须按**主渲染线程**（pid/tid）过滤，跨线程求和会把耗时放大数倍；
  ③ 「慢读取调用栈聚合」比逐项猜测高效得多（一次就把 8042ms 定位到星空 canvas）。
- **反直觉点**：`content-visibility: auto` 的消融（5.4×）其实指向同一根因（屏幕外内容
  跳过渲染 → 动画/布局成本消失），若只按它去改会误入「虚拟化」方向；最终定位到
  「无限动画数量」才是最小且语义正确的修复面。
- **保留观察项**：ThinkingBlock 展开时块内贴底仍读 `scrollHeight`（约 0.4ms/次，
  仅用户主动展开时发生；实测帧间隔仍 17ms、无长任务），未改动以控制回归面。

## 结论

根因：**每条历史 assistant 消息的 Portal 头像带不可合成的无限 SVG 旋转动画**，
长会话下数百个并发无限动画使主线程每帧做全文档样式重算与重排；
次因是星空 canvas 与贴底滚动的逐帧强制布局读取。
修复：头像旋转只保留给正在流式的消息（+ 两处布局读取消除）。
修复后流式 DOM 延迟 p50 = 21ms（≈1 帧）、帧间隔锁定 60fps、0 长任务，
长会话与短会话表现一致，且 bug1/2/3 不变量与滚动/折叠/头像动效均无回归。
