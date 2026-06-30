#!/usr/bin/env python3
# Description: task3 test - verify LoadLoopsContext() implementation
import json
import sys
import os
import subprocess

GO_TEST_CONTENT = r'''package prompt

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
'''


def main():
    errors = []

    script_path = os.path.abspath(__file__)
    project_root = script_path
    for _ in range(6):
        project_root = os.path.dirname(project_root)

    print(f"project_root: {project_root}", file=sys.stderr)

    test_file = os.path.join(project_root, "internal", "prompt", "context_helpers_test.go")

    # Write Go test file
    try:
        with open(test_file, "w", encoding="utf-8") as f:
            f.write(GO_TEST_CONTENT)
        print(f"Written Go test file: {test_file}", file=sys.stderr)
    except Exception as e:
        errors.append(f"Failed to write Go test file: {str(e)}")
        print(json.dumps({"pass": False, "errors": errors}, ensure_ascii=False))
        sys.exit(1)

    # Run go test on the prompt package (scope: only internal/prompt)
    try:
        proc = subprocess.run(
            ["go", "test", "-v", "-run", "TestLoadLoopsContext", "./internal/prompt/..."],
            cwd=project_root,
            capture_output=True,
            text=True,
            timeout=120,
        )

        print("go test stdout:", file=sys.stderr)
        print(proc.stdout, file=sys.stderr)
        print("go test stderr:", file=sys.stderr)
        print(proc.stderr, file=sys.stderr)

        if proc.returncode != 0:
            combined = (proc.stdout + proc.stderr).strip()
            errors.append(f"go test failed (exit {proc.returncode}): {combined}")
    except subprocess.TimeoutExpired:
        errors.append("go test timed out after 120 seconds")
    except Exception as e:
        errors.append(f"Failed to run go test: {str(e)}")

    result = {"pass": len(errors) == 0, "errors": errors}
    print(json.dumps(result, ensure_ascii=False))
    sys.exit(0 if result["pass"] else 1)


if __name__ == "__main__":
    main()
