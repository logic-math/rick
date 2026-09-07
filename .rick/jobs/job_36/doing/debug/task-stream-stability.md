# 依赖关系
无

# 写域
web/src/styles/theme.css
web/src/components/chat/MessageList.tsx
web/src/components/chat/MessageBubble.tsx
web/src/components/chat/ThinkingBlock.tsx
web/src/components/chat/ToolCallCard.tsx
web/src/components/chat/StreamText.tsx

# 任务目标
打字机流式稳定性三修复（用户实测：pi 流式返回时整个聊天页面闪烁/不稳定——机制已诊断）：
①滚动条出现/消失导致宽度变化→整页文本重排闪烁；②外层贴底滚动每帧无条件拽到底；③流式时全列表组件每帧 re-render。

# 关键结果

1. **scrollbar-gutter: stable**（theme.css 或 MessageList 容器）：消息滚动容器（MessageList 的 `overflow-y-auto` div）加 `scrollbar-gutter: stable`（也可用全局 `.overflow-y-auto { scrollbar-gutter: stable; }` 保守限定）——滚动条出现/消失不再改变容器宽度 → 消除文本重排跳变。验证：流式从不足一屏跨过一屏时，滚动容器 clientWidth 不变。

2. **贴底滚动防抖**（MessageList.tsx）：
   - 现有：`useEffect(() => { if (!el || !pinned) return; el.scrollTop = el.scrollHeight; }, [items, pinned]);` —— items 每帧变 → 每帧无条件拽底。
   - 改为：**跟随条件 = pinned（用户近底 <60px）且内容高度确实变化**；执行用 rAF 合并（一帧内多次 items 更新只滚一次）：
     ```js
     // 只在 pinned 时跟随；rAF 合并避免每帧同步滚动
     const rafRef = useRef(0);
     useEffect(() => {
       if (!pinned || !scrollRef.current) return;
       cancelAnimationFrame(rafRef.current);
       rafRef.current = requestAnimationFrame(() => {
         const el = scrollRef.current;
         if (el && pinnedRef.current) el.scrollTop = el.scrollHeight;
       });
     }, [items, pinned]);  // items 变化仍触发，但滚动合并到 rAF
     ```
     （pinnedRef 保持最新 pinned——注意闭包陷阱；清理 cancelAnimationFrame on unmount）
   - 目标：流式期间 scrollTop 采样稳定（不出现「用户被拽走」的跳变）；打字机正文增长时底部平滑跟随。

3. **流式重渲染优化（React.memo）**：
   - 已落定（非 streaming）的历史 item 组件在 vm 每帧重建时不 re-render：
     - `UserBubble`、`AssistantBubble`（streaming=false 时）、`ThinkingBlock`（streaming=false）、`ToolCallCard`（status!=running）用 `memo` 包裹（浅比较 props——props 含 item 对象，需确保**已落定 item 的对象引用稳定**：ChatView 的 items = [...history.items, ...live]，history.items 在流式期间不变（同一引用）→ memo 生效；live 中已落定（message_end 后移入 st.items 的）item——buildChatViewModel 每次重建 st.items 是新对象数组但 item 对象本身是否新？检查：落定 item 在 applyMessageEnd 从 live 移入 items 时创建新对象；已落定后不再变——但 vm 每次重建整个 st，st.items 里的对象是**每帧新建**还是复用？看 buildChatViewModel——applyEvent 在 st 上累积，st.items 的已落定 item 对象跨帧复用（同一 st 对象？不——每次 buildChatViewModel 都 new BuildState！st.items 每帧全新）→ **memo 浅比较会失败**（item 引用每帧变）！
     - 因此 memo 需要配合**引用稳定化**：要么 buildChatViewModel 缓存已落定 item（复杂），要么用**自定义比较函数**（memo 的第二参数：按 id+内容指纹比较——content 不变则跳过）。简单可靠方案：给每个 item 组件加 `memo(Comp, (prev, next) => 关键字段相同)`——比较 item.id + item 的内容相关字段（text/status/output/toolName 等），streaming=true 的永远不等（允许重渲）。
   - 目标：长会话（250+ 条）流式期间，非活跃 item 不 re-render（可 console.count 或 React Profiler 验证 render 次数）。

4. **回归保护**：现有全部前端功能不回归（消息渲染/折叠/工具卡/历史分页/时间戳/归档/星场不动——写域限定 chat 组件与 theme.css）。

# 测试方法
- npx tsc --noEmit + npm run build
- playwright（6173 验证实例，HOME=/tmp/rick-e2e-home）：①打开真实会话——历史完整渲染（无回归）；②MockEventSource 注入流式 text_delta（50 帧）——scrollTop 采样稳定（不被拽跳）、滚动容器 clientWidth 稳定（scrollbar-gutter 生效）、非活跃组件 render 次数不随帧增长（若可测）；③无页面错误

# 上下文提示
- MessageList.tsx 现结构：scrollRef + pinned state + onScroll + 贴底 effect（145 行附近）
- 流式期间 ChatView 的 items useMemo 每帧重建（history.items 引用稳定——memo 对 history 部分有效；live 落定部分受 st 每帧新建影响——用自定义比较函数兜底）
- React.memo 第二参数自定义比较：`(prev, next) => prev.item.id === next.item.id && prev.item.text === next.item.text && ...`
- 不改行为语义（折叠/展开/贴底 UX 保持），只改稳定性
