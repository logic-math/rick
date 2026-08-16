# skill:check-mechanism（Check 机制工作原理）

## 触发场景

learning_check / dream_check 命令失败，需要理解失败原因或扩展新检查规则时使用。（doing 门禁已下沉为 rick-gates 确定性脚本，plan_check/doing_check 已删除。）

## 预期效果

- 能读懂 check 失败输出，精准修复
- 能扩展新 check 规则而不破坏现有规则
- 每次运行 check 后能判断是否需要重新 commit

## 核心内容

### check 命令和各自验证内容

| 命令 | 验证内容 | 通过条件 |
|------|--------|--------|
| `rick tools learning_check job_N` | SUMMARY.md 非空且含 `# Job` 标题 | exit code 0，输出 `✅ PASS` |
| `rick tools dream_check job_N` | dream_run_*_log.md 五要素格式 | exit code 0，输出 `✅ PASS` |
| `python3 .rick/skills/rick-gates/helper.py <doing_dir>` | tasks.json 可解析 / 无 zombie running / success 有 commit_hash | exit code 0 |

### 手动运行

```bash
# 优先使用本地构建版（含当前代码）
./bin/rick tools learning_check job_N
./bin/rick tools dream_check job_N
python3 .rick/skills/rick-gates/helper.py .rick/jobs/job_N/doing
```

### check 失败时的修复流程

1. 读取失败输出，定位具体 check 项
2. 修复对应文件（SUMMARY.md / dream log / tasks.json）
3. 如果是 tasks.json 状态问题 → 参考 [mark_task_success_skill](../mark_task_success_skill/skill.md)
4. 修复后重新提交 + 再次运行 check

### rick-gates 门禁常见失败原因

| 失败信息 | 原因 | 修复 |
|---------|------|------|
| `missing commit_hash` | success 任务缺少 commit_hash | 补写 git rev-parse HEAD 的结果 |
| `zombie running` | tasks.json 中遗留 running 任务 | 将卡住任务改为 failed 或补全 commit |
| `tasks.json 不可解析` | tasks.json 非法 JSON | 修复 JSON 语法 |

### 扩展新 check 规则

在对应的 `internal/cmd/tools_learning_check.go` / `tools_dream_check.go` 的 `run*Check()` 函数末尾追加检查项，并同步更新其 fix prompt。

构建后验证：`./scripts/build.sh && ./bin/rick tools learning_check job_N`
