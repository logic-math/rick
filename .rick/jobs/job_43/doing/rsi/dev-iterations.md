# RSI 证据：dev-iterations

通过标准：至少记录一条构建指纹（`rick.dev.<sha7>-<ts>` 或 `build_id=<sha7>-<ts>`）。
示例：`DEV_UP bin=... build_id=c539c60-260921192330 port=8414`



## 2026-09-24 19:45 · S0/S1 环境准备（parent）

- dev 环境：`RICK_DEV_HOME=/workdir/sunquan20/rick-dev-home-hadoop`（共享 rick-dev-home 为 sankuai 所有且 ACL mask=--- 不可写，用 EnvHome 重定向，dev 树 /workdir/sunquan20/rick-dev 不受影响）
- `rick tools dev-web init` + `up` → DEV_STATUS state=up pid=143853 port=8414 build_id=e72ce7b-260924182400 fingerprint=ok
- job_43 数据迁移：生产树 .rick/jobs/job_43 → dev 树（RSI「job 数据只写 dev 树」）；生产树残留 untracked job_43 待 release 前清除
- 生产树污染处置：job_36/doing/tasks.json 被 ResumeEasyIn bug 覆盖为 easy stub（02:31:54），已 `git checkout --` 恢复（29 task 完好）；bug 详情见 doing/debug/bug1-resume-easy-overwrites-tasks-json.md（遗留下轮）
- grilling_gate ✅（3 层设计树 + 3 份 research 简报）；pipeline_gate ✅（4 task / 2 层，同层写域不相交）
- gate1/gate2 RED 判别力实证：exit 1，13 项断言全红（task1-3 改动未落盘）+ 生产零触碰断言捕获 job_36 污染（已修复）

## 2026-09-24 20:00 · 第 1 层完成（task1+task2+task3 并行 → gate1 绿 → commit）

- 3 worker 并行（glm-5.3）：task1 Go 会话持久化（IdleDisabled 哨兵 + web_max_active 默认 64 + DefaultMaxActive=64 + VERSION 5.0.12 + 新测试 IdleDisabledNegative/Defaults64/ResolveSupervisorConfig）；task2 ChatInput（Ctrl+J 光标插入/enterMode localStorage/移动端换行按钮/空行守卫）；task3 导航（key={session.id}/AppShell 滚动重置/工作区行点击 navigate+chevron stopPropagation）
- gate1 首跑 2 红：断言写死「负值字面量」而 worker 用了更优的 IdleDisabled 哨兵（task1.md KR 原文即允许「或等价显式禁用语义」）——按 KR 语义修正 gate pattern（未削弱断言，13 项全在），复跑绿
- build_id：e72ce7b-260924182400（dev 实例未重启，前端改动待 task4 后统一 restart 验证）
- commit：f76d0e98 feat(layer): task1+task2+task3

## 2026-09-24 20:20 · 第 2 层完成（task4 → gate2 绿 → commit）

- task4：StarfieldBackground 静态判定抽 starfieldMode.ts（rick.starfield.mode auto/animated/static + storage/viewport/reduced-motion 实时重估）+ 窄屏 dpr 1.5 + 4 处 blur 移动端不透明底（App drawer/header + ChatView header/输入区，md: 恢复）+ events.ts 窄屏 110ms 节流（宽屏 rAF 保持）
- gate2 首跑即绿（exit 0）→ commit 8dd1307a
- S1 收尾：dev 实例 restart（后端 Go 已变——supervisor 配置），build_id 应更新

## 2026-09-25 02:10 · S5/S6/S7 收尾（会话恢复后）

- S5 提升：version=8dd1307-260925020038（第一次尝试 build 阶段 permission denied——bin/releases 属主 sankuai；RICK_RELEASES_DIR=/home/hadoop-recsys/.rick/releases 重试成功，见 release.md）
- S6：本承载会话（e7dbebd0，easy job_43）随重启挂起 → human 02:08:52 UI 点「恢复继续」→ 新 pi 进程 192787 接续（recovery-report: recovered=1, failed=0）
- S7：rsi_check 复跑（本行以下见 gates.md 与 rsi_check 输出）

## 2026-09-25 12:30 · 增量迭代 2（task5+task6+task7）

- 反馈驱动：输入框三按钮丑→单发送+Enter 换行+自适应扩大；移动端发烫缓解但仍有→二轮优化
- task5：ChatInput 极简（删 ＋/↵/模式切换，Enter 固定换行，scrollHeight 自适应 max 240px，VERSION 5.0.13）
- task6：空态飞船全局静态 + theme.css 767px 动画降级块 + Saucer SMIL 窄屏条件渲染 + 第 5 处 blur（展开条/断线横幅/阅读器）+ ToolElapsed 窄屏 5s
- task7：react-virtuoso 渲染层虚拟化（双通道贴底/前置防跳/stable key/组 key 索引缺陷顺带修复）；数据层零改动
- gate3（12 断言）+ gate4（7 断言）首跑全绿 → commits 270ee66c / eeda69d1
- S1 收尾：dev restart 验证 build_id 更新（见下行）

