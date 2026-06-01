# skill:tc (Test Case Design)

Use when writing test cases to ensure complete and unambiguous coverage.

## Test Case Four Elements

Every test case must have all four elements:

### 1. 前置条件（Preconditions）
The state of the system before the test runs:
- What data must exist in the database/cache?
- What environment variables or configs must be set?
- What prior operations must have completed?
- Example: "User account exists with status=active, balance=100"

### 2. 输入参数（Input Parameters）
The exact inputs fed to the system under test:
- All function arguments with concrete values (not "some value")
- Edge cases: empty string, nil, zero, negative, max int
- Invalid inputs that should trigger error handling
- Example: `userID=42, amount=-5, currency="CNY"`

### 3. 操作序列（Operation Sequence）
The ordered steps to execute:
1. Set up preconditions
2. Call the function/endpoint with input parameters
3. Observe the result
4. Verify side effects (database state, events emitted)

### 4. 预期输出（Expected Output）
The exact observable outcomes:
- Return value (with specific type and value)
- Changed state in storage
- Errors or exceptions raised (with message/code)
- Events or logs emitted
- Example: "returns error{code: INSUFFICIENT_FUNDS}, balance unchanged"

## Anti-patterns to Avoid
- Vague preconditions: "some user exists" → specify exact fields
- Vague expected output: "should work" or "returns something" → exact value
- Missing side-effect verification: check DB state after writes
