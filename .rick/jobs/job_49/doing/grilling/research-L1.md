# research-L1 简报（job_49 恢复 rick web 生产功能）

HEAD=02d144d 只读调研（未改任何文件）；实测基准：2026-10-30 晚。所有 file:line 均指当前 HEAD。
**新增重要事实（交接单之后的状态漂移）：生产 web（6137）此刻已死**——lsof 无监听、curl /api/health 无响应、web.lock=34061 为死 pid（ps 无此进程）；dev 实例 8414 存活（pid 76547，rick-dev-home）。恢复动作须先于 P0 修复（临时处置同交接单 §5）。

## 1. release.go 函数群（internal/env/release/release.go）
- a) **可执行性预校验仅存在于 Build 内**：smokeCheck（L556-563，`bin web --help` 冒烟）在 Build L436 调用，失败→stage=build、exit 3。Promote/Rollback/Restart 对**已存在的版本目录零校验**（不做 magic/冒烟）；错平台二进制最早在 restart 阶段暴露（startProd L833 exec 失败，start-web.sh 报 cannot execute binary file），或 health 阶段 build_id 不符。置信 0.95（代码+web.log 实证）。
- b) **Rollback()（L1000-1031）近无条件**：仅校验 .last 非空+版本目录存在（dirExists L1010）；无二进制可执行性校验；切链+restoreOverlay，自身不重启（重启由 CLI 层 tools_release.go L226 Restart 完成）。回滚点本身是 ELF → 回滚后生产彻底不可启动（F4 成立）。置信 0.95。
- c) **ownedByProd（L1276-1306）三断言**：①proc.Cwd(pid)==prodRepo（EvalSymlinks 归一）②proc.Exe(pid) 前缀∈prodRepo/bin ③proc.Environ HOME==ProdHome。darwin 上 Environ=ErrUnsupported（procfs_darwin.go:100）→HOME 兜底死分支；Cwd/Exe lsof 失败→分支静默跳过→全 false。**skipped=[11135] 产生于 stopProd L816-818**（!ownedByProd→append skipped，宁可不杀）。置信 0.9。
- d) **prodEnv（L869-893）白名单**：HOME=ProdHome（=真实家目录，非 ~/.rick）、PATH、RICK_STATE_DIR、RICK_PI_AGENT_DIR、TMPDIR/LANG/LC_ALL/TERM、6 个 proxy 变量。HOME 在白名单→startProd 拉起的进程理论可被 HOME 兜底认领，但 darwin Environ 不可用→兜底实际失效（交接单 R2 推断的一半成立）。**实测本机 lsof 对同用户活进程可用**（76547 的 cwd/exe 均取到）→削弱「lsof/TCC 失败」假设，11135 更可能是裸构建（build_id=dev）+cwd/exe 不在生产形态→断言按设计拒绝认领（R7-1 待 P0-3 复现定论）。置信 0.75。
- e) **waitHealthy（L896-929）**：轮询 GET /api/health 至 status=ok（L903），返回（pid 文件→findProdPIDs 兜底的 pid，耗时，build_id）；失败报「生产未就绪（url）: lastErr」+Restart 层附 logTail（L985，web.log 尾 12 行）。**盲区**：不区分「旧进程占端口（读到旧 build_id）」vs「新进程没起来」——本案 health 读的是漏杀的 11135（build_id=dev），报错文案误导成「新构建没生效」（release-state.json 实证）。置信 0.9。

## 2. wrongPlatformBinary（internal/env/relocate/import.go）
- 逻辑（L812-823）：sniff 前 4 字节——macho=0xfeedface/facf 或 0xcafebabe（fat）；elf=0x7f'ELF'；linux→macho 即错、darwin→elf 即错、其他→任一即错。
- 唯一调用方 adaptPlatform（L783-807），且**只作用于 `<AgentDir>/bin/rg`**（错平台直接删除）；原生 rick 二进制只暂存为 `<StateDir>/bin/rick-import-<goos>-<goarch>`（L795）+「手动 cp」提示。
- **F6 根因（实测确认）**：~/.rick/bin/rick 由 dump/import 的 state 树原样搬运（mergeState 逐文件 copy，无校验），import 不覆盖它——实测 ~/.rick/bin/rick 仍为 **ELF x86-64**（10-15 时间戳），而旁边 rick-import-darwin-arm64 是可用 Mach-O。置信 0.95。

