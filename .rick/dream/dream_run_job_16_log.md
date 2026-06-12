# Dream Run: job_16

## 处理概述

- **处理时间**: 2026-06-12
- **Job 状态**: 已完成反思
- **数据来源**: debug.md（1 条目 + SUMMARY.md 3 问题）+ tasks.json（4 tasks, all success）+ act-path（task1/task4）

## 反思发现

1. **全局 config 污染测试超时（debug1）**：`~/.rick/config.json` 的 `max_retries:16` 导致 retry sleep 累计 = 1+2+...+15 = 120s，超过 60s timeout；修复：测试开头注入 `t.Setenv("HOME", tmpDir)` + 写入 `{"max_retries":2}` 的本地 config。新增至 test_script_best_practices.md 陷阱8
2. **go test 范围过宽导致 task 误判（SUMMARY 问题2）**：`go test ./internal/...` 全量混入依赖真实 API key 的无关测试；修复：精确匹配改动包。新增至 SPEC.md 开发规范
3. **commit_hash 缺失导致 doing_check 失败（SUMMARY 问题3）**：act-path task1 显示 2 次 doing_check 错误均为 commit_hash 字段缺失；已有 wiki/tasks_json_commit_hash.md 覆盖
4. **core_skills_injection.md 注入表与源码严重不符（subagent_5 发现）**：plan 行缺少 write_spec/tdd-zh/testing-anti-patterns-zh；doing 行 skill 名称错误（tdd vs tdd-zh）；dream 行缺少 source-context-consistency/refactor-rfc；已全面修正
5. **debug/ 目录机制未在 SPEC 路径约定中描述**：job_16 引入的 `LoadDebugContext()` 和 `doing/debug/bug*.md` 是核心机制，已补充至 SPEC.md

## 变更记录

### Skills 变更
- 新增: 无
- 修改: `test_script_best_practices.md` — 新增陷阱8（全局 config 污染 + retry 累计延迟因果链）
- 删除: 无

### SPEC.md 变更
- 新增「doing/debug/bug*.md 路径约定」（LoadDebugContext 回退机制）
- 新增「go test 范围精确性」开发规范条目
- 修正 dream `--dry-run` 描述（补充 source-context-consistency、refactor-rfc 两个 skill）

### Wiki 文档
- `core_skills_injection.md`：全面修正注入映射表（plan/doing/easy/dream 4 阶段），更新文件树（补充 write_spec.md、tdd-zh.md、testing-anti-patterns-zh.md）
- `core_skills_injection.md`：更新验证示例从 super-debugging → debug-skill

### RFC
- 新增: `RFC-refactor-2.md`（P1: debug_dir.go 的 extractBugFrontmatter 逻辑在 easy_prompt.go 中重复实现，建议提取到 internal/parser/）

## 下次建议关注

1. RFC-refactor-2 的 `extractBugFrontmatter` 重复逻辑 — 建议提取到 `internal/parser/frontmatter.go`，消除循环依赖导致的代码复制
2. TODO 2026-08 标记的 debug.md fallback 路径 — 4 处兼容代码，时到可统一清理
3. **P0 合并+清理**：`tc.md` 四要素内容合并进 `tdd-zh.md` 后删除；`tdd.md`、`tdd/testing-anti-patterns.md` 英文版直接 `git rm`；已记录至 RFC-refactor-2（§2.1 含具体合并方案）
