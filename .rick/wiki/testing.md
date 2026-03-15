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

Rick CLI 采用多层次的测试策略，确保代码质量和系统稳定性。测试金字塔从底层到顶层包括：

### 1. 单元测试（Unit Tests）

**目标**: 验证单个函数、方法和模块的正确性

**特点**:
- 使用 Go 标准库 `testing` 包
- 测试文件命名规范：`*_test.go`
- 覆盖率目标：≥ 70%
- 执行速度快，隔离性强

**覆盖范围**:
- 核心模块：parser, executor, prompt, git, config, workspace
- 工具包：pkg/errors, pkg/feedback
- 命令处理器：internal/cmd

### 2. 集成测试（Integration Tests）

**目标**: 验证多个模块协同工作的正确性

**特点**:
- 使用 Shell 脚本实现（`scripts/test_*.sh`）
- 测试真实的命令行交互
- 验证文件系统操作、Git 操作等

**覆盖范围**:
- 安装脚本集成测试（`test_integration.sh`）
- 版本管理测试（`test_version.sh`）
- 环境检查测试（`test_check_env.sh`）
- 完整工作流测试（plan → doing → learning）

### 3. 端到端测试（E2E Tests）

**目标**: 验证完整的用户工作流

**特点**:
- 模拟真实用户场景
- 测试任务执行的完整生命周期
- 验证 Claude Code CLI 集成

**覆盖范围**:
- 任务规划 → 执行 → 学习的完整循环
- 多任务 DAG 执行
- 失败重试机制
- Git 自动提交

### 4. 任务测试脚本（Task Test Scripts）

**目标**: 验证每个任务的执行结果

**特点**:
- 自动生成 Python 测试脚本
- JSON 格式定义测试用例
- 支持多种断言类型

**覆盖范围**:
- 文件存在性检查
- 文件内容验证
- 命令执行结果验证
- 代码质量检查

---

## 单元测试

### 测试框架：Go testing

Rick CLI 使用 Go 标准库中的 `testing` 包进行单元测试，无需额外依赖。

#### 测试文件结构

```
internal/
  executor/
    dag.go              # 实现代码
    dag_test.go         # 测试代码
    topological.go
    topological_test.go
  parser/
    markdown.go
    markdown_test.go
```

#### 基本测试模式

##### 1. 简单函数测试

```go
package executor

import "testing"

// 测试单个函数
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

##### 2. 表驱动测试（Table-Driven Tests）

表驱动测试是 Go 社区推荐的测试模式，适合测试多个输入输出组合：

```go
func TestTopologicalSort(t *testing.T) {
    tests := []struct {
        name     string
        tasks    []*parser.Task
        wantErr  bool
        validate func(*testing.T, []*parser.Task)
    }{
        {
            name: "empty DAG",
            tasks: []*parser.Task{},
            wantErr: false,
            validate: func(t *testing.T, result []*parser.Task) {
                if len(result) != 0 {
                    t.Errorf("Expected empty result, got %d tasks", len(result))
                }
            },
        },
        {
            name: "single task",
            tasks: []*parser.Task{
                createTestTask("task1", "Task 1", "Goal 1", []string{}),
            },
            wantErr: false,
            validate: func(t *testing.T, result []*parser.Task) {
                if len(result) != 1 {
                    t.Errorf("Expected 1 task, got %d", len(result))
                }
                if result[0].ID != "task1" {
                    t.Errorf("Expected task1, got %s", result[0].ID)
                }
            },
        },
        {
            name: "circular dependency",
            tasks: []*parser.Task{
                createTestTask("task1", "Task 1", "Goal 1", []string{"task2"}),
                createTestTask("task2", "Task 2", "Goal 2", []string{"task1"}),
            },
            wantErr: true,
            validate: nil,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            dag, err := NewDAG(tt.tasks)
            if err != nil {
                if !tt.wantErr {
                    t.Fatalf("Unexpected error: %v", err)
                }
                return
            }

            result, err := dag.TopologicalSort()
            if (err != nil) != tt.wantErr {
                t.Errorf("TopologicalSort() error = %v, wantErr %v", err, tt.wantErr)
                return
            }

            if tt.validate != nil {
                tt.validate(t, result)
            }
        })
    }
}
```

##### 3. 测试辅助函数

创建辅助函数简化测试代码：

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

// Helper function to compare task slices
func compareTasks(t *testing.T, got, want []*parser.Task) {
    if len(got) != len(want) {
        t.Errorf("Length mismatch: got %d, want %d", len(got), len(want))
        return
    }
    for i := range got {
        if got[i].ID != want[i].ID {
            t.Errorf("Task %d: got %s, want %s", i, got[i].ID, want[i].ID)
        }
    }
}
```

