# 设计树：rick 自进化迭代（job_43）

## 第 1 层（根层）：O / KR——全局目标

> 状态：**进行中（等待 human L3 追问澄清）**

**O（Objective）**：对 rick 进行一轮功能级（非架构级）自进化迭代——按 human 指定的具体功能需求，交付可验证的改进并合入生产，rick 版本从 5.0.11 前进一个功能版本。

**KR（候选集，待 human 澄清需求后收敛）**：
- KR1（**待澄清**）：改进方向已被 human 明确指定（哪些功能面：CLI / Web / loop 体系 / tools 校验 / 其他），且改进项具体到可验证
- KR2（**待澄清**）：改动不触碰架构约束（DIP 组合根、四层架构、web-first 战略、~/.rick/pi 隔离），仅功能级
- KR3（**待澄清**）：全部改动通过门禁（go test + 相关集成测试）+ rick-gates 校验，走标准 RSI/doing 流程提交合入
- KR4（**待澄清**）：新功能有对应的验证方式（测试/dry-run/手动验收路径），并在 domain/learning 中留痕

**本层 pipeline（模块划分——先粗后细，下钻后细化）**：

```
[需求澄清] → [方案设计（设计树下钻）] → [任务分解（plan）] → [实现（doing）] → [验收+留痕（learning/RSI）]
```

**L1 调研消解记录（轻量自查 + research-L1 简报）**：
- 轻量自查：domain/README、architecture.md、commands.md、bugs.md、git log、CLI help、internal/web 文件清单
- 重量级调研：research-L1.md 已落盘（12 条事实 + 11 个改进候选 C1-C11 + R7 上报 6 项）
- 关键事实：①skills 格式校验空转（P0，loops_check 实测 `skills 0`）；②无 job 列表/状态 CLI；③dream 积压 14 个全成 job；④Dreams 页占位 + learning 产物不可 web 浏览；⑤domain 文档漂移 3 处；⑥F6 level_complete 写回时序为 job_36 指定遗留

**L2 提炼后的判断节点（L3 追问依据）**：

| # | 判断节点 | 调研依据 | 为何需 human 裁决 |
|---|---------|---------|------------------|
| J1 | **改进候选选择**：C1-C11 中选哪些？human 原文说"按照我的需求"——需求具体内容尚未给出 | research-L1 候选清单（11 项均有工作量/价值/风险论证） | 纯意图裁决；且 human 措辞暗示有既定需求 |
| J2 | **迭代规模**：小迭代（1-2 候选）还是标准 job（3-5 候选打包） | 候选工作量：极小×1 / 小×6 / 中×3 | 投入度与交付节奏裁决 |
| J3 | **交付面优先级**：CLI / Web / tools 校验，与 web-first 战略的关系 | 候选横跨 CLI（C3/C4/C5）/ web（C6/C7）/ tools（C1）/ docs（C8） | web-first 战略 vs 本次"功能级小迭代"的张力 |
| J4 | **版本与发布节奏**：是否 bump 版本 + 走 rick tools release 全流程 | 当前 5.0.11；RSI 流程约束（research 流程节） | 流程完备性 vs 轻量交付的取舍 |

---

## 批量追问（L3 第 1 轮——2026-09-24）

以下 4 问基于 research-L1 简报（12 条事实 + 11 候选），待 human 回答：

**Q1【候选选择】本次迭代改进哪些功能？**
- 调研背景：research-L1 产出 11 个功能级候选：C1 修 skills 校验空转（P0）/ C2 skill 生命周期机器化 / C3 `rick jobs` 只读命令 / C4 dream 定向+积压视图 / C5 learning --resume / C6 Dreams 页后端 / C7 learning 白名单 / C8 domain 文档同步 / C9 debug.md 回退清理 / C10 F6 修复 / C11 bugs 待实测验证轮
- 选项：A. 从 C1-C11 中挑选；B. 组合打包（如"校验诚实性+基础体验"）；C. human 有清单外自定需求
- 推荐：B——C1+C3+C4+C7+C8 打包（P0 校验修复 + CLI 基础体验 + dream 积压消化 + web 小补 + domain 诚实性），全部为小-极小工作量，风险低，一轮 job 可完成。但您的原需求措辞是"按照我的需求"——若您心中已有具体功能，请直接说明，以您为准。

**Q2【迭代规模】本轮投入多大？**
- 调研背景：候选工作量分布——极小×1（C7）、小×6、中×3（C2/C10 及 C4 视范围）
- 选项：A. 小迭代（1-2 个候选，单层 2-3 task）；B. 标准 job（3-5 个候选，多层多 task）
- 推荐：A 若您只有 1-2 个明确需求；B 若采纳 Q1 推荐（5 个候选约 4-6 个 task，2-3 层）。

