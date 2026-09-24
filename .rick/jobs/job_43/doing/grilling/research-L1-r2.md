# research-L1-r2 简报：rick web 三痛点根因调研（job_43 L4 事实回流）

> 阶段：L4 根层回流 | 主题：pi 会话持久化 / 左侧导航实时性 / 输入框换行 | 基准：2026-09-24 · 生产 e72ce7b-260924021322（源码同版） | 方法：直接代码考古（文件集预定位，有界调研，未外包叶子）

## 事实性结论（12 条）

1. **worker 与 SSE 连接完全解耦**：`ServeSSE` 断开仅 `Unsubscribe`（internal/web/sse.go:262-280），不触碰 worker——页面切换/浏览器关闭/SSE 断连均**不会**杀 pi 进程。[代码，高]
2. **pi worker 正常情况下常驻**：closeSession 仅由用户显式动作触发（SessionClose/SessionArchive，sessions.go:740/839）；server 退出才 ShutdownWorkers（server.go:181，挂起→suspended 语义）。[代码，高]
3. **痛点1 根因 = 30 分钟空闲回收**：`DefaultIdleTimeout = 30m`（supervisor.go:44，注释"idle web sessions are expensive"）；`reapLoop`（supervisor.go:806-833）在非 streaming 且 `time.Since(lastActivity) > idle` 时 `w.Close()+markDead("idle reap")`。activity 仅由用户命令（writeLine:468）与非 response 事件（stage:710）刷新——挂机 30 分钟必被杀。[代码+测试佐证，高]
4. **被回收后会话变 error 态**：pumpWorker 检测 worker 终止且非用户关闭 → `markWorkerLost` → status=error（sessions.go:1802-1815, 1922-1931），前端显示「已中断」+ Resume 按钮——用户感知"会话被断开"，需手动恢复。[代码，高]
5. **禁用回收的现成语义**：`NewSupervisor` 中 `IdleTimeout==0→默认30m`（supervisor.go:140-141），`>0 才启动 reapLoop`（:310-311）——传**负值**即禁用；但组合根 web.go:236-239 硬用默认值，无配置暴露（config 无 idle 字段）。另 `MaxActive=8` 上限（supervisor.go:43）：持久化后并发 worker 累积，>8 时 Spawn 报 ErrMaxActive。[代码，高]
6. **痛点2 主根因 = 切会话显示旧会话内容**：`/session/:id` 路由下 SessionPage 不重挂（SessionPage.tsx:59-62 无 key）；ChatView 的历史拉取 useEffect（ChatView.tsx:316-386）有 epoch 守卫 `if (historyEpochRef.current === epoch) return`（:319），而 **epoch = resyncCount + historyRefresh 不含 sessionId**——切 A→B 时 effect 重跑但 epoch 相同提前返回，**B 的历史永不拉取，界面保留 A 的消息**（history state 未清）。自 web UI 初版（task9-11，2026-08-30）即存在。[代码逻辑推演，中高；未运行时复现]
7. **MonitorView 无此 bug**（doing 会话监控页按 sessionId 依赖正常拉取，MonitorView.tsx:91-108）——bug 仅影响交互型会话（plan/easy/ctrl/human-loop/learning/interactive dream）。[代码，高]
8. **痛点2 次要因素**：①`<main>` 是滚动容器且路由切换不重置 scrollTop（App.tsx:310）——长页切短页视觉"没刷新"；②工作区行点击=展开/折叠而非导航（WorkspaceListNode.tsx:100-135，默认折叠）；③Dreams 页是占位页（dream 日志"待后端接口支持"，Dreams.tsx:36-39）。[代码，高]
9. **痛点3 事实：换行功能已存在**——ChatInput 是 `<textarea>`（ChatInput.tsx:191），Enter 发送/Shift+Enter 换行（:176-180，含 isComposing IME 保护），rows 随内容 1-6 自适应，SteerBar 复用同组件；NewSessionModal 需求描述也是 textarea（:399，min-h-24）。生产 dist（e72ce7b）已含该逻辑。[代码+生产 grep，高]
10. **痛点3 体验缺口 4 项**：①缺 **Ctrl+J**（pi 终端 newline = shift+enter **或 ctrl+j**，pi keybindings.d.ts:144；Windows Terminal 下 Shift+Enter 不可传）；②移动端软键盘无 Shift → Enter 恒发送，**无法换行**；③视觉单行（min-h-24px + rows=1）；④中文 IM 习惯（QQ 式 Enter=换行/Ctrl+Enter=发送）未支持。用户实际痛点场景需 human 澄清（R7-2）。[代码+pi 运行时，高]
11. **三痛点修复相互独立**：痛点1 改 Go 后端（internal/runtime/supervisor.go + internal/cmd/web.go + config）；痛点2 改前端路由层（SessionPage.tsx 加 key，可选 scroll 复位）；痛点3 改前端 ChatInput.tsx。文件集零重叠，可并行 task。[代码结构，高]
12. **测试基建**：后端有 TestSupervisor_IdleReap（supervisor_test.go:501）等完整测试；前端无单测（仅 tsc typecheck + vite build 门禁）——前端修复验证靠 typecheck+build+手动验收路径。[代码，高]