## 3. web 服务与停机链（internal/cmd/web.go / internal/web / internal/handler）
- **--daemon（web.go L63-106）**：re-exec 自身（RICK_WEB_DAEMON=1，重组显式 flag）+`SysProcAttr{Setsid:true}`+stdout/stderr→<stateDir>/web.log，打印 pid 即返回（Setsid 免 SIGHUP= nohup 等价）。
- **单实例**：web.lock flock LOCK_EX|LOCK_NB（internal/web/statedir.go:135-152）——权威在内核 flock，锁内 pid 纯诊断；死持有者 fd 自动释放→**F5 复核成立**（删锁非必要）。另有 web.pid 存活探测（handler/web.go:143-170，弱约束 J2.7）。
- **「graceful shutdown: context deadline exceeded」产生于 internal/web/server.go:189**（httpServer.Shutdown 超 shutdownGrace=10s，server.go:79；SSE 卡住时超时→force Close→返回该错误）。
- **SIGTERM 链路**：signal.NotifyContext（handler/web.go:112）→Serve ctx.Done（server.go:174）→**先** ShutdownWorkers（sessions.go:1895：suspendRunning→sup.CloseAll；Worker.Close=SIGTERM 整个进程组→TermGrace 5s（supervisor.go:57）→SIGKILL 组）→**再** HTTP drain 10s→超时则报错退出。
- **worker exit 143 后 web 跟着死＝同因非因果**：web 收到外部 SIGTERM→优雅停机先杀 worker（组内 SIGTERM→143）→SSE drain 卡 10s→web 带错误 exit 1；且 cobra 未设 SilenceUsage（全仓无）→每报错打整页 Usage（web.log L42-73 实证：worker 143→Error→Usage）。置信 0.9。

## 4. build_id="dev" 机制（internal/cmd/root.go:26）
- `var BuildID = "dev"`，release 构建经 -ldflags `-X …cmd.BuildID=<version>` 注入（release.go:51 BuildIDLdflag、L509-510 buildBinary）。
- **health build_id="dev" ⇒ 进程来自未注入指纹的构建**（go run/裸 go build/手工产物），绝非 release 链产物——release-state.json 中 11135 build_id=dev 与「裸构建实例占着生产锁」完全吻合。置信 0.9。

## 5. release_test.go 现状（internal/env/release/release_test.go）
- 24 个用例（L178-839）。回滚/重启相关：TestRollbackRestoresPreviousVersion L480、TestRollbackWithoutRollbackPointFails L519、TestReleaseAutoRollsBackWhenHealthFails L777、TestRestartVerifiesBuildIDAndReplacesProcess L326、TestRestartFailsWhenRunningBuildIsStale L368、TestRestartRefusesToKillForeignProcess L397、TestExitCodeForStage L839。
- 基建：newFakeProd（t.TempDir 模拟生产）+fakeBuildBinary（python3 假 rick：写 web.pid、/api/health 返 build_id）。**P0-1/P0-2 用例放同文件合适**（Plan.BuildBinary 可注入；冒烟对可执行脚本可过；P0-2 可直接构造坏版本目录测 Rollback 拒绝）。
- ⚠️ 卫生问题（实测）：**10-18 泄漏的 8+ 个 TestRelease*/TestRestart* fake 进程至今存活**（ps 实测，python 假 rick 占临时端口）——t.Cleanup 的 stopProd 未收掉它们（成因未查，R7-3）。置信 0.85。

## 6. procfs（internal/env/procfs/procfs_darwin.go）
- 枚举：ps -eww -o pid=（L106）；Snapshot=ps -axo pid,comm + lsof -F pc 全量 cwd/txt 共 3 次子进程调用（L135-190）；Cwd/Exe=lsof -a -p <pid> -d cwd/txt（L88/L70，3s 超时 L24）；Environ=ErrUnsupported（L100，注释：KERN_PROCARGS2 env 不可靠）；Alive=kill(0)+ps 僵尸位（L27，含 macOS os.ErrProcessDone 特判）。
- 已知失败模式：跨用户/SIP 保护进程 lsof 部分失败；**本机实测同用户进程 lsof 正常**（见 1d）。置信 0.9。

## 7. e2e 对 release/promote/build_id 的断言（scripts/）
- self-evolve-e2e.sh L286-343：真实跑 `rick tools release --yes --no-gates --json`（临时 HOME 模拟生产）+第二轮 `--rollback`；断言 RELEASE_OK version=、RELEASE_RESTART matches=true、health build_id==version（L314）、RELEASE_RECOVER（L319）、rollback 后 build_id 回到 HB1（L335）。
- rsi-loop-e2e.sh：`go build -ldflags -X …BuildID=$VER` 造模拟生产（L186），rsi_check prod-health 要求 build_id==release.md version，负例不匹配必须 fail（L214-217）。
- **P0 修复预期不需适配**：只要不动上述输出标记与 health 语义；P0-1/P0-2 新增的「拒绝/校验」路径不在现有 e2e 的成功断言路径上（若改 RELEASE_* 标记文案则需同步）。置信 0.85。

