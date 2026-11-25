package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunquan/rick/internal/config"
	"github.com/sunquan/rick/internal/runtime"
)

// TestHumanLoopCreatesDraftDirs tests that running human-loop creates draft directories
func TestHumanLoopCreatesDraftDirs(t *testing.T) {
	tmpDir := t.TempDir()
	mockScript := "#!/bin/sh\nexit 0\n"
	mockPath := filepath.Join(tmpDir, "mock_pi")
	if err := os.WriteFile(mockPath, []byte(mockScript), 0755); err != nil {
		t.Fatal(err)
	}

	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	workDir := t.TempDir()
	if err := os.Chdir(workDir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(orig) }()

	// Set HOME so config.LoadConfig reads from workDir
	origHome := os.Getenv("HOME")
	t.Setenv("HOME", workDir)
	defer os.Setenv("HOME", origHome)

	rickDir := filepath.Join(workDir, ".rick")
	if err := os.MkdirAll(rickDir, 0755); err != nil {
		t.Fatal(err)
	}
	cfgContent := `{"pi_path": "` + mockPath + `", "max_retries": 1}`
	if err := os.WriteFile(filepath.Join(rickDir, "config.json"), []byte(cfgContent), 0644); err != nil {
		t.Fatal(err)
	}

	origDryRun := dryRun
	defer func() { dryRun = origDryRun }()
	dryRun = false

	cmd := NewHumanLoopCmd()
	cmd.SetArgs([]string{"测试主题"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("human-loop failed: %v", err)
	}

	for _, sub := range []string{
		"draft",
		filepath.Join("draft", "loops"),
	} {
		p := filepath.Join(workDir, ".rick", sub)
		if _, err := os.Stat(p); os.IsNotExist(err) {
			t.Errorf("expected directory %s to exist", p)
		}
	}

	// v8 产物结构：本次 loop_N 目录下预建 orig/ 中间结构目录 + 初始化 thought_path.md。
	loopsRoot := filepath.Join(workDir, ".rick", "draft", "loops")
	entries, err := os.ReadDir(loopsRoot)
	if err != nil || len(entries) == 0 {
		t.Fatalf("expected at least one loop_N dir under %s (err=%v)", loopsRoot, err)
	}
	loopDir := filepath.Join(loopsRoot, entries[0].Name())
	if _, err := os.Stat(filepath.Join(loopDir, "orig")); os.IsNotExist(err) {
		t.Errorf("expected orig dir under loop dir %s", loopDir)
	}
	if _, err := os.Stat(filepath.Join(loopDir, "thought_path.md")); os.IsNotExist(err) {
		t.Errorf("expected thought_path.md under loop dir %s", loopDir)
	}

	// v7 遗留目录不再创建（v8 极简：仅 draft/loops 与 loop 产物目录）。
	for _, gone := range []string{
		filepath.Join("draft", "rfc"),
		filepath.Join("draft", "concepts"),
		filepath.Join("draft", "human-learning"),
	} {
		p := filepath.Join(workDir, ".rick", gone)
		if _, err := os.Stat(p); !os.IsNotExist(err) {
			t.Errorf("v7 legacy dir should no longer be created: %s", p)
		}
	}
}

// TestHumanLoopDryRunContainsDraftDir tests dry-run output has draft path, no unreplaced {{draft_dir}}
func TestHumanLoopDryRunContainsDraftDir(t *testing.T) {
	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	workDir := t.TempDir()
	if err := os.Chdir(workDir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(orig) }()

	if err := os.MkdirAll(filepath.Join(workDir, ".rick"), 0755); err != nil {
		t.Fatal(err)
	}

	origDryRun := dryRun
	defer func() { dryRun = origDryRun }()
	dryRun = true

	// Capture stdout: handler.HumanLoopDryRun prints via fmt.Print (os.Stdout),
	// not the cobra command's output writer.
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	cmd := NewHumanLoopCmd()
	cmd.SetArgs([]string{"测试主题"})
	err = cmd.Execute()
	w.Close()
	var buf bytes.Buffer
	buf.ReadFrom(r)
	os.Stdout = old

	if err != nil {
		t.Fatalf("human-loop dry-run failed: %v", err)
	}

	output := buf.String()
	if strings.Contains(output, "{{draft_dir}}") {
		t.Error("dry-run output contains unreplaced {{draft_dir}}")
	}
	if !strings.Contains(output, "draft") {
		t.Error("dry-run output does not contain 'draft'")
	}
	// v8 核心机制锚词：思考门禁 + orig 启发 + thought_path 唯一产出。
	for _, anchor := range []string{"门禁", "orig", "thought_path"} {
		if !strings.Contains(output, anchor) {
			t.Errorf("dry-run output missing v8 anchor %q", anchor)
		}
	}
	// v7 教学机制不得回流。
	for _, gone := range []string{"ZPD", "费曼", "教学", "价值裁决", "mission"} {
		if strings.Contains(output, gone) {
			t.Errorf("dry-run output should not contain v7 teaching keyword %q", gone)
		}
	}
}

// TestHumanLoopCmdCreation tests that NewHumanLoopCmd creates a valid command
func TestHumanLoopCmdCreation(t *testing.T) {
	cmd := NewHumanLoopCmd()
	if cmd == nil {
		t.Fatal("NewHumanLoopCmd returned nil")
	}
	if cmd.Use != "human-loop [topic]" {
		t.Errorf("expected Use to be 'human-loop [topic]', got %s", cmd.Use)
	}
	if cmd.Short == "" {
		t.Error("expected Short to be non-empty")
	}
}

// TestHumanLoopCmdNoArgs tests that human-loop without args returns "topic is required"
func TestHumanLoopCmdNoArgs(t *testing.T) {
	cmd := NewHumanLoopCmd()
	cmd.SetArgs([]string{})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when no topic provided, got nil")
	}
	if !strings.Contains(err.Error(), "topic is required") {
		t.Errorf("expected 'topic is required' error, got: %v", err)
	}
}

