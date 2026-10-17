# RSI 证据：release

通过标准：含 `version=<sha7>-<ts>` 与回滚点（`rollback_point=` 或 `released=yes`）。
示例：
```
version=c539c60-260921192330
rollback_point=bin/releases/a60035a-260921195959
released=yes
```

<!-- TODO: 填写本文件后再跑 rick tools rsi_check（残留本标记即视为未填写） -->


## S3 演练（2026-10-18）
RELEASE_DRYRUN version=e4246ff-261018033233 target_bin=/Users/logic/workdir/AI_CODING/rick/bin/releases/e4246ff-261018033233/rick target_dist=.../e4246ff-261018033233/dist staging=/var/folders/.../rick-release-dryrun-2549987771/e4246ff-261018033233（临时） prod_untouched=true cleanup=ok gate_pass=true(tests+frontend, 48s)

### 演练前修复的 release 链 macOS 跨机缺陷（6 处，S3 演练驱动发现，commits c446e7c/08ebd8d/e4246ff + ea9dfc6 前置）
1. release gate 内层 go test 超时 900s 硬编码→随 GateTimeout（3600s）对齐
2. go test ./... 加 -p 1（包间串行防真构建用例与时敏包并发争 CPU flaky）
3. git merge --no-ff --no-commit 在 macOS 要求 committer 身份→身份 -c 注入前置到 merge 命令
4. ownedByProd 路径断言 /var↔/private/var 双形失配→EvalSymlinks 归一（stopProd 误跳过→旧进程残留→build_id 不一致）
5. darwin Alive 漏检 os.ErrProcessDone（死 pid 误判存活）
6. findProdPIDs/findDevPIDs 全进程逐 pid lsof 分钟级挂死（TestReleaseCmd* 全挂根因）→procfs.Snapshot 批量（252ms/907 进程）
7. （测试侧）realHomeOf 只读 /etc/passwd→补 user.LookupId；devweb Layout 测试路径归一；supervisor 测试路径归一
