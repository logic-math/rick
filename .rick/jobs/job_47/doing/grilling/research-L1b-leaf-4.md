# L1b-leaf-4：web 模式 + 资产/构建/测试（信源=代码直读）

## A. web 模式与进程启动先例
- cmd/web.go:1-11 web.go=组合根：internal/web 装配(registries/supervisor/hub/SessionManager/NewServer)注入 handler.Web；internal/web 依赖 handler，反向 import 即循环（rick-spec 例外三：RunE 唯一组装根）。
- cmd/web.go:65-103 `--daemon`：re-exec 自身(RICK_WEB_DAEMON=1)+Setsid+日志落 stateDir/web.log——CLI 后台化先例。
- cmd/web.go:215-237 webServeComposition：rt=NewPiRuntime(cfg.PiPath, cfg.PiExtraArgs...)；sup=NewSupervisor(resolveSupervisorConfig)；NewSessionManager(...,nil,nil)。
- cmd/web.go:293 resolveSupervisorConfig：IdleTimeout=禁用(会话持久持有)，MaxActive←config web_max_active，默认 64。
- web/sessions.go:1-12 会话执行心脏：sup.Spawn(pi --mode rpc 常驻 worker)+bootstrap prompt；resume 才重 spawn。
- sessions.go:52/55 DoingRunner/DreamRunner func 类型=web 后台执行桥（可注入 seam）。
- sessions.go:108-116 nil→默认 runner（cmd 传 nil 即用默认）。
- sessions.go:389-413 后台型(doing/后台 dream)不 spawn rpc worker→startBackground。
- sessions.go:444-455 交互型：SpawnSpec{SessionID,Dir,MethodFile,PromptFile,SessionIDFlag,CreateNew}→sup.Spawn；458 bootstrap RpcClient.Prompt→worker.Send。
- sessions.go:477-547 startBackground=server 进程内 goroutine 跑 runner，进度→hub.Publish SSE（非子进程）。
- routes.go:896 RegisterRoutes：Go1.22 method-pattern("POST /api/...")+authWrap。
- handler/web.go:1-16 Web()管 singleton/pid/token/信号；WebServeFunc 注入 serve。

## B. 资产/构建/测试
- go:embed 面：web/embed.go:26-31(SrcFS=all:src+public+根配置；DistFS=all:dist，dist 提交仓库，node 不进构建链)；prompt/manager.go:22-52(10 模板 string+skillsFS)；env/settings.go:24(themes/*.json)。
- prompt 机制：manager.go:65 NewPromptManager(dir 可选，空=embed)；77 LoadTemplate 缓存+dir 优先 embed 兜底；222 ReadEmbeddedSkill；234 LoadCoreSkills 拼接；281 WriteSkillFileWithVars={{k}} ReplaceAll 替换落盘；builder.go:39/107 Build/BuildAndSave（预渲染→写文件→交 pi 的 SpawnSpec.PromptFile 同款链路，py 调度器 prompt 预渲染可直接借鉴）。
- 模板资产：internal/prompt/templates/ 11 个 .md + skills/ 19 个 .md。
- Python：.rick/skills/rick-gates/helper.py(git-tracked,64 行)=确定性 doing 门禁(tasks.json 可解析/无 zombie/success 有 commit_hash)；调用=handler/doing.go:207→225-260 runDoingGate `exec.Command("python3",helper,doingDir)`，降级 ~/.rick/pi/agent/extensions/rick-gates/（init-pi 部署，env_test.go:461-533 验证）；.rick/skills/ 下多个 skill 自带 .py（git-tracked）；tests/mock_agent/mock_agent.py=MOCK_SCENARIO 环境变量驱动的假 agent(text/json/rpc 模式+故障场景)；tests/e2e_v2_test.py=4 阶段 e2e(plan/doing/learning/dream)。
- 构建：Makefile 仅 web-dist(npm ci+build)/web-dev(vite 代理 6137)；scripts/build.sh=go build -o bin/rick cmd/rick+验证（go≥1.21）。无 make test。
- go test 约定：单元=同目录 _test.go(internal/web 28 个 go 文件近半为测试)；cli_integration_test.go=进程内 cobra 执行(NewRootCmd+SetArgs+--help)，无 build tag；real-pi 冒烟=//go:build realpi(executor_realpi_smoke_test.go:1)+RICK_REAL_PI_ARGS env 门控，默认 go test 不触真实 pi；另有 tests/*.sh 集成脚本。

## 结论
1. super py 调度器接入 web=DoingRunner 同款注入 seam+新 session type。
2. 交互 spawn 走 sup.Spawn(pi 专用)；py 进程需新增 spawn 路径。
3. py 资产先例=rick-gates helper.py：python3 直调+git 入库+extensions 部署。
