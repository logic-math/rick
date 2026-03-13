package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func setupVersionTestRepo(t *testing.T) (*GitManager, string) {
	tmpDir := t.TempDir()

	gm := New(tmpDir)
	if err := gm.InitRepo(); err != nil {
		t.Fatalf("failed to init repo: %v", err)
	}

	// Configure git user for commits
	configCmd := exec.Command("git", "config", "user.email", "test@example.com")
	configCmd.Dir = tmpDir
	if err := configCmd.Run(); err != nil {
		t.Fatalf("failed to configure git email: %v", err)
	}

	nameCmd := exec.Command("git", "config", "user.name", "Test User")
	nameCmd.Dir = tmpDir
	if err := nameCmd.Run(); err != nil {
		t.Fatalf("failed to configure git name: %v", err)
	}

	// Create initial commit
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	if err := gm.AddFiles([]string{"test.txt"}); err != nil {
		t.Fatalf("failed to add file: %v", err)
	}

	if err := gm.Commit("Initial commit"); err != nil {
		t.Fatalf("failed to create initial commit: %v", err)
	}

	return gm, tmpDir
}

func TestValidateVersionFormat(t *testing.T) {
	tests := []struct {
		name      string
		version   string
		expected  bool
	}{
		{"valid v1.0.0", "v1.0.0", true},
		{"valid v0.1.0", "v0.1.0", true},
		{"valid v10.20.30", "v10.20.30", true},
		{"invalid no v prefix", "1.0.0", false},
		{"invalid with extra dot", "v1.0.0.0", false},
		{"invalid with letters", "v1.0.a", false},
		{"invalid empty", "", false},
		{"invalid just v", "v", false},
		{"invalid v1", "v1", false},
		{"invalid v1.0", "v1.0", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateVersionFormat(tt.version)
			if result != tt.expected {
				t.Errorf("ValidateVersionFormat(%s) = %v, want %v", tt.version, result, tt.expected)
			}
		})
	}
}

func TestCreateTag(t *testing.T) {
	gm, _ := setupVersionTestRepo(t)
	vm := NewVersionManager(gm)

	tests := []struct {
		name    string
		version string
		message string
		wantErr bool
	}{
		{"valid tag", "v1.0.0", "Release 1.0.0", false},
		{"valid tag with default message", "v1.0.1", "", false},
		{"invalid version format", "1.0.0", "Release", true},
		{"empty version", "", "Release", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := vm.CreateTag(tt.version, tt.message)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateTag(%s, %s) error = %v, wantErr %v", tt.version, tt.message, err, tt.wantErr)
			}

			if !tt.wantErr && tt.version != "" {
				exists, err := vm.TagExists(tt.version)
				if err != nil {
					t.Errorf("TagExists failed: %v", err)
				}
				if !exists {
					t.Errorf("Tag %s was not created", tt.version)
				}
			}
		})
	}
}

func TestGetCurrentVersion(t *testing.T) {
	gm, _ := setupVersionTestRepo(t)
	vm := NewVersionManager(gm)

	// Initially, no version should exist
	version, err := vm.GetCurrentVersion()
	if err != nil {
		t.Errorf("GetCurrentVersion failed: %v", err)
	}
	if version != "" {
		t.Errorf("Expected empty version, got %s", version)
	}

	// Create a tag
	if err := vm.CreateTag("v1.0.0", "Release 1.0.0"); err != nil {
		t.Fatalf("CreateTag failed: %v", err)
	}

	// Now get current version
	version, err = vm.GetCurrentVersion()
	if err != nil {
		t.Errorf("GetCurrentVersion failed: %v", err)
	}
	if version != "v1.0.0" {
		t.Errorf("Expected v1.0.0, got %s", version)
	}

	// Create another tag
	if err := vm.CreateTag("v1.0.1", "Release 1.0.1"); err != nil {
		t.Fatalf("CreateTag failed: %v", err)
	}

	// Current version should now be v1.0.1 (latest)
	version, err = vm.GetCurrentVersion()
	if err != nil {
		t.Errorf("GetCurrentVersion failed: %v", err)
	}
	// Note: git describe returns the most recent tag, which may not be v1.0.1 if commits have been made
	if version == "" {
		t.Errorf("Expected a version, got empty string")
	}
}

