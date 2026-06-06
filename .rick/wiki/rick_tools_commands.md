# rick tools 命令体系

## 概述

`rick tools` 是 Rick CLI 的工具链子命令体系，提供 plan/doing/learning/dream 四个阶段的自动校验功能。

```
rick tools plan_check <job_id>    [--auto-fix]
rick tools doing_check <job_id>   [--auto-fix]
rick tools learning_check <job_id>[--auto-fix]
rick tools dream_check
```

## 统一的 check 命令模式

所有 check 命令遵循相同设计：

- **默认行为**：只检查报告问题，不修改文件
- `--auto-fix`：调用 Claude 最多自动修复 3 次
- **输出格式**：`✅ <check> passed: <details>` 或 `❌ <check> failed: <error>`
- **Exit code**：0=通过，1=失败

## plan_check 校验规则

1. `plan/` 目录存在
2. 至少有一个 `task*.md` 文件
3. 每个 task 包含必需章节：`# 依赖关系`、`# 任务名称`、`# 任务目标`、`# 关键结果`、`# 测试方法`
4. 所有依赖引用的 task 文件存在（无悬空引用）
5. 无循环依赖（Kahn 算法检测）

## doing_check 校验规则

1. `tasks.json` 存在且可解析
2. `debug.md` 存在且非空（强制工作日志）
3. `debug.md` 包含 `## task` 记录节
4. 无 zombie 任务（状态为 `"running"` 但实际已停止）
5. 所有 `success` 状态的任务有 `commit_hash`

## learning_check 校验规则

1. `SUMMARY.md` 存在
2. `SUMMARY.md` 包含 `# Job` heading（非空摘要）

## dream_check 校验规则

1. `.rick/dream/` 目录存在（不存在则视为"无运行记录"，直接通过）
2. 所有 `dream_run_*_log.md` 文件名包含合法的 `job_N` 格式 job ID
3. 各 log 文件 job ID 无重复
4. 每个 log 对应的 job 目录实际存在

## 如何使用

```bash
# plan 阶段校验
rick tools plan_check job_1
rick tools plan_check job_1 --auto-fix

# doing 阶段校验
rick tools doing_check job_1
rick tools doing_check job_1 --auto-fix

# learning 阶段校验
rick tools learning_check job_1
rick tools learning_check job_1 --auto-fix

# dream 阶段校验（无需 job_id 参数）
rick tools dream_check
```

## 注意事项

- `--auto-fix` 需要 Claude CLI 在 PATH 中可用
- `doing_check` 的 `debug.md` 检查是强制的：每次任务执行必须有工作日志
- check 命令默认 opt-in auto-fix，保持工具确定性，便于测试中验证失败场景
- **`rick tools merge` 尚未实现**（见 RFC-005），知识合并当前需人工手动 `git merge`

## 相关资源

- 相关 Wiki: [learning_phase_workflow.md](learning_phase_workflow.md)
- 源码: `internal/cmd/tools.go`、`internal/cmd/tools_*.go`
