# 依赖关系
task23

# 写域
internal/cmd/rsi.go
internal/cmd/root.go
internal/prompt/rsi_prompt.go
internal/web/registry.go
internal/web/sessions.go
internal/web/routes.go
internal/prompt/rsi_prompt_test.go
internal/web/sessions_test.go

# 任务目标
让「启动 rick 改进」这个动作**必然携带 loop**：新增会话类型 `rsi`，后端把 `.rick/loops/rick-rsi-loop.md` 全文作为方法注入系统提示词，并对 workspace 做硬校验（与 `human-loop` 同构实现）。

# 关键结果
1. `internal/web/registry.go`：新增 `SessionTypeRSI = "rsi"`；创建请求校验分支（`rsi` 只接受 `workspace`，无需 requirement/job）——参照 `SessionTypeHumanLoop` 的校验写法。
2. `internal/prompt/rsi_prompt.go`：`BuildRSIPrompt(workspacePath string) (prompt string, methodFile string, err error)`：
   - 读取 `<ws>/.rick/loops/rick-rsi-loop.md`，**全文嵌入** bootstrap 提示词，并在开头写明「本次会话必须遵循下述 loop；loop 的产出评估由 `rick tools rsi_check` 机器校验」；
   - 提示词中显式给出 loop 依赖的工具命令（`rick tools dev-web init|restart|status`、`rick tools release --dry-run`、`rick tools release --merge-source`、`rick tools rsi_check`）；
   - `methodFile` = 该 loop 的绝对路径（沿用既有 `_method_file` 机制注入 `--append-system-prompt`）。
3. **workspace 硬校验**（fail-fast，中文错误）：
   - 必须是 rick 源码树（含 `cmd/rick/main.go` 与 `internal/web`）；
   - 必须存在 `.rick/loops/rick-rsi-loop.md`（否则提示先 `rick tools dev-web init` 或补齐 loop）；
   - **不得是生产仓库根**：解析当前进程可执行文件 → 生产仓库根（`filepath.Dir(filepath.Dir(resolve(exe)))`）；若 workspace == 该路径 → 拒绝并说明「RSI 会话必须在 dev 工作区运行，否则会直接改生产源码」。**注意**：dev 实例自己也可能跑在生产仓库里？——不存在此情形；但为稳妥，允许用环境变量 `RICK_RSI_ALLOW_PROD_TREE=1` 显式覆盖（仅用于 E2E 模拟生产）。
4. `internal/web/sessions.go`：`prepareInteractive` 增加 `rsi` 分支（调 `BuildRSIPrompt`，把 promptFile/methodFile 落进 Params）——参照 `human-loop` 分支。
5. `internal/cmd/rsi.go` + `root.go`：`rick rsi`（交互式 CLI 入口），行为与 web 一致（内部复用 `handler`/`prompt` 层，输出与 `rick human-loop` 风格一致）。
6. 测试（`t.TempDir()` 造假 rick 源码树，禁止碰真实仓库）：
   - 合规 workspace → 生成提示词含 loop 全文与关键命令；methodFile 指向 loop；
   - 非 rick 源码树 / 缺 loop 文件 / workspace == 生产仓库根 → 各自明确报错；
   - `rsi` 类型在 web 创建路径上不接受无关参数（校验分支）。

# 测试方法
go test ./internal/web/ ./internal/prompt/ ./internal/cmd/ -timeout 600s
go build ./... && go vet ./internal/web/ ./internal/prompt/ ./internal/cmd/
python3 .rick/jobs/job_36/plan/gates/gate12.py

# 上下文提示
- 参照实现：`human-loop`（`internal/cmd/human_loop.go`、`internal/handler/human_loop.go`、`sessions.go` 的 `SessionTypeHumanLoop` 分支、`registry.go:467` 附近的校验）。
- `_method_file` 是保留字段（wire 投影会剥离），resume 时会重新作为 `--append-system-prompt` 注入 —— 所以 loop 的改动在 resume 后依然生效。
- 与 task25（前端）的契约：类型字符串 `rsi`；创建请求体 `{"workspace_id":"...","type":"rsi"}`；错误为 400 + `{error:{code,message}}`（中文 message 直接给用户看）。
