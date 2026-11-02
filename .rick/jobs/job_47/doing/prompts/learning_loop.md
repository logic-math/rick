---
name: learning-loop
description: Learning 阶段 Loop；每个 Step 由 parent 用 runs.run 派发独立 worker child，Step 3-4 产出需人类反复审核，直至沉淀完成
---

## 全局目标

分析 job_47（super RFC 实现 + 13 轮迭代）的执行记录，沉淀可复用的 skills 和 loops，生成 SUMMARY.md。

**成功标准**：
- `rick tools learning_check job_47` pass
- `/Users/logic/workdir/AI_CODING/rick/.rick/jobs/job_47/learning/SUMMARY.md` 首行 `APPROVED: true`
- 人类已审核 skills/loops/domain 产出

## 素材位置（job_47 特化）

- plan/tasks：`/Users/logic/workdir/AI_CODING/rick/.rick/jobs/job_47/plan/task*.md`（58 个）
- tasks.json：`/Users/logic/workdir/AI_CODING/rick/.rick/jobs/job_47/doing/tasks.json`（58/58 success）
- RSI 证据：`/Users/logic/workdir/rick-dev/.rick/jobs/job_47/doing/rsi/`（dev-iterations.md 13 轮迭代+成本对比 / gates.md / release.md）
- git 轨迹：`git -C /Users/logic/workdir/rick-dev log --oneline --grep job_47`（37 提交）
- super 代码：`/Users/logic/workdir/rick-dev/super/`（superopt 包 + tests 521 全绿）
- env 运行数据：`/Users/logic/workdir/rick-dev/.rick/super/env_9`（在跑）/ env_11（已停，含分析）
- 关键交接文档：`/Users/logic/workdir/AI_CODING/rick/wiki/handover-2026-10-30-web-start-macos.md`（release 事故根因链）
- job_49（后续 job）已修 release P0 四项——job_47 的 release 失败教训由 job_49 沉淀部分知识；本 job learning 聚焦 super 模式本身的实现/迭代经验 + 自改进 loop 的诚实性教训

## 输出目标

- `/Users/logic/workdir/AI_CODING/rick/.rick/skills/{name}_skill/skill.md`
- `/Users/logic/workdir/AI_CODING/rick/.rick/loops/{name}-loop.md`
- `/Users/logic/workdir/AI_CODING/rick/.rick/jobs/job_47/learning/SUMMARY.md`
- `/Users/logic/workdir/AI_CODING/rick/.rick/domain/bugs.md` 等（追加）

## 工作流（同 job_46 模板：Step1-7，人类审核在 Step3-4 后）
