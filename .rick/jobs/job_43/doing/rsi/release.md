version=0d26c68-260929124639
rollback_point=/home/hadoop-recsys/.rick/releases/8dd1307-260925020038
released=yes

# release 记录（job_43）

## 迭代 2 · S5 提升（2026-09-29，含 P0 热修复）

两次提升（detach 进程跨机器休眠期完成）：

**① 3e35ec8-260929093943**（S3 演练版本 eeda69d 的同源后续构建）：merged commit 321b5b5。**上线即 P0**：MessageList 虚拟化自定义 Scroller 丢弃 Virtuoso props → 所有会话消息区空白（用户立刻报告；根因/修复/教训见 doing/debug/bug2-virtuoso-scroller-props-p0.md）。

**② 0d26c68-260929124639**（P0 热修复，其他 agent 完成）：fix(web) 自定义 Scroller 改 `{ style, children, ...rest }` 全量转发。生产 build_id 校验一致，用户验证消息恢复。

- 迭代 2 的 S4 人类批准：at=2026-09-25T12:35:00+08:00（原批准对象 eeda69d；P0 修复版 0d26c68 为其最小 diff 热修复，同属迭代 2 交付）
- 回滚点 .last 链：8dd1307-260925020038（迭代 1 生效版）
- 版本目录：/home/hadoop-recsys/.rick/releases/0d26c68-260929124639

## 迭代 2 · S3 演练（2026-09-25 12:16）

RELEASE_GATE pass=true tests=pass frontend=pass took=6s；version=eeda69d-260925121631；prod_untouched=true

## 迭代 1 · S5 提升（2026-09-25 02:00，已完成）

version=8dd1307-260925020038：merged f87dd0e files=53，生产校验一致，S6 挂起后 human 恢复。
第一次尝试失败于 build 阶段（bin/releases 属主 sankuai 权限拒绝）→ RICK_RELEASES_DIR=/home/hadoop-recsys/.rick/releases 重试成功。
