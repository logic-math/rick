# Dream Run: job_22

## 处理时间

2026-07-02

## 处理概述

- **Job 状态**: 已完成反思
- **数据来源**: act-path（task1~task9，共 9 tasks，无 task8）；无 debug/ 目录，无 SUMMARY.md
- **act-path 信号**: task1/2/3 零报错；task4~7/9 共 3 次报错

## 反思发现

1. **job_22 是架构重构 job**：act-path 显示 task1 创建了 `.rick/loops/` 和 `.rick/skills/` 目录（`mkdir -p`），task2 创建了 `candidate_loop_1.md` 和 `example_loop.md`。这正是 loops/skills 新架构的起点。本次 dream 将这些"candidate"升级为正式 skill 目录结构。
2. **RFC-001 驱动**：task1 首先读取 `RFC-001-context-architecture.md`，确认了 RFC 前置读取模式的普遍性。
3. **doing.md 模板变量注入（task4/5）**：将 `{{job_okr_content}}` 替换为 `{{loops_context}}`，属于全局变量名迁移，3 次报错均在模板变量同步阶段。`global_ref_sync_skill` 和 `template_injection_skill` 的触发场景直接来源于此。
4. **task1~3 零报错**：创建静态目录/文件 + README 任务全部一次成功，验证了零重试任务设计的"单一职责+明确路径"原则。
5. **embed.FS 涉及模板变更后必须 build**：task5/6/7 涉及 doing.md/plan.md 模板修改，每次修改后需 `./scripts/build.sh` 才能生效，这在 `verify_go_changes_skill` 中已明确。

## 变更记录

（与 job_17 log 相同，本次 dream 批量处理，变更集中记录在 job_17 log 中）

### Loops 变更
- `do-check-mark-success-loop`：task4~7/9 的 3 次 doing_check 报错是此 loop 的额外证据
- `tdd-red-green-refactor-loop`：job_22 的 task 编写中有 Go TDD 场景，loop 触发场景得到验证

### Skills 变更
- `template_injection_skill`：task4/5 的 loops_context 变量注入是核心触发场景
- `global_ref_sync_skill`：task4 的全局变量名替换是典型应用
- `verify_go_changes_skill`：task5/6/7 的 embed.FS → build → dry-run 验证链是来源

## 下次建议关注

1. `loops_context` 变量注入（doing.md 模板的 loops/skills 感知）在后续 job 中的实际效果
2. dream_prompt.md 中"可用的项目 Loops"列表需要保持与 `.rick/loops/` 目录同步（当前靠程序自动扫描）
3. 本次全量删除 wiki/tools/SPEC.md/OKR.md 后，下一个 job 的 doing prompt 是否还有残留引用
