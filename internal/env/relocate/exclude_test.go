package relocate

import "testing"

func TestDefaultRulesCoverDecidedExclusions(t *testing.T) {
	rules := DefaultRules()
	// The human-approved exclusion table (job_46 design tree, R2 spec).
	// Each entry: root, relPath expected to match.
	want := []struct {
		root Root
		path string
	}{
		{RootWorkspace, "web/node_modules"},
		{RootWorkspace, "web/node_modules/vite/dist/node/cli.js"},
		{RootWorkspace, "bin"},
		{RootWorkspace, "bin/rick"},
		{RootWorkspace, "web/dist.prev"},
		{RootWorkspace, ".pi"},
		{RootWorkspace, ".pi/subagents/artifacts/x.json"},
		{RootWorkspace, ".morty"},
		{RootState, "web.pid"},
		{RootState, "web.lock"},
		{RootState, "web.log"},
		{RootState, "release.log"},
		{RootState, "config.json.bak.20260817110141"},
		{RootState, "config.json.bak.20261010-glm-to-ds"},
		{RootPIAgent, "web-search-cache"},
		{RootPIAgent, "web-search-cache/deepseek/1.json"},
		{RootPIAgent, "bin/rg"},
		{RootPIAgent, "extensions/catpaw-proxy.ts"},
		{RootPIAgent, "extensions/catpaw-proxy.ts.bak-20260922"},
	}
	m := NewMatcher(rules, false)
	for _, w := range want {
		if rule, ok := m.Match(w.root, w.path); !ok {
			t.Errorf("expected %s:%s to be excluded", w.root, w.path)
		} else if rule.Kind == "" || rule.Reason == "" {
			t.Errorf("rule for %s:%s must carry Kind and Reason, got %+v", w.root, w.path, rule)
		}
	}
}

func TestMatcherKeepsRegularContent(t *testing.T) {
	m := NewMatcher(DefaultRules(), false)
	keep := []struct {
		root Root
		path string
	}{
		{RootWorkspace, "web/dist/index.html"},
		{RootWorkspace, "internal/web/registry.go"},
		{RootWorkspace, "web/binaries.txt"},      // "bin" must not match "web/binaries"
		{RootWorkspace, ".github/workflows.yml"}, // dotfile, not .pi/.morty
		{RootState, "web.json"},
		{RootState, "config.json"},
		{RootState, "web/sessions.json"},
		{RootState, "pi/agent/auth.json"},
		{RootPIAgent, "runtime/node_modules/.bin/pi"},
		{RootPIAgent, "sessions/--workdir-sunquan20-AI_CODING-rick--/s.jsonl"},
		{RootPIAgent, "models-store.json"},
		{RootPIAgent, "settings.json"},
	}
	for _, k := range keep {
		if _, ok := m.Match(k.root, k.path); ok {
			t.Errorf("expected %s:%s to be KEPT, got excluded", k.root, k.path)
		}
	}
}

func TestMatcherRootScoping(t *testing.T) {
	m := NewMatcher(DefaultRules(), false)
	// "web-search-cache" is a pi-agent rule; the same relPath under the
	// workspace root must NOT be excluded — root scoping is the whole point
	// of the Root field. (Note: workspace-root bin/rg IS excluded, but by
	// the workspace "bin" rule — a repo's bin/ is build output.)
	if _, ok := m.Match(RootWorkspace, "web-search-cache"); ok {
		t.Fatal("workspace-root web-search-cache must not be excluded by the pi-agent rule")
	}
	// "web.log" is a state rule; a workspace file web.log stays.
	if _, ok := m.Match(RootWorkspace, "web.log"); ok {
		t.Fatal("workspace-root web.log must not be excluded by the state rule")
	}
	// "bin/rg" is a pi-agent rule but workspace "bin" legitimately covers it.
	if _, ok := m.Match(RootPIAgent, "web.log"); ok {
		t.Fatal("pi-agent-root web.log must not be excluded by the state rule")
	}
}

func TestMatcherDisabledExcludesNothing(t *testing.T) {
	m := NewMatcher(DefaultRules(), true)
	for _, root := range []Root{RootWorkspace, RootState, RootPIAgent} {
		if _, ok := m.Match(root, "web/node_modules"); ok {
			t.Fatalf("disabled matcher must exclude nothing (root=%s)", root)
		}
	}
}

func TestMatcherDirPrefixSemantics(t *testing.T) {
	m := NewMatcher([]Rule{{Pattern: "node_modules", Root: RootWorkspace, Kind: KindCache, Reason: "t"}}, false)
	if _, ok := m.Match(RootWorkspace, "node_modules"); !ok {
		t.Fatal("exact dir must match")
	}
	if _, ok := m.Match(RootWorkspace, "node_modules/x/y.js"); !ok {
		t.Fatal("dir subtree must match")
	}
	if _, ok := m.Match(RootWorkspace, "node_modulesX"); ok {
		t.Fatal("prefix must be path-segment aware, not string prefix")
	}
	if _, ok := m.Match(RootWorkspace, "foo/node_modules"); ok {
		t.Fatal("nested dir with same name must not match top-level rule")
	}
}

func TestMatcherMatchFuncForRoot(t *testing.T) {
	m := NewMatcher(DefaultRules(), false)
	ws := m.MatchFunc(RootWorkspace) // keep-semantics: true = keep
	if ws("web/node_modules/vite") {
		t.Fatal("workspace matchfunc must exclude web/node_modules (false)")
	}
	if !ws("internal/env/relocate/marker.go") {
		t.Fatal("workspace matchfunc must keep source files (true)")
	}
	st := m.MatchFunc(RootState)
	if !st("web.json") {
		t.Fatal("state matchfunc must keep web.json (true)")
	}
	if st("web.pid") {
		t.Fatal("state matchfunc must exclude web.pid (false)")
	}
}

func TestMatcherGlobSuffix(t *testing.T) {
	m := NewMatcher([]Rule{{Pattern: "config.json.bak.*", Root: RootState, Kind: KindCache, Reason: "t"}}, false)
	if _, ok := m.Match(RootState, "config.json.bak.20260817"); !ok {
		t.Fatal("glob suffix must match")
	}
	if _, ok := m.Match(RootState, "config.json"); ok {
		t.Fatal("glob must not match the base name")
	}
	if _, ok := m.Match(RootState, "config.json.bak"); ok {
		t.Fatal("glob requires the dot separator — bare .bak must not match")
	}
}

func TestMatcherRootItselfNeverExcluded(t *testing.T) {
	m := NewMatcher(DefaultRules(), false)
	for _, root := range []Root{RootWorkspace, RootState, RootPIAgent} {
		if _, ok := m.Match(root, ""); ok {
			t.Fatalf("root itself (relPath \"\") must never be excluded (root=%s)", root)
		}
	}
}
