# task7：长会话虚拟化（virtuoso）

## 任务目标

MessageList 渲染层虚拟化：长会话（≤500 envelope）只渲染可视区消息，根治滚动卡顿与 DOM 内存（现状实测 2.5 万 DOM 节点全量渲染）。桌面/移动同享。

## 关键结果

- KR8a：`react-virtuoso` 引入（web/package.json + lock），MessageList 用 Virtuoso 组件渲染消息列表
- KR8b：**流式贴底兼容**：followOutput（或等价）——流式输出时自动贴底，用户上滚则停止跟随（现有 pinned 语义保持）；「回到底部」按钮行为不变
- KR8c：**动态高度兼容**：markdown 渲染高度异步变化（代码块/图片加载）不跳动——virtuoso 原生支持动态高度，验证 ThinkingBlock/ToolCallCard 展开收起高度变化正确
- KR8d：**历史前置插入兼容**：向上加载历史（现有 scroll anchoring / 顶部触达分页逻辑）保持——virtuoso firstItemIndex 机制或等价
- KR8e：展开态 overrides（ThinkingBlock/工具卡折叠状态）跨滚动保持——组件卸载不丢状态（用 stable key + 状态提升，或 followOutput/overscan 足够）
- 数据层不变：MAX_HISTORY/补拉逻辑/vm useMemo 结构不动，只换渲染层

# 写域

- web/src/components/chat/
- web/package.json
- web/package-lock.json

# 依赖关系

- task6（MessageList.tsx 顺序写：task6 先改空态飞船/展开条，本 task 再虚拟化）

## 实现要点

- 先 `npm install react-virtuoso`（dev 树 web/）
- Virtuoso 关键 API：`followOutput={'smooth' | false}`、`firstItemIndex`（前置插入防跳）、`atTopStateChange`（触顶分页）、`computeItemKey`（stable key=envelope 身份锚定，沿用现有 id 锚定方案）
- 现有贴底机制（BOTTOM_SENTINEL/ResizeObserver/身份锚定——bug2/3/4 优化产物）与 Virtuoso 的 followOutput 二选一整合，避免双重滚动控制打架
- 高度变化场景重点自测：ThinkingBlock 展开收起、工具卡展开、代码块 highlight 后高亮、长 markdown
- 移动端滚动性能是主收益（500 条从全量 DOM → 可视区 ~10 条）

## 测试方法

- `cd web && npm run typecheck && npm run build` 绿
- gate4 静态断言：package.json 含 react-virtuoso；MessageList import Virtuoso；followOutput 存在；computeItemKey/firstItemIndex 存在
- 手动路径（dev UI 8414）：长会话滚动流畅；流式贴底；上滚停止跟随+回底按钮；顶部触达分页；展开态滚动保持

## 禁止事项

- 只做渲染层虚拟化，不改数据层（MAX_HISTORY/补拉/事件流不动）；不做任何 git 操作；只在 dev 树
