# 错误分析和假设生成 - 实现细节

## 核心算法

### analyzeError
```go
func (trm *TaskRetryManager) analyzeError(errMsg string, output string) string {
    hypotheses := []string{}

    // 分析错误消息
    if strings.Contains(errMsg, "timeout") {
        hypotheses = append(hypotheses, "执行超时 - 可能是任务太复杂或资源不足")
    } else if strings.Contains(errMsg, "not found") || strings.Contains(errMsg, "does not exist") {
        hypotheses = append(hypotheses, "文件或资源不存在 - 可能是路径错误或文件未创建")
    } else if strings.Contains(errMsg, "permission") {
        hypotheses = append(hypotheses, "权限不足 - 需要检查文件/目录权限")
    } else if strings.Contains(errMsg, "connection") {
        hypotheses = append(hypotheses, "网络连接失败 - 检查网络或服务可用性")
    } else if strings.Contains(errMsg, "test did not pass") {
        hypotheses = append(hypotheses, "测试未通过 - 任务执行结果不符合预期")
    } else if strings.Contains(errMsg, "failed to generate test script") {
        hypotheses = append(hypotheses, "测试脚本生成失败 - 检查测试方法定义")
    } else {
        hypotheses = append(hypotheses, "未知错误 - 需要详细分析输出日志")
    }

    // 分析输出中的线索
    if strings.Contains(output, "FAIL") {
        hypotheses = append(hypotheses, "测试断言失败")
    }
    if strings.Contains(output, "ERROR") {
        hypotheses = append(hypotheses, "运行时错误")
    }
    if strings.Contains(output, "SyntaxError") {
        hypotheses = append(hypotheses, "Python语法错误")
    }
    if strings.Contains(output, "ImportError") || strings.Contains(output, "ModuleNotFoundError") {
        hypotheses = append(hypotheses, "缺少Python模块依赖")
    }

    if len(hypotheses) == 0 {
        return "未知错误 - 需要人工分析"
    }

    return strings.Join(hypotheses, "; ")
}
```

## 错误模式表

| 关键词 | 假设 |
|--------|------|
| timeout | 执行超时 - 可能是任务太复杂或资源不足 |
| not found / does not exist | 文件或资源不存在 - 可能是路径错误或文件未创建 |
| permission | 权限不足 - 需要检查文件/目录权限 |
| connection | 网络连接失败 - 检查网络或服务可用性 |
| test did not pass | 测试未通过 - 任务执行结果不符合预期 |
| FAIL (output) | 测试断言失败 |
| ERROR (output) | 运行时错误 |
| SyntaxError (output) | Python语法错误 |
| ImportError (output) | 缺少Python模块依赖 |

## 使用示例

### 示例1: 文件不存在
```go
errMsg := "test did not pass"
output := "File not found: output.txt"

hypothesis := analyzeError(errMsg, output)
// Output: "文件或资源不存在 - 可能是路径错误或文件未创建; 测试未通过 - 任务执行结果不符合预期"
```

### 示例2: Python 错误
```go
errMsg := "test failed"
output := `
Traceback (most recent call last):
  File "test.py", line 1
    print("hello"
SyntaxError: unexpected EOF while parsing
`

hypothesis := analyzeError(errMsg, output)
// Output: "未知错误 - 需要详细分析输出日志; Python语法错误"
```

### 示例3: 超时
```go
errMsg := "execution timeout after 30s"
output := ""

hypothesis := analyzeError(errMsg, output)
// Output: "执行超时 - 可能是任务太复杂或资源不足"
```

## 扩展: 添加新的错误模式

```go
// 添加磁盘空间不足模式
if strings.Contains(errMsg, "no space left") || strings.Contains(output, "disk full") {
    hypotheses = append(hypotheses, "磁盘空间不足 - 需要清理临时文件")
}

// 添加内存不足模式
if strings.Contains(errMsg, "out of memory") || strings.Contains(output, "MemoryError") {
    hypotheses = append(hypotheses, "内存不足 - 需要优化内存使用或增加资源")
}
```

---

*参考: `internal/executor/retry.go`*
*最后更新: 2026-03-14*
