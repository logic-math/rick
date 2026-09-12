---
summary: "思考过程（thinking）展开/流式显示时「不断闪烁刷新」——根因：会话事件环形缓冲（MAX_PER_SESSION=500）的 trimBuffer 把「正在流式消息（打开块）」的 message_update 增量帧也当可裁项丢掉 → ① live 块 id 取自「最旧存活 delta 的 envelope.seq」→ 每次裁剪 id 平移 → React key 变 → thinking 块重挂 + chat-item-enter 进场动画重放（实测单次 900 帧流触发 56 次 rm-chat-in 动画）；② 累积文本头被吃掉（T0, → T401,）→ 内容每帧跳变。已修 trimBuffer：打开块（最近 message_end 之后的帧）的增量帧永不裁。"
status: "✅ 已解决"
---

# bug3-thinking-stream-flash.md

## Phase 1: 构建反馈回路

**用户报告**（第三次；前两次修复见 bug1/bug2）：思考过程（thinking）在页面显示时**仍不断闪烁刷新**；用户明确要求「要能看思考过程且不闪」，不接受再靠「默认折叠」绕过。

**复现环境**：
- 6173 verify 实例（`/tmp/start-verify.sh`，HOME=/tmp/rick-e2e-home），使用其 registry 中已存在的交互型会话 `bea28aef-7bb3-4187-80df-a19727d39dff`（type=plan，status=error；ChatView 正常渲染，entries 8 条真实历史）。
- Playwright headless + MockEventSource（修正版：`ev.data` 必须是 JSON 字符串）注入 thinking 流。
- 脚本：`/tmp/ui-verify/repro10_thinking_flash.mjs`。

**DOM 检测手段**（沿用 bug2 结论：禁用像素差分）：
- `[data-think-block]` 标记 thinking 容器（标题含「思考过程」）；探针只认「含 live 注入标记 `T<数字>,`」的块（避开历史 thinking 块）。
- 40ms 采样：块 DOM 节点 identity（出现/替换/消失）、`<p>` 文本长度/头部 14 字符、块内 `div.overflow-y-auto` 的 scrollTop/scrollHeight/clientHeight。
- `animationstart` 捕获（捕获阶段）：动画名 + 是否在 chat-item-enter 包裹层内 + 包裹层文本前缀。

**对照实验**（同脚本，仅注入帧数不同）：

| 组 | 注入 | thinking 块 出现/替换/消失 | chat-item-enter 动画 | 文本头 distinct | 末帧头部 |
|---|---|---|---|---|---|
| A（<500 帧，不触发裁剪）| 300 帧 thinking_delta | 1 / **0** / 1 | 5（均首屏挂载）| **1** | `T0,T1,T2,...` ✓ 完整 |
| B（>500 帧，触发裁剪）| 900 帧 thinking_delta | 6 / **1** / 6 | **56** | **7** | `T401,T402,...` ✗ 头部被吃 |

B 组动画目标串（去重）：
```
▾思考过程24BT0,T1,T2,T3,T4,T5,
▾思考过程3KT5,T6,T7,T8,T9,T10,
▾思考过程3KT13,T14,T15,T16,T17
▾思考过程3KT21,T22,T23,T24,T25
▾思考过程3KT29,T30,T31,T32,T33
```
→ **同一个 thinking 块被反复重挂**（每次重挂文本头都往后跳一段），每次重挂重放 `rm-chat-in` 进场动画 = 用户所见「不断闪烁刷新」。

## Phase 2: 复现最小化

**纯函数级（秒级、决定性）**：`esbuild src/stores/events.ts src/components/chat/viewModel.ts → node`（脚本 `/tmp/think_repro1.mjs`）——模拟长会话缓冲（400 条结构/旧增量帧，trimBuffer 后）→ 逐条 append thinking_delta（每帧 trimBuffer + buildChatViewModel）：

