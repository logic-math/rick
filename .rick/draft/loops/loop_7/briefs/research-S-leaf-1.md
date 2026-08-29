# DeepSeek Harness (dsh) 架构与定制化调研（2026-08-13 开源，快照 08-21~23）

## 1. 架构
- 内核 Cordis：源码 vendored 进 monorepo 并更名 @deepseek-ai/cordis / cordis-plugin-*（自持框架层，可审计可打补丁），设计见于论文 "A Programming Paradigm for Spatiotemporal Composability"。
- 插件向共享 ctx 贡献 services / typed events / reversible effects；一切注册都是 effect，插件卸载（配置编辑、热重载、显式 dispose、必需服务丢失）时自动回滚，无特权核心可 patch。
- 「everything is a plugin」可换能力：model adapter、tool registry、session log、agent loop 本身（官方 reference）；官网另列 models、tools、skills、sessions、sandboxes、storage、loops、scheduling、UI；沙箱三模式 read-only/workspace-write/danger-full-access；可选 Codex/Claude Code provider 在 base bundle 之外。
- 启动组合：运行中 dsh = 有序 layers 组合的 plugin tree；profile = $DSH_HOME/profiles/<name>/ 命名组合；层序为 profile manifest（dsh.profile.bundles）各 bundle patch → profile cordis.patch.yml → home 级 cordis.patch.yml → --patch（argv 序），后层逐行覆盖整行；dsh-base bundle（模型适配器、默认模型、工具、持久化、策略、凭据、遥测、spawn/fork subagent）最先应用，dsh-web-app / dsh-headless 模式 bundle 叠加；app-boot 组装 env→profile→bundle→patch，失败以 dsh: 前缀退出。

## 2. 插件开发 API
- 插件形态：实现 Service 的对象——带可选 inject 与 apply(ctx) 的函数，或 Service 子类；ESM 模块 named-export apply(ctx: Context)。
- cordis.yml 列插件条目（name=模块 specifier：相对路径或 npm 包），条目并发启动；ctx.plugin() 从代码挂载；ctx.effect() 包裹定时器/连接/watcher 并返回 disposer。
- 官方 Cordis tutorial（可运行章节）与 cookbook「adding a package」指导新增 @deepseek-ai/dsh-* 工作区包。

## 3. 语言/运行时/许可证
- TypeScript 为主（另有 JS/CSS/Shell 等）；Node 需 22.19+ 或 24+（engines ^22.19.0 || >=24.0.0，排除 23.x；CI 覆盖 22.19/24/26）；pnpm@11.7.0（Corepack）+ Git≥2.26；MIT（THIRD_PARTY_NOTICES.md 列三方依赖）；零安装 npx @deepseek-ai/dsh web（默认 127.0.0.1:3080）。

## 4. 活跃度（截至 2026-08-24）
- stars 189,495 / forks 21,139（GitHub 快照）；commits 7,090 / committers 34（Ecosyste.ms）；top：tianyicui 5268、LegGasai 1506、imccyu 1262。
- repo 创建 2026-08-13；releases 全为 pre-release：最新 v0.1.1-rc.2（08-21，imccyu），此前 0.1.0-rc.5（08-13）、rc.7（08-17）、rc.8（08-19）、0.1.1-rc.1（08-21）。

## 5. 成熟度
- README：developer preview，"THERE WILL BE COMPATIBILITY-BREAKING CHANGES"。
- 官方已知限制（各包 README "Known Limitations"）：session 分支树推迟、fork() 仅限活跃会话稳定边界；ACP 仅 fresh session（不支持 load/list/resume/delete/fork）、仅栅格图+单 workspace；subagent SDK 每次运行新建进程、无池化；沙箱后端不可用则 fail-closed SANDBOX_UNAVAILABLE；Node<22.19 曾报含混 ESM 崩溃（Discussions #2327 已改为清晰诊断）。

## 来源
- GitHub repo: https://github.com/deepseek-ai/deepseek-harness
- 官网: https://deepseek.com/harness/en/
- 架构文档: https://deepseek-harness.github.io/deepseek-harness/en/reference/ ；https://github.com/deepseek-ai/deepseek-harness/blob/master/docs/architecture.md
- Cordis 入门/教程: https://deepseek-harness.github.io/deepseek-harness/en/reference/cordis-primer ；https://deepseek-harness.github.io/deepseek-harness/en/develop/cordis-tutorial/
- 启动/组合: https://github.com/deepseek-ai/deepseek-harness/blob/master/apps/cli/reference/README.md ；https://github.com/deepseek-ai/deepseek-harness/blob/master/apps/cli/composition.md
- vendor/开发环境: https://github.com/deepseek-ai/deepseek-harness/blob/master/vendor/README.md ；https://github.com/deepseek-ai/deepseek-harness/blob/master/docs/development.md
- Releases: https://github.com/deepseek-ai/deepseek-harness/releases
- 统计: https://summary.ecosyste.ms/projects/378897
- 已知限制: packages/core/session/README.md；packages/acp/acp/README.md；packages/subagent/subagent-dsh-sdk/README.md（均在同 repo master）；https://github.com/deepseek-ai/deepseek-harness/discussions/2327
