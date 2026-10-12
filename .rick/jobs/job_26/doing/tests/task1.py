#!/usr/bin/env python3
import json
import sys
import os
import subprocess
import tempfile
import shutil

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
            capture_output=True, text=True, timeout=timeout
        )
        return result.returncode, result.stdout, result.stderr
    except subprocess.TimeoutExpired:
        return -1, "", "command timed out"
    except Exception as e:
        return -1, "", str(e)

def main():
    errors = []
    project_root = get_project_root()
    paths_go = os.path.join(project_root, "internal", "workspace", "paths.go")
    human_loop_go = os.path.join(project_root, "internal", "cmd", "human_loop.go")
    human_loop_prompt_go = os.path.join(project_root, "internal", "prompt", "human_loop_prompt.go")
    rick_bin = os.path.join(project_root, "bin", "rick")

    # Build binary before integration tests
    build_rc, _, build_err = run_cmd(
        ["bash", os.path.join(project_root, "scripts", "build.sh")],
        cwd=project_root, timeout=120
    )
    if build_rc != 0:
        errors.append(f"build.sh failed (exit {build_rc}): {build_err[:300]}")
        print(json.dumps({"pass": False, "errors": errors}, ensure_ascii=False))
        sys.exit(1)

    # ---------------------------------------------------------------------------
    # Test 1: workspace.GetDraftDir() returns {tmpDir}/.rick/draft, error nil
    # ---------------------------------------------------------------------------

    # 1a: Source-level check — function must be declared in paths.go
    try:
        with open(paths_go, "r") as f:
            paths_content = f.read()
        if "func GetDraftDir()" not in paths_content:
            errors.append("Test1: GetDraftDir() not defined in internal/workspace/paths.go")
    except Exception as e:
        errors.append(f"Test1: cannot read paths.go: {e}")

    # 1b: Functional check via inline Go test written to the workspace package
    go_test1 = '''\
package workspace_test

import (
\t"os"
\t"path/filepath"
\t"testing"

\t"github.com/sunquan/rick/internal/workspace"
)

func TestGetDraftDirReturnValue(t *testing.T) {
\ttmpDir := t.TempDir()
\tif err := os.MkdirAll(filepath.Join(tmpDir, ".rick"), 0755); err != nil {
\t\tt.Fatal(err)
\t}
\torig, _ := os.Getwd()
\tdefer os.Chdir(orig)
\tif err := os.Chdir(tmpDir); err != nil {
\t\tt.Fatal(err)
\t}
\tdir, err := workspace.GetDraftDir()
\tif err != nil {
\t\tt.Fatalf("GetDraftDir() error: %v", err)
\t}
\texpected := filepath.Join(tmpDir, ".rick", "draft")
\tif dir != expected {
\t\tt.Errorf("GetDraftDir() = %q, want %q", dir, expected)
\t}
}
'''
    test_file1 = os.path.join(project_root, "internal", "workspace", "_py_task1_draft_test.go")
    try:
        with open(test_file1, "w") as f:
            f.write(go_test1)
        rc, stdout, stderr = run_cmd(
            ["go", "test", "./internal/workspace/...", "-run", "TestGetDraftDirReturnValue", "-v"],
            cwd=project_root, timeout=60
        )
        if rc != 0:
            errors.append(f"Test1: GetDraftDir() functional test failed:\n{stdout[-400:]}{stderr[-200:]}")
    except Exception as e:
        errors.append(f"Test1: go test write/run error: {e}")
    finally:
        if os.path.exists(test_file1):
            os.remove(test_file1)

    # ---------------------------------------------------------------------------
    # Test 2: rick human-loop creates draft/, draft/concepts/, draft/human-learning/
    # ---------------------------------------------------------------------------

    # 2a: Source-level check — human_loop.go must reference draft dir creation
    try:
        with open(human_loop_go, "r") as f:
            hl_content = f.read()
        if "draft" not in hl_content.lower():
            errors.append("Test2: human_loop.go has no reference to draft directory creation")
    except Exception as e:
        errors.append(f"Test2: cannot read human_loop.go: {e}")

    # 2b: Integration test — run binary with mock claude, verify draft dirs exist
    work_dir2 = tempfile.mkdtemp()
    try:
        rick_dir2 = os.path.join(work_dir2, ".rick")
        os.makedirs(rick_dir2)

        mock_claude = os.path.join(work_dir2, "mock_claude")
        with open(mock_claude, "w") as f:
            f.write("#!/bin/sh\nexit 0\n")
        os.chmod(mock_claude, 0o755)

        cfg = {"claude_code_path": mock_claude}
        with open(os.path.join(rick_dir2, "config.json"), "w") as f:
            json.dump(cfg, f)

        env2 = os.environ.copy()
        env2["HOME"] = work_dir2
        rc, stdout, stderr = run_cmd(
            [rick_bin, "human-loop", "测试主题"],
            cwd=work_dir2, env=env2, timeout=30
        )
        if rc != 0:
            errors.append(f"Test2: rick human-loop exit {rc}: {stderr[:300]}")
        else:
            draft_root = os.path.join(work_dir2, ".rick", "draft")
            for d in [
                draft_root,
                os.path.join(draft_root, "concepts"),
                os.path.join(draft_root, "human-learning"),
            ]:
                if not os.path.isdir(d):
                    errors.append(f"Test2: expected directory not created: {d}")
    except Exception as e:
        errors.append(f"Test2: integration test error: {e}")
    finally:
        shutil.rmtree(work_dir2, ignore_errors=True)

    # ---------------------------------------------------------------------------
    # Test 3: dry-run output contains 'draft' path, no {{draft_dir}} placeholder
    # ---------------------------------------------------------------------------

    work_dir3 = tempfile.mkdtemp()
    try:
        os.makedirs(os.path.join(work_dir3, ".rick"))

        rc, stdout, stderr = run_cmd(
            [rick_bin, "human-loop", "--dry-run", "测试主题"],
            cwd=work_dir3, timeout=30
        )
        if rc != 0:
            errors.append(f"Test3: dry-run exit {rc}: {stderr[:300]}")
        else:
            if "draft" not in stdout:
                errors.append("Test3: dry-run stdout does not contain 'draft' path")
            if "{{draft_dir}}" in stdout:
                errors.append("Test3: dry-run stdout contains unreplaced {{draft_dir}} placeholder")
    except Exception as e:
        errors.append(f"Test3: dry-run test error: {e}")
    finally:
        shutil.rmtree(work_dir3, ignore_errors=True)

    # ---------------------------------------------------------------------------
    # Test 4: GenerateHumanLoopPromptFile(topic, rfcDir, draftDir, pm) injects
    #         draftDir into think/express sub-agent files, no {{draft_dir}} left
    # ---------------------------------------------------------------------------

    # 4a: Source-level check — function must accept draftDir parameter
    try:
        with open(human_loop_prompt_go, "r") as f:
            hlp_content = f.read()
        if "draftDir" not in hlp_content:
            errors.append(
                "Test4: GenerateHumanLoopPromptFile does not have draftDir parameter "
                "in internal/prompt/human_loop_prompt.go"
            )
    except Exception as e:
        errors.append(f"Test4: cannot read human_loop_prompt.go: {e}")

    # 4b: Functional check — inline Go test calling new 4-arg signature
    go_test4 = '''\
package prompt_test

import (
\t"os"
\t"strings"
\t"testing"

\t"github.com/sunquan/rick/internal/prompt"
)

func TestGenerateHumanLoopPromptFileDraftDirInjection(t *testing.T) {
\ttmpDir := t.TempDir()
\trfcDir := tmpDir
\tdraftDir := "/tmp/test-draft-py-task1"

\tpm := prompt.NewPromptManager()
\tmainFile, subFiles, err := prompt.GenerateHumanLoopPromptFile("topic", rfcDir, draftDir, pm)
\tif err != nil {
\t\tt.Fatalf("GenerateHumanLoopPromptFile error: %v", err)
\t}
\tdefer func() {
\t\tos.Remove(mainFile)
\t\tfor _, f := range subFiles {
\t\t\tos.Remove(f)
\t\t}
\t}()

\tfor _, f := range subFiles {
\t\tif strings.Contains(f, "learn") {
\t\t\tcontinue // learn sub-agent may not reference draft_dir
\t\t}
\t\tcontent, err := os.ReadFile(f)
\t\tif err != nil {
\t\t\tt.Errorf("cannot read sub-agent file %s: %v", f, err)
\t\t\tcontinue
\t\t}
\t\tif !strings.Contains(string(content), draftDir) {
\t\t\tt.Errorf("sub-agent file %s missing draftDir %q", f, draftDir)
\t\t}
\t\tif strings.Contains(string(content), "{{draft_dir}}") {
\t\t\tt.Errorf("sub-agent file %s has unreplaced {{draft_dir}}", f)
\t\t}
\t}
}
'''
    test_file4 = os.path.join(project_root, "internal", "prompt", "_py_task1_draftdir_test.go")
    try:
        with open(test_file4, "w") as f:
            f.write(go_test4)
        rc, stdout, stderr = run_cmd(
            ["go", "test", "./internal/prompt/...", "-run", "TestGenerateHumanLoopPromptFileDraftDirInjection", "-v"],
            cwd=project_root, timeout=60
        )
        if rc != 0:
            errors.append(f"Test4: draftDir injection test failed:\n{stdout[-400:]}{stderr[-200:]}")
    except Exception as e:
        errors.append(f"Test4: go test write/run error: {e}")
    finally:
        if os.path.exists(test_file4):
            os.remove(test_file4)

    result = {"pass": len(errors) == 0, "errors": errors}
    print(json.dumps(result, ensure_ascii=False))
    sys.exit(0 if result["pass"] else 1)

if __name__ == "__main__":
    main()