**Q3【交付面】CLI 与 Web 的优先关系？**
- 调研背景：产品战略（2026-09-22 human 裁决）为 web-first；但候选中 C3/C4 是 CLI 命令、C6/C7 是 web 补齐；CLI 与 web 需保持同能力
- 选项：A. 严格 web-first（新能力先做 web 入口）；B. 按候选自然形态（CLI 候选走 CLI，web 候选走 web）
- 推荐：B——本次均为既有能力的补齐/修复（非全新能力），按候选自然交付面走；web-first 约束新能力的首发入口选择，不阻塞既有功能面的补齐。

**Q4【版本与流程】是否走完整 RSI 发布流程？**
- 调研背景：job_43 本身即 RSI 迭代（dev 树改动 + 层门禁 + release 提升）；当前版本 5.0.11
- 选项：A. 完整流程（rick-rsi-loop S0-S7，含 rick tools release --detach 提升，版本 bump 到 5.0.12 或 5.1.0）；B. 只合 main 不发布
- 推荐：A——既然走 RSI loop，S5 提升是制度闭环的一部分；版本号建议 5.0.12（功能级小迭代用 patch+1）。

---

## human 回答（L3 第 1 轮——2026-09-24）

- **Q1（候选）**：human 自定需求（不在 C1-C11 清单中），三个 web 痛点：
  1. **pi 会话持久化**：只要 server 存在、不主动断开，pi session 就应持久存在
  2. **左侧导航**：点击不同页面不会立即刷新，不能实时跳转
  3. **输入框**：不能换行、只有一行，输入体验差
- **Q2（规模）**：视情况而定（按调研后修复面决定）
- **Q3（交付面）**：修复 web 功能为主
- **Q4（流程）**：A——完整 rick-rsi-loop S0-S7 + release 提升

**根层 OKR 收敛（依 human 回答重写）**：

**O**：rick web 功能级修复迭代——pi 会话持久持有 + 左侧导航实时切换 + 输入框多行体验，走完整 RSI 流程交付 5.0.12。

**KR 集**：
- KR1：pi 会话生命周期改为持久持有——server 存续期间不主动断开 pi session（除非人工主动断开）
- KR2：左侧页面导航点击后立即实时切换/刷新
- KR3：web 会话输入框支持多行输入（换行），输入体验对齐终端
- KR4：走 rick-rsi-loop S0-S7 全流程（层门禁 + 人类审批 + `rick tools release --detach` 提升 5.0.12）

**充分性自检**：KR1∧KR2∧KR3 涵盖 human 列出的全部三个痛点 ⟹ 修复目标达成；KR4 保证交付合规 ⟹ O 成立。✅

**L4 事实回流**：三个痛点需追加技术调研（当前实现根因）→ research-L1-r2.md

---

## 事实回流（L4 · research-L1-r2，2026-09-24）

三痛点根因（文件:行号级证据，详见 research-L1-r2.md）：

| 痛点 | 根因 | 关键事实 |
|------|------|---------|
| 1 pi 会话持久化 | `DefaultIdleTimeout=30m` 空闲回收（supervisor.go:44/806-833）：挂机 30 分钟 → `w.Close()` + `markDead("idle reap")` → 会话 error 态需手动 Resume | SSE 断连**不**杀 worker（解耦）；正常情况下 worker 常驻；禁用回收有现成负值语义；`MaxActive=8` 上限 |
| 2 导航不实时 | 主根因：SessionPage 无 key + ChatView epoch 守卫不含 sessionId（ChatView.tsx:319）→ 切 A→B 时 B 的历史**永不拉取**，界面保留 A 旧消息 | 一行 `key={session.id}` 修复；次要：路由切换 scrollTop 不重置、工作区行点击=展开而非导航 |
| 3 输入框换行 | **换行已存在**（textarea + Shift+Enter + IME 保护）——真实缺口：缺 Ctrl+J（pi 终端有）、移动端无 Shift 无法换行、视觉单行、中文 IM 习惯未适配 | 需 human 澄清使用环境 |

三痛点文件集零重叠（Go 后端 / 前端路由层 / 前端 ChatInput），可并行 task。

**第 2 层（下钻预告）**：三个修复模块的 pipeline（改动面已定位，待 L3 第 2 轮裁决后展开叶子层）。

---

## human 回答（L3 第 2 轮——2026-09-24）

