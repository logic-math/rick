package errors

import "fmt"

// ConfigError represents a configuration-related error
type ConfigError struct {
	message string
}

// NewConfigError creates a new ConfigError
func NewConfigError(msg string) *ConfigError {
	return &ConfigError{message: msg}
}

func (e *ConfigError) Error() string {
	return fmt.Sprintf("[ConfigError] %s", e.message)
}

// WorkspaceError represents a workspace-related error
type WorkspaceError struct {
	message string
}

// NewWorkspaceError creates a new WorkspaceError
func NewWorkspaceError(msg string) *WorkspaceError {
	return &WorkspaceError{message: msg}
}

func (e *WorkspaceError) Error() string {
	return fmt.Sprintf("[WorkspaceError] %s", e.message)
}

// ParserError represents a parser-related error
type ParserError struct {
	message string
}

// NewParserError creates a new ParserError
func NewParserError(msg string) *ParserError {
	return &ParserError{message: msg}
}

func (e *ParserError) Error() string {
	return fmt.Sprintf("[ParserError] %s", e.message)
}

// ExecutorError represents an executor-related error
type ExecutorError struct {
	message string
}

// NewExecutorError creates a new ExecutorError
func NewExecutorError(msg string) *ExecutorError {
	return &ExecutorError{message: msg}
}

func (e *ExecutorError) Error() string {
	return fmt.Sprintf("[ExecutorError] %s", e.message)
}

// GitError represents a git-related error
type GitError struct {
	message string
}

// NewGitError creates a new GitError
func NewGitError(msg string) *GitError {
	return &GitError{message: msg}
}

func (e *GitError) Error() string {
	return fmt.Sprintf("[GitError] %s", e.message)
}
