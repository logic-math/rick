package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// getConfigPath is a variable that can be overridden for testing
var getConfigPath = func() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	// Determine the rick directory name based on the binary name
	binaryPath := os.Args[0]
	binaryName := filepath.Base(binaryPath)
	rickDirName := ".rick"
	if strings.HasSuffix(binaryName, "_dev") {
		rickDirName = ".rick_dev"
	}

	return filepath.Join(home, rickDirName, "config.json"), nil
}

// GetConfigPath returns the path to the config file
func GetConfigPath() (string, error) {
	return getConfigPath()
}

// GetDefaultConfig returns the default configuration
func GetDefaultConfig() *Config {
	home, _ := os.UserHomeDir()

	// Determine the rick directory name based on the binary name
	binaryPath := os.Args[0]
	binaryName := filepath.Base(binaryPath)
	rickDirName := ".rick"
	if strings.HasSuffix(binaryName, "_dev") {
		rickDirName = ".rick_dev"
	}

	return &Config{
		MaxRetries:       5,
		ClaudeCodePath:   "",
		DefaultWorkspace: filepath.Join(home, rickDirName),
		Git: GitConfig{
			UserName:  "Rick CLI",
			UserEmail: "rick@localhost",
		},
	}
}

// LoadConfig loads configuration from ~/.rick/config.json
func LoadConfig() (*Config, error) {
	configPath, err := GetConfigPath()
	if err != nil {
		return nil, err
	}

	// If config file doesn't exist, create it with default config
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		cfg := GetDefaultConfig()
		if err := SaveConfig(cfg); err != nil {
			return nil, fmt.Errorf("failed to create default config: %w", err)
		}
		return cfg, nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &cfg, nil
}

// SaveConfig saves configuration to ~/.rick/config.json
func SaveConfig(cfg *Config) error {
	configPath, err := GetConfigPath()
	if err != nil {
		return err
	}

	// Ensure the directory exists
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// ValidateConfig validates the configuration
func ValidateConfig(cfg *Config) error {
	if cfg.MaxRetries < 0 {
		return fmt.Errorf("MaxRetries must be non-negative, got %d", cfg.MaxRetries)
	}

	// Check if ClaudeCodePath exists (only if it's not empty)
	if cfg.ClaudeCodePath != "" {
		if _, err := os.Stat(cfg.ClaudeCodePath); os.IsNotExist(err) {
			return fmt.Errorf("ClaudeCodePath does not exist: %s", cfg.ClaudeCodePath)
		}
	}

	return nil
}
