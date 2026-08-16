package cmd

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunquan/rick/internal/handler"
)

// TestDryRun_PlanPrintsPrompt verifies that rick plan --dry-run prints the full plan prompt.
func TestDryRun_PlanPrintsPrompt(t *testing.T) {
	// Set up a temp project root with .rick directory
	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	tmpDir := t.TempDir()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(orig)

	if err := os.MkdirAll(filepath.Join(tmpDir, ".rick"), 0755); err != nil {
		t.Fatal(err)
	}

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	origDryRun := dryRun
	dryRun = true
	defer func() {
		dryRun = origDryRun
		os.Stdout = old
	}()

	err = handler.PlanDryRun()
	w.Close()

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()
	os.Stdout = old

	if err != nil {
		t.Fatalf("handler.PlanDryRun() returned error: %v", err)
	}

	// Must contain the dry-run header
	if !strings.Contains(output, "[DRY-RUN]") {
		t.Error("plan dry-run output must contain '[DRY-RUN]'")
	}

	// Must contain actual prompt content (not just a one-liner)
	if !strings.Contains(output, "Plan 阶段") && !strings.Contains(output, "plan") {
		t.Error("plan dry-run output must contain prompt content (plan template text)")
	}

	// Must not be a trivial single-line output
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) < 5 {
		t.Errorf("plan dry-run output too short (%d lines), expected full prompt", len(lines))
	}
}

// TestDryRun_LearningPrintsPrompt verifies that rick learning --dry-run prints the full learning prompt.
func TestDryRun_LearningPrintsPrompt(t *testing.T) {
	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	tmpDir := t.TempDir()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(orig)

	if err := os.MkdirAll(filepath.Join(tmpDir, ".rick"), 0755); err != nil {
		t.Fatal(err)
	}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err = handler.LearningDryRun("job_test")
	w.Close()

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()
	os.Stdout = old

	if err != nil {
		t.Fatalf("handler.LearningDryRun() returned error: %v", err)
	}

	if !strings.Contains(output, "[DRY-RUN]") {
		t.Error("learning dry-run output must contain '[DRY-RUN]'")
	}

	// Must contain actual prompt content
	if !strings.Contains(output, "learning") && !strings.Contains(output, "学习") {
		t.Error("learning dry-run output must contain prompt content (learning template text)")
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) < 5 {
		t.Errorf("learning dry-run output too short (%d lines), expected full prompt", len(lines))
	}
}

// TestDryRun_PlanPromptContainsJobPlanDir verifies plan dry-run includes job_plan_dir.
func TestDryRun_PlanPromptContainsJobPlanDir(t *testing.T) {
	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	tmpDir := t.TempDir()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(orig)

	if err := os.MkdirAll(filepath.Join(tmpDir, ".rick"), 0755); err != nil {
		t.Fatal(err)
	}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	origDryRun := dryRun
	dryRun = true
	defer func() { dryRun = origDryRun }()

	err = handler.PlanDryRun()
	w.Close()

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()
	os.Stdout = old

	if err != nil {
		t.Fatalf("handler.PlanDryRun() returned error: %v", err)
	}

	// The plan prompt template uses {{job_plan_dir}} which should be replaced
	if strings.Contains(output, "{{job_plan_dir}}") {
		t.Error("plan dry-run output must not contain unreplaced {{job_plan_dir}} variable")
	}

	// Should contain loops_context section (from plan template)
	if !strings.Contains(output, "loops") {
		t.Errorf("plan dry-run output must contain loops reference; output snippet:\n%s", truncate(output, 500))
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return fmt.Sprintf("%s...(truncated)", s[:n])
}
