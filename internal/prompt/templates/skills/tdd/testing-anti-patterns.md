# skill:tdd/testing-anti-patterns (Forbidden Test Patterns)

Use when writing or reviewing tests to identify patterns that create false confidence.

## Anti-Pattern Checklist

### 1. 空断言测试（Empty Assertion Test）
```go
// WRONG: test passes even if function does nothing
func TestProcess(t *testing.T) {
    err := Process(input)
    _ = err  // ignored!
}

// CORRECT: assert the expected outcome
func TestProcess(t *testing.T) {
    err := Process(input)
    if err != nil {
        t.Fatalf("expected no error, got: %v", err)
    }
}
```

### 2. 永远为真的断言（Tautology Assertion）
```go
// WRONG: this always passes regardless of implementation
if result != nil {
    t.Log("got a result")
}

// CORRECT: check the actual value
if result.Count != expectedCount {
    t.Errorf("expected %d, got %d", expectedCount, result.Count)
}
```

### 3. 只测快乐路径（Happy Path Only）
Missing tests for: nil input, empty input, max value, negative numbers, concurrent access.
Every exported function needs at least one error-case test.

### 4. 测试实现而非行为（Testing Implementation）
```go
// WRONG: tests internal state
if service.internalCache["key"] == value { ... }

// CORRECT: tests observable behavior
result, err := service.Get("key")
```

### 5. 依赖外部状态的测试（External State Dependency）
Tests that rely on specific files, network, time, or databases without setup/teardown.
Use `t.TempDir()`, mocks, or test fixtures.

### 6. 非确定性测试（Non-deterministic Test）
Tests that pass sometimes and fail sometimes (flaky tests):
- Using `time.Sleep` instead of synchronization
- Using random data without seeding
- Depending on goroutine scheduling order

### Detection Rule
If a test can pass with an empty/no-op implementation of the function under test, it's a fake test.
