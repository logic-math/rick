# Research: pi vs dsh 生态/社区/活跃度事实（截至 2026-08-24）

## ① pi 活跃度
- 仓库 earendil-works/pi（原 badlogic/pi-mono，2025-08-09 建，已迁组织）：≈96.1k★、≈11.9k forks、133 open issues；贡献者 217–242（口径不一，15 天 +13）[1][2][3]；bus factor=1（badlogic 占 69% commits）[4]
- 最新版 0.84.2：GitHub release 与 npm 均为 2026-08-14 → 本地 0.84.2 即最新 [5][6]
- 发布频率高：0.84.0(8/6)→0.84.1(8/7)→0.84.2(8/14)[5]；@earendil-works scope 自 2026-05-07 迁移后 41 版本（≈2-3 版/周）[6]；跨 scope 累计 254 版 [7]
- npm 周下载：npmjs 页 2,173,217 [6]；第三方口径 1.4M–1.65M/周（峰值 2.2M、4 周均 1.5M，8/16 数据）[8][9]

## ② pi 社区生态
- 官方 Discord：discord.com/invite/3cU7Bz4UPx（README badge），设包分享频道 [10]
- npm keyword:pi-package 第三方包：官方 pi.dev/packages 目录 ≈4,300 个 [11]；非官方索引 getpipher/pi-package-index 每日刷新 [11]
- pi-web-access：维护者 nicobailon，多引擎搜索/抓取扩展，活跃 [12]
- pi-subagents：nicobailon/pi-subagents（异步子代理委派）[13]；@tintinweb/pi-subagents 948★/207 forks/55 open issues（2026-03-05 建）[14]；@ifi/pi-extension-subagents 基于其上 [15]

## ③ dsh 社区
- stars 轨迹：8/13 开源日 ~27.5k★ [16]；1 小时破 20k（史上最快）[17]；8/14 达 66,343 [18]；2 天 96.8k [19]；破 75k(8/14)/100k(8/15)/150k(8/17) [20]；8/17 达 141,532（作者 API 核实）[17]；8/20 ~171k [21]；现 189,495★/21,139 forks、open issues=0，近 7 天 +37.2k★ [20][22]
- 代码：7,090 commits / 34 committers（top：tianyicui 5268、LegGasai 1506、imccyu 1262）[23]
- 社区：GitHub Discussions + DeepSeek Discord（README 指引）[22]；官方文档 deepseek.com/harness/en/ [24]；第三方站 deepseekdocs.com / dshdocs.com / dsharness.org
- 版本：GitHub 最新 release v0.1.1-rc.1（8/21）[25]；npm latest 标签仍为 0.1.0-rc.7，rc.8 走 next 标签 [26]

## ④ 许可证
- pi：MIT [1][6]；dsh：MIT [22][24]；dsh 将 Cordis 框架源码 vendor 入仓、以 @deepseek-ai scope 再发布，均 MIT、保留上游 LICENSE [27]

## ⑤ limitations 与公开争议
- pi 极简哲学来源博文：mariozechner.at/posts/2025-11-30-pi-coding-agent/；核心=仅 4 工具（read/write/edit/bash），无 plan mode、无子代理、无 MCP、无 permission popups [28][29]
- pi 其他限制：新贡献者 issue/PR 默认自动关闭、维护者每日复查 [30]；TypeScript-only
- dsh：developer preview，README 大写警告 "THERE WILL BE COMPATIBILITY-BREAKING CHANGES" 置于安装步骤前 [22][31]；rc.8 破坏性变更记于 Chores [26]；首装下载重、耗时数分钟 [32]；官方文档薄 [33]
- dsh star 增速异常讨论：0xran 评论「star 通胀快过津巴布韦币」（Pi 一年≈90k vs dsh 两天 96.8k）[19]；VentureBeat 称数字只是快照、非采用指标 [16]；背景：Cyabra 曾发现 3,388 假账号推广 DeepSeek（2025-02，早于 dsh，非直接指控 dsh star 造假）[34]

## Sources
- [1] https://github.com/earendil-works/pi
- [2] https://repositoryradar.dev/repo/earendil-works/pi
- [3] https://trendshift.io/repositories/15471
- [4] https://inspect.software/software/earendil-works/pi
- [5] https://pi.dev/news/releases
- [6] https://www.npmjs.com/package/@earendil-works/pi-coding-agent
- [7] https://mise-tools.jdx.dev/tools/pi
- [8] https://amplifying.ai/coding-agents/pi
- [9] https://depscope.dev/pkg/npm/@earendil-works/pi-coding-agent
- [10] https://github.com/badlogic/pi-mono/blob/156a9052/README.md
- [11] https://github.com/getpipher/pi-package-index/blob/main/README.md
- [12] https://github.com/nicobailon/pi-web-access
- [13] https://github.com/nicobailon/pi-subagents
- [14] https://github.com/tintinweb/pi-subagents
- [15] https://registry.npmjs.org/@ifi/pi-extension-subagents
- [16] https://venturebeat.com/technology/deepseek-harness-launches-as-open-source-rival-to-claude-code-alongside-v4-pro-on-api-with-higher-prices
- [17] https://pasqualepillitteri.it/en/news/11573/deepseek-harness-fastest-github-stars-record
- [18] https://www.open-harness.net/
- [19] https://0xran.com/en/blog/dsh-vs-pi/
- [20] https://gittrend.io/repo/deepseek-ai/deepseek-harness
- [21] https://findarepo.com/repo/deepseek-ai/deepseek-harness/
- [22] https://github.com/deepseek-ai/deepseek-harness
- [23] https://summary.ecosyste.ms/projects/378897
- [24] https://deepseek.com/harness/en/
- [25] https://dsharness.org/changelog
- [26] https://dshdocs.com/guides/should-you-upgrade-to-dsh-rc-8/
- [27] https://github.com/deepseek-ai/deepseek-harness/blob/master/THIRD_PARTY_NOTICES.md
- [28] https://mariozechner.at/posts/2025-11-30-pi-coding-agent/
- [29] https://alexander.holbreich.org/posts/2026/pi-coding-agent/
- [30] https://gittrend.io/repo/earendil-works/pi
- [31] https://dev.to/renolu/deepseek-harness-puts-its-breaking-changes-warning-before-the-install-steps-1gai
- [32] https://flowtivity.ai/blog/deepseek-harness-open-source-agent-explained/
- [33] https://dshdocs.com/
- [34] https://cyabra.com/blog/deepseek-hype-fueled-by-fake-profiles/

## Gaps
- pi Discord 成员数、dsh Discord 成员数未获公开数字
- npm 周下载不同口径差异较大（1.4M–2.2M），建议以 npm registry API 实时核实
- dsh 7,090 commits 含 vendored Cordis 历史，净新增 commit 数无法拆分
