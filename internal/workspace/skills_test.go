package workspace

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadSkillsIndex_FileExists(t *testing.T) {
	tmpDir := t.TempDir()
	skillsDir := filepath.Join(tmpDir, SkillsDirName)
	if err := os.MkdirAll(skillsDir, 0755); err != nil {
		t.Fatalf("Failed to create skills dir: %v", err)
	}
	content := "# Skills Index\n\n| 文件 | 描述 |\n|------|------|\n| check_go_build.py | 检查 Go 编译 |\n"
	if err := os.WriteFile(filepath.Join(skillsDir, "index.md"), []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write index.md: %v", err)
	}

	result, err := LoadSkillsIndex(tmpDir)
	if err != nil {
		t.Fatalf("LoadSkillsIndex() error: %v", err)
	}
	if result != content {
		t.Errorf("Expected content %q, got %q", content, result)
	}
}

func TestLoadSkillsIndex_FileNotExists(t *testing.T) {
	tmpDir := t.TempDir()
	// No skills directory or index.md

	result, err := LoadSkillsIndex(tmpDir)
	if err != nil {
		t.Fatalf("LoadSkillsIndex() must not error when file is missing, got: %v", err)
	}
	if result != "" {
		t.Errorf("Expected empty string when index.md missing, got %q", result)
	}
}

func TestLoadSkillsIndex_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	skillsDir := filepath.Join(tmpDir, SkillsDirName)
	if err := os.MkdirAll(skillsDir, 0755); err != nil {
		t.Fatalf("Failed to create skills dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(skillsDir, "index.md"), []byte(""), 0644); err != nil {
		t.Fatalf("Failed to write empty index.md: %v", err)
	}

	result, err := LoadSkillsIndex(tmpDir)
	if err != nil {
		t.Fatalf("LoadSkillsIndex() error: %v", err)
	}
	if result != "" {
		t.Errorf("Expected empty string for empty index.md, got %q", result)
	}
}

func TestLoadSkillsIndex_SkillsDirExistsButNoIndex(t *testing.T) {
	tmpDir := t.TempDir()
	skillsDir := filepath.Join(tmpDir, SkillsDirName)
	if err := os.MkdirAll(skillsDir, 0755); err != nil {
		t.Fatalf("Failed to create skills dir: %v", err)
	}
	// skills dir exists but no index.md

	result, err := LoadSkillsIndex(tmpDir)
	if err != nil {
		t.Fatalf("LoadSkillsIndex() must not error when index.md is absent, got: %v", err)
	}
	if result != "" {
		t.Errorf("Expected empty string when index.md absent, got %q", result)
	}
}