// TestHumanLoopCmdDryRun tests human-loop in dry-run mode
func TestHumanLoopCmdDryRun(t *testing.T) {
	origDryRun := dryRun
	defer func() { dryRun = origDryRun }()

	dryRun = true

	cmd := NewHumanLoopCmd()
	cmd.SetArgs([]string{"如何重构?"})

	err := cmd.Execute()
	if err != nil {
		t.Errorf("human-loop dry-run failed: %v", err)
	}
}

// TestHumanLoopCmdWithMockClaude tests the full human-loop flow with a mock Claude binary
func TestHumanLoopCmdWithMockClaude(t *testing.T) {
	// Create a mock claude script
	tmpDir := t.TempDir()
	mockScript := "#!/bin/sh\nexit 0\n"
	mockPath := filepath.Join(tmpDir, "mock_pi")
	if err := os.WriteFile(mockPath, []byte(mockScript), 0755); err != nil {
		t.Fatal(err)
	}

	// Switch to a temp working dir so .rick/draft/RFC is created there
	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	workDir := t.TempDir()
	if err := os.Chdir(workDir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(orig) }()

	if err := os.MkdirAll(filepath.Join(workDir, ".rick"), 0755); err != nil {
		t.Fatal(err)
	}

	// Write config pointing to mock pi
	cfgContent := `{"pi_path": "` + mockPath + `"}`
	cfgDir := filepath.Join(workDir, ".rick")
	if err := os.WriteFile(filepath.Join(cfgDir, "config.json"), []byte(cfgContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Override config load by patching env or using direct call
	cfg := &config.Config{PiPath: mockPath}

	// v8：不再预建 draft/rfc（v7 历史 RFC 目录）；产物目录为 loop_N/{orig,thought_path.md}。

	// Test runtime.CallCLI with mock (interactive mode)
	promptFile := filepath.Join(tmpDir, "prompt.md")
	if err := os.WriteFile(promptFile, []byte("# Test human-loop prompt"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := runtime.CallCLI(false, cfg, promptFile, runtime.ModeInteractive); err != nil {
		t.Errorf("runtime.CallCLI with mock failed: %v", err)
	}
}
