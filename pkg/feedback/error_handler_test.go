package feedback

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

func TestNewErrorHandler(t *testing.T) {
	i18n := DefaultI18nMessages(LangEnglish)
	eh := NewErrorHandler(i18n)

	if eh == nil {
		t.Fatal("NewErrorHandler returned nil")
	}
	if eh.includeStackTrace {
		t.Error("expected includeStackTrace to be false by default")
	}
	if eh.maxStackDepth != 10 {
		t.Errorf("expected maxStackDepth to be 10, got %d", eh.maxStackDepth)
	}
}

func TestSetIncludeStackTrace(t *testing.T) {
	i18n := DefaultI18nMessages(LangEnglish)
	eh := NewErrorHandler(i18n)

	eh.SetIncludeStackTrace(true)
	if !eh.includeStackTrace {
		t.Error("expected includeStackTrace to be true")
	}

	eh.SetIncludeStackTrace(false)
	if eh.includeStackTrace {
		t.Error("expected includeStackTrace to be false")
	}
}

func TestSetMaxStackDepth(t *testing.T) {
	i18n := DefaultI18nMessages(LangEnglish)
	eh := NewErrorHandler(i18n)

	eh.SetMaxStackDepth(5)
	if eh.maxStackDepth != 5 {
		t.Errorf("expected maxStackDepth to be 5, got %d", eh.maxStackDepth)
	}
}

func TestHandle(t *testing.T) {
	i18n := DefaultI18nMessages(LangEnglish)
	eh := NewErrorHandler(i18n)

	err := errors.New("test error")
	ewc := eh.Handle("TestError", err)

	if ewc == nil {
		t.Fatal("Handle returned nil")
	}
	if ewc.Message != "test error" {
		t.Errorf("expected message 'test error', got '%s'", ewc.Message)
	}
	if ewc.ErrorType != "TestError" {
		t.Errorf("expected error type 'TestError', got '%s'", ewc.ErrorType)
	}
	if ewc.Cause != err {
		t.Error("expected Cause to be the original error")
	}
}

func TestHandleWithMessage(t *testing.T) {
	i18n := DefaultI18nMessages(LangEnglish)
	eh := NewErrorHandler(i18n)

	err := errors.New("original error")
	ewc := eh.HandleWithMessage("CustomError", "custom message", err)

	if ewc == nil {
		t.Fatal("HandleWithMessage returned nil")
	}
	if ewc.Message != "custom message" {
		t.Errorf("expected message 'custom message', got '%s'", ewc.Message)
	}
	if ewc.ErrorType != "CustomError" {
		t.Errorf("expected error type 'CustomError', got '%s'", ewc.ErrorType)
	}
}

func TestAddContext(t *testing.T) {
	i18n := DefaultI18nMessages(LangEnglish)
	eh := NewErrorHandler(i18n)

	err := errors.New("test error")
	ewc := eh.Handle("TestError", err)

	ewc.AddContext("file", "config.json")
	ewc.AddContext("line", 42)

	if ewc.Context["file"] != "config.json" {
		t.Errorf("expected context file to be 'config.json', got '%v'", ewc.Context["file"])
	}
	if ewc.Context["line"] != 42 {
		t.Errorf("expected context line to be 42, got '%v'", ewc.Context["line"])
	}
}

func TestFormat(t *testing.T) {
	i18n := DefaultI18nMessages(LangEnglish)
	eh := NewErrorHandler(i18n)

	err := errors.New("test error")
	ewc := eh.Handle("TestError", err)
	ewc.AddContext("file", "config.json")

	formatted := ewc.Format(i18n, false)
	if !strings.Contains(formatted, "TestError") {
		t.Errorf("expected formatted error to contain 'TestError', got '%s'", formatted)
	}
	if !strings.Contains(formatted, "test error") {
		t.Errorf("expected formatted error to contain 'test error', got '%s'", formatted)
	}
}

