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
	// WebMaxActive caps concurrent live web-session workers (job_43).
	// 0/negative falls back to the runtime supervisor default (64); the web
	// composition root passes this to NewSupervisor. Sessions persist for
	// the server lifetime (idle reaping disabled), so a generous default is
	// intentional.
	WebMaxActive int `json:"web_max_active,omitempty"`

	// Super carries `rick super` mode settings (job_47/task11): the super
	// workspace root plus the PySupervisor cadence defaults. Zero fields
	// fall back to DefaultSuperConfig() at the call site (loader.go needs no
	// change: json round-trips omitempty-transparently, same as WebToken).
	Super SuperConfig `json:"super"`
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

// SuperConfig carries `rick super` mode settings (job_47/task11). Every
// value mirrors the py-side super/superopt/config.py DEFAULTS so Go (writer
// of super.json) and py (loader) agree unless explicitly overridden — the
// cross-language default parity is asserted in handler/super_test.go and
// gate2/gate4.
type SuperConfig struct {
	// DefaultSuperDir is the root directory holding one super workspace per
	// --name (default ~/.rick/super/<name>/).
	DefaultSuperDir string `json:"default_super_dir,omitempty"`
	// PythonPath launches the py scheduler (default "python3").
	PythonPath string `json:"python_path,omitempty"`
	// HeartbeatIntervalMs / HeartbeatTimeoutMs / TermGraceMs parameterize the
	// Go-side PySupervisor (defaults 30000/90000/5000).
	HeartbeatIntervalMs int `json:"heartbeat_interval_ms,omitempty"`
	HeartbeatTimeoutMs int `json:"heartbeat_timeout_ms,omitempty"`
	TermGraceMs        int `json:"term_grace_ms,omitempty"`
	// AdjustInterval is the adjust-agent cadence in iterations (default 5).
	AdjustInterval int `json:"adjust_interval,omitempty"`
	// DebugMaxAttempts is the debug agent retry budget M (default 3).
	DebugMaxAttempts int `json:"debug_max_attempts,omitempty"`
}

// DefaultSuperConfig returns the built-in super defaults (mirror of the py
// DEFAULTS in super/superopt/config.py).
func DefaultSuperConfig() SuperConfig {
	return SuperConfig{
		DefaultSuperDir:    "",
		PythonPath:         "python3",
		HeartbeatIntervalMs: 30000,
		HeartbeatTimeoutMs: 90000,
		TermGraceMs:        5000,
		AdjustInterval:     5,
		DebugMaxAttempts:   3,
	}
}
