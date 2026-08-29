# Research: 插件架构系统的插件故障实测数据(dsh 插件故障率代理证据)

检索日 2026-08-26。环境限制:github.com / api.github.com / bugs.eclipse.org 均无法直连(DNS ENOTFOUND),GitHub/Bugzilla 实时搜索计数不可得;以下计数均引可验证公开原文并标日期,严禁编造已遵守。

## ① VS Code Extension Host(部分实测数据存在)
- 计数(社区实测,非官方遥测):microsoft/vscode#79782(2019-09)原文:当时 "Extension host terminated unexpectedly" 在 VS Code 仓已有 **800+ 报告**,GitHub 全域 vscode 相关仓 **850+**。https://github.com/microsoft/vscode/issues/79782
- 官方确认机制:成员 alexdima 在 #100413(2020-06,被标 *caused-by-extension):"所有扩展共享一个扩展宿主进程,单个扩展的致命 bug 可拖垮整个宿主进程"。https://github.com/microsoft/vscode/issues/100413
- 独立进程隔离后主编辑器存活:宿主崩溃后自动重启循环、主窗口仍可用——#318279(2026-05,30 秒周期被杀后自动重启)、#327512(2026-07)。官方文档 https://code.visualstudio.com/api/advanced-topics/extension-host
- 官方崩溃遥测:未公开(数据不足)。当前实时 issue 计数:不可验证(数据不足)。

## ② Eclipse/OSGi(仅定性证据;量级数据不足)
个案跨 20 年:bugs.eclipse.org #52119(2004,bundle 未解析致启动失败)、#430458(2014,插件导出包冲突→依赖插件全不启动)、#465593(2015,安装破坏已解析 bundle,-clean 恢复)、#553773(2019)、#579338(2022,升级后无法启动);StackOverflow 71654299(2022)、79286700(2024)。https://bugs.eclipse.org/bugs/show_bug.cgi?id=430458 等。聚合故障率:无公开数据。

## ③ Koishi/cordis(机制文档+个案;计数数据不足)
- 官方文档:插件可逆、热重载,称 3000+ 插件规模下仍能妥善处理加载/卸载/更新(koishi.chat/zh-CN/cookbook/design/disposable.html);运行时错误仅记 warning、创建/连接阶段错误记 error,不中止全局(api/utils/logger.html);using 声明依赖缺失时插件挂起不加载、服务恢复后自动重载(guide/aspect/service.html)。→ 有降级机制,无进程级隔离(单进程 Node)。
- 真实故障:koishijs/koishi#713(2022-06):"cannot resolve plugin" 仅 [E] 日志,其余插件继续加载;#1254(2023-11-01 报,11-13 修复):dispose 回调抛错可致 Koishi 崩溃(隔离不完整);koishijs/webui#328(2024-05,单插件 EMFILE 启动失败)。

## ④ Chromium(机制文档存在;崩溃率数据不足)
- 官方 stability 文档:扩展运行于独立 extension renderer 进程;崩溃时弹窗提示用户重载,策略安装/组件扩展自动重启,浏览器进程不受影响。https://chromium.googlesource.com/chromium/src/+/main/docs/stability.md
- 定量片段:UMA 直方图 Renderer.ProcessLifetime3 显示约 **40% renderer 进程为扩展宿主**(commit 6b5b92a)。扩展崩溃率百分比:未见公开(数据不足)。

## 数据分级
- 实测数据存在:①2019 年社区计数(800+/850+,二手引文);④"40% renderer 为扩展宿主"。
- 仅定性:①单扩展拖垮整宿主(官方确认);②全部个案;③机制与个案。
- 数据不足:任何系统的"插件故障率"百分比;②③④聚合报告量级;①当前实时 issue 计数。

## 结论
1. VS Code 官方确认单扩展可拖垮整宿主;2019 社区实测 800+ 崩溃报告。
2. Eclipse/Koishi/Chromium 均有真实插件故障个案,均无公开故障率数据。
3. 对照系统靠隔离+重启/降级存活;dsh 全树事务最严苛,自身无实测频率。