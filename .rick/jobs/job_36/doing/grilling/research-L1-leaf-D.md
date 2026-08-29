# 调研：Go 内嵌 Web UI 技术栈选型惯例（D 组）

## 前端栈对比

**① SPA（React/Vue/Svelte + Vite → dist → go:embed）** —— 对「流式 delta + 工具卡片 + 会话树 + 移动端响应式」适配最佳：react-virtuoso 内置聊天式流式列表（动态高度/自动吸底/分组），TanStack Virtual 官方支持 chat/AI 流的 end-anchoring 与反向加载；Vue/Svelte 虚拟滚动生态明显弱于 React。代价：常规流程需 node 参与 CI/本地构建。先例：**Prometheus**（React mantine-ui，static/ 产物 embed 进二进制，npm 构建在 CI、不提交）；**Gitea**（webpack→Vite 迁移完成，`make frontend` 需 Node 22+，bindata tag 嵌入，dist 不提交）；**Woodpecker**（Vue，CI 用 node 镜像构建）。**「预构建 dist 提交进仓库」有明确先例**：**AdGuardHome** master 直接 `//go:embed build`（注释即 "prebuilt client"，另设 FRONTEND_PREBUILT 开关）；**kubeaquarium** 专 commit dist 以保 `go install` 免 node；有项目用 CI 校验「提交的 dist 与源码重构建一致」；**Vaultwarden** 用独立仓库（bw_web_builds）放预构建产物、构建期下载；**filebrowser** 前端独立仓库、构建期嵌入。反例哲学：**GoToSocial** 完全 backend-first，不内嵌 SPA，仅静态模板页。

**② htmx/服务端渲染** —— 零前端构建链、html/template 出页面，htmx 有 SSE 扩展；但高频 token delta 每事件触发 DOM 替换、无虚拟滚动生态、会话树交互需自写 JS，对实时轨迹流适配度中下。

**③ 纯 vanilla JS** —— 构建链最干净。先例：**Syncthing**（gui/ 原生 JS+Bootstrap 直接 embed 进二进制，零 npm）；**ddns-go**（embed 单 HTML 模板）。但无组件化/虚拟滚动/响应式生态，长轨迹+树导航手写成本高。

## WS/SSE 库选型

- **标准库 SSE（net/http + http.Flusher）**：零依赖；LLM 业界事实标准（OpenAI/Anthropic/Google 全用 SSE 推 token）；客户端命令走普通 POST，最贴「单向推送+偶发命令」。坑：代理缓冲——Go ReverseProxy 需 FlushInterval=-1（官方 issue #47359/#64045），nginx 需 `X-Accel-Buffering: no`；EventSource 自带断线重连。
- **gorilla/websocket**：2022-12 归档 → 2023-07 复活（新维护团队），现活跃、存量最大。
- **coder/websocket（原 nhooyr.io/websocket）**：Coder 接管维护、活跃；context 原生、内置并发写保护，新项目推荐。
- **gobwas/ws**：零分配、百万连接级性能天花板，但 API 偏底层，v1.4.0（2024-05）后更新缓慢；rick 规模用不到。

## 结论

1. 选 Vite+SPA、预构建 dist 提交仓库，node 不进 rick 构建链，先例充分【信源：一级，项目仓库/官方文档】
2. 传输用标准库 SSE + POST 命令，LLM 流式事实标准，代理坑可解【信源：一级，Go 官方 issue + 官方文档】
3. 暂不引 WS 库；确需双向时选 coder/websocket 或 gorilla【信源：一级，官方仓库公告】

## 信源

- Gitea 官方文档（Node 22+ 必需、make frontend/bindata）: https://docs.gitea.com/development/hacking-on-gitea
- Gitea webpack→Vite 迁移 PR: https://github.com/go-gitea/gitea/pull/37002
- Prometheus web/ui README（React 产物 embed）: https://github.com/prometheus/prometheus/blob/main/web/ui/README.md
- AdGuardHome main.go（embed 预构建 build/ 目录）: https://github.com/AdguardTeam/AdGuardHome/blob/master/main.go
- AdGuardHome 预构建 tarball 议题: https://github.com/AdguardTeam/AdGuardHome/issues/2958
- kubeaquarium commit「commit prebuilt frontend so go install works」: https://github.com/gabriel-dantas98/kubeaquarium/commit/38ec12b992207b6e1e951d73bd1982e4721ac593
- Vaultwarden 预构建产物仓库 bw_web_builds: https://github.com/dani-garcia/bw_web_builds
- Syncthing 官方文档（gui 目录直接打包进二进制）: https://docs.syncthing.net/dev/web.html
- GoToSocial 文档（backend-first、无内嵌客户端）: https://docs.gotosocial.org/en/latest/
- filebrowser Vue3 迁移 commit: https://github.com/filebrowser/filebrowser/commit/5100e587d73831ecdb5e3bd35a78fef96ad248a4
- Woodpecker CI 配置（node 镜像构建前端）: https://github.com/woodpecker-ci/woodpecker/blob/817ff37d/.woodpecker/docker.yaml
- ddns-go（embed 单 HTML）: https://github.com/jeessy2/ddns-go
- gorilla 复活公告（2023-07-17）: https://gorilla.github.io/blog/2023-07-17-project-status-update/
- gorilla 归档始末（Chainguard）: https://www.chainguard.dev/unchained/the-archiving-of-the-gorilla-web-toolkit-a-tale-of-two-software-security-risks
- coder/websocket 仓库（含 gorilla 对比）: https://github.com/coder/websocket
- gobwas/ws 仓库: https://github.com/gobwas/ws
- Go 官方 issue：ReverseProxy 需 Flusher 才能 SSE: https://github.com/golang/go/issues/64045
- Go 官方 issue：flushinterval 与 text/event-stream: https://github.com/golang/go/issues/47359
- SSE vs WebSocket LLM 流式综述: https://speedtesthq.com/guides/ai/streaming-llm-responses-sse-vs-websocket
- TanStack Virtual chat/AI 流文档: https://tanstack.com/virtual/latest/docs/chat
- react-virtuoso（聊天流式列表）: https://virtuoso.dev/
- Go WebSocket 库选型指南: https://websocket.org/guides/languages/go/
