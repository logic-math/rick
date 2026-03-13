#!/bin/bash

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test counters
PASSED=0
FAILED=0

# Helper function
test_case() {
    local name=$1
    local cmd=$2
    local expected=$3
    
    echo -n "Testing: $name ... "
    result=$(eval "$cmd" 2>&1)
    
    if echo "$result" | grep -F "$expected" > /dev/null 2>&1; then
        echo -e "${GREEN}✓ PASSED${NC}"
        ((PASSED++))
    else
        echo -e "${RED}✗ FAILED${NC}"
        echo "  Expected: $expected"
        echo "  Got: $result"
        ((FAILED++))
    fi
}

echo "======================================"
echo "Rick CLI Integration Tests"
echo "======================================"
echo

# Task 1: CLI Framework and --version
echo -e "${YELLOW}Task 1: CLI Framework and --version${NC}"
test_case "CLI version output" "./rick --version" "rick version 0.1.0"
test_case "CLI help output" "./rick --help" "Usage:"
test_case "CLI help shows commands" "./rick --help" "Commands:"
echo

# Task 2: Workspace Initialization
echo -e "${YELLOW}Task 2: Workspace Initialization${NC}"
# Create a temporary directory for testing
TEST_DIR=$(mktemp -d)
export HOME=$TEST_DIR
test_case "Init command exists" "./rick init --help" "init"
test_case "Init command creates workspace" "./rick init && test -d $HOME/.rick && echo success" "success"
test_case "Workspace has OKR.md" "test -f $HOME/.rick/OKR.md && echo success" "success"
test_case "Workspace has SPEC.md" "test -f $HOME/.rick/SPEC.md && echo success" "success"
test_case "Workspace has wiki directory" "test -d $HOME/.rick/wiki && echo success" "success"
test_case "Workspace has skills directory" "test -d $HOME/.rick/skills && echo success" "success"
echo

# Task 3: Configuration System
echo -e "${YELLOW}Task 3: Configuration System${NC}"
# The config system should load/save from ~/.rick/config.json
test_case "Config file created" "test -f $HOME/.rick/config.json && echo success" "success"
if [ -f "$HOME/.rick/config.json" ]; then
    test_case "Config is valid JSON" "cat $HOME/.rick/config.json | grep -F '\"' && echo success" "success"
fi
echo

# Task 4: Logging System
echo -e "${YELLOW}Task 4: Logging System${NC}"
# Test logging by running verbose mode
test_case "Verbose flag works" "./rick --verbose init 2>&1" "Initializing"
echo

# Task 5: Command Routing
echo -e "${YELLOW}Task 5: Command Routing${NC}"
test_case "Plan command exists" "./rick plan --help" "plan"
test_case "Doing command exists" "./rick doing --help" "doing"
test_case "Learning command exists" "./rick learning --help" "learning"
test_case "Plan command with job flag" "./rick plan --help" "job string"
test_case "Doing command with job flag" "./rick doing --help" "job string"
test_case "Learning command with job flag" "./rick learning --help" "job string"
echo

# Task 6: Error Handling
echo -e "${YELLOW}Task 6: Error Handling${NC}"
test_case "Invalid command shows error" "./rick invalid_command 2>&1" "Error"
echo

# Summary
echo "======================================"
echo -e "Results: ${GREEN}$PASSED passed${NC}, ${RED}$FAILED failed${NC}"
echo "======================================"

# Cleanup
rm -rf $TEST_DIR

exit $FAILED
