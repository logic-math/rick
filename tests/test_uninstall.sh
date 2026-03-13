#!/bin/bash
#
# test_uninstall.sh - Test script for uninstall.sh functionality
#
# This script tests all aspects of the uninstall.sh script:
# - Parameter parsing (--dev, --all, --prefix)
# - Script syntax and structure
# - Error handling
#

set -e

# Script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
SCRIPTS_DIR="$PROJECT_DIR/scripts"

# Test configuration
TEST_TEMP_DIR=$(mktemp -d)
TEST_LOG_FILE="$TEST_TEMP_DIR/test.log"
TEST_PASSED=0
TEST_FAILED=0

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Functions
print_test_header() {
    echo ""
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}Test: $1${NC}"
    echo -e "${BLUE}========================================${NC}"
}

print_test_pass() {
    echo -e "${GREEN}[PASS]${NC} $1"
    ((TEST_PASSED++))
}

print_test_fail() {
    echo -e "${RED}[FAIL]${NC} $1"
    ((TEST_FAILED++))
}

print_info() {
    echo -e "${YELLOW}[INFO]${NC} $1"
}

print_debug() {
    echo -e "${BLUE}[DEBUG]${NC} $1"
}

cleanup_test() {
    rm -rf "$TEST_TEMP_DIR"
}

trap cleanup_test EXIT

# Test 1: Help message
test_help_message() {
    print_test_header "Help Message"

    local output=$("$SCRIPTS_DIR/uninstall.sh" -h 2>&1 || true)

    if echo "$output" | grep -q "Usage:"; then
        print_test_pass "Help message displayed correctly"
    else
        print_test_fail "Help message not displayed"
    fi

    if echo "$output" | grep -q "\-\-dev"; then
        print_test_pass "Help includes --dev option"
    else
        print_test_fail "Help missing --dev option"
    fi

    if echo "$output" | grep -q "\-\-all"; then
        print_test_pass "Help includes --all option"
    else
        print_test_fail "Help missing --all option"
    fi

    if echo "$output" | grep -q "\-\-prefix"; then
        print_test_pass "Help includes --prefix option"
    else
        print_test_fail "Help missing --prefix option"
    fi
}

# Test 2: Invalid parameter handling
test_invalid_parameters() {
    print_test_header "Invalid Parameter Handling"

    local output=$("$SCRIPTS_DIR/uninstall.sh" --invalid 2>&1 || true)

    if echo "$output" | grep -q "Unknown option"; then
        print_test_pass "Invalid parameter error detected"
    else
        print_test_fail "Invalid parameter not handled correctly"
    fi
}

# Test 3: Script syntax check
test_script_syntax() {
    print_test_header "Script Syntax Check"

    if bash -n "$SCRIPTS_DIR/uninstall.sh" 2>/dev/null; then
        print_test_pass "Script syntax is valid"
    else
        print_test_fail "Script has syntax errors"
    fi
}

# Test 4: Verify script structure
test_script_structure() {
    print_test_header "Script Structure"

    local script_content=$(cat "$SCRIPTS_DIR/uninstall.sh")

    # Check for required functions
    if echo "$script_content" | grep -q "parse_args()"; then
        print_test_pass "parse_args function exists"
    else
        print_test_fail "parse_args function missing"
    fi

    if echo "$script_content" | grep -q "confirm_uninstall()"; then
        print_test_pass "confirm_uninstall function exists"
    else
        print_test_fail "confirm_uninstall function missing"
    fi

    if echo "$script_content" | grep -q "delete_symlink()"; then
        print_test_pass "delete_symlink function exists"
    else
        print_test_fail "delete_symlink function missing"
    fi

    if echo "$script_content" | grep -q "uninstall_version()"; then
        print_test_pass "uninstall_version function exists"
    else
        print_test_fail "uninstall_version function missing"
    fi

    if echo "$script_content" | grep -q "main()"; then
        print_test_pass "main function exists"
    else
        print_test_fail "main function missing"
    fi
}

# Test 5: Verify script is executable
test_script_executable() {
    print_test_header "Script Executable"

    if [[ -x "$SCRIPTS_DIR/uninstall.sh" ]]; then
        print_test_pass "uninstall.sh is executable"
    else
        print_test_fail "uninstall.sh is not executable"
    fi
}

# Test 6: Verify parameter parsing logic
test_parameter_parsing() {
    print_test_header "Parameter Parsing Logic"

    local script_content=$(cat "$SCRIPTS_DIR/uninstall.sh")

    # Check for --dev flag handling
    if echo "$script_content" | grep -q "UNINSTALL_DEV=true"; then
        print_test_pass "--dev flag parsing implemented"
    else
        print_test_fail "--dev flag parsing missing"
    fi

    # Check for --all flag handling
    if echo "$script_content" | grep -q "UNINSTALL_ALL=true"; then
        print_test_pass "--all flag parsing implemented"
    else
        print_test_fail "--all flag parsing missing"
    fi

    # Check for --prefix handling
    if echo "$script_content" | grep -q 'PREFIX="\$2"'; then
        print_test_pass "--prefix parameter parsing implemented"
    else
        print_test_fail "--prefix parameter parsing missing"
    fi
}

