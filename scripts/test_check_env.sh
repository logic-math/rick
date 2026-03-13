#!/bin/bash
#
# test_check_env.sh - Test script for check_env.sh
#
# This script tests the environment check functionality
#

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Test counters
TESTS_PASSED=0
TESTS_FAILED=0

# Test functions
print_test_header() {
    echo ""
    echo "========================================"
    echo "Test: $1"
    echo "========================================"
}

assert_exit_code() {
    local expected=$1
    local actual=$2
    local test_name=$3

    if [ "$actual" -eq "$expected" ]; then
        echo -e "${GREEN}✓ PASS${NC}: $test_name (exit code: $actual)"
        ((TESTS_PASSED++))
        return 0
    else
        echo -e "${RED}✗ FAIL${NC}: $test_name (expected: $expected, got: $actual)"
        ((TESTS_FAILED++))
        return 1
    fi
}

assert_output_contains() {
    local output=$1
    local pattern=$2
    local test_name=$3

    if echo "$output" | grep -q "$pattern"; then
        echo -e "${GREEN}✓ PASS${NC}: $test_name"
        ((TESTS_PASSED++))
        return 0
    else
        echo -e "${RED}✗ FAIL${NC}: $test_name (pattern not found: $pattern)"
        ((TESTS_FAILED++))
        return 1
    fi
}

assert_json_valid() {
    local json=$1
    local test_name=$2

    if echo "$json" | python3 -m json.tool > /dev/null 2>&1; then
        echo -e "${GREEN}✓ PASS${NC}: $test_name (valid JSON)"
        ((TESTS_PASSED++))
        return 0
    else
        echo -e "${RED}✗ FAIL${NC}: $test_name (invalid JSON)"
        ((TESTS_FAILED++))
        return 1
    fi
}

# Test 1: Basic execution
print_test_header "Basic execution"
output=$("$PROJECT_DIR/scripts/check_env.sh" 2>&1)
exit_code=$?
assert_exit_code 1 "$exit_code" "Should exit with code 1 (Claude Code not installed)"

# Test 2: Check for Go version output
print_test_header "Go version check"
assert_output_contains "$output" "Go version" "Should output Go version"

# Test 3: Check for Git output
print_test_header "Git check"
assert_output_contains "$output" "Git version" "Should output Git version"

# Test 4: Check for report header
print_test_header "Report output"
assert_output_contains "$output" "Environment Check Report" "Should output report header"

# Test 5: JSON output format
print_test_header "JSON output format"
json_output=$("$PROJECT_DIR/scripts/check_env.sh" --json 2>&1)
json_exit=$?
assert_json_valid "$json_output" "JSON output should be valid"

# Test 6: JSON contains required fields
print_test_header "JSON structure"
assert_output_contains "$json_output" '"status"' "JSON should contain status field"
assert_output_contains "$json_output" '"checks"' "JSON should contain checks field"
assert_output_contains "$json_output" '"go_version"' "JSON should contain go_version check"
assert_output_contains "$json_output" '"claude_code"' "JSON should contain claude_code check"
assert_output_contains "$json_output" '"git"' "JSON should contain git check"
assert_output_contains "$json_output" '"path"' "JSON should contain path check"

# Test 7: Verbose output
print_test_header "Verbose output"
verbose_output=$("$PROJECT_DIR/scripts/check_env.sh" --verbose 2>&1)
verbose_exit=$?
assert_output_contains "$verbose_output" "Installation:" "Verbose output should show installation paths"

# Test 8: Help output
print_test_header "Help output"
help_output=$("$PROJECT_DIR/scripts/check_env.sh" --help 2>&1)
help_exit=$?
assert_exit_code 0 "$help_exit" "Help should exit with code 0"
assert_output_contains "$help_output" "Usage:" "Help should show usage"

# Test 9: Invalid option handling
print_test_header "Invalid option handling"
invalid_output=$("$PROJECT_DIR/scripts/check_env.sh" --invalid 2>&1)
invalid_exit=$?
assert_exit_code 1 "$invalid_exit" "Invalid option should exit with code 1"
assert_output_contains "$invalid_output" "Unknown option" "Should report unknown option"

# Test 10: Script is executable
print_test_header "Script permissions"
if [ -x "$PROJECT_DIR/scripts/check_env.sh" ]; then
    echo -e "${GREEN}✓ PASS${NC}: check_env.sh is executable"
    ((TESTS_PASSED++))
else
    echo -e "${RED}✗ FAIL${NC}: check_env.sh is not executable"
    ((TESTS_FAILED++))
fi

# Print summary
echo ""
echo "========================================"
echo "Test Summary"
echo "========================================"
echo -e "Passed: ${GREEN}$TESTS_PASSED${NC}"
echo -e "Failed: ${RED}$TESTS_FAILED${NC}"
echo "Total:  $((TESTS_PASSED + TESTS_FAILED))"
echo ""

if [ $TESTS_FAILED -eq 0 ]; then
    echo -e "${GREEN}All tests passed!${NC}"
    exit 0
else
    echo -e "${RED}Some tests failed!${NC}"
    exit 1
fi
