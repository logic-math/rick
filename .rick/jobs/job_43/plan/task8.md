# task8：贴底间距修复（Virtuoso Footer spacer）

## 任务目标

发送消息后，最后一条消息与输入框之间恢复呼吸间距（虚拟化前旧实现的 py-3 底部 12px），不再紧贴。

## 关键结果

- KR9a：MessageList 的 Virtuoso 增加 **Footer** 组件（固定高度 spacer，≥12px，如 `h-3`/`h-4`），随虚拟化机制滚动——贴底时最后一条消息下缘与输入框上缘之间有间距
- KR9b：间距走 Footer 而非容器 padding（Virtuoso 自管滚动区，容器 padding 会破坏其测量）；桌面/移动一致
- KR9c：不影响 followOutput/scrollToIndex 贴底行为与「回到底部」按钮（贴底目标含 Footer，消息不再贴边即可）

# 写域

- web/src/components/chat/MessageList.tsx

# 依赖关系

- task7（虚拟化引入的回归）

## 实现要点

- 旧实现（eeda69d~1）：滚动容器 `py-3` 提供上下 12px——虚拟化重写时丢失
- Virtuoso 用 `components={{ Footer }}` 或 `footer` prop；Footer 返回 `<div style={{ height: 12 }} />`（或 h-3）；引用稳定（useMemo/模块级常量组件，参照现有 Scroller 的稳定引用做法——P0 教训：components 变化会触发重挂）
- 若顶部也需要对称间距（旧 py-3 顶部），已有 Header（加载更早历史）机制——只在有间距缺失时补，勿破坏分页 Header

## 测试方法

- `cd web && npm run typecheck && npm run build` 绿
- gate5 断言：MessageList 含 Footer/footer spacer（height 12 或 h-3）
- 手动路径（dev UI 8414）：发消息 → 最后一条与输入框间有间距；流式输出贴底正常；回到底部按钮正常

## 禁止事项

- 只改 MessageList.tsx；不改数据层/贴底逻辑（followOutput/scrollToIndex 不动，仅加 Footer）；不做任何 git 操作；只在 dev 树
