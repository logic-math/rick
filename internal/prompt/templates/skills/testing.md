# skill:testing (Test Execution Specification)

Use when executing tests to diagnose failures and ensure test quality.

## Test Execution Rules

### Before Running Tests
- Ensure the code compiles (`go build ./...` or equivalent)
- Run tests in isolation first (`go test ./path/to/package/...`)
- Never run `go test ./...` as the first step when debugging a failure

### Interpreting Test Output
- `PASS` — all tests in the package passed
- `FAIL` — at least one test failed; read the specific failure message
- `no test files` — package has no tests (may be intentional)
- Build errors before tests = compilation failure, fix first

### Diagnosing Failures
1. Read the FULL error message, not just the last line
2. Find the exact file and line number from the stack trace
3. Check if the failure is:
   - Assertion failure: expected vs actual mismatch
   - Panic: nil dereference, out of bounds, etc.
   - Timeout: test hung, likely deadlock or infinite loop
   - Setup failure: precondition not met

### Test Isolation
- Each test must be independent — no shared mutable state
- Use `t.TempDir()` for file system tests (auto-cleaned)
- Use table-driven tests for multiple input scenarios
- Mock external dependencies at boundaries, not internal functions

### Running Specific Tests
```bash
go test ./pkg/... -run TestFunctionName -v       # specific test with verbose
go test ./pkg/... -run TestSuite/SubTest -v      # subtests
go test ./pkg/... -count=1                       # disable test cache
go test ./pkg/... -race                          # race condition detection
```

### After All Tests Pass
- Run `go test ./...` to ensure no regressions
- Check that new tests are actually exercising the code (coverage hint: `-cover`)
