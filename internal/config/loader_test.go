package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestGetDefaultConfig(t *testing.T) {
	cfg := GetDefaultConfig()

	if cfg.MaxRetries != 5 {
		t.Errorf("expected MaxRetries=5, got %d", cfg.MaxRetries)
	}

	if cfg.DefaultWorkspace == "" {
		t.Error("expected DefaultWorkspace to be set")
	}
}

func TestLoadConfigFileNotExists(t *testing.T) {
	// Temporarily override getConfigPath to use a non-existent file
	oldGetConfigPath := getConfigPath
	getConfigPath = func() (string, error) {
		return "/tmp/nonexistent_rick_config_" + t.Name() + ".json", nil
	}
	defer func() { getConfigPath = oldGetConfigPath }()

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if cfg == nil {
		t.Fatal("expected config to be returned")
	}

	if cfg.MaxRetries != 5 {
		t.Errorf("expected default MaxRetries=5, got %d", cfg.MaxRetries)
	}
}

func TestSaveAndLoadConfig(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()
	testConfigPath := filepath.Join(tmpDir, ".rick", "config.json")

	// Override getConfigPath for this test
	oldGetConfigPath := getConfigPath
	getConfigPath = func() (string, error) {
		return testConfigPath, nil
	}
	defer func() { getConfigPath = oldGetConfigPath }()

	// Create a test config
	testCfg := &Config{
		MaxRetries:       10,
		ClaudeCodePath:   "/usr/local/bin/claude",
		DefaultWorkspace: filepath.Join(tmpDir, ".rick"),
	}

	// Save the config
	err := SaveConfig(testCfg)
	if err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(testConfigPath); os.IsNotExist(err) {
		t.Fatal("config file was not created")
	}

	// Load the config
	loadedCfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// Verify values match
	if loadedCfg.MaxRetries != testCfg.MaxRetries {
		t.Errorf("expected MaxRetries=%d, got %d", testCfg.MaxRetries, loadedCfg.MaxRetries)
	}

	if loadedCfg.ClaudeCodePath != testCfg.ClaudeCodePath {
		t.Errorf("expected ClaudeCodePath=%s, got %s", testCfg.ClaudeCodePath, loadedCfg.ClaudeCodePath)
	}

	if loadedCfg.DefaultWorkspace != testCfg.DefaultWorkspace {
		t.Errorf("expected DefaultWorkspace=%s, got %s", testCfg.DefaultWorkspace, loadedCfg.DefaultWorkspace)
	}
}

func TestValidateConfigMaxRetriesNegative(t *testing.T) {
	cfg := &Config{
		MaxRetries:       -1,
		ClaudeCodePath:   "",
		DefaultWorkspace: "/home/user/.rick",
	}

	err := ValidateConfig(cfg)
	if err == nil {
		t.Error("expected validation error for negative MaxRetries")
	}
}

func TestValidateConfigClaudeCodePathNotExists(t *testing.T) {
	cfg := &Config{
		MaxRetries:       5,
		ClaudeCodePath:   "/nonexistent/path/to/claude",
		DefaultWorkspace: "/home/user/.rick",
	}

	err := ValidateConfig(cfg)
	if err == nil {
		t.Error("expected validation error for non-existent ClaudeCodePath")
	}
}

func TestValidateConfigValid(t *testing.T) {
	// Create a temporary file to use as ClaudeCodePath
	tmpFile, err := os.CreateTemp("", "claude")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	cfg := &Config{
		MaxRetries:       5,
		ClaudeCodePath:   tmpFile.Name(),
		DefaultWorkspace: "/home/user/.rick",
	}

	err = ValidateConfig(cfg)
	if err != nil {
		t.Fatalf("expected no validation error, got %v", err)
	}
}

func TestValidateConfigEmptyClaudeCodePath(t *testing.T) {
	cfg := &Config{
		MaxRetries:       5,
		ClaudeCodePath:   "",
		DefaultWorkspace: "/home/user/.rick",
	}

	err := ValidateConfig(cfg)
	if err != nil {
		t.Fatalf("expected no validation error for empty ClaudeCodePath, got %v", err)
	}
}

