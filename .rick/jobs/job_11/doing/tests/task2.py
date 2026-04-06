#!/usr/bin/env python3
import json
import sys
import os
import subprocess

def run_cmd(cmd, cwd=None):
    """Run a shell command and return (returncode, stdout, stderr)."""
    result = subprocess.run(cmd, shell=True, capture_output=True, text=True, cwd=cwd)
    return result.returncode, result.stdout, result.stderr

def main():
    errors = []

    project_root = "/Users/sunquan/ai_coding/CODING/rick"

    # Test 1: go build ./... compiles successfully
    print("Running go build...", file=sys.stderr)
    rc, stdout, stderr = run_cmd("go build ./...", cwd=project_root)
    if rc != 0:
        errors.append(f"go build ./... failed: {stderr.strip()}")

    # Test 2: go test ./internal/executor/... -v passes
    print("Running executor tests...", file=sys.stderr)
    rc, stdout, stderr = run_cmd("go test ./internal/executor/... -v -count=1", cwd=project_root)
    if rc != 0:
        errors.append(f"go test ./internal/executor/... failed:\n{stdout[-2000:] if len(stdout) > 2000 else stdout}\n{stderr[-500:] if len(stderr) > 500 else stderr}")

    # Test 3: retry.go no longer has output[:500] hard truncation
    retry_go = os.path.join(project_root, "internal", "executor", "retry.go")
    try:
        with open(retry_go, "r") as f:
            content = f.read()
        if "output[:500]" in content:
            errors.append("retry.go still contains output[:500] hard truncation — it should be removed or replaced with smart truncation")
    except Exception as e:
        errors.append(f"Failed to read retry.go: {e}")

    # Test 4: runner.go test failure path includes testOutput (not just errors join)
    runner_go = os.path.join(project_root, "internal", "executor", "runner.go")
    try:
        with open(runner_go, "r") as f:
            content = f.read()

        # The failed path (testResult.Pass == false) should reference testOutput, not only testResult.Errors
        # Check that result.Error in the failure branch contains testOutput content
        # A simple heuristic: the line setting result.Error when test fails should not be ONLY a strings.Join of errors
        # We look for testOutput being used in the failure path
        if 'strings.Join(testResult.Errors, "; ")' in content and 'testOutput' not in content.split('strings.Join(testResult.Errors')[0].split('\n')[-5:]:
            # More precise: check if the failure branch result.Error assignment uses testOutput
            import re
            # Find the block after "if testResult.Pass {"
            # The else/failure branch should reference testOutput
            fail_block_pattern = re.search(
                r'result\.Error\s*=\s*fmt\.Sprintf\([^)]*testOutput[^)]*\)',
                content
            )
            if not fail_block_pattern:
                # Also accept if result.Error is set to include testOutput in some form
                # Check if testOutput appears near result.Error assignment in the failure path
                lines = content.split('\n')
                error_lines = [i for i, l in enumerate(lines) if 'result.Error' in l and 'testOutput' not in l and 'test did not pass' in l]
                if error_lines:
                    errors.append(
                        "runner.go: test failure path sets result.Error without including testOutput — "
                        "it should include the full test output, not just strings.Join(testResult.Errors)"
                    )
    except Exception as e:
        errors.append(f"Failed to read runner.go: {e}")

    # Test 5: full test suite passes
    print("Running full test suite...", file=sys.stderr)
    rc, stdout, stderr = run_cmd("go test ./... -count=1", cwd=project_root)
    if rc != 0:
        errors.append(f"go test ./... failed:\n{stdout[-2000:] if len(stdout) > 2000 else stdout}\n{stderr[-500:] if len(stderr) > 500 else stderr}")

    result = {
        "pass": len(errors) == 0,
        "errors": errors
    }

    print(json.dumps(result))
    sys.exit(0 if result["pass"] else 1)

if __name__ == "__main__":
    main()
