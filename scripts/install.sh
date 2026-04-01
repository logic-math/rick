#!/bin/bash
#
# install.sh - Installation script for Rick CLI
#
# Usage:
#   ./scripts/install.sh [OPTIONS]
#
# Options:
#   --source                Install from source code (default)
#   --binary                Install pre-built binary from GitHub releases
#   --dev                   Install to development directory (~/.rick_dev)
#   --prefix PREFIX         Custom installation prefix (default: ~/.rick or ~/.rick_dev)
#   --version VERSION       Specify version to install (default: latest)
#   -h, --help              Show this help message
#
# Examples:
#   ./scripts/install.sh                    # Install production version from source
#   ./scripts/install.sh --dev              # Install development version from source
#   ./scripts/install.sh --binary           # Install production version from binary
#   ./scripts/install.sh --binary --dev     # Install development version from binary
#

set -e

# Script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

# Default values
INSTALL_MODE="source"  # source or binary
IS_DEV_MODE=false
PREFIX=""
VERSION="latest"

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
            --source)
                INSTALL_MODE="source"
                shift
                ;;
            --binary)
                INSTALL_MODE="binary"
                shift
                ;;
            --dev)
                IS_DEV_MODE=true
                shift
                ;;
            --prefix)
                PREFIX="$2"
                shift 2
                ;;
            --version)
                VERSION="$2"
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

install_from_source() {
    local install_dir="$1"
    local bin_dir="$install_dir/bin"
    local build_output="$bin_dir/rick"

    print_info "Installing from source code..."

    # Create temporary build directory
    local temp_build_dir=$(mktemp -d)
    trap "rm -rf $temp_build_dir" EXIT

    print_debug "Building to temporary location: $temp_build_dir/rick"

    # Call build.sh to build the binary
    if ! "$SCRIPT_DIR/build.sh" --output "$temp_build_dir/rick"; then
        print_error "Build failed. Installation aborted."
        return 1
    fi

    # Create installation directory structure
    if ! mkdir -p "$bin_dir"; then
        print_error "Failed to create installation directory: $bin_dir"
        return 1
    fi

    # Copy binary to installation directory
    if ! cp "$temp_build_dir/rick" "$build_output"; then
        print_error "Failed to copy binary to installation directory"
        return 1
    fi

    # Make binary executable
    if ! chmod +x "$build_output"; then
        print_error "Failed to make binary executable"
        return 1
    fi

    print_success "Binary installed to: $build_output"
    return 0
}

download_binary_from_github() {
    local install_dir="$1"
    local version="$2"
    local bin_dir="$install_dir/bin"

    print_info "Downloading binary from GitHub releases..."

    # Determine OS and architecture
    local os=$(uname -s | tr '[:upper:]' '[:lower:]')
    local arch=$(uname -m)

    # Map architecture names
    case "$arch" in
        x86_64)
            arch="amd64"
            ;;
        aarch64)
            arch="arm64"
            ;;
        *)
            print_error "Unsupported architecture: $arch"
            return 1
            ;;
    esac

    # Construct download URL
    local release_version="$version"
    if [[ "$version" == "latest" ]]; then
        # Get latest version from GitHub API
        print_debug "Fetching latest version from GitHub..."
        release_version=$(curl -s https://api.github.com/repos/anthropics/rick/releases/latest | grep '"tag_name"' | head -1 | sed 's/.*"v\([^"]*\)".*/\1/')

        if [[ -z "$release_version" ]]; then
            print_error "Failed to fetch latest version from GitHub"
            return 1
        fi
    fi

    local download_url="https://github.com/anthropics/rick/releases/download/v${release_version}/rick-${os}-${arch}"
    local binary_path="$bin_dir/rick"

    print_debug "Download URL: $download_url"

    # Create binary directory
    if ! mkdir -p "$bin_dir"; then
        print_error "Failed to create binary directory: $bin_dir"
        return 1
    fi

    # Download binary
    print_debug "Downloading binary..."
    if ! curl -fsSL -o "$binary_path" "$download_url"; then
        print_error "Failed to download binary from: $download_url"
        return 1
    fi

    # Make binary executable
    if ! chmod +x "$binary_path"; then
        print_error "Failed to make binary executable"
        return 1
    fi

    print_success "Binary downloaded and installed to: $binary_path"
    return 0
}

create_symlink() {
    local install_dir="$1"
    local command_name="$2"
    local bin_dir="$install_dir/bin"
    local binary_path="$bin_dir/rick"
    local symlink_path="$HOME/.local/bin/$command_name"

    print_info "Creating symbolic link..."

    # Create ~/.local/bin if it doesn't exist
    if ! mkdir -p "$HOME/.local/bin"; then
        print_error "Failed to create ~/.local/bin directory"
        return 1
    fi

    # Remove existing symlink if it exists
    if [[ -L "$symlink_path" ]]; then
        print_debug "Removing existing symlink: $symlink_path"
        rm -f "$symlink_path"
    fi

    # Create new symlink
    if ! ln -s "$binary_path" "$symlink_path"; then
        print_error "Failed to create symbolic link: $symlink_path -> $binary_path"
        return 1
    fi

    print_success "Symbolic link created: $symlink_path -> $binary_path"
    return 0
}

verify_installation() {
    local command_name="$1"

    print_info "Verifying installation..."

    # Check if command is available in PATH
    if ! command -v "$command_name" &> /dev/null; then
        print_error "Command '$command_name' not found in PATH"
        print_info "Make sure ~/.local/bin is in your PATH environment variable"
        return 1
    fi

    # Try to run the command
    if ! "$command_name" --version &> /dev/null; then
        print_error "Failed to run '$command_name --version'"
        return 1
    fi

    local version=$("$command_name" --version 2>&1 || echo "unknown")
    print_success "Installation verified. Version: $version"
    return 0
}

