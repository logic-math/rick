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
