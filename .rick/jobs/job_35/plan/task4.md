# 依赖关系
task2

# 任务名称
重构 runtime 模块（pi 调用逻辑收口到 runtime 层）

# 任务目标
按 spec（task2）落地 KR2.4：把 pi 调用逻辑（参数解析 + 调用）收口到 runtime 层。将 `internal/agent/piagent` 迁移为 `internal/runtime`（pi 调用封装）。

**本 task 只做「包迁移 + session 就绪判定显式化」，不改 `Execute` 签名、不删 `internal/agent` 接口、不删 `internal/actpath`**——这些由 task8（做薄 cutover）与删 executor 同批完成；否则 executor（仍引用 agent/actpath/prompt）会编译断裂。

迁移内容（均在 `internal/agent/piagent/`）：`cli.go`（FindBinary/piPathOrDefault/buildArgs/CallCLI/mergeExtraArgs/CLIMode）、`executor.go`（Executor/parseStream/piEvent/piMessage/piSession）、`agentdir.go`（AgentDir/SettingsPath/RuntimeDir/RuntimeBin/FileExists/EnsureAgentDir/AgentEnv）。`internal/runtime` **继续实现 `internal/agent` 的 AgentExecutor/AgentSession**（保持 executor 可编译）。

session 就绪判定显式化：`Executor` 已从 pi JSONL 解析 session header 提取 sessionID、以 `agent_settled` 为终止信号；本 task 抽出「sessionID 非空 && settled」的判定函数并补单测（fake JSONL：有/无 `agent_settled` 两种）。注意：当前 `parseStream` 缺 `agent_settled` 时**不报错**只回退计时（`executor_test.go` 有对应断言），本 task **不改变该行为、不把 `isSessionReady` 接入 `Execute`**（只定义函数 + 单测，Execute 仍返回原行为）。

**扩展 seam（为将来 dsh runtime 留扩展位）**：`internal/runtime` 定义 `Runtime` 接口 `{ Name() string; Run(methodText string, promptFile string, cfg *config.Config) (sessionID string, trace *Trace, err error) }`——`methodText` 走 `--append-system-prompt` 注入（系统提示词、会话前注入），`promptFile` 走 user prompt（实例上下文）；`piRuntime` 为实现——handler 依赖此接口而非具体 piRuntime，将来新增 dsh 只需加 `dshRuntime` 实现并注册；`Run` 最终签名在 task8 与 `Execute→Run` 切换同批落地，本 task 先定义接口 + `piRuntime` 骨架（接口签名仅占位，以 task8 为准；同时保留 AgentExecutor 兼容 executor）。config 增加 `runtime` 字段（默认 `"pi"`），handler/env/builder 按它选实现。

参考：domain/architecture.md「DIP 组合根」；skill `verify_go_changes_skill`、`pi_runtime_verification_skill`、`fake_binary_script_skill`、`subprocess_env_isolation_skill`、`global_ref_sync_skill`；bugs.md「pi --session vs --session-id」「pi 解析器 message_end role==assistant」「托管运行时优先 PATH-fake 命中真实 pi」。

# 关键结果
1. 新建 `internal/runtime` 包，迁移 piagent 全部调用/解析逻辑；继续实现 `internal/agent` 接口（AgentExecutor/AgentSession），executor 编译不破
2. 抽出 session 就绪判定函数（`isSessionReady`：sessionID 非空 && settled），单测覆盖有/无 `agent_settled` 两种 fake JSONL
3. 所有 `internal/agent/piagent` import 改为 `internal/runtime`；`internal/agent` 接口 + `internal/actpath` **保留**（task8 删）
4. 迁移测试全绿：cli/executor/agentdir 测试随包迁移（改 package/import 路径）；真实 pi 冒烟测试（realpi/realds）在无 pi 时跳过
5. `go build` + `go test ./internal/runtime/... ./internal/cmd/... -timeout 60s` 全绿
6. 定义 `Runtime` 接口（`Name`/`Run(methodText, promptFile, cfg)`）+ `piRuntime` 骨架 + config `runtime` 字段（默认 `"pi"`）+ `Trace` 结构体（sessionID/toolCalls/finalMessage/rawLogPath/duration/settled，等价承载原 act-path + session 信息）；`Run` 启动 pi 时把 `methodText` **落盘临时文件、经 `--append-system-prompt <method文件路径>` 注入**（会话前注入系统提示词，保留 pi 默认骨架，避免长文本 inline 传参；**临时文件由 runtime 创建并 `defer` 清理、用完即删**），`promptFile` 作为 user prompt；**`CallCLI` 同样经 extraArgs 支持 `--append-system-prompt <methodFile>` 注入（交互命令 plan/easy/human-loop/ctrl 也注入 method）**；handler 依赖 `Runtime` 接口，组合根按 config 选实现（dsh 扩展位）；「`runtime` 空值 → `"pi"`」归一化落在 `LoadConfig`（`json.Unmarshal` 不回填默认值，补单测），**并同步在 `GetDefaultConfig()` 加 `Runtime:"pi"`（覆盖「无 config 文件」分支）**；**组合根落点 = `internal/cmd` 根命令的 RunE 内懒加载**（每次命令执行时读 `config.runtime` 实例化 piRuntime/piEnv/pibuilder 注入 handler，task6/7 落地）——**禁止在 `NewRootCmd`/`NewXxxCmd` 构造期 `LoadConfig`**（会触发 `~/.rick/config.json` 落盘副作用，破坏 `--help`/`--version`/测试）

# 测试方法
（本 task 测试用例设计遵循 skill:tdd + skill:testing-anti-patterns：先写失败测试；测试真实行为不 mock 被隔离接口；PATH fake 用 RICK_PI_AGENT_DIR 隔离避免命中真实 pi——见 bugs.md。）

1. 正常路径：前置条件 = task2 完成；输入 = 无；操作 = `go build -o bin/rick ./cmd/rick && go test ./internal/runtime/... -v`；预期 = build 成功，runtime 测试全绿（cli/executor/agentdir 迁移后仍绿）。
2. 边界（session 就绪判定 + config 默认值）：前置条件 = 用 fake pi 输出一段含 `{"type":"session","id":"s123"}` + `{"type":"agent_settled"}` 的 JSONL；输入 = 该 JSONL；操作 = 调 `isSessionReady` 检查返回；预期 = 有 `agent_settled` 返回 true；去掉 `agent_settled` 返回 false（本 task 不改变 Execute 不报错行为）。另 `TestLoadConfig_RuntimeDefault`：空 config → `cfg.Runtime == "pi"`（归一化）。
3. 异常（pi 缺失 + 接口保留）：前置条件 = `RICK_PI_AGENT_DIR` 指向空 temp 且 PATH 无 pi；输入 = `runtime.FindBinary(nil)`；操作 = 调用检查 error；预期 = 返回 error 含 `pi binary not found`。另 `grep -rn "internal/agent/piagent" internal/` 无残留（但 `internal/agent` 接口仍存在）。
