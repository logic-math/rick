# task2：输入框多行体验（前端 ChatInput）

## 任务目标

会话输入框补齐多行输入体验：Ctrl+J 换行、视觉加高、Enter 语义可切换（中文 IM 习惯）、移动端显式换行按钮。

## 关键结果

- KR3a：`ChatInput.tsx` keydown 增加 **Ctrl+J → 换行**（对齐 pi 终端键位；现有 Shift+Enter 换行与 isComposing IME 保护保持）
- KR3b：输入框视觉加高（min-h 约 2 行高度，rows 自适应逻辑保持 1-6）
- KR3c：placeholder 提示换行键位（如「Shift+Enter / Ctrl+J 换行」）
- KR3d：Enter 语义模式切换：localStorage 持久化（建议 key `rick.chat.enterMode`，值 `send`（默认）/`newline`）；newline 模式下 Enter=换行、Ctrl/Cmd+Enter=发送；**空内容/纯空白不发送**（防误发）；切换入口放 ChatInput 内（小图标/菜单，不动 Settings 页）
- KR3e：移动端（窄屏/pointer:coarse）显示显式换行按钮（如「↵」），点击在光标处插入换行
- SteerBar 复用 ChatInput 自动受益，无需单独改

# 写域

- web/src/components/chat/ChatInput.tsx

# 依赖关系

无

## 实现要点

- 现状：`ChatInput.tsx:191` 为 `<textarea>`；`:176-180` Enter 发送 / Shift+Enter 换行（含 isComposing 守卫）；rows 随内容 1-6 自适应
- Ctrl+J 在浏览器 keydown 中 `e.key === "j" && e.ctrlKey`（注意 preventDefault，避免浏览器默认行为）；换行插入应在**光标位置**（selectionStart/End 处理），插入后保持焦点并滚动到光标
- 模式切换 UI 极简化：一个可点击的小图标按钮（title 提示当前模式），点击在 send/newline 间切换；状态读 localStorage，初始化用 lazy useState
- 移动端判定与现有断点一致（web/src/routes/hooks.ts 的 JS 断点常量，<768px）；换行按钮在移动端常显、桌面端隐藏（或 pointer:coarse 媒体特性）
- **空行不发送**：send 时 trim 后为空 → 忽略（两模式统一）

## 测试方法

- `cd web && npm run typecheck && npm run build` 绿
- 代码级断言（gate1）：ChatInput.tsx 含 ctrl+j 处理、enterMode localStorage、换行按钮、空行守卫
- 手动路径（dev UI 8414）：Enter 发送 / Shift+Enter 换行 / Ctrl+J 换行 / 切 newline 模式后 Enter 换行 Ctrl+Enter 发送 / 空行不发送 / 移动端视口见换行按钮

## 禁止事项

- 只改 ChatInput.tsx 一个文件；不碰 ChatView/Settings/App；不执行任何 git 操作；一切改动只在 dev 树
