---
summary: "移动端思考流式时「整个上下文闪烁」+折叠块点不开——根因是历史 item id 含构建计数器（不稳定），SSE resync 重建 history 只拉 50 条缩水 254 条 → 全部 item 重挂 + enter 动画重放 = 整页闪烁；override 展开态因 key 变失效。已修：id 用稳定 entry.id + 重建拉全量 + 贴底 anchoring 方向守卫 + 流式 thinking 默认折叠。"
status: "✅ 已解决"
---

# bug1-mobile-stream-flash.md

## Phase 1: 构建反馈回路

**用户报告**：手机端（iPhone Safari）打开会话，thinking 思考持续流式更新时「整个上下文闪烁」；且部分事件块折叠后点不开。

**复现路径（本机 headless iPhone 13 模拟 + MockEventSource）**：
- 修正 MockEventSource 前无法复现——**Mock 有致命格式错误**：mock 把完整 SSE 帧（`id:..\ndata:..\n\n`）塞给 `ev.data`，但原生 EventSource 的 `ev.data` 只是 `data:` 行 JSON → sse.ts `JSON.parse` 失败静默丢弃。**此前所有 worker/桌面"流式稳定性验证"都是无效的**（测的是无流式静态页）。
- 修正 mock（`ev.data = JSON.stringify(env)`）后流式真实渲染（MOCKTHINK 进 DOM）。
- 本机 headless 渲染层有固有抖动（静止页 5 次截图 MD5 全不同、24 万像素差异遍布全屏——swiftshader 环境伪影），像素差分不可靠 → 改用 DOM/布局/动画事件检测。

## Phase 2: 复现最小化

- 真实流式（60 帧 thinking_delta，50ms/帧）期间：scrollTop/scrollHeight/文本位置/文本内容全稳定、DOM 0 mutations → 布局层面无重排。
- 决定性复现：**SSE resync（重发 server_info）触发 history 重建** → 全部历史 item 重挂 → `animationstart` 风暴（rm-chat-in 数百次）。这是「整页闪烁」的真实可测信号。
- 折叠块点不开：点击 thinking 按钮 → useExpandState onToggle 被调用（插桩确认）但 aria-expanded 不变——override 写入失效。

## Phase 3: 可证伪假设

### 假设 1（高优先级，确认）
- **观察证据**：点击前后 render 日志的 useExpandState key 不同（点击前 `think-h-think-2-1`，点击后出现 `h-think-3-1/40-27`）——key 每次渲染在变。
- **推断**：历史 item id 含构建时 `${items.length}` 计数器（viewModel.ts `id: \`h-think-${seq}-${items.length}\``）——history 重建时（若 entries 数量/起始不同）同一消息的 id 变化 → React 全量卸载重挂。
- **验证方法**：插桩打印 key + 触发 resync 看 key 变化与重挂动画。
- **结果**：✅ 确认。buildHistoryItems 的 id 用 `seq`（遍历计数器）+ `items.length`（构建时数组长度）——SSE resync 重建 history（只拉 limit=50，254 条缩水到 50）→ 全量 key 变 → 重挂。

### 假设 2（确认，次因）
- **观察证据**：打开会话 scrollTop=6461 停在历史 21% 处（内容 30087），不贴底。
- **推断**：历史分页前置合并触发浏览器 scroll anchoring（保持视觉位置向上调 scrollTop）→ onScroll → 旧逻辑无条件 `setPinned(distance<60)` → pinned=false → 贴底 effect 不再执行。
- **验证方法**：onScroll 改为方向守卫（scrollTop 减少才解除贴底）后打开贴底正常。
- **结果**：✅ 确认。

### 假设 3（部分确认）
- **观察证据**：流式 thinking 默认展开（isLastStreamingThink=true），展开块增长推下方内容。
- **推断**：在长历史中部展开的增长块每帧推下方布局 → 视觉跳动；用户手动折叠后新一轮 thinking 又自动展开。
- **验证方法**：流式 thinking 默认折叠（defaultOpen=false）后消除。
- **结果**：✅ 折叠后测试通过；用户可手动展开观看。

