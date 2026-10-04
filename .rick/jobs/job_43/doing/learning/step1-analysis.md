# Learning Step 1：job_43 执行记录分析

> 分析时间：2026-10-04 | 数据源：debug/（2 条）、tasks.json（13 task）、plan/task1-13.md、rsi/{dev-iterations,gates,release,resume}.md、grilling/（design-tree + 11 份 research 简报）、requirement.md
> 说明：本 job 为 web 承载的 easy 会话，无 runtime-trace.md（act-path 不生成）——以 rsi/dev-iterations.md + gates.md 为轨迹替代源。

## 一、debug/ 分析（根因与解决过程）

### bug1：ResumeEasyIn 无条件覆盖非 easy 作业的 tasks.json（job 数据破坏）

- **发现**：2026-09-24 19:40，迭代 1 的 gate1「生产零触碰」断言（`git status --porcelain -uno`）捕获——生产树 job_36/doing/tasks.json（29 task 完整成功史，tracked）被 easy_session stub（449B）覆盖。
- **根因**：`internal/handler/easy.go:144` `ResumeEasyIn` 在 resume 会话结束后无条件调用 `writeEasyTasksJSON(doingDir)` 重写 tasks.json。用户 `rick easy --resume <非 easy 作业 id>`（job_36 是 plan/doing 型 29-task 作业）时真实任务史被 stub 覆盖。触发时间线：2026-09-24 02:31:54 一次 resume job_36、pi 会话秒退、stub 落盘。
- **处置**：应急 `git checkout --` 恢复（RSI loop「main 上脏 job 数据先修干净再 release」），29 task 完好。**代码未修复**——超 job_43 范围，记遗留下轮（修复方向：writeEasyTasksJSON 条件写入，仅当文件不存在或本身是 easy stub 时写入）。
- **深层关联**：job_36 曾因 F6（level_complete 写回时序）手工补记过 tasks.json——同族问题「job 数据被非合并路径改写」。gate 的生产零触碰断言是有效捕获机制（判别力实证）。

### bug2：MessageList 虚拟化自定义 Scroller 丢弃 Virtuoso props → 会话消息区全空白（P0）

- **发现**：2026-09-29，迭代 2 的 release（3e35ec8）上线后用户立刻报告「会话消息没有了」。
- **根因**：task7（eeda69d）引入。MessageList 换 react-virtuoso 时自定义 Scroller 组件只解构 `{ style, children }`，丢弃了 Virtuoso 通过 props 传入的其余 props——尤其 `data-virtuoso-scroller` 标记属性。Virtuoso 测量回调沿父链找标记节点（`for(;g.dataset.virtuosoScroller===void 0;)g=g.parentElement`）找不到 → 遍历到 null → `TypeError: Cannot read properties of null (reading 'dataset')` → 尺寸状态永不就绪 → 所有会话消息区渲染空白。数据完好，纯渲染层故障。
- **修复**：其他 agent 热修复（0d26c68，2026-09-29 12:46）：Scroller 改 `{ style, children, ...rest }` 全量转发（`<div {...rest}>`）。当轮重新 release（0d26c68-260929124639），用户验证恢复。
- **教训（防再犯）**：①虚拟化库的自定义 components 必须全量转发 props（关键标记/回调藏在 props 里，挑着解构=静默破坏内部协议）②gate4 静态断言测不到「props 转发缺失」这类运行时协议破坏——**渲染层重写类改动必须先在 dev UI 真渲染验收再 release**（本次只验证了 bundle 构建）③应急响应路径：用户报告→定位（刚上线的虚拟化为首要嫌疑）→热修复优于回滚（回滚点在 8dd1307）→当轮 release。
- **后果链**：此后每次 release 前都保留 dev UI（8414）真渲染验收通道——迭代 3/4/5 均执行，迭代 3/4 的小修复均由 human 在 dev UI 验收后批准。

## 二、执行轨迹还原（6 轮增量迭代 × 13 task × 10 gate × 6 release）

### 总体形态

单次 easy 会话（job_43，web 承载）内完成 **6 轮 human 反馈驱动的增量迭代**，每轮走完整 RSI 闭环：需求澄清（grilling/research）→ task 分解（plan/写域）→ 并行 worker 实现 → 层门禁（gate）→ level_complete 提交 → S3 演练 → S4 human 批准 → S5 release → S6 挂起恢复 → S7 rsi_check。13/13 task success（全部含 commit_hash），10 条 gate 记录全绿，6 次 human 批准留痕，6 次生产 release（版本链：8dd1307→0d26c68→69dc824→4b25f12→8dfa897→3d37918）。

### 迭代 1（2026-09-24，grilling 驱动 · 4 task · gate1/2 · release 8dd1307）

