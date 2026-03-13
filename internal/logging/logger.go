package logging

import (
	"fmt"
	"io"
	"log"
	"os"
)

// Logger represents a simple logging system using Go's standard library log
type Logger struct {
	infoLogger  *log.Logger
	warnLogger  *log.Logger
	errorLogger *log.Logger
	debugLogger *log.Logger
}

// NewLogger creates a new Logger instance with output to stdout
func NewLogger() *Logger {
	return NewLoggerWithWriter(os.Stdout)
}

// NewLoggerWithWriter creates a new Logger instance with custom writer
func NewLoggerWithWriter(w io.Writer) *Logger {
	return &Logger{
		infoLogger:  log.New(w, "[INFO] ", log.LstdFlags),
		warnLogger:  log.New(w, "[WARN] ", log.LstdFlags),
		errorLogger: log.New(w, "[ERROR] ", log.LstdFlags),
		debugLogger: log.New(w, "[DEBUG] ", log.LstdFlags),
	}
}

// NewLoggerWithFile creates a new Logger instance with output to a file
func NewLoggerWithFile(filepath string) (*Logger, error) {
	file, err := os.OpenFile(filepath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	return &Logger{
		infoLogger:  log.New(file, "[INFO] ", log.LstdFlags),
		warnLogger:  log.New(file, "[WARN] ", log.LstdFlags),
		errorLogger: log.New(file, "[ERROR] ", log.LstdFlags),
		debugLogger: log.New(file, "[DEBUG] ", log.LstdFlags),
	}, nil
}

// Info logs an info level message
func (l *Logger) Info(format string, args ...interface{}) {
	l.infoLogger.Printf(format, args...)
}

// Warn logs a warning level message
func (l *Logger) Warn(format string, args ...interface{}) {
	l.warnLogger.Printf(format, args...)
}

// Error logs an error level message
func (l *Logger) Error(format string, args ...interface{}) {
	l.errorLogger.Printf(format, args...)
}

// Debug logs a debug level message
func (l *Logger) Debug(format string, args ...interface{}) {
	l.debugLogger.Printf(format, args...)
}
