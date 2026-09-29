# task11：前端全局工具箱「打开文件」入口

## 任务目标

顶栏右上角加「工具箱」入口 → 路径输入窗口 → 提交后打开 md 阅读界面（FileReaderPanel 全局挂载，任何页面可用）。

## 关键结果

- KR12a：App.tsx 顶栏右侧（Settings 齿轮旁）加工具箱按钮（扳手/箱子图标，aria-label「打开文件」）→ 点击弹出 Dialog（复用现有 Dialog 组件）：输入框（placeholder 提示支持绝对路径/相对当前 workspace/.rick 路径）+「打开」按钮 + Enter 提交
- KR12b：**FileReaderProvider + FileReaderPanel 提升到 App 壳层全局挂载**：ChatView 移除本地 openFile state/Provider/Panel，改用全局 Context（reader.open 行为不变——消息里的链接点击照旧工作）；顶栏工具箱提交路径 → 全局 reader.open(path)
- KR12c：移动端：面板本就是 fixed inset-0 全屏（FileReaderPanel 现状），全局挂载后行为保持；Dialog 移动端可用
- KR12d：输入校验：空路径不提交；打开失败（invalid_path 等）错误显示在 FileReaderPanel 内（现状）——Dialog 可顺带显示最近一次错误提示（可选，简洁优先）
- KR12e：ChatView 移除后回归验证：会话页消息里的 LocalAwareLink 点击 → 全局面板打开（行为与提升前一致）

# 写域

- web/src/App.tsx
- web/src/components/chat/ChatView.tsx
- web/src/components/files/FileReaderPanel.tsx
- web/src/components/files/FileReaderContext.ts

# 依赖关系

- task9（pipeline 顺序；与 task10 后端并行，文件集不相交）

## 实现要点

- FileReaderContext.ts 现有 `FileReaderApi { open, current }`——全局化后 current（当前打开路径）状态放 App 层 useState，open=（p)=>setOpenFile(p)
- ChatView:657-658 readerApi/706 Provider/841 Panel 三处移除，import 改从 App 侧（或 Context 文件导出 hook 不变——ChatView 内 useFileReader 已有，仅去掉 Provider 包裹）
- Dialog 组件：web/src/components/common/Dialog.tsx 已存在（grep 确认签名），复用；输入框受控 + autofocus
- 图标：现有 Portal/Saucer 风格 SVG 小图标（简单扳手或文档图标）
- App.tsx 顶栏结构：ConnectionDot + Settings 齿轮——工具箱按钮插在齿轮左侧

## 测试方法

- `cd web && npm run typecheck && npm run build` 绿
- gate7 断言：App 含工具箱入口（工具箱/打开文件 aria-label 或 title）+ 全局 FileReaderProvider；ChatView 不再含 FileReaderProvider
- 手动路径（dev UI 8414）：任意页面点工具箱 → 输入 `.rick/jobs/job_43/plan/task1.md` → 右侧打开阅读；输入其他 workspace 绝对路径（dev 树，需 task10 后端配合）→ 可读；会话页消息链接点击照旧

## 禁止事项

- 只改写域 4 文件；不动 MessageList/Markdown/LocalAwareLink（链接链路保持）；不做任何 git 操作；只在 dev 树
