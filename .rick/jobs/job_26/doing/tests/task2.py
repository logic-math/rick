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
    think_tmpl = os.path.join(
        project_root, "internal", "prompt", "templates", "human_loop_think.md"
    )

    # ---------------------------------------------------------------------------
    # Test 1: human_loop_think.md contains judgment.md, {{draft_dir}}, 判断记录协议
    # ---------------------------------------------------------------------------
    try:
        with open(think_tmpl, "r", encoding="utf-8") as f:
            content = f.read()
        for keyword in ["judgment.md", "{{draft_dir}}", "判断记录协议"]:
            if keyword not in content:
                errors.append(f"Test1: human_loop_think.md missing keyword: {keyword!r}")
    except Exception as e:
        errors.append(f"Test1: cannot read human_loop_think.md: {e}")

    # ---------------------------------------------------------------------------
    # Test 2: human_loop_think.md contains loops.md and four-field structure keywords
    # ---------------------------------------------------------------------------
    try:
        with open(think_tmpl, "r", encoding="utf-8") as f:
            content = f.read()
        # loops.md must be present
        if "loops.md" not in content:
            errors.append("Test2: human_loop_think.md missing 'loops.md'")
        # Four-field structure: 做什么 / 难度感受, 前置依赖, 掌握程度
        # 做什么 or 难度感受 must appear (accept either)
        if "做什么" not in content and "难度感受" not in content:
            errors.append("Test2: human_loop_think.md missing '做什么' or '难度感受' (loops.md field)")
        for field in ["前置依赖", "掌握程度"]:
            if field not in content:
                errors.append(f"Test2: human_loop_think.md missing loops.md field: {field!r}")
    except Exception as e:
        errors.append(f"Test2: cannot read human_loop_think.md: {e}")

    # ---------------------------------------------------------------------------
    # Test 3: GenerateHumanLoopPromptFile injects draftDir into think file,
    #          no {{draft_dir}} placeholder remains in the think sub-agent file
    # ---------------------------------------------------------------------------
    go_test3 = '''\
package prompt_test

import (
\t"os"
\t"strings"
\t"testing"

\t"github.com/sunquan/rick/internal/prompt"
)

func TestTask2ThinkFileInjectsDraftDir(t *testing.T) {
\ttmpDir := t.TempDir()
\tdraftDir := "/tmp/test-draft"

\tpm := prompt.NewPromptManager()
\t_, subFiles, err := prompt.GenerateHumanLoopPromptFile("topic", tmpDir, draftDir, pm)
\tif err != nil {
\t\tt.Fatalf("GenerateHumanLoopPromptFile error: %v", err)
\t}
\tdefer func() {
\t\tfor _, f := range subFiles {
\t\t\tos.Remove(f)
\t\t}
\t}()

\tfor _, f := range subFiles {
\t\tif !strings.Contains(f, "think") {
\t\t\tcontinue
\t\t}
\t\tcontent, err := os.ReadFile(f)
\t\tif err != nil {
\t\t\tt.Fatalf("cannot read think file %s: %v", f, err)
\t\t}
\t\ts := string(content)
\t\tif strings.Contains(s, "{{draft_dir}}") {
\t\t\tt.Errorf("think file %s still contains unreplaced {{draft_dir}}", f)
\t\t}
\t\tif !strings.Contains(s, draftDir) {
\t\t\tt.Errorf("think file %s does not contain draftDir %q", f, draftDir)
\t\t}
\t}
}
'''
    test_file3 = os.path.join(
        project_root, "internal", "prompt", "_py_task2_think_inject_test.go"
    )
    try:
        with open(test_file3, "w", encoding="utf-8") as f:
            f.write(go_test3)
        rc, stdout, stderr = run_cmd(
            ["go", "test", "./internal/prompt/...", "-run", "TestTask2ThinkFileInjectsDraftDir", "-v"],
            cwd=project_root, timeout=60
        )
        if rc != 0:
            errors.append(
                f"Test3: think draftDir injection failed:\n{stdout[-400:]}{stderr[-200:]}"
            )
    except Exception as e:
        errors.append(f"Test3: go test write/run error: {e}")
    finally:
        if os.path.exists(test_file3):
            os.remove(test_file3)

    result = {"pass": len(errors) == 0, "errors": errors}
    print(json.dumps(result, ensure_ascii=False))
    sys.exit(0 if result["pass"] else 1)

if __name__ == "__main__":
    main()
