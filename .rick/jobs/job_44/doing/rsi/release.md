# RSI 证据：release（S3 演练 / dry-run）

命令（dev 树 cwd）：`go run ./cmd/rick tools release --dry-run`
（本环境需 `RICK_PROD_REPO=/workdir/sunquan20/AI_CODING/rick`，见下方“环境前置”）

S3 结论：演练通过，`prod_untouched=true`（未写生产树、未换链、未重启；所有会话/任务不被打断）。

## 关键行（dry-run 真实输出）

```
RELEASE_PLAN prod_repo=/workdir/sunquan20/AI_CODING/rick dev_tree=/workdir/sunquan20/rick-dev state_dir=/home/hadoop-recsys/.rick port=8413 listen=0.0.0.0
RELEASE_PLAN start_script=/home/hadoop-recsys/.rick/start-web.sh current=1c3f085-260922193340 rollback_point=59ac918-260922143616 keep=3
RELEASE_GATE pass=true tests=pass frontend=pass took=38s
RELEASE_BUILD bin=/tmp/rick-release-dryrun-1185327023/57f599c-261010152801/rick dist_files=12 sha256=5a544bde2fea
RELEASE_DRYRUN version=57f599c-261010152801 dist_files=12 sha256=5a544bde2fea
RELEASE_DRYRUN target_bin=/workdir/sunquan20/AI_CODING/rick/bin/releases/57f599c-261010152801/rick
RELEASE_DRYRUN target_dist=/workdir/sunquan20/AI_CODING/rick/bin/releases/57f599c-261010152801/dist
RELEASE_DRYRUN staging=/tmp/rick-release-dryrun-1185327023/57f599c-261010152801 (临时；不影响生产树)
RELEASE_DRYRUN prod_untouched=true（未写生产树、未换链、未重启；所有会话/任务不会被打断）
RELEASE_DRYRUN cleanup=ok
```

## 演练计划（供 S4 人类确认）

```
version=57f599c-261010152801
rollback_point=59ac918-260922143616
prod_untouched=true
```

- 目标端口：8413（生产，重启期间优雅退出预算 ≤12s、健康就绪 ~0.2s）
- 影响面：提升会重启生产；所有在跑会话/后台 job 变为「挂起」，刷新页面后人工点「恢复继续」，**不自动续跑**。

## 环境前置（可复制）

本环境裸跑 `release --dry-run` 报 `无法确定生产仓库路径：请显式传 --prod-repo`，
故需显式提供生产仓库（gate3 的 [D] 亦依赖该环境变量）：

```
cd /workdir/sunquan20/rick-dev
RICK_PROD_REPO=/workdir/sunquan20/AI_CODING/rick go run ./cmd/rick tools release --dry-run
```

## S5 状态

S5（`release --merge-source --detach`）尚未执行 —— 需 S4 人类确认后由 parent 执行。
执行后本文件追加一行 `RELEASE_MERGE merged=true version=<sha7>-<ts> rollback_point=<path>`。

## S3 演练（v6，2026-10-12，dev HEAD d7686c4）

命令（dev 树 cwd）：
```
RICK_PROD_REPO=/workdir/sunquan20/AI_CODING/rick no_proxy=localhost,127.0.0.1 go run ./cmd/rick tools release --dry-run
```

关键行：
```
RELEASE_PLAN start_script=/home/hadoop-recsys/.rick/start-web.sh current=1c3f085-260922193340 rollback_point=59ac918-260922143616 keep=3
RELEASE_GATE pass=true tests=pass frontend=pass took=43s
RELEASE_DRYRUN version=d7686c4-261012210346 dist_files=12 sha256=772d3cc7dff7
RELEASE_DRYRUN prod_untouched=true（未写生产树、未换链、未重启；所有会话/任务不会被打断）
RELEASE_DRYRUN cleanup=ok
```

演练计划（供 S4 人类确认 → 已获批准）：
```
version=d7686c4-261012210346
prod_untouched=true
```

影响面：提升会重启生产 8413（优雅退出 ≤12s、就绪 ~0.2s）；所有在跑会话/后台 job 变「挂起」，需人工点「恢复继续」，**不自动续跑**。
**特殊**：本 job 的承载会话由生产 web（8413）承载，S5 会同时挂起它 —— S6/S7 需 human 在 UI 恢复后继续。

## 发布前置（merge 阻塞与处置）

生产树存在 untracked `.rick/jobs/job_44/`、`job_45/`（属主 sankuai，agent 无写权限）， dev 分支已跟踪同名路径 → `git merge` 会因「untracked working tree files would be overwritten」中止。处置：由 human 授权 agent 写权限（或代为 stash/移出）后继续 S5。
```

## S3 演练（发布版本对齐，2026-10-12，dev HEAD 16c493a）

```
RELEASE_PLAN start_script=/home/hadoop-recsys/.rick/start-web.sh current=1c3f085-260922193340 rollback_point=59ac918-260922143616 keep=3
RELEASE_GATE pass=true tests=pass frontend=pass took=5s
RELEASE_DRYRUN version=16c493a-261012211207 dist_files=12 sha256=64ba98ed3365
RELEASE_DRYRUN prod_untouched=true（未写生产树、未换链、未重启；所有会话/任务不会被打断）
```

## S5 发布（进行中）

命令：`RICK_PROD_REPO=/workdir/sunquan20/AI_CODING/rick <built-bin> tools release --merge-source --detach`
载荷定界：已排除 `.rick/jobs/job_44`、`.rick/jobs/job_45`（见 commit 16c493a）；生产树这两条路径的 human 副本零改动。

## S5 第一次尝试：失败（权限阻塞，未留中间态）

```
RELEASE_GATE pass=true tests=pass frontend=pass took=24s
RELEASE_FAIL stage=build detail=build: mkdir /workdir/sunquan20/AI_CODING/rick/bin/releases/3c86908-261012211221: permission denied
```

现场核对（失败后立即探测）：
- 生产 main HEAD 仍为 `22f186c`（**未产生合并提交**）；`.git/MERGE_HEAD` 不存在；tracked 工作树干净 → **无中间态**
- 生产 8413 `/api/health` → `build_id="dev"`（仍是重启前旧实例）→ **未换链、未重启，无损害**
- 根因：`<prod>/bin/releases/` 属主 sankuai、ACL `mask::r-x` → agent（hadoop-recsys）无写权限，无法 mkdir 版本目录；`setfacl` 非属主被拒（Operation not permitted），`sudo` 需密码。
- 发布要素可写性体检：`bin/releases` ❌ / `bin/releases/.last` ❌；`bin/rick` ✅ / `~/.rick/start-web.sh` ✅ / `~/.rick/web` ✅ / `release.log|state.json` ✅ —— 仅一项需授权。

所需授权（human 执行一条命令）：
```
setfacl -R -m u:hadoop-recsys:rwx -m m::rwx /workdir/sunquan20/AI_CODING/rick/bin/releases
```
授权后重跑 S5 即可（release 会重新 gate+build+换链+重启）。
