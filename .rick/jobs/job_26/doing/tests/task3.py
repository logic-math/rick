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
    express_tmpl = os.path.join(
        project_root, "internal", "prompt", "templates", "human_loop_express.md"
    )

    # ---------------------------------------------------------------------------
    # Test 1: human_loop_express.md contains judgment.md, 清洗/review, {{draft_dir}}
    # ---------------------------------------------------------------------------
    try:
        with open(express_tmpl, "r", encoding="utf-8") as f:
            content = f.read()
        if "judgment.md" not in content:
            errors.append("Test1: human_loop_express.md missing keyword: 'judgment.md'")
        if "清洗" not in content and "review" not in content:
            errors.append("Test1: human_loop_express.md missing '清洗' or 'review'")
        if "{{draft_dir}}" not in content:
            errors.append("Test1: human_loop_express.md missing '{{draft_dir}}'")
    except Exception as e:
        errors.append(f"Test1: cannot read human_loop_express.md: {e}")

    # ---------------------------------------------------------------------------
    # Test 2: human_loop_express.md contains progress.md, ZPD/难度感受, loops.md
    # ---------------------------------------------------------------------------
    try:
        with open(express_tmpl, "r", encoding="utf-8") as f:
            content = f.read()
        if "progress.md" not in content:
            errors.append("Test2: human_loop_express.md missing 'progress.md'")
        if "ZPD" not in content and "难度感受" not in content:
            errors.append("Test2: human_loop_express.md missing 'ZPD' or '难度感受'")
        if "loops.md" not in content:
            errors.append("Test2: human_loop_express.md missing 'loops.md'")
    except Exception as e:
        errors.append(f"Test2: cannot read human_loop_express.md: {e}")

    # ---------------------------------------------------------------------------
    # Test 3: GenerateHumanLoopPromptFile injects draftDir into express file,
    #          no {{draft_dir}} placeholder remains
    # ---------------------------------------------------------------------------
    go_test3 = '''\
package prompt_test

import (
\t"os"
\t"strings"
\t"testing"

\t"github.com/sunquan/rick/internal/prompt"
)

func TestTask3ExpressFileInjectsDraftDir(t *testing.T) {
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
\t\tif !strings.Contains(f, "express") {
\t\t\tcontinue
\t\t}
\t\tcontent, err := os.ReadFile(f)
\t\tif err != nil {
\t\t\tt.Fatalf("cannot read express file %s: %v", f, err)
\t\t}
\t\ts := string(content)
\t\tif strings.Contains(s, "{{draft_dir}}") {
\t\t\tt.Errorf("express file %s still contains unreplaced {{draft_dir}}", f)
\t\t}
\t\tif !strings.Contains(s, draftDir) {
\t\t\tt.Errorf("express file %s does not contain draftDir %q", f, draftDir)
\t\t}
\t}
}
'''
    test_file3 = os.path.join(
        project_root, "internal", "prompt", "_py_task3_express_inject_test.go"
    )
    try:
        with open(test_file3, "w", encoding="utf-8") as f:
            f.write(go_test3)
        rc, stdout, stderr = run_cmd(
            ["go", "test", "./internal/prompt/...", "-run", "TestTask3ExpressFileInjectsDraftDir", "-v"],
            cwd=project_root, timeout=60
        )
        if rc != 0:
            errors.append(
                f"Test3: express draftDir injection failed:\n{stdout[-400:]}{stderr[-200:]}"
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
