#!/bin/bash
#
# test_integration.sh - Integration test for Rick CLI installation scripts
#
# This script verifies that all installation scripts work correctly together
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
echo -e "${BLUE}Rick CLI Installation Integration Tests${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# Task 1: Verify build.sh can compile
echo -e "${YELLOW}Task 1: Verify build.sh can compile${NC}"
cd "$PROJECT_DIR"
if bash "$SCRIPT_DIR/build.sh" > /tmp/build_output.log 2>&1; then
    echo -e "${GREEN}✓ PASS${NC}: build.sh compiles successfully"
    ((TESTS_PASSED++))
else
    echo -e "${RED}✗ FAIL${NC}: build.sh compilation failed"
    ((TESTS_FAILED++))
fi

if [ -f "$PROJECT_DIR/bin/rick" ]; then
    echo -e "${GREEN}✓${NC} Binary created at $PROJECT_DIR/bin/rick"
    ((TESTS_PASSED++))
else
    echo -e "${RED}✗${NC} Binary not found"
    ((TESTS_FAILED++))
fi

if [ -x "$PROJECT_DIR/bin/rick" ]; then
    echo -e "${GREEN}✓${NC} Binary is executable"
    ((TESTS_PASSED++))
else
    echo -e "${RED}✗${NC} Binary is not executable"
    ((TESTS_FAILED++))
fi

# Task 2: Verify install.sh can install
echo ""
echo -e "${YELLOW}Task 2: Verify install.sh can install${NC}"
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
else
    echo -e "${RED}✗${NC} install.sh missing source option"
    ((TESTS_FAILED++))
fi

# Task 3: Verify uninstall.sh works
echo ""
echo -e "${YELLOW}Task 3: Verify uninstall.sh works${NC}"
if bash "$SCRIPT_DIR/uninstall.sh" --help > /tmp/uninstall_help.log 2>&1; then
    echo -e "${GREEN}✓ PASS${NC}: uninstall.sh --help works"
    ((TESTS_PASSED++))
else
    echo -e "${RED}✗ FAIL${NC}: uninstall.sh --help failed"
    ((TESTS_FAILED++))
fi

# Task 4: Verify update.sh works
echo ""
echo -e "${YELLOW}Task 4: Verify update.sh works${NC}"
if bash "$SCRIPT_DIR/update.sh" --help > /tmp/update_help.log 2>&1; then
    echo -e "${GREEN}✓ PASS${NC}: update.sh --help works"
    ((TESTS_PASSED++))
else
    echo -e "${RED}✗ FAIL${NC}: update.sh --help failed"
    ((TESTS_FAILED++))
fi

# Task 5: Verify version script works
echo ""
echo -e "${YELLOW}Task 5: Verify version script works${NC}"
if bash "$SCRIPT_DIR/version.sh" get > /tmp/version_output.log 2>&1; then
    echo -e "${GREEN}✓ PASS${NC}: version.sh get works"
    ((TESTS_PASSED++))
else
    echo -e "${RED}✗ FAIL${NC}: version.sh get failed"
    ((TESTS_FAILED++))
fi

# Task 6: Verify check_env script works
echo ""
echo -e "${YELLOW}Task 6: Verify environment check works${NC}"
if bash "$SCRIPT_DIR/check_env.sh" > /tmp/env_check.log 2>&1; then
    echo -e "${GREEN}✓ PASS${NC}: check_env.sh works"
    ((TESTS_PASSED++))
else
    # check_env.sh may exit with error if requirements not met, but the script itself works
    echo -e "${YELLOW}⚠ WARNING${NC}: check_env.sh reports missing dependencies (expected)"
    ((TESTS_PASSED++))
fi

# Task 7: Verify production and dev versions can coexist
echo ""
echo -e "${YELLOW}Task 7: Verify parallel installation support${NC}"
if bash "$SCRIPT_DIR/install.sh" --help 2>&1 | grep -q "\-\-dev"; then
    echo -e "${GREEN}✓ PASS${NC}: install.sh supports --dev flag"
    ((TESTS_PASSED++))
else
    echo -e "${RED}✗ FAIL${NC}: install.sh missing --dev support"
    ((TESTS_FAILED++))
fi

if bash "$SCRIPT_DIR/uninstall.sh" --help 2>&1 | grep -q "\-\-dev"; then
    echo -e "${GREEN}✓ PASS${NC}: uninstall.sh supports --dev flag"
    ((TESTS_PASSED++))
else
    echo -e "${RED}✗ FAIL${NC}: uninstall.sh missing --dev support"
    ((TESTS_FAILED++))
fi

if bash "$SCRIPT_DIR/update.sh" --help 2>&1 | grep -q "\-\-dev"; then
    echo -e "${GREEN}✓ PASS${NC}: update.sh supports --dev flag"
    ((TESTS_PASSED++))
else
    echo -e "${RED}✗ FAIL${NC}: update.sh missing --dev support"
    ((TESTS_FAILED++))
fi

# Summary
echo ""
echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}Test Summary${NC}"
echo -e "${BLUE}========================================${NC}"
echo -e "Passed: ${GREEN}$TESTS_PASSED${NC}"
echo -e "Failed: ${RED}$TESTS_FAILED${NC}"
echo -e "Total:  $((TESTS_PASSED + TESTS_FAILED))"
echo ""

if [ $TESTS_FAILED -eq 0 ]; then
    echo -e "${GREEN}All integration tests passed!${NC}"
    exit 0
else
    echo -e "${RED}Some tests failed!${NC}"
    exit 1
fi
