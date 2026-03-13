#!/bin/bash
#
# update.sh - Update script for Rick CLI
#
# Usage:
#   ./scripts/update.sh [OPTIONS]
#
# Options:
#   --dev                   Update development version (~/.rick_dev)
#   --version VERSION       Update to specific version (default: latest)
#   --prefix PREFIX         Custom installation prefix
#   -h, --help              Show this help message
#
# Examples:
#   ./scripts/update.sh                 # Update production version to latest
#   ./scripts/update.sh --dev           # Update development version to latest
#   ./scripts/update.sh --version 1.0.0 # Update to specific version
#

set -e

# Script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

# Default values
IS_DEV_MODE=false
VERSION="latest"
PREFIX=""
BACKUP_DIR=""

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
                IS_DEV_MODE=true
                shift
                ;;
            --version)
                VERSION="$2"
                shift 2
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

determine_install_dir() {
    if [[ -n "$PREFIX" ]]; then
        echo "$PREFIX"
    elif [[ "$IS_DEV_MODE" == true ]]; then
        echo "$HOME/.rick_dev"
    else
        echo "$HOME/.rick"
    fi
}

determine_command_name() {
    if [[ "$IS_DEV_MODE" == true ]]; then
        echo "rick_dev"
    else
        echo "rick"
    fi
}

get_latest_version() {
    print_debug "Fetching latest version from GitHub..."

    local latest_version=$(curl -s https://api.github.com/repos/anthropics/rick/releases/latest | grep '"tag_name"' | head -1 | sed 's/.*"v\([^"]*\)".*/\1/')

    if [[ -z "$latest_version" ]]; then
        print_error "Failed to fetch latest version from GitHub"
        return 1
    fi

    echo "$latest_version"
    return 0
}

get_current_version() {
    local command_name="$1"

    if ! command -v "$command_name" &> /dev/null; then
        echo "unknown"
        return 0
    fi

    "$command_name" --version 2>&1 | head -1 || echo "unknown"
}

confirm_update() {
    local command_name="$1"
    local current_version="$2"
    local new_version="$3"

    echo ""
    echo -e "${YELLOW}Update Information:${NC}"
    echo "  Command: $command_name"
    echo "  Current version: $current_version"
    echo "  New version: $new_version"
    echo ""

    read -p "Do you want to update? (y/N) " -n 1 -r
    echo

    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        print_info "Update cancelled."
        return 1
    fi

    return 0
}

backup_installation() {
    local install_dir="$1"
    local command_name="$2"

    if [[ ! -d "$install_dir" ]]; then
        print_debug "Installation directory does not exist, no backup needed"
        return 0
    fi

    print_info "Creating backup of current installation..."

    # Create backup directory
    BACKUP_DIR=$(mktemp -d)

    if ! cp -r "$install_dir" "$BACKUP_DIR/backup"; then
        print_error "Failed to create backup"
        rm -rf "$BACKUP_DIR"
        return 1
    fi

    print_debug "Backup created at: $BACKUP_DIR/backup"
    return 0
}

restore_from_backup() {
    local install_dir="$1"

    if [[ -z "$BACKUP_DIR" ]] || [[ ! -d "$BACKUP_DIR/backup" ]]; then
        print_error "No backup available for rollback"
        return 1
    fi

    print_info "Rolling back to previous version..."

    # Remove failed installation
    if [[ -d "$install_dir" ]]; then
        rm -rf "$install_dir"
    fi

    # Restore from backup
    if ! cp -r "$BACKUP_DIR/backup" "$install_dir"; then
        print_error "Failed to restore from backup"
        return 1
    fi

    print_success "Rollback completed successfully"
    return 0
}

cleanup_backup() {
    if [[ -n "$BACKUP_DIR" ]] && [[ -d "$BACKUP_DIR" ]]; then
        rm -rf "$BACKUP_DIR"
    fi
}

perform_update() {
    local install_dir="$1"
    local command_name="$2"
    local version="$3"

    print_info "Starting update process..."

    # Backup current installation
    if ! backup_installation "$install_dir" "$command_name"; then
        return 1
    fi

    # Uninstall current version (without confirmation)
    print_info "Uninstalling current version..."

    if [[ -d "$install_dir" ]]; then
        # Delete symbolic link first
        local symlink_path="$HOME/.local/bin/$command_name"
        if [[ -L "$symlink_path" ]]; then
            rm -f "$symlink_path"
        fi

        # Remove installation directory
        if ! rm -rf "$install_dir"; then
            print_error "Failed to uninstall current version"
            restore_from_backup "$install_dir"
            return 1
        fi
    fi

    # Install new version
    print_info "Installing new version..."

    # Determine install mode (default: source)
    local install_mode="source"

    # Call install.sh
    local install_args=(--prefix "$install_dir")

    if [[ "$IS_DEV_MODE" == true ]]; then
        install_args+=(--dev)
    fi

    # Try source installation first
    if ! "$SCRIPT_DIR/install.sh" "${install_args[@]}"; then
        print_error "Installation of new version failed"
        restore_from_backup "$install_dir"
        return 1
    fi

    print_success "Update completed successfully"
    return 0
}

print_update_summary() {
    local command_name="$1"
    local new_version="$2"

    echo ""
    echo -e "${BLUE}========================================${NC}"
    echo -e "${GREEN}Update Complete!${NC}"
    echo -e "${BLUE}========================================${NC}"
    echo ""
    echo "Rick CLI has been updated successfully."
    echo "Command: $command_name"
    echo "New version: $new_version"
    echo ""
    echo "Test the update:"
    echo "  $command_name --help"
    echo ""
    echo -e "${BLUE}========================================${NC}"
}

# Main execution
main() {
    parse_args "$@"

    local install_dir=$(determine_install_dir)
    local command_name=$(determine_command_name)

    print_info "Starting update process..."
    print_debug "Is dev mode: $IS_DEV_MODE"
    print_debug "Installation directory: $install_dir"
    print_debug "Command name: $command_name"
    print_debug "Target version: $VERSION"

    # Determine target version
    local target_version="$VERSION"
    if [[ "$VERSION" == "latest" ]]; then
        if ! target_version=$(get_latest_version); then
            exit 1
        fi
        print_info "Latest version: $target_version"
    fi

    # Get current version
    local current_version=$(get_current_version "$command_name")
    print_info "Current version: $current_version"

    # Check if update is needed
    if [[ "$current_version" == "$target_version" ]]; then
        print_info "Already on version $target_version. No update needed."
        exit 0
    fi

    # Confirm update
    if ! confirm_update "$command_name" "$current_version" "$target_version"; then
        exit 1
    fi

    # Perform update
    if ! perform_update "$install_dir" "$command_name" "$target_version"; then
        exit 1
    fi

    # Cleanup backup
    cleanup_backup

    # Print summary
    print_update_summary "$command_name" "$target_version"
    print_success "Rick CLI update completed successfully!"
}

# Run main function
main "$@"
