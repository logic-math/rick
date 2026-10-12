APPROVED: true

# Job job_11 执行总结

## 执行概述

**项目目标**: 强化 Check 机制的正确性与强制集成，提升 Doing 失败反馈质量
**实际完成**: 3/3 任务全部完成，零重试率
**整体评价**: ⭐⭐⭐⭐⭐

## 关键成就

1. **Check 工具准确性修复（task3）**: 修复了 plan_check/doing_check/learning_check 三个工具与实际产出格式的不一致问题。plan_check 新增 OKR.md 存在性检查；doing_check 增强 debug.md 内容验证（非空 + 含 `## task` 记录）；learning_check 增强 SUMMARY.md 内容验证（非空 + 含 `# Job` 标题）。同步新增了 9 个单元测试覆盖新检查项。

2. **Check 机制强制集成（task1）**: 将 check 命令强制集成到 plan/doing/learning 三个阶段的 Agent 提示词模板中。plan.md 新增"强制验证步骤"章节，doing.md 新增第7条行为约束，learning.md Step 3 强化为"必须通过才能进入 Step 4"。同时修复了 plan_prompt.go 和 doing_prompt.go 中 `rick_bin_path` 和 `job_id` 变量未注入的问题，使模板中的 check 命令可被正确替换。

3. **失败信息传递优化（task2）**: 移除了 retry.go 中 500 字符硬截断限制，改为智能截断策略（保留最近 2 次失败记录，总长度上限 3000 字符）。同时修复了 runner.go 中 test 失败路径只传递 errors join 的问题，改为包含完整 testOutput（含 stderr/traceback）。

## 问题与教训

### 问题1: 测试脚本使用系统 rick 命令而非本地构建版本

**根本原因**: task3 的测试脚本最初使用系统安装的 `rick` 命令运行 check，但系统版本不包含本次新增的检查逻辑，导致测试无法验证新代码行为。

**解决方案**: 测试脚本先构建 `./bin/rick`，所有 `rick tools` 命令改用本地构建的二进制路径。

**经验教训**: 测试脚本应始终使用本地构建的二进制，而非依赖系统安装版本，否则测试验证的是旧代码而非新代码。

### 问题2: auto-fix 机制干扰测试预期

**根本原因**: 测试预期 `rick tools plan_check job_9` 因缺少 OKR.md 而失败，但上一轮 auto-fix 已经在 job_9/plan 创建了 OKR.md，且即使临时删除，plan_check 的 auto-fix 机制会自动恢复。

**解决方案**: 改为静态检查源码是否包含 OKR.md 检查逻辑，依赖 Go 单元测试（TestRunPlanCheck_MissingOKR）验证行为。

**经验教训**: 对于有 auto-fix 副作用的功能测试，应优先使用单元测试（隔离环境）而非集成测试（依赖真实文件系统状态）。

## 技术总结

### 关键技术决策

- **appendFailureFeedback 函数设计**: 采用"按分隔符分割 → 保留最近 N 条 → 超限从尾部截断并对齐行边界"的策略，既保留最新失败信息，又防止 prompt 膨胀。
- **模板变量注入位置**: `rick_bin_path` 和 `job_id` 在 `GeneratePlanPrompt` / `GenerateDoingPrompt` 函数内注入，而非在模板层面，保持了模板的通用性。
- **extractJobIDFromPath 函数**: 从路径末尾向前扫描 `job_` 前缀段，比正则表达式更健壮，适应路径格式变化。

### 知识沉淀清单
- [x] wiki/check_mechanism.md - Check 机制工作原理与强制集成
- [x] wiki/failure_feedback_propagation.md - Doing 重试循环的失败信息传递机制
- [x] skills/resolve_local_binary.py - 解析本地构建二进制路径（优先 ./bin/<name>，fallback 系统版）
- [x] skills/rick_tools_check_pattern.py - 更新：check_doing/check_learning 升级为内容验证模式（job_11 新增）
