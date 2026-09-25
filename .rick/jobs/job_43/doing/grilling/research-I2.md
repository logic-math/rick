# research-I2 简报：输入框现状 + 移动端剩余发烫/卡顿源（job_43 增量迭代 2）

> 阶段：增量迭代 2 调研 | 基准：dev HEAD 9d24bf9（= 生产 5.0.12）| 方法：直接代码考古（ChatInput/Saucer/Starfield/events/stores 全读 + grep 全量盘点）

## A：输入框现状（为「单按钮 + Enter=换行 + 自适应扩大」改造）

1. **当前输入条 5 元素、4 可点**（ChatInput.tsx:288-360）：附件占位「＋」(disabled) / textarea / 移动端换行按钮「↵」(md:hidden) / Enter 语义切换按钮「↵发送|↵换行」(常显) / 发送按钮。用户嫌丑的就是中间三个（＋、↵、模式切换）。
2. **Enter 语义已可切到 newline**（enterMode，localStorage `rick.chat.enterMode`）：newline 模式 = Enter 换行 / Ctrl+Enter 发送（ChatInput.tsx:229-241）。human 要的「Enter=换行」= 把 newline 变成**固定行为**（删除模式切换按钮与设置，或默认翻转）。
3. **textarea 扩大逻辑两个限制点**（:306-312）：①`rows=min(6, 换行符数)`——**只按 \n 计数，长句自动折行不增长**（一行很长文字 rows 仍是 1）；②`max-h-36`(144px) 硬顶。用户「随文字增加扩大」需改 scrollHeight 自适应 + 更大上限（如 max-h-48~60 或 vh 比例）。
4. **SteerBar 完全复用 ChatInput**（SteerBar.tsx:119）——steer 场景自动同步单按钮新 UI；streaming 时 ChatInput 上方另有 Abort 按钮（SteerBar.tsx:105-110，在状态条里，不在输入条内）。
5. **Ctrl+J/insertNewline/空行守卫/IME 保护**均为独立函数（:95-110/:143-172），简化按钮不影响。

## B：移动端剩余发烫/卡顿源（二轮）

**已生效确认**：starfield 静态路径正确——移动端 reevaluate()→stopAnimation()，无 rAF 循环（StarfieldBackground.tsx:384-397/452）；resize 仅事件驱动重绘一帧。SSE 无心跳轮询（仅断线指数退避重连）；stores 无 setInterval 轮询。blur 4 处已按 md: 断点切换。→ 空闲态（无会话活动）恒定成本≈0。

**剩余源（按嫌疑排序，均为「会话活跃/流式期」成本）**：

6. **P0：空会话空态的 Saucer 飞船常驻动画**（MessageList.tsx:246）：`<Saucer size={64} flying>`——「会话已就绪」空态常驻 SVG SMIL `<animate>`（航灯 5 个无限闪烁）+ `rm-saucer-hover`/`rm-beam` CSS 无限动画（Saucer.tsx:47/37/74-82）。SVG transform 动画不可 GPU 合成 → 主线程逐帧样式重算。**新开会话停在空态=持续烧**。
7. **P0：Saucer flying 于工具运行期常驻**：SteerBar:54（streaming 状态条）+ MessageList:54（工具组 running）+ MonitorView:176（任务 running）——agent 长跑（分钟级）期间同样无限动画。手机跑 rick 任务=全程烧。
8. **P1：展开控制工具条 backdrop-blur 常驻**（MessageList.tsx:284）：时间线顶部悬浮条 `bg-surface-raised/90 backdrop-blur` **无 md: 降级**——流式期它下方内容每 110ms 变化 → blur 层每帧 GPU 重合成。同族漏改（job_43 首轮只改了 4 处，此处第 5 处）。
9. **P1：状态点 rm-pulse 无限动画常驻**：ChatView:93（active 会话 2.4s 呼吸）+ ChatView:83（streaming 0.9s）+ StreamText:23（流式光标 0.9s 闪烁）+ SessionBadge:90 / WorkspaceListNode:125（侧栏活跃徽章）+ StatusDot.tsx:55。active 会话存在时侧栏+头部恒有 1-2 个 opacity 动画（opacity 可合成，成本低于 transform，但叠加）。
10. **P1：ToolElapsed 每秒 setInterval**（ToolCallCard.tsx:138-147 useTick）：每个运行中工具卡每秒 setState 重渲（独立子组件设计，父卡 memo 不受累）——秒级成本小，但多工具并行时叠加。
11. **P2：长会话 500 条无虚拟化**（ChatView.tsx:41-42/388-392）：vm=useMemo 重建 ≤500 envelope + history 合并 + MessageList 全量渲染；bug4 实证 2.5 万 DOM 节点。流式期每 110ms：buffers 换引用 → 选择器 → vm useMemo → live 块重渲 + markdown 重解析 + ThinkingBlock useLayoutEffect 贴底。滚动/内存=「卡」主源。虚拟化方案评估见 §C。
12. **P2：其他 blur 残留（按需修）**：App.tsx:114（断线横幅，仅 reconnecting/closed 时显示）、MessageList.tsx:284（见 #8）、Dialog.tsx:53（模态遮罩，开时才有）、TokenGate.tsx:35（锁屏）、FileReaderPanel.tsx:92（移动端全屏阅读器 `bg-space/95 backdrop-blur` **常驻**——移动端看文件全程 blur）。

