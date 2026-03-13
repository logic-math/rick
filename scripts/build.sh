#!/bin/bash
#
# build.sh - Build script for Rick CLI
#
# Usage:
#   ./scripts/build.sh [--output OUTPUT_PATH]
#
# Options:
#   --output OUTPUT_PATH    Specify output binary path (default: ./bin/rick)
#   -h, --help              Show this help message
#

set -e

# Script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

# Default output path
OUTPUT_PATH="${PROJECT_DIR}/bin/rick"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Functions
print_error() {
    echo -e "${RED}[ERROR]${NC} $1" >&2
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_info() {
    echo -e "${YELLOW}[INFO]${NC} $1"
}

show_help() {
    sed -n '2,/^$/p' "$0" | sed 's/^# //' | sed 's/^#//'
}

check_go_version() {
    if ! command -v go &> /dev/null; then
        print_error "Go is not installed. Please install Go 1.21 or later."
        return 1
    fi

    local go_version=$(go version | awk '{print $3}' | sed 's/go//')
    local required_version="1.21"

    # Compare versions
    if ! printf '%s\n' "$required_version" "$go_version" | sort -V -C; then
        print_error "Go version $go_version is not supported. Please use Go $required_version or later."
        return 1
    fi

    print_info "Go version $go_version is compatible."
    return 0
}

parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            --output)
                OUTPUT_PATH="$2"
                shift 2
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

build_binary() {
    # Ensure output directory exists
    local output_dir=$(dirname "$OUTPUT_PATH")
    if ! mkdir -p "$output_dir"; then
        print_error "Failed to create output directory: $output_dir"
        return 1
    fi

    print_info "Building Rick CLI..."
    print_info "Output path: $OUTPUT_PATH"

    # Build the binary
    if ! go build -o "$OUTPUT_PATH" "$PROJECT_DIR/cmd/rick"; then
        print_error "Build failed. Please check the error messages above."
        return 1
    fi

    print_success "Build completed successfully."
    return 0
}

verify_binary() {
    if [[ ! -f "$OUTPUT_PATH" ]]; then
        print_error "Binary file not found: $OUTPUT_PATH"
        return 1
    fi

    if [[ ! -x "$OUTPUT_PATH" ]]; then
        print_error "Binary is not executable: $OUTPUT_PATH"
        return 1
    fi

    # Try to run the binary with --version or --help
    if ! "$OUTPUT_PATH" --help &> /dev/null; then
        print_error "Binary verification failed. The binary may be corrupted."
        return 1
    fi

    print_info "Binary verification passed."
    return 0
}

# Main execution
main() {
    parse_args "$@"

    print_info "Starting build process..."

    if ! check_go_version; then
        exit 1
    fi

    if ! build_binary; then
        exit 1
    fi

    if ! verify_binary; then
        exit 1
    fi

    print_success "Rick CLI build completed successfully!"
    echo ""
    echo "Binary location: $OUTPUT_PATH"
}

# Run main function
main "$@"