install_skills() {
    local skills_src="$PROJECT_DIR/skills"
    local claude_skills_dir="$HOME/.claude/skills"

    if [[ ! -d "$skills_src" ]]; then
        print_info "No skills directory found, skipping skills installation."
        return 0
    fi

    print_info "Installing Rick skills..."
    mkdir -p "$claude_skills_dir"

    local installed=0
    local skipped=0

    for skill_dir in "$skills_src"/*/; do
        if [[ -f "$skill_dir/SKILL.md" ]]; then
            local skill_name
            skill_name=$(basename "$skill_dir")

            local target_claude="$claude_skills_dir/$skill_name"
            if [[ -L "$target_claude" ]]; then
                rm "$target_claude"
            fi
            if [[ -d "$target_claude" ]]; then
                print_debug "Skipping (non-symlink exists): $skill_name"
                skipped=$((skipped + 1))
            else
                ln -s "$skill_dir" "$target_claude"
                print_success "Skill installed (Claude Code): $skill_name"
                installed=$((installed + 1))
            fi
        fi
    done

    print_success "Skills installed: $installed, skipped: $skipped"
}

verify_skills() {
    local skills_src="$PROJECT_DIR/skills"
    local claude_skills_dir="$HOME/.claude/skills"
    local required_skill="sense-human-loop"

    print_info "Verifying skills installation..."

    # Check sense-human-loop is installed and SKILL.md is readable
    local skill_link="$claude_skills_dir/$required_skill"
    if [[ ! -e "$skill_link" ]]; then
        print_error "Required skill not found: $skill_link"
        return 1
    fi

    local skill_md="$skill_link/SKILL.md"
    if [[ ! -f "$skill_md" ]]; then
        print_error "SKILL.md missing: $skill_md"
        return 1
    fi

    # Verify the SKILL.md contains expected content
    if ! grep -q "sense-human-loop" "$skill_md"; then
        print_error "SKILL.md does not look valid: $skill_md"
        return 1
    fi

    print_success "Skill verified: $required_skill -> $(readlink "$skill_link")"

    # List all installed skills
    print_info "Installed skills:"
    for skill_dir in "$skills_src"/*/; do
        if [[ -f "$skill_dir/SKILL.md" ]]; then
            local skill_name
            skill_name=$(basename "$skill_dir")
            local desc
            desc=$(grep -m1 '^description:' "$skill_dir/SKILL.md" 2>/dev/null | sed 's/^description: *"//' | sed 's/"$//' | cut -c1-60 || echo "")
            echo "  • $skill_name${desc:+: $desc...}"
        fi
    done

    return 0
}

print_path_hint() {
    local command_name="$1"

    echo ""
    echo -e "${BLUE}========================================${NC}"
    echo -e "${GREEN}Installation Complete!${NC}"
    echo -e "${BLUE}========================================${NC}"
    echo ""
    echo "Command name: $command_name"
    echo "Installation directory: $(determine_install_dir)"
    echo ""
    echo "To use the command, ensure ~/.local/bin is in your PATH:"
    echo ""
    echo "  # Add to ~/.bashrc, ~/.zshrc, or equivalent:"
    echo "  export PATH=\"\$HOME/.local/bin:\$PATH\""
    echo ""
    echo "Then reload your shell:"
    echo "  source ~/.bashrc  # or source ~/.zshrc"
    echo ""
    echo "Or start a new terminal session."
    echo ""
    echo "Test the installation:"
    echo "  $command_name --help"
    echo ""
    echo -e "${BLUE}========================================${NC}"
}

# Main execution
main() {
    parse_args "$@"

    local install_dir=$(determine_install_dir)
    local command_name=$(determine_command_name)

    print_info "Starting installation process..."
    print_debug "Install mode: $INSTALL_MODE"
    print_debug "Installation directory: $install_dir"
    print_debug "Command name: $command_name"
    print_debug "Is dev mode: $IS_DEV_MODE"

    # Check prerequisites
    if [[ "$INSTALL_MODE" == "source" ]]; then
        if ! check_go_version; then
            exit 1
        fi
    fi

    # Install binary
    if [[ "$INSTALL_MODE" == "source" ]]; then
        if ! install_from_source "$install_dir"; then
            exit 1
        fi
    elif [[ "$INSTALL_MODE" == "binary" ]]; then
        if ! download_binary_from_github "$install_dir" "$VERSION"; then
            exit 1
        fi
    else
        print_error "Unknown install mode: $INSTALL_MODE"
        exit 1
    fi

    # Create symbolic link
    if ! create_symlink "$install_dir" "$command_name"; then
        print_error "Failed to create symbolic link, but binary is installed at: $install_dir/bin/rick"
        print_info "You can manually add $install_dir/bin to your PATH"
        exit 1
    fi

    # Verify installation
    if ! verify_installation "$command_name"; then
        print_info "Installation completed, but verification failed."
        print_info "Try running: $command_name --help"
        print_info "Or add ~/.local/bin to your PATH and try again."
    fi

    # Install and verify skills
    install_skills
    if ! verify_skills; then
        print_error "Skills verification failed. Check $PROJECT_DIR/skills/ directory."
        exit 1
    fi

    print_path_hint "$command_name"
    print_success "Rick CLI installation completed successfully!"
}

# Run main function
main "$@"
