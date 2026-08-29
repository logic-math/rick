package config

// Config represents the global configuration for Rick CLI
type Config struct {
	MaxRetries       int             `json:"max_retries"`
	Runtime          string          `json:"runtime"`
	PiPath           string          `json:"pi_path"`
	PiExtraArgs      []string        `json:"pi_extra_args,omitempty"`
	DefaultWorkspace string          `json:"default_workspace"`
	Git              GitConfig       `json:"git"`
	HumanLoop        HumanLoopConfig `json:"human_loop"`
	// WebToken is the single-user auth token for `rick web` (job_36).
	// Empty disables auth (local development mode). Auto-generated on first
	// `rick web` start when neither --token nor this field is set (written
	// back + printed once). loader.go needs no change: json round-trips
	// omitempty-transparently.
	WebToken string `json:"web_token,omitempty"`
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
