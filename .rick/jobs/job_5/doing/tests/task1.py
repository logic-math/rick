#!/usr/bin/env python3
import json
import sys
import os
import subprocess

def run_cmd(cmd, cwd=None):
    result = subprocess.run(cmd, shell=True, capture_output=True, text=True, cwd=cwd)
    return result.returncode, result.stdout, result.stderr

def main():
    errors = []

    project_root = "/Users/sunquan/ai_coding/CODING/rick"

    # Test 1: go build ./...
    rc, stdout, stderr = run_cmd("go build ./...", cwd=project_root)
    if rc != 0:
        errors.append(f"go build failed: {stderr.strip()}")

    # Test 2: go test ./...
    rc, stdout, stderr = run_cmd("go test ./...", cwd=project_root)
    if rc != 0:
        errors.append(f"go test failed: {stderr.strip() or stdout.strip()}")

    # Test 3: no go-git in go.mod/go.sum
    rc, stdout, stderr = run_cmd("grep -r 'go-git' go.mod go.sum", cwd=project_root)
    if rc == 0 and stdout.strip():
        errors.append(f"go-git still present in go.mod/go.sum: {stdout.strip()}")

    # Test 4: no pkg/feedback in internal/
    rc, stdout, stderr = run_cmd("grep -r 'pkg/feedback' internal/", cwd=project_root)
    if rc == 0 and stdout.strip():
        errors.append(f"pkg/feedback still referenced in internal/: {stdout.strip()}")

    # Test 5: pkg/feedback directory does not exist
    feedback_dir = os.path.join(project_root, "pkg", "feedback")
    if os.path.isdir(feedback_dir):
        errors.append("pkg/feedback directory still exists")

    # Test 6: workspace.GetProjectName() is dynamic (not hardcoded "Rick CLI")
    # Check that the source does not hardcode "Rick CLI" in prompt templates or workspace
    rc, stdout, stderr = run_cmd(
        'grep -r \'"Rick CLI"\' internal/ --include="*.go"',
        cwd=project_root
    )
    if rc == 0 and stdout.strip():
        errors.append(f'Hardcoded "Rick CLI" still found in internal/: {stdout.strip()}')

    # Test 7: run unit test for GetProjectName if it exists
    rc, stdout, stderr = run_cmd(
        "go test ./internal/workspace/... -run TestGetProjectName -v",
        cwd=project_root
    )
    if rc != 0:
        # Only fail if the test exists but fails; if no test found, skip
        if "no test files" not in stderr and "no tests to run" not in stdout and "no tests to run" not in stderr:
            if "FAIL" in stdout or "FAIL" in stderr:
                errors.append(f"TestGetProjectName failed: {stdout.strip()}")
        print(f"TestGetProjectName skipped or not found: {stderr.strip()}", file=sys.stderr)

    result = {
        'pass': len(errors) == 0,
        'errors': errors
    }

    print(json.dumps(result))
    sys.exit(0 if result['pass'] else 1)

if __name__ == '__main__':
    main()