func TestConfigJSONMarshaling(t *testing.T) {
	cfg := &Config{
		MaxRetries:       7,
		ClaudeCodePath:   "/path/to/claude",
		DefaultWorkspace: "/home/user/.rick",
	}

	// Marshal to JSON
	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("failed to marshal config: %v", err)
	}

	// Unmarshal back
	var loadedCfg Config
	err = json.Unmarshal(data, &loadedCfg)
	if err != nil {
		t.Fatalf("failed to unmarshal config: %v", err)
	}

	// Verify values match
	if loadedCfg.MaxRetries != cfg.MaxRetries {
		t.Errorf("expected MaxRetries=%d, got %d", cfg.MaxRetries, loadedCfg.MaxRetries)
	}

	if loadedCfg.ClaudeCodePath != cfg.ClaudeCodePath {
		t.Errorf("expected ClaudeCodePath=%s, got %s", cfg.ClaudeCodePath, loadedCfg.ClaudeCodePath)
	}

	if loadedCfg.DefaultWorkspace != cfg.DefaultWorkspace {
		t.Errorf("expected DefaultWorkspace=%s, got %s", cfg.DefaultWorkspace, loadedCfg.DefaultWorkspace)
	}
}

func TestGetConfigPath(t *testing.T) {
	path, err := GetConfigPath()
	if err != nil {
		t.Fatalf("failed to get config path: %v", err)
	}

	if path == "" {
		t.Error("expected non-empty config path")
	}

	// Verify it ends with .rick/config.json
	if !filepath.HasPrefix(path, os.ExpandEnv("$HOME")) {
		t.Errorf("expected config path to be in home directory, got %s", path)
	}
}

func TestSaveConfigCreatesDirectory(t *testing.T) {
	// Create a temporary directory
	tmpDir := t.TempDir()
	testConfigPath := filepath.Join(tmpDir, "nested", "dirs", ".rick", "config.json")

	// Override getConfigPath for this test
	oldGetConfigPath := getConfigPath
	getConfigPath = func() (string, error) {
		return testConfigPath, nil
	}
	defer func() { getConfigPath = oldGetConfigPath }()

	cfg := &Config{
		MaxRetries:       5,
		ClaudeCodePath:   "",
		DefaultWorkspace: tmpDir,
	}

	err := SaveConfig(cfg)
	if err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	// Verify directory and file were created
	if _, err := os.Stat(testConfigPath); os.IsNotExist(err) {
		t.Fatal("config file was not created")
	}
}

func TestLoadConfigWithInvalidJSON(t *testing.T) {
	// Create a temporary directory
	tmpDir := t.TempDir()
	testConfigPath := filepath.Join(tmpDir, "config.json")

	// Override getConfigPath for this test
	oldGetConfigPath := getConfigPath
	getConfigPath = func() (string, error) {
		return testConfigPath, nil
	}
	defer func() { getConfigPath = oldGetConfigPath }()

	// Write invalid JSON to the file
	err := os.WriteFile(testConfigPath, []byte("{invalid json}"), 0644)
	if err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Try to load it
	_, err = LoadConfig()
	if err == nil {
		t.Error("expected error when loading invalid JSON")
	}
}

func TestValidateConfigZeroMaxRetries(t *testing.T) {
	cfg := &Config{
		MaxRetries:       0,
		ClaudeCodePath:   "",
		DefaultWorkspace: "/home/user/.rick",
	}

	err := ValidateConfig(cfg)
	if err != nil {
		t.Fatalf("expected no validation error for zero MaxRetries, got %v", err)
	}
}

func TestLoadConfigReadError(t *testing.T) {
	// Create a directory instead of a file to cause read error
	tmpDir := t.TempDir()
	testConfigPath := filepath.Join(tmpDir, "config_dir")

	// Override getConfigPath for this test
	oldGetConfigPath := getConfigPath
	getConfigPath = func() (string, error) {
		return testConfigPath, nil
	}
	defer func() { getConfigPath = oldGetConfigPath }()

	// Create a directory where the config file should be
	os.MkdirAll(testConfigPath, 0755)

	// Try to load it
	_, err := LoadConfig()
	if err == nil {
		t.Error("expected error when reading directory as file")
	}
}
