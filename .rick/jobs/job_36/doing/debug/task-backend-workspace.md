# 依赖关系
无

# 写域
internal/web/routes.go
internal/web/registry.go
internal/web/jobs.go

# 任务目标
后端补三块能力：
1. **工作区浏览 API**：GET /api/workspaces/browse?path=/some/dir —— 返回该目录下含 .rick 的子目录列表（[{path, name}]，深度 1 层，最多 50 项；路径不存在或非目录 → 400 invalid_params）。前端「选择机器上已有的工作区」用它。
2. **工作区创建 API**：POST /api/workspaces/create {path, name?} —— path 不存在则创建目录 + 初始化最小 .rick 结构（mkdir .rick + 子目录 jobs/draft/domain/loops/skills/dream 等 rick 标准结构——参考 rick 的 workspace 初始化：查 internal/workspace 或 env 里有没有 init 逻辑可复用；若 rick 无现成 init 函数则手动 mkdir 标准目录 + git init（可选，有 git 则 init））；path 已存在且无 .rick → 同样补建 .rick 结构；已含 .rick → 幂等返回 200。创建成功后调用 Workspaces.Add 注册并返回 entry（201）。
3. **prompt 原文接口**：GET /api/workspaces/{ws}/sessions/{id}/prompt —— 返回该会话的 prompt 文件原文（method + instance 两份，{method: "...", instance: "..."}；文件定位：从 session 注册表拿 workspace+type → 按类型找 prompt 文件——plan=plan/prompts/plan_prompt.md（+method 文件）、easy=doing/prompts/、ctrl=doing/prompts/ctrl_prompt.md、human-loop=draft/loops/loop_N/prompts/、learning=doing/prompts/、dream=...；若 session 注册表无此信息则按 workspace 下最新 job 猜；找不到 → 404 not_found）。文件读取复用 jobs.go 的 ReadJobFile 白名单机制（若路径在白名单外则直接 os.ReadFile + 安全校验）。

# 测试方法
go build ./... + go test ./internal/web/...（补：browse 正常/非法路径、create 幂等/目录已存在无 .rick 补建、prompt 接口 plan 类型返回 plan_prompt.md 原文）；真实验证：重启 6173 隔离实例（注意 pkill -f '[p]ort 6173' 防自匹配）后 curl 三个接口。

# 上下文提示
- registry.go 的 WorkspaceRegistry.Add 已存在（校验 .rick 目录 + 幂等返回 created bool）——create API 复用它注册
- jobs.go 的 ReadJobFile 白名单是 plan/** doing/** —— prompt 文件多在 plan/prompts 或 doing/prompts 下，白名单内；human-loop/dream 在 draft/ 或 .rick/dream/ 下（白名单外）——本 task 可在 ReadJobFile 增加 readPromptFile 独立实现（安全校验：路径 Clean + 前缀校验 + 2MB 上限）
- 前端并行 task-A 会调用这些接口（契约：browse 返回 {workspaces:[{path,name}]}、create 返回 WorkspaceEntry、prompt 返回 {method,instance}）——本 task 实现后若前端未就绪，curl 验证即可
