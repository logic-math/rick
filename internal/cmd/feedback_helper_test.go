package cmd

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/sunquan/rick/pkg/feedback"
)

func createTestCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use: "test",
	}
	cmd.Flags().BoolP("verbose", "v", false, "Enable verbose output")
	return cmd
}

func TestNewFeedbackContext(t *testing.T) {
	cmd := createTestCmd()
	fc := NewFeedbackContext(cmd)

	if fc == nil {
		t.Fatal("NewFeedbackContext returned nil")
	}
	if fc.I18n == nil {
		t.Error("expected I18n to be initialized")
	}
	if fc.ErrorHandler == nil {
		t.Error("expected ErrorHandler to be initialized")
	}
	if fc.Logger == nil {
		t.Error("expected Logger to be initialized")
	}
	if fc.StatusInd == nil {
		t.Error("expected StatusIndicator to be initialized")
	}
}

func TestNewFeedbackContextWithVerbose(t *testing.T) {
	cmd := createTestCmd()
	cmd.Flags().Set("verbose", "true")
	fc := NewFeedbackContext(cmd)

	if !fc.Logger.IsVerbose() {
		t.Error("expected verbose mode to be enabled")
	}
}

func TestSetLanguage(t *testing.T) {
	cmd := createTestCmd()
	fc := NewFeedbackContext(cmd)

	fc.SetLanguage(feedback.LangChinese)
	msg := fc.I18n.Get("ERR_INVALID_JOB_ID", "test")
	if !strings.Contains(msg, "test") {
		t.Errorf("expected Chinese message, got '%s'", msg)
	}
}

func TestSetVerbose(t *testing.T) {
	cmd := createTestCmd()
	fc := NewFeedbackContext(cmd)

	fc.SetVerbose(true)
	if !fc.Logger.IsVerbose() {
		t.Error("expected verbose to be enabled")
	}

	fc.SetVerbose(false)
	if fc.Logger.IsVerbose() {
		t.Error("expected verbose to be disabled")
	}
}

func TestHandleError(t *testing.T) {
	cmd := createTestCmd()
	fc := NewFeedbackContext(cmd)

	err := errors.New("test error")
	fc.HandleError("TestError", err)
	// Should not panic
}

func TestHandleErrorWithMessage(t *testing.T) {
	cmd := createTestCmd()
	fc := NewFeedbackContext(cmd)

	err := errors.New("original error")
	fc.HandleErrorWithMessage("TestError", "custom message", err)
	// Should not panic
}

func TestCreateProgressBar(t *testing.T) {
	cmd := createTestCmd()
	fc := NewFeedbackContext(cmd)

	pb := fc.CreateProgressBar("Test", 10)
	if pb == nil {
		t.Fatal("CreateProgressBar returned nil")
	}
	if fc.ProgressBar != pb {
		t.Error("expected ProgressBar to be stored in context")
	}
}

func TestGetLocalizedError(t *testing.T) {
	cmd := createTestCmd()
	fc := NewFeedbackContext(cmd)

	msg := fc.GetLocalizedError("ERR_INVALID_JOB_ID", "bad_id")
	if !strings.Contains(msg, "bad_id") {
		t.Errorf("expected localized error message, got '%s'", msg)
	}
}

func TestGetLocalizedSuggestion(t *testing.T) {
	cmd := createTestCmd()
	fc := NewFeedbackContext(cmd)

	msg := fc.GetLocalizedSuggestion("SUG_INVALID_JOB_ID")
	if !strings.Contains(msg, "💡") {
		t.Errorf("expected suggestion with emoji, got '%s'", msg)
	}
}

func TestPrintStepStart(t *testing.T) {
	cmd := createTestCmd()
	fc := NewFeedbackContext(cmd)

	fc.PrintStepStart("Starting task")
	// Should not panic
}

func TestPrintStepComplete(t *testing.T) {
	cmd := createTestCmd()
	fc := NewFeedbackContext(cmd)

	fc.PrintStepComplete("Task completed")
	// Should not panic
}

func TestPrintStepWarning(t *testing.T) {
	cmd := createTestCmd()
	fc := NewFeedbackContext(cmd)

	fc.PrintStepWarning("Task warning")
	// Should not panic
}

func TestPrintStepError(t *testing.T) {
	cmd := createTestCmd()
	fc := NewFeedbackContext(cmd)

	fc.PrintStepError("Task error")
	// Should not panic
}

func TestGetMsgWithContext(t *testing.T) {
	cmd := createTestCmd()
	cmd.Flags().Set("verbose", "true")
	fc := NewFeedbackContext(cmd)

	details := map[string]interface{}{
		"file": "config.json",
		"line": 42,
	}

	fc.GetMsgWithContext("LoadConfig", details)
	// Should not panic
}

func TestLogVerbose(t *testing.T) {
	cmd := createTestCmd()
	cmd.Flags().Set("verbose", "true")
	fc := NewFeedbackContext(cmd)

	fc.LogVerbose("Verbose message: %s", "test")
	// Should not panic
}

func TestLogDebug(t *testing.T) {
	cmd := createTestCmd()
	cmd.Flags().Set("verbose", "true")
	fc := NewFeedbackContext(cmd)

	fc.LogDebug("Debug message: %s", "test")
	// Should not panic
}

