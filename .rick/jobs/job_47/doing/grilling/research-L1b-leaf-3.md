# research-L1b-leaf-3：pi 派发机制 + workspace job 模型（信源=代码直读）

## A. pi 派发机制
1. rick=bootloader，pi（Node pi-coding-agent）以子进程执行（internal/runtime/runtime.go:13-15）。三个调用面：
2. 一次性 json 模式（doing/easy）：runtime.go:93-146 组装 `pi --mode json [PiExtraArgs] --append-system-prompt <methodFile> --append-system-prompt <promptFile> -p "开始：…"`（bootstrapMessage=cli.go:63）；cmd.Dir=工作区根、Env=AgentEnv()。输出=NDJSON camelCase 事件流：agent_settled=终止信号、tool_execution_start/end、sessionId 字段（executor.go:12-33,88-158）；就绪=sessionId≠空∧settled（runtime.go:186-190）。
3. 常驻 rpc 模式（web 会话）：supervisor.go:207 `--mode rpc` + `--session-id`(新建)/`--session`(续载)；stdin 写 JSONL 命令（prompt/steer/follow_up/new_session/switch_session/get_entries…，rpc.go:50-460），stdout 事件流；Setpgid 进程组+SIGTERM→grace(默认5s)→SIGKILL（supervisor.go:228,66）。
4. 交互/print 模式：cli.go:14-25（`pi -p`；pi 无权限弹窗，无需 skip-permissions flag）；flag 次序=cfg.PiExtraArgs（全局 --provider/--model/--api-key）+per-call（cli.go:90-95,121-131）。
5. pi 二进制解析序：cfg.PiPath→~/.rick/pi/agent/runtime/node_modules/.bin/pi→PATH（cli.go:36-50, agentdir.go:48-50）；每子进程注入 PI_CODING_AGENT_DIR（agentdir.go:68-70）。
6. 子代理派发：rick 不自起 worker——doing parent 会话的 prompt 内嵌 rick 生成好的 workflowScript，由 pi 的 subagent 工具执行：runs.all([{key,agent:'worker',timeoutMs,task}])（builder/orchestration.go:370-374 渲染；timeoutMs 按 task 工作量 20-90min 动态估算）；监控={action:"status",view:"transcript",id}+.pi/subagents/artifacts/<runId>*_meta.json；steer/stop 干预；resume 与 agent 互斥（templates/doing.md:17-30）。
7. maxSubagentDepth=2 是 pi 侧封底，rick 仅注释引用（internal/env/agents.go:44）；rick 自定义 pi agent（think/research/exporter，fullTools 含 subagent，agents.go:32）注册于 ~/.rick/pi/agent/agents/{name}.md（agents.go:70+）。
8. worktree 隔离：doing **不用**——worker 共仓，靠 plan `# 写域` 声明+rick 确定性两两互斥校验（orchestration.go:16-80），worker 不碰 git，层检查点 gate{N}.py→level_complete→单 commit。真 git worktree 仅 devweb dev 环境（internal/env/devweb/devweb.go:123-131 IsDevWorktree：.git 为文件）。
9. **py 侧派发途径（J4 裁决输入）**：①subprocess 直调 `pi --mode json -p --append-system-prompt <f> <bootstrap>` 解析 stdout NDJSON——现成可行（executor_realpi_smoke_test.go 有真 pi 用例，go:build realpi）；②`pi --mode rpc` 常驻协议（stdin 命令/stdout 事件，可 steer/get_entries 游标续传）；③复用 rick 托管配置：py 设 PI_CODING_AGENT_DIR=~/.rick/pi/agent 即继承 rick agents/扩展；路径解析序同第 5 条。

## B. workspace/job 目录模型
10. `.rick/{loops,skills,domain,jobs,dream,draft}`（workspace/paths.go:11-26）；job 目录 `jobs/job_N/{plan,doing,learning}`（workspace.go:47-70 CreateJobStructure）；NextJobID 扫 max job_N（paths.go:170-193）。
11. doing/tasks.json：tasks[]{task_id,task_name,status(pending/running/success/failed/retrying),dependencies,attempts,error,commit_hash…}（workspace/tasks_json.go:18-44）；**rick-gates hook 是唯一 writer**（handler/doing.go:443-447）；人工恢复先 running→pending 归一化（doing.go:528-590）。
12. doing 链路（handler/doing.go:66-200 DoingIn）：EnsureTasksJSON（扫 plan/task*.md 全 pending）→ensureGitRepo（非 git 仓先 init+首 commit，doing.go:263-295）→循环 cfg.MaxRetries(默认3)：SaveDoingPrompt（只重编 pending task）→tasks.json watcher（2s 轮询 diff→进度回调）→rt.RunIn(工作区根,method,promptFile,cfg)→存 doing/session_id→runDoingGate（python3 .rick/skills/rick-gates/helper.py <doingDir>；缺则 ~/.rick/pi/agent/extensions/rick-gates/helper.py；doing.go:225-260）→失败重试。resume：pi --session-id 交互续（doing.go:31-63）。
13. super 可借：job 目录模型+NextJobID；tasks.json 单写者纪律+watcher 模式；python gate 脚本先例（helper.py/gate{N}.py）；workflowScript 编排渲染（orchestration.go）；session_id 持久化；Runtime 接口 DIP（runtime.go:44-57，cmd/doing.go:88-103 组合根注入）。

## 结论
1. py 直调 pi CLI 可行：json 一次性 + rpc 常驻两条现成路。
2. rick 不自管 worker 池：派发=pi subagent，depth=2 pi 侧封底。
3. doing 无 worktree 隔离，靠写域互斥；worktree 仅 devweb 先例。