```
frame=  0 buf=402 id=live-402 len=    3 head="T0,"
frame=100 buf=500 id=live-402 len=  395 head="T0,..."
frame=250 buf=500 id=live-402 len= 1145 head="T0,..."
frame=300 buf=500 id=live-404 len= 1389 head="T2,T3,..."      ← 首次裁剪起点平移
frame=350 buf=500 id=live-454 len= 1447 head="T52,T53,..."
frame=400 buf=500 id=live-504 len= 1495 head="T102,..."
frame=499 buf=500 id=live-603 len= 1495 head="T201,..."       ← 头部持续被吃
```
- live 块 id `live-402` → `live-603`（**每次裁剪平移 = React key 变 = 重挂**）
- 文本长度在 1495 处**不再增长**（头部增量被丢），头部从 `T0,` 退到 `T201,`

## Phase 3: 可证伪假设

### 假设 H6（✅ 确认，主因）
- **观察**：`trimBuffer` 只判断「是否 delta 帧」，把**正在流式的消息（打开块）**的 `message_update` 帧也当可裁项；live 块 id = `live-${首个存活 delta 的 envelope.seq}`。
- **推断**：buffers 超过 500 后，每来一帧就丢一帧**最旧的 delta**——而最旧 delta 往往正属于当前打开块 → live 块 id 每次平移 + 累积文本头被丢 → 重挂（动画重放）+ 内容跳变 = 用户所见「不断闪烁刷新」。
- **验证**：纯函数（id 平移 + 头部被吃，Phase 2）+ DOM（56 次 rm-chat-in、块节点反复替换、头部 7 次变化，Phase 1）。
- **结果**：✅ 确认。**这是 bug2 修复（裁剪优先丢增量帧）引入/暴露的缺陷**——bug2 只让「落定条目」id 稳定（改用 envSeq），但 live 块 id 仍依赖「首个存活 delta」，且 live 文本完整性依赖打开块的 delta 全在。

### 假设 H1（部分成立，非主因）
- **推断**：`ThinkingBlock` 展开时每次 text 变都 `scrollTop = scrollHeight` 贴底 → 块内文字持续向上滚动 = 「刷新感」。
- **验证**：对照组 A（不裁剪）scrollTop distinct=1（内容未溢出 max-h-72，无滚动）；问题组 B scrollTop 0→22（溢出后才跟随一次）。
- **结果**：⚠️ 非本次主因（对照 A 无裁剪时不滚也不闪）；保留现有「接近底部才跟随」策略。

### 假设 H2（❌ 被 H6 解释；settle 期一次性重挂另计）
- **推断**：`chat-item-enter` 进场动画在节点重挂时重放 = 闪。
- **验证**：DOM `animationstart` 抓取——闪的**载体**确实是 `chat-item-enter`（rm-chat-in），但**重挂的驱动因**是 H6 的 key 平移（对照 A 无裁剪 → 0 次重挂）。
- **结果**：❌ 作为独立主因不成立；作为「可见闪」的表现层成立——因此修复还需让 live→落定切换不重放动画（见 Phase 5 F2）。

### 假设 H3/H4/H5（❌ 排除）
- H3（标题行字节计数每帧变）/H5（块高度增长推动外层贴底）：数值变化存在但都是必要渲染，非闪源（对照 A 同样存在却零重挂/零动画重放）。
- H4（StreamText 光标 `rm-pulse` 0.9s 呼吸）：thinking 块光标为静态方块（无动画）；`rm-pulse` 未出现在 thinking 块内。

## Phase 4: 插桩观察

