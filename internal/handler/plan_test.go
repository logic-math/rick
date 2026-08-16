package handler

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestReEnterPlan_NonExistentJob tests error when job plan dir does not exist.
func TestReEnterPlan_NonExistentJob(t *testing.T) {
	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(orig) }()

	if err := os.MkdirAll(filepath.Join(dir, ".rick"), 0755); err != nil {
		t.Fatal(err)
	}

	err = ReEnterPlan("job_999", "test requirement", Options{})
	if err == nil {
		t.Fatal("expected error for non-existent job plan directory")
	}
	if !strings.Contains(err.Error(), "job job_999 plan directory does not exist") {
		t.Errorf("unexpected error message: %v", err)
	}
}

// TestPlan_WithMockPi tests Plan with mock pi binary.
func TestPlan_WithMockPi(t *testing.T) {
	mockDir := t.TempDir()
	mockScript := "#!/bin/sh\nexit 0\n"
	mockPath := filepath.Join(mockDir, "pi")
	if err := os.WriteFile(mockPath, []byte(mockScript), 0755); err != nil {
		t.Fatal(err)
	}

	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(orig) }()

	if err := os.MkdirAll(filepath.Join(dir, ".rick"), 0755); err != nil {
		t.Fatal(err)
	}

	// Isolate HOME so LoadConfig reads this test's config (never the real
	// ~/.rick/config.json), and point pi_path at the mock pi so the workflow
	// resolves the mock directly — not the real managed runtime or PATH pi.
	home := t.TempDir()
	t.Setenv("HOME", home)

	// Write a config with mock pi path
	cfgContent := fmt.Sprintf(`{"pi_path": "%s"}`, mockPath)
	cfgDir := filepath.Join(home, ".rick")
	if err := os.MkdirAll(cfgDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "config.json"), []byte(cfgContent), 0644); err != nil {
		t.Fatal(err)
	}

	origPath := os.Getenv("PATH")
	_ = os.Setenv("PATH", mockDir+":"+origPath)
	defer os.Setenv("PATH", origPath)

	err = Plan("test requirement", Options{})
	t.Logf("Plan returned: %v", err)
}