- **需求**：grilling 三轮追问（Q1-Q12）收敛四痛点：pi 会话持久化（30m 空闲回收杀 worker）、导航不实时（SessionPage 无 key + epoch 守卫致切会话显示旧内容）、输入框多行（Shift+Enter 已存在，真实缺口 Ctrl+J/移动端/IM 习惯）、移动端发烫（canvas 30fps 常驻 + blur 每帧重合成）。3 份 research（L1 功能面+11 候选、r2 三痛点根因、r3 移动端性能根因）支撑。
- **task 分解**：L1 并行 task1（Go：IdleDisabled 哨兵禁用回收 + web_max_active 默认 64 + 保留心跳 + VERSION 5.0.12 + 3 组新测试）∥ task2（ChatInput：Ctrl+J/enterMode/移动端按钮/空行守卫）∥ task3（导航：key={session.id}/AppShell 滚动重置/工作区行点击跳转）；L2 task4（移动端性能：starfieldMode 三档/窄屏 dpr 1.5/4 处 blur md: 化/流式 110ms 节流）。
- **执行**：3 worker 并行（glm-5.3）+ 1 worker 串行。gate1 首跑 2 红（断言写死「负值字面量」而 worker 用更优的 IdleDisabled 哨兵——按 KR 语义修正 gate pattern，未削弱断言）复跑绿；gate2 首跑绿。commits f76d0e98/8dd1307a。
- **release**：8dd1307-260925020038。第一次尝试 build 阶段 permission denied（bin/releases 属主 sankuai）→ RICK_RELEASES_DIR 重试成功。S5 重启挂起本会话 → human UI 恢复 → S7 rsi_check 6/6。
- **额外**：环境准备阶段发现生产树 job_36 被 bug1 污染（见上）；RICK_DEV_HOME 重定向（共享 dev home 属主 sankuai 不可写）。

### 迭代 2（2026-09-25，验收反馈驱动 · 3 task · gate3/4 · release 3e35ec8→P0→0d26c68）

- **需求**：human 验收反馈「输入框三按钮丑→单发送+Enter 换行+自适应扩大；发烫缓解但仍有」。
- **task 分解**：L1 并行 task5（ChatInput 极简）∥ task6（性能二轮：空态飞船全局静态/窄屏 SMIL+CSS 动画降级/第 5 处 blur 补漏/ToolElapsed 窄屏 5s）；L2 task7（react-virtuoso 虚拟化：双通道贴底/前置防跳/stable key）。
- **执行**：gate3（12 断言）+ gate4（7 断言）首跑全绿，commits 270ee66c/eeda69d1。
- **P0 事故**：release 3e35ec8 上线即全站消息空白（bug2，task7 引入）。其他 agent 热修复 0d26c68 重新 release，用户验证恢复。**本 job 最大事故，催生「dev UI 真渲染验收」纪律**。
- **环境事故二连**：机器休眠期 ACL 权限重置——job_43/doing/{,rsi,prompts} mask→r-x 且属主变更 sankuai → 修复：jobs/ 父目录可写 → job_43 整体 rename + git checkout 重建（38 文件全 tracked 零丢失）。

### 迭代 3（2026-09-29，验收反馈 · 1 task · gate5 · release 69dc824）

- **需求**：「最后一条消息贴着输入框，难受」——task7 虚拟化丢失旧实现 py-3 底部间距。
- **task8**：Virtuoso Footer spacer 12px（计入 footerHeight 贴底计算）+ 容器 py-3→pt-3。gate5 绿，commit f9c83628。
- **首次执行 dev UI 真渲染验收**（bug2 教训落地）：human 在 8414 验收间距后批准。dev HOME 二次权限重置 → 同法重建。

### 迭代 4（2026-09-29，验收反馈 · 1 task · gate6 · release 4b25f12）

- **需求**：「会话信息里点收起原文没反应」。
- **根因**（research-I5 链路考古 + 澄清）：loadPrompt 守卫 `prompt||promptLoading` 提前返回 + promptOpen 只有置 true 路径——**旧缺陷**（非虚拟化回归），首次使用暴露。
- **task9**：togglePrompt 四态分派（加载/收起/再展开/重试）。gate6 绿，commit fb01dde5。dev UI 验收后批准。

### 迭代 5（2026-09-29，产品建议驱动 · 3 task · gate7/8 · release 8dfa897）

- **需求**：human 反馈 MOGIC_HOME 路径不可点击阅读（裸路径不链接化 + 白名单外），并主动给出产品方案：「右上角工具箱 + 路径输入窗口 → 打开 md 阅读」。
- **task 分解**：L1 并行 task10（后端白名单扩展：全注册 workspaces + ~/.rick + ExtraFileRoots）∥ task11（前端：FileReaderProvider/Panel 提升到 App 全局挂载 + 顶栏工具箱 + 路径输入 Dialog）；验收后追加 task12（图标语义化：扳手→OpenPathIcon 文档轮廓+斜杠路径线）。
- **执行**：gate7 首跑红——4 个旧 web 测试读真实 ~/.rick/web.pid 撞生产单例，根因 **RICK_STATE_DIR env 穿透 t.Setenv(HOME) 隔离**（pi 会话环境带该变量）→ gate 加 env 净化（剥离 RICK_*）复跑绿。gate8（图标）绿。commits b946d992/95942726。
- **release**：8dfa897-260929215314（human 在 dev UI 验收图标「ok 了」后批准）。