- `events.ts: trimBuffer`——`if (deltas.length >= overflow) { const keep = new Set(deltas.slice(overflow)); return buf.filter(env => !isDeltaFrame(env) || keep.has(env)); }`：**无「打开块」概念**，纯按「delta 优先裁」；打开块的 delta 与已落定块的 delta 同等对待。
- `viewModel.ts: applyMessageUpdate`——`liveText/liveThinking` 由 **首个** delta 创建，id = `liveId(st.idSeq)` = `live-${首个存活 delta 的 seq}`；文本 = 缓冲内所有存活 delta 的顺序拼接 → **id 与文本都依赖「打开块 delta 全在」**。
- `viewModel.ts: applyMessageEnd`——`liveText=null; liveThinking=null`，落定条目 id = `think-{message_end 的 envSeq}`（结构帧，永不裁）→ **message_end 是「打开块」的天然边界**。
- `events.ts: flush`——每次 rAF flush 对缓冲追加并 trimBuffer；流式期间 flush ≈ 每帧 → 每次 flush 都可能平移 live 块 id。
- 结构性帧（message_end/tool_execution_*/agent_*）**从不被裁**（bug2 已保证）→ 可作为稳定锚点。

## Phase 5: 修复回归

**修复（3 处，最小改动）**：

### F1 `web/src/stores/events.ts` — 环形裁剪保护「打开块」（根因修复）
- 新增 `openBlockStart(buf)`：返回最近一次 `message_end` 之后的索引（无 message_end → 0）。依据：`viewModel.applyMessageEnd` 在 message_end 清空 liveText/liveThinking → 打开块的完整证据只在该边界之后。
- `trimBuffer` 重写：
  - **打开块的增量帧永不裁**（live 块文本与 key 的唯一来源）；
  - 只裁「已落定（打开块之前）」的最旧增量帧——完整事实由 message_end / tool_execution_end 承载；
  - 结构事件照旧永不裁；
  - 安全阀 `OPEN_BLOCK_MAX = 20000`：失控单条消息（>2 万帧）才退化丢打开块最旧增量（极端情况文本头可能缺失；正常消息远低于此）。
- 代价（已知取舍）：打开块 + 结构帧可能使缓冲暂时超过 `MAX_PER_SESSION=500`（打开块完整性优先）。实测：900 帧流期间缓冲约 1301（结构 400 + 打开块 900）。
- **对照（纯函数，同一 900 帧流每帧 trim）**：
  | | live id 种类 | 文本头种类 | 末帧长度 | 缓冲 |
  |---|---|---|---|---|
  | 旧 trimBuffer | 1 | **9（头被吃）** | 1495（截断） | 500 |
  | 新 trimBuffer | 1 | **1（完整 `T0,`）** | **4390（完整）** | 1301 |
- 安全阀回归：25000 帧单条消息 → 保留 20000（有界，4ms）；5000 帧常规消息 → 全保留且文本完整（未退化）。

### F2 `web/src/components/chat/viewModel.ts` — 消息块身份锚定（消除 live→落定重挂）
- BuildState 增 `boundarySeq`（**前置** message_end 的 envelope.seq；0=会话首条消息）与 `liveOrder`（live 块创建序）。
- 新增 `msgBlockId(anchor, kind, idx)` → `msg-{anchor}-{kind}[-{idx}]`：
  - **live 块与落定块共用同一 id**（锚点为结构帧 seq，跨裁剪/重放稳定）→ settle 是**同一 React key 的就地更新**：不重挂、不重放 `rm-chat-in`、`useExpandState` 的展开 override 不丢（此前 settle 会把用户展开的思考块折叠 + 闪一下）。
  - 同类块用**同类序号**（textIdx/thinkIdx）而非共享 blockIdx —— 否则 `[thinking, text]` 消息里 text 落定 id 会带 `-1` 后缀、与 live id 不匹配。
- live 收尾按 `liveOrder` 输出（此前固定「先 text 后 thinking」，与落定 block 序不一致 → settle 位置跳变）。
- 纯函数回归：live/settled id 完全一致且唯一；裁剪后 settled id 零漂移；跨消息锚点隔离；多同类块无碰撞。

### F3 `web/src/components/chat/ThinkingBlock.tsx` — 块内贴底跟随改用 `useLayoutEffect`
- `useEffect` 在**绘制后**才设 scrollTop → 每帧先按旧滚动位绘制（内容已增长，文字先下移一行）再被纠正 = 逐帧抖动。`useLayoutEffect` 在绘制前同步 → 消除该抖动（effect 内只读布局属性，无副作用）。

