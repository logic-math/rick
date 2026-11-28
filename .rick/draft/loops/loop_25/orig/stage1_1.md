# stage1_1 调研素材 — 问题还原：生产 daemon 冒烟·重启连续性

> orig agent 首启调研（2026-11-29）。用途：阶段 1（问题还原）——帮助 human 分别说清**期望 / 现状 / 差距**。
> 全部为带来源的事实与张力；价值判断单独标出交 human 裁决。本文不含任何「应该怎么设计冒烟」的答案。

## A. 代码事实：本仓库「重启连续性」的现行契约（一手来源）

### A1. daemon 架构承诺了什么

来源：`internal/runtime/daemon.go:1-20`（文件头注释）

- 架构：`Go web（可重启）── unix socket ──→ shim（自有进程组，持有管道）── pipes ──→ pi --mode rpc`。
- 明示承诺（注释原话）：「a restart loses nothing the pi process emitted in the meantime」——服务重启期间 pi 产出的任何东西都不丢。
- 生命周期四路径：`spawnDaemon`（新 worker）/ `Attach`（服务重启后重连活 socket）/ `DetachAll`（服务关闭——断连接、**pi 存活**）/ `CleanupOrphans`（无主 socket → 杀进程组 + 删文件）。

### A2. 服务端重启对账语义（ReconcileOnStart）

来源：`internal/web/sessions.go:187-295`

- 注册表里 active/running 的交互型会话（有 PISessionID）：先试 socket 探活 Attach——活着的 pi **保持 active 零中断**；socket 死了才落回 suspended。
- suspended 会话也探活（2026-11-28 dev e2e 实测修复）：live daemon 存在时「suspended」是谎言，必须恢复 active。
- `ShutdownWorkers`（`sessions.go:2223-2245`）：SIGTERM → grace → SIGKILL 是**显式关闭**路径；服务重启走的是 DetachAll——status 保持 active 落盘，pi 继续跑（工具调用/子代理不断）。
- 收尾 `CleanupOrphansExcept`：无主 daemon socket 杀组删文件。

### A3. shim 缓冲回放契约（有界）

来源：`internal/runtime/daemon_shim.mjs:12-14, 35, 65-66, 90-98`

- 服务断连期间 pi 的 stdout 行积累在内存 ring（`BUFFER_CAP = 2000` 行），**最旧的被丢弃并计数**。
- 重连时先发 `{"type":"shim_attached","buffered":N,"dropped":M}` 再回放缓冲行。
- 即：「重启不丢产出」的承诺是**有界的**——超过 2000 行的停机窗口内输出会被截断（有计数、可观测，但仍是丢）。

### A4. 生产重启的三条真实路径

来源：`internal/env/release/release.go`（Restart≈L876 / waitHealthy≈L1071 / stopProd≈L937）、`.rick/domain/env.md`（launchd 守护环境条目）、`wiki/handover-2026-10-30-web-start-macos.md`

- **release 提升路径**：`Restart` = stopProd（pid 文件 + 进程枚举 + 所有权断言，宁可不杀绝不误杀）→ startProd → `waitHealthy`（轮询 `GET /api/health`，只验 `status=ok` + `build_id` 匹配）→ build_id 不一致即报错并给出 portDiagnose。
- **launchd 守护路径**：生产跑在 `~/Library/LaunchAgents/com.rick.web.plist`（KeepAlive + RunAtLoad + ThrottleInterval=5）。macOS 睡眠/唤醒会给后台 daemon 发 SIGTERM——`--daemon`（setsid+nohup）防 SIGHUP **防不了 SIGTERM**（env.md 实证：web 两次同签名死亡，worker exit 143）。KeepAlive 复活实例带的是 plist 硬编码 env。
- **人工路径**：手工 restart / plist 成对重载（bootout + bootstrap）。

### A5. 现有验证覆盖（测试现状）

来源：`internal/web/daemon_e2e_test.go`（文件头 + 5 个测试）、`internal/runtime/executor_realpi_smoke_test.go`（文件头）、`scripts/rsi-loop-e2e.sh`（头部纪律块）

