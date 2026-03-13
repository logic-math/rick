#!/bin/bash
#
# test_update.sh - Test script for update.sh
#

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test counters
TESTS_PASSED=0
TESTS_FAILED=0

# Script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

# Functions
print_test() {
    echo -e "${BLUE}[TEST]${NC} $1"
}

print_pass() {
    echo -e "${GREEN}[PASS]${NC} $1"
    ((TESTS_PASSED++))
}

print_fail() {
    echo -e "${RED}[FAIL]${NC} $1"
    ((TESTS_FAILED++))
}

print_info() {
    echo -e "${YELLOW}[INFO]${NC} $1"
}

# Test 1: Script exists and is executable
test_script_exists() {
    print_test "Script exists and is executable"

    if [[ ! -f "$SCRIPT_DIR/update.sh" ]]; then
        print_fail "update.sh script not found"
        return 1
    fi

    if [[ ! -x "$SCRIPT_DIR/update.sh" ]]; then
        print_fail "update.sh is not executable"
        return 1
    fi

    print_pass "update.sh exists and is executable"
    return 0
}

# Test 2: Help option works
test_help_option() {
    print_test "Help option works"

    if ! output=$("$SCRIPT_DIR/update.sh" --help 2>&1); then
        print_fail "Help option failed"
        return 1
    fi

    if ! echo "$output" | grep -q "Usage:"; then
        print_fail "Help output does not contain 'Usage:'"
        return 1
    fi

    print_pass "Help option works correctly"
    return 0
}

# Test 3: Script syntax is valid
test_script_syntax() {
    print_test "Script syntax is valid"

    if ! bash -n "$SCRIPT_DIR/update.sh"; then
        print_fail "Script syntax error"
        return 1
    fi

    print_pass "Script syntax is valid"
    return 0
}

# Test 4: Parameter parsing works
test_parameter_parsing() {
    print_test "Parameter parsing works"

    # Test --dev flag
    if ! output=$("$SCRIPT_DIR/update.sh" --dev --help 2>&1); then
        print_fail "Failed to parse --dev flag"
        return 1
    fi

    # Test --version parameter
    if ! output=$("$SCRIPT_DIR/update.sh" --version 1.0.0 --help 2>&1); then
        print_fail "Failed to parse --version parameter"
        return 1
    fi

    print_pass "Parameter parsing works correctly"
    return 0
}

# Test 5: Invalid parameters are rejected
test_invalid_parameters() {
    print_test "Invalid parameters are rejected"

    if output=$("$SCRIPT_DIR/update.sh" --invalid-option 2>&1); then
        print_fail "Invalid parameter was not rejected"
        return 1
    fi

    if ! echo "$output" | grep -q "Unknown option"; then
        print_fail "Error message for invalid parameter is incorrect"
        return 1
    fi

    print_pass "Invalid parameters are correctly rejected"
    return 0
}

# Test 6: Script contains required functions
test_required_functions() {
    print_test "Script contains required functions"

    local required_functions=(
        "parse_args"
        "determine_install_dir"
        "determine_command_name"
        "get_latest_version"
        "get_current_version"
        "confirm_update"
        "backup_installation"
        "restore_from_backup"
        "perform_update"
        "cleanup_backup"
    )

    for func in "${required_functions[@]}"; do
        if ! grep -q "^${func}()" "$SCRIPT_DIR/update.sh"; then
            print_fail "Function $func not found"
            return 1
        fi
    done

    print_pass "All required functions are present"
    return 0
}

# Test 7: Script handles dev mode correctly
test_dev_mode() {
    print_test "Dev mode handling"

    # Check that script contains dev mode logic
    if ! grep -q "IS_DEV_MODE=true" "$SCRIPT_DIR/update.sh"; then
        print_fail "Dev mode logic not found in script"
        return 1
    fi

    if ! grep -q 'rick_dev' "$SCRIPT_DIR/update.sh"; then
        print_fail "rick_dev command not found in script"
        return 1
    fi

    print_pass "Dev mode handling works correctly"
    return 0
}

# Test 8: Script handles install directory correctly
test_install_directory() {
    print_test "Install directory handling"

    # Check that script contains install directory logic
    if ! grep -q '~/.rick' "$SCRIPT_DIR/update.sh"; then
        print_fail "Production install path not found in script"
        return 1
    fi

    if ! grep -q '~/.rick_dev' "$SCRIPT_DIR/update.sh"; then
        print_fail "Dev install path not found in script"
        return 1
    fi

    if ! grep -q 'PREFIX' "$SCRIPT_DIR/update.sh"; then
        print_fail "Custom prefix handling not found in script"
        return 1
    fi

    print_pass "Install directory handling works correctly"
    return 0
}

# Test 9: Script error handling
test_error_handling() {
    print_test "Error handling"

    # Test with non-existent directory (should handle gracefully)
    if output=$("$SCRIPT_DIR/update.sh" --prefix /nonexistent/path 2>&1); then
        # This might fail during actual update, but parsing should work
        print_pass "Error handling works"
    else
        # Check if error message is meaningful
        if echo "$output" | grep -q "ERROR"; then
            print_pass "Error handling works"
        else
            print_fail "Error handling not working properly"
            return 1
        fi
    fi

    return 0
}

# Test 10: Script creates backup directory
test_backup_mechanism() {
    print_test "Backup mechanism"

    # Check that script contains backup logic
    if ! grep -q "backup_installation" "$SCRIPT_DIR/update.sh"; then
        print_fail "Backup mechanism not found in script"
        return 1
    fi

    if ! grep -q "restore_from_backup" "$SCRIPT_DIR/update.sh"; then
        print_fail "Restore mechanism not found in script"
        return 1
    fi

    if ! grep -q "BACKUP_DIR" "$SCRIPT_DIR/update.sh"; then
        print_fail "Backup directory variable not found in script"
        return 1
    fi

    print_pass "Backup mechanism works"
    return 0
}

# Main execution
main() {
    echo ""
    echo -e "${BLUE}========================================${NC}"
    echo -e "${GREEN}Running update.sh tests${NC}"
    echo -e "${BLUE}========================================${NC}"
    echo ""

    # Run all tests
    test_script_exists || true
    test_help_option || true
    test_script_syntax || true
    test_parameter_parsing || true
    test_invalid_parameters || true
    test_required_functions || true
    test_dev_mode || true
    test_install_directory || true
    test_error_handling || true
    test_backup_mechanism || true

    # Print summary
    echo ""
    echo -e "${BLUE}========================================${NC}"
    echo -e "${GREEN}Test Summary${NC}"
    echo -e "${BLUE}========================================${NC}"
    echo ""
    echo -e "Tests passed: ${GREEN}$TESTS_PASSED${NC}"
    echo -e "Tests failed: ${RED}$TESTS_FAILED${NC}"
    echo ""

    if [[ $TESTS_FAILED -eq 0 ]]; then
        echo -e "${GREEN}All tests passed!${NC}"
        echo -e "${BLUE}========================================${NC}"
        return 0
    else
        echo -e "${RED}Some tests failed!${NC}"
        echo -e "${BLUE}========================================${NC}"
        return 1
    fi
}

# Run main function
main "$@"
