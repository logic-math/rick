package config

// Config represents the global configuration for Rick CLI
type Config struct {
	MaxRetries       int              `json:"max_retries"`
	ClaudeCodePath   string           `json:"claude_code_path"`
	DefaultWorkspace string           `json:"default_workspace"`
	Git              GitConfig        `json:"git"`
	HumanLoop        HumanLoopConfig  `json:"human_loop"`
}

// GitConfig represents Git-related configuration
type GitConfig struct {
	UserName  string `json:"user_name"`
	UserEmail string `json:"user_email"`
}

// HumanLoopConfig represents human-loop specific configuration
type HumanLoopConfig struct {
	MaxRetries int `json:"max_retries"`
}
