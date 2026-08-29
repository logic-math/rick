# rick web ui 流水线执行前全链路预演报告（sim-report）

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
