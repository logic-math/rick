# rick web ui 流水线执行前全链路预演报告（sim-report）—— 第 1 棵设计树（初版 Web UI，已交付）

> 本节是 **job_36 初版 Web UI 流水线（task1-15 / gate1-6）** 的预演报告，保留原文不动；
> 本增量的「rick 自进化」验收报告见文件末尾的第二节。


> 预演视角：import/编译顺序/文件系统/进程/网络/npm/git 语义。输入=task1-15 + gate1-6 + api-contract + design-tree + 真实代码库实证。
> 方法：逐 task 逐文件核对 import/写域/并行竞争 + gate 逐行审查 + 关键断言本机实证（go build/test 全绿基线、pgrep 自匹配、代理拦截、SSE read 阻塞、embed 多 directive、gitignore 行为均已跑通验证）。
> 统计：**阻塞 7 / 高危 6 / 中 8 / 低 6**。基线健康：go build ./... ✓、go vet ✓、go test ./... 全绿 ✓、node v24.16.0 + npm 11.13.0 + registry PONG ✓、pi rpc.md 存在 ✓。

## 一、卡点清单

| # | 层/task | 卡点（计算机视角） | 严重度 | 证据 | 预解方案 |
|---|---------|-------------------|--------|------|----------|
| B1 | gate2/3/4 | `bash -c "pgrep -f 'mode rpc'"` 中 bash 自身 cmdline 含 pattern → pgrep 永远匹配到包裹 bash → 「残留 pi rpc 进程」无条件假红（实证：任意不可能 pattern 也返回 pid） | 阻塞 | gate2.py:56 / gate3.py:51 / gate4.py:47；本机实证 pids 2350091/2350095 | pattern 改 `'[m]ode rpc'`（方括号技巧，实证输出为空） |
| B2 | gate6(L6) | 本机 `http_proxy/https_proxy=http://10.229.18.27:8412` 且无 no_proxy → urllib.request 全部 127.0.0.1 请求走公司代理 → 503（实证：urlopen 18999 端口返回 HTTP 503） | 阻塞 | env 输出；gate6.py `http()` 全部经 urllib.request.urlopen | gate6 顶部建 `opener = urllib.request.build_opener(urllib.request.ProxyHandler({}))`，所有 urlopen→opener.open（全局 opener 有缓存，改 env 不保险） |
| B3 | gate6(L6) | SSE 检查 `resp.read(4096)`：流式响应不足 4096 字节 → 阻塞至 5s socket 超时抛 TimeoutError → 被 except 捕获报「SSE 连接失败」（实证：代理旁路后 read(4096) 5.0s 超时） | 阻塞 | gate6.py SSE 段；本机 python3 复现 TimeoutError | 改行读+截止时间循环（见修订清单 G6-3） |
| B4 | L1/task3 + level_complete | `git add -A` 在 gate1 绿后提交：web/node_modules 未被任何 .gitignore 覆盖（根 .gitignore 仅 6 行无 node_modules；task3 KR 未建 web/.gitignore）→ 数百 MB/数万文件入库 | 阻塞 | 根 .gitignore 内容；`git check-ignore web/node_modules` exit=1；task3 KR1-9 无 .gitignore 条目 | task3 新增 KR：建 `web/.gitignore`（内容 `node_modules/`，实证有效）；dist 保持入库（rick 哲学） |
| B5 | L1/task2 | KR4 要求 `runtime.Runtime` 接口加 `RunIn(dir,...)` + piRuntime 实现（cmd.Dir=dir），但 Runtime 接口与 piRuntime.Run 均定义于 internal/runtime/runtime.go:44/78——不在 task2 写域（internal/handler/ + 6 个 cmd 文件）→ 越界或卡死 | 阻塞 | runtime.go:44-46/78；task2 写域声明 | task2 写域 += `internal/runtime/runtime.go`。与同层 task1（rpc.go/rpc_test.go）同包不同文件，无文件冲突；全仓仅 piRuntime 一个实现（grep 证实无 mock 需同步改） |
| B6 | L5/task14 | J2.4/M8 裁决 token 存 `~/.rick/config.json` 的 `web_token` 字段，但 Config struct（internal/config/config.go）无该字段；task14 KR2③「flag>config>自动生成写回」必须改 Config——config.go 不在 task14 写域 | 阻塞 | config.go struct 定义；design-tree J2.4/M8；task14 写域 | task14 写域 += `internal/config/config.go`（仅加 `WebToken string` 字段；LoadConfig/SaveConfig json 往返自动兼容，无需改 loader.go） |
| B7 | L4/task13+L5/task14 | /api/config 需输出 rick_version，但 VERSION 常量在 package main（cmd/rick/main.go:10 `const VERSION="4.4.15"`）——internal/web、internal/cmd 均不可 import main；task13 Deps 结构也未列 Version 字段 | 阻塞 | cmd/rick/main.go:10；task13 KR1 Deps；契约 Server 节 | task14 KR3 改 `NewWebCmd(version string)`（root.go 在 task14 写域内，NewRootCmd(version) 已持有该参数）；task13 Deps += `Version string` 传入 /api/config |