func TestFormatWithVerbose(t *testing.T) {
	i18n := DefaultI18nMessages(LangEnglish)
	eh := NewErrorHandler(i18n)

	err := errors.New("test error")
	ewc := eh.Handle("TestError", err)
	ewc.AddContext("file", "config.json")

	formatted := ewc.Format(i18n, true)
	if !strings.Contains(formatted, "Context:") {
		t.Errorf("expected formatted error to contain 'Context:' in verbose mode, got '%s'", formatted)
	}
	if !strings.Contains(formatted, "config.json") {
		t.Errorf("expected formatted error to contain context in verbose mode, got '%s'", formatted)
	}
}

func TestFormatWithStackTrace(t *testing.T) {
	i18n := DefaultI18nMessages(LangEnglish)
	eh := NewErrorHandler(i18n)
	eh.SetIncludeStackTrace(true)

	err := errors.New("test error")
	ewc := eh.Handle("TestError", err)

	formatted := ewc.Format(i18n, false)
	if len(ewc.StackTrace) == 0 {
		t.Error("expected stack trace to be captured")
	}
	if !strings.Contains(formatted, "Stack Trace:") {
		t.Errorf("expected formatted error to contain 'Stack Trace:', got '%s'", formatted)
	}
}

func TestGetSuggestion(t *testing.T) {
	i18n := DefaultI18nMessages(LangEnglish)
	eh := NewErrorHandler(i18n)

	err := errors.New("invalid job ID")
	ewc := eh.Handle("INVALID_JOB_ID", err)

	suggestion := ewc.GetSuggestion(i18n)
	if suggestion == "" {
		t.Error("expected suggestion to be non-empty")
	}
	if !strings.Contains(suggestion, "💡") {
		t.Errorf("expected suggestion to contain emoji, got '%s'", suggestion)
	}
}

func TestErrorInterface(t *testing.T) {
	i18n := DefaultI18nMessages(LangEnglish)
	eh := NewErrorHandler(i18n)

	err := errors.New("test error")
	ewc := eh.Handle("TestError", err)

	// Should implement error interface
	var _ error = ewc

	errStr := ewc.Error()
	if !strings.Contains(errStr, "TestError") {
		t.Errorf("expected error string to contain 'TestError', got '%s'", errStr)
	}
}

func TestCaptureStackTrace(t *testing.T) {
	frames := captureStackTrace(5)

	if len(frames) == 0 {
		t.Error("expected at least one stack frame")
	}

	// Check that frames have expected fields
	for _, frame := range frames {
		if frame.File == "" {
			t.Error("expected frame to have a file")
		}
		if frame.Function == "" {
			t.Error("expected frame to have a function")
		}
		if frame.Line == 0 {
			t.Error("expected frame to have a line number")
		}
	}
}

func TestErrorHandlerMultipleErrors(t *testing.T) {
	i18n := DefaultI18nMessages(LangEnglish)
	eh := NewErrorHandler(i18n)

	err1 := errors.New("error 1")
	err2 := errors.New("error 2")

	ewc1 := eh.Handle("Error1", err1)
	ewc2 := eh.Handle("Error2", err2)

	if ewc1.Message == ewc2.Message {
		t.Error("expected different error messages")
	}
	if ewc1.ErrorType == ewc2.ErrorType {
		t.Error("expected different error types")
	}
}

func TestFormatWithoutContext(t *testing.T) {
	i18n := DefaultI18nMessages(LangEnglish)
	eh := NewErrorHandler(i18n)

	err := errors.New("test error")
	ewc := eh.Handle("TestError", err)

	formatted := ewc.Format(i18n, false)
	if strings.Contains(formatted, "Context:") {
		t.Error("expected no Context in non-verbose mode")
	}
}

func TestErrorHandlerWithWriter(t *testing.T) {
	i18n := DefaultI18nMessages(LangEnglish)
	eh := NewErrorHandler(i18n)

	err := errors.New("test error")
	ewc := eh.Handle("TestError", err)

	var buf bytes.Buffer
	formatted := ewc.Format(i18n, true)
	buf.WriteString(formatted)

	if buf.Len() == 0 {
		t.Error("expected formatted error to be written")
	}
}
