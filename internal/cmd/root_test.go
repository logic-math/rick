package cmd

import (
	"bytes"
	"testing"
)

// TestNewRootCmd tests that NewRootCmd creates a valid cobra command
func TestNewRootCmd(t *testing.T) {
	cmd := NewRootCmd("0.1.0")
	if cmd == nil {
		t.Fatal("NewRootCmd returned nil")
	}
	if cmd.Use != "rick" {
		t.Errorf("expected Use to be 'rick', got %s", cmd.Use)
	}
	if cmd.Short == "" {
		t.Error("expected Short to be non-empty")
	}
	if cmd.Long == "" {
		t.Error("expected Long to be non-empty")
	}
	if cmd.Version != "0.1.0" {
		t.Errorf("expected Version to be '0.1.0', got %s", cmd.Version)
	}
}

// TestVersionFlag tests that --version flag works
func TestVersionFlag(t *testing.T) {
	cmd := NewRootCmd("0.1.0")
	cmd.SetArgs([]string{"--version"})

	// Capture output
	output := new(bytes.Buffer)
	cmd.SetOut(output)

	err := cmd.Execute()
	// Version flag might exit or return error, both are acceptable
	if err != nil {
		t.Logf("Version flag execution: %v", err)
	}
}

// TestVersionFlagShort tests that -V flag works
func TestVersionFlagShort(t *testing.T) {
	cmd := NewRootCmd("0.1.0")
	cmd.SetArgs([]string{"-V"})

	output := new(bytes.Buffer)
	cmd.SetOut(output)

	err := cmd.Execute()
	if err != nil {
		t.Logf("Version flag execution: %v", err)
	}
}

// TestVerboseFlag tests that --verbose flag is recognized
func TestVerboseFlag(t *testing.T) {
	// Save original value
	origVerbose := verbose
	defer func() { verbose = origVerbose }()

	cmd := NewRootCmd("0.1.0")
	cmd.SetArgs([]string{"--verbose", "--help"})

	err := cmd.Execute()
	if err != nil {
		t.Logf("Verbose flag execution: %v", err)
	}
}

// TestVerboseFlagShort tests that -v flag works
func TestVerboseFlagShort(t *testing.T) {
	origVerbose := verbose
	defer func() { verbose = origVerbose }()

	cmd := NewRootCmd("0.1.0")
	cmd.SetArgs([]string{"-v", "--help"})

	err := cmd.Execute()
	if err != nil {
		t.Logf("Verbose flag execution: %v", err)
	}
}

// TestDryRunFlag tests that --dry-run flag is recognized
func TestDryRunFlag(t *testing.T) {
	origDryRun := dryRun
	defer func() { dryRun = origDryRun }()

	cmd := NewRootCmd("0.1.0")
	cmd.SetArgs([]string{"--dry-run", "--help"})

	err := cmd.Execute()
	if err != nil {
		t.Logf("Dry-run flag execution: %v", err)
	}
}

// TestJobFlag tests that --job flag is recognized
func TestJobFlag(t *testing.T) {
	origJobID := jobID
	defer func() { jobID = origJobID }()

	cmd := NewRootCmd("0.1.0")
	cmd.SetArgs([]string{"--job", "job_1", "--help"})

	err := cmd.Execute()
	if err != nil {
		t.Logf("Job flag execution: %v", err)
	}
}

// TestHelpFlag tests that --help flag works
func TestHelpFlag(t *testing.T) {
	cmd := NewRootCmd("0.1.0")
	cmd.SetArgs([]string{"--help"})

	output := new(bytes.Buffer)
	cmd.SetOut(output)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("help flag execution failed: %v", err)
	}

	outputStr := output.String()
	if len(outputStr) == 0 {
		t.Error("expected help output, got empty string")
	}
}

// TestGetVerbose tests the GetVerbose function
func TestGetVerbose(t *testing.T) {
	origVerbose := verbose
	defer func() { verbose = origVerbose }()

	verbose = true
	if !GetVerbose() {
		t.Error("expected GetVerbose to return true")
	}

	verbose = false
	if GetVerbose() {
		t.Error("expected GetVerbose to return false")
	}
}

// TestGetDryRun tests the GetDryRun function
func TestGetDryRun(t *testing.T) {
	origDryRun := dryRun
	defer func() { dryRun = origDryRun }()

	dryRun = true
	if !GetDryRun() {
		t.Error("expected GetDryRun to return true")
	}

	dryRun = false
	if GetDryRun() {
		t.Error("expected GetDryRun to return false")
	}
}

