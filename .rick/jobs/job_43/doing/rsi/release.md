version=f9c8362-260929135749
rollback_point=/home/hadoop-recsys/.rick/releases/0d26c68-260929124639
released=no

# release 记录（job_43）

## 迭代 3 · S3 演练（2026-09-29 13:57，待 S4 人类批准）

```
RICK_RELEASES_DIR=/home/hadoop-recsys/.rick/releases rick tools release --dry-run --prod-repo /workdir/sunquan20/AI_CODING/rick
RELEASE_GATE pass=true tests=pass frontend=pass took=6s
RELEASE_DRYRUN version=f9c8362-260929135749 dist_files=12 sha256=670248916dce
RELEASE_DRYRUN prod_untouched=true
```

dev HEAD = f9c83628（task8 贴底间距修复）；dev 实例 build_id=f9c8362-260929135824、bundle index-CYCA3JSB.js（含 Footer spacer）。
按 bug2 教训：dev UI（8414）已起新构建供真渲染验收（发消息看间距）。

## 迭代 2 · 实际生效版本 0d26c68-260929124639（P0 热修复版）

3e35ec8 上线即 P0（虚拟化 Scroller 丢 props 全空白，见 doing/debug/bug2）→ 热修复 0d26c68 重新提升，用户验证恢复。迭代 2 批准 at=2026-09-25T12:35（human）。

## 迭代 1 · version=8dd1307-260925020038（2026-09-25 02:00，已完成）

merged f87dd0e files=53；第一次尝试 build 阶段 bin/releases 权限拒绝 → RICK_RELEASES_DIR 重试成功；S6 human 恢复。
