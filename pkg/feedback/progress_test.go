package feedback

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestNewProgressBar(t *testing.T) {
	pb := NewProgressBar("Test", 10)

	if pb == nil {
		t.Fatal("NewProgressBar returned nil")
	}
	if pb.title != "Test" {
		t.Errorf("expected title 'Test', got '%s'", pb.title)
	}
	if pb.total != 10 {
		t.Errorf("expected total 10, got %d", pb.total)
	}
	if pb.current != 0 {
		t.Errorf("expected current 0, got %d", pb.current)
	}
}

func TestNewProgressBarWithWriter(t *testing.T) {
	var buf bytes.Buffer
	pb := NewProgressBarWithWriter("Test", 10, &buf)

	if pb == nil {
		t.Fatal("NewProgressBarWithWriter returned nil")
	}
	if pb.writer != &buf {
		t.Error("expected writer to be set")
	}
}

func TestProgressBarUpdate(t *testing.T) {
	var buf bytes.Buffer
	pb := NewProgressBarWithWriter("Test", 10, &buf)

	pb.Update(5)
	if pb.current != 5 {
		t.Errorf("expected current to be 5, got %d", pb.current)
	}

	pb.Update(10)
	if pb.current != 10 {
		t.Errorf("expected current to be 10, got %d", pb.current)
	}
}

func TestProgressBarIncrement(t *testing.T) {
	var buf bytes.Buffer
	pb := NewProgressBarWithWriter("Test", 10, &buf)

	pb.Increment()
	if pb.current != 1 {
		t.Errorf("expected current to be 1, got %d", pb.current)
	}

	pb.Increment()
	if pb.current != 2 {
		t.Errorf("expected current to be 2, got %d", pb.current)
	}
}

func TestProgressBarComplete(t *testing.T) {
	var buf bytes.Buffer
	pb := NewProgressBarWithWriter("Test", 10, &buf)

	pb.Complete()
	if pb.current != 10 {
		t.Errorf("expected current to be 10 after complete, got %d", pb.current)
	}

	output := buf.String()
	if !strings.Contains(output, "Test") {
		t.Errorf("expected output to contain title, got '%s'", output)
	}
}

func TestFormatDuration(t *testing.T) {
	testCases := []struct {
		duration time.Duration
		check    func(string) bool
	}{
		{100 * time.Millisecond, func(s string) bool { return strings.Contains(s, "ms") }},
		{5 * time.Second, func(s string) bool { return strings.Contains(s, "s") }},
		{2 * time.Minute, func(s string) bool { return strings.Contains(s, "m") }},
	}

	for _, tc := range testCases {
		formatted := formatDuration(tc.duration)
		if !tc.check(formatted) {
			t.Errorf("unexpected format for %v: %s", tc.duration, formatted)
		}
	}
}

func TestNewStatusIndicator(t *testing.T) {
	si := NewStatusIndicator()

	if si == nil {
		t.Fatal("NewStatusIndicator returned nil")
	}
}

func TestNewStatusIndicatorWithWriter(t *testing.T) {
	var buf bytes.Buffer
	si := NewStatusIndicatorWithWriter(&buf)

	if si == nil {
		t.Fatal("NewStatusIndicatorWithWriter returned nil")
	}
	if si.writer != &buf {
		t.Error("expected writer to be set")
	}
}

func TestStatusIndicatorSuccess(t *testing.T) {
	var buf bytes.Buffer
	si := NewStatusIndicatorWithWriter(&buf)

	si.Success("Operation completed")
	output := buf.String()

	if !strings.Contains(output, "✅") {
		t.Errorf("expected output to contain success emoji, got '%s'", output)
	}
	if !strings.Contains(output, "Operation completed") {
		t.Errorf("expected output to contain message, got '%s'", output)
	}
}

func TestStatusIndicatorError(t *testing.T) {
	var buf bytes.Buffer
	si := NewStatusIndicatorWithWriter(&buf)

	si.Error("Operation failed")
	output := buf.String()

	if !strings.Contains(output, "❌") {
		t.Errorf("expected output to contain error emoji, got '%s'", output)
	}
	if !strings.Contains(output, "Operation failed") {
		t.Errorf("expected output to contain message, got '%s'", output)
	}
}

