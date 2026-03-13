package git

import (
	"fmt"
	"os/exec"
	"regexp"
	"sort"
	"strings"
	"time"
)

// VersionTag represents a version tag in the repository
type VersionTag struct {
	Tag     string
	Hash    string
	Message string
	Date    time.Time
}

// VersionManager handles version management operations
type VersionManager struct {
	gm *GitManager
}

// NewVersionManager creates a new VersionManager instance
func NewVersionManager(gm *GitManager) *VersionManager {
	return &VersionManager{
		gm: gm,
	}
}

// ValidateVersionFormat validates version number format (vMAJOR.MINOR.PATCH)
func ValidateVersionFormat(version string) bool {
	// Pattern: v followed by MAJOR.MINOR.PATCH (all numeric)
	pattern := `^v\d+\.\d+\.\d+$`
	matched, err := regexp.MatchString(pattern, version)
	return err == nil && matched
}

// CreateTag creates a new version tag
// version should be in format: vMAJOR.MINOR.PATCH (e.g., v1.0.0)
func (vm *VersionManager) CreateTag(version, message string) error {
	if version == "" {
		return fmt.Errorf("version cannot be empty")
	}

	if !ValidateVersionFormat(version) {
		return fmt.Errorf("invalid version format: %s (expected vMAJOR.MINOR.PATCH)", version)
	}

	if message == "" {
		message = fmt.Sprintf("Release %s", version)
	}

	// Create annotated tag with message
	cmd := exec.Command("git", "tag", "-a", version, "-m", message)
	cmd.Dir = vm.gm.repoPath
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create tag %s: %w", version, err)
	}

	return nil
}

// GetCurrentVersion returns the current version (latest tag)
// Returns empty string if no tags exist
func (vm *VersionManager) GetCurrentVersion() (string, error) {
	// Get the latest tag
	cmd := exec.Command("git", "describe", "--tags", "--abbrev=0")
	cmd.Dir = vm.gm.repoPath
	output, err := cmd.Output()
	if err != nil {
		// No tags exist
		return "", nil
	}

	version := strings.TrimSpace(string(output))
	return version, nil
}

// ListVersions returns all version tags sorted in descending order (newest first)
func (vm *VersionManager) ListVersions() ([]VersionTag, error) {
	// Get all tags
	cmd := exec.Command("git", "tag", "-l")
	cmd.Dir = vm.gm.repoPath
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list tags: %w", err)
	}

	var versions []VersionTag
	tags := strings.Split(strings.TrimSpace(string(output)), "\n")

	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		if tag == "" {
			continue
		}

		// Only include tags that match version format
		if !ValidateVersionFormat(tag) {
			continue
		}

		// Get tag details (hash, message, date)
		hash, err := vm.getTagHash(tag)
		if err != nil {
			continue
		}

		message, err := vm.getTagMessage(tag)
		if err != nil {
			continue
		}

		date, err := vm.getTagDate(tag)
		if err != nil {
			continue
		}

		versions = append(versions, VersionTag{
			Tag:     tag,
			Hash:    hash,
			Message: message,
			Date:    date,
		})
	}

	// Sort in descending order (newest first)
	sort.Slice(versions, func(i, j int) bool {
		return versions[i].Date.After(versions[j].Date)
	})

	return versions, nil
}

// Checkout switches to a specific version
// version can be a tag name (e.g., v1.0.0) or commit hash
func (vm *VersionManager) Checkout(version string) error {
	if version == "" {
		return fmt.Errorf("version cannot be empty")
	}

	cmd := exec.Command("git", "checkout", version)
	cmd.Dir = vm.gm.repoPath
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to checkout version %s: %w", version, err)
	}

	return nil
}

// getTagHash returns the commit hash of a tag
func (vm *VersionManager) getTagHash(tag string) (string, error) {
	cmd := exec.Command("git", "rev-list", "-n", "1", tag)
	cmd.Dir = vm.gm.repoPath
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get tag hash: %w", err)
	}

	return strings.TrimSpace(string(output)), nil
}

// getTagMessage returns the message of an annotated tag
func (vm *VersionManager) getTagMessage(tag string) (string, error) {
	cmd := exec.Command("git", "tag", "-l", tag, "-n", "1")
	cmd.Dir = vm.gm.repoPath
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get tag message: %w", err)
	}

	// Output format: "tag_name message"
	line := strings.TrimSpace(string(output))
	parts := strings.SplitN(line, " ", 2)
	if len(parts) > 1 {
		return parts[1], nil
	}

	return "", nil
}

// getTagDate returns the date of a tag
func (vm *VersionManager) getTagDate(tag string) (time.Time, error) {
	cmd := exec.Command("git", "log", "-1", "--format=%ai", tag)
	cmd.Dir = vm.gm.repoPath
	output, err := cmd.Output()
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to get tag date: %w", err)
	}

	dateStr := strings.TrimSpace(string(output))
	date, err := time.Parse("2006-01-02 15:04:05 -0700", dateStr)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to parse tag date: %w", err)
	}

	return date, nil
}

// DeleteTag deletes a version tag
func (vm *VersionManager) DeleteTag(tag string) error {
	if tag == "" {
		return fmt.Errorf("tag cannot be empty")
	}

	cmd := exec.Command("git", "tag", "-d", tag)
	cmd.Dir = vm.gm.repoPath
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to delete tag %s: %w", tag, err)
	}

	return nil
}

// TagExists checks if a tag exists
func (vm *VersionManager) TagExists(tag string) (bool, error) {
	cmd := exec.Command("git", "tag", "-l", tag)
	cmd.Dir = vm.gm.repoPath
	output, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("failed to check tag existence: %w", err)
	}

	return strings.TrimSpace(string(output)) != "", nil
}

// GetVersionInfo returns detailed information about a specific version
func (vm *VersionManager) GetVersionInfo(version string) (*VersionTag, error) {
	if version == "" {
		return nil, fmt.Errorf("version cannot be empty")
	}

	// Check if tag exists
	exists, err := vm.TagExists(version)
	if err != nil {
		return nil, err
	}

	if !exists {
		return nil, fmt.Errorf("version tag %s not found", version)
	}

	// Get tag details
	hash, err := vm.getTagHash(version)
	if err != nil {
		return nil, err
	}

	message, err := vm.getTagMessage(version)
	if err != nil {
		return nil, err
	}

	date, err := vm.getTagDate(version)
	if err != nil {
		return nil, err
	}

	return &VersionTag{
		Tag:     version,
		Hash:    hash,
		Message: message,
		Date:    date,
	}, nil
}
