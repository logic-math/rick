package env

import (
	"os"
	"path/filepath"
	"testing"
)

// All tests isolate HOME so machine state (~/.rick/web) is never touched.
func TestDeployWebScaffoldIdempotent(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	deployed, err := DeployWebScaffold()
	if err != nil {
		t.Fatalf("first DeployWebScaffold: %v", err)
	}
	if !deployed {
		t.Fatal("first deploy should report deployed=true")
	}

	// Scaffold tree spot checks (KR: src/main.tsx exists, root configs present).
	dir := WebStateDir()
	for _, rel := range []string{
		"src/main.tsx",
		"src/styles/theme.css",
		"package.json",
		"vite.config.ts",
		"tsconfig.json",
		"index.html",
		".rick-managed",
	} {
		if _, err := os.Stat(filepath.Join(dir, rel)); err != nil {
			t.Errorf("scaffold file missing after deploy: %s (%v)", rel, err)
		}
	}

	// Second run: idempotent no-op (user customizations preserved).
	deployed2, err := DeployWebScaffold()
	if err != nil {
		t.Fatalf("second DeployWebScaffold: %v", err)
	}
	if deployed2 {
		t.Fatal("second deploy should report deployed=false (idempotent skip)")
	}
}

func TestDeployWebScaffoldPreservesUserEdits(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	if _, err := DeployWebScaffold(); err != nil {
		t.Fatalf("deploy: %v", err)
	}

	// Simulate an agent customization.
	edited := filepath.Join(WebStateDir(), "src", "main.tsx")
	if err := os.WriteFile(edited, []byte("// user-customized\n"), 0644); err != nil {
		t.Fatal(err)
	}

	if _, err := DeployWebScaffold(); err != nil {
		t.Fatalf("redeploy: %v", err)
	}
	data, err := os.ReadFile(edited)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "// user-customized\n" {
		t.Fatalf("redeploy clobbered user edit: %q", string(data))
	}
}

func TestResetWebCustomization(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	if _, err := DeployWebScaffold(); err != nil {
		t.Fatalf("deploy: %v", err)
	}
	// Simulate a built overlay + registry state that must survive.
	dir := WebStateDir()
	if err := os.MkdirAll(filepath.Join(dir, "dist"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "dist", "index.html"), []byte("overlay"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "sessions.json"), []byte(`{"version":1,"sessions":[]}`), 0644); err != nil {
		t.Fatal(err)
	}

	if err := ResetWebCustomization(); err != nil {
		t.Fatalf("reset: %v", err)
	}

	// Frontend customization layer gone...
	for _, rel := range []string{"src", "dist", "package.json", ".rick-managed"} {
		if _, err := os.Stat(filepath.Join(dir, rel)); !os.IsNotExist(err) {
			t.Errorf("reset should remove %s (stat err=%v)", rel, err)
		}
	}
	// ...but the session registry survives (contract: user data preserved).
	if _, err := os.Stat(filepath.Join(dir, "sessions.json")); err != nil {
		t.Errorf("reset must preserve sessions.json: %v", err)
	}

	// Reset on a non-existent state dir is a no-op (fresh machine).
	t.Setenv("HOME", t.TempDir())
	if err := ResetWebCustomization(); err != nil {
		t.Fatalf("reset on fresh HOME: %v", err)
	}
}

func TestCheckWebEmbed(t *testing.T) {
	// The build embeds the real web assets; this must pass in-repo.
	if err := CheckWebEmbed(); err != nil {
		t.Fatalf("CheckWebEmbed: %v", err)
	}
}
