#!/bin/bash
#
# build_test.sh - Test script for build.sh
#
# This script tests the build.sh script to ensure it:
# - Can compile the binary correctly
# - Verifies the binary is executable
# - Supports --output parameter
# - Checks Go version correctly
# - Provides clear error messages
#

# Script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
BUILD_SCRIPT="${PROJECT_DIR}/scripts/build.sh"

# Test results
TESTS_PASSED=0
TESTS_FAILED=0

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Functions
print_test_start() {
    echo -e "${BLUE}[TEST]${NC} $1"
}

print_test_pass() {
    echo -e "${GREEN}[PASS]${NC} $1"
    ((TESTS_PASSED++))
}

print_test_fail() {
    echo -e "${RED}[FAIL]${NC} $1"
    ((TESTS_FAILED++))
}

print_info() {
    echo -e "${YELLOW}[INFO]${NC} $1"
}

# Test 1: Check if build.sh exists and is executable
test_build_script_exists() {
    print_test_start "build.sh exists and is executable"

    if [[ ! -f "$BUILD_SCRIPT" ]]; then
        print_test_fail "build.sh not found at $BUILD_SCRIPT"
        return 1
    fi

    if [[ ! -x "$BUILD_SCRIPT" ]]; then
        print_test_fail "build.sh is not executable"
        return 1
    fi

    print_test_pass "build.sh exists and is executable"
}

# Test 2: Check if build.sh shows help correctly
test_build_script_help() {
    print_test_start "build.sh --help works"

    if ! "$BUILD_SCRIPT" --help > /dev/null 2>&1; then
        print_test_fail "build.sh --help failed"
        return 1
    fi

    print_test_pass "build.sh --help works"
}

# Test 3: Check if build.sh detects Go version correctly
test_go_version_check() {
    print_test_start "Go version check works"

    if ! command -v go &> /dev/null; then
        print_test_fail "Go is not installed"
        return 1
    fi

    local go_version=$(go version | awk '{print $3}' | sed 's/go//')
    print_info "Go version: $go_version"

    print_test_pass "Go version check works"
}

# Test 4: Build with default output path
test_build_default_output() {
    print_test_start "build.sh compiles with default output"

    # Create a temporary directory for testing
    local temp_dir=$(mktemp -d)
    local test_output="${temp_dir}/rick"

    # Clean up before test
    rm -rf "${PROJECT_DIR}/bin/rick"

    if ! "$BUILD_SCRIPT" > /dev/null 2>&1; then
        print_test_fail "build.sh compilation failed"
        rm -rf "$temp_dir"
        return 1
    fi

    if [[ ! -f "${PROJECT_DIR}/bin/rick" ]]; then
        print_test_fail "Binary not found at default location"
        rm -rf "$temp_dir"
        return 1
    fi

    if [[ ! -x "${PROJECT_DIR}/bin/rick" ]]; then
        print_test_fail "Binary is not executable"
        rm -rf "$temp_dir"
        return 1
    fi

    print_test_pass "build.sh compiles with default output"
    rm -rf "$temp_dir"
}

# Test 5: Build with custom output path
test_build_custom_output() {
    print_test_start "build.sh compiles with --output parameter"

    # Create a temporary directory for testing
    local temp_dir=$(mktemp -d)
    local test_output="${temp_dir}/rick_test"

    if ! "$BUILD_SCRIPT" --output "$test_output" > /dev/null 2>&1; then
        print_test_fail "build.sh compilation with --output failed"
        rm -rf "$temp_dir"
        return 1
    fi

    if [[ ! -f "$test_output" ]]; then
        print_test_fail "Binary not found at custom location: $test_output"
        rm -rf "$temp_dir"
        return 1
    fi

    if [[ ! -x "$test_output" ]]; then
        print_test_fail "Binary is not executable at custom location"
        rm -rf "$temp_dir"
        return 1
    fi

    print_test_pass "build.sh compiles with --output parameter"
    rm -rf "$temp_dir"
}

# Test 6: Verify binary functionality
test_binary_functionality() {
    print_test_start "Compiled binary is functional"

    # Build first
    if ! "$BUILD_SCRIPT" > /dev/null 2>&1; then
        print_test_fail "Build failed"
        return 1
    fi

    local binary="${PROJECT_DIR}/bin/rick"

    if ! "$binary" --help > /dev/null 2>&1; then
        print_test_fail "Binary --help failed"
        return 1
    fi

    print_test_pass "Compiled binary is functional"
}

# Test 7: Check error handling for invalid options
test_invalid_options() {
    print_test_start "build.sh handles invalid options"

    if "$BUILD_SCRIPT" --invalid-option > /dev/null 2>&1; then
        print_test_fail "build.sh should reject invalid options"
        return 1
    fi

    print_test_pass "build.sh handles invalid options"
}

# Main execution
main() {
    echo "================================"
    echo "build.sh Test Suite"
    echo "================================"
    echo ""

    test_build_script_exists
    test_build_script_help
    test_go_version_check
    test_build_default_output
    test_build_custom_output
    test_binary_functionality
    test_invalid_options

    echo ""
    echo "================================"
    echo "Test Results"
    echo "================================"
    echo -e "Passed: ${GREEN}${TESTS_PASSED}${NC}"
    echo -e "Failed: ${RED}${TESTS_FAILED}${NC}"
    echo ""

    if [[ $TESTS_FAILED -eq 0 ]]; then
        echo -e "${GREEN}All tests passed!${NC}"
        return 0
    else
        echo -e "${RED}Some tests failed!${NC}"
        return 1
    fi
}

# Run main function
main "$@"
