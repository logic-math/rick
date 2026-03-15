# Rick CLI 测试与验证文档

## 目录

- [测试策略概览](#测试策略概览)
- [单元测试](#单元测试)
- [集成测试](#集成测试)
- [测试脚本生成机制](#测试脚本生成机制)
- [测试覆盖率](#测试覆盖率)
- [测试命令与示例](#测试命令与示例)
- [CI/CD 集成](#cicd-集成)
- [测试最佳实践](#测试最佳实践)

---

## 测试策略概览

Rick CLI 采用多层次测试策略，确保代码质量和功能正确性：

### 测试金字塔

```
         ┌─────────────┐
         │   E2E 测试   │  ← 端到端集成测试（scripts/test_integration.sh）
         └─────────────┘
       ┌───────────────────┐
       │   集成测试        │  ← 模块间集成测试（scripts/test_*.sh）
       └───────────────────┘
    ┌─────────────────────────┐
    │     单元测试            │  ← Go 单元测试（*_test.go）
    └─────────────────────────┘
  ┌───────────────────────────────┐
  │   任务测试脚本（自动生成）    │  ← Python JSON 格式测试脚本
  └───────────────────────────────┘
```

### 测试层次说明

1. **任务测试脚本（Task Test Scripts）**
   - **目的**: 验证每个任务的具体执行结果
   - **格式**: Python 脚本，返回 JSON 格式结果
   - **生成**: 由 Claude Agent 根据任务的 `TestMethod` 自动生成
   - **执行**: 在任务执行后立即运行，验证任务是否完成

2. **单元测试（Unit Tests）**
   - **目的**: 测试单个函数或模块的功能
   - **工具**: Go testing 包
   - **覆盖**: 所有核心模块（parser, executor, prompt, git, config, workspace）
   - **运行**: `go test ./...`

3. **集成测试（Integration Tests）**
   - **目的**: 测试模块间的交互和完整功能流程
   - **工具**: Bash 脚本（scripts/test_*.sh）
   - **覆盖**: 安装脚本、版本管理、命令行工具集成
   - **运行**: `bash scripts/test_integration.sh`

4. **E2E 测试（End-to-End Tests）**
   - **目的**: 测试完整的用户工作流
   - **场景**: plan → doing → learning 完整循环
   - **运行**: 手动测试或 CI/CD 流水线

---

## 单元测试

### Go Testing 包使用

Rick CLI 使用 Go 标准库的 `testing` 包进行单元测试。

#### 基本测试结构

```go
package executor

import (
    "testing"
    "github.com/sunquan/rick/internal/parser"
)

// TestNewDAGEmpty 测试空任务列表
func TestNewDAGEmpty(t *testing.T) {
    dag, err := NewDAG([]*parser.Task{})
    if err != nil {
        t.Fatalf("NewDAG failed with empty list: %v", err)
    }
    if dag.TaskCount() != 0 {
        t.Errorf("Expected 0 tasks, got %d", dag.TaskCount())
    }
}
```

#### 测试辅助函数

```go
// Helper function to create a test task
func createTestTask(id, name, goal string, deps []string) *parser.Task {
    return &parser.Task{
        ID:           id,
        Name:         name,
        Goal:         goal,
        Dependencies: deps,
        KeyResults:   []string{},
        TestMethod:   "test method",
    }
}
```

#### 表驱动测试（Table-Driven Tests）

```go
func TestParseTaskMarkdown(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        wantErr  bool
        expected *Task
    }{
        {
            name: "valid task with dependencies",
            input: `# 依赖关系
task1, task2

# 任务名称
Test Task

# 任务目标
Complete the test`,
            wantErr: false,
            expected: &Task{
                Name:         "Test Task",
                Goal:         "Complete the test",
                Dependencies: []string{"task1", "task2"},
            },
        },
        {
            name:    "invalid task without name",
            input:   "# 任务目标\nSome goal",
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            task, err := ParseTaskMarkdown(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("ParseTaskMarkdown() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if !tt.wantErr && task.Name != tt.expected.Name {
                t.Errorf("ParseTaskMarkdown() Name = %v, want %v", task.Name, tt.expected.Name)
            }
        })
    }
}
```

#### 临时文件测试

```go
func TestWorkspaceCreation(t *testing.T) {
    // Create temporary directory
    tmpDir, err := os.MkdirTemp("", "rick-test-*")
    if err != nil {
        t.Fatalf("Failed to create temp dir: %v", err)
    }
    defer os.RemoveAll(tmpDir) // Clean up

    // Test workspace creation
    ws, err := NewWorkspace(tmpDir)
    if err != nil {
        t.Fatalf("Failed to create workspace: %v", err)
    }

    // Verify directory structure
    if !fileExists(filepath.Join(tmpDir, ".rick")) {
        t.Error(".rick directory not created")
    }
}
```

### 模块测试覆盖

#### 1. Parser 模块测试

**文件**: `internal/parser/*_test.go`

测试内容：
- Markdown 解析（`markdown_test.go`）
- Task 解析（`task_test.go`）
- Debug 日志解析（`debug_test.go`）
- Context 管理（`context_test.go`）
- 集成测试（`integration_test.go`）

示例：
```go
func TestParseMarkdownHeadings(t *testing.T) {
    input := `# Heading 1
Content 1

## Heading 2
Content 2`

    sections, err := ParseMarkdownSections(input)
    if err != nil {
        t.Fatalf("ParseMarkdownSections failed: %v", err)
    }
    if len(sections) != 2 {
        t.Errorf("Expected 2 sections, got %d", len(sections))
    }
}
```

#### 2. Executor 模块测试

**文件**: `internal/executor/*_test.go`

测试内容：
- DAG 构建（`dag_test.go`）
- 拓扑排序（`topological_test.go`）
- 任务执行器（`executor_test.go`）
- 重试机制（`retry_test.go`）
- 任务运行器（`runner_test.go`）
- tasks.json 管理（`tasks_json_test.go`）

示例：
```go
func TestTopologicalSort(t *testing.T) {
    tasks := []*parser.Task{
        createTestTask("task1", "Task 1", "Goal 1", []string{}),
        createTestTask("task2", "Task 2", "Goal 2", []string{"task1"}),
        createTestTask("task3", "Task 3", "Goal 3", []string{"task1"}),
    }

    dag, err := NewDAG(tasks)
    if err != nil {
        t.Fatalf("NewDAG failed: %v", err)
    }

    sorted, err := dag.TopologicalSort()
    if err != nil {
        t.Fatalf("TopologicalSort failed: %v", err)
    }

    // Verify task1 comes before task2 and task3
    task1Idx := findTaskIndex(sorted, "task1")
    task2Idx := findTaskIndex(sorted, "task2")
    task3Idx := findTaskIndex(sorted, "task3")

    if task1Idx > task2Idx || task1Idx > task3Idx {
        t.Error("task1 should come before task2 and task3")
    }
}
```

#### 3. Prompt 模块测试

**文件**: `internal/prompt/*_test.go`

测试内容：
- Prompt 管理器（`manager_test.go`）
- Prompt 构建器（`builder_test.go`）
- Context 管理（`context_test.go`）
- 各阶段 Prompt 生成（`plan_prompt_test.go`, `doing_prompt_test.go`, `learning_prompt_test.go`, `test_prompt_test.go`）
- 嵌入式模板（`embedded_test.go`）

示例：
```go
func TestBuildDoingPrompt(t *testing.T) {
    task := &parser.Task{
        ID:   "task1",
        Name: "Test Task",
        Goal: "Complete the test",
    }

    builder := NewPromptBuilder()
    prompt, err := builder.BuildDoingPrompt(task, "", nil)
    if err != nil {
        t.Fatalf("BuildDoingPrompt failed: %v", err)
    }

    // Verify prompt contains task information
    if !strings.Contains(prompt, "task1") {
        t.Error("Prompt should contain task ID")
    }
    if !strings.Contains(prompt, "Test Task") {
        t.Error("Prompt should contain task name")
    }
}
```

#### 4. Git 模块测试

**文件**: `internal/git/*_test.go`

测试内容：
- Git 操作（`git_test.go`）
- Commit 管理（`commit_test.go`）
- 版本管理（`version_test.go`）
- Rollback 功能（`rollback_test.go`）
- 集成测试（`integration_test.go`）

#### 5. Config 模块测试

**文件**: `internal/config/loader_test.go`

测试内容：
- 配置加载
- 配置验证
- 默认值处理

#### 6. Workspace 模块测试

**文件**: `internal/workspace/workspace_test.go`

测试内容：
- 工作空间创建
- 目录结构验证
- Job 管理

#### 7. CMD 模块测试

**文件**: `internal/cmd/*_test.go`

测试内容：
- 命令处理器（`root_test.go`, `plan_test.go`, `doing_test.go`, `learning_test.go`）
- CLI 集成测试（`cli_integration_test.go`）
- 反馈助手（`feedback_helper_test.go`）

---

## 集成测试

### Shell 脚本测试框架

Rick CLI 使用 Bash 脚本进行集成测试，位于 `scripts/test_*.sh`。

#### 测试脚本结构

```bash
#!/bin/bash
#
# test_integration.sh - Integration test for Rick CLI
#

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

TESTS_PASSED=0
TESTS_FAILED=0

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}Rick CLI Integration Tests${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# Test 1: Verify build
echo -e "${YELLOW}Test 1: Verify build.sh can compile${NC}"
cd "$PROJECT_DIR"
if bash "$SCRIPT_DIR/build.sh" > /tmp/build_output.log 2>&1; then
    echo -e "${GREEN}✓ PASS${NC}: build.sh compiles successfully"
    ((TESTS_PASSED++))
else
    echo -e "${RED}✗ FAIL${NC}: build.sh compilation failed"
    ((TESTS_FAILED++))
fi

# Test 2: Verify binary exists
if [ -f "$PROJECT_DIR/bin/rick" ]; then
    echo -e "${GREEN}✓ PASS${NC}: Binary created"
    ((TESTS_PASSED++))
else
    echo -e "${RED}✗ FAIL${NC}: Binary not found"
    ((TESTS_FAILED++))
fi

# Summary
echo ""
echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}Test Summary${NC}"
echo -e "${BLUE}========================================${NC}"
echo -e "Tests Passed: ${GREEN}${TESTS_PASSED}${NC}"
echo -e "Tests Failed: ${RED}${TESTS_FAILED}${NC}"

if [ $TESTS_FAILED -eq 0 ]; then
    echo -e "${GREEN}All tests passed!${NC}"
    exit 0
else
    echo -e "${RED}Some tests failed!${NC}"
    exit 1
fi
```

### 现有集成测试脚本

#### 1. test_integration.sh

**用途**: 完整的安装流程集成测试

**测试内容**:
- build.sh 编译测试
- 二进制文件验证
- install.sh 安装测试
- 版本检查
- uninstall.sh 卸载测试

**运行**:
```bash
bash scripts/test_integration.sh
```

#### 2. test_install.sh

**用途**: 测试安装脚本的各种场景

**测试内容**:
- 源码安装（生产版）
- 源码安装（开发版）
- 二进制安装
- 配置文件创建
- PATH 环境变量配置

**运行**:
```bash
bash scripts/test_install.sh
```

#### 3. test_version.sh

**用途**: 测试版本管理功能

**测试内容**:
- 生产版本和开发版本并存
- 版本号验证
- 命令可用性检查

**运行**:
```bash
bash scripts/test_version.sh
```

#### 4. test_update.sh

**用途**: 测试更新脚本功能

**测试内容**:
- update.sh 更新流程
- 更新后版本验证
- 配置保留验证

**运行**:
```bash
bash scripts/test_update.sh
```

#### 5. test_check_env.sh

**用途**: 测试环境检查功能

**测试内容**:
- 依赖检查（Go, Python, Git）
- 配置文件验证
- 工作空间检查

**运行**:
```bash
bash scripts/test_check_env.sh
```

---

## 测试脚本生成机制

### Python JSON 格式测试脚本

Rick CLI 的核心创新之一是**自动生成任务测试脚本**。每个任务执行前，系统会根据任务的 `TestMethod` 字段，使用 Claude Agent 自动生成 Python 测试脚本。

### 测试脚本生成流程

```
┌─────────────────────┐
│  Task with          │
│  TestMethod field   │
└──────────┬──────────┘
           │
           ▼
┌─────────────────────┐
│  Generate Test      │
│  Prompt             │
└──────────┬──────────┘
           │
           ▼
┌─────────────────────┐
│  Call Claude Agent  │
│  to generate script │
└──────────┬──────────┘
           │
           ▼
┌─────────────────────┐
│  Python Test Script │
│  (JSON format)      │
└─────────────────────┘
```

### 测试脚本格式规范

#### 必需的 JSON 输出格式

```json
{
  "pass": true,
  "errors": []
}
```

或失败时：

```json
{
  "pass": false,
  "errors": [
    "file.txt does not exist",
    "Expected content not found in output.log"
  ]
}
```

#### 标准测试脚本模板

```python
#!/usr/bin/env python3
import json
import sys
import os

def main():
    errors = []

    # Test step 1: Check file exists
    if not os.path.exists('/path/to/file.txt'):
        errors.append('file.txt does not exist at /path/to/file.txt')

    # Test step 2: Check file content
    try:
        with open('/path/to/file.txt', 'r') as f:
            content = f.read()
            if 'expected_string' not in content:
                errors.append('Expected string not found in file.txt')
    except Exception as e:
        errors.append(f'Failed to read file.txt: {str(e)}')

    # Test step 3: Check directory structure
    if not os.path.isdir('/path/to/directory'):
        errors.append('directory does not exist')

    # Return JSON result
    result = {
        'pass': len(errors) == 0,
        'errors': errors
    }
    print(json.dumps(result))
    sys.exit(0 if result['pass'] else 1)

if __name__ == '__main__':
    main()
```

### 测试脚本生成 Prompt

系统使用以下 Prompt 模板生成测试脚本（参考 `internal/executor/runner.go:buildTestGenerationPromptFile`）：

```markdown
# Test Generation Task

You need to generate a Python test script based on the task's test method.

## Task Information

**Task ID**: task1
**Task Name**: Create documentation
**Task Goal**: Create comprehensive testing documentation

## Test Method

验证文件已创建：`test -f wiki/testing.md && echo "PASS" || echo "FAIL"`
检查包含核心章节：`grep -q "## 测试策略" wiki/testing.md && echo "PASS" || echo "FAIL"`

## Requirements

1. Create a Python test script at: `/tmp/test_task1.py`
2. The script MUST return a JSON result in this format:
   ```json
   {"pass": true/false, "errors": ["error1", "error2"]}
   ```
3. Implement each test step from the test method above
4. The script should be executable with: `python3 /tmp/test_task1.py`
5. Make sure to handle errors gracefully and report them in the errors array
6. Use absolute paths when checking files

## Example Test Script Structure

```python
#!/usr/bin/env python3
import json
import sys
import os

def main():
    errors = []

    # Test step 1
    if not os.path.exists('file.txt'):
        errors.append('file.txt does not exist')

    # Test step 2
    # ...

    result = {
        'pass': len(errors) == 0,
        'errors': errors
    }
    print(json.dumps(result))
    sys.exit(0 if result['pass'] else 1)

if __name__ == '__main__':
    main()
```

Please generate the test script now. Do NOT execute the task itself, ONLY generate the test script.
```

### 测试脚本执行

#### 代码实现（internal/executor/runner.go）

```go
// ExecuteTestScript executes a Python test script and parses JSON result
func (tr *TaskRunner) ExecuteTestScript(scriptPath string) (bool, string, error) {
    if scriptPath == "" {
        return false, "", fmt.Errorf("script path cannot be empty")
    }

    if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
        return false, "", fmt.Errorf("script file does not exist: %s", scriptPath)
    }

    // Execute the test script
    cmd := exec.Command("python3", scriptPath)
    cmd.Dir = tr.config.WorkspaceDir

    var stdout, stderr bytes.Buffer
    cmd.Stdout = &stdout
    cmd.Stderr = &stderr

    err := cmd.Run()
    output := stdout.String()

    // Parse JSON result
    testResult, parseErr := tr.parseTestResult(output)
    if parseErr != nil {
        return false, output, fmt.Errorf("failed to parse test result: %w", parseErr)
    }

    if !testResult.Pass {
        errMsg := strings.Join(testResult.Errors, "; ")
        return false, output, fmt.Errorf("test did not pass: %s", errMsg)
    }

    return true, output, nil
}

// parseTestResult parses JSON test result from script output
func (tr *TaskRunner) parseTestResult(output string) (*TestResult, error) {
    var result TestResult
    if err := json.Unmarshal([]byte(output), &result); err != nil {
        return nil, fmt.Errorf("invalid JSON output: %w", err)
    }
    return &result, nil
}
```

### Task.md 中的 TestMethod 示例

#### 示例 1: 文件验证

```markdown
### 测试方法
验证文件已创建：`test -f wiki/testing.md && echo "PASS" || echo "FAIL"`
检查包含核心章节：`grep -q "## 测试策略\|## 单元测试\|## 集成测试" wiki/testing.md && echo "PASS" || echo "FAIL"`
验证文档长度（至少 100 行）：`wc -l wiki/testing.md | awk '{if($1>=100) print "PASS"; else print "FAIL"}'`
```

#### 示例 2: 代码编译验证

```markdown
### 测试方法
验证代码编译成功：`go build -o /tmp/rick_test ./cmd/rick && echo "PASS" || echo "FAIL"`
验证单元测试通过：`go test ./internal/parser/... && echo "PASS" || echo "FAIL"`
```

#### 示例 3: 功能验证

```markdown
### 测试方法
验证命令可用：`rick --version && echo "PASS" || echo "FAIL"`
验证配置文件创建：`test -f ~/.rick/config.json && echo "PASS" || echo "FAIL"`
验证工作空间创建：`test -d .rick/jobs && echo "PASS" || echo "FAIL"`
```

---

## 测试覆盖率

### 覆盖率要求

Rick CLI 项目要求：

- **单元测试覆盖率**: ≥ 70%
- **核心模块覆盖率**: ≥ 80%（parser, executor, prompt）
- **关键函数覆盖率**: 100%（DAG 构建、拓扑排序、任务执行）

### 测量测试覆盖率

#### 1. 生成覆盖率报告

```bash
# 运行所有测试并生成覆盖率报告
go test -coverprofile=coverage.out ./...

# 查看覆盖率统计
go tool cover -func=coverage.out

# 生成 HTML 覆盖率报告
go tool cover -html=coverage.out -o coverage.html
```

#### 2. 按模块查看覆盖率

```bash
# Parser 模块覆盖率
go test -coverprofile=coverage_parser.out ./internal/parser/...
go tool cover -func=coverage_parser.out

# Executor 模块覆盖率
go test -coverprofile=coverage_executor.out ./internal/executor/...
go tool cover -func=coverage_executor.out

# Prompt 模块覆盖率
go test -coverprofile=coverage_prompt.out ./internal/prompt/...
go tool cover -func=coverage_prompt.out
```

#### 3. 覆盖率报告示例

```
github.com/sunquan/rick/internal/parser/markdown.go:15:     ParseMarkdownSections    100.0%
github.com/sunquan/rick/internal/parser/markdown.go:45:     extractSection           95.2%
github.com/sunquan/rick/internal/parser/task.go:20:         ParseTaskMarkdown        100.0%
github.com/sunquan/rick/internal/parser/task.go:78:         parseTaskField           88.9%
github.com/sunquan/rick/internal/executor/dag.go:25:        NewDAG                   100.0%
github.com/sunquan/rick/internal/executor/dag.go:65:        TopologicalSort          100.0%
github.com/sunquan/rick/internal/executor/executor.go:30:   Execute                  85.7%
total:                                                       (statements)             82.4%
```

### 覆盖率持续监控

#### 在 CI/CD 中集成覆盖率检查

```bash
#!/bin/bash
# scripts/check_coverage.sh

MIN_COVERAGE=70

go test -coverprofile=coverage.out ./...
COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')

echo "Current coverage: ${COVERAGE}%"
echo "Minimum required: ${MIN_COVERAGE}%"

if (( $(echo "$COVERAGE < $MIN_COVERAGE" | bc -l) )); then
    echo "❌ Coverage is below minimum threshold"
    exit 1
else
    echo "✅ Coverage meets requirement"
    exit 0
fi
```

---

## 测试命令与示例

### 快速测试命令

#### 运行所有测试

```bash
# 运行所有单元测试
go test ./...

# 运行所有测试（详细输出）
go test -v ./...

# 运行所有测试（并行）
go test -parallel 4 ./...
```

#### 运行特定模块测试

```bash
# 测试 parser 模块
go test ./internal/parser/...

# 测试 executor 模块
go test ./internal/executor/...

# 测试 prompt 模块
go test ./internal/prompt/...

# 测试 git 模块
go test ./internal/git/...
```

#### 运行特定测试函数

```bash
# 运行单个测试函数
go test -run TestNewDAG ./internal/executor/

# 运行匹配模式的测试
go test -run TestDAG.* ./internal/executor/

# 运行特定文件的测试
go test ./internal/parser/markdown_test.go
```

#### 运行集成测试

```bash
# 运行完整集成测试
bash scripts/test_integration.sh

# 运行安装测试
bash scripts/test_install.sh

# 运行版本测试
bash scripts/test_version.sh

# 运行更新测试
bash scripts/test_update.sh
```

### 测试选项说明

#### 常用测试标志

```bash
# 显示详细输出
go test -v ./...

# 显示覆盖率
go test -cover ./...

# 生成覆盖率报告
go test -coverprofile=coverage.out ./...

# 运行基准测试
go test -bench=. ./...

# 运行短测试（跳过耗时测试）
go test -short ./...

# 设置超时时间
go test -timeout 30s ./...

# 并行运行测试
go test -parallel 8 ./...

# 显示测试执行时间
go test -v -count=1 ./...
```

### 完整测试流程示例

#### 开发流程中的测试

```bash
# 1. 编写代码和测试
vim internal/parser/markdown.go
vim internal/parser/markdown_test.go

# 2. 运行单元测试
go test ./internal/parser/

# 3. 检查测试覆盖率
go test -coverprofile=coverage.out ./internal/parser/
go tool cover -func=coverage.out

# 4. 如果覆盖率不足，添加更多测试
vim internal/parser/markdown_test.go

# 5. 运行所有测试
go test ./...

# 6. 运行集成测试
bash scripts/test_integration.sh

# 7. 提交代码
git add .
git commit -m "feat(parser): add markdown parsing support"
```

#### 调试失败的测试

```bash
# 1. 运行失败的测试（详细输出）
go test -v -run TestFailingTest ./internal/parser/

# 2. 添加调试输出
# 在测试代码中添加 t.Logf() 或 fmt.Printf()

# 3. 重新运行测试
go test -v -run TestFailingTest ./internal/parser/

# 4. 使用 delve 调试器
dlv test ./internal/parser/ -- -test.run TestFailingTest
```

---

## CI/CD 集成

### GitHub Actions 配置

#### .github/workflows/test.yml

```yaml
name: Test

on:
  push:
    branches: [ main, develop ]
  pull_request:
    branches: [ main, develop ]

jobs:
  test:
    name: Run Tests
    runs-on: ubuntu-latest

    steps:
    - name: Checkout code
      uses: actions/checkout@v3

    - name: Set up Go
      uses: actions/setup-go@v4
      with:
        go-version: '1.21'

    - name: Install dependencies
      run: go mod download

    - name: Run unit tests
      run: go test -v -coverprofile=coverage.out ./...

    - name: Check coverage
      run: |
        COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')
        echo "Coverage: ${COVERAGE}%"
        if (( $(echo "$COVERAGE < 70" | bc -l) )); then
          echo "❌ Coverage is below 70%"
          exit 1
        fi

    - name: Run integration tests
      run: bash scripts/test_integration.sh

    - name: Upload coverage to Codecov
      uses: codecov/codecov-action@v3
      with:
        file: ./coverage.out
        flags: unittests
        name: codecov-umbrella
```

### GitLab CI 配置

#### .gitlab-ci.yml

```yaml
stages:
  - test
  - integration

unit-test:
  stage: test
  image: golang:1.21
  script:
    - go mod download
    - go test -v -coverprofile=coverage.out ./...
    - go tool cover -func=coverage.out
  coverage: '/total:\s+\(statements\)\s+(\d+\.\d+)%/'
  artifacts:
    reports:
      coverage_report:
        coverage_format: cobertura
        path: coverage.xml

integration-test:
  stage: integration
  image: golang:1.21
  script:
    - bash scripts/test_integration.sh
  dependencies:
    - unit-test
```

### Pre-commit Hooks

#### .git/hooks/pre-commit

```bash
#!/bin/bash
#
# Pre-commit hook to run tests before commit
#

echo "Running pre-commit tests..."

# Run unit tests
echo "Running unit tests..."
if ! go test ./...; then
    echo "❌ Unit tests failed. Commit aborted."
    exit 1
fi

# Check test coverage
echo "Checking test coverage..."
go test -coverprofile=coverage.out ./...
COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')

if (( $(echo "$COVERAGE < 70" | bc -l) )); then
    echo "❌ Coverage is ${COVERAGE}%, below 70% threshold. Commit aborted."
    exit 1
fi

echo "✅ All pre-commit checks passed (coverage: ${COVERAGE}%)"
exit 0
```

### Makefile 集成

#### Makefile

```makefile
.PHONY: test test-unit test-integration test-coverage test-all

# Run all unit tests
test-unit:
	go test ./...

# Run all unit tests with verbose output
test-unit-verbose:
	go test -v ./...

# Run integration tests
test-integration:
	bash scripts/test_integration.sh

# Generate coverage report
test-coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Check coverage threshold
test-coverage-check:
	@go test -coverprofile=coverage.out ./...
	@COVERAGE=$$(go tool cover -func=coverage.out | grep total | awk '{print $$3}' | sed 's/%//'); \
	echo "Current coverage: $${COVERAGE}%"; \
	if [ $$(echo "$${COVERAGE} < 70" | bc -l) -eq 1 ]; then \
		echo "❌ Coverage is below 70%"; \
		exit 1; \
	else \
		echo "✅ Coverage meets requirement"; \
	fi

# Run all tests (unit + integration)
test-all: test-unit test-integration

# Clean test artifacts
test-clean:
	rm -f coverage.out coverage.html
	find . -name "*.test" -delete
```

使用方法：

```bash
# 运行单元测试
make test-unit

# 运行集成测试
make test-integration

# 生成覆盖率报告
make test-coverage

# 检查覆盖率是否达标
make test-coverage-check

# 运行所有测试
make test-all

# 清理测试产物
make test-clean
```

---

## 测试最佳实践

### 1. 测试命名规范

```go
// ✅ 好的测试命名
func TestNewDAGEmpty(t *testing.T) { }
func TestNewDAGSingleTask(t *testing.T) { }
func TestNewDAGWithCycle(t *testing.T) { }

// ❌ 不好的测试命名
func TestDAG1(t *testing.T) { }
func TestFunction(t *testing.T) { }
```

### 2. 表驱动测试

```go
// ✅ 使用表驱动测试
func TestParseTask(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    *Task
        wantErr bool
    }{
        {"valid task", "# 任务名称\nTest", &Task{Name: "Test"}, false},
        {"invalid task", "", nil, true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := ParseTask(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("ParseTask() error = %v, wantErr %v", err, tt.wantErr)
            }
            // ... more assertions
        })
    }
}
```

### 3. 使用子测试

```go
func TestTaskExecution(t *testing.T) {
    t.Run("successful execution", func(t *testing.T) {
        // Test successful case
    })

    t.Run("execution with retry", func(t *testing.T) {
        // Test retry case
    })

    t.Run("execution failure", func(t *testing.T) {
        // Test failure case
    })
}
```

### 4. 清理测试资源

```go
func TestWorkspace(t *testing.T) {
    tmpDir, err := os.MkdirTemp("", "rick-test-*")
    if err != nil {
        t.Fatal(err)
    }
    defer os.RemoveAll(tmpDir) // 确保清理

    // Run tests...
}
```

### 5. 测试错误情况

```go
func TestParseTaskMarkdown(t *testing.T) {
    // Test normal case
    t.Run("valid input", func(t *testing.T) {
        // ...
    })

    // Test error cases
    t.Run("empty input", func(t *testing.T) {
        _, err := ParseTaskMarkdown("")
        if err == nil {
            t.Error("Expected error for empty input")
        }
    })

    t.Run("invalid format", func(t *testing.T) {
        _, err := ParseTaskMarkdown("invalid")
        if err == nil {
            t.Error("Expected error for invalid format")
        }
    })
}
```

### 6. 避免测试依赖

```go
// ❌ 不好：测试之间有依赖
var globalTask *Task

func TestCreateTask(t *testing.T) {
    globalTask = &Task{ID: "task1"}
}

func TestUseTask(t *testing.T) {
    // 依赖 TestCreateTask 的结果
    if globalTask == nil {
        t.Fatal("globalTask is nil")
    }
}

// ✅ 好：每个测试独立
func TestCreateTask(t *testing.T) {
    task := &Task{ID: "task1"}
    // Test with task
}

func TestUseTask(t *testing.T) {
    task := &Task{ID: "task1"} // 独立创建
    // Test with task
}
```

### 7. 使用测试辅助函数

```go
// Helper functions
func createTestTask(id, name string) *Task {
    return &Task{
        ID:   id,
        Name: name,
        Goal: "test goal",
    }
}

func assertTaskEqual(t *testing.T, got, want *Task) {
    t.Helper() // 标记为辅助函数
    if got.ID != want.ID {
        t.Errorf("ID = %v, want %v", got.ID, want.ID)
    }
    if got.Name != want.Name {
        t.Errorf("Name = %v, want %v", got.Name, want.Name)
    }
}
```

### 8. 测试并发安全

```go
func TestConcurrentAccess(t *testing.T) {
    var wg sync.WaitGroup
    dag := NewDAG()

    for i := 0; i < 100; i++ {
        wg.Add(1)
        go func(id int) {
            defer wg.Done()
            task := createTestTask(fmt.Sprintf("task%d", id), "Test")
            dag.AddTask(task)
        }(i)
    }

    wg.Wait()

    if dag.TaskCount() != 100 {
        t.Errorf("Expected 100 tasks, got %d", dag.TaskCount())
    }
}
```

---

## 总结

Rick CLI 的测试体系包含：

1. **任务测试脚本**: 自动生成的 Python JSON 格式测试，验证每个任务的执行结果
2. **单元测试**: 使用 Go testing 包，覆盖所有核心模块
3. **集成测试**: 使用 Bash 脚本，测试完整的安装和使用流程
4. **E2E 测试**: 测试完整的 plan → doing → learning 工作流

测试覆盖率要求：
- 单元测试覆盖率 ≥ 70%
- 核心模块覆盖率 ≥ 80%
- 关键函数覆盖率 100%

通过这套完整的测试体系，Rick CLI 确保了代码质量和功能的正确性。