##### 4. 错误处理测试

```go
func TestParseMarkdownInvalidInput(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        wantErr string
    }{
        {
            name:    "empty input",
            input:   "",
            wantErr: "empty markdown content",
        },
        {
            name:    "invalid format",
            input:   "# No dependencies section",
            wantErr: "missing dependencies section",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            _, err := ParseMarkdown(tt.input)
            if err == nil {
                t.Errorf("Expected error containing '%s', got nil", tt.wantErr)
                return
            }
            if !strings.Contains(err.Error(), tt.wantErr) {
                t.Errorf("Expected error containing '%s', got '%s'", tt.wantErr, err.Error())
            }
        })
    }
}
```

#### 测试覆盖的关键模块

##### 1. Parser 模块测试

```go
// internal/parser/markdown_test.go
func TestParseTaskMarkdown(t *testing.T) {
    markdown := `# 依赖关系
task1, task2

# 任务名称
测试任务

# 任务目标
完成测试

# 关键结果
1. 结果1
2. 结果2

# 测试方法
运行测试脚本`

    task, err := ParseTaskMarkdown(markdown)
    if err != nil {
        t.Fatalf("Failed to parse markdown: %v", err)
    }

    if task.Name != "测试任务" {
        t.Errorf("Expected task name '测试任务', got '%s'", task.Name)
    }

    if len(task.Dependencies) != 2 {
        t.Errorf("Expected 2 dependencies, got %d", len(task.Dependencies))
    }
}
```

##### 2. Executor 模块测试

```go
// internal/executor/dag_test.go
func TestDAGCycleDetection(t *testing.T) {
    tasks := []*parser.Task{
        createTestTask("task1", "Task 1", "Goal 1", []string{"task2"}),
        createTestTask("task2", "Task 2", "Goal 2", []string{"task3"}),
        createTestTask("task3", "Task 3", "Goal 3", []string{"task1"}),
    }

    dag, err := NewDAG(tasks)
    if err != nil {
        t.Fatalf("Failed to create DAG: %v", err)
    }

    _, err = dag.TopologicalSort()
    if err == nil {
        t.Error("Expected cycle detection error, got nil")
    }
}
```

##### 3. Prompt 模块测试

```go
// internal/prompt/builder_test.go
func TestPromptBuilder(t *testing.T) {
    template := "Hello {{name}}, your age is {{age}}"
    builder := NewPromptBuilder(template)

    builder.SetVariable("name", "Alice")
    builder.SetVariable("age", "30")

    result, err := builder.Build()
    if err != nil {
        t.Fatalf("Failed to build prompt: %v", err)
    }

    expected := "Hello Alice, your age is 30"
    if result != expected {
        t.Errorf("Expected '%s', got '%s'", expected, result)
    }
}
```

---

## 集成测试

### Shell 脚本测试框架

Rick CLI 使用 Shell 脚本实现集成测试，验证多个模块的协同工作。

#### 测试脚本结构

```bash
#!/bin/bash
#
# test_integration.sh - Integration test for Rick CLI
#

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

# Colors for output
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
echo -e "${YELLOW}Test 1: Verify build${NC}"
if bash "$SCRIPT_DIR/build.sh" > /tmp/build_output.log 2>&1; then
    echo -e "${GREEN}✓ PASS${NC}: build.sh compiles successfully"
    ((TESTS_PASSED++))
else
    echo -e "${RED}✗ FAIL${NC}: build.sh compilation failed"
    ((TESTS_FAILED++))
fi

# Test 2: Verify binary
if [ -f "$PROJECT_DIR/bin/rick" ]; then
    echo -e "${GREEN}✓${NC} Binary created"
    ((TESTS_PASSED++))
else
    echo -e "${RED}✗${NC} Binary not found"
    ((TESTS_FAILED++))
fi

# Summary
echo ""
echo -e "${BLUE}Test Summary${NC}"
echo -e "Passed: ${GREEN}$TESTS_PASSED${NC}"
echo -e "Failed: ${RED}$TESTS_FAILED${NC}"

if [ $TESTS_FAILED -eq 0 ]; then
    echo -e "${GREEN}All tests passed!${NC}"
    exit 0
else
    echo -e "${RED}Some tests failed!${NC}"
    exit 1
fi
```

