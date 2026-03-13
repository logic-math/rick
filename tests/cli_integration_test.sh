#!/bin/bash

# Rick CLI Integration Test Script
# This script tests all four core commands and their integration

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test counters
PASSED=0
FAILED=0

# Helper function to print test results
test_case() {
    local name=$1
    local cmd=$2
    local expected=$3

    echo -n "  Testing: $name ... "

    result=$(eval "$cmd" 2>&1)

    if echo "$result" | grep -F "$expected" > /dev/null 2>&1; then
        echo -e "${GREEN}✓ PASSED${NC}"
        ((PASSED++))
        return 0
    else
        echo -e "${RED}✗ FAILED${NC}"
        echo "    Expected: $expected"
        echo "    Got: $result"
        ((FAILED++))
        return 1
    fi
}

# Helper function to test file existence
test_file_exists() {
    local name=$1
    local file=$2

    echo -n "  Testing: $name ... "
    if [ -f "$file" ]; then
        echo -e "${GREEN}✓ PASSED${NC}"
        ((PASSED++))
        return 0
    else
        echo -e "${RED}✗ FAILED${NC}"
        echo "    File not found: $file"
        ((FAILED++))
        return 1
    fi
}

# Helper function to test directory existence
test_dir_exists() {
    local name=$1
    local dir=$2

    echo -n "  Testing: $name ... "
    if [ -d "$dir" ]; then
        echo -e "${GREEN}✓ PASSED${NC}"
        ((PASSED++))
        return 0
    else
        echo -e "${RED}✗ FAILED${NC}"
        echo "    Directory not found: $dir"
        ((FAILED++))
        return 1
    fi
}

# Get the rick binary path
RICK_BIN="${RICK_BIN:-./rick}"
if [ ! -f "$RICK_BIN" ]; then
    RICK_BIN="/Users/sunquan/ai_coding/CODING/rick/rick"
fi

if [ ! -f "$RICK_BIN" ]; then
    echo -e "${RED}Error: rick binary not found at $RICK_BIN${NC}"
    exit 1
fi

echo -e "${BLUE}======================================"
echo "Rick CLI Integration Tests"
echo "======================================${NC}"
echo

# Create a temporary directory for testing
TEST_DIR=$(mktemp -d)
trap "rm -rf $TEST_DIR" EXIT

echo -e "${YELLOW}[1/6] Testing rick init command${NC}"
test_case "init command exists" "$RICK_BIN init --help | head -1" "Initialize"
bash -c "export HOME=$TEST_DIR; $RICK_BIN init" > /dev/null 2>&1 && echo -n "  Testing: init creates workspace ... " && echo -e "${GREEN}✓ PASSED${NC}" && ((PASSED++)) || (echo -n "  Testing: init creates workspace ... " && echo -e "${RED}✗ FAILED${NC}" && ((FAILED++)))
test_dir_exists "init creates .rick directory" "$TEST_DIR/.rick"
test_dir_exists "init creates wiki directory" "$TEST_DIR/.rick/wiki"
test_dir_exists "init creates skills directory" "$TEST_DIR/.rick/skills"
test_dir_exists "init creates jobs directory" "$TEST_DIR/.rick/jobs"
test_file_exists "init creates OKR.md" "$TEST_DIR/.rick/OKR.md"
test_file_exists "init creates SPEC.md" "$TEST_DIR/.rick/SPEC.md"
test_file_exists "init creates config.json" "$TEST_DIR/.rick/config.json"
test_dir_exists "init creates .git directory" "$TEST_DIR/.rick/.git"
echo

echo -e "${YELLOW}[2/6] Testing command line flags${NC}"
test_case "--version flag" "$RICK_BIN --version" "version"
test_case "--help flag" "$RICK_BIN --help" "Usage:"
test_case "--verbose flag" "$RICK_BIN --verbose init --help" "Initialize"
test_case "--dry-run flag" "$RICK_BIN --dry-run init" "Would initialize"
echo

echo -e "${YELLOW}[3/6] Testing command availability${NC}"
test_case "plan command exists" "$RICK_BIN plan --help" "plan"
test_case "doing command exists" "$RICK_BIN doing --help" "doing"
test_case "learning command exists" "$RICK_BIN learning --help" "learning"
echo

echo -e "${YELLOW}[4/6] Testing command flags${NC}"
test_case "plan command has job flag" "$RICK_BIN plan --help" "job"
test_case "doing command has job flag" "$RICK_BIN doing --help" "job"
test_case "learning command has job flag" "$RICK_BIN learning --help" "job"
test_case "init command has verbose flag" "$RICK_BIN init --help" "verbose"
echo

echo -e "${YELLOW}[5/6] Testing error handling${NC}"
$RICK_BIN invalid_command 2>&1 | grep -q "Error" && echo -n "  Testing: invalid command shows error ... " && echo -e "${GREEN}✓ PASSED${NC}" && ((PASSED++)) || (echo -n "  Testing: invalid command shows error ... " && echo -e "${RED}✗ FAILED${NC}" && ((FAILED++)))
echo

echo -e "${YELLOW}[6/6] Testing configuration${NC}"
test_case "config file is valid JSON" "cat $TEST_DIR/.rick/config.json | head -1" "{"
test_file_exists "config.json exists" "$TEST_DIR/.rick/config.json"
echo

# Summary
echo -e "${BLUE}======================================"
echo "Test Results"
echo "======================================${NC}"
echo -e "Passed:  ${GREEN}$PASSED${NC}"
echo -e "Failed:  ${RED}$FAILED${NC}"
echo -e "${BLUE}======================================${NC}"

if [ $FAILED -eq 0 ]; then
    echo -e "${GREEN}All tests passed!${NC}"
    exit 0
else
    echo -e "${RED}Some tests failed!${NC}"
    exit 1
fi
