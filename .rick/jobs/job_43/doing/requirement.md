进行 rick 的自进化迭代，按照我的需求改进 rick 的功能。不做架构级改动，只做功能级得带
## Grilling 澄清（2026-09-24，进行中）

**根层 L3 追问已发出**（详见 grilling/design-tree.md），4 个判断节点待 human 回答：
- Q1：改进候选选择（research-L1 给出 11 个候选 C1-C11；推荐 C1+C3+C4+C7+C8 打包，或 human 自定需求）
- Q2：迭代规模（小迭代 vs 标准 job）
- Q3：交付面优先级（web-first 战略 vs 候选自然形态）
- Q4：版本与发布流程（完整 RSI S0-S7 + release vs 只合 main）

## Grilling 澄清第 2 轮（L4 回流后，2026-09-24）

根因已落定（research-L1-r2）：①30 分钟空闲回收杀 worker；②SessionPage 无 key + epoch 守卫致切会话显示旧内容；③换行其实已存在（Shift+Enter），真实缺口是 Ctrl+J/移动端/IM 习惯。剩余判断节点见 design-tree.md L3 第 2 轮。

## Grilling 澄清结论（最终——2026-09-24，grilling_gate 已通过）

**O**：rick web 功能级修复迭代——pi 会话持久持有 + 导航实时切换 + 输入框多行体验 + 移动端性能优化，走完整 RSI 流程交付。

**KR（经 human 三轮裁决）**：
- KR1 pi 会话持久化：禁用 30m 空闲回收（server 存续即持久）；保留心跳杀；MaxActive 可配置默认 64（Go：supervisor.go/web.go/config）
- KR2 导航实时性：SessionPage 加 key={session.id} 重挂 + 路由切换滚动重置 + 工作区行点击即跳转（SessionPage.tsx/App.tsx/WorkspaceListNode.tsx）
- KR3 输入框多行：Ctrl+J 换行 + 视觉加高 + Enter 语义可切换（localStorage，默认 send）+ 移动端换行按钮（ChatInput.tsx）
- KR4 移动端性能：「烫」根因=全屏星空 canvas 30fps 常驻+backdrop-blur 每帧重合成，「卡」=流式重渲染+长会话无虚拟化——修：移动端默认静态星空+设置开关 / 移动端去 blur / 窄屏流式节流 100-120ms / dpr 封顶 1.5（StarfieldBackground.tsx/App.tsx/ChatView.tsx/events.ts）
- KR5 走 rick-rsi-loop S0-S7 全流程交付（层门禁 + 人类审批 + release 提升）

**Human 裁决记录**：Q5=A 禁用回收；Q5b 保留心跳；Q5c 默认 64；Q6a/b 全做；Q7 全做；Q9=A1 移动端静态星空+开关；Q10=A 去 blur；Q11 全做；Q12=B 虚拟化推迟下轮。

**遗留（下轮候选）**：P2-A 长会话虚拟化（react-window/virtuoso）；P2-B bundle vendor 分割；交付后真机 profiling 复测。

**产出物**：grilling/design-tree.md（3 层设计树，门禁通过）、research-L1.md（功能面+11 候选）、research-L1-r2.md（三痛点根因）、research-L1-r3.md（移动端性能根因）

## 增量迭代 2（2026-09-25，human 验收反馈）

1. **输入框简化**：当前三个按钮太丑——只要**一个发送按钮**；**Enter=换行**（固定行为）；输入框随文字增加**扩大**，显示更多内容
2. **移动端性能二轮**：发烫有所缓解、没那么卡了，但**仍存在**，需继续优化
