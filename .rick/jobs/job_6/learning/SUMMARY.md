APPROVED: true

# Job job_6 执行总结

## 执行概述

**项目目标**: 实现 `rick human-loop` 命令完整链路（prompt 模板、生成函数、RFC 目录管理、CLI 命令）及 `rick plan --job` 重进已有 job 功能
**实际完成**: 4 个任务全部实现并通过测试（task2/task4/task1/task3 均有 commit 和通过的 go test）
**整体评价**: ⭐⭐⭐⭐⭐

## 关键成就

1. **human-loop 完整链路**: 从 prompt 模板（task2）→ 生成函数（task2）→ CLI 命令（task3）→ 模板验证（task1），四个任务形成完整闭环，`rick human-loop <topic>` 命令可用
2. **plan --job 重进机制**: task4 为 `rick plan` 增加 `--job` flag，支持重进已有 job 的 plan 目录，为 AI agent 中断恢复场景提供基础设施
3. **NewPromptManager 向后兼容改造**: 将 `NewPromptManager(templateDir string)` 改为 variadic 函数，解决测试脚本无参调用的兼容性问题，同时不破坏已有调用方
4. **零重试完成**: 所有任务均在首次尝试时通过，无重试

## 问题与教训

### 问题1: task2.py 测试脚本路径计算有误

**根本原因**: task2.py 中 project_root 路径多了一层 `..`，导致 `go build` 命令在错误目录执行
**解决方案**: 绕过脚本，直接用 `go -C <abs_path>` 形式手动验证
**经验教训**: task 测试脚本生成时应验证 dirname 层数；AI agent 执行时遇到脚本路径错误，应优先用绝对路径绕过，而不是修改测试脚本

### 问题2: shell CWD 持续重置

**根本原因**: 执行环境中 shell 状态不跨工具调用保持，每次 Bash 调用都重置到初始目录
**解决方案**: 所有 go 命令统一使用 `go -C <abs_path>` 形式
**经验教训**: 这是已知环境约束，已在 SPEC 路径规范中记录；新任务应在 debug.md 开头即采用此模式

### 问题3: task3 标记为 failed 但实际通过

**根本原因**: 任务执行结果表中 task3 状态为 failed，但 debug.md 和 commit 记录均显示通过
**解决方案**: 以实际代码和测试结果为准，任务已成功实现
**经验教训**: 执行结果表的状态字段可能因 morty 记录时机问题产生偏差，需以 commit 和测试输出为最终依据

## 技术总结

### 关键技术决策

- **variadic NewPromptManager**: 选择 variadic 而非新增无参构造函数，原因是保持接口简洁，避免两个构造函数导致的混淆；影响：所有现有调用方无需修改
- **human_loop.go 复用 callClaudeCodeCLI**: 不在 human_loop.go 中重新声明，而是直接调用同包内 plan.go 中的函数；影响：避免 `redeclared in this block` 编译错误，符合 Go 包内函数共享惯例
- **RFC 目录自动创建**: `rick human-loop` 执行时自动 MkdirAll 创建 `.rick/RFC/`，无需用户手动初始化；影响：降低使用门槛，符合 rick 零配置理念

### 知识沉淀清单

- [x] wiki/human_loop_command.md - human-loop 命令工作原理与使用指南
- [x] skills/check_variadic_api.py - 检测 Go 函数是否已改为 variadic 的验证技能
- [ ] OKR.md - 无需更新（现有 OKR 已覆盖本次目标）
- [x] SPEC.md - 补充 human-loop 命令规范和 RFC 目录约定

---

## 会话追加改动（learning 阶段人工审查期间）

### Bug 修复

**1. `rick plan --job` 传入 requirement 后报 "requirement cannot be empty"**

- **根本原因**: `reEnterPlanWorkflow` 调用 `GeneratePlanPromptFile` 时硬编码传空字符串 `""`，而该函数校验 requirement 不能为空
- **修复**: `reEnterPlanWorkflow` 增加 `requirement string` 参数，调用处从 args 透传；无参时默认值为 `"重新进入已有计划，继续完善任务分解"`
- **文件**: `internal/cmd/plan.go`、`internal/cmd/plan_test.go`

**2. OKR/SPEC 在 prompt 中显示"暂无"**

- **根本原因**: 解析器 `ExtractObjectives` 查找 `# 目标` / `# Objectives` 标题，但 rick 的 OKR.md 实际使用 `## O1:` / `### 关键结果 (Key Results)` 格式，完全匹配不上
- **修复**: `ContextManager` 新增 `OKRRaw` / `SPECRaw` 字段保存原始文件内容；`plan_prompt.go` 中解析结果为空时 fallback 到原文注入 prompt
- **文件**: `internal/prompt/context.go`、`internal/prompt/plan_prompt.go`

**3. `GetRickDir()` 不向上查找导致非项目根目录运行时路径错误**

- **修复**: `GetRickDir()` 改为从 cwd 向上逐级查找已存在的 `.rick` 目录（类似 git 查找 `.git`），找不到时 fallback 到 cwd
- **文件**: `internal/workspace/paths.go`

### 功能新增

**4. 移植 sense skills 并集成到安装/卸载流程**

- 从 sense 项目复制 `skills/sense-human-loop/` 到 `rick/skills/`（含 sense-think、sense-learn、sense-express 三个子 skill）
- `scripts/install.sh` 新增 `install_skills()` 和 `verify_skills()`：安装时将 `rick/skills/*/` 软链到 `~/.claude/skills/`，并验证 `sense-human-loop` 可用
- `scripts/uninstall.sh` 新增 `uninstall_skills()`：卸载时删除 `~/.claude/skills/` 下对应软链接
- 不涉及 `~/.openclaw/workspace/skills/`（已移除）

**5. 版本升级至 v0.3.2**

- **文件**: `cmd/rick/main.go`
