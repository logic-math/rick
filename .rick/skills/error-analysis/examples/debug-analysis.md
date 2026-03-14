# 案例: 任务失败错误分析

## 场景
Claude Code 执行任务失败，需要自动分析错误原因并生成假设。

## 失败信息
```
Error: test did not pass
Output:
  PASS: File hello.py exists
  FAIL: Output does not contain "Hello, Rick!"
  Expected: Hello, Rick!
  Actual: Hello, World!
```

## 错误分析
```go
errMsg := "test did not pass"
output := `
PASS: File hello.py exists
FAIL: Output does not contain "Hello, Rick!"
Expected: Hello, Rick!
Actual: Hello, World!
`

hypotheses := analyzeError(errMsg, output)
```

## 生成的假设
```
测试未通过 - 任务执行结果不符合预期; 测试断言失败
```

## 集成到 debug.md
```markdown
## debug1: Task task1 - Attempt 1/5

**现象 (Phenomenon)**:
- 测试未通过：输出不符合预期

**猜想 (Hypothesis)**:
- 测试未通过 - 任务执行结果不符合预期
- 测试断言失败

**输出 (Output)**:
```
PASS: File hello.py exists
FAIL: Output does not contain "Hello, Rick!"
Expected: Hello, Rick!
Actual: Hello, World!
```
```

## AI Agent 的理解
Claude Code 在下一轮重试时读取 debug.md，理解到：
1. 文件创建成功了（PASS）
2. 但输出内容错误（FAIL）
3. 期望 "Hello, Rick!"，实际是 "Hello, World!"
4. 需要修改 print 语句的内容

---

*最后更新: 2026-03-14*
