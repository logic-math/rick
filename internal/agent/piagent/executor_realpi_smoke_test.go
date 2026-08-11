//go:build realpi

// Smoke tests against the REAL pi binary (gated behind the `realpi` build tag
// — not run by default; they exec a real LLM which needs a provider API key).
// Run manually: go test -tags realpi ./internal/agent/piagent/...
//
// Provider/model/key are injected via pi's env vars (PI_PROVIDER/PI_MODEL/PI_API_KEY),
// so rick's code path under test is the real Execute() (pi --mode json) — no test-
// specific flags. This mirrors how a user configures pi in their shell.
package piagent

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func skipIfNoPi(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("pi"); err != nil {
		t.Skipf("pi not on PATH: %v", err)
	}
}

// TestRealPi_Smoke executes the genuine pi binary via piagent.Execute and
// confirms rick correctly execs pi + parses the real JSONL event stream. With no
// provider configured pi errors fast (403) but still emits a well-formed stream,
// so the session parses with a non-empty ID.
func TestRealPi_Smoke(t *testing.T) {
	skipIfNoPi(t)
	tmp := t.TempDir()
	pf := filepath.Join(tmp, "prompt.md")
	if err := os.WriteFile(pf, []byte("Reply with exactly: PI_OK"), 0644); err != nil {
		t.Fatal(err)
	}
	sess, err := NewExecutor("").Execute(pf, "smoke", tmp, "raw.log")
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if sess == nil || sess.ID() == "" {
		t.Fatalf("expected non-empty session ID, got sess=%v", sess)
	}
	t.Logf("real pi session: id=%s duration=%v finalMessage=%q toolCalls=%d",
		sess.ID(), sess.Duration(), sess.FinalMessage(), len(sess.ToolCalls()))
}

// TestRealPi_RealToolCall runs the FULL end-to-end: real pi + real LLM provider
// performing a real Read tool call. Provider/model/key are injected as pi flags
// via NewExecutor's extraArgs (the mechanism rick's config.PiExtraArgs feeds).
// Set RICK_REAL_PI_ARGS="provider=deepseek,model=deepseek-v4-flash,api-key=sk-..."
// to run; skips otherwise.
func TestRealPi_RealToolCall(t *testing.T) {
	skipIfNoPi(t)
	raw := os.Getenv("RICK_REAL_PI_ARGS")
	if raw == "" {
		t.Skip("skipping real tool-call test: set RICK_REAL_PI_ARGS=provider=X,model=Y,api-key=Z to run")
	}
	// Parse "k1=v1,k2=v2" -> ["--k1","v1","--k2","v2"].
	var extraArgs []string
	for _, kv := range strings.Split(raw, ",") {
		parts := strings.SplitN(kv, "=", 2)
		if len(parts) != 2 {
			continue
		}
		extraArgs = append(extraArgs, "--"+strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]))
	}

	// Target file the LLM is asked to read.
	target := filepath.Join(t.TempDir(), "target.txt")
	if err := os.WriteFile(target, []byte("real tool-call payload"), 0644); err != nil {
		t.Fatal(err)
	}

	tmp := t.TempDir()
	pf := filepath.Join(tmp, "prompt.md")
	if err := os.WriteFile(pf, []byte("Read "+target+" and say DONE"), 0644); err != nil {
		t.Fatal(err)
	}

	sess, err := NewExecutor("", extraArgs...).Execute(pf, "realtool", tmp, "raw.log")
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if sess == nil {
		t.Fatal("nil session")
	}
	if sess.ID() == "" {
		t.Error("expected non-empty session ID")
	}
	// A real Read tool call must have been captured.
	calls := sess.ToolCalls()
	if len(calls) == 0 {
		t.Fatalf("expected at least one tool call, got 0.\nraw log:\n%s", readRaw(t, sess.RawLogPath()))
	}
	foundRead := false
	for _, c := range calls {
		if strings.EqualFold(c.Name, "read") {
			foundRead = true
			if c.Output == "" {
				t.Errorf("read tool output empty; call=%+v", c)
			}
		}
	}
	if !foundRead {
		t.Errorf("no read tool call found in %d calls: %+v", len(calls), calls)
	}
	if sess.FinalMessage() == "" {
		t.Error("expected non-empty assistant FinalMessage")
	}
	t.Logf("real tool call: id=%s toolCalls=%d finalMessage=%q",
		sess.ID(), len(calls), sess.FinalMessage())
}

func readRaw(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		return "<read error: " + err.Error() + ">"
	}
	return string(b)
}