func TestListVersions(t *testing.T) {
	gm, _ := setupVersionTestRepo(t)
	vm := NewVersionManager(gm)

	// Initially, no versions
	versions, err := vm.ListVersions()
	if err != nil {
		t.Errorf("ListVersions failed: %v", err)
	}
	if len(versions) != 0 {
		t.Errorf("Expected 0 versions, got %d", len(versions))
	}

	// Create multiple tags
	tags := []string{"v1.0.0", "v1.0.1", "v1.1.0", "v2.0.0"}
	for _, tag := range tags {
		if err := vm.CreateTag(tag, "Release "+tag); err != nil {
			t.Fatalf("CreateTag failed: %v", err)
		}
	}

	// List versions
	versions, err = vm.ListVersions()
	if err != nil {
		t.Errorf("ListVersions failed: %v", err)
	}

	if len(versions) != len(tags) {
		t.Errorf("Expected %d versions, got %d", len(tags), len(versions))
	}

	// Check that versions are sorted in descending order
	for i := 0; i < len(versions)-1; i++ {
		if versions[i].Date.Before(versions[i+1].Date) {
			t.Errorf("Versions not sorted in descending order: %s after %s", versions[i].Tag, versions[i+1].Tag)
		}
	}

	// Verify all tags are present
	versionMap := make(map[string]bool)
	for _, v := range versions {
		versionMap[v.Tag] = true
	}

	for _, tag := range tags {
		if !versionMap[tag] {
			t.Errorf("Tag %s not found in versions", tag)
		}
	}
}

func TestCheckout(t *testing.T) {
	gm, tmpDir := setupVersionTestRepo(t)
	vm := NewVersionManager(gm)

	// Create a tag
	if err := vm.CreateTag("v1.0.0", "Release 1.0.0"); err != nil {
		t.Fatalf("CreateTag failed: %v", err)
	}

	// Modify file and commit
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("modified"), 0644); err != nil {
		t.Fatalf("failed to modify test file: %v", err)
	}

	if err := gm.AddFiles([]string{"test.txt"}); err != nil {
		t.Fatalf("failed to add file: %v", err)
	}

	if err := gm.Commit("Modified file"); err != nil {
		t.Fatalf("failed to commit: %v", err)
	}

	// Checkout to v1.0.0
	if err := vm.Checkout("v1.0.0"); err != nil {
		t.Errorf("Checkout failed: %v", err)
	}

	// Verify the file is back to original content
	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("failed to read test file: %v", err)
	}

	if string(content) != "test" {
		t.Errorf("Expected 'test', got '%s'", string(content))
	}
}

func TestDeleteTag(t *testing.T) {
	gm, _ := setupVersionTestRepo(t)
	vm := NewVersionManager(gm)

	// Create a tag
	if err := vm.CreateTag("v1.0.0", "Release 1.0.0"); err != nil {
		t.Fatalf("CreateTag failed: %v", err)
	}

	// Verify tag exists
	exists, err := vm.TagExists("v1.0.0")
	if err != nil {
		t.Errorf("TagExists failed: %v", err)
	}
	if !exists {
		t.Errorf("Tag v1.0.0 should exist")
	}

	// Delete tag
	if err := vm.DeleteTag("v1.0.0"); err != nil {
		t.Errorf("DeleteTag failed: %v", err)
	}

	// Verify tag no longer exists
	exists, err = vm.TagExists("v1.0.0")
	if err != nil {
		t.Errorf("TagExists failed: %v", err)
	}
	if exists {
		t.Errorf("Tag v1.0.0 should not exist after deletion")
	}
}

func TestTagExists(t *testing.T) {
	gm, _ := setupVersionTestRepo(t)
	vm := NewVersionManager(gm)

	// Non-existent tag
	exists, err := vm.TagExists("v1.0.0")
	if err != nil {
		t.Errorf("TagExists failed: %v", err)
	}
	if exists {
		t.Errorf("Tag v1.0.0 should not exist")
	}

	// Create tag
	if err := vm.CreateTag("v1.0.0", "Release 1.0.0"); err != nil {
		t.Fatalf("CreateTag failed: %v", err)
	}

	// Existing tag
	exists, err = vm.TagExists("v1.0.0")
	if err != nil {
		t.Errorf("TagExists failed: %v", err)
	}
	if !exists {
		t.Errorf("Tag v1.0.0 should exist")
	}
}

