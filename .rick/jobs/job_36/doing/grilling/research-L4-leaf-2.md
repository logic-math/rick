# L4-leaf-2：双实例对同一 workspace 的写冲突（Q5）

## 结论（3 条）
1. 全库无 flock/跨进程锁（grep 实测为空），一切状态靠进程内 mutex + tmp+rename → 双实例必然丢更新。
2. 最致命：ReconcileOnStart（sessions.go:163-166）无条件把在跑的 doing/dream 标 error 并全量回写 sessions.json → dev 启动即刻"误杀"prod 全部会话。
3. worker 注册纯内存（supervisor.go:294），同 session 可被两实例各起一个 pi 进程，共写同一 session jsonl。

## 1) 机器级状态文件写点（写方式：均 tmp+rename 原子，但无跨进程互斥）
| 文件 | 写入函数链 | 原子性/锁 |
|---|---|---|
| ~/.rick/web.json | registry.go:133/150/191 → save() registry.go:218 → writeFileAtomic registry.go:355-371 | 原子 rename；无锁 |
| ~/.rick/web/sessions.json | registry.go:291/304 → save() registry.go:341 → writeFileAtomic registry.go:355 | 全量覆盖；无锁 |
| ~/.rick/web/archived.json | archived.go:70/87 → persist() archived.go:106-124 | 原子；无锁 |
| ~/.rick/web/job-names.json | jobnames.go:139 → saveLocked() jobnames.go:143-172（CreateTemp+Rename） | 原子；无锁 |
实测：`grep -rn "flock\|Flock\|LOCK_EX\|\.lock" internal/ pkg/ --include=*.go | grep -v _test` → 空。
后果：最后写者胜。registry 是**整份内存快照**序列化，A 实例只改 1 个 session，却会用自己陈旧的内存覆盖 B 刚写的全部内容（丢更新，非损坏）。web.json 丢更新 = 丢工作区/顺序（major）；job-names/archived 丢更新 = 别名/归档丢失（minor）。

## 2) workspace 级写点（`<ws>/.rick/`）
| 写点 | file:line | 方式 |
|---|---|---|
| doing 初始 tasks.json | builder/orchestration.go:408-445（WriteFile 445）← handler/doing.go:115-117 | 非原子；幂等（413-414 已存在即跳过，不会重置状态） |
| easy 合成 tasks.json | handler/easy.go:235（writeEasyTasksJSON）← easy.go:189、sessions.go:1877 | 非原子 |
| hook 改任务状态 | .rick/skills/mark_task_success_skill/mark_task_success.py:73-74（open(w) 截断写） | 非原子；契约见 builder/orchestration.go:392「tasks.json 只由 hook 写」 |
| easy close 翻 success | sessions.go:808（updateEasyTasksStatus，def 783）← 调用点 768 | 非原子 |
| job 目录创建 | sessions.go:1836+1842（plan）、1863+1868（easy）；scanNextJobID sessions.go:1987-2006 | TOCTOU：先扫 max job_N 再 +1，两实例同扫必撞同一 job_N+1（major，后写者覆盖前者 tasks.json/requirement.md） |
| git 提交 | handler/doing.go:311-332 ensureGitRepo：`git init` + `git add -A`（319）+ commit（322）；repoRoot=filepath.Dir(rickDir)（311） | 两实例并发 doing 同一 workspace：`add -A` 把对方半成品纳入同一 commit（major） |
| session_id / method.md | sessions.go:456-457、1979 | 覆盖写 |
补充实测：task 级 commit **不在 Go 代码内**——`grep -rn "git commit" internal/ .rick/skills/ scripts/` 仅命中 doing.go:329、orchestration.go:392（契约文本）、scripts/version.sh:212；即由 pi agent 自己 bash 提交，两实例同 job 会交错提交。

## 3) 只读但双实例互踩
- watcher.go:41-44、103-173：2s 轮询 tasks.json 与 dist，只读 → 两实例各发 jobs_update，前端收双份事件（minor，不损坏）。
- **ReconcileOnStart sessions.go:158-173（调用点 internal/cmd/web.go:107）**：doing/dream 命中 163-166 **无条件**标 error；交互型在 168-171 因本进程无 worker 也标 error。dev 实例读到共享 sessions.json 即把 prod 所有在跑会话改 error 并全量回写；prod 之后任一事件用内存快照回写又顶掉 → 状态抖动，prod 侧显示 error/Resume。**blocker**。

## 4) worker 归属
- supervisor 纯内存 map：注册 supervisor.go:294，`Get` 316-321，重复 id 仅本进程拒绝（178/284）→ 两实例互不可见。
- spawn 用 `--session-id` / `--session`（supervisor.go:204-206）恢复既有 pi session；同一 web session 被两实例同时 resume → 两个 pi 进程写同一 AgentDir/sessions/**/*.jsonl，且 sessions.go:457 的 session_id 互相覆盖。**blocker**（会话文件损坏风险，jsonl 追加交错程度待验证）。

## 5) 缓解建议
dev 实例必须：独立 HOME（状态/pid/sessions.json 天然隔离，见 L4 主体）+ 独立 workspace 注册表**只指向 dev worktree**（避免与 prod 共享任何 `<ws>/.rick`）+ 独立 RICK_PI_AGENT_DIR（避免 jsonl 互写）。仅隔离 HOME 而不隔离 workspace 时，dev 会话仍会写 prod workspace 的 `.rick`（tasks.json/job 目录/git add -A），必须视为泄漏点。
