# rick tools 命令体系

## 概述

`rick tools` 是 Rick CLI 的工具链子命令体系，提供 plan/doing/learning 三阶段的自动校验和知识合并功能。设计目标是让 AI agent 和人类都能快速验证每个阶段的产出质量，并在出错时提供清晰的错误信息。

## 背景

在 Rick 的 plan→doing→learning 循环中，每个阶段都会产出特定格式的文件。如果这些文件格式不正确，会导致后续阶段执行失败或产出质量低下。`rick tools` 体系解决了以下问题：

- plan 阶段：task 文件格式错误、依赖关系错误（循环/悬空）
- doing 阶段：tasks.json 损坏、debug.md 缺失、zombie 任务状态
- learning 阶段：SUMMARY.md 缺失、skills 脚本语法错误、OKR/SPEC 格式不合规

## 工作原理

### 统一的 check 命令模式

所有 check 命令遵循相同的设计模式：

```
rick tools <check_cmd> <job_id> [--auto-fix]
```

- 默认行为：只检查并报告问题，不修改任何文件
- `--auto-fix`：调用 Claude 自动修复发现的问题（最多 3 次尝试）
- 输出格式：`✅ <check> passed: <details>` 或 `❌ <check> failed: <error>`
- Exit code：0=通过，1=失败

### plan_check 校验规则

1. plan/ 目录存在
2. 至少有一个 task*.md 文件
3. 每个 task 包含必需章节：`# 依赖关系`、`# 任务名称`、`# 任务目标`、`# 关键结果`、`# 测试方法`
4. 所有依赖引用的 task 文件存在
5. 无循环依赖（使用 Kahn 算法检测）
6. task 文件路径约束（必须在 plan/ 目录内）

### doing_check 校验规则

1. tasks.json 存在且可解析
2. debug.md 存在（强制工作日志）
3. 无 zombie 任务（状态为 "running" 但实际已停止）
4. 所有 success 状态的任务有 commit_hash

### learning_check 校验规则

1. SUMMARY.md 存在
2. skills/*.py 文件通过 Python 语法检查
3. OKR.md（如存在）包含 `## O` 和 `### 关键结果` 章节
4. SPEC.md（如存在）包含四个必需章节：技术栈、架构设计、开发规范、工程实践

### merge 流程

```
rick tools merge <job_id>
```

1. 检查 SUMMARY.md 第一行是否为 `<!-- APPROVED: true -->`
2. 获取当前分支
3. 创建并切换到 `learning/job_N` 分支
4. 复制 learning/wiki/ → .rick/wiki/
5. 复制 learning/skills/ → .rick/skills/
6. 复制 learning/OKR.md → .rick/OKR.md（如存在）
7. 复制 learning/SPEC.md → .rick/SPEC.md（如存在）
8. 重新生成 .rick/wiki/README.md 和 .rick/skills/README.md
9. git commit "learning: merge job_N knowledge"
10. 切换回原分支
11. 输出结构化摘要（供 AI agent 展示给人类）

人工审核后执行：`git merge --no-ff learning/job_N`

## 如何控制

### 运行 plan_check

```bash
# 基本检查
rick tools plan_check job_1

# 检查并自动修复
rick tools plan_check job_1 --auto-fix
```

### 运行 doing_check

```bash
# 检查 doing 阶段产出
rick tools doing_check job_1

# 自动修复（调用 Claude）
rick tools doing_check job_1 --auto-fix
```

### 运行 learning_check

```bash
# 检查 learning 阶段产出
rick tools learning_check job_1

# 自动修复
rick tools learning_check job_1 --auto-fix
```

### 执行 merge

```bash
# 先确保 SUMMARY.md 第一行是 <!-- APPROVED: true -->
# 然后执行 merge
rick tools merge job_1

# 审核 diff 后合并
git merge --no-ff learning/job_1
git branch -D learning/job_1
```

## 注意事项

- `--auto-fix` 需要 Claude CLI 在 PATH 中可用
- `merge` 命令要求 SUMMARY.md 第一行必须是 `<!-- APPROVED: true -->`，这是防止意外合并的安全门
- `doing_check` 的 debug.md 检查是强制的：debug.md 是每次任务执行的强制工作日志，不是可选的
- check 命令默认不修复（opt-in auto-fix），保持工具的确定性，方便在测试中验证失败场景

## 相关资源

- 相关 Wiki: [learning_phase_workflow.md](learning_phase_workflow.md)
- 相关 Skill: [rick_tools_check_pattern.py](../skills/rick_tools_check_pattern.py)
- 源码: `internal/cmd/tools*.go`
