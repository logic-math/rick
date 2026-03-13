#!/bin/bash

# DAG Executor Integration Test Script
# Tests the complete dag_executor module functionality

set -e

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test counters
PASSED=0
FAILED=0
TESTS=()

# Logging function
log_test() {
    local name=$1
    local result=$2
    if [ "$result" = "PASS" ]; then
        echo -e "${GREEN}✓ PASS${NC}: $name"
        ((PASSED++))
    else
        echo -e "${RED}✗ FAIL${NC}: $name"
        ((FAILED++))
    fi
    TESTS+=("$name: $result")
}

# Helper function to run Go tests
run_go_test() {
    local test_name=$1
    local test_file=$2
    local test_func=$3

    echo -n "Running: $test_name ... "
    if go test -run "$test_func" -v "$test_file" > /tmp/test_output.log 2>&1; then
        log_test "$test_name" "PASS"
    else
        log_test "$test_name" "FAIL"
        echo "  Error output:"
        tail -5 /tmp/test_output.log | sed 's/^/    /'
    fi
}

# Helper function to verify file exists
verify_file() {
    local file=$1
    local desc=$2

    if [ -f "$file" ]; then
        log_test "$desc" "PASS"
        return 0
    else
        log_test "$desc" "FAIL"
        return 1
    fi
}

# Helper function to verify directory exists
verify_dir() {
    local dir=$1
    local desc=$2

    if [ -d "$dir" ]; then
        log_test "$desc" "PASS"
        return 0
    else
        log_test "$desc" "FAIL"
        return 1
    fi
}

echo "======================================"
echo "DAG Executor Integration Tests"
echo "======================================"
echo

# Test 1: Verify implementation files exist
echo -e "${BLUE}Test 1: Verify Implementation Files${NC}"
verify_file "internal/executor/dag.go" "DAG implementation exists"
verify_file "internal/executor/topological.go" "Topological sort implementation exists"
verify_file "internal/executor/tasks_json.go" "Tasks JSON implementation exists"
verify_file "internal/executor/runner.go" "Task runner implementation exists"
verify_file "internal/executor/retry.go" "Retry mechanism implementation exists"
verify_file "internal/executor/executor.go" "Executor coordinator implementation exists"
echo

# Test 2: Verify test files exist
echo -e "${BLUE}Test 2: Verify Test Files${NC}"
verify_file "internal/executor/dag_test.go" "DAG tests exist"
verify_file "internal/executor/topological_test.go" "Topological sort tests exist"
verify_file "internal/executor/tasks_json_test.go" "Tasks JSON tests exist"
verify_file "internal/executor/runner_test.go" "Task runner tests exist"
verify_file "internal/executor/retry_test.go" "Retry tests exist"
verify_file "internal/executor/executor_test.go" "Executor tests exist"
echo

# Test 3: Verify test job structure
echo -e "${BLUE}Test 3: Verify Test Job Structure${NC}"
verify_dir "tests/dag_executor_test_job" "Test job directory exists"
verify_dir "tests/dag_executor_test_job/tasks" "Test tasks directory exists"
verify_file "tests/dag_executor_test_job/tasks/task1.md" "Test task 1 exists"
verify_file "tests/dag_executor_test_job/tasks/task2.md" "Test task 2 exists"
verify_file "tests/dag_executor_test_job/tasks/task3.md" "Test task 3 exists"
verify_file "tests/dag_executor_test_job/tasks/task4.md" "Test task 4 exists"
echo

# Test 4: Verify Go code compiles
echo -e "${BLUE}Test 4: Verify Go Code Compiles${NC}"
echo -n "Compiling Go code ... "
if go build -o /tmp/rick_test ./cmd/rick/main.go > /tmp/build_output.log 2>&1; then
    log_test "Go code compilation" "PASS"
else
    log_test "Go code compilation" "FAIL"
    echo "  Build errors:"
    head -10 /tmp/build_output.log | sed 's/^/    /'
fi
echo

# Test 5: Run DAG unit tests
echo -e "${BLUE}Test 5: Run DAG Unit Tests${NC}"
echo -n "Running DAG tests ... "
if go test -v ./internal/executor -run TestDAG > /tmp/dag_tests.log 2>&1; then
    # Count passed tests
    dag_count=$(grep -c "^ok.*TestDAG" /tmp/dag_tests.log || echo "0")
    log_test "DAG unit tests" "PASS"
    echo "  Passed DAG tests: $dag_count"
