APPROVED: true

# Job job_17 执行总结

## 执行概述

**项目目标**: 清除 RFC-refactor-2 和 RFC-refactor-go-codebase 记录的代码欠债：删除死 skill 文件、消除 frontmatter 解析重复、移除 callClaudeCodeCLI 内部重复函数、清理 tools merge 残留文档引用。

**实际完成**: 4/4 任务全部 success，0 重试，覆盖 KR1-KR4 全部目标。

**整体评价**: ⭐⭐⭐⭐⭐

## 关键成就

1. **死代码清零**: 删除 `tc.md`/`tdd.md`/`tdd/testing-anti-patterns.md` 三个死 skill 文件，`tc.md` 内容无损合并进 `tdd-zh.md`，所有测试通过。
2. **frontmatter 解析统一**: 提取 `internal/parser.ExtractBugFrontmatter`，消除 `debug_dir.go` 和 `easy_prompt.go` 的重复实现，新增独立测试。
3. **callClaudeCodeCLI 接口收敛**: 用 variadic `extraArgs ...string` 统一签名，删除 `callClaudeCodeCLIEasy`/`callClaudeCodeCLIResume` 两个冗余函数，功能完整保留。
4. **文档与实现对齐**: SPEC.md / wiki / OKR.md 中所有 `tools merge`/`RFC-005`/`rick easy 独立命令` 引用全部清除或更新为人工合并工作流描述。

## 问题与教训

### 问题1: 每个 task 末尾 doing_check 均因 tasks.json 未更新失败

**根本原因**: agent 在代码 commit 之后直接运行 doing_check，而 doing_check 要求 tasks.json 中对应 task 的 `status` 为 "success"，此时 tasks.json 仍是 "running"。

**解决方案**: 每次都是：read tasks.json → 手动编辑 status/commit_hash → git commit tasks.json → doing_check。重复 4 次，每次浪费 3-4 个工具调用。

**经验教训**: 代码 commit 完成后应立即用 `mark_task_success.py` 工具更新 tasks.json，再做第二次 commit，最后才运行 doing_check，可保证一次通过。

## 知识沉淀清单

- [x] `.rick/tools/mark_task_success.py` — 自动更新 tasks.json status=success + commit_hash 工具
- [x] `.rick/wiki/mark_task_success_workflow.md` — tasks.json 两阶段提交工作流说明
- [x] `.rick/SPEC.md` — 技能列表新增 mark-task-success 条目
