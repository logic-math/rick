# bug1：ResumeEasyIn 无条件覆盖非 easy 作业的 tasks.json（job 数据破坏）

**发现时间**：2026-09-24 19:40（job_43 门禁「生产零触碰」断言捕获） | **状态**：已应急恢复，**未修复**（超出 job_43 范围，记遗留）

## 症状

生产树 `.rick/jobs/job_36/doing/tasks.json`（29 个 task 的完整成功史，tracked）被 `easy_session` stub（449B，status=success，created_at=2026-09-24T02:31:54）覆盖。

## 根因（Phase 1-3）

`internal/handler/easy.go:144` `ResumeEasyIn` 在 resume 会话结束后调用 `writeEasyTasksJSON(doingDir)`——**无条件**把该 job 的 tasks.json 重写为合成 easy_session stub。当用户 `rick easy --resume <非 easy 作业 id>`（如 job_36 这种 plan/doing 多 task 作业）时，真实任务史被 stub 覆盖。

时间线：2026-09-24 02:31:54 一次 `rick easy --resume job_36` 执行，pi 会话立即退出，stub 覆盖落盘。job_36 真实 tasks.json（29 task success + commit_hash）在 git HEAD 中完好。

## 应急处置

`git checkout -- .rick/jobs/job_36/doing/tasks.json` 恢复（RSI loop「main 上脏 job 数据先修干净再 release」）。

## 修复方向（遗留，下轮候选）

`writeEasyTasksJSON` 改为**条件写入**：仅当 tasks.json 不存在或本身是 easy_session stub 时写入；或 ResumeEasyIn 对含多 task 的作业跳过重写。测试：resume 一个 plan 型作业断言 tasks.json 原样。

## 关联

- gate1/gate2 的「生产零触碰」断言（`git status --porcelain -uno`）正是捕获此问题的机制——判别力实证
- job_36 曾因 F6（level_complete 写回时序）手工补记过 tasks.json（90a6f6f）——本 bug 属同族「job 数据被非合并路径改写」
