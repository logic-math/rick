# Research: pi coding agent 扩展(extension)故障真实案例

## Summary
pi(badlogic/pi-mono → earendil-works/pi)仓库中确有「扩展导致主进程崩溃」的真实案例(约 10–12 起,均有 issue 编号与日期,多数当天由 badlogic 关闭并官方修复);更大量的是「扩展加载失败→跳过/降级」类报告(约 15 条)。频率量级为数十条(占 8600+ issues 的 <0.5%),集中在 2026-03~05,其后官方以多个系统性 PR 修复。值得注意:#3556 显示 tool_call 事件的 handler 隔离缺口被官方关为 not_planned,第三方(pi-lens#1655)证实该路径至今会因扩展抛错而阻断执行。

## Findings
1. **#2431 扩展致启动崩溃(已修复)** — 2026-03-19,zats 报告:扩展调用 `pi.registerProvider()` 缺 `api` 参数,pi 虽报出扩展错误,仍在模型刷新阶段崩溃("instacrashes pi on startup");当日由 badlogic 关闭(completed)。严重度:主进程崩溃。[Source](https://github.com/earendil-works/pi/issues/2431)
2. **#3556 tool_call 隔离缺口(closed not_planned)** — 2026-04-22,samfoy 报告:`emitToolCall()` 是 runner.ts 中唯一未包 try/catch 的 emit 方法,扩展 `tool_call` handler 抛错即崩 agent 循环;2.5 小时后被 badlogic 关闭为 not_planned。第三方 pi-lens#1655(2026-08)证实该路径至今抛 "Extension failed, blocking execution"。严重度:agent 循环崩溃/阻断。[Source](https://github.com/earendil-works/pi/issues/3556) [pi-lens](https://github.com/apmantza/pi-lens/issues/1655)
3. **#3578/#3595 footer 扩展关机崩溃(官方修复)** — 2026-04-23,mackenney 与 vegarsti(MEMBER)分别报告:自定义 footer 扩展(ctx.ui.setFooter)在退出时访问 stale ctx → 关机/doRender 崩溃,终端残留 raw mode。官方修复:PR#3597(commit ff220fa)"tear down extension UI before runner invalidation on shutdown",当日合并。严重度:主进程崩溃。[Source](https://github.com/earendil-works/pi/issues/3595) [Source](https://github.com/earendil-works/pi/issues/3578) [PR](https://github.com/badlogic/pi-mono/pull/3597)
4. **#3606 stale 扩展上下文(官方修复)** — 2026-04-23:官方示例扩展 handoff.ts 在 `ctx.newSession()` 后报 "This extension instance is stale";commit f0cf8a5 "handle stale extension contexts" 修复。严重度:扩展报错/降级。[Source](https://github.com/badlogic/pi-mono/issues/3606) [commit](https://github.com/badlogic/pi-mono/commit/f0cf8a59d29e5d8da02cf608924523f0710576c9)
5. **#3510 扩展工具超大文本渲染崩溃** — 2026-04-21,Gabrielgvl:扩展工具返回超大文本块 → 渲染路径 `sanitizeBinaryOutput()` 抛 RangeError(Node 25.9.0),pi 崩溃;已关闭。严重度:主进程崩溃。[Source](https://github.com/badlogic/pi-mono/issues/3510)
6. **#4909 全部扩展工具崩溃 v0.75.4(官方修复)** — 2026-05-22,Jaraxxxx:所有扩展工具/命令触发 `message.content is not iterable` 崩溃(exit 1),连 no-op 工具也触发;根因是无类型 JS 扩展工具可注入 null content。官方修复:PR#6343(commit 8c0ccd1)在摄入边界归一化 null content。严重度:主进程崩溃。[Source](https://github.com/earendil-works/pi/issues/4909) [commit](https://github.com/earendil-works/pi/commit/8c0ccd14b34b6e5c403363518e331094b69ebf6c)
7. **#4333 pi-web-providers 扩展崩溃** — 2026-05-09,beyondhumanwork:该扩展下按 Ctrl+O(expand tools)抛 "Theme not initialized" 崩溃;已关闭。严重度:主进程崩溃。[Source](https://github.com/earendil-works/pi/issues/4333)
8. **#7187 第三方包 manifest 致启动崩溃(生产影响)** — 日期未取得:任一已安装包的 pi manifest 字段类型错(如 `"skills": "./skills"` 而非数组)→ 会话启动即 unhandled TypeError 崩溃;screenpipe(内嵌 pi)生产用户全部会话被杀。`pi -ne` 无效,因为崩溃发生在扩展运行前的包解析。修复状态未查证。严重度:主进程崩溃。[Source](https://github.com/earendil-works/pi/issues/7187)
9. **#7731 widget Proxy 无限递归崩溃** — 2026-08-06,iknowkungfubar:0.84.0 起扩展 widget 工厂收到的 Proxy 使包装 `tui.render` 无限递归(RangeError);当日关闭。严重度:主进程崩溃。[Source](https://github.com/earendil-works/pi/issues/7731)
10. **PR#4426 官方确认扩展是 uncaughtException 最常见触发者** — 约 2026-05 合并:官方 PR 自述 "The most common trigger is an extension's async ChildProcess exit callback that throws (e.g. from accessing a stale ctx after session replacement)";修复为进程级 `process.on("uncaughtException")` 兜底恢复终端,而非改扩展隔离。[Source](https://github.com/earendil-works/pi/pull/4426)
11. **#3084 afterToolCall 钩子抛错中止整批工具(官方修复)** — ≤2026-04-16(0.67.6 前后):并行工具调用收尾时钩子抛错会中止整批;commit b9cd557 改为转为错误 tool result。[Source](https://github.com/badlogic/pi-mono/commit/b9cd557d1d6773abc4e1ddc22c4758b9ecda2c0c)
12. **加载失败/跳过类(pi 主进程不受影响)** — #681(Bun 二进制发行版全部扩展 Failed to load);#1820/#1831(v0.56.0 全局 npm 安装下 Cannot find module @mariozechner/pi-ai/pi-tui);#4528(pi-subagents 缺依赖);#6455(扩展 npm 依赖解析失败/原生插件崩);#6222(2026-07-01,扩展突然加载失败且 `pi extension list` 亦不可用);#7985(2026-08-11,报错误导);#8092(pnpm 布局);#8237(SEA 单可执行);#1896(本地扩展被静默丢弃);#7154(0.82.x compaction 致扩展运行时进程内永久失效)。[681](https://github.com/badlogic/pi-mono/issues/681) [1831](https://github.com/earendil-works/pi/issues/1831) [6222](https://github.com/earendil-works/pi/issues/6222) [7985](https://github.com/earendil-works/pi/issues/7985) [7154](https://github.com/earendil-works/pi/issues/7154)
13. **官方机制佐证** — CHANGELOG 记录 "Fixed extension-related crash and startup-failure reporting to suggest restarting with `pi -ne`"(官方为扩展崩溃/启动失败提供 `pi -ne` 无扩展逃生模式);#8424 修"失败的扩展工厂残留订阅/注册"。[CHANGELOG](https://github.com/earendil-works/pi/blob/main/packages/coding-agent/CHANGELOG.md)
14. **第三方生态案例** — pi-messenger#25:扩展缓存 ctx 的 15 秒心跳定时器在会话重载后抛 uncaughtException 杀死 pi(第三方扩展仓库 issue)。[Source](https://github.com/nicobailon/pi-messenger/issues/25)
15. **频率量级(估计)** — 仓库 issue 编号至 #8600+(2026-08,#8606 创建于 08-25);多轮 web 检索定位扩展故障报告约 25–30 条(崩溃级约 10–12、加载失败/降级级约 15),占比 <0.5%,量级=数十条;集中 2026-03~05,其后系统性修复。github.com 在本环境 DNS 不可直连,无法取 GitHub issue 精确计数,属「频率可估计(量级)」而非精确值。