# Test 7: Verify uninstall logic
test_uninstall_logic() {
    print_test_header "Uninstall Logic"

    local script_content=$(cat "$SCRIPTS_DIR/uninstall.sh")

    # Check for directory deletion
    if echo "$script_content" | grep -q "rm -rf"; then
        print_test_pass "Directory deletion logic present"
    else
        print_test_fail "Directory deletion logic missing"
    fi

    # Check for symlink removal
    if echo "$script_content" | grep -q "rm -f.*symlink"; then
        print_test_pass "Symlink removal logic present"
    else
        print_test_fail "Symlink removal logic missing"
    fi

    # Check for HOME directory usage
    if echo "$script_content" | grep -q '\$HOME'; then
        print_test_pass "HOME directory reference present"
    else
        print_test_fail "HOME directory reference missing"
    fi
}

# Test 8: Verify error handling
test_error_handling() {
    print_test_header "Error Handling"

    local script_content=$(cat "$SCRIPTS_DIR/uninstall.sh")

    # Check for error messages
    if echo "$script_content" | grep -q "print_error"; then
        print_test_pass "Error printing function used"
    else
        print_test_fail "Error printing function not used"
    fi

    # Check for success messages
    if echo "$script_content" | grep -q "print_success"; then
        print_test_pass "Success printing function used"
    else
        print_test_fail "Success printing function not used"
    fi

    # Check for info messages
    if echo "$script_content" | grep -q "print_info"; then
        print_test_pass "Info printing function used"
    else
        print_test_fail "Info printing function not used"
    fi
}

# Test 9: Verify colors and formatting
test_colors_and_formatting() {
    print_test_header "Colors and Formatting"

    local script_content=$(cat "$SCRIPTS_DIR/uninstall.sh")

    # Check for color definitions
    if echo "$script_content" | grep -q "RED="; then
        print_test_pass "Color definitions present"
    else
        print_test_fail "Color definitions missing"
    fi

    # Check for summary output
    if echo "$script_content" | grep -q "Uninstallation Complete"; then
        print_test_pass "Completion summary message present"
    else
        print_test_fail "Completion summary message missing"
    fi
}

# Test 10: Verify documentation
test_documentation() {
    print_test_header "Documentation"

    local script_content=$(cat "$SCRIPTS_DIR/uninstall.sh")

    # Check for header comment
    if echo "$script_content" | grep -q "uninstall.sh - Uninstallation script"; then
        print_test_pass "Script header documentation present"
    else
        print_test_fail "Script header documentation missing"
    fi

    # Check for usage examples
    if echo "$script_content" | grep -q "Examples:"; then
        print_test_pass "Usage examples present"
    else
        print_test_fail "Usage examples missing"
    fi
}

# Test 11: Verify production vs dev version handling
test_version_handling() {
    print_test_header "Version Handling"

    local script_content=$(cat "$SCRIPTS_DIR/uninstall.sh")

    # Check for production version directory
    if echo "$script_content" | grep -q '\.rick'; then
        print_test_pass "Production version directory (~/.rick) referenced"
    else
        print_test_fail "Production version directory not referenced"
    fi

    # Check for development version directory
    if echo "$script_content" | grep -q '\.rick_dev'; then
        print_test_pass "Development version directory (~/.rick_dev) referenced"
    else
        print_test_fail "Development version directory not referenced"
    fi

    # Check for command name handling
    if echo "$script_content" | grep -q "rick_dev"; then
        print_test_pass "Development command name (rick_dev) referenced"
    else
        print_test_fail "Development command name not referenced"
    fi
}

# Main test execution
main() {
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}Rick CLI Uninstall Script Test Suite${NC}"
    echo -e "${BLUE}========================================${NC}"

    test_script_executable
    test_script_syntax
    test_help_message
    test_invalid_parameters
    test_script_structure
    test_parameter_parsing
    test_uninstall_logic
    test_error_handling
    test_colors_and_formatting
    test_documentation
    test_version_handling

    # Print summary
    echo ""
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}Test Summary${NC}"
    echo -e "${BLUE}========================================${NC}"
    echo -e "${GREEN}Passed: $TEST_PASSED${NC}"
    echo -e "${RED}Failed: $TEST_FAILED${NC}"
    echo -e "${BLUE}========================================${NC}"

    if [[ $TEST_FAILED -eq 0 ]]; then
        echo -e "${GREEN}All tests passed!${NC}"
        return 0
    else
        echo -e "${RED}Some tests failed.${NC}"
        return 1
    fi
}

main "$@"