- `internal/web/daemon_e2e_test.go`（进程内 e2e，**fake pi**）：覆盖 ①RestartContinuity（create → spin 长任务 → ShutdownWorkers → 新 manager 重连：PISessionID 不丢、pi 进程组不死、事件回放、attach 后新回合可用）②closed 会话不复活 ③NoDuplicateWorker（幂等）④crashed pi → suspended ⑤SuspendedSessionReattachesToLiveDaemon（旧二进制错标回归）。**全部在同一测试进程内模拟重启**（fresh manager over same registries/socket dir），fake pi 不调真 LLM。2026-11-29 实跑：`go test ./internal/web/ -run TestDaemonE2E` 通过（8.4s）。
- `internal/runtime/executor_realpi_smoke_test.go`：`realpi` build tag 的真 pi 冒烟先例（非默认跑；需 provider API key）——但只覆盖 executor `Run` 路径，**不含 web/daemon 重启连续性**。
- `scripts/rsi-loop-e2e.sh`（进程级 e2e 先例，头部「绝对纪律」原话）：「真实生产（~/.rick）**只读**：脚本只读它的 sessions.json/web.json 指纹与 /api/health；**绝不写、绝不 restart**」——所有模拟实例跑临时 HOME + 临时端口 + 临时 state dir，收尾断言生产指纹与健康未变。

### A6. 生产实时快照（2026-11-29 实测，只读探测）

来源：`launchctl list`、`curl http://127.0.0.1:6137/api/health`、`~/.rick/pi/agent/sockets/`、`~/.rick/web/sessions.json`（全部只读）

- launchd `com.rick.web` 活跃：pid 92727；`/api/health` = `{"status":"ok","build_id":"5e59492-261128231340"}`（与 git HEAD `5e59492` 一致）；`web.pid`/`web.lock` 都在。
- `~/.rick/pi/agent/sockets/` 现存 **2 个 live daemon socket**（各带 meta.json：shim_pid/pi_pid/pgid）。
- `web/sessions.json` 约 140 条会话：2 条 active（`dd4ecf8f` easy + `ffec029c` human-loop，即**本次 loop 所在会话**）、约 10 条 suspended、若干 error，其余 closed。
- 注意：`ffec029c`（human-loop f4e21408）就是当前这个 human-loop 会话本身——**「生产上正在跑的重启连续性实验对象」里包含冒烟测试的设计者自己**。

## B. 行业事实（常识性置信信息源）

### B1. 冒烟测试的定义与边界

来源：Microsoft Engineering Fundamentals Playbook（microsoft.github.io/code-with-engineering-playbook/automated-testing/smoke-testing/）、MinimumCD Practice Guide（beyond.minimumcd.org）、deployment.to glossary、Semaphore tutorial

- 共识定义：部署/构建后**立即执行的小而快**的检查，验证系统起来了、**最关键路径**能用——是门禁（gate），不是全量测试。
- Microsoft Playbook：目标是「as early as possible 发现应用是否工作」，作为后续更贵测试/真实流量的前置门禁。
- 边界共识：只查关键功能存活（vitality/critical paths），不做穷尽验证。
- 部署期细分（Semaphore）：配置冒烟（部署前验配置）→ 基础设施冒烟 → **部署后冒烟**（post-deployment smoke）。

### B2. 「进程重启」≠「服务连续」（进程监督语义）

来源：XB Field Notes「systemd supervises process lifecycles」（xbcatm.com）、Ikoma's Blog（systemd socket activation 让连接等待而非拒绝）

- systemd 把进程起停变成可观测的状态转移（restart policy），但**它不能证明重启后应用能应答请求、也不能证明重启修复了底层故障**——进程级健康（pid/端口/build_id）与应用级连续是两层。
- socket activation 模式下 systemd 重启期间连接**排队等待而非被拒绝**——与 rick shim 的 ring buffer 回放同构：停机窗口内的产出由中间层持有。

### B3. 优雅重启/连续性模式（daemon 界先例）

来源：Envoy hot restart 文档、uWSGI「The Art of Graceful Reloading」、Cloudflare Oxy blog、einhyrningsins(1)

- 共同模式：新旧进程交接监听 fd 或由中间层持有连接；旧进程 drain 而非死；**连接/会话要么交接要么在停机窗口被缓冲**。
- Envoy 明示边界：hot restart 期间既有连接不迁移到新进程——必须在 drain 窗口完成或被终止。即业界也把「重启期间既有连接怎么办」当作**要显式定义的契约**，不是默认成立。

### B4. 重启作为混沌实验（test in production 谱系）

来源：AWS Prescriptive Guidance（混沌实验生命周期：先定义稳态假设→注入→观察）、Harness chaos faults（Linux service restart / process kill 是标准故障注入）、Riot Games「Controlled Chaos」（先连续 100 次通过再上生产）、Istio「Test in production」

- 共识方法：①先写下**稳态假设**（steady state：重启前后哪些指标/状态必须不变）②受控注入重启/杀进程 ③观察稳态是否保持。
- Riot 的分级：低风险环境 100% 通过后才在生产注入。
- Ionize 文档示例恰好是 Postgres 重启：「Expected: 服务断连→自动重连→**会话恢复后仍有效**」——重启连续性的稳态假设写法先例。

