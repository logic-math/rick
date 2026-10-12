# task5：输入框极简（单发送按钮 + Enter 固定换行 + 自适应扩大）

## 任务目标

输入条只留**一个发送按钮**；Enter 固定=换行（删除 enterMode 模式与切换 UI）；textarea 随内容（含长句自动折行）自适应扩大，显示更多内容；保留 Ctrl/Cmd+Enter 发送与 Ctrl+J 换行（隐形快捷键）。

## 关键结果

- KR6a：删除「＋」附件占位、「↵」移动端换行按钮、enterMode 模式切换按钮——输入条仅剩 textarea + 发送按钮
- KR6b：Enter **固定=换行**（删除 ENTER_MODE_KEY/localStorage 逻辑与 readEnterMode/toggleEnterMode）；Ctrl/Cmd+Enter=发送（桌面隐形快捷键）；Ctrl+J=换行保留；Shift+Enter=换行保留；IME 组合中不拦截（isComposing 守卫保持）；空内容/纯空白不发送
- KR6c：textarea **scrollHeight 自适应**：长句自动折行也要增长（不只按 \n 计数），上限放宽（max-h-36 → 约 max-h-60 或 vh 比例），超出上限内部滚动
- KR6d：VERSION → "5.0.13"（cmd/rick/main.go）
- SteerBar 复用同组件自动受益（接口不变：onSend/onCommand/placeholder/disabled/busy）

# 写域

- web/src/components/chat/ChatInput.tsx
- cmd/rick/main.go

# 依赖关系

无

## 实现要点

- 现状（research-I2 §A）：5 元素 4 可点（ChatInput.tsx:288-360）；rows=min(6,\n数) + max-h-36（:306-312）——两个限制点都要改
- 自适应方案：受控 textarea + useEffect 按 scrollHeight 设 rows（或直接 style.height = min(scrollHeight, 上限)）；初始 1 行高度；中文长句折行自然撑高
- 删除 enterMode 后 keydown 逻辑简化：Enter（非 composing）→ preventDefault + insertNewline；Enter+ctrl/meta → send；j+ctrl → insertNewline
- 发送按钮样式保持现有（主按钮）；placeholder 提示改为「Enter 换行 · Ctrl+Enter 发送」
- SteerBar 的 Abort 按钮在状态条（不在输入条），不受影响

## 测试方法

- `cd web && npm run typecheck && npm run build` 绿
- gate3 静态断言：无 enterMode/ENTER_MODE_KEY/↵ 按钮；Enter→insertNewline；ctrl+enter→send；scrollHeight 自适应；VERSION 5.0.13
- 手动路径（dev UI 8414）：Enter 换行 / Ctrl+Enter 发送 / 长句撑高 / 超上限内滚 / 空行不发送 / 移动端单按钮

## 禁止事项

- 只改写域 2 文件；不碰 SteerBar/MessageList（task6/7 域）；不做任何 git 操作；只在 dev 树（/workdir/sunquan20/rick-dev）