else
    log_test "DAG unit tests" "FAIL"
    tail -5 /tmp/dag_tests.log | sed 's/^/    /'
fi
echo

# Test 6: Run Topological Sort unit tests
echo -e "${BLUE}Test 6: Run Topological Sort Unit Tests${NC}"
echo -n "Running topological sort tests ... "
if go test -v ./internal/executor -run TestTopological > /tmp/topo_tests.log 2>&1; then
    log_test "Topological sort unit tests" "PASS"
else
    log_test "Topological sort unit tests" "FAIL"
    tail -5 /tmp/topo_tests.log | sed 's/^/    /'
fi
echo

# Test 7: Run Tasks JSON unit tests
echo -e "${BLUE}Test 7: Run Tasks JSON Unit Tests${NC}"
echo -n "Running tasks JSON tests ... "
if go test -v ./internal/executor -run TestTasksJSON > /tmp/json_tests.log 2>&1; then
    log_test "Tasks JSON unit tests" "PASS"
else
    log_test "Tasks JSON unit tests" "FAIL"
    tail -5 /tmp/json_tests.log | sed 's/^/    /'
fi
echo

# Test 8: Run Task Runner unit tests
echo -e "${BLUE}Test 8: Run Task Runner Unit Tests${NC}"
echo -n "Running task runner tests ... "
if go test -v ./internal/executor -run TestTaskRunner > /tmp/runner_tests.log 2>&1; then
    log_test "Task runner unit tests" "PASS"
else
    log_test "Task runner unit tests" "FAIL"
    tail -5 /tmp/runner_tests.log | sed 's/^/    /'
fi
echo

# Test 9: Run Retry unit tests
echo -e "${BLUE}Test 9: Run Retry Unit Tests${NC}"
echo -n "Running retry mechanism tests ... "
if go test -v ./internal/executor -run TestRetry > /tmp/retry_tests.log 2>&1; then
    log_test "Retry mechanism unit tests" "PASS"
else
    log_test "Retry mechanism unit tests" "FAIL"
    tail -5 /tmp/retry_tests.log | sed 's/^/    /'
fi
echo

# Test 10: Run Executor unit tests
echo -e "${BLUE}Test 10: Run Executor Unit Tests${NC}"
echo -n "Running executor tests ... "
if go test -v ./internal/executor -run TestExecutor > /tmp/executor_tests.log 2>&1; then
    log_test "Executor unit tests" "PASS"
else
    log_test "Executor unit tests" "FAIL"
    tail -5 /tmp/executor_tests.log | sed 's/^/    /'
fi
echo

# Test 11: Run all executor tests with coverage
echo -e "${BLUE}Test 11: Run All Executor Tests with Coverage${NC}"
echo -n "Running all tests with coverage ... "
if go test -v -coverprofile=/tmp/executor_coverage.out ./internal/executor > /tmp/all_tests.log 2>&1; then
    coverage=$(go tool cover -func=/tmp/executor_coverage.out | grep total | awk '{print $3}')
    log_test "All executor tests" "PASS"
    echo "  Coverage: $coverage"
else
    log_test "All executor tests" "FAIL"
    tail -5 /tmp/all_tests.log | sed 's/^/    /'
fi
echo

# Test 12: Verify test task files have correct format
echo -e "${BLUE}Test 12: Verify Test Task File Format${NC}"
for task_file in tests/dag_executor_test_job/tasks/task*.md; do
    if grep -q "^# 依赖关系" "$task_file" && \
       grep -q "^# 任务名称" "$task_file" && \
       grep -q "^# 任务目标" "$task_file" && \
       grep -q "^# 关键结果" "$task_file" && \
       grep -q "^# 测试方法" "$task_file"; then
        log_test "$(basename $task_file) format" "PASS"
    else
        log_test "$(basename $task_file) format" "FAIL"
    fi
done
echo

# Summary
echo "======================================"
echo -e "Results: ${GREEN}$PASSED passed${NC}, ${RED}$FAILED failed${NC}"
echo "======================================"
echo

# Detailed summary
if [ $FAILED -eq 0 ]; then
    echo -e "${GREEN}All integration tests passed!${NC}"
    exit 0
else
    echo -e "${RED}Some tests failed. Details:${NC}"
    for test in "${TESTS[@]}"; do
        echo "  $test"
    done
    exit 1
fi
