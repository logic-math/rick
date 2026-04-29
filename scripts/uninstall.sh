#!/bin/bash
#
# uninstall.sh - Uninstallation script for Rick CLI
#
# Usage:
#   ./scripts/uninstall.sh [OPTIONS]
#
# Options:
#   --dev                   Uninstall development version (~/.rick_dev)
#   --all                   Uninstall both production and development versions
#   --prefix PREFIX         Custom installation prefix to uninstall from
#   -h, --help              Show this help message
#
# Examples:
#   ./scripts/uninstall.sh              # Uninstall production version
#   ./scripts/uninstall.sh --dev        # Uninstall development version
#   ./scripts/uninstall.sh --all        # Uninstall both versions
#

set -e

# Script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

# Default values
UNINSTALL_DEV=false
UNINSTALL_ALL=false
PREFIX=""

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
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

print_debug() {
    echo -e "${BLUE}[DEBUG]${NC} $1"
}

show_help() {
    sed -n '2,/^$/p' "$0" | sed 's/^# //' | sed 's/^#//'
}

parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            --dev)
                UNINSTALL_DEV=true
                shift
                ;;
            --all)
                UNINSTALL_ALL=true
                shift
                ;;
            --prefix)
                PREFIX="$2"
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

confirm_uninstall() {
    local install_dir="$1"
    local command_name="$2"

    echo -e "${YELLOW}Warning:${NC} This will uninstall $command_name from:"
    echo "  $install_dir"
    echo ""
    read -p "Are you sure you want to continue? (y/N) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        print_info "Uninstallation cancelled."
        return 1
    fi
    return 0
}

delete_symlink() {
    local command_name="$1"
    local symlink_path="$HOME/.local/bin/$command_name"

    if [[ -L "$symlink_path" ]]; then
        print_debug "Removing symbolic link: $symlink_path"
        if ! rm -f "$symlink_path"; then
            print_error "Failed to remove symbolic link: $symlink_path"
            return 1
        fi
        print_success "Symbolic link removed: $symlink_path"
    else
        print_debug "Symbolic link not found: $symlink_path"
    fi

    return 0
}

uninstall_version() {
    local install_dir="$1"
    local command_name="$2"

    if [[ ! -d "$install_dir" ]]; then
        print_info "Installation directory not found: $install_dir"
        return 0
    fi

    print_info "Uninstalling $command_name from $install_dir..."

    # Confirm uninstallation
    if ! confirm_uninstall "$install_dir" "$command_name"; then
        return 1
    fi

    # Delete symbolic link first
    if ! delete_symlink "$command_name"; then
        print_error "Failed to delete symbolic link"
        return 1
    fi

    # Remove installation directory
    print_debug "Removing installation directory: $install_dir"
    if ! rm -rf "$install_dir"; then
        print_error "Failed to remove installation directory: $install_dir"
        return 1
    fi

    print_success "Uninstalled $command_name from: $install_dir"
    return 0
}

print_uninstall_summary() {
    echo ""
    echo -e "${BLUE}========================================${NC}"
    echo -e "${GREEN}Uninstallation Complete!${NC}"
    echo -e "${BLUE}========================================${NC}"
    echo ""
    echo "Rick CLI has been successfully uninstalled."
    echo ""
    if [[ -d "$HOME/.local/bin" ]] && [[ -z "$(ls -A "$HOME/.local/bin" 2>/dev/null)" ]]; then
        echo "Note: ~/.local/bin is now empty. You may want to remove it:"
        echo "  rmdir ~/.local/bin"
    fi
    echo ""
    echo -e "${BLUE}========================================${NC}"
}

# Main execution
main() {
    parse_args "$@"

    print_info "Starting uninstallation process..."

    local uninstall_count=0

    # Determine what to uninstall
    if [[ "$UNINSTALL_ALL" == true ]]; then
        # Uninstall both versions
        print_debug "Uninstalling both production and development versions"

        if uninstall_version "$HOME/.rick" "rick"; then
            ((uninstall_count++))
        fi

        if uninstall_version "$HOME/.rick_dev" "rick_dev"; then
            ((uninstall_count++))
        fi
    elif [[ "$UNINSTALL_DEV" == true ]]; then
        # Uninstall development version
        print_debug "Uninstalling development version"
        local install_dir="${PREFIX:-$HOME/.rick_dev}"

        if uninstall_version "$install_dir" "rick_dev"; then
            ((uninstall_count++))
        fi
    else
        # Uninstall production version (default)
        print_debug "Uninstalling production version"
        local install_dir="${PREFIX:-$HOME/.rick}"

        if uninstall_version "$install_dir" "rick"; then
            ((uninstall_count++))
        fi
    fi

    if [[ $uninstall_count -gt 0 ]]; then
        print_uninstall_summary
        print_success "Rick CLI uninstallation completed successfully!"
        return 0
    else
        print_error "Uninstallation failed or was cancelled."
        return 1
    fi
}

# Run main function
main "$@"
