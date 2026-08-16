//go:build realpi

// Smoke tests against the REAL pi binary (gated behind the `realpi` build tag
// — not run by default; they exec a real LLM which needs a provider API key).
// Run manually: go test -tags realpi ./internal/runtime/...
package runtime

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

// TestRealPi_Smoke executes the genuine pi binary via runtime.Run and confirms
// rick correctly execs pi + parses the real JSONL event stream.
func TestRealPi_Smoke(t *testing.T) {
	skipIfNoPi(t)
	tmp := t.TempDir()
	pf := filepath.Join(tmp, "prompt.md")
	if err := os.WriteFile(pf, []byte("Reply with exactly: PI_OK"), 0644); err != nil {
		t.Fatal(err)
	}
	sessionID, trace, err := NewPiRuntime("").Run("", pf, nil)
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if sessionID == "" {
		t.Fatalf("expected non-empty session ID")
	}
	t.Logf("real pi session: id=%s duration=%v finalMessage=%q toolCalls=%d",
		sessionID, trace.Duration, trace.FinalMessage, len(trace.ToolCalls))
}

// TestRealPi_RealToolCall runs the FULL end-to-end with a real Read tool call.
func TestRealPi_RealToolCall(t *testing.T) {
	skipIfNoPi(t)
	raw := os.Getenv("RICK_REAL_PI_ARGS")
	if raw == "" {
		t.Skip("skipping real tool-call test: set RICK_REAL_PI_ARGS=provider=X,model=Y,api-key=Z to run")
	}
	var extraArgs []string
	for _, kv := range strings.Split(raw, ",") {
		parts := strings.SplitN(kv, "=", 2)
		if len(parts) != 2 {
			continue
		}
		extraArgs = append(extraArgs, "--"+strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]))
	}

	target := filepath.Join(t.TempDir(), "target.txt")
	if err := os.WriteFile(target, []byte("real tool-call payload"), 0644); err != nil {
		t.Fatal(err)
	}

	tmp := t.TempDir()
	pf := filepath.Join(tmp, "prompt.md")
	if err := os.WriteFile(pf, []byte("Read "+target+" and say DONE"), 0644); err != nil {
		t.Fatal(err)
	}

	sessionID, trace, err := NewPiRuntime("", extraArgs...).Run("", pf, nil)
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if sessionID == "" {
		t.Error("expected non-empty session ID")
	}
	calls := trace.ToolCalls
	if len(calls) == 0 {
		t.Fatalf("expected at least one tool call, got 0.\nraw log:\n%s", readRaw(t, trace.RawLogPath))
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
	if trace.FinalMessage == "" {
		t.Error("expected non-empty assistant FinalMessage")
	}
	t.Logf("real tool call: id=%s toolCalls=%d finalMessage=%q",
		sessionID, len(calls), trace.FinalMessage)
}

func readRaw(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		return "<read error: " + err.Error() + ">"
	}
	return string(b)
}
