# skill:tdd/testing-anti-patterns（禁止的测试反模式）

编写或评审测试时，识别会制造虚假信心的模式时使用。

## 反模式检查清单

### 1. 空断言测试
```go
// 错误：即使函数什么都不做，测试也会通过
func TestProcess(t *testing.T) {
    err := Process(input)
    _ = err  // 被忽略了！
}

// 正确：断言期望的结果
func TestProcess(t *testing.T) {
    err := Process(input)
    if err != nil {
        t.Fatalf("期望无错误，但得到: %v", err)
    }
}
```

### 2. 永远为真的断言
```go
// 错误：无论实现如何，这个断言总是通过
if result != nil {
    t.Log("得到了一个结果")
}

// 正确：检查实际的值
if result.Count != expectedCount {
    t.Errorf("期望 %d，实际 %d", expectedCount, result.Count)
}
```

### 3. 只测快乐路径
缺少以下情况的测试：nil 输入、空输入、最大值、负数、并发访问。
每个导出函数至少需要一个错误场景的测试。

### 4. 测试实现而非行为
```go
// 错误：测试内部状态
if service.internalCache["key"] == value { ... }

// 正确：测试可观测的行为
result, err := service.Get("key")
```

### 5. 依赖外部状态的测试
依赖特定文件、网络、时间或数据库而没有 setup/teardown 的测试。
使用 `t.TempDir()`、mock 或测试固件。

### 6. 非确定性测试（脆弱测试）
有时通过有时失败的测试：
- 用 `time.Sleep` 而不是同步机制
- 使用随机数据却不设置随机种子
- 依赖 goroutine 调度顺序

### 检测规则
如果一个测试在被测函数是空实现/无操作实现时也能通过，它就是一个假测试。
