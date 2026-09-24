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
