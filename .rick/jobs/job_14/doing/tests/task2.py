#!/usr/bin/env python3
import json
import sys
import os
import subprocess

def main():
    errors = []
    project_root = "/Users/sunquan/ai_coding/CODING/rick"

    # Test 1: go build ./internal/agent/claudecode/...
    try:
        result = subprocess.run(
            ["go", "build", "./internal/agent/claudecode/..."],
            cwd=project_root,
            capture_output=True,
            text=True
        )
        if result.returncode != 0:
            errors.append(f"go build failed: {result.stderr.strip()}")
    except Exception as e:
        errors.append(f"go build error: {str(e)}")

    # Test 2: go test ./internal/agent/claudecode/... -v
    if not errors:
        try:
            result = subprocess.run(
                ["go", "test", "./internal/agent/claudecode/...", "-v", "-run", "TestExecute_ParseNDJSON|TestExecute_SkipNonJSON"],
                cwd=project_root,
                capture_output=True,
                text=True,
                timeout=120
            )
            output = result.stdout + result.stderr

            if result.returncode != 0:
                errors.append(f"go test failed (exit {result.returncode}): {output[-2000:]}")
            else:
                # Check TestExecute_ParseNDJSON passed
                if "--- PASS: TestExecute_ParseNDJSON" not in output:
                    errors.append("TestExecute_ParseNDJSON did not pass")
                # Check TestExecute_SkipNonJSON passed
                if "--- PASS: TestExecute_SkipNonJSON" not in output:
                    errors.append("TestExecute_SkipNonJSON did not pass")
        except subprocess.TimeoutExpired:
            errors.append("go test timed out after 120s")
        except Exception as e:
            errors.append(f"go test error: {str(e)}")

    # Test 3: verify executor.go exists
    executor_path = os.path.join(project_root, "internal/agent/claudecode/executor.go")
    if not os.path.exists(executor_path):
        errors.append(f"executor.go does not exist at {executor_path}")

    result = {
        "pass": len(errors) == 0,
        "errors": errors
    }

    print(json.dumps(result))
    sys.exit(0 if result["pass"] else 1)

if __name__ == "__main__":
    main()
