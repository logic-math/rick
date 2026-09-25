version=eeda69d-260925121631
rollback_point=/home/hadoop-recsys/.rick/releases/8dd1307-260925020038
released=no

# release 记录（job_43）

## 迭代 2 · S3 演练（2026-09-25 12:16，待 S4 人类批准）

```
RICK_RELEASES_DIR=/home/hadoop-recsys/.rick/releases rick tools release --dry-run --prod-repo /workdir/sunquan20/AI_CODING/rick
RELEASE_GATE pass=true tests=pass frontend=pass took=6s
RELEASE_DRYRUN version=eeda69d-260925121631 dist_files=12 sha256=76f1cfec2bec
RELEASE_DRYRUN prod_untouched=true
```

dev HEAD = eeda69d1（task5/6/7 全部提交）；dev 实例 build_id=eeda69d-260925121618、二进制 5.0.13、新 bundle index-KgY9Vklm.js 生效。

## 迭代 1 · S5 提升（2026-09-25 02:00，已完成）

version=8dd1307-260925020038 → 已提升成功（详见 git 历史：此文件迭代 1 版本）。
第一次尝试失败：build 阶段 mkdir <prod>/bin/releases permission denied（bin/releases 属主 sankuai）——用 RICK_RELEASES_DIR=/home/hadoop-recsys/.rick/releases 重试成功。
RELEASE_MERGE merged=true commit=f87dd0e files=53；生产 build_id 校验一致；S6 本会话挂起后 human 01:08 恢复。
回滚点 .last=e72ce7b-260924021322。