#### 集成测试示例

##### 1. 安装脚本集成测试

```bash
# scripts/test_integration.sh

# Test install.sh
echo -e "${YELLOW}Test: Verify install.sh${NC}"
if bash "$SCRIPT_DIR/install.sh" --help > /tmp/install_help.log 2>&1; then
    echo -e "${GREEN}✓ PASS${NC}: install.sh --help works"
    ((TESTS_PASSED++))
else
    echo -e "${RED}✗ FAIL${NC}: install.sh --help failed"
    ((TESTS_FAILED++))
fi

if grep -q "source" /tmp/install_help.log; then
    echo -e "${GREEN}✓${NC} install.sh supports source installation"
    ((TESTS_PASSED++))
fi
```

##### 2. 版本管理测试

```bash
# scripts/test_version.sh

# Test version.sh get
echo -e "${YELLOW}Test: Get version${NC}"
if bash "$SCRIPT_DIR/version.sh" get > /tmp/version_output.log 2>&1; then
    echo -e "${GREEN}✓ PASS${NC}: version.sh get works"
    ((TESTS_PASSED++))
else
    echo -e "${RED}✗ FAIL${NC}: version.sh get failed"
    ((TESTS_FAILED++))
fi

# Verify version format
VERSION=$(cat /tmp/version_output.log)
if [[ $VERSION =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
    echo -e "${GREEN}✓${NC} Version format is valid: $VERSION"
    ((TESTS_PASSED++))
else
    echo -e "${RED}✗${NC} Invalid version format: $VERSION"
    ((TESTS_FAILED++))
fi
```

##### 3. 环境检查测试

```bash
# scripts/test_check_env.sh

# Test environment check
echo -e "${YELLOW}Test: Environment check${NC}"
if bash "$SCRIPT_DIR/check_env.sh" > /tmp/env_check.log 2>&1; then
    echo -e "${GREEN}✓ PASS${NC}: check_env.sh works"
    ((TESTS_PASSED++))
else
    # May exit with error if requirements not met
    echo -e "${YELLOW}⚠ WARNING${NC}: check_env.sh reports missing dependencies"
    ((TESTS_PASSED++))
fi
```

---

## 测试脚本生成机制

Rick CLI 在执行任务时会自动生成测试脚本，验证任务执行结果。测试脚本使用 **Python + JSON** 格式定义。

### 测试脚本格式

#### JSON 格式定义

```json
{
  "test_name": "验证文件创建",
  "test_cases": [
    {
      "name": "检查文件存在",
      "type": "file_exists",
      "path": "wiki/testing.md",
      "expected": true
    },
    {
      "name": "检查文件内容",
      "type": "file_contains",
      "path": "wiki/testing.md",
      "pattern": "## 测试策略",
      "expected": true
    },
    {
      "name": "检查行数",
      "type": "command",
      "command": "wc -l wiki/testing.md | awk '{print $1}'",
      "expected": ">= 100"
    },
    {
      "name": "检查代码示例",
      "type": "regex_match",
      "path": "wiki/testing.md",
      "pattern": "```(go|python|bash)",
      "expected": true
    }
  ]
}
```

#### Python 测试脚本示例

```python
#!/usr/bin/env python3
"""
Rick CLI Task Test Script
Auto-generated test script for task validation
"""

import json
import os
import re
import subprocess
import sys

def file_exists(path):
    """Check if file exists"""
    return os.path.exists(path)

def file_contains(path, pattern):
    """Check if file contains pattern"""
    try:
        with open(path, 'r', encoding='utf-8') as f:
            content = f.read()
            return pattern in content
    except Exception as e:
        print(f"Error reading file: {e}")
        return False

