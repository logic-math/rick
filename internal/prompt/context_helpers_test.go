package prompt

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadLoopsContext_MultipleFiles(t *testing.T) {
	tmpDir := t.TempDir()

	writeFile(t, filepath.Join(tmpDir, "loop1.md"), "---\nname: 调试循环\ntrigger: 当出现 bug 时\nscope: doing\n---\n\nbody\n")
	writeFile(t, filepath.Join(tmpDir, "loop2.md"), "---\nname: 代码审查\ntrigger: 当 PR 需要审查时\nscope: doing\n---\n\nbody\n")

	result := LoadLoopsContext(tmpDir)

	check(t, result, "## 可用的项目 Loops")
	check(t, result, "**调试循环**")
	check(t, result, "当出现 bug 时")
	check(t, result, "**代码审查**")
	check(t, result, "当 PR 需要审查时")
}

func TestLoadLoopsContext_EmptyDir(t *testing.T) {
	tmpDir := t.TempDir()
	result := LoadLoopsContext(tmpDir)
	check(t, result, "暂无项目 Loop 记录")
}

func TestLoadLoopsContext_NonExistentDir(t *testing.T) {
	result := LoadLoopsContext("/nonexistent/path/that/should/not/exist/abc123")
	check(t, result, "暂无项目 Loop 记录")
}

func TestLoadLoopsContext_MissingTrigger(t *testing.T) {
	tmpDir := t.TempDir()

	// File without trigger field - should be skipped
	writeFile(t, filepath.Join(tmpDir, "no_trigger.md"), "---\nname: 无触发词循环\nscope: doing\n---\n\nbody\n")
	// File with complete fields - should appear
	writeFile(t, filepath.Join(tmpDir, "complete.md"), "---\nname: 完整循环\ntrigger: 有触发词时\nscope: doing\n---\n\nbody\n")

	result := LoadLoopsContext(tmpDir)

	check(t, result, "**完整循环**")
	checkAbsent(t, result, "无触发词循环")
}

func TestLoadLoopsContext_NonMdFilesIgnored(t *testing.T) {
	tmpDir := t.TempDir()

	writeFile(t, filepath.Join(tmpDir, "loop.md"), "---\nname: 有效循环\ntrigger: 触发条件\n---\n")
	writeFile(t, filepath.Join(tmpDir, "README.txt"), "name: 无效文件\ntrigger: 不应出现\n")

	result := LoadLoopsContext(tmpDir)

	check(t, result, "**有效循环**")
	checkAbsent(t, result, "无效文件")
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func check(t *testing.T, result, want string) {
	t.Helper()
	if !strings.Contains(result, want) {
		t.Errorf("expected %q in result, got:\n%s", want, result)
	}
}

func checkAbsent(t *testing.T, result, absent string) {
	t.Helper()
	if strings.Contains(result, absent) {
		t.Errorf("expected %q to be absent, got:\n%s", absent, result)
	}
}
