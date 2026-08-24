package runtime

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestIsSessionReady covers the two fake-JSONL cases from the task's boundary
// spec: with agent_settled the session is ready (sessionID non-empty && settled),
// without it the session is not ready (parseStream still returns the partial
// session — no error — so readiness is a caller decision, not a parse failure).
func TestIsSessionReady(t *testing.T) {
	withSettled := `{"type":"session","id":"s123"}
{"type":"agent_settled"}
`
	sess := mustParse(t, withSettled)
	if !isSessionReady(sess.sessionID, sess.settled) {
		t.Error("isSessionReady: want true with session id + agent_settled")
	}

	withoutSettled := `{"type":"session","id":"s123"}
`
	sess = mustParse(t, withoutSettled)
	if sess.sessionID != "s123" {
		t.Fatalf("session id: want s123, got %q", sess.sessionID)
	}
	if sess.settled {
		t.Fatal("expected settled=false without agent_settled")
	}
	if isSessionReady(sess.sessionID, sess.settled) {
		t.Error("isSessionReady: want false without agent_settled")
	}

	// Empty session id is never ready, even if settled.
	if isSessionReady("", true) {
		t.Error("isSessionReady: want false with empty session id")
	}
}

// TestPiRuntimeRun_AppendSystemPrompt verifies the runtime skeleton: methodText
// is written to a temp file, injected via --append-system-prompt <methodFile>,
// promptFile is passed last as the user prompt, the temp method file is removed
// (defer cleanup) once Run returns, and the JSONL session id/trace are returned.
func TestPiRuntimeRun_AppendSystemPrompt(t *testing.T) {
	tmp := t.TempDir()
	argvFile := filepath.Join(tmp, "argv.txt")
	methodContentFile := filepath.Join(tmp, "method-content.txt")
	mockPath := filepath.Join(tmp, "mock_pi")

	// Mock pi: record argv; capture the method file content (the arg following
	// --append-system-prompt); then emit a minimal JSONL session + agent_settled.
	// Only shell builtins are used (printf/echo/read), so no PATH dependency.
	script := "#!/bin/sh\n" +
		"printf '%s\\n' \"$@\" > \"" + argvFile + "\"\n" +
		"prev=\"\"\n" +
		"method=\"\"\n" +
		"for a in \"$@\"; do\n" +
		"  if [ -z \"$method\" ] && [ \"$prev\" = \"--append-system-prompt\" ]; then method=\"$a\"; fi\n" +
		"  prev=\"$a\"\n" +
		"done\n" +
		"if [ -n \"$method\" ]; then\n" +
		"  while IFS= read -r line || [ -n \"$line\" ]; do printf '%s\\n' \"$line\"; done < \"$method\" > \"" + methodContentFile + "\"\n" +
		"fi\n" +
		"echo '{\"type\":\"session\",\"id\":\"s123\"}'\n" +
		"echo '{\"type\":\"agent_settled\"}'\n"
	if err := os.WriteFile(mockPath, []byte(script), 0755); err != nil {
		t.Fatal(err)
	}

	promptFile := filepath.Join(tmp, "prompt.md")
	if err := os.WriteFile(promptFile, []byte("# instance context"), 0644); err != nil {
		t.Fatal(err)
	}

	sessionID, trace, err := NewPiRuntime(mockPath).Run("the method text", promptFile, nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if sessionID != "s123" {
		t.Errorf("sessionID: want s123, got %q", sessionID)
	}
	if trace == nil {
		t.Fatal("trace is nil")
	}
	if !trace.Settled {
		t.Error("trace.Settled: want true")
	}
	if trace.SessionID != "s123" {
		t.Errorf("trace.SessionID: want s123, got %q", trace.SessionID)
	}

	// argv: --mode json [--append-system-prompt <methodFile>] <promptFile>
	argv := readArgv(t, argvFile)
	if len(argv) < 2 || argv[0] != "--mode" || argv[1] != "json" {
		t.Fatalf("argv should start with --mode json, got %v", argv)
	}
	methodFile := ""
	for i := 0; i+1 < len(argv); i++ {
		if argv[i] == "--append-system-prompt" && methodFile == "" {
			methodFile = argv[i+1] // 第一个 append = method 临时文件（v4.4.5 起第二个是 promptFile）
		}
	}
	if methodFile == "" {
		t.Fatalf("argv missing --append-system-prompt <methodFile>: %v", argv)
	}
	// v4.4.5: promptFile 也走 --append-system-prompt（协议常驻系统提示词），
	// user 消息是 bootstrap 触发。
	promptInjected := false
	for i := 0; i+1 < len(argv); i++ {
		if argv[i] == "--append-system-prompt" && argv[i+1] == promptFile {
			promptInjected = true
		}
	}
	if !promptInjected {
		t.Errorf("argv missing --append-system-prompt %q: %v", promptFile, argv)
	}

	// methodText was written to the temp method file…
	content, err := os.ReadFile(methodContentFile)
	if err != nil {
		t.Fatalf("read captured method content: %v", err)
	}
	if strings.TrimSpace(string(content)) != "the method text" {
		t.Errorf("method file content: want %q, got %q", "the method text", string(content))
	}

	// …and the temp method file is removed on return (defer cleanup, deleted once used).
	if _, err := os.Stat(methodFile); !os.IsNotExist(err) {
		t.Errorf("method temp file should be removed after Run, stat err=%v", err)
	}
}
