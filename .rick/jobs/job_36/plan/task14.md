# 依赖关系
task13

# 写域
internal/env/web.go
internal/env/web_test.go
internal/handler/web.go
internal/cmd/web.go
internal/cmd/root.go
internal/config/config.go

# 任务目标
命令层收口：env 前端基座部署（customize/reset）+ handler.Web 编排（singleton 检测）+ cobra 命令注册。

# 关键结果
1. `internal/env/web.go`：`DeployWebScaffold() (bool, error)`——幂等抽取 embed src（webassets.SrcFS）到 `~/.rick/web/src/`（已存在且含 marker 文件 `.rick-managed` → 跳过返回 false；否则整树写出+marker）+ 写出 package.json/vite.config.ts/tsconfig（SrcFS 含这些根文件——task3 的 embed 范围确认含 src/ 全树；若不含根文件本 task 补 embed 范围调整——**注意写域只 internal/env/**，若需改 web/embed.go 则在回执中记录由 task15 收口）；`ResetWebCustomization() error`（删 ~/.rick/web/{src,dist}——保留 web.json/sessions.json）；`CheckWebEmbed() error`
2. `internal/handler/web.go`：`func Web(opts WebOptions) error`——①singleton：PidPath 存在→读 pid→probe 进程存活（syscall.Kill(pid,0)）→活=报「已在运行 PID x，http://127.0.0.1:port」退出；死=清 pid 文件继续 ②写 pid 文件（defer 删除）③config token 解析（flag>config>自动生成写回 config+打印）④组装 ServerConfig（Addr=listen:port，默认 127.0.0.1:6137；WebFS=webassets.DistFS）⑤NewServer+Serve（ctx=os.Interrupt）⑥watcher 启动 ⑦优雅退出（pid 清理）
3. `internal/config/config.go`：Config struct 加 `WebToken string \`json:"web_token,omitempty"\``（loader.go 无需动——json 往返自动兼容）；`internal/cmd/web.go`：`NewWebCmd(version string)`——runWeb（flags --port 6137 --listen 127.0.0.1 --token --verbose；version 透传至 ServerConfig/Deps 供 /api/config）+ 子命令 `customize`（调 env.DeployWebScaffold，打印指引「在任意会话中让 agent 编辑 ~/.rick/web/src 后 npm run build」）+ `reset`（确认提示+调 env.ResetWebCustomization）；`internal/cmd/root.go` AddCommand(NewWebCmd())
4. `internal/cmd/web_test.go`/`internal/env/web_test.go`：HOME 隔离——DeployWebScaffold 两连跑幂等（第二次 false）；scaffold 文件树抽查（src/main.tsx 存在）；reset 清理；singleton：占 pid 文件+假活进程（自身 pid）→ Web() 返回 already-running 错误；token 自动生成写回 config 断言
5. `go build ./...` + `go test ./internal/cmd/ ./internal/env/ ./internal/handler/... ./internal/web/... -timeout 300s` 全绿 + 构建二进制 `go build -o bin/rick ./cmd/rick` 后 `./bin/rick web --help` 显示三命令

# 测试方法
go test 三包全绿；rick web --help/--port 等输出目检；singleton 手工：两终端同起第二个报 already running

# 上下文提示
- ⚠️ 若 task3 的 embed 范围缺根配置文件（package.json 等不在 //go:embed all:src 内），本 task **不改 web/**（写域）——先在 env 内用最小 vite.config 补丁文件方案或回执上报，task15 统一收口 web/embed.go
- pid 判活注意陈旧文件（进程不存在→清文件继续）；端口占用（listen EADDRINUSE）报错含「已占用，可能已有实例」提示
- token 打印安全：只打印一次（自动生成时）；后续启动不重复打印