## 8. 平台校验/脚本生成点（全仓搜索）
- 「平台校验」仅 import.go wrongPlatformBinary（只护 rg）；全仓 exec format 字样仅 tools_release.go:415 注释（相对路径问题，已修）。**无其他 magic/ENOBJEXEC 处理**。
- **start-web.sh 非 rick 源码生成**：属 dump/import 搬运的部署文件（tools_dump.go:42/153 列为搬运对象；import.go:316 经 remap.go:302 RewriteStartScript 只重写 cd 行）；改脚本=改 ~/.rick/start-web.sh 文件本身。
- **deploy-web.sh 全仓零引用=纯手写**（修它=直接改 ~/.rick/deploy-web.sh）；其 `cd /workdir/sunquan20/...` 旧机路径，本机必失败（F9 成立）。置信 0.9。

## 9. web customize / 覆盖层（internal/cmd/web.go L355-373 + internal/env/web.go + internal/web/static.go）
- customize：幂等抽取内嵌源码基线到 ~/.rick/web/（env/web.go:51，.rick-managed 标记，已存在则跳过）。
- **优先级：覆盖层逐请求优先于内嵌 dist**——StaticHandler(embedFS, <stateDir>/dist)（static.go:45-52；NewServer 组装点 server.go:119-121）；overlay 缺 index.html 回落 embed。
- reset（env/web.go:83）：删 ~/.rick/web/ 全部内容**仅保留 sessions.json**（web.json 在 ~/.rick/ 根不受影响）。
- P1-5（前端对齐）处置选项：①`rm -rf ~/.rick/web/dist`——只回滚前端到内嵌基线、保留 src 自定义层（deploy-web.sh 注释同法）②`rick web reset --yes`——连 src 一起清 ③release promote 同版重跑（syncOverlay 同版推进+dist.prev 留点，最贴「前后端同版」）。置信 0.85。

## 10. config.json 相关项（token 已脱敏）
- default_workspace=/Users/logic/.rick（=state 目录，非工作树）；pi_extra_args=--provider catpaw-proxy --model glm-5.3 --thinking high（worker 模型链）；pi_path 空、runtime=pi；web_max_active 未设（默认 64，cmd/web.go resolveSupervisorConfig）；human_loop.{max_retries=5,sense_max_backflows=3,think_min_assumptions=5,think_top_n=3,research_source_weights}；git user。
- ~/.rick/web.json 注册 4 工作区（MT_INFER/RICK-RSI/MOGIC_HOME/PLATOIM），均为新机路径（无旧机残留）。置信 0.9。

## 交接单锚点核实结论表（wiki/handover-2026-10-30-web-start-macos.md §11）
| 交接单锚点 | HEAD 实测 | 判定 |
|---|---|---|
| Restart≈L760 | release.go L764 | ✅ 准确 |
| waitHealthy≈L895 | L896 | ✅ |
| Rollback≈L999 | L1000 | ✅ |
| stopProd≈L793 | L795 | ✅ |
| ownedByProd≈L1276 | L1276 | ✅ 精确 |
| prodEnv≈L865 | L869 | ✅ |
| root.go:26 BuildID="dev" | L26 | ✅ 精确 |
| wrongPlatformBinary（import.go） | L812 | ✅ |
| --daemon setsid+nohup（web.go） | L63-106 | ✅ |
| F1-F9 事实 | F5/F6/F9/ELF 分布经实测复核均成立 | ✅ |
| release-state.json 引用 | skipped=[11135]/build_id=dev/错误链逐字一致 | ✅ |
| **§1「16:50 生产在跑」** | **6137 已死（无监听、web.lock=34061 死 pid）** | ⚠️ 漂移（新事实，恢复优先） |

## R7 上报（无法在本轮澄清）
1. 11135 漏杀精确根因未定论：lsof 当次失败 vs 裸构建进程 cwd/exe 不匹配（进程已死无现场）——需按交接单 §12 在 6199 复现（P0-3）。
2. SSE 卡满 10s drain 超时的具体路径未考证（internal/web SSE handler 未读）。
3. 10-18 泄漏测试 fake 进程未被 t.Cleanup 收掉的成因未查。
4. start-web.sh 历史生成方式未考证（当前源码只搬运不生成）。

叶子文件：无（全部自研，无外包）。