- **Q5**：A（禁用空闲回收）；**Q5b**：保留心跳杀；**Q5c**：MaxActive 可配置，**默认 64**
- **Q6a**：按推荐（主根因+次要因素全修）；**Q6b**：全做（A key 修复 + B 滚动重置 + C 工作区行点击跳转）
- **Q7**：输入框问题**全做**（A Ctrl+J + B Enter 语义切换 + C 移动端换行按钮）
- **Q8**：按标准 job 执行
- **新增痛点 4**：移动端使用 rick web 手机很烫、很卡——需分析前端页面原因，找端侧优化方法

**根层 OKR 最终收敛（含痛点 4）**：

**O**：rick web 功能级修复迭代——pi 会话持久持有 + 导航实时切换 + 输入框多行体验 + 移动端性能优化，走完整 RSI 流程交付。

**KR 集（最终）**：
- KR1：禁用 30m 空闲回收（server 存续即持久）；保留心跳杀；MaxActive 可配置默认 64
- KR2：SessionPage key 重挂 + 路由切换滚动重置 + 工作区行点击跳转
- KR3：Ctrl+J 换行 + 输入框视觉加高 + Enter 语义可切换（localStorage）+ 移动端换行按钮
- KR4：移动端发烫/卡顿——定位前端渲染根因并落地端侧优化（待 research-L1-r3）
- KR5：rick-rsi-loop S0-S7 全流程交付（层门禁 + 人类审批 + release 提升）

**充分性自检**：KR1-4 分别覆盖 human 全部四个痛点（KR5 保证交付合规）⟹ O 成立。✅

**L4 事实回流（第 2 轮）**：痛点 4 需专项调研（前端渲染性能分析）→ research-L1-r3.md

---

## human 回答（L3 第 3 轮——2026-09-24，grilling 最后一轮）

- **Q9（星空 P0-A）**：A1——移动端默认静态星空 + 设置项开关（桌面默认动画）
- **Q10（毛玻璃 P0-B）**：A——移动端断点去 blur 换不透明底（`md:` 恢复桌面）
- **Q11（P1 性能项）**：P1-A 流式节流 + P1-B dpr 降档，两个都做
- **Q12（虚拟化 P2-A）**：B——推迟下轮（本轮不含虚拟化）

**痛点 4 KR 收敛**：KR4 = P0-A（移动端静态星空+开关）+ P0-B（移动端去 blur）+ P1-A（流式节流）+ P1-B（dpr 1.5）；虚拟化留待下轮（P2-A 记入遗留）。

---

## 第 2 层：修复模块 pipeline（4 模块，文件集证据见 research-L1-r2/r3）

> **本层全部消解，无判断节点**：模块级事实（根因定位、修复选项、文件:行号、独立性）已由 research-L1-r2/r3 消解；模块间策略判断已由 human 在 L3 第 2/3 轮全部裁决（Q5-Q12）。

```
[M1 pi 会话持久化 (Go)]   [M2 导航实时性 (前端路由)]   [M3 输入框 (ChatInput)]   [M4 移动端性能 (perf)]
   internal/runtime/          web/src/routes/             web/src/components/       web/src/components/starfield/
   internal/cmd/web.go        SessionPage.tsx             chat/ChatInput.tsx        + App.tsx + ChatView.tsx
   internal/config            + App.tsx                                             + stores/events.ts
                              + WorkspaceListNode.tsx
```

**模块职责与调用关系**（相互独立，无跨模块调用；M2 与 M4 共享 App.tsx 不同段——plan 阶段需分层错开写域）：
- **M1（KR1）**：禁用空闲回收 + MaxActive 可配置默认 64 + 保留心跳杀
- **M2（KR2）**：SessionPage key 重挂 + 路由滚动重置 + 工作区行点击跳转
- **M3（KR3）**：Ctrl+J + 视觉加高 + Enter 语义切换（localStorage）+ 移动端换行按钮
- **M4（KR4）**：移动端静态星空+开关 + 去 blur + 流式节流 + dpr 1.5

**OKR 充分性自检**：M1-M4 分别落实 KR1-KR4，联合达成 ⟹ O（四痛点修复+RSI 交付）成立。✅

## 第 3 层（叶子层）：四维度落实

> **本层全部消解，无判断节点**：实现细节均为事实性问题，全部来自 research 简报的文件:行号证据与 human 已裁决的修复策略（Q5-Q12）；无遗留权衡点。

### M1 pi 会话持久化（Go）
- **关键代码**：`internal/cmd/web.go`（组合根）：`NewSupervisor` 调用处改传禁用回收的显式语义（`IdleTimeout: -1` 或引入 `DisableIdleReap bool`，取更清晰者）；`internal/runtime/supervisor.go`：负值语义已有（:310-311 `>0 才启动 reapLoop`），补注释澄清；`MaxActive` 从 config 读取（默认 64）
- **文件结构**：改 internal/cmd/web.go、internal/runtime/supervisor.go（注释/语义）、internal/config（新增 `web.max_active` 字段）；TestSupervisor_IdleReap 不受影响（测试显式传值）
- **工具调用**：`go build ./...` + `go test ./internal/runtime/ ./internal/cmd/`
- **环境依赖**：无新依赖；config.json 新字段向后兼容（缺省默认 64）

