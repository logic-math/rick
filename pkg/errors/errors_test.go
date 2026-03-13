package errors

import (
	"strings"
	"testing"
)

func TestNewConfigError(t *testing.T) {
	msg := "invalid config file"
	err := NewConfigError(msg)

	if err == nil {
		t.Fatal("NewConfigError returned nil")
	}

	errStr := err.Error()
	if !strings.Contains(errStr, "[ConfigError]") {
		t.Errorf("expected [ConfigError] prefix, got: %s", errStr)
	}
	if !strings.Contains(errStr, msg) {
		t.Errorf("expected message %q, got: %s", msg, errStr)
	}
}

func TestNewWorkspaceError(t *testing.T) {
	msg := "workspace not found"
	err := NewWorkspaceError(msg)

	if err == nil {
		t.Fatal("NewWorkspaceError returned nil")
	}

	errStr := err.Error()
	if !strings.Contains(errStr, "[WorkspaceError]") {
		t.Errorf("expected [WorkspaceError] prefix, got: %s", errStr)
	}
	if !strings.Contains(errStr, msg) {
		t.Errorf("expected message %q, got: %s", msg, errStr)
	}
}

func TestNewParserError(t *testing.T) {
	msg := "invalid markdown syntax"
	err := NewParserError(msg)

	if err == nil {
		t.Fatal("NewParserError returned nil")
	}

	errStr := err.Error()
	if !strings.Contains(errStr, "[ParserError]") {
		t.Errorf("expected [ParserError] prefix, got: %s", errStr)
	}
	if !strings.Contains(errStr, msg) {
		t.Errorf("expected message %q, got: %s", msg, errStr)
	}
}

func TestNewExecutorError(t *testing.T) {
	msg := "execution failed"
	err := NewExecutorError(msg)

	if err == nil {
		t.Fatal("NewExecutorError returned nil")
	}

	errStr := err.Error()
	if !strings.Contains(errStr, "[ExecutorError]") {
		t.Errorf("expected [ExecutorError] prefix, got: %s", errStr)
	}
	if !strings.Contains(errStr, msg) {
		t.Errorf("expected message %q, got: %s", msg, errStr)
	}
}

func TestNewGitError(t *testing.T) {
	msg := "git command failed"
	err := NewGitError(msg)

	if err == nil {
		t.Fatal("NewGitError returned nil")
	}

	errStr := err.Error()
	if !strings.Contains(errStr, "[GitError]") {
		t.Errorf("expected [GitError] prefix, got: %s", errStr)
	}
	if !strings.Contains(errStr, msg) {
		t.Errorf("expected message %q, got: %s", msg, errStr)
	}
}

// Test that all error types implement error interface
func TestErrorInterface(t *testing.T) {
	var _ error = NewConfigError("test")
	var _ error = NewWorkspaceError("test")
	var _ error = NewParserError("test")
	var _ error = NewExecutorError("test")
	var _ error = NewGitError("test")
}

// Test error messages with special characters
func TestErrorMessagesWithSpecialChars(t *testing.T) {
	testCases := []struct {
		name      string
		errFunc   func(string) error
		prefix    string
		message   string
	}{
		{
			name:    "ConfigError with path",
			errFunc: func(msg string) error { return NewConfigError(msg) },
			prefix:  "[ConfigError]",
			message: "file not found: /home/user/.rick/config.json",
		},
		{
			name:    "WorkspaceError with newline",
			errFunc: func(msg string) error { return NewWorkspaceError(msg) },
			prefix:  "[WorkspaceError]",
			message: "failed to create directory\ncause: permission denied",
		},
		{
			name:    "ParserError with quotes",
			errFunc: func(msg string) error { return NewParserError(msg) },
			prefix:  "[ParserError]",
			message: `unexpected token: "}"`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.errFunc(tc.message)
			errStr := err.Error()
			if !strings.Contains(errStr, tc.prefix) {
				t.Errorf("expected prefix %q, got: %s", tc.prefix, errStr)
			}
			if !strings.Contains(errStr, tc.message) {
				t.Errorf("expected message %q, got: %s", tc.message, errStr)
			}
		})
	}
}

// Test empty error messages
func TestEmptyErrorMessages(t *testing.T) {
	testCases := []struct {
		name    string
		errFunc func(string) error
		prefix  string
	}{
		{
			name:    "ConfigError empty",
			errFunc: func(msg string) error { return NewConfigError(msg) },
			prefix:  "[ConfigError]",
		},
		{
			name:    "WorkspaceError empty",
			errFunc: func(msg string) error { return NewWorkspaceError(msg) },
			prefix:  "[WorkspaceError]",
		},
		{
			name:    "ParserError empty",
			errFunc: func(msg string) error { return NewParserError(msg) },
			prefix:  "[ParserError]",
		},
		{
			name:    "ExecutorError empty",
			errFunc: func(msg string) error { return NewExecutorError(msg) },
			prefix:  "[ExecutorError]",
		},
		{
			name:    "GitError empty",
			errFunc: func(msg string) error { return NewGitError(msg) },
			prefix:  "[GitError]",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.errFunc("")
			errStr := err.Error()
			if !strings.Contains(errStr, tc.prefix) {
				t.Errorf("expected prefix %q, got: %s", tc.prefix, errStr)
			}
		})
	}
}
