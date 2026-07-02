# Dream Run: job_17

## 处理时间

2026-07-02

## 处理概述

- **Job 状态**: 已完成反思
- **数据来源**: act-path（task1~task4，共 4 tasks）；无 debug/ 目录，无 SUMMARY.md
- **act-path 信号**: task1~task4 各有 1 次报错，合计 4 次，均为 doing_check 状态未更新

## 反思发现

1. **doing_check tasks.json 状态未同步（task1-4 全部）**：4 个 task 代码提交后未执行 `mark_task_success.py`，doing_check 报 `task status != success`。此模式在多个 job 中高频出现（job_17 全部 4 个 task），是本次 dream 创建 `do-check-mark-success-loop` 的直接证据。
2. **本 job 工作内容**（从 act-path 推断）：job_17 涉及 skill 文件编写和 template 变量注入相关工作（act-path 中读取了 tdd-zh.md、tc.md、manager_test.go 等）；task1 26 次工具调用，较为复杂。
3. **跨 job 共性确认**：job_17 与 job_18/22 均出现相同 doing_check 失败模式，验证了 `do-check-mark-success-loop` 的必要性。

## 变更记录

### Loops 变更
- 新增: `do-check-mark-success-loop.md`（本 job task1-4 全部报错均为此模式，是主要证据来源）
- 升级: `example_loop.md` → 重命名为 `tdd-red-green-refactor-loop.md`，frontmatter name 同步更新
- 淘汰: `candidate_loop_1.md` → 移至 `deprecated/`（stub 文件，无实际内容）

### Skills 变更
- 新增: `test_script_practices_skill`（含 8 个陷阱清单，涵盖 job_17 中 binary 版本问题等）
- 新增: `mark_task_success_skill`（含 mark_task_success.py + build_rick.py 辅助脚本）
- 新增: `check_mechanism_skill`
- 新增: `verify_go_changes_skill`
- 新增: `template_injection_skill`
- 新增: `global_ref_sync_skill`
- 新增: `zero_retry_task_design_skill`
- 新增: `dag_task_decomposition_skill`
- 新增: `failure_feedback_skill`
- 淘汰: `candidate_skill_1.md` → 移至 `deprecated/`（stub 文件）

### 全局重构变更（本次 dream 专项）
- 删除: `.rick/wiki/`（所有 20 个 wiki 文件，知识已迁移到 skills）
- 删除: `.rick/tools/`（所有 7 个 Python 脚本，已迁移到对应 skill 目录）
- 删除: `.rick/SPEC.md`（知识已分散到各 skill 和 loop 文件）
- 删除: `.rick/OKR.md`（job 级 OKR 由 plan 阶段生成，全局 OKR 已废弃）

## 下次建议关注

1. 观察 `do-check-mark-success-loop` 在 job_23+ 中的实际触发情况，验证 trigger 条件是否足够清晰
2. `test_script_practices_skill` 中引用了 `.rick/skills/mark_task_success_skill/build_rick.py`，需确认 future task.py 改用新路径（不再引用 `.rick/tools/`）
