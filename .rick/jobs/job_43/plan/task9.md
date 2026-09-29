# task9：会话信息「收起原文」toggle 修复

## 任务目标

会话信息面板的「查看原文/收起原文」按钮改为真正的 toggle：首次点击加载并展开；已加载时点击收起/再展开。

## 关键结果

- KR10a：ChatView.tsx 的按钮 onClick 改 toggle 语义：`!prompt` → loadPrompt()（加载+展开）；`prompt && promptOpen` → setPromptOpen(false)（收起）；`prompt && !promptOpen` → setPromptOpen(true)（再展开）
- KR10b：加载失败态（promptError）时按钮仍可重试（点击再 loadPrompt——注意守卫需允许 error 后重试，prompt 仍 null 可重进）
- KR10c：加载中禁用（promptLoading）保持；文案逻辑不变（加载中…/收起原文/查看原文——可按 promptOpen 精化：prompt && promptOpen ? "收起原文" : prompt ? "展开原文" : "查看原文"）

# 写域

- web/src/components/chat/ChatView.tsx

# 依赖关系

- task8（ChatView 同文件族顺序无冲突，仅保持 pipeline 顺序）

## 实现要点

- 根因：`loadPrompt` 守卫 `if (... || prompt || promptLoading) return`——第二次点击提前 return，`promptOpen` 无任何置 false 路径（旧缺陷，非虚拟化回归）
- 修法：新增 `togglePrompt()`（或在 onClick 内联）：已加载则 `setPromptOpen(v => !v)`；未加载才走 loadPrompt；loading 中直接忽略
- 不动 Collapse 组件与 API 层

## 测试方法

- `cd web && npm run typecheck && npm run build` 绿
- gate6 断言：ChatView 含 toggle 逻辑（setPromptOpen(false) 存在 / promptOpen 取反）
- 手动路径（dev UI 8414）：查看原文 → 展开；收起原文 → 收起；再点 → 展开；加载失败 → 重试

## 禁止事项

- 只改 ChatView.tsx；不做任何 git 操作；只在 dev 树
