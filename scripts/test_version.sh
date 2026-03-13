#!/bin/bash
#
# test_version.sh - Test script for version management
#
# Tests the version.sh script functionality
#

# Script directory
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
VERSION_SCRIPT="${SCRIPT_DIR}/version.sh"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test counter
TESTS_PASSED=0
TESTS_FAILED=0

echo -e "${YELLOW}========================================${NC}"
echo -e "${YELLOW}Version Management Script Tests${NC}"
echo -e "${YELLOW}========================================${NC}"

# Test 1: Get version
echo -e "\n${YELLOW}Test 1: Get Version${NC}"
version=$("$VERSION_SCRIPT" get)
if [[ "$version" == "0.1.0" ]]; then
    echo -e "${GREEN}✓ PASS${NC}: Get current version (got: $version)"
    ((TESTS_PASSED++))
else
    echo -e "${RED}✗ FAIL${NC}: Get current version (expected: 0.1.0, got: $version)"
    ((TESTS_FAILED++))
fi

# Test 2: Validate version format
echo -e "\n${YELLOW}Test 2: Validate Version Format${NC}"

if "$VERSION_SCRIPT" validate 1.2.3 >/dev/null 2>&1; then
    echo -e "${GREEN}✓ PASS${NC}: Validate version 1.2.3"
    ((TESTS_PASSED++))
else
    echo -e "${RED}✗ FAIL${NC}: Validate version 1.2.3"
    ((TESTS_FAILED++))
fi

if "$VERSION_SCRIPT" validate v1.2.3 >/dev/null 2>&1; then
    echo -e "${GREEN}✓ PASS${NC}: Validate version v1.2.3"
    ((TESTS_PASSED++))
else
    echo -e "${RED}✗ FAIL${NC}: Validate version v1.2.3"
    ((TESTS_FAILED++))
fi

if ! "$VERSION_SCRIPT" validate 1.2 >/dev/null 2>&1; then
    echo -e "${GREEN}✓ PASS${NC}: Reject invalid version 1.2"
    ((TESTS_PASSED++))
else
    echo -e "${RED}✗ FAIL${NC}: Reject invalid version 1.2"
    ((TESTS_FAILED++))
fi

if ! "$VERSION_SCRIPT" validate abc >/dev/null 2>&1; then
    echo -e "${GREEN}✓ PASS${NC}: Reject invalid version abc"
    ((TESTS_PASSED++))
else
    echo -e "${RED}✗ FAIL${NC}: Reject invalid version abc"
    ((TESTS_FAILED++))
fi

# Test 3: Set version
echo -e "\n${YELLOW}Test 3: Set Version${NC}"

# Backup original
cp "${PROJECT_DIR}/cmd/rick/main.go" "${PROJECT_DIR}/cmd/rick/main.go.bak"

# Set to 0.2.0
"$VERSION_SCRIPT" set 0.2.0 >/dev/null 2>&1
version=$("$VERSION_SCRIPT" get)
if [[ "$version" == "0.2.0" ]]; then
    echo -e "${GREEN}✓ PASS${NC}: Set version to 0.2.0"
    ((TESTS_PASSED++))
else
    echo -e "${RED}✗ FAIL${NC}: Set version to 0.2.0 (got: $version)"
    ((TESTS_FAILED++))
fi

# Set to v0.3.0 (with prefix)
"$VERSION_SCRIPT" set v0.3.0 >/dev/null 2>&1
version=$("$VERSION_SCRIPT" get)
if [[ "$version" == "0.3.0" ]]; then
    echo -e "${GREEN}✓ PASS${NC}: Set version to v0.3.0"
    ((TESTS_PASSED++))
else
    echo -e "${RED}✗ FAIL${NC}: Set version to v0.3.0 (got: $version)"
    ((TESTS_FAILED++))
fi

# Restore original
mv "${PROJECT_DIR}/cmd/rick/main.go.bak" "${PROJECT_DIR}/cmd/rick/main.go"

# Test 4: Generate changelog
echo -e "\n${YELLOW}Test 4: Generate Changelog${NC}"

# Backup and remove
if [[ -f "${PROJECT_DIR}/CHANGELOG.md" ]]; then
    cp "${PROJECT_DIR}/CHANGELOG.md" "${PROJECT_DIR}/CHANGELOG.md.bak"