def regex_match(path, pattern):
    """Check if file matches regex pattern"""
    try:
        with open(path, 'r', encoding='utf-8') as f:
            content = f.read()
            return re.search(pattern, content) is not None
    except Exception as e:
        print(f"Error reading file: {e}")
        return False

def run_command(command):
    """Run shell command and return output"""
    try:
        result = subprocess.run(
            command,
            shell=True,
            capture_output=True,
            text=True,
            timeout=30
        )
        return result.stdout.strip()
    except Exception as e:
        print(f"Error running command: {e}")
        return None

def evaluate_comparison(actual, expected):
    """Evaluate comparison expression"""
    if expected.startswith(">="):
        threshold = int(expected.split(">=")[1].strip())
        return int(actual) >= threshold
    elif expected.startswith(">"):
        threshold = int(expected.split(">")[1].strip())
        return int(actual) > threshold
    elif expected.startswith("<="):
        threshold = int(expected.split("<=")[1].strip())
        return int(actual) <= threshold
    elif expected.startswith("<"):
        threshold = int(expected.split("<")[1].strip())
        return int(actual) < threshold
    else:
        return str(actual) == str(expected)

def run_tests(test_config):
    """Run all test cases"""
    passed = 0
    failed = 0

    print(f"\n{'='*60}")
    print(f"Test: {test_config['test_name']}")
    print(f"{'='*60}\n")

    for i, test_case in enumerate(test_config['test_cases'], 1):
        name = test_case['name']
        test_type = test_case['type']

        print(f"[{i}/{len(test_config['test_cases'])}] {name}...", end=' ')

        result = False

        if test_type == 'file_exists':
            result = file_exists(test_case['path']) == test_case['expected']

        elif test_type == 'file_contains':
            result = file_contains(test_case['path'], test_case['pattern']) == test_case['expected']

        elif test_type == 'regex_match':
            result = regex_match(test_case['path'], test_case['pattern']) == test_case['expected']

        elif test_type == 'command':
            output = run_command(test_case['command'])
            if output is not None:
                result = evaluate_comparison(output, test_case['expected'])

        if result:
            print("✓ PASS")
            passed += 1
        else:
            print("✗ FAIL")
            failed += 1

    print(f"\n{'='*60}")
    print(f"Results: {passed} passed, {failed} failed")
    print(f"{'='*60}\n")

    return failed == 0

if __name__ == '__main__':
    # Load test configuration
    test_config = {
        "test_name": "Task Validation",
        "test_cases": [
            # Test cases will be generated here
        ]
    }

    success = run_tests(test_config)
    sys.exit(0 if success else 1)
```

### 测试脚本生成流程

1. **任务执行前**：解析 task.md 中的 `# 测试方法` 部分
2. **生成测试脚本**：根据测试方法生成 Python 测试脚本
3. **执行任务**：调用 Claude Code CLI 执行任务
4. **运行测试**：执行生成的测试脚本验证结果
5. **记录结果**：
   - 测试通过：标记任务完成，git commit
   - 测试失败：记录到 debug.md，重试或退出

### 支持的测试类型

| 测试类型 | 说明 | 示例 |
|---------|------|------|
| `file_exists` | 检查文件是否存在 | `test -f wiki/testing.md` |
| `file_contains` | 检查文件包含特定内容 | `grep -q "测试策略" wiki/testing.md` |
| `regex_match` | 检查文件匹配正则表达式 | `grep -E '```(go\|python)' wiki/testing.md` |
| `command` | 执行命令并验证输出 | `wc -l wiki/testing.md \| awk '{print $1}'` |
| `comparison` | 数值比较 | `>= 100`, `< 50`, `== 42` |

---

## 测试覆盖率

### 覆盖率目标

Rick CLI 的测试覆盖率目标：

| 模块 | 目标覆盖率 | 当前状态 |
|------|-----------|---------|
| Parser | ≥ 80% | ✓ |
| Executor | ≥ 80% | ✓ |
| Prompt | ≥ 70% | ✓ |
| Git | ≥ 70% | ✓ |
| Config | ≥ 70% | ✓ |
| Workspace | ≥ 70% | ✓ |
| 整体 | ≥ 70% | ✓ |

### 生成覆盖率报告

#### 1. 基本覆盖率检查

```bash
# 运行所有测试并显示覆盖率
go test -cover ./...
```