func TestStatusIndicatorWarning(t *testing.T) {
	var buf bytes.Buffer
	si := NewStatusIndicatorWithWriter(&buf)

	si.Warning("Warning message")
	output := buf.String()

	if !strings.Contains(output, "⚠️") {
		t.Errorf("expected output to contain warning emoji, got '%s'", output)
	}
}

func TestStatusIndicatorInfo(t *testing.T) {
	var buf bytes.Buffer
	si := NewStatusIndicatorWithWriter(&buf)

	si.Info("Info message")
	output := buf.String()

	if !strings.Contains(output, "ℹ️") {
		t.Errorf("expected output to contain info emoji, got '%s'", output)
	}
}

func TestStatusIndicatorDebug(t *testing.T) {
	var buf bytes.Buffer
	si := NewStatusIndicatorWithWriter(&buf)

	si.Debug("Debug message")
	output := buf.String()

	if !strings.Contains(output, "🐛") {
		t.Errorf("expected output to contain debug emoji, got '%s'", output)
	}
}

func TestStatusIndicatorRunning(t *testing.T) {
	var buf bytes.Buffer
	si := NewStatusIndicatorWithWriter(&buf)

	si.Running("Running task")
	output := buf.String()

	if !strings.Contains(output, "🔄") {
		t.Errorf("expected output to contain running emoji, got '%s'", output)
	}
}

func TestStatusIndicatorPending(t *testing.T) {
	var buf bytes.Buffer
	si := NewStatusIndicatorWithWriter(&buf)

	si.Pending("Pending task")
	output := buf.String()

	if !strings.Contains(output, "⏳") {
		t.Errorf("expected output to contain pending emoji, got '%s'", output)
	}
}

func TestStatusIndicatorTip(t *testing.T) {
	var buf bytes.Buffer
	si := NewStatusIndicatorWithWriter(&buf)

	si.Tip("Helpful tip")
	output := buf.String()

	if !strings.Contains(output, "💡") {
		t.Errorf("expected output to contain tip emoji, got '%s'", output)
	}
}

func TestNewSpinner(t *testing.T) {
	spinner := NewSpinner()

	if spinner == nil {
		t.Fatal("NewSpinner returned nil")
	}
	if len(spinner.frames) == 0 {
		t.Error("expected spinner to have frames")
	}
	if spinner.index != 0 {
		t.Errorf("expected initial index to be 0, got %d", spinner.index)
	}
}

func TestSpinnerNext(t *testing.T) {
	spinner := NewSpinner()
	initialFrames := len(spinner.frames)

	frame1 := spinner.Next()
	frame2 := spinner.Next()

	if frame1 == frame2 {
		t.Error("expected different frames")
	}

	// After cycling through all frames, should return to start
	for i := 0; i < initialFrames-2; i++ {
		spinner.Next()
	}

	frame := spinner.Next()
	if frame != frame1 {
		t.Errorf("expected to cycle back to first frame, got '%s'", frame)
	}
}

func TestSpinnerPrint(t *testing.T) {
	var buf bytes.Buffer
	spinner := NewSpinner()
	spinner.writer = &buf

	spinner.Print("Loading")
	output := buf.String()

	if !strings.Contains(output, "Loading") {
		t.Errorf("expected output to contain 'Loading', got '%s'", output)
	}
}

func TestProgressBarRender(t *testing.T) {
	var buf bytes.Buffer
	pb := NewProgressBarWithWriter("Progress", 100, &buf)

	pb.Update(50)
	output := buf.String()

	if !strings.Contains(output, "Progress") {
		t.Errorf("expected output to contain title, got '%s'", output)
	}
	if !strings.Contains(output, "50/100") {
		t.Errorf("expected output to contain progress, got '%s'", output)
	}
}

func TestProgressBarZeroTotal(t *testing.T) {
	var buf bytes.Buffer
	pb := NewProgressBarWithWriter("Progress", 0, &buf)

	pb.Update(0)
	output := buf.String()

	// Should not render if total is 0
	if output != "" {
		t.Errorf("expected no output for zero total, got '%s'", output)
	}
}

func TestStatusIndicatorMultipleMessages(t *testing.T) {
	var buf bytes.Buffer
	si := NewStatusIndicatorWithWriter(&buf)

	si.Success("First")
	si.Error("Second")
	si.Warning("Third")

	output := buf.String()
	if !strings.Contains(output, "First") || !strings.Contains(output, "Second") || !strings.Contains(output, "Third") {
		t.Errorf("expected all messages in output, got '%s'", output)
	}
}