### 迭代 6（2026-10-01~04，裁决驱动 · 1 task · gate9 · release 3d37918）

- **需求**：human 质疑「后端白名单是什么？不要搞特殊逻辑」→ **推翻迭代 5 task10 的白名单设计**；另提出 PWA 跨平台安装需求。
- **task13**：去白名单——移除 allowedFileRoots 根判定与 ExtraFileRoots 注入，直接读任意路径（保留 token 认证/仅文件非目录/大小上限）。gate9 绿，commit 56e683d9。
- **PWA 调研**（research-I6）：Chrome 正式安装硬要求 HTTPS；iOS 例外（http 也可 standalone A2HS）；方案 A（apple-touch-icon 1 行）+ B（--tls-cert/key 自签 HTTPS）呈报后 human 裁决「暂时放弃 PWA，暂时这么用着」。
- **release**：3d37918-261004190207。**生产接管**：休眠期外部启动的 ~/.rick/bin/rick（build_id=dev，PID 6440）被本次 release 停止并接管为 release 通道二进制。S7 rsi_check 6/6。

## 三、未解决/遗留问题清单

1. **bug1 代码未修复**（ResumeEasyIn 覆盖非 easy 作业 tasks.json）——已应急恢复但代码路径仍在，遗留下轮：writeEasyTasksJSON 改条件写入。
2. **PWA 暂缓**（human 裁决「暂时这么用着吧」）——快路径 A（apple-touch-icon 1 行，iOS 图标修复）随时可加；方案 B（自签 HTTPS + mkcert + --tls-cert/key）已调研留档。
3. **job_43.locked/ 备份残留**（15 个 ACL 锁死文件）+ **rick-dev-home-hadoop.locked/**（dev home 备份）——权限重置应对的产物，待环境侧权限恢复后删除，无害。
4. **生产休眠期被外部变更**（~/.rick/bin/rick dev 实例曾运行 10-01~10-04，来源未知）——已在迭代 6 release 接管对齐，但该模式提示机器休眠期存在外部操作面，值得警惕。
5. **早期遗留（迭代 1 记录）**：P2-A 长会话虚拟化已在迭代 2 完成（task7）；P2-B bundle vendor 分割（1.7MB 单 chunk，仅首载成本）未做；移动端真机 profiling 复测未做（发烫改善靠代码层推导 + human 主观确认「缓解」）。
6. **测试环境净化**：RICK_STATE_DIR 穿透测试 HOME 隔离的问题在 gate 层修复（剥离 RICK_* env），但 internal/cmd 的 4 个 web 单例测试本身仍依赖环境（未加 RICK_STATE_DIR 净化的测试运行方式会复现失败）——可考虑后续在测试内加 t.Setenv("RICK_STATE_DIR","") 固化。
7. **iterate 5 白名单设计的推翻成本**：task10（后端白名单扩展）与 task13（去白名单）同属一次会话内的设计往返——教训：涉及「安全边界/特殊逻辑」的产品决策应在 grilling 阶段问透（「要不要白名单」本质是产品裁决而非实现细节），避免一轮内建了又拆。

## 四、轨迹数据摘要（供 Step 2 评估）

| 迭代 | task | gate | commit | release 版本 | 备注 |
|------|------|------|--------|-------------|------|
| 1 | task1-4 | gate1(2红修正)/gate2 | f76d0e98/8dd1307a | 8dd1307 | gate 捕获 bug1；release 首试权限失败 |
| 2 | task5-7 | gate3/gate4 首跑绿 | 270ee66c/eeda69d1 | 3e35ec8→**P0**→0d26c68 | P0 事故+权限重置 |
| 3 | task8 | gate5 | f9c83628 | 69dc824 | dev UI 真渲染验收首例 |
| 4 | task9 | gate6 | fb01dde5 | 4b25f12 | 旧缺陷修复 |
| 5 | task10-12 | gate7(env净化修正)/gate8 | b946d992/95942726 | 8dfa897 | 工具箱+图标 |
| 6 | task13 | gate9 | 56e683d9 | 3d37918 | 白名单推翻+生产接管 |

- 报错/修正事件：gate1 pattern 修正、release 权限失败重试、P0 热修复、ACL 权限重置×2、gate7 env 净化、release.md 头部三行两次误删重写、job_43.locked 误提交后移出。
- 6 次 S6 挂起-恢复（本承载会话每次 release 后由 human 点恢复）。
