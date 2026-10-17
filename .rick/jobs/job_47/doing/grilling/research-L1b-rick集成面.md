# research-L1b 简报：rick 集成面盘点（super 模式事实底座）

> 阶段 L1b · 对象 /workdir/sunquan20/AI_CODING/rick（Go CLI，VERSION=5.0.13，cmd/rick/main.go:10）· 时间基准 2026-10-10 · 信源=代码直读（Read/Grep 直核为主，叶子文件 leaf-1/3/4 辅助，见文末清单）

## §1 CLI 注册模式
- 框架=cobra（root.go:8）。NewRootCmd(version) 内 AddCommand 注册 9 子命令：plan/doing/learning/dream/tools/human-loop/ctrl/easy/web（root.go:67-75）。【0.95】
- 全局 PersistentFlags：--verbose/--dry-run/--job/--resume（root.go:57-60），包级 getter GetVerbose/GetDryRun/GetJobID/GetResume。【0.95】
- 新增子命令标准模式（easy.go 全文 56L）：internal/cmd/super.go 定义 NewSuperCmd() *cobra.Command → Flags().StringVarP 局部 flag（easy.go:51-53）→ RunE 内组 handler.Options{Verbose,DryRun,JobID} 调 handler.Super(...)；root.go 加一行 AddCommand。【0.95】
- 分工：cmd=flag 解析+组合根（doing.go:93-98 LoadConfig+runtime.NewPiRuntime 注入 handler.Doing）；handler=调度聚合层，禁 import internal/cmd（options.go:1-9，防循环依赖），编排 env→builder→runtime。web.go 是组合根例外（装配 internal/web 后注入 handler.Web）。【0.95】

## §2 监督树原语清单（internal/runtime/supervisor.go，877L）
- Supervisor：NewSupervisor(cfg)(:148)、Spawn(spec SpawnSpec)(*Worker,error)(:172)、Get(sessionID)(:329)、Count()(:336)、Remove(:353)、CloseAll(:360)。【0.95】
- Worker：SessionID()(:412)、IsDead()(:415)、Reason()(:425)、Dropped()int64(:434)、Events()<-chan *RpcEvent(:438)、StderrTail()string(:441)、State()*WorkerState(:445)、LastEntryID()(:457)、Send([]byte)error(:466)、Close()(:504)。【0.95】
- 类型：SupervisorConfig{MaxActive,IdleTimeout,HeartbeatInterval,HeartbeatTimeout,ProbeTimeout,TermGrace,PiPath,ExtraArgs,OnDead func(sessionID,lastEntryID string)}(:82-107)；SpawnSpec{SessionID,Dir,MethodFile,PromptFile,SessionIDFlag,CreateNew}(:111)；WorkerState{IsStreaming,IsCompacting,SessionID,SessionFile}(:131)。【0.95】
- Setpgid：Spawn 设 cmd.SysProcAttr{Setpgid:true}（:219 附近，注释明示 kill(-pgid) 收割整树含 bash 孙进程）。【0.95】
- 整树收割梯子：Worker.Close()(:504-541)=先 RpcClient.Abort() 冲刷 → syscall.Kill(-pid,SIGTERM) → 等 TermGrace(默认 5s，:67) → SIGKILL → 再等 2s → 关 stdin。【0.95】
- 心跳/超时：heartbeatLoop(:786) get_state 节奏（默认 30s，超时 2×interval），未应答 markDead("heartbeat timeout")；ProbeTimeout=500ms 探活窗；reapLoop(:820) IdleTimeout 空闲收割（IdleDisabled=-1s 哨兵禁用）。默认值常量组 :42-74（MaxActive=64/eventChanBuffer=256/stderrTailSize=4096/maxEventLineSize=64MB）。【0.95】
- OnDead 游标重放：markDead(:552) deadOnce 单发，携 lastEntryID（get_entries 游标），goroutine 回调 cb(sessionID,lastEntryID)——调用方可从游标重放事件。【0.95】
- stderrTail：stderr drain goroutine 写 byteRing(:854) 环形缓冲 4096B，StderrTail() 供探活失败诊断。【0.95】
- 调用方（现役唯一=web 模式）：组合根 cmd/web.go:236 NewSupervisor(resolveSupervisorConfig(cfg))（web.go:293-309：MaxActive←config web_max_active 默认 64，IdleTimeout=IdleDisabled 持久持有，心跳/探活/宽限用包默认，**OnDead 生产未接线=传 nil**）；internal/web/sessions.go:452（新建 CreateNew:true+Prompt(BootstrapMessage)）/1005（resume）/1158（并发 continue），Get/IsDead/Send/Remove/State/Events/CloseAll 散见 :671-1904（Events 关闭→:1811 markWorkerLost 判死）；**recovery.go:221/245：关停快照先于 CloseAll 用 LastEntryID 落盘——游标重放的生产先例**。CLI doing 不用 supervisor。【0.9】
- 测试：supervisor_test.go 21 用例（fake pi 脚本驱动，via PiPath+RICK_PI_AGENT_DIR，无真 pi 依赖）——含 GroupKillReachesGrandchild(:341 孙进程收割实证)/CloseForceKill/CloseGraceful/HeartbeatDeath(:604)/OnDeadOnExternalKill(:445 外杀→OnDead 携 lastEntryID 单发)/IdleReap/MaxActive/SpawnDirAndEnv/SpawnFlagSemantics/EventsOverflowDropsOldest 等。【0.9】
- ⚠️ 关键限定：Spawn 为 **pi RPC 专用**（硬编码 `--mode rpc`+RpcClient JSONL 协议+get_state 心跳语义）——py 调度器进程不能直接 Spawn；可复用的是原语语义（Setpgid/kill 梯子/OnDead 游标/stderrTail/探活）+测试模式（fake 脚本）+recovery 快照先例，需抽通用 proc 层或新写 PySupervisor。RFC §4.1「原语组现成」=六原语全部真实有测试（孙进程收割有实证）、但入口耦合 pi 协议且 OnDead 生产未接线。【0.9】

