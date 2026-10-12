# 依赖关系
task24

# 写域
internal/env/release/merge.go
internal/env/release/merge_test.go
internal/env/release/release.go
internal/cmd/tools_release.go
internal/cmd/tools_release_test.go

# 任务目标
给 `rick tools release` 加 `--merge-source`：把 dev 分支合并进生产 main（源码随二进制一起生效），**冲突即中止报错**并把现场留给 AI 修复（human 裁决 J-RSI-2）。

# 关键结果
1. `internal/env/release/merge.go`：`MergeSource(plan Plan) (MergeResult, error)`
   - 前置校验（任一失败即中止，**不产生任何改动**）：生产工作树干净（`git -C <prodRepo> status --porcelain` 为空；不干净 → 错误提示先提交/暂存）；dev 分支名可解析（`git -C <devTree> rev-parse --abbrev-ref HEAD`）；生产 main 分支名可解析。
   - 合并：在**生产仓库工作树**里执行 `git merge --no-ff --no-commit <devBranch>`（本机实测：冲突 exit=1、`CONFLICT` 输出、`git diff --name-only --diff-filter=U` 可列举未合并文件；`git merge --abort` 可完全恢复）。
   - 冲突处理：**`git merge --abort` 回滚到干净状态**，返回错误，错误信息含：冲突文件清单 + 「请 AI 修复合并后重跑 `rick tools release --merge-source`」+ 提示用 `--no-merge-source` 可只提升产物。
   - 无冲突：`git commit -m "chore(release): merge <devBranch> → <main> (version=<ver>)"` 提交合并；返回 `{merged:true, branch, main, commit, files:int}`。
   - 幂等：dev 分支未领先（无新提交）→ `merged:false, reason:"already up to date"`（不算失败）。
2. `internal/cmd/tools_release.go`：
   - 新 flag `--merge-source`（默认 **false**，保持既有语义不变；E2E 与 loop 流程里显式开启）与 `--no-merge-source`（显式覆盖配置时用）；若同时给了两者 → 报参数冲突错误。
   - 提升序列调整为：门禁 → 构建 → **（--merge-source 时）合并源码（失败即中止并报错，不继续提升）** → 记录 `.last` 回滚点 → 原子换链 → 前端投放 → 重启 → build_id 校验。
   - 回执打印合并结果（`RELEASE_MERGE merged=true branch=… commit=… files=…`）。
   - `--rollback` 语义不变（只回滚产物，不回滚源码；在 help 与回执里写清）。
3. 测试 `merge_test.go`（`t.TempDir()` 造两个真实 git 仓库/worktree，**禁止碰真实生产仓库**）：
   - 干净 + 无冲突 → 合并成功且 main 有新 merge commit；
   - **冲突 → 报错且生产仓库工作树恢复干净**（`status --porcelain` 为空，`f.txt` 回到合并前内容）；
   - 脏工作树 → 中止且无改动；
   - dev 无新提交 → `merged:false` 不算失败。
4. `tools_release_test.go`：flag 冲突（`--merge-source` + `--no-merge-source`）报错；默认不加 fuse 时**不调用** merge（用注入点断言）。

# 测试方法
go test ./internal/env/release/ ./internal/cmd/ -timeout 900s
go build ./... && go vet ./internal/env/release/ ./internal/cmd/
python3 .rick/jobs/job_36/plan/gates/gate13.py

# 上下文提示
- 本机实测事实（本 task 可直接依赖）：`git merge --no-ff --no-commit <branch>` 冲突时 **exit=1** 且 stdout/stderr 含 `CONFLICT`；未合并文件用 `git diff --name-only --diff-filter=U` 列出；`git merge --abort` 后 `git status --porcelain` 为空、文件内容回到合并前。
- 生产工作树当前**可能不干净**（历史 job 文件未提交）——错误信息要明确指向该原因，而不是笼统的 merge 失败。
- 与 task21 的既有语义：`.last` 回滚点、`ln -sfn`+`mv -T` 原子换链、前端 `dist.prev`、build_id 校验、`--detach` 自保——**全部保持不变**。
