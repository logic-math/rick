package config

// Config represents the global configuration for Rick CLI
type Config struct {
	MaxRetries       int             `json:"max_retries"`
	PiPath           string          `json:"pi_path"`
	PiExtraArgs      []string        `json:"pi_extra_args,omitempty"`
	DefaultWorkspace string          `json:"default_workspace"`
	Git              GitConfig       `json:"git"`
	HumanLoop        HumanLoopConfig `json:"human_loop"`
}

// GitConfig represents Git-related configuration
type GitConfig struct {
	UserName  string `json:"user_name"`
	UserEmail string `json:"user_email"`
}

// HumanLoopConfig represents human-loop specific configuration
type HumanLoopConfig struct {
	MaxRetries            int                `json:"max_retries"`
	ResearchSourceWeights map[string]float64 `json:"research_source_weights,omitempty"`
	ThinkTopN             int                `json:"think_top_n,omitempty"`
	SenseMaxBackflows     int                `json:"sense_max_backflows,omitempty"`
	ThinkMinAssumptions   int                `json:"think_min_assumptions,omitempty"`
}
