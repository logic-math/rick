# RSI 证据：release

version=47befe6-261030233503
rollback_point=4fa5dd8-261030115302
released=yes

（头部三行是 rsi_check 的机器契约——version= 取首个匹配行做生产 build_id 校验，必须置首。）

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

## S5 提升（2026-10-30 23:35，human 已批准）

```
RELEASE_MERGE merged=true branch=dev/self-evolve main=main commit=de906bd files=7
RELEASE_RESTART pid=89800 health_ms=202 build_id=47befe6-261030233503 matches=true stopped=[46274] skipped=[]
RELEASE_PROMOTE version=47befe6-261030233503 prev=4fa5dd8-261030115302 bin=.../bin/releases/47befe6-261030233503/rick overlay=/Users/logic/.rick/web/dist dist_files=12
RELEASE_OK version=47befe6-261030233503
```

- version=47befe6-261030233503 rollback_point=4fa5dd8-261030115302（Mach-O arm64 可执行——坏 ELF 回滚点 eb8048c 消亡）
- stopped=[46274] skipped=[]：真实生产提升中 stopProd 干净收割旧实例（P0-3 实战验证）
- 手工回滚步骤：`rick tools release --rollback --yes`（回滚点已校验可执行）；或手工
  `cd /Users/logic/workdir/AI_CODING/rick/bin/releases && ln -sfn 4fa5dd8-261030115302 current && /Users/logic/.rick/start-web.sh`

## S5 后验收

- /api/health → {"status":"ok","build_id":"47befe6-261030233503"} ✅
- 覆盖层 assets 与发布包 diff 为空（P1-5 达成）✅
- 首页 HTTP 200 ✅
- start-web.sh 加固版验收：冷启动→daemon→health 校验 ✅；幂等 ✅
