# RSI 证据：release

通过标准：含 `version=<sha7>-<ts>` 与 `rollback_point=`。

## S3 演练（dry-run，2026-10-30 22:40）

```
RELEASE_PLAN prod_repo=/Users/logic/workdir/AI_CODING/rick dev_tree=/Users/logic/workdir/rick-dev state_dir=/Users/logic/.rick port=6137 listen=0.0.0.0
RELEASE_PLAN start_script=/Users/logic/.rick/start-web.sh current=4fa5dd8-261030115302 rollback_point=eb8048c-261012235415 keep=3
RELEASE_DRYRUN version=4fa5dd8-261030222810 dist_files=12 sha256=f65504eaff69
RELEASE_DRYRUN target_bin=/Users/logic/workdir/AI_CODING/rick/bin/releases/4fa5dd8-261030222810/rick
RELEASE_DRYRUN target_dist=/Users/logic/workdir/AI_CODING/rick/bin/releases/4fa5dd8-261030222810/dist
RELEASE_DRYRUN prod_untouched=true（未写生产树、未换链、未重启）
RELEASE_DRYRUN cleanup=ok
RELEASE_WARN hosted_by_prod=true（祖先进程工作目录是生产工作树——S5 用 --detach 脱离执行）
```

注：rollback_point 显示 eb8048c（当前 .last 坏值）——S5 promote 时会把 .last 更新为本次 promote 前的 current（4fa5dd8-261030115302，可执行好版本），坏回滚点随之消亡。

## S5 提升

（待 human 批准后填写）
