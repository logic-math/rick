# worker 任务简报：job_49 S1 隔离开发（迭代 2——收敛迭代 1 的 3 个红测试）

你是 job_49 的 doing worker（迭代 2）。迭代 1 已完成大部分实现（dev 树未提交改动 778 行），**编译通过、relocate 包全绿**，但 release 包有 3 个新测试红。你的任务：收敛这 3 个红 → 全量回归绿 → 补证据 → 输出回执。

## 上下文（按序读）

1. `/Users/logic/workdir/AI_CODING/rick/.rick/jobs/job_49/doing/prompts/worker-task-1.md` —— 迭代 1 任务简报（背景/写域/实现清单/验收标准，仍然有效）
2. `/Users/logic/workdir/AI_CODING/rick/.rick/jobs/job_49/doing/grilling/research-L1.md` —— 代码事实简报
3. dev 树现有未提交改动：`git -C /Users/logic/workdir/rick-dev diff --stat`（release.go +270 / release_test.go +433 / import.go+26 / import_test.go +80）——迭代 1 的产出，先读懂再改

## 3 个红测试的精确失败分析（parent 已定位，直接按此修）

跑法：`cd /Users/logic/workdir/rick-dev && env -u RICK_STATE_DIR -u RICK_PI_AGENT_DIR -u PI_CODING_AGENT_DIR go test ./internal/env/release/ -run 'TestStopProd|TestReleaseAutorollback' -count=1 -v`

### 红 1：TestStopProdClaimsBareBuildHoldingLock（裸实例持锁仍被拒杀）
- 场景：fake rick（python shebang 脚本）用自己的 state-dir 写了生产 web.pid，测试手工写 web.lock 记录其 pid
- 失败链：claimBySingleton 被调用 → 「rick 名义」检查 `proc.Exe(pid)` 返回 **python 解释器路径**（shebang 脚本的 exe 是解释器，不是脚本）→ name 不含 "rick" → 拒认领
- **修法**：名义认定加 argv 兜底——exe 不含 rick 时用 `ps -o command= -p <pid>` 取进程完整命令行，argv[0]（脚本/二进制路径）basename 含 "rick" → 名义成立。真实场景映射：11135 是裸 go build 的 rick（exe 本身含 rick，走快路径）；shebang fake 走 argv 兜底。procfs.ProcSnapshot 只有 Comm（无 argv）——可在 procfs 加 Command 采集（Snapshot 的 ps 加一列 `command=`，或新增 per-pid Command(pid) 函数），或在 release 包内直接调 ps（注意超时与错误容忍：取不到→不认领，宁可不杀）

### 红 2：TestStopProdClaimsPortSquatterRickProcess（端口占用裸 rick 完全不在候选列表）
- 场景：裸 rick 监听**生产端口**、用自己的 state-dir（不写生产 web.pid、不持生产锁）、cwd/exe 都不在生产形态
- 失败链：stopProd 候选集只有 ①web.pid 记录 ②findProdPIDs（cwd/exe 断言）——端口占用者**两者都不沾** → stopProd 根本不知道它存在（skipped_why=[] 空）→ 新进程绑不上端口
- **修法**：stopProd 候选集补两个来源：**生产端口占用者**（`portOwner(p.Port)`，已有此函数）和 **web.lock 文件记录的持有者 pid**（读 p.ProdLockPath()）。候选进 claimBySingleton（名义+持锁/占端口双条件）判定
- 误杀防线不变：非 rick 名义（exe/argv 都不含 rick）的进程仍宁可不杀并记录 skipped_why

### 红 3：TestReleaseAutorollbackRefusesBadRollbackTarget（自动回滚未点名坏回滚点）
- 场景：promote 健康失败 → 自动回滚 → 回滚点是 ELF → 应拒绝切链并在错误信息**点名「回滚点不可执行」**
- 失败链：大概率是红 1/红 2 的链式影响（stopProd 失败 → health 失败 → 进自动回滚 → 回滚点校验应拦截但错误组装没把「坏回滚点」信息传出来）。先修红 1/2 再看此测试是否连带变绿；若仍红，检查 Rollback() 的 verifyRunnable 失败错误如何在 Restart/autorollback 错误链里透传（应含「回滚点不可执行」字样）
- 测试断言（release_test.go:976 附近）：错误信息应点名坏回滚点

## 完成标准（全达标才输出回执）

1. `env -u RICK_STATE_DIR -u RICK_PI_AGENT_DIR -u PI_CODING_AGENT_DIR go test ./internal/env/... ./internal/env/relocate/... -count=1` 全绿
2. 全量 `go test ./... -count=1` 无回归（TestWebSingleton 相关注意 env 隔离，已知坑见 domain/bugs.md）
3. `rick tools dev-web restart`（CLI 用 /Users/logic/workdir/AI_CODING/rick/bin/rick）→ DEV_UP 新 build_id → curl http://127.0.0.1:8414/api/health 确认 build_id 更新
4. **补迭代 1 欠的证据**：在 `/Users/logic/workdir/AI_CODING/rick/.rick/jobs/job_49/doing/rsi/dev-iterations.md` 追加（先删掉 TODO 标记行）：一行/迭代，格式示例：
   `迭代1 build_id=<dev-web restart 后的指纹> 改动=release.go P0-1/2/3/4+import.go P1-2 测试=新增X个 结论=3红收敛中`
   迭代 2 完成后同样追加你的一行
5. 输出回执：改动文件清单（限写域：release.go/release_test.go/import.go/import_test.go + procfs 若动）+ 每处修改要点 + 测试结果摘要（数量/绿）+ dev-iterations.md 已补 + 遗留问题

## 写域与禁区（不变，硬约束）

- 代码只写 dev 树 /Users/logic/workdir/rick-dev：internal/env/release/、internal/env/relocate/、internal/env/procfs/（如需）
- 证据只写 main 树绝对路径 /Users/logic/workdir/AI_CODING/rick/.rick/jobs/job_49/doing/rsi/dev-iterations.md
- 禁止：碰 main 树源码/6137（生产）/8414 实例（除 dev-web restart 命令）；任何 git 操作；碰 dev 树 .rick/（job 数据只在 main 树）
- ⚠️ dev 树的 .rick/jobs/job_49 已被 parent 删除（13:08 事故残留 stub，防 S5 合并冲突）——若你的操作又在 dev 树重生它，说明 rick CLI 在 dev cwd 有 job 引导副作用，报告它但不要手动清理

## 失败处理

先搜 /Users/logic/workdir/AI_CODING/rick/.rick/domain/bugs.md；仍卡 → skill:debug-skill（/Users/logic/workdir/AI_CODING/rick/.rick/jobs/job_49/doing/prompts/skill_debug_skill.md），debug 记录写 main 树 /Users/logic/workdir/AI_CODING/rick/.rick/jobs/job_49/doing/debug/bug{N}-*.md
