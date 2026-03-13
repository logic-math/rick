package feedback

import (
	"fmt"
	"runtime"
	"strings"
)

// ErrorWithContext represents an error with context information
type ErrorWithContext struct {
	Message    string
	ErrorType  string
	StackTrace []StackFrame
	Cause      error
	Context    map[string]interface{}
}

// StackFrame represents a single frame in the stack trace
type StackFrame struct {
	File     string
	Function string
	Line     int
}

// ErrorHandler manages error handling and formatting
type ErrorHandler struct {
	i18n              *I18nMessages
	includeStackTrace bool
	maxStackDepth     int
}

// NewErrorHandler creates a new error handler
func NewErrorHandler(i18n *I18nMessages) *ErrorHandler {
	return &ErrorHandler{
		i18n:              i18n,
		includeStackTrace: false,
		maxStackDepth:     10,
	}
}

// SetIncludeStackTrace enables or disables stack trace inclusion
func (eh *ErrorHandler) SetIncludeStackTrace(include bool) {
	eh.includeStackTrace = include
}

// SetMaxStackDepth sets the maximum depth of stack trace
func (eh *ErrorHandler) SetMaxStackDepth(depth int) {
	eh.maxStackDepth = depth
}

// Handle creates an ErrorWithContext from an error
func (eh *ErrorHandler) Handle(errorType string, err error) *ErrorWithContext {
	ewc := &ErrorWithContext{
		Message:   err.Error(),
		ErrorType: errorType,
		Context:   make(map[string]interface{}),
		Cause:     err,
	}

	if eh.includeStackTrace {
		ewc.StackTrace = captureStackTrace(eh.maxStackDepth)
	}

	return ewc
}

// HandleWithMessage creates an ErrorWithContext with a custom message
func (eh *ErrorHandler) HandleWithMessage(errorType, message string, err error) *ErrorWithContext {
	ewc := &ErrorWithContext{
		Message:   message,
		ErrorType: errorType,
		Context:   make(map[string]interface{}),
		Cause:     err,
	}

	if eh.includeStackTrace {
		ewc.StackTrace = captureStackTrace(eh.maxStackDepth)
	}

	return ewc
}

// AddContext adds context information to the error
func (ewc *ErrorWithContext) AddContext(key string, value interface{}) {
	ewc.Context[key] = value
}

// Format returns a formatted error message
func (ewc *ErrorWithContext) Format(i18n *I18nMessages, verbose bool) string {
	var sb strings.Builder

	// Error message
	sb.WriteString(fmt.Sprintf("❌ [%s] %s\n", ewc.ErrorType, ewc.Message))

	// Context information
	if verbose && len(ewc.Context) > 0 {
		sb.WriteString("\n📋 Context:\n")
		for key, value := range ewc.Context {
			sb.WriteString(fmt.Sprintf("  %s: %v\n", key, value))
		}
	}

	// Stack trace
	if len(ewc.StackTrace) > 0 {
		sb.WriteString("\n📍 Stack Trace:\n")
		for i, frame := range ewc.StackTrace {
			sb.WriteString(fmt.Sprintf("  %d. %s() at %s:%d\n", i+1, frame.Function, frame.File, frame.Line))
		}
	}

	// Cause
	if ewc.Cause != nil && verbose {
		sb.WriteString(fmt.Sprintf("\n🔗 Caused by: %v\n", ewc.Cause))
	}

	return sb.String()
}

// captureStackTrace captures the current stack trace
func captureStackTrace(maxDepth int) []StackFrame {
	var frames []StackFrame

	// Skip this function and the caller
	pcs := make([]uintptr, maxDepth+2)
	n := runtime.Callers(2, pcs)

	for i := 0; i < n && i < maxDepth; i++ {
		pc := pcs[i]
		fn := runtime.FuncForPC(pc)
		if fn == nil {
			continue
		}

		file, line := fn.FileLine(pc)
		frames = append(frames, StackFrame{
			File:     file,
			Function: fn.Name(),
			Line:     line,
		})
	}

	return frames
}

// GetSuggestion returns a suggestion for the error type
func (ewc *ErrorWithContext) GetSuggestion(i18n *I18nMessages) string {
	suggestionKey := "SUG_" + strings.ToUpper(ewc.ErrorType)
	suggestion := i18n.Get(suggestionKey)
	if suggestion == suggestionKey {
		return "" // No suggestion found
	}
	return suggestion
}

// Error implements the error interface
func (ewc *ErrorWithContext) Error() string {
	return fmt.Sprintf("[%s] %s", ewc.ErrorType, ewc.Message)
}
