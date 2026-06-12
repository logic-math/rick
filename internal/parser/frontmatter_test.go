package parser

import "testing"

func TestExtractBugFrontmatter(t *testing.T) {
	t.Run("normal frontmatter", func(t *testing.T) {
		content := "---\nsummary: \"修复 nil 指针\"\nstatus: resolved\n---\n正文"
		summary, status := ExtractBugFrontmatter(content)
		if summary != "修复 nil 指针" {
			t.Errorf("expected summary '修复 nil 指针', got %q", summary)
		}
		if status != "resolved" {
			t.Errorf("expected status 'resolved', got %q", status)
		}
	})

	t.Run("no frontmatter returns empty", func(t *testing.T) {
		content := "只有正文，没有 frontmatter"
		summary, status := ExtractBugFrontmatter(content)
		if summary != "" || status != "" {
			t.Errorf("expected empty strings, got summary=%q status=%q", summary, status)
		}
	})

	t.Run("single-quoted values", func(t *testing.T) {
		content := "---\nsummary: '单引号值'\n---\n"
		summary, _ := ExtractBugFrontmatter(content)
		if summary != "单引号值" {
			t.Errorf("expected '单引号值', got %q", summary)
		}
	})

	t.Run("double-quoted values", func(t *testing.T) {
		content := "---\nsummary: \"nil pointer in runner\"\nstatus: 'open'\n---\n"
		summary, status := ExtractBugFrontmatter(content)
		if summary != "nil pointer in runner" {
			t.Errorf("expected summary without quotes, got %q", summary)
		}
		if status != "open" {
			t.Errorf("expected status without quotes, got %q", status)
		}
	})

	t.Run("field absent", func(t *testing.T) {
		content := "---\nsummary: only summary\n---\n"
		summary, status := ExtractBugFrontmatter(content)
		if summary != "only summary" {
			t.Errorf("expected 'only summary', got %q", summary)
		}
		if status != "" {
			t.Errorf("expected empty status, got %q", status)
		}
	})
}