## R7 上报项（4 项）

1. 痛点1：human 的"不主动断开"是否含**心跳超时**路径（pi 卡死 60s 无响应即杀）——若含，需改 heartbeat 语义或加自动重启策略，工作量升级
2. 痛点2：human 遇到的具体场景未确认——是"切不同会话显示旧内容"（主根因）还是"Sessions/Jobs/Dreams 切换不刷新"（次要因素）还是移动端 drawer？影响修复范围（建议 human 补一句或全修）
3. 痛点3：human 的使用环境（桌面 Chrome/移动端/Windows Terminal 习惯）决定 Ctrl+J / Ctrl+Enter / 移动端换行按钮哪个是主修项
4. MaxActive=8 与持久化的张力：若 human 常开 >8 会话需同步调整上限（或做 LRU 降级），需裁决

## 修复方向选项（供 L3 追问与 plan 参考）

### 痛点1：pi 会话持久化
- **A（推荐，小）**：组合根禁用空闲回收（web.go 传 `IdleTimeout: -1` 或引入 `DisableIdleReap` 显式语义）+ MaxActive 提升为可配置（默认 8→16 或 config 暴露）。改动面：web.go ~10 行 + supervisor.go 语义澄清 + TestSupervisor_IdleReap 不受影响（测试显式传值）。风险：长挂机 worker 累积内存（pi 进程 ~百 MB 级）——单用户本地工具可接受；文档标注。
- **B（中）**：A + 回收前自动 suspend（而非 error），用户回来点一下即恢复（保留现有挂起语义，不违"绝不自动续跑"）。改动面：+reapLoop 回调区分 idle-reap 与 crash（OnDead 已带死因可判）。
- **C（大）**：worker 死亡自动透明 respawn（`--session <id>` 恢复）——**违背 human 已裁决的"绝不自动续跑"原则**（pi 不修复悬挂 toolCall），不建议。

### 痛点2：左侧导航实时性
- **A（推荐，极小）**：`SessionPage.tsx` 给 ChatView/MonitorView 加 `key={session.id}` 强制重挂——一行修复主根因（epoch 守卫、history、refs、滚动全部随 key 重置）。
- **B（小，建议叠加）**：路由切换重置 `<main>` scrollTop（App.tsx 加一个 location pathname useEffect）。
- **C（可选）**：工作区行点击直接导航到该工作区 sessions（折叠/展开改由箭头图标承担）——交互变更需 human 裁决；Dreams 页占位属 research-L1 候选 C6，不在本痛点最小集内。

### 痛点3：输入框换行
- **A（推荐，极小）**：ChatInput 增加 **Ctrl+J 换行**（对齐 pi 终端键位）+ placeholder 提示补 Ctrl+J；textarea min-h 提升到 ~2 行视觉。
- **B（小）**：A + **Ctrl/Cmd+Enter 发送、Enter 换行**模式切换（设置项持久化 localStorage）——适配中文 IM 习惯与移动端（移动端 Enter 即换行，发送靠按钮）。
- **C（中）**：B + 移动端显式换行按钮。风险：Enter 语义反转需防误发（空行不发）；发送模式切换增加设置 UI 面。

## 叶子文件
- 无外包叶子（文件集预定位的有界代码考古，自己完成；调研过程在会话记录）