fi
rm -f "${PROJECT_DIR}/CHANGELOG.md"

# Generate changelog
"$VERSION_SCRIPT" changelog >/dev/null 2>&1

# Check if CHANGELOG.md was created
if [[ -f "${PROJECT_DIR}/CHANGELOG.md" ]]; then
    echo -e "${GREEN}✓ PASS${NC}: Changelog file created"
    ((TESTS_PASSED++))
else
    echo -e "${RED}✗ FAIL${NC}: Changelog file not created"
    ((TESTS_FAILED++))
fi

# Check if version entry exists
if grep -q "## \[0.1.0\]" "${PROJECT_DIR}/CHANGELOG.md"; then
    echo -e "${GREEN}✓ PASS${NC}: Version entry in changelog"
    ((TESTS_PASSED++))
else
    echo -e "${RED}✗ FAIL${NC}: Version entry not in changelog"
    ((TESTS_FAILED++))
fi

# Restore or remove
if [[ -f "${PROJECT_DIR}/CHANGELOG.md.bak" ]]; then
    rm -f "${PROJECT_DIR}/CHANGELOG.md"
    mv "${PROJECT_DIR}/CHANGELOG.md.bak" "${PROJECT_DIR}/CHANGELOG.md"
else
    rm -f "${PROJECT_DIR}/CHANGELOG.md"
fi

# Test 5: Create git tag
echo -e "\n${YELLOW}Test 5: Create Git Tag${NC}"

# Remove existing tag if present
git -C "$PROJECT_DIR" tag -d v0.1.0 2>/dev/null || true

# Create tag
"$VERSION_SCRIPT" tag >/dev/null 2>&1

# Check if tag was created
if git -C "$PROJECT_DIR" rev-parse v0.1.0 >/dev/null 2>&1; then
    echo -e "${GREEN}✓ PASS${NC}: Git tag v0.1.0 created"
    ((TESTS_PASSED++))
else
    echo -e "${RED}✗ FAIL${NC}: Git tag v0.1.0 not created"
    ((TESTS_FAILED++))
fi

# Clean up
git -C "$PROJECT_DIR" tag -d v0.1.0 2>/dev/null || true

# Test 6: Invalid version format rejection
echo -e "\n${YELLOW}Test 6: Reject Invalid Version Format${NC}"

# Backup original
cp "${PROJECT_DIR}/cmd/rick/main.go" "${PROJECT_DIR}/cmd/rick/main.go.bak"

# Try to set invalid version
if ! "$VERSION_SCRIPT" set "invalid" >/dev/null 2>&1; then
    echo -e "${GREEN}✓ PASS${NC}: Invalid version rejected"
    ((TESTS_PASSED++))
else
    echo -e "${RED}✗ FAIL${NC}: Should reject invalid version"
    ((TESTS_FAILED++))
fi

# Restore original
mv "${PROJECT_DIR}/cmd/rick/main.go.bak" "${PROJECT_DIR}/cmd/rick/main.go"

# Test 7: Help message
echo -e "\n${YELLOW}Test 7: Help Message${NC}"

if "$VERSION_SCRIPT" --help 2>&1 | grep -q "Usage:"; then
    echo -e "${GREEN}✓ PASS${NC}: Help message displayed"
    ((TESTS_PASSED++))
else
    echo -e "${RED}✗ FAIL${NC}: Help message not displayed"
    ((TESTS_FAILED++))
fi

# Print summary
echo -e "\n${YELLOW}========================================${NC}"
echo -e "${YELLOW}Test Results${NC}"
echo -e "${YELLOW}========================================${NC}"
echo -e "${GREEN}Passed: $TESTS_PASSED${NC}"
echo -e "${RED}Failed: $TESTS_FAILED${NC}"
echo -e "${YELLOW}Total: $((TESTS_PASSED + TESTS_FAILED))${NC}"

if [[ $TESTS_FAILED -eq 0 ]]; then
    echo -e "\n${GREEN}All tests passed!${NC}"
    exit 0
else
    echo -e "\n${RED}Some tests failed!${NC}"
    exit 1
fi
