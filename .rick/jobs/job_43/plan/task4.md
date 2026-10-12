# task4：移动端性能优化（前端）

## 任务目标

消除移动端「空闲也发烫」的恒定成本（全屏 canvas 30fps 常驻 + backdrop-blur 每帧 GPU 重合成）并缓解流式期「卡」：移动端默认静态星空+设置开关、移动端去 blur、窄屏流式节流、dpr 降档。

## 关键结果

- KR4a：`StarfieldBackground.tsx` 移动端（<768px）默认**静态渲染**（复用 prefers-reduced-motion 静态代码路径，一帧定格）；设置项开关控制（默认 auto=移动端静态/桌面动画，用户可强制开/关）；canvas dpr 移动端封顶 **1.5**（光栅减 ~44%）
- KR4b：`App.tsx`（侧栏 drawer :250 / 顶 header :282）与 `ChatView.tsx`（chat header :685 / 输入区 :801）的 backdrop-blur 移动端断点替换为不透明底（`md:` 前缀恢复桌面毛玻璃）
- KR4c：`stores/events.ts` 窄屏 SSE 事件 flush 从 rAF 节流到 **100-120ms** 定时（流式期 CPU 降 ~50%，打字机效果无感知差异）；宽屏行为保持
- KR4d：`Settings.tsx` 增加星空模式设置项 UI（auto/动画/静态，localStorage 持久化，独立 key 如 `rick.starfield.mode`）
- 桌面端视觉零回归（默认仍动画+毛玻璃）

# 写域

- web/src/components/starfield/
- web/src/App.tsx
- web/src/components/chat/ChatView.tsx
- web/src/stores/
- web/src/routes/Settings.tsx

# 依赖关系

- task3（App.tsx 顺序写：task3 先改 App.tsx 滚动重置，本 task 再改 blur classNames）

## 实现要点

- 静态路径复用：StarfieldBackground 已有 `prefers-reduced-motion: reduce → 静态星点+固定相位星球/传送门/飞船` 逻辑——抽判定为「shouldAnimate = 用户设置(animated) || (auto && 宽屏 && !reducedMotion)」
- 移动端判定与 task2/task3 断点一致（<768px，或复用 routes/hooks.ts 断点常量；也可监听 matchMedia 变化动态切换）
- dpr：`Math.min(window.devicePixelRatio, 1.5)`（仅窄屏；宽屏保持现有封顶 2）
- blur 替换模式：`bg-space-2/80 backdrop-blur-md` → `bg-space-2 md:bg-space-2/80 md:backdrop-blur-md`（4 处：drawer/header/chat header/输入区；具体类名以现状为准）
- events.ts 节流：flush 调度处（:166-182 一带 rAF 批量+50ms 兜底）窄屏改为 setInterval/setTimeout ~110ms；保证卸载清理
- Settings 设置项读写 localStorage，StarfieldBackground 初始化读取并监听 storage 事件（跨页签同步可选）

## 测试方法

- `cd web && npm run typecheck && npm run build` 绿；`go build ./...` 绿（保险）
- 代码级断言（gate2）：starfield 静态判定+dpr 1.5；App/ChatView 的 md: blur 模式；events 节流常量；Settings 星空设置项
- 手动路径（dev UI 8414 移动视口）：星空静态一帧；无毛玻璃；流式输出流畅；桌面视口动画+毛玻璃保持
- 交付后真机复测（R7-3 遗留）

## 禁止事项

- 不做长会话虚拟化（human 裁决推迟下轮）；不做 bundle 分割；只改写域 5 处；不执行任何 git 操作；一切改动只在 dev 树
