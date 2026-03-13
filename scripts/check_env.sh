#!/bin/bash
#
# check_env.sh - Environment check script for Rick CLI
#
# Usage:
#   ./scripts/check_env.sh [--verbose] [--json]
#
# Options:
#   --verbose               Show detailed information
#   --json                  Output results in JSON format
#   -h, --help              Show this help message
#

# Script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Options
VERBOSE=false
JSON_OUTPUT=false

# Check results (using simple variables instead of associative arrays for portability)
GO_VERSION_STATUS="UNKNOWN"
GO_VERSION_MSG=""
CLAUDE_CODE_STATUS="UNKNOWN"
CLAUDE_CODE_MSG=""
GIT_STATUS="UNKNOWN"
GIT_MSG=""
PATH_STATUS="UNKNOWN"
PATH_MSG=""

# Functions
print_error() {
    if [ "$JSON_OUTPUT" != "true" ]; then
        echo -e "${RED}[ERROR]${NC} $1" >&2
    fi
}

print_success() {
    if [ "$JSON_OUTPUT" != "true" ]; then
        echo -e "${GREEN}[✓]${NC} $1"
    fi
}

print_warning() {
    if [ "$JSON_OUTPUT" != "true" ]; then
        echo -e "${YELLOW}[⚠]${NC} $1"
    fi
}

print_info() {
    if [ "$JSON_OUTPUT" != "true" ]; then
        echo -e "${BLUE}[INFO]${NC} $1"
    fi
}

print_verbose() {
    if [ "$VERBOSE" = "true" ] && [ "$JSON_OUTPUT" != "true" ]; then
        echo "  $1"
    fi
}

show_help() {
    sed -n '2,/^$/p' "$0" | sed 's/^# //' | sed 's/^#//'
}

parse_args() {
    while [ $# -gt 0 ]; do
        case $1 in
            --verbose)
                VERBOSE=true
                shift
                ;;
            --json)
                JSON_OUTPUT=true
                shift
                ;;
            -h|--help)
                show_help
                exit 0
                ;;
            *)
                print_error "Unknown option: $1"
                show_help
                exit 1
                ;;
        esac
    done
}

check_go_version() {
    local required_version="1.21"

    if ! command -v go &> /dev/null; then
        GO_VERSION_STATUS="FAIL"
        GO_VERSION_MSG="Go is not installed"
        print_error "Go is not installed. Please install Go $required_version or later."
        return 1
    fi

    local go_version=$(go version | awk '{print $3}' | sed 's/go//')

    # Compare versions
    if ! printf '%s\n' "$required_version" "$go_version" | sort -V -C; then
        GO_VERSION_STATUS="FAIL"
        GO_VERSION_MSG="Go version $go_version is not supported (required: >= $required_version)"
        print_error "Go version $go_version is not supported. Please use Go $required_version or later."
        return 1
    fi

    GO_VERSION_STATUS="PASS"
    GO_VERSION_MSG="Go version $go_version"
    print_success "Go version: $go_version"
    print_verbose "Installation: $(command -v go)"
    return 0
}

check_claude_code() {
    if ! command -v claude-code &> /dev/null; then
        CLAUDE_CODE_STATUS="FAIL"
        CLAUDE_CODE_MSG="Claude Code CLI is not installed or not in PATH"
        print_error "Claude Code CLI is not installed or not in PATH."
        print_verbose "Install with: npm install -g @anthropic-ai/claude-code-cli"
        return 1
    fi

    local claude_version=$(claude-code --version 2>/dev/null || echo "unknown")

    CLAUDE_CODE_STATUS="PASS"
    CLAUDE_CODE_MSG="Claude Code CLI is installed (version: $claude_version)"
    print_success "Claude Code CLI is installed"
    print_verbose "Version: $claude_version"
    print_verbose "Installation: $(command -v claude-code)"
    return 0
}

check_git() {
    if ! command -v git &> /dev/null; then
        GIT_STATUS="FAIL"
        GIT_MSG="Git is not installed"
        print_error "Git is not installed. Please install Git."
        return 1
    fi

    local git_version=$(git --version | awk '{print $3}')

    GIT_STATUS="PASS"
    GIT_MSG="Git version $git_version"
    print_success "Git version: $git_version"
    print_verbose "Installation: $(command -v git)"
    return 0
}

