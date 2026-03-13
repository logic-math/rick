package logging

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNewLogger(t *testing.T) {
	logger := NewLogger()
	if logger == nil {
		t.Fatal("NewLogger returned nil")
	}
	if logger.infoLogger == nil || logger.warnLogger == nil || logger.errorLogger == nil || logger.debugLogger == nil {
		t.Fatal("Logger not properly initialized")
	}
}

func TestNewLoggerWithWriter(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewLoggerWithWriter(buf)
	if logger == nil {
		t.Fatal("NewLoggerWithWriter returned nil")
	}

	logger.Info("test info")
	output := buf.String()
	if !strings.Contains(output, "[INFO]") {
		t.Fatalf("Expected [INFO] prefix, got: %s", output)
	}
	if !strings.Contains(output, "test info") {
		t.Fatalf("Expected 'test info' in output, got: %s", output)
	}
}

func TestInfoMethod(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewLoggerWithWriter(buf)

	logger.Info("test message")
	output := buf.String()

	if !strings.Contains(output, "[INFO]") {
		t.Fatalf("Expected [INFO] prefix, got: %s", output)
	}
	if !strings.Contains(output, "test message") {
		t.Fatalf("Expected 'test message', got: %s", output)
	}
}

func TestInfoMethodWithArgs(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewLoggerWithWriter(buf)

	logger.Info("test %s %d", "message", 42)
	output := buf.String()

	if !strings.Contains(output, "[INFO]") {
		t.Fatalf("Expected [INFO] prefix, got: %s", output)
	}
	if !strings.Contains(output, "test message 42") {
		t.Fatalf("Expected formatted message, got: %s", output)
	}
}

func TestWarnMethod(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewLoggerWithWriter(buf)

	logger.Warn("warning message")
	output := buf.String()

	if !strings.Contains(output, "[WARN]") {
		t.Fatalf("Expected [WARN] prefix, got: %s", output)
	}
	if !strings.Contains(output, "warning message") {
		t.Fatalf("Expected 'warning message', got: %s", output)
	}
}

func TestWarnMethodWithArgs(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewLoggerWithWriter(buf)

	logger.Warn("warning %s", "test")
	output := buf.String()

	if !strings.Contains(output, "[WARN]") {
		t.Fatalf("Expected [WARN] prefix, got: %s", output)
	}
	if !strings.Contains(output, "warning test") {
		t.Fatalf("Expected formatted message, got: %s", output)
	}
}

func TestErrorMethod(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewLoggerWithWriter(buf)

	logger.Error("error message")
	output := buf.String()

	if !strings.Contains(output, "[ERROR]") {
		t.Fatalf("Expected [ERROR] prefix, got: %s", output)
	}
	if !strings.Contains(output, "error message") {
		t.Fatalf("Expected 'error message', got: %s", output)
	}
}

func TestErrorMethodWithArgs(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewLoggerWithWriter(buf)

	logger.Error("error %s: %d", "code", 500)
	output := buf.String()

	if !strings.Contains(output, "[ERROR]") {
		t.Fatalf("Expected [ERROR] prefix, got: %s", output)
	}
	if !strings.Contains(output, "error code: 500") {
		t.Fatalf("Expected formatted message, got: %s", output)
	}
}

func TestDebugMethod(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewLoggerWithWriter(buf)

	logger.Debug("debug message")
	output := buf.String()

	if !strings.Contains(output, "[DEBUG]") {
		t.Fatalf("Expected [DEBUG] prefix, got: %s", output)
	}
	if !strings.Contains(output, "debug message") {
		t.Fatalf("Expected 'debug message', got: %s", output)
	}
}

func TestDebugMethodWithArgs(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewLoggerWithWriter(buf)

	logger.Debug("debug %v", map[string]string{"key": "value"})
	output := buf.String()

	if !strings.Contains(output, "[DEBUG]") {
		t.Fatalf("Expected [DEBUG] prefix, got: %s", output)
	}
	if !strings.Contains(output, "key") {
		t.Fatalf("Expected debug output, got: %s", output)
	}
}

func TestLogFormatIncludesTimestamp(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewLoggerWithWriter(buf)

	logger.Info("test")
	output := buf.String()

	// Check for timestamp format (HH:MM:SS)
	parts := strings.Fields(output)
	if len(parts) < 3 {
		t.Fatalf("Expected timestamp in output, got: %s", output)
	}
}

func TestNewLoggerWithFile(t *testing.T) {
	// Create temporary file
	tmpdir := t.TempDir()
	logfile := filepath.Join(tmpdir, "test.log")

	logger, err := NewLoggerWithFile(logfile)
	if err != nil {
		t.Fatalf("Failed to create logger with file: %v", err)
	}
	if logger == nil {
		t.Fatal("NewLoggerWithFile returned nil")
	}

	logger.Info("test info")
	logger.Warn("test warn")
	logger.Error("test error")
	logger.Debug("test debug")

	// Give it a moment to flush
	time.Sleep(100 * time.Millisecond)

	// Read file and verify content
	content, err := os.ReadFile(logfile)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	output := string(content)
	if !strings.Contains(output, "[INFO]") || !strings.Contains(output, "test info") {
		t.Fatalf("Expected INFO log in file, got: %s", output)
	}
	if !strings.Contains(output, "[WARN]") || !strings.Contains(output, "test warn") {
		t.Fatalf("Expected WARN log in file, got: %s", output)
	}
	if !strings.Contains(output, "[ERROR]") || !strings.Contains(output, "test error") {
		t.Fatalf("Expected ERROR log in file, got: %s", output)
	}
	if !strings.Contains(output, "[DEBUG]") || !strings.Contains(output, "test debug") {
		t.Fatalf("Expected DEBUG log in file, got: %s", output)
	}
}

func TestMultipleLogCalls(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewLoggerWithWriter(buf)

	logger.Info("first")
	logger.Warn("second")
	logger.Error("third")
	logger.Debug("fourth")

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")

	if len(lines) < 4 {
		t.Fatalf("Expected 4 log lines, got %d: %s", len(lines), output)
	}

	if !strings.Contains(lines[0], "[INFO]") || !strings.Contains(lines[0], "first") {
		t.Fatalf("First line should be INFO, got: %s", lines[0])
	}
	if !strings.Contains(lines[1], "[WARN]") || !strings.Contains(lines[1], "second") {
		t.Fatalf("Second line should be WARN, got: %s", lines[1])
	}
	if !strings.Contains(lines[2], "[ERROR]") || !strings.Contains(lines[2], "third") {
		t.Fatalf("Third line should be ERROR, got: %s", lines[2])
	}
	if !strings.Contains(lines[3], "[DEBUG]") || !strings.Contains(lines[3], "fourth") {
		t.Fatalf("Fourth line should be DEBUG, got: %s", lines[3])
	}
}

func TestPlainTextFormat(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewLoggerWithWriter(buf)

	logger.Info("test")
	output := buf.String()

	// Ensure no JSON markers
	if strings.Contains(output, "{") || strings.Contains(output, "}") || strings.Contains(output, "\"") {
		t.Fatalf("Expected plain text format, got: %s", output)
	}
}
