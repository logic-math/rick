package env

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sunquan/rick/internal/config"
)

// TestRecordSourceTreeIfRepo_CloneCwd（job_55 方案 2）：cwd 在 rick clone 内
// （含 marker）→ config source_tree 记录 clone 根；clone 子目录也记录 clone 根
// （上溯到 marker 所在目录）。
func TestRecordSourceTreeIfRepo_CloneCwd(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	repo := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repo, "super", "superopt"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "super", "superopt", "__main__.py"), []byte("#!py\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	sub := filepath.Join(repo, "scripts")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}

	t.Chdir(sub) // clone 子目录：应记录 clone 根而非子目录
	if err := recordSourceTreeIfRepo(); err != nil {
		t.Fatalf("record: %v", err)
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	// t.Chdir 会同步 PWD → os.Getwd 可能返回逻辑路径（/var/folders/…）
	// 或物理路径（/private/var/…），两种都合法（super 侧读取时 findSuperRoot
	// 会 EvalSymlinks 归一），断言接受任一形态。
	wantPhysical, err := filepath.EvalSymlinks(repo)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.SourceTree != repo && cfg.SourceTree != wantPhysical {
		t.Errorf("SourceTree = %q, want clone root %q (or %q)", cfg.SourceTree, repo, wantPhysical)
	}
}

// TestRecordSourceTreeIfRepo_NonRepoCwdUntouched：cwd 不在 rick clone 内 →
// 既有指针不动（绝不误清/误改）。
func TestRecordSourceTreeIfRepo_NonRepoCwdUntouched(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	// 预置既有指针。
	cfg := config.GetDefaultConfig()
	cfg.SourceTree = "/existing/clone"
	if err := config.SaveConfig(cfg); err != nil {
		t.Fatalf("save: %v", err)
	}

	plain := t.TempDir()
	t.Chdir(plain)
	if err := recordSourceTreeIfRepo(); err != nil {
		t.Fatalf("record: %v", err)
	}

	got, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if got.SourceTree != "/existing/clone" {
		t.Errorf("SourceTree = %q, want untouched %q", got.SourceTree, "/existing/clone")
	}
}
