# skill:check-mechanism（Check 机制工作原理）

## 触发场景

plan/doing/learning_check 命令失败，需要理解失败原因或扩展新检查规则时使用。

## 预期效果

- 能读懂 check 失败输出，精准修复
- 能扩展新 check 规则而不破坏现有规则
- 每次运行 check 后能判断是否需要重新 commit

## 核心内容

### 三个 check 命令和各自验证内容

| 命令 | 验证内容 | 通过条件 |
|------|--------|--------|
| `rick tools plan_check job_N` | tasks.json 存在；tasks/*.md 格式正确 | exit code 0，输出 `✅ PASS` |
| `rick tools doing_check job_N` | tasks.json 可解析；debug/bug*.md 格式 | exit code 0，输出 `✅ PASS` |
| `rick tools learning_check job_N` | SUMMARY.md 非空且含 `# Job` 标题 | exit code 0，输出 `✅ PASS` |

### 手动运行

```bash
# 优先使用本地构建版（含当前代码）
./bin/rick tools doing_check job_N
./bin/rick tools plan_check job_N
./bin/rick tools learning_check job_N
```

### check 失败时的修复流程

1. 读取失败输出，定位具体 check 项
2. 修复对应文件（tasks.json / debug/bug*.md / SUMMARY.md）
3. 如果是 tasks.json 状态问题 → 参考 [mark_task_success_skill](../mark_task_success_skill/skill.md)
4. 修复后重新提交 + 再次运行 check

### doing_check 常见失败原因

| 失败信息 | 原因 | 修复 |
|---------|------|------|
| `task status != success` | tasks.json 中 task 未标记完成 | 运行 `mark_task_success.py` |
| `commit_hash is empty` | tasks.json 中缺少 commit_hash | 补充 git rev-parse HEAD 的结果 |
| `debug format error` | bug*.md frontmatter 不完整 | 检查 frontmatter 含 title/status/summary |

### 扩展新 check 规则

在对应的 `internal/cmd/tools_*_check.go` 文件的 `run*Check()` 函数末尾追加：

```go
// 新增：检查某文件存在性
xPath := filepath.Join(jobDir, "X.md")
if _, err := os.Stat(xPath); os.IsNotExist(err) {
    errors = append(errors, fmt.Sprintf("X.md not found: %s", xPath))
}
```

同步更新 `write*CheckFixPrompt` 函数，加入新检查项的修复说明。

构建后验证：`./scripts/build.sh && ./bin/rick tools doing_check job_N`