## C：优化选项（供 human 裁决/plan 参考）

- **O1 空态飞船静态化**（极小）：MessageList 空态 `<Saucer flying>` → `flying={false}` 或静态装饰；视觉损失小（空态本来就静）。
- **O2 工具运行期动画移动端降级**（小）：Saucer 的 flying 动画（SMIL+CSS）在窄屏关掉：SMIL `<animate>` 改条件渲染、CSS animation 窄屏 class 关闭；桌面保持。或统一「窄屏 all animation-duration:0.01ms」全局降级（theme.css 已有 prefers-reduced-motion 同款块，加 `@media (max-width:767px)` 即可，一行级改动但影响所有动画含流式光标）。
- **O3 展开条+断线横幅+阅读器 blur 降级**（极小）：MessageList:284/App.tsx:114/FileReaderPanel:92 补 md: 模式（同首轮 4 处手法）。
- **O4 状态点脉冲降频/降级**（极小）：rm-pulse 2.4s→更慢或窄屏关；SessionBadge/WorkspaceListNode 活跃徽章窄屏不 pulse。
- **O5 ToolElapsed 移动端降频**（极小）：useTick 窄屏 1s→5s（或 >=2min 才警示色场景保持 1s）。
- **O6 长会话虚拟化**（中，human 已改口「应继续优化」）：候选 react-window/virtuoso。兼容点：①流式贴底（pinned rAF 滚动）②历史前置插入（scroll anchoring）③展开态 overrides Map ④ThinkingBlock useLayoutEffect 贴底 ⑤回到底部按钮 ⑥顶部触达分页。**推荐 virtuoso**（动态高度原生支持 + followOutput API 对贴底友好），迁移面 MessageList 一层。风险：与 scroll anchoring/分页逻辑重写量大，建议独立 task + 桌面/移动同享。
- **O7 输入框改造**（A 部分，小）：删「＋/↵/模式切换」三控件 → 仅发送；Enter 固定=换行（删 enterMode 或默认 newline）；textarea scrollHeight 自适应 + 上限放宽（max-h-48~60）；Ctrl/Cmd+Enter 发送保留（桌面隐形快捷键）；Ctrl+J 保留。

## R7 上报（无法自行澄清）

1. Enter=换行是**删除** enterMode 设置还是保留设置但默认 newline？（human「只有一个发送。按回车键是换行」语义偏删除=固定行为，但删设置是不可逆交互简化，需确认）
2. 空态飞船/运行期动画降级选「仅移动端关」还是「全局关」（O2 两案）？
3. 虚拟化本轮做不做（工作量中，交互风险最高单项）？

## 叶子文件

无（有界直接调研；过程在本会话记录）
