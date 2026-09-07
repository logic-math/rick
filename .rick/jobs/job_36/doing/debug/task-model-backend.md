# 依赖关系
无

# 写域
internal/runtime/rpc.go
internal/web/sessions.go
internal/web/routes.go
internal/runtime/rpc_test.go

# 任务目标
后端模型切换能力（对应 pi RPC 的 set_model/get_available_models/set_thinking_level/get_available_thinking_levels）。

# 关键结果

1. `internal/runtime/rpc.go` 新增 4 个命令方法（严格按 pi rpc.md 协议——本地路径 /home/hadoop-recsys/.rick/pi/agent/runtime/node_modules/@earendil-works/pi-coding-agent/docs/rpc.md Model/Thinking 节）：
   - `SetModel(provider, modelID string) ([]byte, error)` → {"type":"set_model","provider":...,"modelId":...}
   - `CycleModel() ([]byte, error)` → {"type":"cycle_model"}
   - `GetAvailableModels() ([]byte, error)` → {"type":"get_available_models"}
   - `SetThinkingLevel(level string) ([]byte, error)` → {"type":"set_thinking_level","level":...}
   - `GetAvailableThinkingLevels() ([]byte, error)` → {"type":"get_available_thinking_levels"}
   - 遵循现有方法风格（buildNoPayload/buildWithEnvelope + 自增 id）
2. `internal/web/sessions.go` SessionManager 加三个 handler（复用 command() 模式——查现 SessionPrompt/SessionSteer 怎么向 worker stdin 发命令）：
   - `GET /api/sessions/{id}/models` → 200 {"models":[{id,name,provider,contextWindow?}], "current":{...}}——向 worker 发 get_available_models，解析 response 的 data.models（Model 对象数组，字段 id/name/provider/baseUrl/contextWindow 等——透传主要字段即可）；worker 不存在/离线 → 409（前端可提示不可用）
   - `POST /api/sessions/{id}/model {provider, model_id}` → 202（发 set_model；响应错误 → 400 with 错误信息）
   - `POST /api/sessions/{id}/thinking {level}` → 202（发 set_thinking_level）
3. `internal/web/routes.go` 挂三个路由（authWrap 包裹，与现有 sessions 路由同款）
4. rpc_test.go 补 4 命令的 JSON 精确断言（字段名/嵌套）；sessions 测试补 models 接口（fake worker 返回 canned get_available_models 响应 → handler 解析出 models 数组）

# 测试方法
go build ./... + go test ./internal/runtime/ ./internal/web/...（含新增测试）；重启 6173 验证实例（pkill -f '[p]ort 6173' + /tmp/start-verify.sh）后 curl 验证：GET /api/sessions/{真实plan会话}/models 返回模型列表（glm-5.3/catpaw-proxy 等——注意 6173 实例的 worker 可能不在（无 rpc worker）→ 409 也接受，验证错误语义；若想要 200 可用真实 6137 实例但别打扰用户——curl 只读不打扰）。

# 上下文提示
- 现有 command() helper（sessions.go:429）负责「向 worker 发命令 + 处理 response」——新 handler 复用它或同款模式
- get_available_models 响应在 rpc 事件的 response data 里（json.RawMessage）——解析时先定位 data.models
- Model 对象字段（rpc.md Types 节）：id/name/api/provider/baseUrl/contextWindow/maxTokens/cost——前端展示用 id/name/provider