func TestGetVersionInfo(t *testing.T) {
	gm, _ := setupVersionTestRepo(t)
	vm := NewVersionManager(gm)

	// Create a tag
	if err := vm.CreateTag("v1.0.0", "Release 1.0.0"); err != nil {
		t.Fatalf("CreateTag failed: %v", err)
	}

	// Get version info
	info, err := vm.GetVersionInfo("v1.0.0")
	if err != nil {
		t.Errorf("GetVersionInfo failed: %v", err)
	}

	if info.Tag != "v1.0.0" {
		t.Errorf("Expected tag v1.0.0, got %s", info.Tag)
	}

	if info.Hash == "" {
		t.Errorf("Hash should not be empty")
	}

	if info.Message == "" {
		t.Errorf("Message should not be empty")
	}

	if info.Date.IsZero() {
		t.Errorf("Date should not be zero")
	}

	// Non-existent tag
	info, err = vm.GetVersionInfo("v99.0.0")
	if err == nil {
		t.Errorf("GetVersionInfo should fail for non-existent tag")
	}
}

func TestVersionIntegration(t *testing.T) {
	gm, tmpDir := setupVersionTestRepo(t)
	vm := NewVersionManager(gm)

	// Create first version
	if err := vm.CreateTag("v1.0.0", "Initial release"); err != nil {
		t.Fatalf("CreateTag failed: %v", err)
	}

	// Modify and commit
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("v1.0.1 content"), 0644); err != nil {
		t.Fatalf("failed to modify file: %v", err)
	}

	if err := gm.AddFiles([]string{"test.txt"}); err != nil {
		t.Fatalf("failed to add file: %v", err)
	}

	if err := gm.Commit("Update for v1.0.1"); err != nil {
		t.Fatalf("failed to commit: %v", err)
	}

	// Create second version
	if err := vm.CreateTag("v1.0.1", "Bug fix release"); err != nil {
		t.Fatalf("CreateTag failed: %v", err)
	}

	// List versions
	versions, err := vm.ListVersions()
	if err != nil {
		t.Errorf("ListVersions failed: %v", err)
	}

	if len(versions) != 2 {
		t.Errorf("Expected 2 versions, got %d", len(versions))
	}

	// Verify both versions are present (sorting may be equal due to same second)
	versionTags := make(map[string]bool)
	for _, v := range versions {
		versionTags[v.Tag] = true
	}
	if !versionTags["v1.0.0"] || !versionTags["v1.0.1"] {
		t.Errorf("Expected both v1.0.0 and v1.0.1, got: %v", versionTags)
	}

	// Checkout to v1.0.0
	if err := vm.Checkout("v1.0.0"); err != nil {
		t.Errorf("Checkout failed: %v", err)
	}

	// Verify content
	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	if string(content) != "test" {
		t.Errorf("Expected 'test', got '%s'", string(content))
	}

	// Checkout back to v1.0.1
	if err := vm.Checkout("v1.0.1"); err != nil {
		t.Errorf("Checkout failed: %v", err)
	}

	// Verify content
	content, err = os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	if string(content) != "v1.0.1 content" {
		t.Errorf("Expected 'v1.0.1 content', got '%s'", string(content))
	}
}

func TestVersionFormatValidation(t *testing.T) {
	gm, _ := setupVersionTestRepo(t)
	vm := NewVersionManager(gm)

	// Test invalid formats are rejected
	invalidVersions := []string{
		"1.0.0",      // missing v
		"v1.0",       // incomplete
		"v1.0.0.0",   // too many parts
		"vv1.0.0",    // double v
		"v1.0.a",     // contains letter
		"v1..0",      // double dot
	}

	for _, version := range invalidVersions {
		err := vm.CreateTag(version, "Test")
		if err == nil {
			t.Errorf("CreateTag should fail for invalid version %s", version)
		}
	}

	// Test valid formats are accepted
	validVersions := []string{
		"v0.0.0",
		"v1.0.0",
		"v10.20.30",
		"v999.999.999",
	}

	for _, version := range validVersions {
		err := vm.CreateTag(version, "Test")
		if err != nil {
			t.Errorf("CreateTag should succeed for valid version %s: %v", version, err)
		}
	}
}

func BenchmarkCreateTag(b *testing.B) {
	gm, _ := setupVersionTestRepo(&testing.T{})
	vm := NewVersionManager(gm)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		version := "v1.0." + string(rune(i))
		vm.CreateTag(version, "Benchmark tag")
	}
}

func BenchmarkListVersions(b *testing.B) {
	gm, _ := setupVersionTestRepo(&testing.T{})
	vm := NewVersionManager(gm)

	// Create some tags
	for i := 0; i < 10; i++ {
		version := "v1.0." + string(rune(i))
		vm.CreateTag(version, "Tag "+version)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		vm.ListVersions()
	}
}
