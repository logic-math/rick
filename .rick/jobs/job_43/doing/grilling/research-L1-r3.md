# research-L1-r3 简报：移动端发烫/卡顿根因 + 端侧优化方法（job_43）

> 阶段：L4 回流第 2 轮 | 主题：web 移动端发烫/卡顿 | 基准：2026-09-24 · dist e72ce7b | 方法：直接代码考古（starfield/stores/sse/构建配置全读，未外包叶子）

## 事实性结论（12 条）

1. **StarfieldBackground 常驻全路由**（App.tsx:233，fixed inset-0 -z-10 全屏 canvas），30fps 帧循环（StarfieldBackground.tsx:31 FRAME_MS=1000/30）；仅 document 隐藏暂停、prefers-reduced-motion 静态化；**无移动端降级**。[代码，高]
2. **每帧绘制成本高**：全屏 clearRect+渐变 fill×2 + ~120 星（arc+十字 stroke）+ 3 个每帧新建径向渐变星云 + 2 星球（辉光渐变/clip/5 椭圆纹理/环）+ 2 传送门（各 10 条 44 段螺线 stroke、每条新建 createLinearGradient→**每帧 20 个渐变对象**，PortalArt.tsx:82-110）+ 2 飞船；移动端 dpr 封顶 2（390×844→780×1688≈1.3M px backing）×30fps≈40M px/s 光栅+数百路径段。[代码推演，高]
3. **backdrop-blur 与动画 canvas 叠加**：顶 header（App.tsx:282 常驻可见）、ChatView header+输入区（685/801）、移动端侧栏 drawer（App.tsx:250 bg-space-2/80 backdrop-blur）——canvas 每帧重绘都使 blur 区域失效→GPU 每帧重合成；移动端 3-4 层 blur 持续重算。[代码，高]
4. **空闲态恒定成本 = canvas 30fps + blur 重合成**——与使用强度无关，屏幕亮着就烧；「手机一直烫」主根因。[代码推演，高]
5. **流式期成本**：events store rAF 批量 flush（50ms 兜底）每次整体换 buffers Map 引用（stores/events.ts:166-182）→ChatView 选择器取新数组（ChatView.tsx:261）→vm useMemo 重建遍历 ≤500 envelope（:391）→live 块重渲染+markdown 重解析+ThinkingBlock useLayoutEffect 贴底（ThinkingBlock.tsx:42-48）。已多轮优化（BOTTOM_SENTINEL/ResizeObserver/身份锚定），仍是流式期主 CPU 项。[代码，高]
6. **长会话无虚拟化**：MAX_HISTORY=500 自动补拉全渲染（ChatView.tsx:41/365-390），无 react-window/virtuoso；bug4 注释实证长会话 2.5 万 DOM 节点——「卡」（滚动/内存）根因。[代码，高]
7. **SVG g CSS transform 动画不可合成**（Portal.tsx:20-27 自证注释）：Saucer flying（rm-saucer-hover 2.2s+rm-beam 1.6s 无限动画）在工具运行期常驻（MessageList.tsx:54/246、SteerBar、MonitorView）+ ToolElapsed setInterval 每秒（ToolCallCard.tsx:139）——工具长跑期附加主线程逐帧样式重算。[代码，高]
8. **SSE 健康**：指数退避重连+seq 去重+游标持久化（api/sse.ts:437-452）；stores 零轮询（grep setInterval 仅 events/sse 批处理与退避）。非发烫因素。[代码，高]
9. **CSS 动画面小且已收敛**：rm-pulse 仅重连/运行徽章；rm-chat-in 0.15s 单次；每消息 Portal spin 已移除（历史优化）。[代码，高]
10. **bundle 1.7MB 单 chunk**（dist/assets/index-CQywn-az.js，无 manualChunks/code splitting；rehype-highlight lowlight common ~35 语言全打包）——仅首载成本（SW 预缓存），非热因素。[构建产物，高]
11. viewport meta 正确（index.html:5）；无自定义 touch 监听；无 will-change 滥用；PWA SW 仅预缓存被动。[代码，高]
12. **历史优化集中在流式路径**（bug2/3/4），**空闲恒定成本（canvas+blur）从未处理**——与「一直烫」症状精确吻合。[代码+debug 注释，中高]

## R7 上报项（4 项）

1. 星空背景是否为产品视觉身份核心（human 裁决）——决定移动端默认静态化取舍
2. 移动端去毛玻璃换不透明底的视觉裁决
3. 无浏览器环境，未做真机 profiling——静态推演排序，建议交付后真机复测
4. MessageList 虚拟化与流式贴底/展开态交互复杂——是否本轮做需裁决

## 嫌疑点排序（发烫贡献度）+ 端侧优化选项

| # | 嫌疑点（贡献度） | 优化选项 | 量 | 风险/裁决点 |
|---|---|---|---|---|
| P0-A | 全屏 canvas 30fps 常驻（idle 60-80%） | 移动端默认静态（复用 reduced-motion 代码路径）+ 设置项开关（桌面默认开）；折中：移动端 10fps | 小 | 视觉变更需 human 裁决 |
| P0-B | backdrop-blur×动画 canvas 重合成（GPU 放大器） | 移动端断点去 blur 换不透明底（md: 前缀恢复桌面）：侧栏/header/chat header/输入区 4 处 className | 极小 | 视觉变更需 human 裁决 |
| P1-A | 流式 flush 每帧全量链（React+markdown） | 窄屏 flush rAF→100-120ms 定时节流，流式期 CPU 降 ~50% | 小 | 打字机 100ms 仍流畅，感知无差 |
| P1-B | canvas dpr=2 光栅偏重 | 移动端 dpr 封顶 2→1.5（光栅减 ~44%） | 极小 | 星点略软（背景装饰可接受） |
| P2-A | 长会话 500 条无虚拟化（卡+内存） | react-window/virtuoso 虚拟化 | 中 | 与贴底/展开态交互复杂，建议独立 task |
| P2-B | 1.7MB 单 bundle | vendor 代码分割 | 小 | 仅首载收益，非热因素 |
| 可选 | 空闲超时仍全速动画 | canvas 无交互 N 分钟→静态帧「屏保」 | 小 | 唤醒时机语义需设计 |

**独立性**：P0-A/P0-B/P1-A 文件集零重叠（StarfieldBackground.tsx / App.tsx+ChatView.tsx / events.ts），可并行 task；P1-B 与 P0-A 同文件不同段，串行安全。

## 结论

「很烫」= 空闲恒定成本（canvas 动画+blur 重合成）——P0-A/P0-B 根治；「很卡」= 流式重渲染+长会话 DOM——P1-A 缓解、P2-A 根治。历史优化都打在流式路径上，空闲成本是本次盲区。

## 叶子文件

无（有界直接调研，过程在会话记录）。