输出示例：

```
ok      github.com/sunquan/rick/internal/parser    0.234s  coverage: 82.5% of statements
ok      github.com/sunquan/rick/internal/executor  0.156s  coverage: 85.3% of statements
ok      github.com/sunquan/rick/internal/prompt    0.198s  coverage: 76.8% of statements
```

#### 2. 详细覆盖率报告

```bash
# 生成覆盖率文件
go test -coverprofile=coverage.out ./...

# 查看覆盖率详情
go tool cover -func=coverage.out

# 生成 HTML 报告
go tool cover -html=coverage.out -o coverage.html
```

#### 3. 按模块查看覆盖率

```bash
# 查看特定模块的覆盖率
go test -cover ./internal/parser
go test -cover ./internal/executor
go test -cover ./internal/prompt
```

#### 4. 查找未覆盖的代码

```bash
# 生成覆盖率报告并查找未覆盖的行
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out | grep -v "100.0%"
```

### 覆盖率分析

#### HTML 覆盖率报告

```bash
# 生成并打开 HTML 报告
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

HTML 报告特点：
- 🟢 绿色：已覆盖的代码
- 🔴 红色：未覆盖的代码
- 🟡 灰色：不可执行的代码（注释、声明等）

---

## 测试命令与示例

### 单元测试命令

#### 1. 运行所有测试

```bash
# 运行所有测试
go test ./...

# 显示详细输出
go test -v ./...

# 并行运行测试
go test -parallel 4 ./...
```

#### 2. 运行特定模块的测试

```bash
# 测试 parser 模块
go test ./internal/parser

# 测试 executor 模块
go test ./internal/executor

# 测试 prompt 模块
go test ./internal/prompt
```

#### 3. 运行特定测试函数

```bash
# 运行特定测试
go test -run TestNewDAG ./internal/executor

# 运行匹配模式的测试
go test -run "TestDAG.*" ./internal/executor

# 运行多个测试
go test -run "TestNewDAG|TestTopological" ./internal/executor
```

#### 4. 测试超时控制

```bash
# 设置测试超时（默认 10 分钟）
go test -timeout 30s ./...

# 禁用超时
go test -timeout 0 ./...
```

#### 5. 测试失败时继续

```bash
# 失败后继续运行其他测试
go test -failfast=false ./...

# 遇到第一个失败就停止
go test -failfast ./...
```

### 集成测试命令

#### 1. 运行集成测试脚本

```bash
# 运行所有集成测试
bash scripts/test_integration.sh

# 运行特定集成测试
bash scripts/test_install.sh
bash scripts/test_version.sh
bash scripts/test_check_env.sh
```

#### 2. 端到端测试

```bash
# 完整工作流测试
cd /tmp/rick-test
rick plan "测试项目"
rick doing job_0
rick learning job_0
```

### 测试调试

#### 1. 打印调试信息

```go
func TestDebugExample(t *testing.T) {
    result := SomeFunction()
    t.Logf("Result: %+v", result)  // 打印调试信息

    if result != expected {
        t.Errorf("Expected %v, got %v", expected, result)
    }
}
```

#### 2. 跳过测试

```go
func TestLongRunning(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping long-running test in short mode")
    }
    // Long-running test code
}
```

运行时跳过长测试：

```bash
go test -short ./...
```

#### 3. 并行测试

```go
func TestParallel1(t *testing.T) {
    t.Parallel()  // 标记为并行测试
    // Test code
}

func TestParallel2(t *testing.T) {
    t.Parallel()
    // Test code
}
```

### 完整测试示例

```bash
#!/bin/bash
# 完整的测试流程

echo "=== Running Unit Tests ==="
go test -v -cover ./...

echo ""
echo "=== Running Integration Tests ==="
bash scripts/test_integration.sh

echo ""
echo "=== Generating Coverage Report ==="
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html

echo ""
echo "=== Coverage Summary ==="
go tool cover -func=coverage.out | tail -1

echo ""
echo "✓ All tests completed!"
```

---

## CI/CD 集成

### GitHub Actions 配置

创建 `.github/workflows/test.yml`：

```yaml
name: Test