check_path() {
    # Check if current directory is in PATH
    local rick_bin_path="$HOME/.rick/bin"
    local rick_dev_bin_path="$HOME/.rick_dev/bin"

    if echo ":$PATH:" | grep -q ":$rick_bin_path:"; then
        PATH_STATUS="PASS"
        PATH_MSG="Rick production bin directory is in PATH"
        print_success "Rick bin directory is in PATH: $rick_bin_path"
        return 0
    elif echo ":$PATH:" | grep -q ":$rick_dev_bin_path:"; then
        PATH_STATUS="PASS"
        PATH_MSG="Rick dev bin directory is in PATH"
        print_success "Rick dev bin directory is in PATH: $rick_dev_bin_path"
        return 0
    else
        PATH_STATUS="WARNING"
        PATH_MSG="Rick bin directory not in PATH"
        print_warning "Rick bin directory not in PATH"
        print_verbose "Add to PATH with: export PATH=\"\$HOME/.rick/bin:\$PATH\""
        return 0  # This is a warning, not a failure
    fi
}

generate_report() {
    if [ "$JSON_OUTPUT" = "true" ]; then
        # Generate JSON report
        local overall_status="PASS"
        if [ "$GO_VERSION_STATUS" = "FAIL" ] || [ "$CLAUDE_CODE_STATUS" = "FAIL" ] || [ "$GIT_STATUS" = "FAIL" ]; then
            overall_status="FAIL"
        elif [ "$PATH_STATUS" = "WARNING" ]; then
            overall_status="WARNING"
        fi

        cat <<EOF
{
  "status": "$overall_status",
  "checks": {
    "go_version": {
      "status": "$GO_VERSION_STATUS",
      "message": "$GO_VERSION_MSG"
    },
    "claude_code": {
      "status": "$CLAUDE_CODE_STATUS",
      "message": "$CLAUDE_CODE_MSG"
    },
    "git": {
      "status": "$GIT_STATUS",
      "message": "$GIT_MSG"
    },
    "path": {
      "status": "$PATH_STATUS",
      "message": "$PATH_MSG"
    }
  },
  "timestamp": "$(date -u +'%Y-%m-%dT%H:%M:%SZ')"
}
EOF
    else
        # Generate text report
        echo ""
        print_info "Environment Check Report"
        echo "========================"
        echo ""

        # Determine overall status
        local overall_status="✓ PASS"
        if [ "$GO_VERSION_STATUS" = "FAIL" ] || [ "$CLAUDE_CODE_STATUS" = "FAIL" ] || [ "$GIT_STATUS" = "FAIL" ]; then
            overall_status="✗ FAIL"
        elif [ "$PATH_STATUS" = "WARNING" ]; then
            overall_status="⚠ WARNING"
        fi

        echo "Overall Status: $overall_status"
        echo ""
        echo "Detailed Results:"
        echo "  Go Version:      $GO_VERSION_STATUS - $GO_VERSION_MSG"
        echo "  Claude Code:     $CLAUDE_CODE_STATUS - $CLAUDE_CODE_MSG"
        echo "  Git:             $GIT_STATUS - $GIT_MSG"
        echo "  PATH:            $PATH_STATUS - $PATH_MSG"
        echo ""
        echo "Timestamp: $(date)"
        echo ""
    fi
}

# Main execution
main() {
    parse_args "$@"

    if [ "$JSON_OUTPUT" != "true" ]; then
        print_info "Checking Rick CLI environment requirements..."
        echo ""
    fi

    local all_passed=true

    if ! check_go_version; then
        all_passed=false
    fi

    if ! check_claude_code; then
        all_passed=false
    fi

    if ! check_git; then
        all_passed=false
    fi

    check_path  # PATH check is a warning, not a failure

    echo ""
    generate_report

    if [ "$all_passed" = "false" ]; then
        exit 1
    fi

    exit 0
}

# Run main function
main "$@"
