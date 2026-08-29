# 依赖关系
task5

# 写域
internal/web/sse.go
internal/web/sse_test.go
internal/web/auth.go
internal/web/auth_test.go

# 任务目标
SSE 事件总线（单流多路复用 + Last-Event-ID 重放）+ token 认证中间件。

# 关键结果
1. `internal/web/sse.go`：
   - `type Hub struct` + `NewHub(bufSize int)`（重放环形缓冲默认 1000）：`Publish(evt Envelope)`（全局单调 seq 分配+缓冲记录+广播所有订阅者）；`Subscribe(lastEventID int64) *Subscription`（含 buffered chan 256 + 重放：把缓冲中 seq>lastEventID 的先推）；`Unsubscribe(sub)`；慢消费者：chan 满丢最旧+计数（被丢事件数暴露给 /api/config 诊断或日志）
   - `type Envelope struct{Seq int64; Type string; SessionID string; Data json.RawMessage}`（序列化格式=api-contract SSE 节）
   - `ServeSSE(w, r, hub, token)`：text/event-stream；写注释心跳 `: ping` 每 15s（ticker）；Last-Event-ID 头解析；客户端断开（r.Context().Done()）清理；响应中途错误不断流重连（一直 200）
   - 事件构造辅助：`SessionEvent(sessionID, piEvent)`（透传 rpc 事件原样入 Data）、`SessionStateEvent`、`JobsUpdateEvent`、`FrontendReloadEvent`
2. `internal/web/auth.go`：`TokenAuth(token string) func(http.Handler) http.Handler`——校验 Bearer header 或 `?token=`（双通道：REST 走 header，SSE/静态外的 GET 也接受 query——EventSource 场景）；空 token 配置=不鉴权（本地开发模式）；失败 401 统一错误体
3. `internal/web/sse_test.go`：httptest 覆盖——订阅后 Publish 事件可达；Last-Event-ID 重放只补 seq 之后的；多订阅者广播；心跳存在（短 ticker 注入测试）；慢消费者丢旧不阻塞 Publish；断开清理（goroutine 泄漏检查 goleak 可选）
4. `internal/web/auth_test.go`：无 token 401；Bearer 通过；query token 通过（SSE 场景）；空 token 配置全放行
5. `go build ./...` + `go test ./internal/web/ -run "TestSSE|TestAuth" -timeout 60s` 通过

# 测试方法
go test ./internal/web/ -run "TestSSE|TestAuth" -v 全绿；curl -N 手工 smoke（本地起 httptest 服务器临时 main 可选）

# 上下文提示
- 协议细节来自 dashboard 断线续传设计（事件序号+subscribe(lastSeq)）映射到 SSE Last-Event-ID（research-L2.md）
- Go 标准库实现（http.Flusher 每事件后 Flush）；不引 WS 库——单流 SSE 已裁决
- ⚠️ 代理缓冲坑：写 X-Accel-Buffering: no 响应头（nginx 反代场景）
