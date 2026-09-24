# RSI 证据：dev-iterations

通过标准：至少记录一条构建指纹（`rick.dev.<sha7>-<ts>` 或 `build_id=<sha7>-<ts>`）。
示例：`DEV_UP bin=... build_id=c539c60-260921192330 port=8414`

<!-- TODO: 填写本文件后再跑 rick tools rsi_check（残留本标记即视为未填写） -->


## 2026-09-24 19:45 · S0/S1 环境准备（parent）

- dev 环境：`RICK_DEV_HOME=/workdir/sunquan20/rick-dev-home-hadoop`（共享 rick-dev-home 为 sankuai 所有且 ACL mask=--- 不可写，用 EnvHome 重定向，dev 树 /workdir/sunquan20/rick-dev 不受影响）
- `rick tools dev-web init` + `up` → DEV_STATUS state=up pid=143853 port=8414 build_id=e72ce7b-260924182400 fingerprint=ok
- job_43 数据迁移：生产树 .rick/jobs/job_43 → dev 树（RSI「job 数据只写 dev 树」）；生产树残留 untracked job_43 待 release 前清除
- 生产树污染处置：job_36/doing/tasks.json 被 ResumeEasyIn bug 覆盖为 easy stub（02:31:54），已 `git checkout --` 恢复（29 task 完好）；bug 详情见 doing/debug/bug1-resume-easy-overwrites-tasks-json.md（遗留下轮）
- grilling_gate ✅（3 层设计树 + 3 份 research 简报）；pipeline_gate ✅（4 task / 2 层，同层写域不相交）
- gate1/gate2 RED 判别力实证：exit 1，13 项断言全红（task1-3 改动未落盘）+ 生产零触碰断言捕获 job_36 污染（已修复）

## 2026-09-24 20:00 · 第 1 层完成（task1+task2+task3 并行 → gate1 绿 → commit）

- 3 worker 并行（glm-5.3）：task1 Go 会话持久化（IdleDisabled 哨兵 + web_max_active 默认 64 + DefaultMaxActive=64 + VERSION 5.0.12 + 新测试 IdleDisabledNegative/Defaults64/ResolveSupervisorConfig）；task2 ChatInput（Ctrl+J 光标插入/enterMode localStorage/移动端换行按钮/空行守卫）；task3 导航（key={session.id}/AppShell 滚动重置/工作区行点击 navigate+chevron stopPropagation）
- gate1 首跑 2 红：断言写死「负值字面量」而 worker 用了更优的 IdleDisabled 哨兵（task1.md KR 原文即允许「或等价显式禁用语义」）——按 KR 语义修正 gate pattern（未削弱断言，13 项全在），复跑绿
- build_id：e72ce7b-260924182400（dev 实例未重启，前端改动待 task4 后统一 restart 验证）
- commit：f76d0e98 feat(layer): task1+task2+task3
