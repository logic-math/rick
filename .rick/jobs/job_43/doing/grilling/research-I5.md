# research-I5 简报：「会话消息里 .rick md 文件无法点击打开阅读界面」回归定位（job_43 增量 5 前置）

> 阶段：增量 5 前置调研 | 基准：dev HEAD 2d7a617（= 生产 4b25f12）| 方法：代码链路逐层核对（git diff cbf66a1..HEAD）+ 生产后端 API 实测（带 token 直调）

## 事实性结论（10 条）

1. **前端链接链路代码完好**：Markdown.tsx 的 `a:`→LocalAwareLink / `code:`→LocalAwareCode、LocalAwareLink 的 reader.open 分派、FileReaderContext.ts（.ts 非 .tsx）、ChatView:706 FileReaderProvider 包住全视图（含 :807 MessageList）——cbf66a1（功能引入）→ HEAD **零 diff**（git log 空）。[代码，高]
2. **虚拟化未断 Context**：react-virtuoso ^4.18.15 的 itemContent 是普通 React 子树（非 portal），Context 正常穿透；renderItem→AssistantBubble→Markdown 传参结构与 cbf66a1 一致（仅动画 class 差异）。[代码+库行为，高]
3. **后端生产实测全通**（8413，token 直调，workspace a8ceb938=/workdir/sunquan20/AI_CODING/rick）：`.rick/jobs/job_43/plan/task1.md`（相对）、`/workdir/.../rick/.rick/...`（绝对）、`file://...` 三种形态均 200 返回内容。[运行时实测，高]
4. **dev 树绝对路径被白名单拒绝（实测）**：`/workdir/sunquan20/rick-dev/.rick/jobs/job_43/plan/task1.md` → `invalid_path: path is outside the workspace (and ~/.rick)`——dev 树不在生产 workspace 根也不在 ~/.rick。[运行时实测，高]
5. **本会话（e7dbebd0）消息充满 dev 树绝对路径**：四轮迭代的 task/简报/证据路径全部是 `/workdir/sunquan20/rick-dev/...`——human 在本会话消息里点击这些路径必然落入结论 4 的 invalid_path。[数据事实，高]
6. **invalid_path 时面板行为**：FileReaderPanel 仍会打开（path 已 set，aside 渲染）→ API 报错 → 面板内显示错误文案（非"点击无反应"）。[代码，高]
7. **工具卡输出从不可点击**：ToolCallCard 的 output 是 `<pre>`/`<Ansi>` 纯文本（:275/:323-327），write 工具写的 md 内容在工具卡里**从来没有**可点击路径——若 human 指的是工具卡里的文件内容，那不是回归而是从未实现（路径通常在 assistant 正文的反引号里）。[代码，高]
8. **streaming 态不可点击是设计**：流式中 AssistantBubble 用 StreamText 纯文本直渲（MessageBubble.tsx:150），落定（message_end）后才走 Markdown；历史加载来自 entries（已落定）→ 正常应可点击。挂起/恢复边界：若 live 块无 message_end 即断连，重连 replay 后可能停在 streaming 纯文本（低概率，未复现）。[代码，中]
9. **四次迭代对链路文件的改动仅为样式**：task6 对 FileReaderPanel 只改 bg/blur class；App/ChatView 的 pointer-events/inset/z 层无新增遮挡（diff 核对）。[diff，高]
10. **移动端面板为 fixed inset-0 z-40 全屏**（md:static 右栏）——桌面端右侧栏在 <main> 流内；task3 的 AppShell scrollTop 重置不构成遮挡。[代码，高]

## 根因判定（按可能性排序）

| # | 假设 | 证据强度 | 修复方向 |
|---|------|---------|---------|
| **H1** | **human 点击的是本会话消息里的 dev 树绝对路径**（`/workdir/sunquan20/rick-dev/...`）→ 后端 invalid_path → 面板报错「路径不在工作区内」被感知为"打不开" | 高（结论 4+5 实测链） | 后端白名单**加入 dev worktree**（`<prodRepo 祖父>/rick-dev`——与 release/dev-web 的布局约定一致）；或前端对 invalid_path 显示更醒目的引导 |
| **H2** | human 指的是**工具卡（write bash 等）输出里的 md 内容**——那里从无可点击链接（非回归） | 中 | 工具卡 output 的路径增强为可点击（LocalAwareCode 化）——属新功能非修复 |
| **H3** | 挂起/恢复后 live 块停留在 streaming 纯文本（无 message_end）→ 该条消息永久不可点击 | 低（未复现） | 恢复对账时对无 message_end 的 live 块强制终态化 |
| H4 | 移动端某遮挡层拦截点击 | 极低（diff 已排除） | — |

**无法仅凭代码定论 H1 vs H2**——需 human 一句话澄清：点击的是**消息正文里的路径文字**（反引号/链接）还是**工具卡片展开里的内容**？以及是否面板有弹出但显示错误文案。

## 修复方案选项（供 human 裁决）

- **F1（推荐，小）**：后端 allowedFileRoots 增加 `RICK_DEV_TREE`（默认 `<prodRepo 祖父>/rick-dev`，环境变量已有该语义）——本会话/RSI 场景的 dev 树路径全部可读；只读白名单扩展，安全面不变（仍是显式根列表）
- **F2（小）**：H3 修复：恢复对账强制终态化 live 块（若确认存在）
- **F3（中）**：H2 增强：ToolCallCard 输出里的文件路径 LocalAwareCode 化（新功能）
- F0（免费）：前端 invalid_path 错误文案加复制路径快捷（已有）+ 引导提示

## R7 上报

1. human 实际点击的 UI 位置（正文路径 vs 工具卡内容）——决定 F1 vs F3
2. 面板是否弹出（弹出+报错 → H1；完全无反应 → 需浏览器调试，前端无 instrumentation）

## 叶子文件
无（直接调研；过程在会话记录）