| # | 层/task | 卡点（计算机视角） | 严重度 | 证据 | 预解方案 |
|---|---------|-------------------|--------|------|----------|
| H1 | L1/task3 | Tailwind 4 无法按 KR1 devDeps 列表接入：vite 集成必须 `@tailwindcss/vite` 插件（或 @tailwindcss/postcss），devDeps 只列了 tailwindcss@4 → `@import "tailwindcss"` 无法解析/vite.config import 失败 → npm run build 红；KR4 的 `@theme` 指令也只有 Tailwind 处理 CSS 才生效 | 高危 | task3 KR1 devDeps 清单；Tailwind v4 官方接入方式 | task3 KR1 devDeps += `@tailwindcss/vite`；KR2 vite.config.ts plugins += `tailwindcss()` |
| H2 | L2/task4→L3/task11 | task11 发 bootstrap prompt 需要 `bootstrapMessage` 常量——internal/runtime/cli.go:63 定义为**未导出** const，包外（internal/web）不可引用 → 编译缺口 | 高危 | cli.go:63 `const bootstrapMessage = "开始：…"`；task11 KR1「rick 常量 bootstrapMessage」 | task4 supervisor.go（新文件、写域内）加导出别名：`// BootstrapMessage re-exports the CLI bootstrap trigger for web sessions.\nvar BootstrapMessage = bootstrapMessage`（同包引用，零重复） |
| H3 | L1/task2 | handler core 化若保留 cwd 依赖的 workspace 辅助函数（NextJobID/GetJobPlanDir/GetDraftDir/GetJobsDir——paths.go:58/76/154/188 均无 rickDir 参数；handler 内调用 2+2+2+0 处）→ web 从服务端 cwd=A 调 planCore(rickDir=B) 时扫 A 的 jobs → job id 冲突/产物落错工作区（编译期无错、task11 期功能性错误） | 高危 | paths.go 函数签名；`grep workspace\.` handler 计数（GetRickDir16/SelectPendingJobs2/NextJobID2/LoadDebugContext2/GetJobPlanDir2/GetDraftDir2/NextLoopID1/New1/LoadTasksJSON1） | task2 KR1 补硬约束：core 内路径派生一律 `filepath.Join(rickDir,...)`；NextJobID/GetJobPlanDir/GetDraftDir/GetJobsDir 的 cwd 版禁入 core——handler 内自实现 `nextJobIDIn(rickDir)`（写域内，DAG 不变） |
| H4 | L4/task13 | watcher 监听 `~/.rick/web/dist`——首次启动（gate6 的 HOME）该目录**不存在**；fsnotify watch 不存在目录返回错误，若 watcher 启动路径把该错误当 fatal → 服务启动即崩 → gate6 健康检查假红 | 高危 | task13 KR3「fsnotify 监听 ~/.rick/web/dist」无缺失目录处理条款；gate6 起服时 HOME 无 web/ 目录（实证 ~/.rick/web.json 不存在） | task13 KR3 补一句：「监听目录不存在→跳过并轮询等待目录出现（2s mtime 兜底循环内重试），不得 fatal」 |
| H5 | L3/task9+task10 | 同层两前端 task 各自 KR 尾都要求 `npm run build` 通过——两个 vite build 并发清空/写同一 web/dist → 竞争：一方 emptyOutDir 删掉另一方刚写的 assets → worker 自验假红/假绿 | 高危 | task9 KR8 / task10 KR7；vite build 默认 emptyOutDir=true | task9/task10 验证条款改为仅 `npx tsc --noEmit`（build 统一留给层后 gate3 跑）；另 task9 KR6 补「ExtensionUIDialog 自包含（不 import 同层 task10 的 components/common/，避免中间态缺模块 tsc 红）」 |
| H6 | L1/task3→L5/task14 | embed 范围缺根配置文件：KR7 仅 `//go:embed all:dist` + `all:src`——package.json/vite.config.ts/tsconfig.json/index.html 在 web/ 根**不在 src/** → task14 DeployWebScaffold 从 SrcFS 取不到根文件（customize 链路断），task14 写域不能改 web/embed.go 只能回执上报、task15 收口——链路晚两层才通 | 高危 | task3 KR7 embed 模式；task14 KR1「SrcFS 含这些根文件…若不含则回执上报」 | task3 KR7 修订为多 directive 累积（Go 官方支持，本机实证编译+读取通过）：`//go:embed all:src` + `//go:embed package.json vite.config.ts tsconfig.json index.html` 同一 SrcFS 变量。task12 建的 web/public/ 不能预埋（task3 时不存在，embed 无匹配=编译错）——task15 终检补 `all:public` |

| # | 层/task | 卡点（计算机视角） | 严重度 | 证据 | 预解方案 |
|---|---------|-------------------|--------|------|----------|
| M1 | L3/task11 | SessionEntries 离线读路径写死 `~/.rick/pi/agent/sessions/**`——不尊重 RICK_PI_AGENT_DIR → 测试隔离失效（KR4 声明了双隔离但硬编码路径读真实 HOME）且与 AgentEnv 的隔离语义不一致 | 中 | task11 KR1 路径字面量；agentdir.go:27 AgentDir() 尊重 env | task11 KR1 明确「sessions 根经 `runtime.AgentDir()` 解析（尊重 RICK_PI_AGENT_DIR），再全树按 uuid 前缀枚举」 |
| M2 | L4/task13 | 若 worker 顺手 `go get fsnotify` → 改 go.mod/go.sum——两文件不在任何 task 写域（越界）且引入网络依赖 | 中 | go.mod 仅 cobra/goldmark；task13 KR3「倾向轻量自实现轮询」非硬约束 | task13 KR3 补硬约束：「v1 禁止引入 fsnotify/任何新依赖（不改 go.mod），用自实现轮询」 |
| M3 | L5/gate5 | gate5 测试清单 `./internal/cmd/ ./internal/env/ ./internal/web/...` 缺 `./internal/handler/...`——task14 写了 internal/handler/web.go，若破坏现有 handler 测试要拖到 gate6 才发现 | 中 | gate5.py 测试循环列表；task14 KR5 同样缺 handler | gate5.py 测试列表 += `"./internal/handler/..."`；task14 KR5 同步补 |
| M4 | L6/gate6 | E2E 无 HOME 隔离：POST /api/workspaces 把 ROOT 注册进真实 `~/.rick/web.json`（机器级状态污染；且 gate6 的 pid 残留检查用真实 HOME——与服务端一致但同受污染） | 中 | gate6 Popen 无 env；task5 注册表跟随 HOME | gate6 Popen 加 `env={**os.environ, "HOME": tmpdir, "no_proxy": "127.0.0.1,localhost"}`；pid 检查路径同步改为 `os.path.join(tmpdir, ".rick", "web.pid")`（LoadConfig 在空 HOME 自动建默认 config，实证无碍） |
| M5 | L1/task5 | 契约幂等语义：POST /api/workspaces 同 path 已注册→200、新建→201——task5 `Add(path,name)` 只返回 entry 无 created 信号 → task13 无法区分 200/201（假 201） | 中 | api-contract Workspaces 节；task5 KR2 Add 签名 | task5 KR2 Add 签名改 `Add(path, name) (WorkspaceEntry, bool, error)`（bool=created）；registry_test 补断言 |
| M6 | L1/task5+L4/task13 | 契约 GET /api/workspaces 响应含 `jobs_count`——WorkspaceEntry 无该字段，task13 需 join ListJobs 计算，spec 未声明该职责 | 中 | api-contract Workspaces 节；task5 KR2 结构 | task13 KR1 补：「GET /api/workspaces 响应由 routes 层 join Jobs.ListJobs 补 jobs_count（读取失败计 -1 或省略字段）」 |
| M7 | parent/契约 | api-contract.md 缺三端点：/api/health、/api/web/customize、/api/web/reset——task12/13 以「契约补充项」双边声明，单源名存实亡；gate4 已按字符串检查 | 中 | api-contract.md 全文；task12 上下文提示；gate4.py 端点列表 | parent 派发前（L1 前）更新 api-contract.md 增补三端点定义（SSE 节说明 health 无鉴权；customize/reset 请求/响应体） |
| M8 | L3/task11 | spec 引用的 builder 函数名失准：SavePlanPrompt/SaveEasyPrompt/SaveDreamPrompt 实际不存在——真名 prompt.GeneratePlanPromptFile/GenerateEasyPromptFile/GenerateCtrlPromptFile/GenerateDreamPromptFile | 中 | internal/prompt/*_prompt.go 导出函数清单 | task11 KR1 引用名更正（worker 读代码可自查，但 spec 应准确防误建同名包装） |

**低危（简列）**：L1 task4 Spawn 伪代码 `exec.Command(piBin,"--mode","rpc",extraArgs...,"--append...")` 中 `...` 后跟参数为非法 Go 语法——worker 自会改建 args slice，建议 spec 措辞改「args 切片拼接」；L2 task11「已 closed 200」与契约 close→202 不一致（统一 202）；L3 契约 knowledge 扩展名缺 .ts（task6 含 .ts——统一为含）；L4 gate1 GetRickDir 统计 grep 含 *_test.go（当前测试文件 0 处，无碍，知悉即可）；L5 同层 go build 并发（task1/task2 同包不同文件）偶发中间态红——建议 worker 指引「build 失败先单次重试再判红」；L6 task15 `git add -f bin/rick`——bin/rick 已 tracked，普通 add 即可（-f 无害保留）；L1 层 commit 会夹带当前 untracked 的 .rick/draft/loop_7、rfc 草稿与 job_36 plan 资产（git add -A 语义，预期内，知悉）。

## 二、假绿假红风险（逐 gate 判别力审查）

**gate1（L1）**：判别力良好。go build 先行兜底编译；四 task 各有专属测试/文件存在断言；npm run build 需 node_modules（task3 worker 失装则红=正确信号）。GetRickDir≤7 阈值=重构后 7 个 CLI 包装入口各 1 次，紧贴设计值——判别力强（第 8 处即红）但 worker 若在 handler 新增合法调用会假红，属预期收紧。假绿盲区：task2 的 RunIn 行为无专门断言（仅编译级覆盖）——可接受（KR5 现有测试全绿约束行为零变化）。**修 B4 后本 gate 无假红项**。

**gate2（L2）**：pgrep 检查无条件假红（B1）——修后判别力良好。TestRpc/TestSupervisor/TestJobs/TestSSE/TestAuth 分包定向跑符合 testing-conventions（禁全量）；tsc+build 覆盖 task7 类型层。假绿盲区：task7 stores 无单测（spec 声明可选）——tsc 类型自洽为下限，设计内。

**gate3（L3）**：pgrep 同 B1。task9/10 仅文件存在+tsc+build（组件无自动化测试——设计内，手工 smoke 声明）；task11 TestSession 是本层最重断言（fake pi 集成）。修 B1 后无假红项。

**gate4（L4）**：pgrep 同 B1。routes.go 端点字符串检查可被「注释写字符串」绕过（理论假绿）——但 task13 KR5 server_test 全路由 smoke httptest 弥补，实际判别力够。PWA manifest 检查（dist 列表含 manifest）有效。internal/web 全量测试 600s 含合流回归 ✓。

**gate5（L5）**：判别力良好：三包测试+二进制构建+`rick web --help` 断言 --port/--listen+两子命令 help。缺 handler 包测试（M3）。`--help` returncode=0（cobra 语义）✓。构建 bin/rick 使 L5 commit 含二进制 diff（tracked 文件，符合仓库惯例）。

**gate6（L6）**：**两处无条件假红（B2 代理 + B3 SSE read）——修复前本 gate 必红，pipeline 在 L6 死锁**。修后判别力强：401/200 矩阵、SPA fallback、真实 jobs 数据断言（本仓 job_1..19 均有 doing/tasks.json，实证「job_」必命中）、pid 清理、优雅退出。jobs 断言依赖仓库已有 jobs——本机满足；纯净 checkout 场景理论假红（可接受，开发机语义）。HOME 副作用见 M4。

## 三、无需改动的确认项（预演通过）

1. 六 gate ROOT 五级上溯推导一致正确（gates/→plan→job_36→jobs→.rick→rick 根）。
2. LoadConfig 在 config.json 缺失时自动创建默认配置（loader.go:71-85）→ gate6 E2E 空服务可启动，无 config 依赖卡点（实证逻辑）。
3. 基线全绿：go build ./... ✓ go vet ✓ go test ./... 全绿 ✓（全量跑通 EXIT=0）；realpi 测试有 build tag 隔离不会误入 gate。
4. npm registry 可达（PONG 614ms）、node v24.16.0/npm 11.13.0——task3 首装与 make web-dist 的 npm ci 可行；gate 的 npm build 600s 预算充足。
5. task1 输入依赖 pi rpc.md 存在（~/.rick/pi/agent/runtime/node_modules/@earendil-works/pi-coding-agent/docs/rpc.md，41KB）。
6. AgentEnv() 已导出（agentdir.go:68）；AgentDir() 尊重 RICK_PI_AGENT_DIR——task4 spawn env 与 task11 测试隔离可用现成设施。
7. Runtime 接口唯一实现 piRuntime、零 mock 实现（grep 证实）→ B5 写域扩展爆破半径=runtime.go 单文件。
8. pi sessions 实际布局 `sessions/<encoded-workdir>/<ts>_<uuid>.jsonl`（实证目录树）与 task11「全树按 uuid 前缀枚举」策略吻合。
9. .rick/jobs/job_1..job_19 均有 doing/tasks.json、job_36 亦有 → gate6 jobs 断言可满足。
10. bin/rick 已 tracked（git ls-files 证实）→ git add -A 会带上 gate5/6 重建二进制；task15 的 -f 兜底与 bugs.md job_35 坑位一致。
11. embed 语义实证：`//go:embed all:dist`+多 directive 根文件累积同一 FS 编译且可读（本机 /tmp 验证）；空 dist 占位（task3 dist/index.html）防「no matching files」编译错；node_modules 在 web/ 根不在 web/src → all:src 不吞入。
12. 写域互斥逐层核对：L1（task1 rpc*.go / task2 handler+cmd6+[B5 扩展 runtime.go] / task3 web+Makefile / task5 internal/web 基础三文件）、L2/L3/L4 各 task 文件集合两两不相交（task12 与 task13 零交集）；跨层复写文件（vite.config.ts、App.tsx/main.tsx：task3 建→task12 改）均层序串行。
13. DAG 依赖闭合：task11←{task4,task2,task8}、task13←{task11,task6,task8,task3}、task14←task13、task15←{task12,task14}——全部前层就绪；task6→task2 依赖实为松依赖（jobs 读层不消费 core 产物），无害。
14. handler 现有签名（Plan/Easy/Ctrl/Learning/Dream/HumanLoop/Doing）与 task2 core 化改造形态吻合；watchTasksJSON(watchDone, doingDir) 已路径参数化；SelectPendingJobs(rickDir,jobNum)/LoadTasksJSON(path)/NextLoopID(draftDir) 本就路径参数化——重构阻力小于预期。
15. fake pi 双坑位（PATH 恢复 job_33 / RICK_PI_AGENT_DIR 隔离 job_34）与 RFC3339 时区坑（job_35）已在 task4/task5/task11 spec 明文声明。
16. 契约字段对齐：SessionEntry↔SessionInfo ✓、SSE Envelope json tag ↔ 契约 envelope ✓、entries {entries,leaf_id} ✓、错误体 {error:{code,message}} ✓、鉴权三通道（Bearer/query/静态豁免）✓、幂等/409/404 语义 ✓（200/201 区分见 M5）。

## 四、修订清单（可直接执行；均保持 DAG 依赖与写域互斥不变——B5/B6 为写域扩充，扩充文件不与任何同层 task 冲突）

### task2.md
- **写域**追加一行：`internal/runtime/runtime.go`（B5：Runtime 接口 RunIn + piRuntime cmd.Dir；同层 task1 写 rpc*.go 同包不同文件无冲突）。
- KR1 末追加：「core 内路径派生一律 `filepath.Join(rickDir, ...)`；workspace.NextJobID/GetJobPlanDir/GetDraftDir/GetJobsDir 等 cwd 依赖函数禁入 core——在 handler 内自实现 `nextJobIDIn(rickDir)`（扫 `<rickDir>/jobs` 取最大 job_N+1）；SelectPendingJobs(rickDir)/LoadTasksJSON(path)/NextLoopID(draftDir) 已路径参数化可直接复用」（H3）。
- 上下文提示「24 处调用」更正为「16 处」（实测 grep 计数，防 worker 迷惑）。

### task3.md
- KR1 devDeps 追加：`@tailwindcss/vite`（H1）；KR2 vite.config.ts plugins 明确 `react() + tailwindcss()`。
- KR7 修订 embed 模式（H6，多 directive 累积，已本机实证）：
  ```go
  //go:embed all:src
  //go:embed package.json vite.config.ts tsconfig.json index.html
  var SrcFS embed.FS
  //go:embed all:dist
  var DistFS embed.FS
  ```
- **新增 KR10**（B4）：「`web/.gitignore` 内容 `node_modules/`（node_modules 绝不入库；dist 与 package-lock.json 正常入库供 npm ci）」；验证条款追加 `git check-ignore web/node_modules` 命中。
- KR8 措辞「Makefile 追加」→「Makefile 新建（仓库现无 Makefile），含 web-dist（cd web && npm ci && npm run build）与 web-dev 目标」。

### task4.md
- KR1 末追加一行（H2）：「导出别名 `var BootstrapMessage = bootstrapMessage`（cli.go 的启动触发消息，供 task11 web 会话 bootstrap）」。
- KR1 Spawn 措辞：exec.Command 伪代码改为「args 切片拼接（Go 不允许 ... 后跟参数）」（低危 L1）。

### task5.md
- KR2 Add 签名改 `Add(path, name) (WorkspaceEntry, bool, error)`，bool=是否新建（M5：task13 据此区分 201/200）；registry_test 补幂等返回 false 断言。

### task9.md
- KR6 追加：「ExtensionUIDialog 自包含——不 import 同层 task10 的 components/common/（避免并行中间态缺模块 tsc 红）」（H5）。
- KR8 验证改：「`npx tsc --noEmit` 通过（npm run build 留给 gate3 层完成后统一执行——同层 task10 并行 build 会踩 dist）」（H5）。

### task10.md
- KR7 验证改：同上（仅 tsc，build 留给 gate3）（H5）。

### task11.md
- KR1 SessionEntries 修订（M1）：「sessions 根经 `runtime.AgentDir()` 解析（尊重 RICK_PI_AGENT_DIR），全树按 pi_session_id 文件名前缀枚举」。
- KR1 引用名更正（M8）：SavePlanPrompt→`prompt.GeneratePlanPromptFile`、SaveEasyPrompt→`prompt.GenerateEasyPromptFile`、GenerateCtrlPromptFile ✓、SaveDreamPrompt→`prompt.GenerateDreamPromptFile`、bootstrap prompt→`runtime.BootstrapMessage`（task4 导出）。
- KR 命令面：「SessionClose 幂等（已 closed 200）」→「已 closed 仍 202」（低危 L2，对齐契约）。

### task13.md
- KR1 Deps 补 `Version string`（B7：/api/config 的 rick_version 来源，由 task14 的 NewWebCmd(version) 注入）。
- KR1 补（M6）：「GET /api/workspaces 响应由 routes 层 join ListJobs 补 jobs_count」。
- KR3 追加（H4+M2）：「监听目录不存在→跳过+轮询等待出现，不得 fatal；v1 禁止引入 fsnotify 等新依赖（不改 go.mod），watcher 自实现轮询（2s mtime）+可选 fsnotify 改为『仅标准库』」。

### task14.md
- **写域**追加一行：`internal/config/config.go`（B6：Config 加 `WebToken string \`json:"web_token,omitempty"\``；loader.go 无需动）。
- KR3 修订（B7）：`NewWebCmd(version string)`（root.go 调用点 `rootCmd.AddCommand(NewWebCmd(version))`——NewRootCmd 已持有 version 参数）；KR2 WebOptions/组装链补 version 透传至 ServerConfig/Deps。
- KR5 测试命令补 `./internal/handler/...`（M3）。

### gate2.py / gate3.py / gate4.py（同一修订，B1）
```python
r = run(["bash", "-c", "pgrep -f '[m]ode rpc' | head -5"])
```
（方括号技巧：包裹 bash 的 cmdline 含 `[m]ode rpc` 不被正则 `[m]ode rpc` 匹配——本机实证输出为空；真实 `--mode rpc` 进程仍会被捕获。）

### gate5.py（M3）
- 测试循环列表 `["./internal/cmd/", "./internal/env/", "./internal/web/..."]` → 追加 `"./internal/handler/..."`。

### gate6.py（B2+B3+M4）
1. 顶部（import 后）追加（B2）：
   ```python
   opener = urllib.request.build_opener(urllib.request.ProxyHandler({}))
   ```
   全文 `urllib.request.urlopen(` → `opener.open(`（http() 与 SSE/POST 段共 5 处）——本机 http_proxy 已实证拦截 127.0.0.1（503）。
2. SSE 段替换为行读+截止（B3）：
   ```python
   try:
       req = urllib.request.Request(f"{base}/api/events?token=e2etest-token")
       with opener.open(req, timeout=5) as resp:
           deadline = time.time() + 3; data = ""
           while time.time() < deadline and "server_info" not in data:
               line = resp.fp.readline()
               if not line: break
               data += line.decode("utf-8", "replace")
           if "server_info" not in data:
               errors.append(f"SSE 未收到 server_info: {data[:200]}")
   except Exception as e:
       errors.append(f"SSE 连接失败: {e}")
   ```
3. Popen 加 env 隔离（M4）：`env={**os.environ, "HOME": e2e_home}`（e2e_home=tempfile.mkdtemp()）；末尾 pid 检查改 `os.path.join(e2e_home, ".rick", "web.pid")`。（可选简化：不隔离则接受 ~/.rick/web.json 被写入 repo 条目——幂等无害，但机器状态被 gate 污染。）

### parent 派发前动作（M7，非 task 修订）
- 更新 plan/api-contract.md：增补 `GET /api/health → 200 {"status":"ok"}`（无鉴权）、`POST /api/web/customize → {ok:true,scaffolded:bool}`、`POST /api/web/reset → {ok:true}` 三端点（gate4 字符串断言与 task12/13 双边声明已有，仅契约文件补齐单源）。

### 修订影响面自检
- 写域扩充仅 B5（task2+=runtime.go）/B6（task14+=config.go）两处，扩充文件均无同层他 task 写入 → 互斥保持；DAG 零变化。
- 所有 gate 修订不改变绿判据语义，仅修假红（pgrep/代理/SSE read）与补漏（gate5 handler）。
- task9/10 验证收缩为 tsc-only 不削弱层判别力（gate3 仍统一跑 build+tsc）。

---

# 仿真/验收报告 —— rick 自进化增量（job_36 收尾功能）

- 范围：设计树第 2 棵（`KR1 隔离实例` / `KR2 开发闭环` / `KR3 受控提升` / `KR4 挂起可恢复`）
- 交付物：`rick tools dev-web`、`rick tools release`、`--state-dir` + flock singleton + 隔离守卫、
  build_id 指纹、`suspended` 语义 + `/continue` + `/api/recovery`、前端挂起 UI、本手册与 E2E 脚本
- 工作树纪律：全部改动落在 **dev 工作树** `/workdir/sunquan20/rick-dev`；生产工作树零写入
- 验收环境：Linux 云 IDE，Go 1.25，node ≥22，生产实例 8413 在线（`pid 172761`，`bin/rick` md5 `643b54a3…`）
- 复现命令：`bash scripts/self-evolve-e2e.sh`（末行单行 JSON 结论）

## 0. 生产只读基线（E2E 首尾各测一次）

| 项 | 值 | 结论 |
|---|---|---|
| `~/.rick/web/sessions.json` md5 | 单次运行内首尾一致（如 `45e5b780…`） | ✅ |
| `~/.rick/web.json` md5 | 单次运行内首尾一致（如 `46b78810…`） | ✅ |
| `http://127.0.0.1:8413/api/health` | `200 {"status":"ok",…}` | 提升/回滚/重启全程在线 ✅ |

> ⚠️ **读数说明**：生产是**活的**——本次验收的 AI 会话本身就托管在生产实例上，每次工具调用都会触发
> busy/status 等注册表写入（实测 `sessions.json` mtime 与「现在」相差 3 秒）。因此 md5 **不能跨运行比较**，
> 断言的含义是「**同一次 E2E/gate10 运行内首尾一致**」，以及「8413 全程健康」。E2E 自身的所有状态都写在
> `$TMP` 的临时 HOME/端口里，对 `~/.rick` 只有 `md5sum` 与 `/api/health` 两种只读访问。

> 说明：E2E 的所有 release/rollback 都针对 **$TMP 下的模拟生产**（临时 repo + 临时 home + 临时端口 +
> 自带 `start-web.sh`）；对真实生产只做 `md5sum` 与 `/api/health` 的只读断言。

## 1. KR1 隔离实例 —— 证据与结论

| # | 断言 | 证据（实测输出） | 结论 |
|---|---|---|---|
| 1.1 | dev 实例用独立 HOME/端口起来且指纹自洽 | `DEV_UP bin=…/devhome/bin/rick.dev.0a47372-260921195122 pid=504924 health_ms=201 build_id=0a47372-260921195122 port=46357` | ✅ |
| 1.2 | 状态隔离：dev 看不到生产会话 | dev `/api/sessions` → `[]`（生产 8413 有 6 条，md5 `45e5b780…`） | ✅ |
| 1.3 | flock singleton：同 state-dir 第二实例被拒 | 第二实例退出码非 0，输出含「另一个 rick web 正在使用该状态目录 / lock」 | ✅ |
| 1.4 | dev 语义实例拒绝注册生产工作区 | `POST /api/workspaces {path:/workdir/sunquan20/BERT_KEETA}` → **HTTP 400** | ✅ |
| 1.5 | `dev-web down` 只杀经归属断言的 dev 进程 | 生产 8413 进程全程未受影响（health 200 + md5 不变） | ✅ |
| 1.6 | 生产 singleton 缺陷已修（原先 `~/.rick/web.pid` 缺失即可同 HOME 起第二实例并误杀会话） | flock 由内核保证，pid 文件可删不再构成绕过路径 | ✅（gate7） |

**残留缺口（见 §5 finding F1）**：守卫只在 `--state-dir ≠ $HOME/.rick` 时安装；`dev-web` 的实际形态
（HOME 换掉、state-dir == `$HOME/.rick`）下守卫未安装 → 该形态可注册生产工作区（E2E 记为 FINDING，非通过项）。

## 2. KR2 开发闭环 —— 证据与结论

| # | 断言 | 证据 | 结论 |
|---|---|---|---|
| 2.1 | 前端 overlay 热更即时生效（**不重启进程**） | 两次替换 `<dev-state>/web/dist/index.html` → `GET /` 分别返回 `e2e-overlay-A`/`e2e-overlay-B`；`kill -0 <dev pid>` 仍存活 | ✅ |
| 2.2 | 后端源码改动经 `restart` **行为可见** | 探针：`cmd/rick/main.go` 的 `VERSION` → `4.4.15+e2e` → `dev-web restart` → `/api/config` 返回 `rick_version=4.4.15+e2e` | ✅ |
| 2.3 | 重启后构建指纹变化且被复核 | `build_id 0a47372-260921195122 → 0a47372-260921195126`；`dev-web up` 自校验失败即报 `fingerprint` 错 | ✅ |
| 2.4 | 反馈回路可被 AI 消费（不依赖 SSE） | `DEV_UP`/`DEV_FAIL stage=…` 单行回执 + 退出码 `0/2/3/4` + `/api/health.build_id`（免认证） | ✅ |
| 2.5 | 自举：开发会话（生产托管、workspace=dev 树）不受 dev 重启影响 | 本增量实施全程即为此形态：dev 侧反复 build/restart/down，会话未断（本次验收过程中 dev 实例被重启 ≥8 次） | ✅（实测观察） |

## 3. KR3 受控提升 —— 证据与结论

| # | 断言 | 证据 | 结论 |
|---|---|---|---|
| 3.1 | 提升后运行的就是新构建 | `RELEASE_OK version=0a47372-260921195138` / `RELEASE_RESTART … matches=true`；随后 `/api/health` → `build_id=0a47372-260921195138`（与版本一致） | ✅ |
| 3.2 | 前端与二进制同版推进 | `<sim-state>/web/dist/index.html` 存在（旧覆盖层留 `dist.prev`） | ✅ |
| 3.3 | 版本历史 + 原子换链 | `<sim-repo>/bin/releases/<version>/{rick,dist}` + `current` 链；`bin/rick` → `releases/current/rick` | ✅（gate9 单测覆盖换链/GC/`--keep`） |
| 3.4 | 二次提升 → build_id 递进 | `0a47372-260921195138 → 0a47372-260921195139` | ✅ |
| 3.5 | 一键回滚 | `release --yes --rollback` → 重启后 `build_id=0a47372-260921195138`（回到上一版）且健康 | ✅ |
| 3.6 | 失败自动回滚 | gate9 单测：门禁失败不动生产；健康/指纹校验失败 → 自动回滚到 `.last` | ✅（单测） |
| 3.7 | 提升不误杀无关进程 | `stopProd` 带归属断言（`cwd`/`exe` 前缀/`HOME` 三选一），断言不过记入 `skipped` 而不杀 | ✅（代码 + gate9） |
| 3.8 | 人类确认语义 | 无 `--yes` 时交互确认；由生产托管的进程执行会被识别为 `hosted_by_prod` 并拒绝（除非 `--yes`/`--detach`） | ✅ |

## 4. KR4 挂起可恢复 —— 证据与结论

| # | 断言 | 证据 | 结论 |
|---|---|---|---|
| 4.1 | 优雅关停/重启把在跑会话标 **suspended**（不是 error） | 关停后注册表：`sess-doing=suspended sess-easy=suspended`；重启后仍为 `suspended` | ✅ |
| 4.2 | intent 落盘 | `<state-dir>/suspend.json` 存在；`GET /api/recovery` 返回 `{at,suspended,recovered,failed}` | ✅ |
| 4.3 | **不自动恢复**（核心安全语义） | 重启后 `busy != true`；无 worker 被 spawn；代码层反向断言「出现 `AutoResume` 即 gate8 失败」 | ✅ |
| 4.4 | 人工一键恢复（交互型） | `POST /api/sessions/{id}/continue` 接受挂起会话并尝试 `--session <pi_id>` 恢复（HTTP 202；无真实 pi 时允许 spawn 失败） | ✅ |
| 4.5 | 人工一键恢复（doing/dream） | `/continue` 先把 `tasks.json` 里遗留 `running → pending`（实测：`task1=success task2=pending`）再续跑剩余 task | ✅ |
| 4.6 | 幂等与边界 | `/continue` 对不存在会话 → 404；重复点击 → 202 `already_active`（不重复起 worker） | ✅ |
| 4.7 | 平台自动回来 | 提升/回滚后 `/api/health` 200 且 `build_id` 正确；SSE 陈旧游标 → `replay_overflow` → 前端清游标 + 全量重拉 | ✅ |

## 5. 发现（findings）

| ID | 级别 | 内容 | 复现/证据 | 建议 |
|---|---|---|---|---|
| **F1** | 中 | **dev 工作区守卫触发条件偏窄**：守卫只在 `IsDevStateDir`（`--state-dir ≠ $HOME/.rick`）时安装；`dev-web` 的实际形态（换了 HOME，state-dir 恰为 `$HOME/.rick`）不安装 → 可注册生产已注册工作区（201） | E2E 步骤 2c：`FINDING guard_not_installed_for_home_swap … HTTP 201`（步骤 2b 的 dev 语义形态返回 400） | 触发条件改为「`ProductionStateDir() != ""`（当前 HOME ≠ passwd home）**或** `IsDevStateDir`」；补 1 条单测 |
| **F2** | 中 | **`release --prod-home` 不会连带改默认启动脚本**：`--start-script` 默认仍取真实家目录的 `~/.rick/start-web.sh` → 只改 `--prod-home/--port` 的演练会去跑真生产的启动脚本 | `internal/env/release/defaults.go:19-90`（script 由 `realHomeDir()` 推导，早于 `--prod-home` 覆盖） | 演练纪律：**必须显式传 `--start-script`**（E2E 已如此）；或让 `Validate()` 校验 `StartScript` 落在 `ProdHome` 之下 |
| F3 | 低 | doing 门禁 `helper.py` 路径硬编码 `UserHomeDir()`，不尊重 `RICK_PI_AGENT_DIR` | `internal/handler/doing.go:235-238` | dev 实例跑 doing 时读生产 helper（只读，无写风险）；后续可参数化 |
| F4 | 低 | PWA service worker 注册必然失败（`non-precached-url`） | `web/dist/sw.js` precache 不含 `index.html` 却 `createHandlerBoundToURL("index.html")` | 独立议题；修好后必须同时处理 SW 缓存，否则前端热更退化；dev 构建已支持 `RICK_DEV_NO_PWA=1` |
| F5 | 低 | `web/dist` 入库 + dev 树自带 `.rick` 快照 | `git ls-files web/dist`；worktree 创建于 `a60035a` | dev 会话需生产最新 `.rick` 数据时用绝对路径读生产树 |

## 6. 未覆盖 / 需人工确认

| 项 | 为什么没在脚本里 | 手工验证步骤 |
|---|---|---|
| **真实 pi 会话的端到端恢复** | 脚本不愿为验收消耗模型配额（真实会话恢复会加载大上下文）；且需要真实 pi 会话文件 | ① dev UI 建会话并发一条消息 → ② `rick tools dev-web restart` → ③ UI 显示「⏸ 已挂起（平台升级）」且**没有自动跑** → ④ 点「▶ 恢复继续」→ 会话恢复可对话（pi 以 `--session <id>` 加载既有会话） |
| 真实生产的提升/回滚 | 本次验收**刻意不碰生产**（人类指令：prod 隔离出来） | 需要时由人类执行 `rick tools release --dry-run` → `--yes`；回滚 `--yes --rollback` |
| 并发多标签页 | 需要浏览器多实例编排 | `sse.ts` 游标重放 + `replay_overflow` 路径已由 gate6/E2E 间接覆盖 |

## 7. 结论

- **KR1 ✅（含 F1 残留缺口）**：状态/进程/端口/二进制/pi 沙盒全部隔离；flock 消除 singleton 缺陷；
  守卫在 dev 语义形态下生效，但在 `dev-web` 的 HOME-swap 形态下未安装（F1，建议一行修）。
- **KR2 ✅**：前端热更零重启即时生效；后端改动经 `restart` 行为可见并可被指纹复核；自举形态（会话托管在
  生产、workspace=dev 树）经本次实施全过程实测成立。
- **KR3 ✅**：`rick tools release` 门禁→构建→原子换链→同版前端→重启→**build_id 校验**→挂起清单，
  支持 `--dry-run/--rollback/--keep/--detach`；失败路径自动回滚（单测覆盖）。
- **KR4 ✅**：平台自动回来 + 会话/后台 job 一律**挂起待人工一键恢复**（不自动续跑，规避悬挂 toolCall
  重复副作用与配额静默空转）；doing 续跑前归一化 `running → pending`。
- **生产零触碰 ✅**：E2E 首尾断言 `sessions.json`/`web.json` 指纹与 8413 健康一致，`prod_touched=false`。

---

# 附：交付期发现并修复的 4 个真实缺陷（增量验收补充）

这 4 个缺陷**全部只在「按产品真实调用路径操作」时暴露**——纯单元测试与「按实现分支构造」的门禁都测不到，是本次增量验收最有价值的产出。

| # | 缺陷 | 严重度 | 暴露方式 | 修复 | 门禁加固 |
|---|---|---|---|---|---|
| **F1** | 隔离守卫只在 `--state-dir` 形态安装；`dev-web` 用**换 HOME** 形态 → 守卫等于没装，dev 可注册并写生产工作区（实测 `POST /api/workspaces` 返回 **201**）| 高（绕过 human 裁决的 fail-fast 纪律）| E2E 用**真实启动形态**跑守卫用例 | 新增 `DevModeInfo`（按「= 真实用户生产状态目录」判定，两形态都装守卫；`RICK_PI_AGENT_DIR` 强制仅在 state-dir-only 形态）| gate7 第 ⑨ 条：换 HOME 形态下注册生产工作区必须 4xx |
| **F2** | `release --dry-run` 被自保检查误拦（检查在 150 行、dry-run 分支在 195 行）| 中（拿不到演练计划）| 在**被生产托管**的会话里跑 dry-run | 拦截条件加 `!opts.dryRun`（警告仍打印）+ 可注入 `hostedByProd` 便于单测 | gate9（release 契约）|
| **F3** | dry-run 的构建产物写进生产树 `<prod>/bin/releases/<ver>/`（与「不动生产」承诺不符）| 中（污染生产树）| 同一次 dry-run 后核对 prod 树 | dry-run 构建落 `os.MkdirTemp` 暂存并清理；**`Promote()` 拒收 `Staged` 产物**（纵深防御）；输出 `target_bin/target_dist/prod_untouched/cleanup` | gate9 + `TestDryRunNeverTouchesProd` |
| **F4** | `dev-web` 的 dev 树解析跟随 cwd，从 dev 树内执行算成 `<祖父>/rick-dev`（不存在）| 中（用户最常见操作直接失败）| 收尾更新 dev 实例时 `cd <dev-tree> && dev-web restart` | 解析顺序：显式 flag/env → **内容探测 worktree**（`.git` 是文件 + `cmd/rick` + `web/package.json`）→ 回退；`home = <tree>-home`；树校验先于 `EnsureToken` | gate8：从 dev 树内与从生产仓库执行必须报告同一 tree/home |

**共同教训**：门禁必须按**用户/产品实际调用路径**构造（真实命令、真实启动形态、真实宿主形态），而不是按实现者以为的分支；「安全的演练命令」必须被显式断言为**无副作用**；命令的默认参数解析不能依赖 cwd 的偶然形态。

---

# 增量 3 验收报告：rick-rsi-loop（自进化制度化）

> 设计树：`doing/grilling/design-tree.md` 第 3 棵（L0'' OKR + KR1-KR4 + 层映射 + 判断节点 J-RSI-1..3）
> 流水线：6 task / 4 层 / gate11-14（本文件为层 4 / task28 交付物）
> 验收方式：`scripts/rsi-loop-e2e.sh`（可重放、幂等、真实生产只读）+ 4 个层门禁
> 结论：**E2E `pass=true` / `prod_touched=false`（34 步全绿）；gate11-14 全绿**

## 逐条对照（证据 → 结论）

### KR1 制度载体（loop + loops_check）
| 证据 | 结论 |
|---|---|
| `.rick/loops/rick-rsi-loop.md` 存在，frontmatter（name/trigger/scope）+ 五要素小节齐备；E2E 步「loop 文件存在」「loops_check pass」 | ✅ 制度文本落地 |
| `rick tools loops_check --dir .rick` → `✅ loops_check passed: … (loops 6 / skills 0)`（gate11 + E2E 各跑一次） | ✅ 载体合规可机器校验（该校验器此前只在代码里、无命令入口） |
| loop 正文含机制命令（`dev-web` / `release` / `rsi_check` / `门禁`）与「必须在 dev 工作区」硬约束（gate11 断言） | ✅ loop 可执行而非愿望清单 |
| `.rick/loops/README.md` 与实况对齐（补 `go-refactor-migration-loop` 与 `rick-rsi-loop`） | ✅ 修掉已 stale 的目录 |

### KR2 入口绑定（rsi 会话类型 + workspace 硬校验）
| 证据 | 结论 |
|---|---|
| E2E：模拟生产上 `POST /api/sessions {"type":"rsi"}` → **HTTP 201**，随后 `GET /api/sessions/{id}/prompt` 命中 `rick-rsi-loop`、`产出评估`、`dev-web`、`release`、`approval`，且内嵌 loop 正文片段（`S0 设计`、`产出评估（Output Evaluation）`、`停止标准`） | ✅ **启动 RSI 会话 = loop 必然加载**（全文注入 + `_method_file` 供 resume 重注入） |
| 守卫负例①：workspace = 生产仓库根（`RICK_RSI_PROD_REPO` 显式声明）→ **HTTP 400**，message 含「RSI 会话必须在 dev 工作区运行」 | ✅ 最危险的绕过路径被堵（否则直接改生产源码） |
| 守卫负例②：缺 `.rick/loops/rick-rsi-loop.md` 的源码树 → **HTTP 400** | ✅ 无 loop 不可启动 |
| 守卫负例③：非 rick 源码树 → **HTTP 400** | ✅ RSI 产物就是 rick 本身，非源码树无意义 |
| gate12：`rick rsi --help` 可用；前端类型/入口存在且 `tsc` 通过 | ✅ CLI 与 Web 双入口一致 |

### KR3 闭环执行（release --merge-source，冲突即中止）
| 证据 | 结论 |
|---|---|
| E2E 无冲突用例：`RELEASE_MERGE merged=true`，模拟生产 main 出现 merge commit；提升后 `/api/health` 的 `build_id` == 本次 version | ✅ 源码与二进制**同版生效** |
| E2E 冲突用例：release **中止（rc=4）**，报错含 `CONFLICT`，随后模拟生产 `git status --porcelain` **为空**、`MERGE_HEAD` 不存在、`HEAD` 与中止前一致（**零副作用**） | ✅ human 裁决 J-RSI-2 兑现：冲突即中止并恢复干净，交 AI 修复 |
| E2E 修复后重跑：dev 侧解冲突并提交 → 再次 `release --merge-source` → `merged=true` | ✅ 「AI 修复后重跑」路径可用 |
| gate13：`release --help` 含 `--merge-source`/`--no-merge-source`；merge 单测（含冲突中止/恢复）全绿 | ✅ 语义有单测兜底 |

### KR4 可机器校验产出（rsi_check）
| 证据 | 结论 |
|---|---|
| E2E：缺证据 → fail `0/6` 且逐项给出**中文下一步**（含「跑 dev-web status 把 build_id 贴进 …」） | ✅ 失败可操作 |
| E2E：`--init` 生成骨架后**仍 fail**（残留 `<!-- TODO` 即视为未填写） | ✅ 防自欺（否则契约退化为走过场） |
| E2E：补全 6 项证据（含 `APPROVED by=human at=…`）→ **pass 6/6**；其中 `prod-health` 是对模拟生产的**实时探测**且 `build_id` 必须等于 `release.md` 的 version | ✅ 人类确认是硬门槛；「生产真的跑上了」由探测**证伪**而非自述 |
| E2E 负例：`gates.md` 含 `pass=false` → fail（禁止带病下钻） | ✅ 门禁语义被校验器继承 |
| E2E 负例：`release.md` 的 `version` ≠ 生产 `build_id` → fail | ✅ 证伪路径有效（自述与事实不符会被抓出） |

### 端到端与生产回归
| 证据 | 结论 |
|---|---|
| `bash scripts/rsi-loop-e2e.sh` 末行 `{"pass": true, "steps": [...34 步全 ok...], "prod_touched": false}` | ✅ 全链路闭环 |
| 真实生产只读回归：`/api/health` = 200、`sessions.json`/`web.json` 指纹与基线一致、生产树无 `bin/releases` 残留 | ✅ 验收全程零触碰生产 |
| 真实生产端到端（含真实 release 与重启）**由人类在首次迭代时执行** —— 脚本只覆盖可在隔离环境验证的部分 | ⏭ 已写入手册 §8/§11 的说明 |

## 残留风险（诚实记录）
1. **真实提升的第一次仍未经生产验证**：`--merge-source` 对真实生产仓库的合并、以及重启后 8413 的挂起/恢复，只在「模拟生产」上验过；首次真实 release 由人类执行并观察。
2. **`rsi_check` 的第 6 项依赖可达的生产地址**：非本进程所在主机的部署需显式 `--prod-url`/`RICK_RSI_PROD_URL`。
3. **loop 的纪律靠证据而非强制**：`rsi_check` 保证「没证据不算完成」，但不阻止 agent 绕过 loop 直接改代码（可由人类在 review 时以证据缺失驳回）。
4. **合并只覆盖 dev 分支 → 生产分支**：多分支/多 dev 工作区并行时的合并顺序未定义（当前设计为单 dev 工作区）。
