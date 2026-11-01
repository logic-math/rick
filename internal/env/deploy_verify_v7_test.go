package env

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDeployV7AgentsSmoke(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := deployRickAgents(); err != nil {
		t.Fatalf("deployRickAgents: %v", err)
	}
	agentsDir := filepath.Join(home, ".rick", "pi", "agent", "agents")
	for _, name := range []string{"research", "zpd-assessor", "feynman", "syllabus-builder", "lesson-writer", "originality-sparker", "recall-review"} {
		data, err := os.ReadFile(filepath.Join(agentsDir, name+".md"))
		if err != nil {
			t.Fatalf("%s.md not deployed: %v", name, err)
		}
		c := string(data)
		// frontmatter: name + tools + timeout + rick-managed
		for _, want := range []string{"name: " + name, "tools: read, grep, find, ls, bash, write, edit, web_search, fetch_content, subagent", "defaultContext: fresh", "thinking: high", "timeoutMs: 3600000", "systemPromptMode: replace", "rick-managed: true"} {
			if !strings.Contains(c, want) {
				t.Errorf("%s.md missing frontmatter %q", name, want)
			}
		}
		// body: skill wiki header with the agent name (dash)
		if !strings.Contains(c, "skill:"+name) {
			t.Errorf("%s.md body missing skill:%s header", name, name)
		}
		if len(c) < 500 {
			t.Errorf("%s.md body suspiciously short (%d bytes)", name, len(c))
		}
	}
	// teach must NOT be deployed
	if _, err := os.Stat(filepath.Join(agentsDir, "teach.md")); !os.IsNotExist(err) {
		t.Error("teach.md should not be deployed in v7")
	}
}
