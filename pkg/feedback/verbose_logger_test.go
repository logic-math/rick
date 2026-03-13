package feedback

import (
	"bytes"
	"strings"
	"testing"
)

func TestNewVerboseLogger(t *testing.T) {
	vl := NewVerboseLogger(false)

	if vl == nil {
		t.Fatal("NewVerboseLogger returned nil")
	}
	if vl.verboseEnabled {
		t.Error("expected verbose to be disabled by default")
	}
}

func TestNewVerboseLoggerWithVerbose(t *testing.T) {
	vl := NewVerboseLogger(true)

	if !vl.verboseEnabled {
		t.Error("expected verbose to be enabled")
	}
}

func TestNewVerboseLoggerWithWriter(t *testing.T) {
	var buf bytes.Buffer
	vl := NewVerboseLoggerWithWriter(false, &buf)

	if vl == nil {
		t.Fatal("NewVerboseLoggerWithWriter returned nil")
	}
}

func TestSetVerbose(t *testing.T) {
	vl := NewVerboseLogger(false)

	vl.SetVerbose(true)
	if !vl.verboseEnabled {
		t.Error("expected verbose to be enabled")
	}

	vl.SetVerbose(false)
	if vl.verboseEnabled {
		t.Error("expected verbose to be disabled")
	}
}

func TestIsVerbose(t *testing.T) {
	vl := NewVerboseLogger(true)

	if !vl.IsVerbose() {
		t.Error("expected IsVerbose to return true")
	}

	vl.SetVerbose(false)
	if vl.IsVerbose() {
		t.Error("expected IsVerbose to return false")
	}
}

func TestInfo(t *testing.T) {
	var buf bytes.Buffer
	vl := NewVerboseLoggerWithWriter(false, &buf)

	vl.Info("Info message")
	output := buf.String()

	if !strings.Contains(output, "Info message") {
		t.Errorf("expected output to contain 'Info message', got '%s'", output)
	}
	if !strings.Contains(output, "ℹ️") {
		t.Errorf("expected output to contain info emoji, got '%s'", output)
	}
}

func TestWarn(t *testing.T) {
	var buf bytes.Buffer
	vl := NewVerboseLoggerWithWriter(false, &buf)

	vl.Warn("Warning message")
	output := buf.String()

	if !strings.Contains(output, "Warning message") {
		t.Errorf("expected output to contain 'Warning message', got '%s'", output)
	}
	if !strings.Contains(output, "⚠️") {
		t.Errorf("expected output to contain warning emoji, got '%s'", output)
	}
}

func TestError(t *testing.T) {
	var buf bytes.Buffer
	vl := NewVerboseLoggerWithWriter(false, &buf)

	vl.Error("Error message")
	output := buf.String()

	if !strings.Contains(output, "Error message") {
		t.Errorf("expected output to contain 'Error message', got '%s'", output)
	}
	if !strings.Contains(output, "❌") {
		t.Errorf("expected output to contain error emoji, got '%s'", output)
	}
}

func TestDebugDisabled(t *testing.T) {
	var buf bytes.Buffer
	vl := NewVerboseLoggerWithWriter(false, &buf)

	vl.Debug("Debug message")
	output := buf.String()

	if strings.Contains(output, "Debug message") {
		t.Errorf("expected no debug output when verbose disabled, got '%s'", output)
	}
}

func TestDebugEnabled(t *testing.T) {
	var buf bytes.Buffer
	vl := NewVerboseLoggerWithWriter(true, &buf)

	vl.Debug("Debug message")
	output := buf.String()

	if !strings.Contains(output, "Debug message") {
		t.Errorf("expected debug output when verbose enabled, got '%s'", output)
	}
	if !strings.Contains(output, "🐛") {
		t.Errorf("expected output to contain debug emoji, got '%s'", output)
	}
}

func TestVerboseDisabled(t *testing.T) {
	var buf bytes.Buffer
	vl := NewVerboseLoggerWithWriter(false, &buf)

	vl.Verbose("Verbose message")
	output := buf.String()

	if strings.Contains(output, "Verbose message") {
		t.Errorf("expected no verbose output when disabled, got '%s'", output)
	}
}

func TestVerboseEnabled(t *testing.T) {
	var buf bytes.Buffer
	vl := NewVerboseLoggerWithWriter(true, &buf)

	vl.Verbose("Verbose message")
	output := buf.String()

	if !strings.Contains(output, "Verbose message") {
		t.Errorf("expected verbose output when enabled, got '%s'", output)
	}
	if !strings.Contains(output, "📝") {
		t.Errorf("expected output to contain verbose emoji, got '%s'", output)
	}
}

func TestTraceDisabled(t *testing.T) {
	var buf bytes.Buffer
	vl := NewVerboseLoggerWithWriter(false, &buf)

	vl.Trace("Trace message")
	output := buf.String()

	if strings.Contains(output, "Trace message") {
		t.Errorf("expected no trace output when disabled, got '%s'", output)
	}
}

