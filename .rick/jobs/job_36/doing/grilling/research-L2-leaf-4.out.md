# Research: @firstpick Pi WebUI 生态 vs @earendil-works/pi 官方实验包（L2-leaf-4）

快照：2026-08 下旬。注：沙箱内 fetch 直连（registry.npmjs.org / github.com / jsdelivr）均报 DNS ENOTFOUND，证据经 web_search 快照+原文摘录获得，下载量/stars 等数字存在时滞。

## A) @firstpick/pi-package-webui 及其姊妹包

**License / 元数据**：`@firstpick/pi-package-webui` v0.10.0（npm 首发 2026-06-01，更新 2026-08-24），MIT（© 2026 Firstpick），周下载约 1.5k；姊妹包 `@firstpick/pi-package-remote-webui` v0.1.8/0.1.9（首发 06-13，07-24 更新），MIT，周下载数百。两包均收录官方目录 pi.dev（类型 extension）。源码双仓 Firstp1ck/pi-coding-agent-forge 与 Firstp1ck/npm-packages（npm 元数据分指两仓）。

**分发形态（pi 包）**：pi 包=普通 npm 包+Pi package manifest（pi.dev 解析 manifest JSON），统一 `pi install npm:@firstpick/<name>` 安装；webui 另支持 `npm i -g` 后独立 `pi-webui` CLI。webui 以 optionalDependencies 捆绑姊妹生态（代码 OPTIONAL_FEATURE_PACKAGES 映射 17 个：btw、safety-guard、git-guided-workflow、natural-conversation、themes-bundle 等），启动只读审计、缺失降级不阻塞。

**技术栈**：服务端纯 Node 脚本 `bin/pi-webui.mjs`（无 Web 框架）；每 tab spawn 一个 Pi 子进程走官方 RPC 模式（stdin/stdout JSONL，`PiRpcProcess`+`attachJsonlReader`），多 tab 由 detached supervisor 托管（重启 HTTP 不杀 Pi 进程，`PI_WEBUI_RPC_SUPERVISOR=0` 回退）；前端 `public/` 原生 JS/HTML/CSS+Service Worker（PWA、手机紧凑视图），非 React/Vue；Git 状态走 SSE；Windows ConPTY 走可选 node-pty；测试为 Node 静态检查。

**PIN/QR 接入实现**：webui 默认只绑 127.0.0.1:31415。remote-webui 注册 `/remote` 命令：①复用/启动 webui；②询问开启 Remote PIN auth（随机 4 位 PIN，非本机浏览器须验证）；③调 localhost-only `POST /api/network/open` 重绑 0.0.0.0 开放 LAN；④`qrcode-terminal` 渲染终端二维码，QR 指向 `/remote-auth#pin=xxxx`，手机扫码免输登录（fragment 提交前清洗），屏显 PIN 兜底；`/remote close` 关闭并清 QR。README 明示为"受信 LAN 便捷门禁"，非多用户强认证。

**活跃度**：高。webui 三个月 0.1.7→0.4.x→0.9.x→0.10.0 多轮发版；仓库 6–8 月密集 feature 提交（主题、tab 恢复、supervisor、PWA）。

## B) 上游官方 @earendil-works/pi-server / pi-client / pi-protocol

**License / 元数据**：三包同属 earendil-works/pi monorepo（MIT，约 98.7k stars/12.2k forks，2025-08 创建）。`pi-protocol` v0.84.2（MIT，约 150k 周下载，3 版）；`pi-client` v0.84.3（MIT，约 1.1M 周下载——含主 CLI 被动安装，08-06 首发，4 版）；`pi-server` v0.84.2（08-14）。版本随 monorepo 锁步：0.80.3(06-30)→0.84.2(08-14)，约 6 周 20+ 版。

**进展**：#4737（TUI 连 RPC backend，05-19）维护者 badlogic 答"server mode 大重构进行中"，后由 server 架构承接；#7344（07-30 合并，+2047 行）落地 pi-protocol：CBOR+长度前缀分帧+校验的命令/事件/快照/错误 schema（125 协议测试）；#7409（07-31 合并）落地 pi-client：连接管理、exclusive/shared SessionLease、RemoteSession、client API；#7708（08 月）listSessions 破坏性改持久 SessionMetadata。#8481（08-22，本地 TUI 跑在 RemoteSession 上）按新贡献者政策当日 auto-close，未复开。仓库已含实验性 `pi client`/`pi server` CLI；但 **pi-server 无独立 CLI、无内置 coding-agent service**，须自实现 `PiServerService`；默认 Unix socket（`unix:///path`），listener 可换 WebSocket（升级期认证）；SQLite 后端（fenced writer lease、FTS5）。

