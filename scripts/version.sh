#!/bin/bash
#
# version.sh - Version management script for Rick CLI
#
# Usage:
#   ./scripts/version.sh get              # Get current version
#   ./scripts/version.sh set VERSION      # Set version
#   ./scripts/version.sh validate VERSION # Validate version format
#   ./scripts/version.sh tag              # Create git tag for current version
#   ./scripts/version.sh changelog        # Generate changelog
#   ./scripts/version.sh release VERSION  # Full release process
#   ./scripts/version.sh -h, --help       # Show this help message
#

# Script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Version file path
VERSION_FILE="${PROJECT_DIR}/cmd/rick/main.go"
CHANGELOG_FILE="${PROJECT_DIR}/CHANGELOG.md"

# Helper functions
print_error() {
    echo -e "${RED}Error: $1${NC}" >&2
}

print_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

print_info() {
    echo -e "${YELLOW}ℹ $1${NC}"
}

show_help() {
    sed -n '2,/^$/p' "$0" | sed 's/^# //'
}

# Get current version from main.go
get_version() {
    grep 'const VERSION = ' "$VERSION_FILE" | sed 's/.*const VERSION = "\([^"]*\)".*/\1/' || {
        print_error "Could not parse version from $VERSION_FILE"
        return 1
    }
}

# Validate version format (vMAJOR.MINOR.PATCH or MAJOR.MINOR.PATCH)
validate_version() {
    local version="$1"

    # Remove leading 'v' if present
    version="${version#v}"

    # Check format: MAJOR.MINOR.PATCH
    if [[ ! "$version" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
        print_error "Invalid version format: $1. Expected: vMAJOR.MINOR.PATCH or MAJOR.MINOR.PATCH"
        return 1
    fi

    return 0
}

# Set version in main.go
set_version() {
    local new_version="$1"

    # Validate format
    if ! validate_version "$new_version"; then
        return 1
    fi

    # Remove leading 'v' if present
    new_version="${new_version#v}"

    # Update main.go
    if [[ "$OSTYPE" == "darwin"* ]]; then
        # macOS
        sed -i '' "s/const VERSION = \"[^\"]*\"/const VERSION = \"$new_version\"/" "$VERSION_FILE"
    else
        # Linux
        sed -i "s/const VERSION = \"[^\"]*\"/const VERSION = \"$new_version\"/" "$VERSION_FILE"
    fi

    print_success "Version updated to $new_version in main.go"
    return 0
}

# Create git tag for current version
create_tag() {
    local version="$(get_version)" || return 1
    local tag="v${version}"

    # Check if tag already exists
    if git -C "$PROJECT_DIR" rev-parse "$tag" >/dev/null 2>&1; then
        print_error "Tag $tag already exists"
        return 1
    fi

    # Create annotated tag
    git -C "$PROJECT_DIR" tag -a "$tag" -m "Release $tag" || {
        print_error "Failed to create tag $tag"
        return 1
    }

    print_success "Created tag $tag"
    return 0
}

# Generate changelog entry
generate_changelog_entry() {
    local version="$1"
    local date="$(date '+%Y-%m-%d')"

    # Remove leading 'v' if present
    version="${version#v}"

    cat <<EOF
## [$version] - $date

### Added
- Add your changes here

### Fixed
- Fix your changes here

### Changed
- Change your changes here

EOF
}

# Generate changelog
generate_changelog() {
    local version="$(get_version)" || return 1

    # Check if CHANGELOG.md exists
    if [[ ! -f "$CHANGELOG_FILE" ]]; then
        print_info "Creating new CHANGELOG.md"
        cat > "$CHANGELOG_FILE" <<'EOF'
# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

EOF
    fi

    # Generate new entry
    local new_entry="$(generate_changelog_entry "$version")"

    # Create temporary file with new entry
    local tmp_file="$(mktemp)"
    {
        head -n 6 "$CHANGELOG_FILE"
        echo ""
        echo "$new_entry"
        tail -n +7 "$CHANGELOG_FILE"
    } > "$tmp_file"
    mv "$tmp_file" "$CHANGELOG_FILE"

    print_success "Changelog entry generated for version $version"
    return 0
}

# Full release process
release() {
    local new_version="$1"

    if [[ -z "$new_version" ]]; then
        print_error "Version required for release"
        return 1
    fi

    # Validate version
    if ! validate_version "$new_version"; then
        return 1
    fi

    # Check if working directory is clean
    if ! git -C "$PROJECT_DIR" diff-index --quiet HEAD --; then
        print_error "Working directory is not clean. Commit or stash changes first."
        return 1
    fi

    print_info "Starting release process for version $new_version"

    # Update version
    if ! set_version "$new_version"; then
        return 1
    fi

    # Generate changelog entry
    if ! generate_changelog "$new_version"; then
        return 1
    fi

    # Commit version changes
    git -C "$PROJECT_DIR" add "$VERSION_FILE" "$CHANGELOG_FILE" || {
        print_error "Failed to stage version files"
        return 1
    }

    git -C "$PROJECT_DIR" commit -m "chore: bump version to $new_version" || {
        print_error "Failed to commit version changes"
        return 1
    }

    # Create tag
    if ! create_tag; then
        return 1
    fi

    print_success "Release process completed successfully"
    print_info "Next step: git push origin main && git push origin v$new_version"
    return 0
}

# Main command handler
main() {
    local command="${1:-get}"

    case "$command" in
        get)
            get_version
            ;;
        set)
            if [[ -z "$2" ]]; then
                print_error "Version required for 'set' command"
                return 1
            fi
            set_version "$2"
            ;;
        validate)
            if [[ -z "$2" ]]; then
                print_error "Version required for 'validate' command"
                return 1
            fi
            if validate_version "$2"; then
                print_success "Version format is valid: $2"
                return 0
            else
                return 1
            fi
            ;;
        tag)
            create_tag
            ;;
        changelog)
            generate_changelog
            ;;
        release)
            if [[ -z "$2" ]]; then
                print_error "Version required for 'release' command"
                return 1
            fi
            release "$2"
            ;;
        -h|--help|help)
            show_help
            ;;
        *)
            print_error "Unknown command: $command"
            show_help
            return 1
            ;;
    esac
}

main "$@"
