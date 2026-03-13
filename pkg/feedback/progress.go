package feedback

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

// ProgressBar represents a simple progress bar
type ProgressBar struct {
	current   int
	total     int
	title     string
	writer    io.Writer
	startTime time.Time
}

// NewProgressBar creates a new progress bar
func NewProgressBar(title string, total int) *ProgressBar {
	return &ProgressBar{
		current:   0,
		total:     total,
		title:     title,
		writer:    os.Stdout,
		startTime: time.Now(),
	}
}

// NewProgressBarWithWriter creates a new progress bar with custom writer
func NewProgressBarWithWriter(title string, total int, writer io.Writer) *ProgressBar {
	return &ProgressBar{
		current:   0,
		total:     total,
		title:     title,
		writer:    writer,
		startTime: time.Now(),
	}
}

// Update updates the progress bar
func (pb *ProgressBar) Update(current int) {
	pb.current = current
	pb.render()
}

// Increment increments the progress by 1
func (pb *ProgressBar) Increment() {
	pb.current++
	pb.render()
}

// Complete marks the progress bar as complete
func (pb *ProgressBar) Complete() {
	pb.current = pb.total
	pb.render()
	fmt.Fprintf(pb.writer, "\n")
}

// render renders the progress bar
func (pb *ProgressBar) render() {
	if pb.total == 0 {
		return
	}

	percentage := (pb.current * 100) / pb.total
	barLength := 30
	filledLength := (pb.current * barLength) / pb.total

	bar := strings.Repeat("█", filledLength) + strings.Repeat("░", barLength-filledLength)
	elapsed := time.Since(pb.startTime)

	fmt.Fprintf(pb.writer, "\r%s [%s] %d/%d (%d%%) - %s",
		pb.title, bar, pb.current, pb.total, percentage, formatDuration(elapsed))
}

// formatDuration formats duration in a human-readable way
func formatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	minutes := int(d.Minutes())
	seconds := int(d.Seconds()) % 60
	return fmt.Sprintf("%dm%ds", minutes, seconds)
}

// StatusIndicator represents a status indicator
type StatusIndicator struct {
	writer io.Writer
}

// NewStatusIndicator creates a new status indicator
func NewStatusIndicator() *StatusIndicator {
	return &StatusIndicator{
		writer: os.Stdout,
	}
}

// NewStatusIndicatorWithWriter creates a new status indicator with custom writer
func NewStatusIndicatorWithWriter(writer io.Writer) *StatusIndicator {
	return &StatusIndicator{
		writer: writer,
	}
}

// Success prints a success message
func (si *StatusIndicator) Success(message string) {
	fmt.Fprintf(si.writer, "✅ %s\n", message)
}

// Error prints an error message
func (si *StatusIndicator) Error(message string) {
	fmt.Fprintf(si.writer, "❌ %s\n", message)
}

// Warning prints a warning message
func (si *StatusIndicator) Warning(message string) {
	fmt.Fprintf(si.writer, "⚠️  %s\n", message)
}

// Info prints an info message
func (si *StatusIndicator) Info(message string) {
	fmt.Fprintf(si.writer, "ℹ️  %s\n", message)
}

// Debug prints a debug message
func (si *StatusIndicator) Debug(message string) {
	fmt.Fprintf(si.writer, "🐛 %s\n", message)
}

// Running prints a running message
func (si *StatusIndicator) Running(message string) {
	fmt.Fprintf(si.writer, "🔄 %s\n", message)
}

// Pending prints a pending message
func (si *StatusIndicator) Pending(message string) {
	fmt.Fprintf(si.writer, "⏳ %s\n", message)
}

// Tip prints a tip message
func (si *StatusIndicator) Tip(message string) {
	fmt.Fprintf(si.writer, "💡 %s\n", message)
}

// Spinner represents a simple spinner
type Spinner struct {
	frames []string
	index  int
	writer io.Writer
}

// NewSpinner creates a new spinner
func NewSpinner() *Spinner {
	return &Spinner{
		frames: []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
		index:  0,
		writer: os.Stdout,
	}
}

// Next returns the next frame
func (s *Spinner) Next() string {
	frame := s.frames[s.index]
	s.index = (s.index + 1) % len(s.frames)
	return frame
}

// Print prints the spinner with a message
func (s *Spinner) Print(message string) {
	fmt.Fprintf(s.writer, "\r%s %s", s.Next(), message)
}
