package executor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExtractBugFrontmatter(t *testing.T) {
	t.Run("normal frontmatter", func(t *testing.T) {
		content := "---\nsummary: fix auth bug\nstatus: resolved\n---\n\nBody text here."
		summary, status := extractBugFrontmatter(content)
		if summary != "fix auth bug" {
			t.Errorf("expected summary 'fix auth bug', got %q", summary)
		}
		if status != "resolved" {
			t.Errorf("expected status 'resolved', got %q", status)
		}
	})

	t.Run("quoted values", func(t *testing.T) {
		content := "---\nsummary: \"nil pointer in runner\"\nstatus: 'open'\n---\n"
		summary, status := extractBugFrontmatter(content)
		if summary != "nil pointer in runner" {
			t.Errorf("expected summary without quotes, got %q", summary)
		}
		if status != "open" {
			t.Errorf("expected status without quotes, got %q", status)
		}
	})

	t.Run("missing frontmatter", func(t *testing.T) {
		content := "# Bug Report\nNo frontmatter here."
		summary, status := extractBugFrontmatter(content)
		if summary != "" || status != "" {
			t.Errorf("expected empty strings for missing frontmatter, got summary=%q status=%q", summary, status)
		}
	})

	t.Run("field absent", func(t *testing.T) {
		content := "---\nsummary: only summary\n---\n"
		summary, status := extractBugFrontmatter(content)
		if summary != "only summary" {
			t.Errorf("expected 'only summary', got %q", summary)
		}
		if status != "" {
			t.Errorf("expected empty status, got %q", status)
		}
	})
}

func TestLoadDebugDirSummaries(t *testing.T) {
	t.Run("bug*.md files are read", func(t *testing.T) {
		tmpDir := t.TempDir()
		debugDir := filepath.Join(tmpDir, "debug")
		if err := os.MkdirAll(debugDir, 0755); err != nil {
			t.Fatal(err)
		}
		content := "---\nsummary: test crash\nstatus: resolved\n---\nFull body text."
		if err := os.WriteFile(filepath.Join(debugDir, "bug1-crash.md"), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}

		result := LoadDebugDirSummaries(tmpDir)
		if !strings.Contains(result, "test crash") {
			t.Errorf("expected summary in result, got %q", result)
		}
		if strings.Contains(result, "Full body text") {
			t.Error("result should not contain full body text")
		}
	})

	t.Run("non-bug*.md files are skipped", func(t *testing.T) {
		tmpDir := t.TempDir()
		debugDir := filepath.Join(tmpDir, "debug")
		if err := os.MkdirAll(debugDir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(debugDir, "notes.md"), []byte("should be ignored"), 0644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(debugDir, "bug1.md"), []byte("---\nsummary: real bug\nstatus: open\n---\n"), 0644); err != nil {
			t.Fatal(err)
		}

		result := LoadDebugDirSummaries(tmpDir)
		if strings.Contains(result, "should be ignored") {
			t.Error("non-bug file should be skipped")
		}
		if !strings.Contains(result, "real bug") {
			t.Errorf("bug file summary should be present, got %q", result)
		}
	})

	t.Run("directory not exist returns empty", func(t *testing.T) {
		result := LoadDebugDirSummaries("/nonexistent/path/xyz123")
		if result != "" {
			t.Errorf("expected empty string for non-existent dir, got %q", result)
		}
	})

	t.Run("empty workspaceDir returns empty", func(t *testing.T) {
		result := LoadDebugDirSummaries("")
		if result != "" {
			t.Errorf("expected empty string for empty workspaceDir, got %q", result)
		}
	})
}

func TestLoadDebugContext_WithDebugDir(t *testing.T) {
	tmpDir := t.TempDir()
	debugDir := filepath.Join(tmpDir, "debug")
	if err := os.MkdirAll(debugDir, 0755); err != nil {
		t.Fatal(err)
	}
	bugContent := "---\nsummary: connection timeout\nstatus: resolved\n---\nLong body that should not appear."
	if err := os.WriteFile(filepath.Join(debugDir, "bug1.md"), []byte(bugContent), 0644); err != nil {
		t.Fatal(err)
	}
	// Also write debug.md — it should NOT be used when debug/ has content
	if err := os.WriteFile(filepath.Join(tmpDir, "debug.md"), []byte("old debug.md content"), 0644); err != nil {
		t.Fatal(err)
	}

	result := LoadDebugContext(tmpDir)
	if !strings.Contains(result, "connection timeout") {
		t.Errorf("expected summary from debug/, got %q", result)
	}
	if strings.Contains(result, "Long body") {
		t.Error("should not include full body from bug file")
	}
	if strings.Contains(result, "old debug.md content") {
		t.Error("should not fall back to debug.md when debug/ has content")
	}
}

func TestLoadDebugContext_Fallback(t *testing.T) {
	tmpDir := t.TempDir()
	// debug/ dir is absent, only debug.md exists
	debugMdContent := "## debug1: some old issue"
	if err := os.WriteFile(filepath.Join(tmpDir, "debug.md"), []byte(debugMdContent), 0644); err != nil {
		t.Fatal(err)
	}

	result := LoadDebugContext(tmpDir)
	if result != debugMdContent {
		t.Errorf("expected fallback to debug.md content, got %q", result)
	}
}

func TestLoadDebugContext_EmptyWorkspaceDir(t *testing.T) {
	result := LoadDebugContext("")
	if result != "" {
		t.Errorf("expected empty string for empty workspaceDir, got %q", result)
	}
}

func TestLoadDebugContext_NonExistentDir(t *testing.T) {
	result := LoadDebugContext("/nonexistent/dir/xyz")
	if result != "" {
		t.Errorf("expected empty string for non-existent dir, got %q", result)
	}
}