## C. 张力（事实对事实，本仓库内部）

- **T1｜两层「重启成功」语义分裂**：release `waitHealthy` 只验 `status=ok` + `build_id`（进程级：新二进制活着、端口在服务）；daemon 架构承诺的是会话级（pi 不死、事件回放、零中断）。B2 的 systemd 引文同构：进程级监督证明不了应用级连续。→ 现在 release 流程在重启后**没有任何会话级检查**。
- **T2｜连续性契约只被 fake-pi 进程内验证过**：5 个 e2e 场景全部用 fake pi + 同进程内模拟重启（fresh manager over same registries），从未在真实进程边界（SIGTERM 到真实 web 进程 / launchd 复活 / flock 交接 / 真 node + 真 pi）上验证。而生产历史有 4 次实证的「测试假设 vs 生产现实」分歧：web.pid 缺失（单例门禁虚设）、stopProd 拒杀（job_47）、sleep/wake SIGTERM 杀会话（env.md 实证）、旧二进制把 active 错标 suspended（2026-11-28）。→ 本仓库有自己的证据链说明：**生产路径会偏离测试假设**。
- **T3｜「生产只读」纪律 vs 「生产 daemon 冒烟」的字面含义**：rsi-loop-e2e.sh 头部把「绝不写、绝不 restart 真实生产」写成绝对纪律；而「生产 daemon 冒烟：重启连续性」按字面是在生产上**触发一次重启**（写操作，有爆炸半径）。两者不能同时成立——「生产」一词在这两个语境里指的不是同一个东西。
- **T4｜冒烟「小而快」定义 vs 连续性观察窗口**：B1 说冒烟是部署后小而快的门禁；但「重启连续性」的最小证据需要**一个跨越重启的 in-flight 事件序列**（重启前发起 → 停机 → 重启 → 回放/续跑验证）。窗口大小（多久停机、多少事件、真 LLM 还是假负载）直接决定它还是不是「冒烟」量级。
- **T5｜「不丢任何产出」承诺的有界性**：shim ring 是 2000 行封顶、超出丢最旧（A3）；「重启零丢失」的表述与实现之间有一个可观测但未在期望层被讨论过的边界。
- **T6｜自指**：当前生产里仅有的 2 个 active 会话之一（`ffec029c` human-loop f4e21408）就是本次 loop 本身（A6）。在生产上做重启冒烟 = 冒烟对象包含正在设计冒烟的人。这是任何「环境隔离」方案都无法完全消除的结构性事实。

## D. 价值判断（无法从代码/外部来源推出，需 human 裁决）

- **V1「生产」的指称**：①真实 `~/.rick` 实例（launchd + 真会话，in-place）②生产形态的隔离环境（同二进制/同路径形态/临时 state dir）③只读探测真实生产（不触发重启）。rsi-loop-e2e 先例选 ③；本主题字面像 ①；三者验证的是不同的命题。
- **V2 爆炸半径可接受度**：in-place 重启真实生产会打断当前 2 个 active 会话（含本 loop）的 web 侧交互若干秒；按架构承诺 pi 侧不中断——但这恰恰是**未被真实进程边界验证过的**那部分。接受多大风险换多大证据强度，是价值判断。
- **V3 负载真实性**：真 pi（真 LLM API，慢、花钱、不可重复）vs fake pi（可控、可重复，但验证不到真 LLM 交互的时序面）。realpi build tag 先例（executor）选择了默认跳过。
- **V4 冒烟的挂载点**：嵌入 release Restart 流程（每次提升自动跑）vs launchd 侧 kill/revive 演练 vs 独立手动脚本。挂载点决定「部署后门禁」还是「一次性实验」。

## E. 引导性问题（阶段 1 收口用）

**主问题**：当你说「生产 daemon 冒烟：重启连续性」时——
1. **期望**：你期望「重启连续性」在生产上具体指哪些不变量？（pi 进程不死？事件零丢失（含 2000 行边界怎么算）？active 不变 active？closed 不复活？前端 SSE 断线重连？）其中哪些是**冒烟**量级（关键路径、小而快、B1 定义），哪些你其实接受留给 fake-pi e2e 覆盖？
2. **现状**：上面 A 节的哪些事实你认作现状、哪些你不认（需要我继续挖证据）？特别是：release `waitHealthy` 只到进程级、连续性从未过真实进程边界——你同意这是现状吗？
3. **差距**：如果差距是「会话级连续性在生产真实重启路径上无任何验证」，那 D 节四个价值判断（生产指称/爆炸半径/负载真实性/挂载点）里，哪个是你现在就要裁决的、哪个可以先悬置？

（E 只问不答——期望的具体条目、差距的具体表述，由 human 的思考产出。）