### M2 导航实时性（前端）
- **关键代码**：`SessionPage.tsx`：ChatView/MonitorView 加 `key={session.id}`（:59-62 一带）；`App.tsx`：location pathname useEffect 重置 `<main>` scrollTop（:310 一带）；`WorkspaceListNode.tsx`：行点击改导航（useNavigate 到 /ws/:wsId/sessions），箭头图标承担展开/折叠（:100-135）
- **文件结构**：改 3 个文件，无新建
- **工具调用**：`cd web && npm run typecheck && npm run build`
- **环境依赖**：无新依赖（react-router 已有 useNavigate）

### M3 输入框多行体验（前端）
- **关键代码**：`ChatInput.tsx`：keydown 增加 Ctrl+J → insertNewline（:176-180 一带，对齐 pi 终端键位）；min-h 提升到 ~2 行视觉；placeholder 提示「Shift+Enter / Ctrl+J 换行」；Enter 语义模式（`enterMode: "send" | "newline"`，localStorage 持久化，默认 send；newline 模式下 Enter=换行、Ctrl/Cmd+Enter=发送；空行不发送防误发）；移动端（<768px）显示显式换行按钮
- **文件结构**：改 ChatInput.tsx（SteerBar 复用同组件自动受益）；模式切换入口放 ChatInput 内（小图标/tooltip，不动 Settings 页——避免与 M4 设置项冲突）
- **工具调用**：`cd web && npm run typecheck && npm run build`
- **环境依赖**：无新依赖

### M4 移动端性能（前端）
- **关键代码**：`StarfieldBackground.tsx`：新增 `staticOnMobile` 判定（matchMedia 窄屏 或 独立设置项）复用 reduced-motion 静态路径；dpr 封顶 `min(devicePixelRatio, 1.5)`（窄屏时）；设置项开关（默认移动端静态/桌面动画）；`App.tsx`：侧栏 drawer（:250）+ 顶 header（:282）blur 加移动端断点替换（不透明底 + `md:backdrop-blur-*`）；`ChatView.tsx`：chat header（:685）+ 输入区（:801）同样处理；`stores/events.ts`：窄屏 flush 节流 rAF→100-120ms 定时（:166-182）
- **文件结构**：改 4 个文件；设置项落 localStorage（复用 M3 同一 localStorage hook 模式但独立 key，避免文件冲突——各自文件内定义）
- **工具调用**：`cd web && npm run typecheck && npm run build`；交付后真机复测（R7-3）
- **环境依赖**：无新依赖

### 终止判定（对照每层终止条件）
- **根层**：O 具体（四痛点+RSI 交付）✅；KR1-5 经 human 三轮裁决全部收敛 ✅
- **第 2 层**：4 模块命名清晰、职责互斥、pipeline 无调用耦合（App.tsx 共享已在 M2/M4 间显式错开策略）✅
- **叶子层**：四维度（代码/文件/工具/环境）全部落实 ✅
- **遗留记录**：P2-A 长会话虚拟化（下轮）、P2-B bundle 分割（低优）、真机 profiling 复测（交付后）

---

**Grilling 完成——设计树已遍历完毕，所有模块已落实到代码实现/文件结构/工具调用/环境配置。**

---

## 增量迭代 2（2026-09-25，human 验收反馈驱动）

**反馈**：①输入框三按钮太丑→只要一个发送，Enter=换行，随文字扩大 ②移动端发烫缓解但仍有→继续优化

**L1 调研**：research-I2.md（输入框现状 + 剩余发烫源排序 + 优化选项 O1-O7）

**新增 KR**：
- KR6：输入框极简——仅一个发送按钮；Enter 固定=换行；textarea scrollHeight 自适应扩大（长句折行也增长）；Ctrl/Cmd+Enter 发送保留；Ctrl+J 保留
- KR7：移动端二轮——P0 空态飞船常驻动画 + 工具运行期 Saucer 动画降级；P1 第 5 处 blur（展开条）+ FileReaderPanel blur + 状态点脉冲 + ToolElapsed 降频；P2 虚拟化（范围待裁决）

**判断节点（R7，待 L3 追问）**：
- J5：Enter=换行是删除 enterMode 设置（固定行为）还是保留设置默认 newline？
- J6：动画降级仅移动端还是全局？（O2 两案）
- J7：虚拟化本轮做否？（工作量中、交互风险最高单项）
