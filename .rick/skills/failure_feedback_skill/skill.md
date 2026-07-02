# skill:failure-feedback（失败信息传递机制）

## 触发场景

doing 阶段 task 失败重试时，需要理解或调整失败信息如何传递给下一轮 Agent 时使用。

## 预期效果

- 下一轮 Agent 能看到上一轮的完整错误输出（含 traceback/stderr）
- prompt 不因重试次数增加而无限膨胀（最多保留最近 2 次，3000 字符上限）

## 核心内容

### 失败信息流转

```
ExecuteTestScript 返回 testOutput
    → RunTask 构造 result.Error（含 Full test output: ...）
    → RetryTask 生成 "=== Attempt N ===\n{error}" 条目
    → appendFailureFeedback（最近2条，3000字符）
    → GenerateDoingPromptFile 注入 testErrorFeedback
    → 下一轮 Claude Agent
```

### appendFailureFeedback 算法

- 按 `=== Attempt ` 分割现有条目
- 追加新条目，只保留最近 `maxEntries`（默认 2）条
- 合并后超 `maxBytes`（默认 3000）时从尾部截断，对齐到行边界

### 调试：查看传递给下一轮的 prompt

doing prompt 执行后默认被删除。临时调试时注释掉 runner.go 中的删除逻辑：
```go
// defer os.Remove(doingPromptFile)  // 注释以保留文件
```

### 调整参数（internal/executor/retry.go）

```go
// 保留最近条数（默认2）和字符上限（默认3000）
testErrorFeedback = appendFailureFeedback(testErrorFeedback, newEntry, 2, 3000)
```

### 为什么完整 testOutput 重要

旧版（500字符截断）agent 只能看到：
```
test did not pass: assertion failed; key not found
```

新版 agent 能看到：
```
test did not pass: assertion failed; key not found
Full test output:
AssertionError: OKR.md not found at .rick/jobs/job_1/plan/OKR.md
{"pass": false, "errors": ["assertion failed"]}
```

文件路径和行号直接可见，修复效率大幅提升。
