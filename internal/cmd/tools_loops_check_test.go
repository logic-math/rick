package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// repoRoot resolves the repository root from the package dir (internal/cmd).
func repoRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	return filepath.Clean(filepath.Join(wd, "..", ".."))
}

// TestResolveRickDirForCheck pins the two accepted shapes: the .rick dir itself,
// and a workspace root containing .rick/. 只接受一种会让人踩「文件明明在，却报
// 目录不存在」的坑（CLI 的 --dir 由人和脚本混用）。
func TestResolveRickDirForCheck(t *testing.T) {
	t.Run("dotrick dir itself", func(t *testing.T) {
		base := t.TempDir()
		rickDir := filepath.Join(base, ".rick")
		makeLoopsDir(t, rickDir)
		got, err := resolveRickDirForCheck(rickDir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != rickDir {
			t.Fatalf("got %q, want %q", got, rickDir)
		}
	})

	t.Run("workspace root containing .rick", func(t *testing.T) {
		ws := t.TempDir()
		makeLoopsDir(t, filepath.Join(ws, ".rick"))
		got, err := resolveRickDirForCheck(ws)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != filepath.Join(ws, ".rick") {
			t.Fatalf("got %q, want %q", got, filepath.Join(ws, ".rick"))
		}
	})

	t.Run(".rick exists but has no loops/skills", func(t *testing.T) {
		ws := t.TempDir()
		if err := os.MkdirAll(filepath.Join(ws, ".rick"), 0o755); err != nil {
			t.Fatal(err)
		}
		got, err := resolveRickDirForCheck(ws)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != filepath.Join(ws, ".rick") {
			t.Fatalf("got %q, want the .rick dir (empty loops/skills is a pass)", got)
		}
	})

	t.Run("unresolvable dir", func(t *testing.T) {
		empty := t.TempDir()
		if _, err := resolveRickDirForCheck(empty); err == nil {
			t.Fatal("expected an error with an actionable message")
		} else if !strings.Contains(err.Error(), empty) {
			t.Fatalf("error should name the directory looked at, got: %v", err)
		}
	})
}

func TestRunLoopsCheck_CompliantLoopCounts(t *testing.T) {
	rickDir := t.TempDir()
	loopsDir := makeLoopsDir(t, rickDir)
	if err := os.WriteFile(filepath.Join(loopsDir, "my_loop.md"), []byte(compliantLoopContent), 0o644); err != nil {
		t.Fatal(err)
	}
	skillsDir := makeSkillsDir(t, rickDir)
	if err := os.WriteFile(filepath.Join(skillsDir, "my_skill.md"), []byte(compliantSkillContent), 0o644); err != nil {
		t.Fatal(err)
	}
	// README.md must not be counted nor validated
	if err := os.WriteFile(filepath.Join(loopsDir, "README.md"), []byte("# spec"), 0o644); err != nil {
		t.Fatal(err)
	}

	res := runLoopsCheck(rickDir)
	if !res.Pass {
		t.Fatalf("expected pass, got errors: %v", res.Errors)
	}
	if res.Loops != 1 || res.Skills != 1 {
		t.Fatalf("counts = loops %d / skills %d, want 1/1 (README excluded)", res.Loops, res.Skills)
	}
	if res.Dir != rickDir {
		t.Fatalf("dir = %q, want %q", res.Dir, rickDir)
	}
	if res.Errors == nil {
		t.Fatal("Errors must be a non-nil slice (JSON contract: [] not null)")
	}
}

func TestRunLoopsCheck_FailureReportsFileAndReason(t *testing.T) {
	rickDir := t.TempDir()
	loopsDir := makeLoopsDir(t, rickDir)
	bad := `---
name: bad-loop
---

## 目标

## 上下文管理

## 可调用工具

## 停止标准
`
	if err := os.WriteFile(filepath.Join(loopsDir, "bad_loop.md"), []byte(bad), 0o644); err != nil {
		t.Fatal(err)
	}
	res := runLoopsCheck(rickDir)
	if res.Pass {
		t.Fatal("expected failure for a loop missing trigger + 产出评估")
	}
	var sawTrigger, sawSection bool
	for _, e := range res.Errors {
		if strings.Contains(e, "bad_loop.md") && strings.Contains(e, "trigger") {
			sawTrigger = true
		}
		if strings.Contains(e, "bad_loop.md") && strings.Contains(e, "产出评估") {
			sawSection = true
		}
	}
	if !sawTrigger || !sawSection {
		t.Fatalf("errors should name the file and each problem, got: %v", res.Errors)
	}
}

// TestRunLoopsCheck_JSONContract pins the machine contract consumed by tooling
// (e.g. gate11): a single JSON line with these exact keys.
func TestRunLoopsCheck_JSONContract(t *testing.T) {
	rickDir := t.TempDir()
	makeLoopsDir(t, rickDir)
	raw, err := json.Marshal(runLoopsCheck(rickDir))
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	for _, key := range []string{"pass", "dir", "loops", "skills", "errors"} {
		if _, ok := m[key]; !ok {
			t.Errorf("JSON contract missing key %q: %s", key, string(raw))
		}
	}
	if m["pass"] != true {
		t.Errorf("empty loops dir should pass (validator skips missing entries), got: %s", string(raw))
	}
}

// TestRsiLoopIsCompliant guards the institutional artifact itself: the loop that
// the `rsi` session type injects must stay format-compliant (otherwise every RSI
// session would start from a broken contract). 这条断言把「loop 目录长期没人校验」
// 的问题钉死（实测曾经 loops/README 的清单就与实际不一致）。
func TestRsiLoopIsCompliant(t *testing.T) {
	root := repoRoot(t)
	rickDir := filepath.Join(root, ".rick")
	if !isDir(filepath.Join(rickDir, "loops")) {
		t.Skipf("no .rick/loops in %s", root)
	}
	res := runLoopsCheck(rickDir)
	if !res.Pass {
		t.Fatalf("project .rick fails loops_check: %v", res.Errors)
	}
	loopPath := filepath.Join(rickDir, "loops", "rick-rsi-loop.md")
	raw, err := os.ReadFile(loopPath)
	if err != nil {
		t.Fatalf("rick-rsi-loop.md must exist in the rick repo: %v", err)
	}
	body := string(raw)
	for _, want := range []string{
		"name: rick-rsi-loop",
		"trigger:",
		"scope:",
		"## 目标",
		"## 上下文管理",
		"## 可调用工具",
		"## 产出评估",
		"## 停止标准",
		// 机制必须可执行：命令出现在 loop 里，而不是只在别处定义
		"rick tools dev-web",
		"rick tools release",
		"rsi_check",
		// 硬约束：dev 工作区 + 人类确认
		"dev 工作区",
		"人类确认",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("rick-rsi-loop.md missing %q", want)
		}
	}
	if res.Loops < 5 {
		t.Errorf("expected the repo to carry >=5 loops (got %d)", res.Loops)
	}
}
