APPROVED: true
# Job job_5 执行总结

## 执行概述

**项目目标**: 为 Rick CLI 构建完整的工具链支撑体系——包括 plan/doing/learning 三阶段的校验命令、知识合并流程、以及端到端集成测试框架。

**实际完成**: 7/7 任务，零重试，全部 success。

**整体评价**: ⭐⭐⭐⭐⭐

## 关键成就

1. **`rick tools` 子命令框架**：新增 `plan_check` / `doing_check` / `learning_check` / `merge` 四个子命令，覆盖 Plan→Doing→Learning 全流程的自动校验与知识合并。

2. **强制工作日志 debug.md**：将 doing.md 模板中对 debug.md 的描述从"可选"改为硬约束，确保每次任务执行都留下可追溯的工作记录。

3. **Learning 四类产出规范**：重写 learning.md 模板，明确 Wiki（面向人类）/ Skills（Python 脚本）/ OKR.md（完整新版本）/ SPEC.md（完整新版本）四类产出的格式与生成规则。

4. **Skills 注入到 Doing 提示词**：在 `doing_prompt.go` 中自动读取 `.rick/skills/` 并生成"可用的项目 Skills"表格注入提示词，让 AI agent 在执行任务时能感知并复用已有技能。

5. **通用化 project_name**：移除硬编码的 "Rick CLI" 回退字符串，使 Rick 真正成为通用 AI Coding Framework，可用于任意项目。

6. **mock_agent + 端到端集成测试**：实现 `tests/mock_agent/mock_agent.py`（11 种场景）和 `tests/tools_integration_test.sh`（15 个集成测试用例），配合单元测试将核心模块覆盖率提升至 71-76%。

## 问题与教训

### 问题1：autoFix 默认启用导致测试逻辑冲突

**根本原因**: `doing_check` 初始实现中 `autoFix` 默认启用——当测试期望校验失败时，Claude 自动修复后返回成功，导致测试断言失败。

**解决方案**: 将 `autoFix` 改为 opt-in（`--auto-fix` 标志），默认不启用。

**经验教训**: 自动修复行为应为显式触发，不应是默认行为。自动化工具的"修复"能力会干扰测试的确定性。

### 问题2：测试脚本路径计算错误

**根本原因**: `task5.py` 计算 `project_root` 时少了一层 `dirname`（未跳过 `.rick` 目录层级）。

**解决方案**: 将 `dirname` 调用从 5 次改为 6 次。

**经验教训**: 测试脚本的路径计算需要明确数出每一层目录，测试文件位于 `.rick/jobs/job_N/doing/tests/` 需要 6 次 dirname 才能到达项目根目录。

### 问题3：doing.md 模板中的否定引用被误判为软性表述

**根本原因**: 测试脚本用 substring 匹配检测"软性表述"，而模板中"这不是'遇到问题才记录'的可选项"这句话虽然是否定语义，但包含了被检测的关键词。

**解决方案**: 将该行改写为不含被检测关键词的等效表述。

**经验教训**: 基于 substring 的内容校验容易产生误报，需要特别注意否定语境。

## 技术总结

### 关键技术决策

- **tools 子命令模式**: 使用 Cobra 父命令 + 子命令结构，`tools.go` 作为容器，各 check 命令独立文件，易于扩展。
- **auto-fix 设计**: check 命令默认只报告问题，`--auto-fix` 标志才触发 Claude 修复，保持工具的确定性和可测试性。
- **merge 流程用 Git 分支隔离**: `rick tools merge` 在 `learning/job_N` 分支上执行合并，人工审核后才 `git merge --no-ff`，保持主分支干净。
- **Skills 注入机制**: `doing_prompt.go` 在生成提示词时动态读取 `.rick/skills/`，无需配置，自动感知项目已有技能。
- **OKR/SPEC 完整版本替换**: 新 learning 模板要求生成完整的 OKR.md 和 SPEC.md（而非差异 patch），通过顶部注释说明变更，便于 `rick tools merge` 直接覆盖。

### 使用的新技术/模式

- **Python argparse + JSON output 标准**: Skills 脚本统一使用 argparse 解析参数、JSON 输出结果，便于 AI agent 调用和解析。
- **`skipIfNoClaude` guard**: executor 测试中引入 `skipIfNoClaude` 函数，在没有 Claude binary 的 CI 环境中跳过 Claude 依赖测试，提升 CI 可靠性。
- **mock_agent 多场景模拟**: 用 Python 实现 mock AI agent，支持 11 种预定义场景（成功/失败/超时/格式错误），用于集成测试时替代真实 Claude 调用。

### 代码质量指标

- 测试覆盖率: internal/cmd 71.4%，internal/executor 72.9%，internal/prompt 76.1%
- 新增代码: ~4,400 行（含测试）
- 技术债务: `tools_merge.go` 中 Git 操作直接调用 shell 命令（`exec.Command("git", ...)`），未使用 `internal/git` 包，存在一定不一致性

## 建议和改进

### 针对下一个 Job

1. 将 `tools_merge.go` 中的 Git shell 调用迁移到 `internal/git` 包，统一 Git 操作层
2. 为 `rick tools merge` 增加 dry-run 模式，让人工审核更方便
3. Skills 脚本的 Python 语法检查（`learning_check`）可扩展为实际执行验证（`--dry-run` 调用）

### 流程改进

1. **debug.md 作为强制工作日志**已验证有效，建议在 plan 阶段也引入类似的强制记录机制
2. `--auto-fix` 标志的设计模式（默认 off，显式 on）应作为所有 check 命令的统一规范

### 工具和技术

1. `mock_agent.py` 的场景覆盖可进一步扩展，支持随机延迟和部分成功场景
2. `tools_integration_test.sh` 可加入并发测试场景，验证多 job 并行时的隔离性

## 知识沉淀

### 可复用技能

- [x] `rick_tools_check_pattern` → `skills/rick_tools_check_pattern.py`
- [x] `mock_agent_testing` → `skills/mock_agent_testing.py`

### Wiki 文档

- [x] `rick_tools_commands` → `wiki/rick_tools_commands.md`
- [x] `learning_phase_workflow` → `wiki/learning_phase_workflow.md`

### OKR/SPEC 更新

- [x] OKR 更新 → `OKR.md`
- [x] SPEC 更新 → `SPEC.md`
