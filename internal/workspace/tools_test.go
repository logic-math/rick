package workspace

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadToolsList(t *testing.T) {
	t.Run("no tools directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		tools, err := LoadToolsList(tmpDir)
		if err != nil {
			t.Fatalf("LoadToolsList() error: %v", err)
		}
		if len(tools) != 0 {
			t.Errorf("Expected empty slice, got %d tools", len(tools))
		}
	})

	t.Run("empty tools directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		toolsDir := filepath.Join(tmpDir, "tools")
		if err := os.MkdirAll(toolsDir, 0755); err != nil {
			t.Fatalf("Failed to create tools dir: %v", err)
		}
		tools, err := LoadToolsList(tmpDir)
		if err != nil {
			t.Fatalf("LoadToolsList() error: %v", err)
		}
		if len(tools) != 0 {
			t.Errorf("Expected empty slice, got %d tools", len(tools))
		}
	})

	t.Run("single tool with description", func(t *testing.T) {
		tmpDir := t.TempDir()
		toolsDir := filepath.Join(tmpDir, "tools")
		if err := os.MkdirAll(toolsDir, 0755); err != nil {
			t.Fatalf("Failed to create tools dir: %v", err)
		}
		content := "# Description: 示例工具\nprint('hello')\n"
		if err := os.WriteFile(filepath.Join(toolsDir, "sample_tool.py"), []byte(content), 0644); err != nil {
			t.Fatalf("Failed to write tool file: %v", err)
		}

		tools, err := LoadToolsList(tmpDir)
		if err != nil {
			t.Fatalf("LoadToolsList() error: %v", err)
		}
		if len(tools) != 1 {
			t.Fatalf("Expected 1 tool, got %d", len(tools))
		}
		if tools[0].Name != "sample_tool" {
			t.Errorf("Expected name 'sample_tool', got '%s'", tools[0].Name)
		}
		if tools[0].Description != "示例工具" {
			t.Errorf("Expected description '示例工具', got '%s'", tools[0].Description)
		}
	})

	t.Run("tool without description comment", func(t *testing.T) {
		tmpDir := t.TempDir()
		toolsDir := filepath.Join(tmpDir, "tools")
		if err := os.MkdirAll(toolsDir, 0755); err != nil {
			t.Fatalf("Failed to create tools dir: %v", err)
		}
		content := "print('no description')\n"
		if err := os.WriteFile(filepath.Join(toolsDir, "no_desc.py"), []byte(content), 0644); err != nil {
			t.Fatalf("Failed to write tool file: %v", err)
		}

		tools, err := LoadToolsList(tmpDir)
		if err != nil {
			t.Fatalf("LoadToolsList() error: %v", err)
		}
		if len(tools) != 1 {
			t.Fatalf("Expected 1 tool, got %d", len(tools))
		}
		if tools[0].Description != "" {
			t.Errorf("Expected empty description, got '%s'", tools[0].Description)
		}
	})

	t.Run("non-py files are ignored", func(t *testing.T) {
		tmpDir := t.TempDir()
		toolsDir := filepath.Join(tmpDir, "tools")
		if err := os.MkdirAll(toolsDir, 0755); err != nil {
			t.Fatalf("Failed to create tools dir: %v", err)
		}
		if err := os.WriteFile(filepath.Join(toolsDir, "tool.py"), []byte("# Description: py tool\n"), 0644); err != nil {
			t.Fatalf("Failed to write .py file: %v", err)
		}
		if err := os.WriteFile(filepath.Join(toolsDir, "readme.md"), []byte("# readme"), 0644); err != nil {
			t.Fatalf("Failed to write .md file: %v", err)
		}

		tools, err := LoadToolsList(tmpDir)
		if err != nil {
			t.Fatalf("LoadToolsList() error: %v", err)
		}
		if len(tools) != 1 {
			t.Errorf("Expected 1 tool (only .py), got %d", len(tools))
		}
	})
}