## Phase 4: 插桩观察

- useExpandState 加 console.log（onToggle 调用 + 渲染时 key/open）——确认 onToggle 被调用、key=think-h-think-2-1 写入，但渲染 key 已变 → override 永不命中。
- 触发 SSE resync（重发 server_info）→ animationstart 计数（rm-chat-in 重放）确认重挂。
- 方向守卫 onScroll 改造后贴底恢复（scrollTop=27304≈底部）。

## Phase 5: 修复回归

**根因确认**：历史 item id 不稳定（含构建计数器）+ resync 重建缩水 → 全量重挂（闪烁）+ override key 变失效（折叠块点不开）。次因：scroll anchoring 误置 pinned=false（打开不贴底）、流式 thinking 自动展开推布局。

**修复（最小改动）**：
1. `viewModel.ts` buildHistoryItems：item id 改用**稳定 entry.id + 块内序号**（`eid()` helper），不再用 seq/items.length 计数器 → 重建时 key 稳定 → React 复用不重挂。
2. `ChatView.tsx` history effect：`hadHistoryRef` 区分首载/重建——重建（SSE resync）时 `getEntries` 拉**全量**（不带 limit）覆盖，不再「50 条覆盖 254 条缩水」→ 内容一致 + id 稳定 → 零重挂。
3. `MessageList.tsx` 贴底：onScroll 方向守卫（仅 scrollTop 减少=用户向上滚才解除 pinned；scroll anchoring 的 scrollTop 增加不误置）→ 打开默认贴底；贴底 rAF 保留防抖。
4. `MessageList.tsx` renderGrouped：流式 thinking 不再自动展开（defaultOpen=false）→ 消除展开块增长推下方布局的跳动源；用户可手动展开单块（override 优先）。

**回归验证**：
- iPhone 模拟：打开贴底 ✓（scrollTop≈底部）；thinking toggle 展开→折叠 ✓；toolGroup 展开 ✓；**流式 + 中途 resync 期间 animationstart=0（无重挂）**✓；override 跨 resync 保持 ✓。
- 桌面：同上全过 ✓；渲染完整（27853 字/27 工具组/29 thinking）✓；无 JS 错误 ✓。
- `go test ./internal/web/...` 全绿 ✓；`npx tsc --noEmit` + `npm run build` + `go build` 全绿 ✓。

## Phase 6: 清理事后分析

- 移除 useExpandState 插桩 console.log（expand.tsx 还原备份）。
- 诊断脚本留存 /tmp/ui-verify/（不入库）。
- **防范模式**：
  1. **React list key 必须基于数据稳定身份**（entry.id/消息 id），禁止用构建时计数器（seq/items.length/index）——否则任何父级重建都会全量重挂 + 动画重放，表现为「闪烁」且交互状态（override/折叠）丢失。
  2. **history 类 REST 快照的重建不得缩水**：resync 重建用全量或保留已加载深度，先缩水再补拉会造成中间态重挂。
  3. **MockEventSource/SSE 测试的 data 字段必须是 JSON 本身**（原生 EventSource 已剥离 `data:` 前缀），否则静默丢弃 → 假验证。
  4. scroll anchoring（内容前置插入保持视口位置）会触发 scroll 事件——解除贴底须用「scrollTop 减少」方向判定而非无条件 distance 判定。

## 结论

移动端流式闪烁的根因是**历史 item 的 React key 不稳定**（构建计数器 id）+ **resync 重建缩水**：任何 SSE 重连都会把 254 条历史重建为 50 条、key 全变 → 全量卸载重挂 + chat-item-enter 动画重放 = 整页闪烁；同时 override 展开态因 key 变而失效（折叠块点不开）。修复 = id 稳定化（entry.id）+ 重建拉全量 + 贴底 anchoring 守卫 + 流式 thinking 默认折叠。修复后流式+resync 期间 0 重挂、折叠交互恢复、双端回归全绿。