## Sources
- Kept: earendil-works/pi #2431, #3556, #3578, #3595, #3606, #3510, #4909, #4333, #7187, #7731, #681(badlogic/pi-mono), #1820/#1831, #4528, #6222, #6455, #7985, #8092, #8237, #1896, #7154 — 均为扩展故障一手 issue/报告(含编号+日期+现象)
- Kept: PR#3597/ff220fa、PR#4426、PR#6343/8c0ccd1、commit f0cf8a5、commit b9cd557 — 官方修复的一手证据
- Kept: pi CHANGELOG(main)— 官方对扩展崩溃/启动失败与 `pi -ne` 的记录
- Kept: apmantza/pi-lens#1655、nicobailon/pi-messenger#25 — 第三方对 tool_call 隔离缺口与扩展杀进程的佐证
- Dropped: #2716(DOMException AbortError 崩溃)— 未证实与扩展相关,属核心 abort 处理;#1706(footer 宽字符崩溃)— 内置 footer 非扩展;#6915(0.81.0 升级崩溃)— 核心 stream 函数错误,非扩展

## Gaps
- #3556 被关为 not_planned 的官方理由(评论原文)未能取得(仅见状态与关闭人 badlogic)。
- PR#4426 精确合并日期未能确认(依 issue 编号区间推断为 2026-05)。
- #7187 的当前修复状态、部分「已关闭」条目(#3510/#4333)的具体修复 commit 未能逐条核验。
- 频率为搜索抽样估计(约 25–30 条),非 GitHub 精确计数;如需精确值需可直连 GitHub 的环境执行 `is:issue extension crash/failed to load` 计数。

## 结论
1. 已记录约 10–12 起扩展致主进程崩溃 issue,多有官方修复。
2. 加载失败多为跳过降级;官方设 pi -ne 逃生并多次修 loader。
3. 扩展故障量级数十条(<0.5%),2026 上半年集中后回落。
