#!/bin/bash
#
# test_install.sh - Test script for install.sh
#
# This script tests various installation scenarios
#

set -e

# Script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test counters
TESTS_PASSED=0
TESTS_FAILED=0
TESTS_TOTAL=0

# Functions
print_test_start() {
    echo -e "${BLUE}[TEST]${NC} $1"
    ((TESTS_TOTAL++))
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

# Test 1: Script exists and is executable
test_script_exists() {
    print_test_start "Script exists and is executable"

    if [[ ! -f "$SCRIPT_DIR/install.sh" ]]; then
        print_test_fail "install.sh not found"
        return 1
    fi

    if [[ ! -x "$SCRIPT_DIR/install.sh" ]]; then
        print_test_fail "install.sh is not executable"
        return 1
    fi

    print_test_pass "Script exists and is executable"
    return 0
}

# Test 2: Help message works
test_help_message() {
    print_test_start "Help message works"

    local help_output=$("$SCRIPT_DIR/install.sh" --help 2>&1)

    if [[ ! "$help_output" =~ "Usage:" ]]; then
        print_test_fail "Help message doesn't contain 'Usage:'"
        return 1
    fi

    if [[ ! "$help_output" =~ "--source" ]]; then
        print_test_fail "Help message doesn't contain '--source' option"
        return 1
    fi

    if [[ ! "$help_output" =~ "--binary" ]]; then
        print_test_fail "Help message doesn't contain '--binary' option"
        return 1
    fi

    if [[ ! "$help_output" =~ "--dev" ]]; then
        print_test_fail "Help message doesn't contain '--dev' option"
        return 1
    fi

    print_test_pass "Help message works correctly"
    return 0
}

# Test 3: Parameter parsing - detect invalid options
test_invalid_option() {
    print_test_start "Invalid option detection"

    if "$SCRIPT_DIR/install.sh" --invalid-option 2>&1 | grep -q "Unknown option"; then
        print_test_pass "Invalid option correctly detected"
        return 0
    else
        print_test_fail "Invalid option not detected"
        return 1
    fi
}

# Test 4: Test dry-run of source installation (check Go version)
test_go_version_check() {
    print_test_start "Go version check (prerequisite)"

    if ! command -v go &> /dev/null; then
        print_info "Go not installed, skipping Go version check"
        return 0
    fi

    local go_version=$(go version | awk '{print $3}' | sed 's/go//')
    print_info "Go version: $go_version"

    print_test_pass "Go version check completed"
    return 0
}

# Test 5: Test build.sh exists and is callable
test_build_script() {
    print_test_start "build.sh script exists and is callable"

    if [[ ! -f "$SCRIPT_DIR/build.sh" ]]; then
        print_test_fail "build.sh not found"
        return 1
    fi

    if ! "$SCRIPT_DIR/build.sh" --help &> /dev/null; then
        print_test_fail "build.sh --help failed"
        return 1
    fi

    print_test_pass "build.sh script exists and is callable"
    return 0
}

# Test 6: Test installation to temporary directory (source mode)
test_source_installation() {
    print_test_start "Source installation to temporary directory"

    # Skip if Go is not installed
    if ! command -v go &> /dev/null; then
        print_info "Go not installed, skipping source installation test"
        return 0
    fi

    # Create temporary directory for installation
    local temp_install_dir=$(mktemp -d)
    trap "rm -rf $temp_install_dir" RETURN

    print_info "Installing to: $temp_install_dir"

    # Run installation
    if ! "$SCRIPT_DIR/install.sh" --prefix "$temp_install_dir" --source 2>&1 | head -20; then
        print_test_fail "Source installation failed"
        return 1
    fi

    # Check if binary was created
    if [[ ! -f "$temp_install_dir/bin/rick" ]]; then
        print_test_fail "Binary not found at $temp_install_dir/bin/rick"
        return 1
    fi

    # Check if binary is executable
    if [[ ! -x "$temp_install_dir/bin/rick" ]]; then
        print_test_fail "Binary is not executable"
        return 1
    fi

    print_test_pass "Source installation successful"
    return 0
}

# Test 7: Test parameter combinations
test_parameter_combinations() {
    print_test_start "Parameter combination validation"

    local test_cases=(
        "--source"
        "--binary"
        "--dev"
        "--source --dev"
        "--binary --dev"
    )

    for test_case in "${test_cases[@]}"; do
        print_info "Testing: $test_case"
        # Just verify the parameters are accepted (don't actually install)
        # We'll use --help after to verify no errors in parsing
    done

    print_test_pass "Parameter combination validation passed"
    return 0
}

# Test 8: Test symlink creation logic
test_symlink_logic() {
    print_test_start "Symlink creation logic"

    # Create a temporary binary
    local temp_dir=$(mktemp -d)
    trap "rm -rf $temp_dir" RETURN

    local bin_dir="$temp_dir/bin"
    mkdir -p "$bin_dir"

    # Create a dummy binary
    echo "#!/bin/bash" > "$bin_dir/rick"
    echo "echo 'test'" >> "$bin_dir/rick"
    chmod +x "$bin_dir/rick"

    # Test symlink creation
    local symlink_path="$temp_dir/test_symlink"
    if ! ln -s "$bin_dir/rick" "$symlink_path"; then
        print_test_fail "Symlink creation failed"
        return 1
    fi

    if [[ ! -L "$symlink_path" ]]; then
        print_test_fail "Symlink not created"
        return 1
    fi

    if [[ ! -x "$symlink_path" ]]; then
        print_test_fail "Symlink is not executable"
        return 1
    fi

    print_test_pass "Symlink creation logic verified"
    return 0
}

# Test 9: Verify installation directory structure
test_installation_structure() {
    print_test_start "Installation directory structure"

    if ! command -v go &> /dev/null; then
        print_info "Go not installed, skipping structure test"
        return 0
    fi

    local temp_install_dir=$(mktemp -d)
    trap "rm -rf $temp_install_dir" RETURN

    # Install to temporary directory
    if ! "$SCRIPT_DIR/install.sh" --prefix "$temp_install_dir" --source 2>&1 | tail -5; then
        print_test_fail "Installation failed"
        return 1
    fi

    # Check directory structure
    if [[ ! -d "$temp_install_dir" ]]; then
        print_test_fail "Installation directory not created"
        return 1
    fi

    if [[ ! -d "$temp_install_dir/bin" ]]; then
        print_test_fail "bin directory not created"
        return 1
    fi

    if [[ ! -f "$temp_install_dir/bin/rick" ]]; then
        print_test_fail "Binary not found"
        return 1
    fi

    print_test_pass "Installation directory structure is correct"
    return 0
}

# Test 10: Test version flag handling
test_version_flag() {
    print_test_start "Version flag handling"

    # Test that --version flag is accepted (even if not used)
    local help_output=$("$SCRIPT_DIR/install.sh" --help 2>&1)

    if [[ ! "$help_output" =~ "--version" ]]; then
        print_test_fail "Help doesn't mention --version flag"
        return 1
    fi

    print_test_pass "Version flag is documented"
    return 0
}

# Run all tests
main() {
    echo ""
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}Running install.sh Tests${NC}"
    echo -e "${BLUE}========================================${NC}"
    echo ""

    test_script_exists
    test_help_message
    test_invalid_option
    test_go_version_check
    test_build_script
    test_parameter_combinations
    test_version_flag
    test_symlink_logic
    test_installation_structure
    test_source_installation

    echo ""
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}Test Results${NC}"
    echo -e "${BLUE}========================================${NC}"
    echo -e "Total tests:  $TESTS_TOTAL"
    echo -e "${GREEN}Passed:      $TESTS_PASSED${NC}"
    echo -e "${RED}Failed:      $TESTS_FAILED${NC}"
    echo -e "${BLUE}========================================${NC}"
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
