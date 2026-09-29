# task7 回执：长会话虚拟化（react-virtuoso）

**日期**：2026-09-25 | **worker**: task7-impl | **状态**：完成

## 改动清单

| 文件 | 改动 |
|------|------|
| web/src/components/chat/MessageList.tsx | 渲染层虚拟化重写（414 行变更）：`<Virtuoso>` 替换手写滚动容器；followOutput="auto" + pinned 时 scrollToIndex(LAST,end) 补充（覆盖流式原地增高）；atBottomStateChange 驱动 pinned；startReached 触顶分页；components.Header 顶部「加载更早历史」按钮；firstItemIndex 前置递减（Virtuoso 防跳协议）；computeItemKey=稳定 id；入场动画改为仅「新增尾部条目」播（滚动重挂不闪）；自定义 Scroller 保留 rm-chat-scroll/role=log/aria-live |
| web/package.json | + react-virtuoso ^4.18.15 |
| web/package-lock.json | lock 同步 |
| web/dist/* | npm run build 产物（新 bundle index-KgY9Vklm.js） |

## 数据层未动（任务约束）

MAX_HISTORY / 补拉逻辑 / vm useMemo / events store / ChatView 数据流全部未改（ChatView 仅消费，接口未变）。

## KR8 逐项验证（代码级推演）

- **KR8a** ✅ react-virtuoso 4.18.15 引入（package.json:19）；MessageList `<Virtuoso data={flat}>`
- **KR8b** ✅ followOutput="auto"（追加贴底）+ useEffect([flat, pinned]) scrollToIndex(LAST, align:end)（原地增高贴底——followOutput 只管追加，流式文本增长需补充）；atBottomStateChange=setPinned（上滚解除贴底）；「回到底部」scrollToIndex smooth；初始挂载 initialTopMostItemIndex=length-1
- **KR8c** ✅ Virtuoso 原生动态高度（内部 ResizeObserver 重测量）；入场动画仅 opacity/transform（无布局影响）；ThinkingBlock 自身内部滚动（bodyRef）与列表滚动无交互
- **KR8d** ✅ 前置插入（首条 id 变化+长度增长）→ firstItemIndex -= N（Virtuoso 协议：delta=新增数）；startReached 触顶分页 + Header 显式按钮（调用方 loadingEarlierRef 防抖）
- **KR8e** ✅ ExpandCtx overrides Map 提升在 MessageList（条目卸载不丢）；computeItemKey 单条=envelope id、组=grp-<firstToolId>（**顺带修复**：旧 groupKey 用数组索引 grp-${currentRunStart}，会话增长索引偏移会串展开态——新实现按组身份稳定）

## 自测

- `npx tsc --noEmit` → exit 0
- `npm run build` → ✓ built 2.52s（bundle index-KgY9Vklm.js 含 Virtuoso，1808.86 KiB precache）
- gate4.py → pass=true（7 断言全绿：virtuoso 依赖/import/<Virtuoso>/followOutput/computeItemKey/firstItemIndex 存在）
- 生产零触碰：生产树 `git status --porcelain -uno` 干净（gate4 内含）
- dev 实例 8414 已 serve 新 bundle（curl 验证）

## 关键设计决策（实现中发现并处理）

1. **components 引用稳定性**：Virtuoso 的 components（Scroller/Header）类型变化会卸载重挂（Scroller 重挂=滚动位置丢失）。Scroller useMemo([]) 固定；onReachTop 经 ref 间接（ChatView 传内联 lambda，流式期每次渲染变——直接进 Header 依赖会频繁重挂 Header）；components 对象 useMemo 收敛。
2. **followOutput 盲区**：Virtuoso followOutput 只在**追加**时贴底，流式**原地增高**（长度不变）不覆盖 → 补充 effect scrollToIndex(LAST)。两通道目标一致（底部、瞬时 auto），追加帧冗余幂等，无 smooth 打架。
3. **入场动画防闪**：虚拟化下滚动会卸载/重挂条目，无条件播 chat-item-enter 会滚动全程闪入 → 「已见 id 集」边界：仅尾部新增播；前置历史帧不播；滚动重挂（id 已见）不播。
4. **Virtuoso 必须 height:100%**：外层容器无高度会塌陷为 0（无虚拟化视口）→ style={{height:"100%"}}。

## 遗留 / 建议（parent 验收时关注）

- **手动 UI 验收路径**（无浏览器环境，worker 未做运行时验证）：长会话滚动流畅 / 流式贴底 / 上滚停止+回底按钮 / 顶部触达分页+前置不跳 / ThinkingBlock·工具卡展开态跨滚动保持 / 移动视口滚动手感
- `isPrependFrame` 为渲染期 IIFE 读 ref（effect 期写）——React 严格模式双渲染幂等；生产无影响
- SSE resync 全量重建（首条 id 变+长度增）会被计为前置 → firstItemIndex 漂移递减，无害（FIRST_INDEX_BASE=100_000，MAX_HISTORY=500 不会耗尽）
