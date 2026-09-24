version=8dd1307-260925020038
rollback_point=/home/hadoop-recsys/.rick/releases/e72ce7b-260924021322
released=yes

# release 记录（job_43 · S3 演练 + S5 提升）

## S5 提升（2026-09-25 02:00，最终生效）

```
RICK_RELEASES_DIR=/home/hadoop-recsys/.rick/releases rick tools release --merge-source --detach --prod-repo /workdir/sunquan20/AI_CODING/rick
RELEASE_MERGE merged=true branch=dev/self-evolve main=main commit=f87dd0e files=53
RELEASE_RESTART pid=192045 health_ms=201 build_id=8dd1307-260925020038 matches=true stopped=[28268]
RELEASE_PROMOTE version=8dd1307-260925020038 prev=e72ce7b-260924021322 gc=[2cf884f-260924011743]
RELEASE_OK version=8dd1307-260925020038
```

- 生产 /api/health：status=ok, build_id=8dd1307-260925020038 == 本次 version ✓（G4）
- 回滚点：/home/hadoop-recsys/.rick/releases/.last = e72ce7b-260924021322（上一可用版）
- 版本目录：/home/hadoop-recsys/.rick/releases/8dd1307-260925020038（bin/rick 软链 → current → 该目录）

### 第一次尝试失败（已安全重试）

RELEASE_FAIL stage=build：mkdir <prod>/bin/releases/8dd1307-...: permission denied（bin/releases 属主 sankuai、mask r-x——release.go ReleasesRoot 注释记载的历史问题）。失败发生在 build 阶段（先于 merge/换链/重启），生产零影响。修复：RICK_RELEASES_DIR 指向用户可控目录（代码预留机制，与 02:14 的上一次成功 release 同一布局）。

## S3 演练（2026-09-24 20:30，dry-run 版本戳 8dd1307-260924190728）

```
rick tools release --dry-run --prod-repo /workdir/sunquan20/AI_CODING/rick
RELEASE_GATE pass=true tests=pass frontend=pass took=38s
RELEASE_DRYRUN target_bin=<prod>/bin/releases/8dd1307-260924190728/rick（dry-run 暂存路径）
RELEASE_DRYRUN prod_untouched=true（未写生产树、未换链、未重启）
```

dry-run 与最终提升的 version 时间戳不同属正常（version = <sha7>-<构建时间>，两次构建时刻不同；sha7 相同 = 同一 dev HEAD 8dd1307a）。
