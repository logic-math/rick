# 依赖关系
task12, task14

# 写域
web/dist/
web/embed.go
wiki/
README.md

# 任务目标
终局收口：dist 构建提交 + E2E 全链路验证 + embed 范围终检 + 文档。

# 关键结果
1. `make web-dist` 构建最新 dist 并**提交仓库**（web/dist 全树 + .vite/manifest.json）；web/embed.go 终检（src 全树含 package.json/vite.config.ts/tsconfig/index.html——task14 若报缺口在此补齐 embed 范围 `//go:embed all:src all:dist` 确认两 FS 完整）
2. E2E 手册化验证（写进 wiki 验证记录）：`go build -o bin/rick` → `./bin/rick web --port 16374 --token e2etest` → curl 验证：①GET / 200 含 root div ②GET /api/config 无 token 401、带 token 200 ③POST /api/workspaces 注册本仓库 ④GET /api/workspaces 列表含条目 ⑤GET /api/sessions?workspace=… 200 ⑥SSE：curl -N 带 token 3s 内收到 server_info+心跳 ⑦GET /api/workspaces/{id}/jobs 返回真实 jobs ⑧knowledge tree/file 正常 ⑨Ctrl+C 优雅退出 pid 清理 ⑩二次启动正常（pid 无残留）；全部截图/输出记录到 wiki/web-e2e-log.md
3. `wiki/web-ui.md` 用户文档：功能总览（多工作区/session 模型/监控/知识库/自迭代）、启动（rick web 参数表）、PWA 安装说明（localhost/HTTPS 限制）、自迭代指南（customize→对话改造→自动生效→reset 兜底）、安全模型（token/监听/反代 HTTPS 建议）、已知限制（并发 doing 用户自保证等——requirement 澄清结论 4）
4. README.md：架构图节补 WEB-UI 入口一行 + rick web 快速开始 5 行（指向 wiki/web-ui.md）
5. `go build ./...` + `go test ./internal/cmd/ ./internal/env/ ./internal/web/... ./internal/runtime/ ./internal/handler/... -timeout 600s` 全绿（受影响包全回归）+ `bin/rick` 重建并 `git add -f bin/rick`（⚠️ bugs.md：bin/ 被 ignore 必须 -f）
6. dist 新鲜度：源码（web/src）与 dist 一致性——本 task 构建即提交，天然一致

# 测试方法
gate7 全绿即验收（E2E 脚本化部分）+ wiki 验证记录人工可复查；`git status` 确认 dist 与 bin/rick 均已暂存

# 上下文提示
- 本 task 是唯一允许动 web/embed.go 与 README 的 task（终局收口）
- E2E 不启动真实 pi 会话（LLM 调用不在门禁内）——sessions 创建的 fake 路径已在 go test 覆盖；真实 pi spawn 冒烟由 human 手动（wiki 记录指引）
- ⚠️ git add -f bin/rick；dist 不在 ignore 内正常 add；commit 由 parent 在层检查点统一执行（worker 不碰 git——本 task 的「提交」指文件就绪，git 操作由 level_complete 完成）
