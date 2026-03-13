package config

// Config represents the global configuration for Rick CLI
type Config struct {
	MaxRetries       int    `json:"max_retries"`
	ClaudeCodePath   string `json:"claude_code_path"`
	DefaultWorkspace string `json:"default_workspace"`
}