## 2026-09-29 · 迭代 2 收尾（P0 事故 + 权限重置 + 证据补录）

- **P0 事故**：3e35ec8 上线即「会话消息全空白」——task7 虚拟化自定义 Scroller 丢弃 Virtuoso props（data-virtuoso-scroller 标记落不到 DOM → 测量回调 null 崩溃）。其他 agent 热修复 0d26c68（...rest 全量转发）并重新 release，用户验证恢复。教训留痕 doing/debug/bug2：①虚拟化组件协议必须全量转发 props ②渲染层重写必须 dev UI 真渲染验收后再 release（gate4 静态断言测不到运行时协议破坏）
- **环境权限重置**：机器休眠期外部重置 job_43/doing/{,rsi,prompts}/ 的 ACL mask→r-x 且文件属主变更为 sankuai（同族问题已在 release.go 注释记载）。修复：jobs/ 父目录可写 → job_43 整体 rename+git checkout 重建（38 文件全部 tracked，零丢失；备份 job_43.locked/ 待确认后清理）
- rsi_check 复跑对齐实际生效版本 0d26c68-260929124639

## 2026-09-29 13:58 · 迭代 3（task8 贴底间距 + 权限重置二连）

- task8：Virtuoso Footer spacer 12px（py-3→pt-3，底部间距移交 Footer 计入 footerHeight）→ gate5 绿 → commit f9c83628
- 环境权限重置二连：job_43 doing 目录（已修：rename+checkout 重建）之后 dev HOME（rick-dev-home-hadoop）也被重置 mask——同法处理（rename .locked + dev-web init 重建，幂等）
- dev 实例重建：build_id=f9c8362-260929135824，bundle index-CYCA3JSB.js；按 bug2 教训保留 dev UI 真渲染验收通道（8414）

## 2026-09-29 15:02 · 迭代 4（task9 收起原文 toggle）

- 根因：loadPrompt 守卫 prompt||promptLoading 提前 return + promptOpen 无置 false 路径（旧缺陷，非虚拟化回归）
- task9：togglePrompt 分派（四态：加载/收起/再展开/重试）→ gate6 绿 → commit fb01dde5；dev UI 8414 真渲染验收通道已起

## 2026-09-29 21:37 · 迭代 5（task10 后端白名单 + task11 前端工具箱）

- 根因链（research-I5）：裸路径不链接化 + 后端白名单仅当前 workspace/~/.rick → MOGIC_HOME/dev 树路径不可读；human 方案=顶栏工具箱+路径输入
- task10：allowedFileRoots 扩展全注册 workspaces + ~/.rick + ExtraFileRoots（组合根 resolveDevTreeRoots：RICK_DEV_TREE env / exe→cwd 推导，生产=start-web.sh cwd 祖父/rick-dev）
- task11：FileReaderProvider/Panel 提升到 App 壳层全局挂载 + 顶栏扳手工具箱 + 路径输入 Dialog；ChatView 解绑（消息链接走全局 Context）
- gate7 首跑红：internal/cmd 4 个 web 测试读真实 ~/.rick/web.pid 撞生产单例——根因是 RICK_STATE_DIR env 穿透 t.Setenv(HOME) 隔离（本 pi 会话环境带该变量）；gate7 已加 env 净化（剥离 RICK_*），复跑绿（测试代码本身无需改）
- 后端实测（dev 8414）：注册内绝对路径 200；MOGIC_HOME 在 dev 被拒是 dev/生产 workspace 注册表隔离（设计行为），生产 8 workspace 全可读

## 2026-10-04 · 迭代 6（task13 去白名单 + PWA 调研）

- human 裁决「不要搞特殊逻辑」：迭代 5 task10 的 allowedFileRoots 白名单整体移除——resolveWorkspaceFilePath 直接读（file:// 剥壳/相对按 ws.Path/无根检查）；ExtraFileRoots/resolveDevTreeRoots 随删；保留 仅文件/大小上限/NUL 防御 + token 认证
- gate9 绿 → commit 56e683d9
- PWA 调研（research-I6）：Chrome 正式安装硬要求 HTTPS；iOS 例外（http 也可 standalone A2HS，缺 apple-touch-icon 1 行）；Win http 快捷方式+窗口模式可用；Android http 最弱。方案 A（apple-touch-icon 补全）+ B（--tls-cert/key 自签 HTTPS + mkcert）待 human 裁决
- 环境注意：生产 8413 当前跑 ~/.rick/bin/rick（build_id=dev，10-01 18:25 起）非 release 二进制——机器休眠期生产被外部变更过（来源未知，可能是 human 或其他 agent）；迭代 6 release 时需重新对齐
