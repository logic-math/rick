#!/usr/bin/env python3
import json
import sys
import os
import subprocess


def get_project_root():
    # 6 dirnames up: tests/ -> doing/ -> job_26/ -> jobs/ -> .rick/ -> project root
    path = os.path.abspath(__file__)
    for _ in range(6):
        path = os.path.dirname(path)
    return path


def run_cmd(cmd, cwd=None, env=None, timeout=120):
    try:
        result = subprocess.run(
            cmd, cwd=cwd, env=env,
            capture_output=True, timeout=timeout
        )
        stdout = result.stdout.decode("utf-8", errors="replace")
        stderr = result.stderr.decode("utf-8", errors="replace")
        return result.returncode, stdout, stderr
    except subprocess.TimeoutExpired:
        return -1, "", "command timed out"
    except Exception as e:
        return -1, "", str(e)


def main():
    errors = []
    project_root = get_project_root()
    learning_tmpl = os.path.join(
        project_root, "internal", "prompt", "templates", "learning.md"
    )
    rick_bin = os.path.join(project_root, "bin", "rick")

    # ---------------------------------------------------------------------------
    # Test 1: learning.md template contains {{draft_dir}}
    # ---------------------------------------------------------------------------
    try:
        with open(learning_tmpl, "r", encoding="utf-8") as f:
            content = f.read()
        if "{{draft_dir}}" not in content:
            errors.append("Test1: learning.md template does not contain '{{draft_dir}}'")
    except Exception as e:
        errors.append(f"Test1: cannot read learning.md: {e}")

    # ---------------------------------------------------------------------------
    # Test 2: buildLearningPrompt injects draft_dir — no unreplaced placeholder,
    #          output contains "draft" path string
    # ---------------------------------------------------------------------------
    go_test2 = '''\
package cmd

import (
\t"os"
\t"path/filepath"
\t"strings"
\t"testing"
)

func TestTask4BuildLearningPromptInjectsDraftDir(t *testing.T) {
\ttmpDir := t.TempDir()
\trickDir := filepath.Join(tmpDir, ".rick")
\tdoingDir := filepath.Join(rickDir, "jobs", "job_test", "doing")
\tif err := os.MkdirAll(doingDir, 0755); err != nil {
\t\tt.Fatalf("MkdirAll doingDir: %v", err)
\t}

\t// Chdir so workspace.GetRickDir() resolves to this tmpDir
\toldDir, _ := os.Getwd()
\tif err := os.Chdir(tmpDir); err != nil {
\t\tt.Fatalf("Chdir: %v", err)
\t}
\tdefer os.Chdir(oldDir)

\tdata := &ExecutionData{
\t\tJobID:        "job_test",
\t\tRickDir:      rickDir,
\t\tTaskMDPaths:  []string{},
\t\tActPathFiles: []string{},
\t}

\tlearningDir := filepath.Join(rickDir, "jobs", "job_test", "learning")
\tpromptsDir := filepath.Join(learningDir, "prompts")
\tif err := os.MkdirAll(promptsDir, 0755); err != nil {
\t\tt.Fatalf("MkdirAll promptsDir: %v", err)
\t}

\tpromptFile, err := buildLearningPrompt(data, learningDir, promptsDir)
\tif err != nil {
\t\tt.Fatalf("buildLearningPrompt error: %v", err)
\t}

\tcontent, err := os.ReadFile(promptFile)
\tif err != nil {
\t\tt.Fatalf("cannot read prompt file %s: %v", promptFile, err)
\t}

\ts := string(content)
\tif strings.Contains(s, "{{draft_dir}}") {
\t\tt.Errorf("prompt contains unreplaced {{draft_dir}}")
\t}
\texpectedDraftPath := filepath.Join(rickDir, "draft")
\tif !strings.Contains(s, expectedDraftPath) {
\t\tsnippet := s
\t\tif len(snippet) > 300 {
\t\t\tsnippet = snippet[:300]
\t\t}
\t\tt.Errorf("prompt does not contain draft path %q; snippet: %s", expectedDraftPath, snippet)
\t}
}
'''
    # Note: Go ignores files starting with '_', so use a regular name
    test_file2 = os.path.join(
        project_root, "internal", "cmd", "pytask4draft_test.go"
    )
    try:
        with open(test_file2, "w", encoding="utf-8") as f:
            f.write(go_test2)
        rc, stdout, stderr = run_cmd(
            ["go", "test", "./internal/cmd/...", "-run", "TestTask4BuildLearningPromptInjectsDraftDir", "-v"],
            cwd=project_root, timeout=60
        )
        if rc != 0:
            errors.append(
                f"Test2: buildLearningPrompt draft_dir injection failed:\n{stdout[-500:]}{stderr[-300:]}"
            )
    except Exception as e:
        errors.append(f"Test2: go test write/run error: {e}")
    finally:
        if os.path.exists(test_file2):
            os.remove(test_file2)


    # ---------------------------------------------------------------------------
    # Test 3: ./bin/rick learning job_N --dry-run stdout contains no {{draft_dir}}
    # ---------------------------------------------------------------------------
    import tempfile
    with tempfile.TemporaryDirectory() as tmpdir:
        rick_dir_tmp = os.path.join(tmpdir, ".rick")
        os.makedirs(os.path.join(rick_dir_tmp, "loops"), exist_ok=True)
        os.makedirs(os.path.join(rick_dir_tmp, "skills"), exist_ok=True)
        rc, stdout, stderr = run_cmd(
            [rick_bin, "learning", "job_N", "--dry-run"],
            cwd=tmpdir, timeout=30
        )
        print(f"[DEBUG] dry-run rc={rc}", file=sys.stderr)
        print(f"[DEBUG] dry-run stdout[:200]={stdout[:200]}", file=sys.stderr)
        if rc != 0 and "{{draft_dir}}" not in stdout:
            # Non-zero exit is acceptable for dry-run with missing job
            pass
        if "{{draft_dir}}" in stdout:
            errors.append("Test3: dry-run stdout contains unreplaced '{{draft_dir}}'")

    result = {"pass": len(errors) == 0, "errors": errors}
    print(json.dumps(result, ensure_ascii=False))
    sys.exit(0 if result["pass"] else 1)


if __name__ == "__main__":
    main()
