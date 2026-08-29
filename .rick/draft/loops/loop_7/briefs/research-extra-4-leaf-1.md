# dsh(deepseek-harness)插件故障案例调研(检索日 2026-08-26)

## Summary
仓库属实:github.com/deepseek-ai/deepseek-harness,2026-08-13 公开(MIT,≈19 万 star),Issues 关闭、反馈全走 Discussions。经 web 检索确认 35 条插件加载失败/崩溃类讨论(下限,非穷尽);官方逐帖回应缺失,修复主要靠发版与社区工具。注意:dshdocs.com 自述为"独立社区维护手册",非官方站。
约定:下文 #N = github.com/deepseek-ai/deepseek-harness/discussions/N;未标日期=检索片段未含时间戳(非不存在)。

## ① 数量/编号/日期【案例已记录·非穷尽】
- **无法启动(整树/整 profile 拒启)21 条**:#535(08-13)、#328、#880、#913、#917、#1197(08-14)、#1228(08-14)、#1245、#1313、#1404(08-14)、#1473、#1479、#1483、#1724、#2710(08-17)、#2889(08-17)、#2917、#2963、#2999、#3060、#3473。典型报错即 "dsh: plugin tree failed to load: failed to apply loader entry include (cordis:include)"(#535/#1404/#2710 等)。
- **部分功能损失 11 条**:#903、#1258、#1337、#1415(08-14,官方标注 Unanswered)、#1470(08-14)、#1486(08-14)、#1515(08-14)、#1697、#2851(08-17)、#2854、#2905。
- **数据损坏/不可恢复 3 条**:#802、#1538(08-14)、#2034;#1473 跨类(单个损坏会话日志致整树拒启)。
- 汇总/提案帖(非独立案例):#1496(08-14,汇总 #1404/#1197/#1486/#1415/#1413)、#2078、#2130、#2920(08-17)、#3472、#4175(08-23)。
- 任务给定 4 帖核实:#4175=08-23,PoC 提案(不兼容插件阻 Web shell 挂载+插件管理 UI 同殁,属实,作者 franksong2702,提 3 问待维护者答);#3691=08-20,rc.6→rc.8 升级(npm OOM+键控插槽 breaking);#3235=08-19、#3304=08-19,均性能类(TTFT/缓存),非插件故障。

## ② 严重度分级
无法启动 21 例;部分功能损失 11 例(#1486/#1515 工具调用全崩、#2851 Windows PTY 损坏等);数据丢失/损坏 3 例(#802/#1538:插件写入的会话日志不可再加载;#2034:会话永久不可恢复)。【频率可估计】社区工具 dsh-doctor 自述 28 项检查映射 18 条社区报告;【数据不足】讨论区全量基数与真实发生率无法估计。

## ③ 修复路径与官方响应
- **官方回应**:检索范围内未见维护者逐帖回复;#1415 标 Unanswered;dshdocs 称损坏会话日志类"still no maintainer reply on any of the reports"、#2851 至 rc.8 分析时无官方回复。【数据不足】首次官方回应时距无样本可算;#535(08-13)至今 13 天无回应记录。
- **版本修复**(发行说明不引 issue 号,报告→修复映射未证实):rc.7(08-17)修 Linux pty.node 加载失败、INVALID_REPLAY_STATE,新增插件设置卡(官方 PR #2404 plugin-owned-settings-surface 已合并);v0.1.1-rc.1(08-21)修插件 Bubblewrap 沙箱逃逸(dsharness.org 述)。
- **用户绕过**:手改 cordis.patch.yml/manifest(#1197/#913)、仅 CLI 恢复(#1228)、Symbol.for 一行补丁(#2078)、dsh-doctor 预检(boyin111-1、moonquake2004 两版)、dsh-loader 兼容层、锁版本号。
- **修复 PR**:loader fail-fast/重复 entry id/悬空引用类(#1404/#1197/#4175/#2920)未检索到官方修复 PR;插件侧 franksong2702/dsh-codex-connect 仓库 issue #23(08-18 开、同日关,约 10.5 小时)由插件作者适配键控插槽修复。

## ④ dshdocs.com changelog 插件相关 breaking
rc.8(08-19)唯一标注 breaking=SQLite 存储格式不兼容 rc.7(非插件系统)→ 明确标注的插件系统 breaking 为 0/1 条(占比 0%);rc.7(08-17)无 breaking 标注,但存在两条实际插件级 breaking 未标注:①键控插槽 settings.plugin.item 需 options.key(致第三方插件 dsh-codex-connect 加载失败,其 issue #23);②node-pty 1.1.0→1.2.0-beta.15(致 #2851 Windows 持久 PTY 插件崩)。rc.6(08-13)及更早版本段未逐条核实→【数据不足】总占比不可严格计算。

## Gaps
GitHub 站内/API 直连在本环境 DNS 不可达,计数为 web 检索下限;#1413 症状未独立核实(检索显示其标题为功能请求,但被 #1496 列入汇总);多数讨论的回复时间戳未获取;安全类帖(#587/#454)未计入故障统计。

## 核心来源
- 仓库: github.com/deepseek-ai/deepseek-harness(创建 2026-08-13,MIT)
- #4175 / #1496 / #1197 / #3691 / #3235 / #3304: 同域 /discussions/ 编号
- dshdocs.com/guides/changelog/(社区站,rc.7/rc.8 段已核)
- dsh-codex-connect issues/23; dsharness.org/changelog

## 结论
1. 已记录插件故障讨论≥35 条,无法启动类 21 条占多数。
2. 官方逐帖回应缺失,修复靠发版+社区工具,时距不可算。
3. changelog 插件级 breaking 标注 0 条,实际≥2 条未标注。