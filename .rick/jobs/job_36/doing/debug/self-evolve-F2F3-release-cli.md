# F2 / F3：`rick tools release` 的两个真实路径缺陷

均由「在**被生产实例托管的**会话里跑 dry-run」这一真实场景暴露（纯单测与按实现分支构造的门禁都测不到）。

## F2：`--dry-run` 被自保检查误拦
- 现象：`RELEASE_WARN hosted_by_prod=true` 后直接 `RELEASE_FAIL stage=restart`，用户拿不到计划。
- 根因：`runRelease` 里自保检查（~150 行）在 `--dry-run` 分支（~195 行）**之前**；dry-run 不重启任何东西，拦截不适用。
- 修复：拦截条件加 `!opts.dryRun`（**警告仍如实打印**）；新增可注入点 `var hostedByProd = release.HostedByProd` 以便单测构造该场景；非 dry-run 的 `--yes`/`--detach`/setsid 语义不变。

## F3：dry-run 把构建产物写进了生产树
- 现象：`--dry-run` 帮助文本承诺「不动生产」，但 `plan.DryRun()` 仍构建到 `<prod-repo>/bin/releases/<ver>/{rick,dist}`。
- 根因：dry-run 复用了真实提升的构建路径，没有独立的暂存目录。
- 修复：新增 `Plan.StagingDir` / `Plan.BuildDir()`；dry-run 构建到 `os.MkdirTemp` 暂存目录并在结束时清理（清理失败打 `RELEASE_WARN dryrun_staging_left=…`，不静默）；输出预告 `target_bin/target_dist/staging/prod_untouched/cleanup`；
  **纵深防御**：`Promote()` 拒绝 `Staged` 产物（暂存产物绝不可能被真的提升）；`Result` 增 `target_bin/target_dist/staging_dir`。
  真实提升路径语义**不变**（仍落 `<prod>/bin/releases/<ver>/`，那是版本化+原子换链+回滚的载体）。

## 复现（修复后应全绿）
```
$ rick tools release --dry-run --dev-tree <dev> --prod-repo <prod> --prod-home <home> \
    --state-dir <home>/.rick --port 8413 --start-script <home>/start-web.sh
RELEASE_WARN hosted_by_prod=true (…)
RELEASE_GATE pass=true tests=pass frontend=pass took=21s
RELEASE_BUILD bin=/tmp/rick-release-dryrun-…/rick dist_files=12 sha256=…
RELEASE_DRYRUN target_bin=<prod>/bin/releases/<ver>/rick
RELEASE_DRYRUN target_dist=<prod>/bin/releases/<ver>/dist
RELEASE_DRYRUN staging=/tmp/… (临时；不影响生产树)
RELEASE_DRYRUN prod_untouched=true cleanup=ok
```
断言：`<prod>/bin/releases` 不存在、`<prod>/bin/rick` md5 不变、生产健康 200。

## 教训
「安全的演练命令」必须被门禁**显式断言为无副作用**（F3），且其**自保/拦截逻辑的判定顺序**属于契约的一部分（F2）。凡是只在「真实宿主形态」下才出现的分支，都要在 E2E 里覆盖。
