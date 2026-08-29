# research-extra-4 简报 — dsh「插件错误→系统性崩溃」实测数据调研

> 阶段：loop_7 E 临时调研（human 明确请求）| 基准 2026-08-26 | 方法：本地源码/postmortem 自查 + 4 叶子联网（文末）| 前序：research-extra-3.md（机制级证据，本次补实测数据）

## ① dsh 插件故障案例统计

**【案例已记录·web 检索下限】**dsh（github.com/deepseek-ai/deepseek-harness，2026-08-13 开源，~19 万 star，反馈全走 Discussions）开源后 13 天观察窗内，插件故障讨论 **≥35 条**：**无法启动 21**（#535/#1404/#2710 等，典型报错即 `plugin tree failed to load`）、**部分功能损失 11**（#1486/#1515 工具调用全崩、#2851 Windows PTY 损坏）、**数据损坏 3**（#802/#1538 插件写入的会话日志不可再加载、#2034 会话永久不可恢复）。[leaf-1，编号清单在案]

**官方响应**：维护者逐帖回应缺失——#1415 标 Unanswered，#535 至今 13 天无回应，响应时距【数据不足】（0 样本）。修复靠发版+社区自救：rc.7（08-17）修 pty.node 加载失败；社区工具 dsh-doctor（两版）/dsh-loader 兼容层/锁版本/手改 cordis.patch.yml。changelog：rc.7/rc.8 唯一标注 breaking=SQLite 格式（非插件）；实际插件级 breaking ≥2 条未标注（键控插槽致 dsh-codex-connect 崩、node-pty 升级致 #2851）。

**postmortem（本地全量 4 份）**：0001=插件加载期崩溃（第一方 ACP；feature 全失、无数据丢失；PR#41 修）；0002=插件静默失效（!!js 机制）；0003/0004 非插件——4 份均内部开发期缺陷，0 份涉及第三方插件致用户侧故障。[docs/postmortem/]

## ② pi 扩展故障对照

**【案例已记录】**pi（earendil-works/pi，原 badlogic/pi-mono；issue 已至 #8600+）扩展故障约 25–30 条（<0.5%），其中**崩溃级 10–12 条**（扩展搞挂主进程：#2431/#3578/#3595/#4909/#7187/#7731 等），多数有官方快速修复（PR#3597/#6343/#4426，常当日~数日）；加载失败类多为跳过/降级（pi 继续），官方设 `pi -ne` 无扩展逃生口。**关键例外**：#3556（04-22）emitToolCall 为 runner 唯一未包 try/catch 的 emit——扩展 tool_call handler 抛错即崩 agent 循环，关为 not_planned，第三方（pi-lens#1655）证实至今仍抛 "Extension failed, blocking execution"。频率：2026-03~05 集中、官方系统修复后回落【频率可估计（量级）】。[leaf-4]

## ③ 隔离修复 PoC 进展

#4175（08-23，franksong2702）PoC **已落地为 fork 分支** recoverable-bundle-failures-poc（commit c601d5a861，29 文件，+802/−102）：新增 `recoverableBundles` 子集 + 每 bundle 层独立事务（必需层仍 fail-loud），附设计文档，文末 3 问待维护者答——**无官方回应、未转 PR、未合入**。同类提案 #1228（坏 bundle 拖垮整 profile，零回复）、#2920（影子 profile + --recover-optional-plugins）、#1496（倾向安装期防护，对 skip-and-warn 谨慎）均未合并。官方 rc.8→0.1.1-rc.2 无隔离改动；dshdocs 排障页明言「no skip-and-warn」；本地源码复确认 boot 无 skip/降级路径（唯一合法 fiber-less 态=显式 entry.disabled），PoC 需改 29 文件反证上游无此机制。[leaf-3；packages/boot/app-boot/src/index.ts]

## ④ 数据充分性结论

1. **机制已证实**（源码级，r3+本次复确认）：boot 全树事务，无单插件故障隔离。
2. **案例已记录**：「插件错误→系统性崩溃」**不再是纯推测**——13 天窗内 21 条无法启动真实用户案例 + 3 条数据损坏，严重度谱系完整（含「卸载入口与坏插件同殁」模式 #4175）。
3. **频率可估计（仅下限）**：无法启动类 ≈1.6 条/天；真实发生率【数据不足】——分母（装机量/讨论全量）未知，检索为 web 下限非穷尽。
4. **对照**：同构故障在 VS Code 真实存在（2019 社区实测 800+ 崩溃报告 #79782；官方确认单扩展可拖垮共享扩展宿主 #100413）但靠独立进程+自动重启兜底，主编辑器存活；pi 有崩溃案例但多数获官方修复+降级兜底；Koishi（cordis 同源）有降级机制（错误记 warning/插件挂起）但 dispose 抛错仍可崩（#1254）；Eclipse/OSGi 个案跨 20 年无聚合率。**所有对照系统均无公开插件故障率百分比**——dsh 的数据缺口是行业普遍现状，唯其观察窗极短（13 天）且无兜底机制放大风险敞口。[leaf-2]
5. **修正**：dshdocs.com 自述「独立社区维护手册」非官方站——前轮引作官方 changelog 的表述需修正。[leaf-1]

## R7 上报项

1. 35 条讨论的逐帖 open/closed 状态与回复时间戳未全量核实（GitHub 直连 DNS 不可达，计数为 web 检索下限）。
2. dsh 装机量/用户基数无公开数据 → 真实发生率分母缺失。
3. pi #3556（tool_call 路径隔离缺口）在当前 0.84.x 是否仍存在未复核（第三方证据为 07 月；与本地 0.84.2 源码「emit 有 try/catch」存在张力，待核）。
4. #4175 PoC 的官方表态需后续跟踪（当前无回应）。

## 叶子文件

- leaf-1（dsh 案例+changelog 统计）：briefs/research-extra-4-leaf-1.md
- leaf-2（代理证据：VS Code/Koishi/Eclipse/Chromium）：briefs/research-extra-4-leaf-2.md
- leaf-3（隔离 PoC 进展）：briefs/research-extra-4-leaf-3.md
- leaf-4（pi 扩展故障对照）：briefs/research-extra-4-leaf-4.md
- leaf-4 完整底稿（叶子 8K 版）：briefs/research-extra-4-leaf-4-full.md
