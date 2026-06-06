# Dream Run: job_12

## 处理概述

- **处理时间**: 2026-06-05
- **Job 状态**: 已完成反思
- **数据来源**: debug.md（4 条目，含 2 个独立问题）+ tasks.json（4 tasks, all success）

## 反思发现

1. **两个 mock agent 文件各有独立 bug**（task4）：`tests/mock_agent/mock_agent.py` 和 `tools/mock_agent_testing.py` 各自的格式 bug 互不影响，修复时需分别处理。验证了 SPEC 中"Mock Agent 同步要求"条目的必要性
2. **全文搜索导致 section 误判**（task2）：`".py" in output and "skills" in output.lower()` 因 tools section 合法含 `.py` 而永远失败；新增 test_script_best_practices.md 陷阱6（section 精准断言）
3. **dry-run 始终展示 tasks[0]**（task2）：即使 task1 已 success，dry-run 仍展示 task1；修复为从 tasks.json 找第一个非 success 任务。已进入 SPEC 的 `rick doing --dry-run` 规范
4. **build_and_get_rick_bin.py 返回 JSON 非文本**（task4/debug1）：task4.py 期望纯文本路径，但工具返回 JSON；修复方案：调用方用 `json.loads()` 解析。已在 test_script_best_practices.md 陷阱1 注明"注意：返回 JSON"

## 变更记录

### Skills 变更
- 修改: `test_script_best_practices.md` — 陷阱1 新增 JSON 解析说明，新增陷阱6（section 精准断言）

### SPEC.md 变更
- 已有相关条目（Mock Agent 同步要求、测试断言精确性），本 job 是这些条目的补充证据

### Wiki 文档
- `skills_tools_separation.md` 已覆盖本 job 的 RFC-002 实现

## 下次建议关注
1. `tests/mock_agent/mock_agent.py` 和 `tools/mock_agent_testing.py` 同步问题在 job_12 和 job_11 均出现，值得加入 zero_retry_task_design.md 中的 pre-task 检查项
