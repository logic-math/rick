package feedback

import (
	"fmt"
	"io"
	"log"
	"os"
)

// VerboseLogger wraps the standard logger with verbose mode support
type VerboseLogger struct {
	verboseEnabled bool
	infoLogger     *log.Logger
	warnLogger     *log.Logger
	errorLogger    *log.Logger
	debugLogger    *log.Logger
	verboseLogger  *log.Logger
	traceLogger    *log.Logger
}

// NewVerboseLogger creates a new verbose logger
func NewVerboseLogger(verbose bool) *VerboseLogger {
	return NewVerboseLoggerWithWriter(verbose, os.Stdout)
}

// NewVerboseLoggerWithWriter creates a new verbose logger with custom writer
func NewVerboseLoggerWithWriter(verbose bool, w io.Writer) *VerboseLogger {
	return &VerboseLogger{
		verboseEnabled: verbose,
		infoLogger:     log.New(w, "ℹ️  [INFO] ", log.LstdFlags),
		warnLogger:     log.New(w, "⚠️  [WARN] ", log.LstdFlags),
		errorLogger:    log.New(w, "❌ [ERROR] ", log.LstdFlags),
		debugLogger:    log.New(w, "🐛 [DEBUG] ", log.LstdFlags),
		verboseLogger:  log.New(w, "📝 [VERBOSE] ", log.LstdFlags),
		traceLogger:    log.New(w, "🔍 [TRACE] ", log.LstdFlags),
	}
}

// SetVerbose sets the verbose mode
func (vl *VerboseLogger) SetVerbose(verbose bool) {
	vl.verboseEnabled = verbose
}

// IsVerbose returns whether verbose mode is enabled
func (vl *VerboseLogger) IsVerbose() bool {
	return vl.verboseEnabled
}

// Info logs an info level message
func (vl *VerboseLogger) Info(format string, args ...interface{}) {
	vl.infoLogger.Printf(format, args...)
}

// Warn logs a warning level message
func (vl *VerboseLogger) Warn(format string, args ...interface{}) {
	vl.warnLogger.Printf(format, args...)
}

// Error logs an error level message
func (vl *VerboseLogger) Error(format string, args ...interface{}) {
	vl.errorLogger.Printf(format, args...)
}

// Debug logs a debug level message
func (vl *VerboseLogger) Debug(format string, args ...interface{}) {
	if vl.verboseEnabled {
		vl.debugLogger.Printf(format, args...)
	}
}

// Verbose logs a verbose level message (only shown in verbose mode)
func (vl *VerboseLogger) Verbose(format string, args ...interface{}) {
	if vl.verboseEnabled {
		vl.verboseLogger.Printf(format, args...)
	}
}

// Trace logs a trace level message (only shown in verbose mode)
func (vl *VerboseLogger) Trace(format string, args ...interface{}) {
	if vl.verboseEnabled {
		vl.traceLogger.Printf(format, args...)
	}
}

// VerboseF is a helper function that returns formatted output only in verbose mode
func (vl *VerboseLogger) VerboseF(format string, args ...interface{}) string {
	if vl.verboseEnabled {
		return fmt.Sprintf(format, args...)
	}
	return ""
}

// PrintVerbose prints a message only in verbose mode
func (vl *VerboseLogger) PrintVerbose(message string) {
	if vl.verboseEnabled {
		fmt.Println(message)
	}
}

// PrintfVerbose prints a formatted message only in verbose mode
func (vl *VerboseLogger) PrintfVerbose(format string, args ...interface{}) {
	if vl.verboseEnabled {
		fmt.Printf(format, args...)
	}
}

// WithContext returns a formatted message with context information
func (vl *VerboseLogger) WithContext(operation string, details map[string]interface{}) {
	if vl.verboseEnabled {
		vl.Verbose("Operation: %s", operation)
		for key, value := range details {
			vl.Verbose("  %s: %v", key, value)
		}
	}
}

// LogCommand logs a command execution in verbose mode
func (vl *VerboseLogger) LogCommand(cmd string, args []string) {
	if vl.verboseEnabled {
		vl.Trace("Executing command: %s %v", cmd, args)
	}
}

// LogCommandResult logs a command result in verbose mode
func (vl *VerboseLogger) LogCommandResult(cmd string, exitCode int, output string) {
	if vl.verboseEnabled {
		vl.Trace("Command completed: %s (exit code: %d)", cmd, exitCode)
		if output != "" {
			vl.Trace("Output: %s", output)
		}
	}
}
