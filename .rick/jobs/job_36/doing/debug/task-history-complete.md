# 依赖关系
无

# 写域
web/src/components/chat/ChatView.tsx
web/src/components/chat/MessageList.tsx

# 任务目标
会话历史完整性（用户实测：254 条 entries 的会话首屏只显示最近 50 条，更早历史需手动滚动才加载——体验不完整）。方案 A+B：挂载时自动补拉历史直到「能看到最早或达到上限」+ 顶部「加载更早」按钮兜底。

# 关键结果

1. **A：挂载自动补拉**（ChatView.tsx 历史拉取 useEffect）：
   - 现状：挂载拉 `limit=50` 一页；若 `entries.length === 50`（可能还有更早）设 hasMore，但**不自动继续拉**
   - 改为：首屏拉 50 后，若 hasMore（满页），**自动循环补拉**（每次 before=当前最早已加载 entry id，拉 50，前置合并）直到满足任一停止条件：
     - 返回不足一页（拉到底了）→ hasMore=false
     - 累积条目达上限 `MAX_HISTORY = 500`（防超长会话卡死——500 条历史对 254 条会话足够显示全部；超长会话显示最近 500 条 + 顶部「加载更早」按钮继续）
   - 补拉期间不阻塞首屏渲染：先渲染第一页，补拉完成后 setHistory 一次性合并（items 前置 + fingerprints 并集——复用现有 loadEarlier 的合并逻辑）
   - 实现注意：合并逻辑与现有 `loadEarlier` 相同（items: [...built.items, ...prev.items]），可抽公共函数 `prependHistory(built)` 复用；补拉循环用 async/await 顺序执行（避免并发竞态），loadingEarlierRef 防重入
   - 目标：254 条会话打开即见**全部历史**（含最早 bootstrap 消息）；500+ 条会话见最近 500 条

2. **B：顶部「加载更早历史」按钮**（MessageList.tsx + ChatView 传参）：
   - MessageList 顶部（消息列表第一项之前）在 `hasMore` 时显示一个居中按钮：「↑ 加载更早历史 (N+)」——点击调 `onReachTop` 同款逻辑（loadEarlier 拉 50 前置）
   - ChatView 传 `hasMore={paging.hasMore}` 给 MessageList；按钮 busy 态（加载中「加载中…」禁用）；拉到底（hasMore=false）后按钮消失
   - 保留现有「滚动到顶自动加载」（scrollTop<=32 触发）作为补充；按钮是显式兜底（用户可见可点）
   - 按钮样式：小号 ghost 居中（R&M 主题：border-line text-ink-3 hover:text-portal）

3. 回归保护：既有渲染/分页/折叠不回归；补拉与 SSE live 事件并存不冲突（fingerprint 去重已有）

# 测试方法
- npx tsc --noEmit + npm run build
- playwright（6173 验证实例）：真实 254 条会话 aed8d899-766d-4139-8840-0ea5432a495b 打开——**首屏即含最早消息（bootstrap「开始：按系统提示词」文本可见）**（验证自动补拉 254 条全显）；若造超长 mock 则验证顶部按钮出现且点击加载更多；无页面错误

# 上下文提示
- ChatView 历史拉取 useEffect（~270 行）：api.getEntries(sessionId, { limit: HISTORY_PAGE_SIZE }) + setHistory(buildHistoryItems(...)) + setPaging
- loadEarlier（~363 行）：已有 before 游标拉取 + prepend 合并逻辑——抽公共函数复用
- HISTORY_PAGE_SIZE=50（37 行）；新加 MAX_HISTORY=500 常量
- paging state：{ earliestId, hasMore }
- MessageList 顶部渲染点：renderGrouped 输出之前（items 数组最前）
