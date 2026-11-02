package super

import (
	"os"
	"path/filepath"
	"testing"
)

// TestResolveRepoRootExeFallback（job_47 现场回归）：cwd 不含 super/superopt 时
// （外部 workspace 场景：MT_INFER），必须回退到 rick 二进制自身所在 repo。
// 复现方式：临时目录 + symlink 链模拟 <repo>/bin/rick → releases/current/rick。
func TestResolveRepoRootExeFallback(t *testing.T) {

	// 构造假 repo：/tmp/fakerepo/{super/superopt/__main__.py, releases/v1/rick(空文件), bin/rick->releases/current/rick, releases/current->v1}
	tmp := t.TempDir()
	repo := filepath.Join(tmp, "fakerepo")
	if err := os.MkdirAll(filepath.Join(repo, "super", "superopt"), 0o755); err != nil { t.Fatal(err) }
	if err := os.MkdirAll(filepath.Join(repo, "releases", "v1"), 0o755); err != nil { t.Fatal(err) }
	if err := os.MkdirAll(filepath.Join(repo, "bin"), 0o755); err != nil { t.Fatal(err) }
	if err := os.WriteFile(filepath.Join(repo, "super", "superopt", "__main__.py"), []byte("#!py\n"), 0o644); err != nil { t.Fatal(err) }
	if err := os.WriteFile(filepath.Join(repo, "releases", "v1", "rick"), []byte("#!fake\n"), 0o755); err != nil { t.Fatal(err) }
	if err := os.Symlink("v1", filepath.Join(repo, "releases", "current")); err != nil { t.Fatal(err) }
	if err := os.Symlink(filepath.Join(repo, "releases", "current", "rick"), filepath.Join(repo, "bin", "rick")); err != nil { t.Fatal(err) }

	// cwd = 无 super 的外部目录（模拟 MT_INFER）
	ext := filepath.Join(tmp, "external_ws")
	if err := os.MkdirAll(ext, 0o755); err != nil { t.Fatal(err) }
	oldWd, _ := os.Getwd()
	if err := os.Chdir(ext); err != nil { t.Fatal(err) }
	defer os.Chdir(oldWd)

	// cwd 上溯（tmp 根）找不到 marker → 期望 exe fallback…但我们无法改 os.Executable。
	// 退而验证组件：findSuperRoot 对 exe 路径（含 symlink 链）解析正确。
	got := findSuperRoot(filepath.Join(repo, "bin", "rick"))
	// macOS /var ↔ /private/var 双形：EvalSymlinks 归一后比较（release.go 同型处理）
	want, _ := filepath.EvalSymlinks(repo)
	if got != want {
		t.Fatalf("findSuperRoot(exe-symlink) = %q, want %q", got, want)
	}
	// cwd 路径直接命中（repo 内部场景不变）
	got2 := findSuperRoot(ext)
	if got2 != "" { t.Fatalf("findSuperRoot(external) = %q, want empty", got2) }
}