**CBOR+lease 稳定性**：不稳定。server README 与文档均挂 WARNING："experimental…wire protocol (CBOR-based), transport interfaces, and lease management logic are unstable and subject to breaking changes"；0.84.0（08-06）已含 2 处 breaking；版本随 monorepo 周级滚动。缓解：握手有 PROTOCOL_VERSION 协商；`pi-server/testing` 提供协议一致性测试。

**自建 rpc supervisor 的切换成本信号**：生态主流（pi-desktop、firstpick webui）均基于 `pi --mode rpc` 的 JSONL stdin/stdout 协议——官方文档化（docs/rpc.md）且持续维护（0.84.4 仍加 clear_queue），是相对稳定层。官方 CBOR server 尚 experimental、无 CLI、须自实现 service、breaking 频繁，现在直接押注成本高；建议 supervisor 抽象传输层，盯 PROTOCOL_VERSION 与 #8481 类提案，待其稳定再迁移。

## 结论

1. firstpick webui/remote-webui 均 MIT、周更级活跃，已入 pi.dev 官方包目录。
2. 分发形态：npm 包+Pi manifest，`pi install npm:@firstpick/…`，webui 另有全局 CLI。
3. PIN/QR：/remote 重绑 0.0.0.0+4 位随机 PIN，QR 内嵌 #pin= 免输登录，仅受信 LAN 级安全。
4. webui 栈：Node 无框架 HTTP+Pi JSONL RPC 子进程+原生 JS/PWA 前端。
5. 官方三包 MIT、极活跃但全标 experimental，6 周 20+ 版，0.84.0 已含 breaking。
6. #7344/#7409 已合并落地 protocol/client；#8481 被 auto-close 未复开；#4737 由 server 重构承接。
7. pi-server 无 CLI 无内置 service，第三方须自实现 PiServerService，短期切换成本高。
8. 稳妥路线：基于官方 JSONL RPC 自建 supervisor（与 pi-desktop/firstpick 同路线），抽象传输备未来迁移。

## 信源

- npm：@firstpick/pi-package-webui — https://www.npmjs.com/package/@firstpick/pi-package-webui
- npm：@firstpick/pi-package-remote-webui — https://www.npmjs.com/package/@firstpick/pi-package-remote-webui
- pi.dev 包页 — https://pi.dev/packages/@firstpick/pi-package-remote-webui
- Firstp1ck/pi-coding-agent-forge — https://github.com/Firstp1ck/pi-coding-agent-forge
- Firstp1ck/npm-packages 提交 — https://github.com/Firstp1ck/npm-packages/commit/39d69ee9b8aa2d0c2ae480ab3d82c63a35bb9a01
- webui TECHNICAL.md — https://cdn.jsdelivr.net/npm/@firstpick/pi-package-webui@0.9.9/TECHNICAL.md
- webui DEVELOPMENT.md — https://cdn.jsdelivr.net/npm/@firstpick/pi-package-webui@0.9.9/DEVELOPMENT.md
- earendil-works/pi 仓库 — https://github.com/earendil-works/pi
- pi server README — https://github.com/earendil-works/pi/blob/main/packages/server/README.md
- pi server CHANGELOG — https://github.com/earendil-works/pi/blob/main/packages/server/CHANGELOG.md
- pi client README — https://github.com/earendil-works/pi/blob/main/packages/client/README.md
- PR #7344 wire protocol — https://github.com/earendil-works/pi/pull/7344
- PR #7409 client coordination — https://github.com/earendil-works/pi/pull/7409
- Issue #8481 — https://github.com/earendil-works/pi/issues/8481
- Issue #4737 — https://github.com/earendil-works/pi/issues/4737
- pi-protocol npm（DepScope） — https://depscope.dev/pkg/npm/@earendil-works/pi-protocol
- pi-client npm — https://www.npmjs.com/package/@earendil-works/pi-client
- DeepWiki：Session Server/Client/Protocol — https://deepwiki.com/earendil-works/pi/7.3-session-server-client-and-protocol-(experimental)
- pi Releases — https://github.com/earendil-works/pi/releases
