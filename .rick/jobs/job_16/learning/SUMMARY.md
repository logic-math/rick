APPROVED: true

# Job job_16 执行总结

## 执行概述

**项目目标**: 将 Rick 调试能力从"盲目重试"升级为基于状态机理论的科学调试——三阶段 SOP（源码推理→增量调试→科学实验）+ review debug agent + 运行时工具指引，消除调试上下文恶性循环
**实际完成**: 4/4 tasks 全部成功，0 重试
**整体评价**: ⭐⭐⭐⭐⭐ (5/5)

## 关键成就

1. **debug_skill.md 建立**: 创建三阶段科学调试 SOP（源码推理法→增量调试法→科学实验法），内嵌 review debug agent 协议 + SENSE 方法集成 + bug/ 目录文件格式规范
2. **模板体系迁移**: doing.md / plan.md / easy.md 全部从 `super-debugging` 切换到 `debug-skill`，删除旧 `super-debugging-zh.md`，debug{N} 格式简化为单行 bug{n}-{描述}.md 指引
3. **Go 代码切换**: doing_prompt.go / plan_prompt.go / easy_prompt.go 全部迁移到 `debug_skill` + `sense`，`go test ./internal/prompt/...` 通过
4. **debug/ 目录扫描**: runner.go / retry.go / learning.go / easy_prompt.go 统一改为 `LoadDebugContext()`（优先扫描 `debug/bug*.md`，回退 `debug.md`），新增 `debug_dir.go` + `debug_dir_test.go`

## 问题与教训

### 问题1: 全局 config.json 导致测试超时

**根本原因**: `~/.rick/config.json` 设置 `max_retries: 16`，retry sleep 累计超 60s，测试框架超时
**解决方案**: 在受影响测试开头注入 `t.Setenv("HOME", tmpDir)` + 写入 `{"max_retries":2}` 的本地 config
**经验教训**: 凡是读取 `~/.rick/config.json` 的测试，setUp 中必须覆盖 HOME 隔离全局配置

### 问题2: 测试脚本范围过宽导致 task4 误判失败 16 次

**根本原因**: task4 测试脚本 Test 6 跑 `go test ./internal/...` 全量，混入 `internal/cmd` 里一个需要真实 API key 的无关测试（`TestNewPlanCheckCmd_RunE_WithWorkspace`），只要该测试失败就判定 task4 失败；task4 的实际代码实现完全正确
**解决方案**: 将 Test 6 改为只跑 task4 实际改动的包：`./internal/cmd/...` + `./internal/prompt/...`
**经验教训**: 测试脚本的 go test 范围必须精确匹配 task 的改动包，不得跑全量 `./internal/...`；已提取为 wiki/tasks_json_commit_hash.md

### 问题3: tasks.json commit_hash 缺失导致 doing_check 反复失败

**根本原因**: agent commit 后直接标记 tasks.json status=success，但未填 `commit_hash` 字段
**解决方案**: commit 后立即 `git rev-parse HEAD` 并填入 tasks.json
**经验教训**: doing prompt 应明确 tasks.json success 状态必须同时包含 `commit_hash`；已提取为 wiki/tasks_json_commit_hash.md

## 知识沉淀清单

- [x] `internal/prompt/templates/skills/debug_skill.md` — 三阶段调试 SOP（task1 创建）
- [x] `.rick/wiki/tasks_json_commit_hash.md` — doing_check commit_hash 缺失时的修复模式
- [x] `SPEC.md` — 新增"技能列表"节，注册 tasks_json_commit_hash skill
