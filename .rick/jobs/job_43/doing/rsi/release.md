# release 记录（job_43）

## 迭代 5 · S3 演练（2026-09-29 21:37，待 S4 人类批准）

RELEASE_GATE pass=true tests=pass frontend=pass took=38s；version=b946d99-260929213759；prod_untouched=true。
dev 实例 build_id=b946d99-260929213632：后端白名单实测（注册内 workspace 绝对路径 200 / 未注册 invalid_path）；生产实例预期注入 8 workspaces + ~/.rick + rick-dev 根（start-web.sh cwd 推导已核算）。工具箱入口可在 dev UI 8414 验收。

## 迭代 4 · S3 演练（2026-09-29 15:02，待 S4 人类批准）

RELEASE_GATE pass=true tests=pass frontend=pass took=6s；version=4b25f12-260929163400；prod_untouched=true。
dev 实例 build_id=fb01dde-260929150223、bundle index-BjBTWg3K.js（task9 toggle 修复），dev UI 8414 可真渲染验收。

## 迭代 3 · S3 演练（2026-09-29 13:57，待 S4 人类批准）

```
RICK_RELEASES_DIR=/home/hadoop-recsys/.rick/releases rick tools release --dry-run --prod-repo /workdir/sunquan20/AI_CODING/rick
RELEASE_GATE pass=true tests=pass frontend=pass took=6s
RELEASE_DRYRUN version=69dc824-260929140915 dist_files=12 sha256=670248916dce
RELEASE_DRYRUN prod_untouched=true
```

dev HEAD = f9c83628（task8 贴底间距修复）；dev 实例 build_id=f9c8362-260929135824、bundle index-CYCA3JSB.js（含 Footer spacer）。
按 bug2 教训：dev UI（8414）已起新构建供真渲染验收（发消息看间距）。

## 迭代 2 · 实际生效版本 0d26c68-260929124639（P0 热修复版）

3e35ec8 上线即 P0（虚拟化 Scroller 丢 props 全空白，见 doing/debug/bug2）→ 热修复 0d26c68 重新提升，用户验证恢复。迭代 2 批准 at=2026-09-25T12:35（human）。

## 迭代 1 · version=8dd1307-260925020038（2026-09-25 02:00，已完成）

merged f87dd0e files=53；第一次尝试 build 阶段 bin/releases 权限拒绝 → RICK_RELEASES_DIR 重试成功；S6 human 恢复。

## 迭代 3 · S5 提升（2026-09-29 14:09，已完成）

- 批准 at=2026-09-29T14:10（human，dev UI 真渲染验收后）
- 实际构建版本 69dc824-260929140915（批准留痕 commit 推进了 dev HEAD，构建取最新 dev 分支头——含 task8 + 证据提交，sha7 与 dry-run 不同属正常）
- RELEASE_MERGE merged=true commit=156a9cb files=29；RESTART health_ms=201 build_id_matches=true；gc=[3e35ec8-260929093943]
- 生产 /api/health：build_id=69dc824-260929140915 == 本次 version ✓
- S6：本承载会话（e7dbebd0）随重启挂起 → human 点「恢复继续」接续

## 迭代 4 · S5 提升（2026-09-29 16:34，已完成）

- 实际构建版本 4b25f12-260929163400（批准留痕提交推进 dev HEAD，构建取最新——含 task9 + 证据，与 dry-run sha7 不同属正常）
- RELEASE_MERGE merged=true commit=15881a6 files=14；RESTART health_ms=201 build_id_matches=true；gc=[0d26c68-260929124639]
- 生产 /api/health：build_id=4b25f12-260929163400 == 本次 version ✓
- S6：本承载会话（e7dbebd0）挂起 → human 点「恢复继续」接续