func TestLogTrace(t *testing.T) {
	cmd := createTestCmd()
	cmd.Flags().Set("verbose", "true")
	fc := NewFeedbackContext(cmd)

	fc.LogTrace("Trace message: %s", "test")
	// Should not panic
}

func TestLogCommand(t *testing.T) {
	cmd := createTestCmd()
	cmd.Flags().Set("verbose", "true")
	fc := NewFeedbackContext(cmd)

	fc.LogCommand("git", []string{"commit", "-m", "test"})
	// Should not panic
}

func TestLogCommandResult(t *testing.T) {
	cmd := createTestCmd()
	cmd.Flags().Set("verbose", "true")
	fc := NewFeedbackContext(cmd)

	fc.LogCommandResult("git", 0, "Success")
	// Should not panic
}

func TestIsVerboseMode(t *testing.T) {
	cmd := createTestCmd()
	fc := NewFeedbackContext(cmd)

	if fc.IsVerboseMode() {
		t.Error("expected verbose mode to be disabled by default")
	}

	cmd.Flags().Set("verbose", "true")
	fc2 := NewFeedbackContext(cmd)
	if !fc2.IsVerboseMode() {
		t.Error("expected verbose mode to be enabled")
	}
}

func TestPrintInfo(t *testing.T) {
	cmd := createTestCmd()
	fc := NewFeedbackContext(cmd)

	fc.PrintInfo("Info message")
	// Should not panic
}

func TestPrintWarning(t *testing.T) {
	cmd := createTestCmd()
	fc := NewFeedbackContext(cmd)

	fc.PrintWarning("Warning message")
	// Should not panic
}

func TestPrintDebug(t *testing.T) {
	cmd := createTestCmd()
	fc := NewFeedbackContext(cmd)

	fc.PrintDebug("Debug message")
	// Should not panic
}

func TestPrintPending(t *testing.T) {
	cmd := createTestCmd()
	fc := NewFeedbackContext(cmd)

	fc.PrintPending("Pending message")
	// Should not panic
}

func TestFormatErrorMessage(t *testing.T) {
	cmd := createTestCmd()
	fc := NewFeedbackContext(cmd)

	msg := fc.FormatErrorMessage("TestError", "Something went wrong")
	if !strings.Contains(msg, "TestError") {
		t.Errorf("expected error type in message, got '%s'", msg)
	}
	if !strings.Contains(msg, "Something went wrong") {
		t.Errorf("expected error message in result, got '%s'", msg)
	}
}

func TestFeedbackContextChaining(t *testing.T) {
	cmd := createTestCmd()
	cmd.Flags().Set("verbose", "true")
	fc := NewFeedbackContext(cmd)

	// Test multiple operations
	fc.SetVerbose(true)
	fc.PrintStepStart("Starting")
	fc.LogVerbose("Processing: %s", "data")
	fc.PrintStepComplete("Done")

	if !fc.IsVerboseMode() {
		t.Error("expected verbose mode to remain enabled")
	}
}

func TestFeedbackContextWithDifferentLanguages(t *testing.T) {
	cmd := createTestCmd()
	fc := NewFeedbackContext(cmd)

	// Test English
	engMsg := fc.GetLocalizedError("ERR_CONFIG_NOT_FOUND", "config.json")
	if !strings.Contains(engMsg, "config.json") {
		t.Errorf("expected English error message, got '%s'", engMsg)
	}

	// Switch to Chinese
	fc.SetLanguage(feedback.LangChinese)
	zhMsg := fc.GetLocalizedError("ERR_CONFIG_NOT_FOUND", "config.json")
	if !strings.Contains(zhMsg, "config.json") {
		t.Errorf("expected Chinese error message, got '%s'", zhMsg)
	}
}

func TestFeedbackContextErrorWithContext(t *testing.T) {
	cmd := createTestCmd()
	cmd.Flags().Set("verbose", "true")
	fc := NewFeedbackContext(cmd)

	err := errors.New("test error")
	ewc := fc.ErrorHandler.Handle("TestError", err)
	ewc.AddContext("file", "config.json")
	ewc.AddContext("line", 42)

	formatted := ewc.Format(fc.I18n, fc.IsVerboseMode())
	if !strings.Contains(formatted, "TestError") {
		t.Errorf("expected error type in formatted message, got '%s'", formatted)
	}
}

func TestFeedbackContextProgressBar(t *testing.T) {
	cmd := createTestCmd()
	fc := NewFeedbackContext(cmd)

	pb := fc.CreateProgressBar("Processing", 100)
	pb.Update(50)
	pb.Increment()

	// Just verify that the progress bar was created and updated
	if pb == nil {
		t.Error("expected progress bar to be created")
	}
	if fc.ProgressBar != pb {
		t.Error("expected progress bar to be stored in context")
	}
}

func TestFeedbackContextMultipleErrors(t *testing.T) {
	cmd := createTestCmd()
	fc := NewFeedbackContext(cmd)

	err1 := errors.New("error 1")
	err2 := errors.New("error 2")

	fc.HandleError("Error1", err1)
	fc.HandleError("Error2", err2)
	// Should handle multiple errors without issues
}

func TestFeedbackContextWithCustomWriter(t *testing.T) {
	cmd := createTestCmd()
	fc := NewFeedbackContext(cmd)

	var buf bytes.Buffer
	fc.Logger = feedback.NewVerboseLoggerWithWriter(false, &buf)

	fc.PrintInfo("Test message")
	// Should not panic
}
