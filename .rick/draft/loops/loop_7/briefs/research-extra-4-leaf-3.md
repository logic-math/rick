# Research: dsh 插件故障隔离提案进展(检索日 2026-08-26)

## ① Discussion #4175 现状
Ideas 区,franksong2702 2026-08-23 发起;抓取内容仅见发起帖。PoC:基于上游 commit b150a55(v0.1.1-rc.2 合并点)的 fork 分支 franksong2702/recoverable-bundle-failures-poc,commit c601d5a861(2026-08-23T08:28Z,+802/−102,29 文件,"prototype recoverable profile bundle startup"),附设计文档 .agents/notes/proposed/architecture/2026-08-23-recoverable-profile-bundle-startup.md(Status: proposed)。设计:profile 新增可选 `recoverableBundles` 子集(显式优先;旧 profile 缺省时依赖托管 bundle 默认可恢复、安装自带 bundle 必需;`dsh plugin add` 将新装 out-of-tree bundle 记为可恢复);每个 bundle patch 层作独立根 Include/Loader 事务——必需层失败保留 fail-loud 整树释放;可恢复层失败保留已提交树、记结构化故障、继续后续层;client graph 投射 `failurePolicy: "recoverable"`(缺省=必需),浏览器记录被标 entry 的 import/apply/pending 故障后仍挂载必需 uiRenderer;文末向维护者提 3 问。官方回应:未见 deepseek 团队回复;未见转 PR(检索无上游相关 PR;内部 PR 不可见部分未澄清);最新可见更新 2026-08-23;open/answered 状态未澄清。

## ② 其他相关提案(编号均在 github.com/deepseek-ai/deepseek-harness/discussions/ 下;均未见 merged/closed 证据,状态未澄清)
- #1228(fkysly,2026-08-14):坏 bundle 拖垮整 profile、插件管理 UI 同殁;建议按层降级或 `--safe-mode`;页面显示 Replies: 0 comments(零回复)。
- #2920(overact):可恢复启动模式(影子 profile、最大已知良好可选集、`dsh web --recover-optional-plugins`);作者 2026-08-17 补 client 侧就绪边界与 `onEntryError` hook 提案。
- #3473:cordis.patch.yml 一条坏行使整树拒启(ERR_MODULE_NOT_FOUND),无降级。
- #1197:patch 引用不可解析致 `dsh web` 拒启(0.1.0-rc.6)。
- #1496(2026-08-14):汇总 #1404/#1197/#1486/#1415/#1413 的安装防护 advisory;对 skip-and-warn 谨慎(怕掩盖冲突),倾向安装期防护+启动清晰报错。
- #3320(Fable-Forge,2026-08-19):插件树未就绪即对外 HTTP,引 #2920 定复现。相关:#1719(dsh doctor)、#587(安全)。
- 社区止损:ICCuse/dsh-web-safe(per-row isolation 设计草稿,附 #916/#917 评论)、sandbaseai/deepseek-harness-handbook runbook、dsh-overlay-check、check-dsh-profile.mjs。

## ③ dshdocs.com 与官方版本
- changelog(dshdocs.com/guides/changelog/)覆盖至 0.1.0-rc.8(2026-08-19,commit 141eb6f):唯一破坏性变更为 SQLite 存储格式;插件相关新增仅「插件可注册 settings 卡片」;无插件隔离条目。
- 官方 release:v0.1.1-rc.1(2026-08-21:V4-Flash-Vision-Exp、Bubblewrap /proc 逃逸修复)、v0.1.1-rc.2(约 2026-08-23:Files API 图像;diff 为图像管理相关)——均无 loader/插件隔离改动。
- dshdocs 排障页(2026-08-20 核查)明言「There is no skip-and-warn…截至 rc.8 仍 open,无上游修复」;社区 handbook 对 rc.8(141eb6fef8)验证:「rc.8 deliberately fails the boot if any entry is not active…does not expose a supported Web safe-mode button」。未发现官方 roadmap 页载插件隔离计划。

## ④ 本地代码验证(只读,checkout 2026-08-13 版)
- /workdir/sunquan20/AI_CODING/deepseek-harness/packages/boot/app-boot/src/index.ts:boot() 为全树事务——catch 分支 `await ctx.fiber.dispose()` 后抛 `plugin tree failed to load`,无容错/降级;assertEntriesLoaded/assertEntriesActivated 对任一 enabled entry 无 fiber/failed/pending 一律 throw(注释明言 patch「must fail loud at boot, never be silently skipped」);installFailLoud 将晚期 rejection 转 exit(1)。
- 唯一合法 fiber-less 态为配置级 `entry.disabled`(显式禁用),非失败自动跳过;无 skip 选项、无 isolated 字段(「isolate」仅见于 cordis-plugin-group realm 注释)。
- packages/boot/app-boot/src/profile.ts:DshProfileManifest 仅 `bundles?: string[]`,无 recoverableBundles。apps/cli/src/profile-boot.ts:runProfile() 全量 patch 直传 boot(),无安全模式旗标。
- 佐证:PoC commit c601d5a861 需改 29 文件(含上述 index.ts/profile.ts 及 apps/cli/src/plugin.ts、profile-boot.ts),反证上游无此路径。

## 来源(关键 URL)
- #4175: https://github.com/deepseek-ai/deepseek-harness/discussions/4175
- PoC: https://github.com/franksong2702/deepseek-harness/tree/franksong2702/recoverable-bundle-failures-poc (commit c601d5a861)
- changelog: https://dshdocs.com/guides/changelog/ ; 排障: https://dshdocs.com/troubleshooting/failed-to-import-loader-entry-ui-plan/
- releases: https://github.com/deepseek-ai/deepseek-harness/releases/tag/dsh-v0.1.1-rc.1
- handbook: https://github.com/sandbaseai/deepseek-harness-handbook/blob/main/docs/en/troubleshooting/web-client-plugin-boot-failure.md

## 结论
1. #4175(08-23)附完整 PoC,无官方回应、未转 PR,状态未澄清。
2. #1228/#2920/#3473/#1496 等同类提案均未见合并或官方回复。
3. rc.8→0.1.1-rc.2 与本地代码均无单插件失败跳过/降级路径。