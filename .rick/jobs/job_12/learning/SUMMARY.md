APPROVED: true

# Job job_12 执行总结

## 执行概述

**项目目标**: 修复 skills/tools 分离实现（RFC-002）——将错误放置在 `.rick/skills/` 的 `.py` 工具脚本迁移到 `tools/`，重建 `.rick/skills/` 为 Markdown 技能说明书，并更新 learning 提示词模板防止未来重蹈覆辙。

**实际完成**: 3/4 tasks 零重试完成，task4 在修复 mock_agent 预存 bug 后通过。全部 4 个 KR 均已落地验证。

**整体评价**: ⭐⭐⭐⭐ (4星) — 核心目标全部达成，但 task4 遭遇了 mock_agent 预存 bug 导致标记为 failed，实际执行质量高。

## 关键成就

1. **RFC-002 完整落地**: `tools/` 含 5 个 `.py` 工具脚本，`.rick/skills/` 只含 `.md` 说明书，系统架构清晰分离
2. **Learning 模板修正**: `internal/prompt/templates/learning.md` 明确区分 tools（`.py` 确定性脚本）和 skills（`.md` 组合说明书），防止 AI 未来重蹈覆辙
3. **Mock Agent 深度修复**: 发现并修复 `tests/mock_agent/mock_agent.py` 和 `tools/mock_agent_testing.py` 中 4 个预存 bug，集成测试 15/15 通过
4. **Dry-run 改进**: 修复 `doing --dry-run` 始终展示 task1 的 bug，改为展示第一个非 success 任务

## 问题与教训

### 问题1: task4 被标记 failed（mock_agent 预存 bug）

**根本原因**: `tests/tools_integration_test.sh` 使用 `tests/mock_agent/mock_agent.py`，该文件有 4 个预存 bug（debug.md 格式错误、SUMMARY.md 缺少 `# Job` heading），导致集成测试失败。这些 bug 在 job_12 之前就存在，与本次任务无关。

**解决方案**: 分别修复 `tests/mock_agent/mock_agent.py` 和 `tools/mock_agent_testing.py`，使两者的 mock 输出符合 doing_check/learning_check 的格式要求。

**经验教训**: mock_agent 是测试基础设施的核心，其格式必须与 check 命令的期望严格对齐。当 check 命令格式规范变更时，mock_agent 需要同步更新。

### 问题2: dry-run 测试条件误判

**根本原因**: `task2.py` 测试条件 `".py" in output and "skills" in output.lower()` 检查全量 dry-run 输出，但 tools section 合法包含 `.py`，OKR/task 描述文本中也含 "skills" 字样，导致永远误报失败。

**解决方案**: 提取 `## 可用的项目 Skills` 至下一个 `##` 之间的内容，仅对该区间检查 `.py` 条目。

**经验教训**: 测试断言应精确匹配目标区域，避免全文搜索导致误判。特别是 dry-run 输出包含大量上下文文本，断言需要先定位 section 再检查内容。

### 问题3: task4.py 字段名不匹配

**根本原因**: task4.py 断言2 检查 `rick_bin` 字段，但 `build_and_get_rick_bin.py` 实际返回 `bin_path` 字段。任务描述中使用了非实际字段名。

**解决方案**: 修改断言为兼容两种字段名（`'bin_path' not in data and 'rick_bin' not in data`）。

**经验教训**: 测试脚本编写时应先运行工具查看实际输出格式，而非依赖任务描述中的字段名。

## 技术总结

### 关键技术决策

- **skills/tools 二元分离**: skills = 组合知识（`.md`），tools = 确定性执行（`.py`）。这个分离使 AI 在阅读 skills 时获得"何时/如何组合"的知识，在调用 tools 时获得"确定性执行能力"。
- **learning 模板作为规范载体**: 通过修改 learning 提示词模板来约束 AI 未来的产出行为，比文档说明更有约束力。
- **mock_agent 双文件格局**: `tests/mock_agent/mock_agent.py`（集成测试专用）和 `tools/mock_agent_testing.py`（通用 mock 工具）分别维护，需要保持同步。

### 知识沉淀清单

- [x] wiki/skills_tools_separation.md - skills/tools 分离机制工作原理
- [ ] skills/*.py - 无新增（本次 job 聚焦架构修复，无新工具需求）
- [x] SPEC.md - 新增 skills/tools 分离规范、mock_agent 同步要求