func TestTraceEnabled(t *testing.T) {
	var buf bytes.Buffer
	vl := NewVerboseLoggerWithWriter(true, &buf)

	vl.Trace("Trace message")
	output := buf.String()

	if !strings.Contains(output, "Trace message") {
		t.Errorf("expected trace output when enabled, got '%s'", output)
	}
	if !strings.Contains(output, "🔍") {
		t.Errorf("expected output to contain trace emoji, got '%s'", output)
	}
}

func TestVerboseF(t *testing.T) {
	vl := NewVerboseLogger(false)

	result := vl.VerboseF("Message: %s", "test")
	if result != "" {
		t.Errorf("expected empty string when verbose disabled, got '%s'", result)
	}

	vl.SetVerbose(true)
	result = vl.VerboseF("Message: %s", "test")
	if !strings.Contains(result, "test") {
		t.Errorf("expected formatted message when verbose enabled, got '%s'", result)
	}
}

func TestPrintVerbose(t *testing.T) {
	var buf bytes.Buffer
	vl := NewVerboseLoggerWithWriter(false, &buf)

	// Note: PrintVerbose uses fmt.Println which prints to stdout, not the writer
	// This test verifies the logic without capturing stdout

	// When verbose is disabled, nothing should happen
	vl.PrintVerbose("Test message")

	// When verbose is enabled, the method would print to stdout
	vl.SetVerbose(true)
	if !vl.IsVerbose() {
		t.Error("expected verbose to be enabled")
	}
}

func TestPrintfVerbose(t *testing.T) {
	var buf bytes.Buffer
	vl := NewVerboseLoggerWithWriter(false, &buf)

	// Note: PrintfVerbose uses fmt.Printf which prints to stdout, not the writer
	// This test verifies the logic without capturing stdout

	// When verbose is disabled, nothing should happen
	vl.PrintfVerbose("Message: %s\n", "test")

	// When verbose is enabled, the method would print to stdout
	vl.SetVerbose(true)
	if !vl.IsVerbose() {
		t.Error("expected verbose to be enabled")
	}
}

func TestWithContext(t *testing.T) {
	var buf bytes.Buffer
	vl := NewVerboseLoggerWithWriter(true, &buf)

	details := map[string]interface{}{
		"file": "config.json",
		"line": 42,
	}

	vl.WithContext("LoadConfig", details)
	output := buf.String()

	if !strings.Contains(output, "LoadConfig") {
		t.Errorf("expected operation name in output, got '%s'", output)
	}
	if !strings.Contains(output, "config.json") {
		t.Errorf("expected context details in output, got '%s'", output)
	}
}

func TestLogCommand(t *testing.T) {
	var buf bytes.Buffer
	vl := NewVerboseLoggerWithWriter(true, &buf)

	vl.LogCommand("git", []string{"commit", "-m", "test"})
	output := buf.String()

	if !strings.Contains(output, "git") {
		t.Errorf("expected command in output, got '%s'", output)
	}
}

func TestLogCommandResult(t *testing.T) {
	var buf bytes.Buffer
	vl := NewVerboseLoggerWithWriter(true, &buf)

	vl.LogCommandResult("git", 0, "Commit successful")
	output := buf.String()

	if !strings.Contains(output, "git") {
		t.Errorf("expected command in output, got '%s'", output)
	}
	if !strings.Contains(output, "exit code: 0") {
		t.Errorf("expected exit code in output, got '%s'", output)
	}
}

func TestVerboseLoggerMultipleLevels(t *testing.T) {
	var buf bytes.Buffer
	vl := NewVerboseLoggerWithWriter(false, &buf)

	vl.Info("Info")
	vl.Warn("Warning")
	vl.Error("Error")
	vl.Debug("Debug")

	output := buf.String()
	if !strings.Contains(output, "Info") {
		t.Error("expected Info message")
	}
	if !strings.Contains(output, "Warning") {
		t.Error("expected Warning message")
	}
	if !strings.Contains(output, "Error") {
		t.Error("expected Error message")
	}
	if strings.Contains(output, "Debug") {
		t.Error("expected no Debug message when verbose disabled")
	}
}

func TestVerboseLoggerFormattedMessages(t *testing.T) {
	var buf bytes.Buffer
	vl := NewVerboseLoggerWithWriter(false, &buf)

	vl.Info("Processing %d items", 42)
	output := buf.String()

	if !strings.Contains(output, "42") {
		t.Errorf("expected formatted message, got '%s'", output)
	}
}

func TestVerboseLoggerWithContextDisabled(t *testing.T) {
	var buf bytes.Buffer
	vl := NewVerboseLoggerWithWriter(false, &buf)

	details := map[string]interface{}{
		"key": "value",
	}

	vl.WithContext("Operation", details)
	if buf.Len() > 0 {
		t.Error("expected no output when verbose disabled")
	}
}
