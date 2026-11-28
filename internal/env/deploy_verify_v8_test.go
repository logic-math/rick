package env

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDeployV8AgentsSmoke(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := deployRickAgents(); err != nil {
		t.Fatalf("deployRickAgents: %v", err)
	}
	agentsDir := filepath.Join(home, ".rick", "pi", "agent", "agents")
	// v8（job_57）：单 orig agent（human-loop 思考门禁 loop 的调研启发者）。
	for _, name := range []string{"orig"} {
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
		// body: skill wiki header with the agent name
		if !strings.Contains(c, "skill:"+name) {
			t.Errorf("%s.md body missing skill:%s header", name, name)
		}
		if len(c) < 300 {
			t.Errorf("%s.md body suspiciously short (%d bytes)", name, len(c))
		}
	}
	// v8 移除的 agent 不得部署：v7 四教学 agent（research/syllabus-builder/
	// lesson-writer/recall-review）+ 更早下线的 teach/zpd-assessor/feynman/
	// originality-sparker。
	for _, gone := range []string{
		"research", "syllabus-builder", "lesson-writer", "recall-review",
		"teach", "zpd-assessor", "feynman", "originality-sparker",
	} {
		if _, err := os.Stat(filepath.Join(agentsDir, gone+".md")); !os.IsNotExist(err) {
			t.Errorf("%s.md should not be deployed in v8", gone)
		}
	}
}