on:
  push:
    branches: [ main, master, develop ]
  pull_request:
    branches: [ main, master, develop ]

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

    - name: Cache Go modules
      uses: actions/cache@v3
      with:
        path: ~/go/pkg/mod
        key: ${{ runner.os }}-go-${{ hashFiles('**/go.sum') }}
        restore-keys: |
          ${{ runner.os }}-go-

    - name: Download dependencies
      run: go mod download

    - name: Run unit tests
      run: go test -v -race -coverprofile=coverage.out ./...

    - name: Generate coverage report
      run: go tool cover -html=coverage.out -o coverage.html

    - name: Upload coverage report
      uses: actions/upload-artifact@v3
      with:
        name: coverage-report
        path: coverage.html

    - name: Check coverage threshold
      run: |
        COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')
        echo "Total coverage: $COVERAGE%"
        if (( $(echo "$COVERAGE < 70" | bc -l) )); then
          echo "Coverage $COVERAGE% is below threshold 70%"
          exit 1
        fi

    - name: Run integration tests
      run: bash scripts/test_integration.sh

    - name: Build binary
      run: bash scripts/build.sh

    - name: Verify binary
      run: |
        ./bin/rick --version
        ./bin/rick --help

  lint:
    name: Run Linters
    runs-on: ubuntu-latest

    steps:
    - name: Checkout code
      uses: actions/checkout@v3

    - name: Set up Go
      uses: actions/setup-go@v4
      with:
        go-version: '1.21'

    - name: Run golangci-lint
      uses: golangci/golangci-lint-action@v3
      with:
        version: latest
        args: --timeout=5m
```

### GitLab CI 配置

创建 `.gitlab-ci.yml`：

```yaml
stages:
  - test
  - build
  - deploy

variables:
  GO_VERSION: "1.21"

test:unit:
  stage: test
  image: golang:${GO_VERSION}
  script:
    - go mod download
    - go test -v -race -coverprofile=coverage.out ./...
    - go tool cover -func=coverage.out
  coverage: '/total:\s+\(statements\)\s+(\d+\.\d+)%/'
  artifacts:
    paths:
      - coverage.out
    reports:
      coverage_report:
        coverage_format: cobertura
        path: coverage.out

test:integration:
  stage: test
  image: golang:${GO_VERSION}
  script:
    - bash scripts/test_integration.sh

build:
  stage: build
  image: golang:${GO_VERSION}
  script:
    - bash scripts/build.sh
  artifacts:
    paths:
      - bin/rick
    expire_in: 1 week
```

### 本地 Pre-commit Hook

创建 `.git/hooks/pre-commit`：

```bash
#!/bin/bash
#
# Pre-commit hook: Run tests before commit
#

echo "Running tests before commit..."

# Run unit tests
go test ./...
if [ $? -ne 0 ]; then
    echo "❌ Unit tests failed. Commit aborted."
    exit 1
fi

# Check code formatting
gofmt -l . | grep -v "^$"
if [ $? -eq 0 ]; then
    echo "❌ Code not formatted. Run 'gofmt -w .' and try again."
    exit 1
fi

# Run linter (if available)
if command -v golangci-lint &> /dev/null; then
    golangci-lint run
    if [ $? -ne 0 ]; then
        echo "❌ Linter found issues. Commit aborted."
        exit 1
    fi
fi

echo "✅ All checks passed. Proceeding with commit."
exit 0
```

使 hook 可执行：

```bash
chmod +x .git/hooks/pre-commit
```

### Makefile 集成

创建 `Makefile`：

```makefile
.PHONY: test test-unit test-integration test-coverage test-all clean

# Run unit tests
test-unit:
	@echo "Running unit tests..."
	go test -v ./...

# Run integration tests
test-integration:
	@echo "Running integration tests..."
	bash scripts/test_integration.sh

# Run all tests
test: test-unit test-integration

# Generate coverage report
test-coverage:
	@echo "Generating coverage report..."
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# Run tests with race detector
test-race:
	@echo "Running tests with race detector..."
	go test -race ./...

# Run all tests and checks
test-all: test-unit test-integration test-coverage
	@echo "All tests completed!"

# Clean test artifacts
clean:
	rm -f coverage.out coverage.html
	rm -rf /tmp/rick-test-*
```

使用 Makefile：

```bash
# 运行单元测试
make test-unit