## §3 pi 派发机制（py 侧可用途径分析，J4 裁决输入）
- 4 种调用形态：①ModeInteractive（继承 stdio，cli.go:92）②ModePrint=`pi -p`（cli.go:73-91，pi 无权限弹窗故无需 skip-permissions flag）③一次性 `pi --mode json`（runtime.go:130 RunIn：method/prompt 双层 --append-system-prompt + `-p <bootstrapMessage>`；NDJSON camelCase 事件流，agent_settled=终止，executor.go:101 parseStream）④常驻 `pi --mode rpc`（supervisor.Spawn；stdin JSONL 命令 Prompt/Steer/FollowUp/Abort/NewSession/GetEntries/GetState…，rpc.go:178-335）。【0.95】
- 二进制解析序：cfg.PiPath → RuntimeBin()（~/.rick/pi/agent/runtime）→ PATH（cli.go:30-50）；每个 pi 子进程 Env=AgentEnv()=os.Environ()+PI_CODING_AGENT_DIR=<~/.rick/pi/agent>（agentdir.go:68-73，配置隔离）。【0.95】
- doing 的 worker 派发=rick **不自起 worker 进程**：internal/builder/orchestration.go:357+ renderWorkflowSection 把 `runs.all([{key,agent:'worker',timeoutMs,task}])` workflowScript 渲染进 doing prompt，由 parent pi 会话的 subagent 工具执行；监控/干预={action:"status",view:"transcript"}、steer/stop、.pi/subagents/artifacts/<runId>_*_meta.json（templates/doing.md:14-17）。【0.9】
- maxSubagentDepth=2 是 **pi 侧**封底约束，rick 代码仅注释引用（internal/env/agents.go:44），无 Go 侧强制。【0.9】
- worktree 隔离：doing **不用** git worktree——worker 共仓，靠 plan `# 写域` 声明+orchestration.go 确定性两两互斥校验；真 worktree 仅 devweb dev 沙盒（env/devweb/devweb.go:123 IsDevWorktree）。SpawnSpec.Dir 只是 cwd 锚定（工作区根），非 git 隔离。【0.9】
- py 进程派发 pi 的现成途径：①subprocess 直调 `pi --mode json -p --append-system-prompt <f> <bootstrap>` 解析 stdout NDJSON——现成可行（executor_realpi_smoke_test.go 有真 pi 用例，go:build realpi + RICK_REAL_PI_ARGS 门控）；②`pi --mode rpc` 常驻协议（可 steer/get_entries 游标续传，适合长会话池）；③py 设 PI_CODING_AGENT_DIR=~/.rick/pi/agent 即继承 rick 托管 agents（think/research/exporter 注册于 ~/.rick/pi/agent/agents/*.md，env/agents.go:70+）/扩展/skills。【0.85】

## §4 web 模式接入点
- 组合根：cmd/web.go:215-237（NewPiRuntime+NewSupervisor(resolveSupervisorConfig)+SessionManager+NewServer 装配后注入 handler.Web；internal/web 反向 import handler，web.go 是唯一组装根）。【0.9】
- 「从 server 启动后台进程」三先例：①CLI --daemon 后台化（web.go:65-103：os.Executable re-exec + RICK_WEB_DAEMON=1 + SysProcAttr{Setsid:true} + 日志追加 stateDir/web.log）——仓库内唯一的"脱离终端拉起独立进程"先例；②server 内 startBackground goroutine（sessions.go:477：不 spawn 子进程，goroutine 跑 DoingRunner/DreamRunner 注入缝（:52-55 类型、:108-116 nil→默认 handler.DoingIn/DreamIn），进度→hub.Publish SSE）；③sup.Spawn pi rpc 常驻 worker（sessions.go:444-455 交互型会话）。【0.9】
- super py 调度器接入点：a) DoingRunner 型注入缝+新 session 类型（最自然，复用 SSE/状态机）；b) 仿 --daemon 的 re-exec 后台化（CLI 侧）；c) sup.Spawn 型新 spawn 路径（py 非 pi 协议，需新原语）。路由：routes.go:896 RegisterRoutes，Go1.22 method-pattern+authWrap。【0.85】

## §5 配置系统
- Config（config.go:4-34）：MaxRetries/Runtime/PiPath/PiExtraArgs/DefaultWorkspace/Git{user_name,user_email}/HumanLoop{max_retries,research_source_weights,think_top_n,sense_max_backflows,think_min_assumptions}/WebToken/WebMaxActive。【0.95】
- 加载：仅 ~/.rick/config.json 单一 JSON（loader.go:26,70），无 env/viper/flag 绑定；文件不存在自动写默认（loader.go:35-68）；二进制名 _dev 后缀→.rick_dev 目录（loader.go:22-26）。config.example.json 仅示例非 schema。【0.95】
- super 配置挂法：Config 顶层加 `SuperConfig` 嵌套结构体 + json:"super,omitempty"——完全照 HumanLoopConfig(:29-34)+WebToken/WebMaxActive(:17-23) 先例（注释明示 omitempty 对 json round-trip 透明、loader 零改动）。字段建议：py 解释器路径/super 数据目录/任务 env/pi 池参数。【0.9】

## §6 资产/构建/测试机制
- go:embed 三处：web/embed.go:26-31（all:src+public+根配置；DistFS=all:dist，**dist 提交仓库，node 不进 rick 构建链**）；internal/prompt/manager.go:22-52（10 个 .md 模板 string + skills FS）；internal/env/settings.go:24（themes/*.json）。【0.9】
- prompt 注入：manager.go:65 NewPromptManager(dir 可选，空=embed)、:77 LoadTemplate 缓存+dir 优先 embed 兜底、:281 WriteSkillFileWithVars（{{k}} ReplaceAll）；builder.go:39/107 Build/BuildAndSave=预渲染→写文件→交 pi。**py 调度器 prompt 预渲染可直接借鉴此链路（预渲染落文件→路径传递）**。【0.9】
- 仓库 Python：①.rick/skills/rick-gates/helper.py（**git-tracked**，64 行确定性门禁）——调用=handler/doing.go:225-260 runDoingGate `exec.Command("python3",helper,doingDir)`，兜底 ~/.rick/pi/agent/extensions/rick-gates/（init-pi 部署，env_test.go 验证）；②tests/mock_agent/mock_agent.py（MOCK_SCENARIO 环境变量驱动假 agent：text/json/rpc 模式+故障注入，测试基建）；③tests/e2e_v2_test.py（4 阶段 e2e）。另 .rick/skills/ 下多 skill 自带 .py 均 git-tracked。【0.9】
- 构建：Makefile 仅 web-dist(npm ci+build)/web-dev(vite 代理 127.0.0.1:6137)；主构建=scripts/build.sh:96 `go build -o bin/rick cmd/rick`（go≥1.21）；**无 make test 目标**。【0.95】
- 测试约定：单测=同目录 _test.go（internal/web 近半文件为测试）；cli_integration_test.go=进程内 cobra NewRootCmd+SetArgs+--help（无 build tag）；真实 pi 冒烟=//go:build realpi+RICK_REAL_PI_ARGS env 门控（默认 go test 不触真 pi）；另有 tests/*.sh 脚本级集成。【0.9】

## §7 workspace/jobs 模型
- 目录：.rick/{loops,skills,domain,jobs,dream,draft}（workspace/paths.go:11-26）；job=jobs/job_N/{plan,doing,learning}（workspace.go:47-70 CreateJobStructure）；NextJobID 扫 max job_N（paths.go:188）。【0.9】
- doing/tasks.json：tasks[]{task_id,task_name,task_file,status(pending/running/success/failed/retrying),dependencies,attempts,error,output,commit_hash,created_at,updated_at}（tasks_json.go:18-44）；**rick-gates hook 是唯一 writer**（doing.go:442-447）；watcher 2s 轮询 diff→进度回调（doing.go:441 watchTasksJSONIn）。【0.9】
- doing 链路（handler/doing.go:85 DoingIn）：EnsureTasksJSON（扫 plan/task*.md 全 pending）→ensureGitRepo（非 git 仓先 init+首 commit，:263-295）→重试循环（cfg.MaxRetries）：SaveDoingPrompt（只重编 pending task）→rt.RunIn(工作区根,method,promptFile,cfg)→存 doing/session_id→runDoingGate→失败重试。resume=pi --session-id 交互续（doing.go:31-63）。Runtime 为接口 DIP（runtime.go:44-57），组合根注入。【0.9】
- super 可借：job 目录+NextJobID、tasks.json 单写者纪律+watcher 模式、python gate 先例（helper.py）、session_id 持久化、orchestration 的 workflowScript 编排渲染、Runtime DIP。【0.85】

## §8 对 super 模式的落地建议
1. 命令：internal/cmd/super.go 照 easy.go 模式（NewSuperCmd+AddCommand 一行+组合根注入），handler 侧 XxxCore(rickDir) 显式路径规约。【0.9】
2. py 调度器进程监督：supervisor.go 原语语义（Setpgid/kill 梯子/OnDead 游标/stderrTail/探活）+21 个测试模式直接照搬，但 Spawn 耦合 pi RPC——抽通用 proc 原语层或新写 PySupervisor（复用 byteRing/kill ladder 语义）；py 侧需设计等价 get_state 心跳协议（无现成先例）。【0.85】
3. py→pi 派发（J4）：最薄路径=subprocess 直调 `pi --mode json`（realpi 测试已证形态可行），必设 PI_CODING_AGENT_DIR 继承托管配置；长会话池用 --mode rpc。【0.85】
4. prompt：复用 builder BuildAndSave 预渲染→文件→--append-system-prompt 链路（Go 侧预渲染或 py 侧自渲染均可，模板可 go:embed 或 git-tracked 文件两种先例）。【0.85】
5. web 集成（J5）：DoingRunner 型注入缝最自然；CLI 侧可仿 --daemon re-exec+Setsid 先例。【0.85】
6. 资产/测试：py 调度器脚本照 rick-gates helper.py 先例 git 入库+python3 直调；agent 池测试照 mock_agent.py（MOCK_SCENARIO）+realpi tag 门控模式。【0.85】

## R7 上报项
- pi CLI 的完整 flag 集与 --mode json/rpc 输出 schema 以 pi 上游为准——本仓只见 rick 使用子集，未核实 pi 侧文档（缺 pi 上游信源）。
- py 调度器↔Go 监督树的心跳协议（get_state 等价物）无仓库先例，需设计（缺设计裁决，影响 J4/KR1）。
- leaf-2 已补交付（超时线后落盘）：核心新信息已并入 §2（OnDead 生产 nil、recovery.go 游标快照先例、fake pi 脚本用 RICK_PI_AGENT_DIR）。
- pi 子代理 worktree 隔离在 pi 侧如何实现未核实（rick 侧无代码，doing 靠写域互斥；缺 pi 上游信源）。

## 叶子文件
- research-L1b-leaf-1.md（§1+§5）· leaf-2.md（§2）· leaf-3.md（§3+§7）· leaf-4.md（§4+§6）——均与本 agent 直读交叉验证；同目录 design-tree.md（父任务设计树）