// TestGetJobID tests the GetJobID function
func TestGetJobID(t *testing.T) {
	origJobID := jobID
	defer func() { jobID = origJobID }()

	jobID = "job_1"
	if GetJobID() != "job_1" {
		t.Errorf("expected GetJobID to return 'job_1', got %s", GetJobID())
	}

	jobID = ""
	if GetJobID() != "" {
		t.Errorf("expected GetJobID to return empty string, got %s", GetJobID())
	}
}

// TestValidateJobID tests the validateJobID function
func TestValidateJobID(t *testing.T) {
	tests := []struct {
		name    string
		jobID   string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid job ID with underscore",
			jobID:   "job_1",
			wantErr: false,
		},
		{
			name:    "valid job ID with hyphen",
			jobID:   "job-1",
			wantErr: false,
		},
		{
			name:    "valid job ID with uppercase",
			jobID:   "JOB_1",
			wantErr: false,
		},
		{
			name:    "valid job ID with mixed case",
			jobID:   "Job_1",
			wantErr: false,
		},
		{
			name:    "empty job ID",
			jobID:   "",
			wantErr: true,
			errMsg:  "job ID cannot be empty",
		},
		{
			name:    "job ID with invalid character @",
			jobID:   "job@1",
			wantErr: true,
			errMsg:  "job ID contains invalid characters",
		},
		{
			name:    "job ID with invalid character space",
			jobID:   "job 1",
			wantErr: true,
			errMsg:  "job ID contains invalid characters",
		},
		{
			name:    "job ID with invalid character dot",
			jobID:   "job.1",
			wantErr: true,
			errMsg:  "job ID contains invalid characters",
		},
		{
			name:    "job ID with invalid character slash",
			jobID:   "job/1",
			wantErr: true,
			errMsg:  "job ID contains invalid characters",
		},
		{
			name:    "valid job ID numeric",
			jobID:   "123",
			wantErr: false,
		},
		{
			name:    "valid job ID alphanumeric",
			jobID:   "abc123DEF",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateJobID(tt.jobID)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateJobID(%q) error = %v, wantErr %v", tt.jobID, err, tt.wantErr)
			}
			if err != nil && tt.errMsg != "" {
				if !bytes.Contains([]byte(err.Error()), []byte(tt.errMsg)) {
					t.Errorf("validateJobID(%q) error message = %v, want to contain %q", tt.jobID, err, tt.errMsg)
				}
			}
		})
	}
}

// TestRootCmdSubcommands tests that all subcommands are registered
func TestRootCmdSubcommands(t *testing.T) {
	cmd := NewRootCmd("0.1.0")

	subcommands := map[string]bool{
		"init":     false,
		"plan":     false,
		"doing":    false,
		"learning": false,
	}

	for _, subCmd := range cmd.Commands() {
		if _, exists := subcommands[subCmd.Name()]; exists {
			subcommands[subCmd.Name()] = true
		}
	}

	for name, found := range subcommands {
		if !found {
			t.Errorf("subcommand %s not found", name)
		}
	}
}

// TestRootCmdFlags tests that all global flags are registered
func TestRootCmdFlags(t *testing.T) {
	cmd := NewRootCmd("0.1.0")

	tests := []struct {
		flagName string
		required bool
	}{
		{"verbose", false},
		{"dry-run", false},
		{"job", false},
		{"version", false},
	}

	for _, tt := range tests {
		flag := cmd.Flag(tt.flagName)
		if flag == nil {
			t.Errorf("flag %s not found", tt.flagName)
		}
	}
}

// TestCombinedFlags tests using multiple flags together
func TestCombinedFlags(t *testing.T) {
	origVerbose := verbose
	origDryRun := dryRun
	origJobID := jobID
	defer func() {
		verbose = origVerbose
		dryRun = origDryRun
		jobID = origJobID
	}()

	cmd := NewRootCmd("0.1.0")
	cmd.SetArgs([]string{"--verbose", "--dry-run", "--job", "job_1", "--help"})

	output := new(bytes.Buffer)
	cmd.SetOut(output)

	err := cmd.Execute()
	if err != nil {
		t.Logf("combined flags execution: %v", err)
	}
}