**DOM 回归（桌面 + iPhone 13 模拟）**：
| 场景 | 修复前 | 修复后 |
|---|---|---|
| 900 帧 thinking（触发裁剪）| 块重挂 1 次、**56 次 rm-chat-in**、文本头 7 种（`T0,`→`T401,`）| 块重挂 **0**、**5 次 rm-chat-in（全部首屏挂载）**、文本头 **1 种（`T0,`）**、长度 4390 完整 |
| 流式中手动展开 → 落定（repro11）| thinking 动画 2 次（live 挂载 + **settle 重挂**），展开态跨 settle 有丢失风险 | thinking 动画 **1 次（仅挂载）**，`aria-expanded` 全程 true |
| 3 段 thinking 轮次（repro13）| — | 3 段全部 key 稳定（重挂 0）、头完整、无消失、9 次动画均为挂载 |
| 落定正文 + 900 帧流（repro12，bug2 不变量）| — | 落定正文 出现/替换/消失 = 1/0/0，正文仍在（裁剪账目：300 条落定 delta 全裁、900 条打开块 delta 全留）|
| JS 错误 | — | 无 |

**工程回归**：`npx tsc --noEmit` ✓、`npm run build` ✓、`go build -o bin/rick ./cmd/rick`（go:embed 重新嵌入 dist）✓。

## Phase 6: 清理事后分析

- 诊断/回归脚本留存 `/tmp/ui-verify/`（repro10_thinking_flash.mjs、repro11_settle.mjs、repro12_think_regression.mjs、repro13_multicycle.mjs）与 `/tmp/think_repro1..5.mjs`（纯函数）——**不入库**。
- 6173 verify 实例：本次为复现而启动（复现前未运行），收尾时停止；使用其 registry 既有会话 `bea28aef-...`（未注入/未修改任何注册数据）。
- 未触碰 git（无 add/commit/stash）。
- **防范模式（沉淀）**：
  1. **环形缓冲裁剪必须区分「已落定」与「打开块」**：纯增量帧可裁的前提是「其完整事实已由结构事件承载」——正在流式的消息不满足该前提（其文本与身份都依赖增量帧本身）。裁 = 内容被吃 + key 漂移 = 闪烁。
  2. **live 块身份不得依赖「首个存活增量帧」**：任何「首个存活」都随裁剪平移；用结构帧锚点（message_end 的 seq）。
  3. **live 块与落定块必须共用同一 React key**：否则 settle 必然重挂 → 进场动画重放 + 展开 override 丢失（用户可见「闪 + 折叠」）。
  4. **滚动跟随用 useLayoutEffect**：绘制后设置 scrollTop = 每帧一格抖动。
  5. 闪烁判定继续沿用「animationstart + 节点 identity」，并区分「新条目挂载」与「同一条目重挂」（多块场景需按块身份键跟踪，不能只看最后一个节点）。

## 结论

「思考过程不断闪烁刷新」根因 = **会话事件环形缓冲的裁剪把正在流式消息（打开块）的增量帧一并裁掉**：
1. live 块 id（原=首个存活 delta 的 env.seq）随裁剪平移 → React key 变 → thinking 块反复重挂 + `chat-item-enter` 进场动画重放（900 帧流实测 **56 次**）= 用户所见「不断闪」；
2. live 文本 = 存活 delta 顺序拼接 → 头部增量被裁 → 文本头持续被吃（`T0,`→`T401,`），内容每帧跳变 = 「刷新感」。

修复 = F1（打开块增量帧永不裁）+ F2（消息块身份锚定前置 message_end，live 与落定共用 key）+ F3（块内跟随改 useLayoutEffect）。
双端 DOM 回归：900 帧流期间 thinking 块 **0 重挂 / 0 额外动画 / 文本头完整**；settle 不再重挂、展开态跨 settle 保持；bug2 不变量（落定正文抗裁剪）仍成立。
