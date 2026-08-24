# 依赖关系
task2, task4

# 任务名称
落地 env 模块（四职责：pi + 生态扩展 + rick 定制 + 就绪 check）

# 任务目标
按 spec（task2）落地 KR2.2 并升级为 env 四职责。新建 `internal/env` 包，把 pi 相关能力收口为「保证 pi 正确启动」的统一管理器，职责：
1. **安装并更新 pi agent**：迁移 `ensurePI`/`installManagedPI`/`requireNodeForPiInstall`/`piVersion`（来自 tools_init_pi.go）
2. **安装并更新 pi 生态扩展/插件/skill**：迁移 `ensureNpmExtension`/`piListContains`/`verifyExtensions`（pi-subagents/pi-web-access/主题）
3. **安装并更新 rick 自有定制（hook/skill/agent）**：新增 `DeployRickCustomizations()`，把 rick-gates hook 扩展、think/research/exporter agent frontmatter、rick skills 落盘到 `~/.rick/pi/agent/`（agents/、extensions/、skills/）——rick 全局方法（「你是 rick 的 agent，遵循 loops/skills/domain 体系」）作为 builder method 的固定前缀（task5），不单独落盘 `APPEND_SYSTEM.md`（避免与 `--append-system-prompt` 覆盖冲突）
4. **就绪 check 函数**：新增 `IsPIReady`/`CheckPIInstalled`/`CheckEcosystemExtensions`/`CheckRickAgents`/`CheckRickHooks`（纯「功能点就绪」，不含 session）

**扩展 seam（为将来 dsh runtime 留扩展位）**：env 四职责按 `RuntimeEnv` 接口组织 `{ Ensure() error; DeployCustomizations() error; CheckReady() []string }`，pi 实现 = `piEnv`；将来 dsh = `dshEnv`（安装方式/扩展机制/定制落盘格式/就绪 check 各自实现），cli/handler 不改。

`internal/cmd/tools_init_pi.go` 变薄为 Cobra 入口，调用 env 导出函数；init-pi 行为不变（幂等、全 ✅）。env 从 `internal/runtime` import `AgentEnv`/`AgentDir`/`RuntimeDir`/`RuntimeBin`/`SettingsPath`/`EnsureAgentDir`/`FileExists` 注入 `PI_CODING_AGENT_DIR=~/.rick/pi/agent` 保持配置隔离（尊重 RICK_PI_AGENT_DIR/HOME）——依赖 task4 先落地 runtime，避免与 piagent 改名冲突。theme 相关（`embeddedThemes` go:embed + `ensureRickTheme`/`setTheme`/`currentTheme`/`purgeTokyoNight`/`piSettingsPath`/`ensureHideThinkingBlock`）随 env 迁移：`internal/cmd/themes/*.json` 移到 `internal/env/themes/`，go:embed 指令随之更新；`tools_theme.go` 变薄调用 env。

参考：loop `agent-runtime-bootstrap-loop`；skill `verify_go_changes_skill`、`global_ref_sync_skill`、`pi_extension_install_verification_skill`、`pi_runtime_verification_skill`、`fake_binary_script_skill`、`subprocess_env_isolation_skill`、`pi_theme_verification_skill`；bugs.md「fake pi PATH 替换」「pi 扩展假成功」「RICK_PI_AGENT_DIR 隔离」；pi docs/extensions.md（hook 扩展入口）。

# 关键结果
1. 新建 `internal/env/`，含四职责对应文件（如 `pi.go`/`extensions.go`/`customizations.go`/`check.go`），迁移函数全导出版本（迁移清单须覆盖：`ensurePI`/`installManagedPI`/`requireNodeForPiInstall`/`piVersion`/`ensureNpmExtension`/`piListContains`/`verifyExtensions`/`piCommand`/`bootstrapAgentSettings`/`ensureRickTheme`/`purgeTokyoNight`/`currentTheme`/`setTheme`/`ensureHideThinkingBlock`/`extensionManagedByRick`/`containsString`/`piSettingsPath`/`legacyPiSettingsPath`，及 `requiredExtensions`/`tokyoNightPkg`/`tokyoNightThemes`）
2. 职责 3 `DeployRickCustomizations()`：**task3 创建仓库侧 `.rick/skills/rick-gates/helper.py` 占位文件 + 目录骨架**（实际门禁校验逻辑 tasks.json 可解析/zombie/commit_hash 由 task8 填充），`DeployRickCustomizations` 幂等复制 rick-gates + rick skills 到 `runtime.AgentDir()/extensions/rick-gates/`；**think/research/exporter agent 文件本 task 不写、完全留给 task9**（避免 task3 占位文件被 task9 幂等跳过）
3. 职责 4 check 函数：`IsPIReady() (bool, []string)` 汇总所有功能点就绪判定；不含任何 session 校验（session 校验归 runtime）
4. `internal/cmd/tools_init_pi.go` 变薄；命令输出与迁移前逐字一致（**「逐字一致」用 golden 输出快照 diff 验证：迁移前捕获 init-pi 输出基线，迁移后 diff 一致**）；`tools_init_pi_test.go` 中引用已迁移未导出 helper 的断言随迁 env 或改写；`tools_theme_test.go` 及共享测试 helper（`setupPiSettings`/`setupLegacyPiSettings`/`readManagedSettings`）**在各自 test 文件内复制/共享（测试工具，不导出为生产 API，避免仅测试用的生产方法）**
5. `go build` + `go test ./internal/env/... ./internal/cmd/... -timeout 60s` 全绿；隔离 HOME 下 `rick tools init-pi` 全 ✅
6. 定义 `RuntimeEnv` 接口（`Ensure`/`DeployCustomizations`/`CheckReady`）+ `piEnv` 实现（四职责收口于 piEnv）；为将来 `dshEnv` 留扩展位

# 测试方法
（本 task 测试用例设计遵循 skill:tdd + skill:testing-anti-patterns：先写失败测试；用 fake 脚本测真实分支非 mock 行为；fake 脚本开头恢复 PATH——见 bugs.md。）

1. 正常路径：前置条件 = task2、task4 完成、隔离 HOME（`t.Setenv("HOME", t.TempDir())`，PATH 指向含 fake pi/node/npm/npx 的目录，或先预置 managed runtime 使 node 检查跳过）；输入 = 无；操作 = `go build -o bin/rick ./cmd/rick && ./bin/rick tools init-pi`；预期 = exit 0 且 stdout 含 `✅ pi environment ready`。
2. 边界（幂等 + 四职责 check）：前置条件 = init-pi 已成功一次；输入 = 无；操作 = 再次 `./bin/rick tools init-pi` + `go test ./internal/env/... -run TestIsPIReady -v`；预期 = 第二次仍 exit 0 且无 `newly installed`；`IsPIReady()` 返回 ok=true、missing 为空，且 `CheckPIInstalled`/`CheckEcosystemExtensions`/`CheckRickAgents`/`CheckRickHooks` 各自就绪（返回 nil/空切片）。
3. 异常（缺 node/npm + check 报告缺失）：前置条件 = runtime 未装；输入 = 无；操作 = `HOME=$(mktemp -d) PATH=$(mktemp -d) ./bin/rick tools init-pi`（PATH 指向空目录，确保 `exec.LookPath("node")` 必然失败；保留 HOME 避免 config 加载失败干扰）；预期 = stderr 含 `requires Node.js`，exit 1，不 panic。另测 `CheckEcosystemExtensions()` 在某扩展缺失时返回非空切片（不就绪即列出）。