# 运行集成测试
make test-integration

# 运行所有测试
make test

# 生成覆盖率报告
make test-coverage

# 运行所有测试和检查
make test-all

# 清理测试产物
make clean
```

---

## 测试最佳实践

### 1. 测试命名规范

```go
// ✓ 好的命名
func TestNewDAGWithEmptyTaskList(t *testing.T) { }
func TestTopologicalSortDetectsCycle(t *testing.T) { }
func TestParseMarkdownInvalidFormat(t *testing.T) { }

// ✗ 不好的命名
func TestDAG1(t *testing.T) { }
func TestSort(t *testing.T) { }
func TestParse(t *testing.T) { }
```

### 2. 测试隔离

```go
// ✓ 每个测试独立，不依赖其他测试
func TestCreateFile(t *testing.T) {
    tmpDir := t.TempDir()  // 自动清理
    file := filepath.Join(tmpDir, "test.txt")
    // Test code
}

// ✗ 测试之间有依赖
var globalState string
func TestSetState(t *testing.T) {
    globalState = "value"
}
func TestUseState(t *testing.T) {
    // 依赖 TestSetState
    if globalState != "value" {
        t.Error("State not set")
    }
}
```

### 3. 使用子测试

```go
func TestParseMarkdown(t *testing.T) {
    t.Run("valid input", func(t *testing.T) {
        // Test valid input
    })

    t.Run("empty input", func(t *testing.T) {
        // Test empty input
    })

    t.Run("invalid format", func(t *testing.T) {
        // Test invalid format
    })
}
```

### 4. 测试错误情况

```go
func TestFunctionWithError(t *testing.T) {
    // Test success case
    result, err := Function(validInput)
    if err != nil {
        t.Fatalf("Unexpected error: %v", err)
    }

    // Test error case
    _, err = Function(invalidInput)
    if err == nil {
        t.Error("Expected error, got nil")
    }
}
```

### 5. 使用测试辅助函数

```go
// 测试辅助函数
func assertEqual(t *testing.T, got, want interface{}) {
    t.Helper()  // 标记为辅助函数
    if got != want {
        t.Errorf("got %v, want %v", got, want)
    }
}

func TestExample(t *testing.T) {
    result := SomeFunction()
    assertEqual(t, result, expected)
}
```

### 6. 清理测试资源

```go
func TestWithCleanup(t *testing.T) {
    // 创建临时资源
    tmpFile, err := os.CreateTemp("", "test-*.txt")
    if err != nil {
        t.Fatalf("Failed to create temp file: %v", err)
    }

    // 注册清理函数
    t.Cleanup(func() {
        os.Remove(tmpFile.Name())
    })

    // Test code
}
```

### 7. 并行测试

```go
func TestParallel(t *testing.T) {
    tests := []struct {
        name string
        input string
    }{
        {"case1", "input1"},
        {"case2", "input2"},
        {"case3", "input3"},
    }

    for _, tt := range tests {
        tt := tt  // 捕获循环变量
        t.Run(tt.name, func(t *testing.T) {
            t.Parallel()  // 并行运行
            // Test code
        })
    }
}
```

### 8. Mock 和 Stub

```go
// 定义接口
type FileReader interface {
    Read(path string) ([]byte, error)
}

// Mock 实现
type MockFileReader struct {
    ReadFunc func(path string) ([]byte, error)
}

func (m *MockFileReader) Read(path string) ([]byte, error) {
    return m.ReadFunc(path)
}

// 测试中使用 Mock
func TestWithMock(t *testing.T) {
    mock := &MockFileReader{
        ReadFunc: func(path string) ([]byte, error) {
            return []byte("test content"), nil
        },
    }

    result := ProcessFile(mock, "test.txt")
    // Assertions
}
```

---

## 总结

Rick CLI 的测试体系包括：

1. **单元测试**：使用 Go testing 包，覆盖率 ≥ 70%
2. **集成测试**：使用 Shell 脚本，验证多模块协同
3. **任务测试脚本**：自动生成 Python 测试脚本，验证任务执行结果
4. **CI/CD 集成**：GitHub Actions / GitLab CI 自动化测试

测试是保证代码质量的关键，遵循测试最佳实践可以提高测试的有效性和可维护性。
