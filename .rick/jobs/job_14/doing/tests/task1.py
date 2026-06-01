#!/usr/bin/env python3
import json
import sys
import os
import subprocess

def main():
    errors = []
    project_root = "/Users/sunquan/ai_coding/CODING/rick"

    # Test 1: go build internal/agent/... internal/actpath/...
    try:
        result = subprocess.run(
            ["go", "build", "./internal/agent/...", "./internal/actpath/..."],
            cwd=project_root,
            capture_output=True,
            text=True
        )
        if result.returncode != 0:
            errors.append(f"go build failed: {result.stderr.strip()}")
    except Exception as e:
        errors.append(f"go build error: {str(e)}")

    # Test 2: interface isolation - actpath must not import claudecode
    try:
        result = subprocess.run(
            ["grep", "-r", "claudecode", "internal/actpath/"],
            cwd=project_root,
            capture_output=True,
            text=True
        )
        if result.returncode == 0 and result.stdout.strip():
            errors.append(f"actpath imports claudecode (isolation violation): {result.stdout.strip()}")
    except Exception as e:
        errors.append(f"grep check error: {str(e)}")

    # Test 3: run generator unit tests
    generator_test = os.path.join(project_root, "internal", "actpath", "generator_test.go")
    if not os.path.exists(generator_test):
        errors.append("internal/actpath/generator_test.go does not exist")
    else:
        try:
            result = subprocess.run(
                ["go", "test", "-v", "-run", "TestGenerate", "./internal/actpath/..."],
                cwd=project_root,
                capture_output=True,
                text=True
            )
            if result.returncode != 0:
                errors.append(f"generator unit tests failed:\n{result.stdout.strip()}\n{result.stderr.strip()}")
            else:
                output = result.stdout
                for test_name in ["TestGenerate_Format", "TestGenerate_EmptyToolCalls", "TestGenerate_CreatesDir"]:
                    if f"--- PASS: {test_name}" not in output and f"=== RUN   {test_name}" not in output:
                        errors.append(f"test {test_name} not found or did not pass")
        except Exception as e:
            errors.append(f"go test error: {str(e)}")

    result = {
        "pass": len(errors) == 0,
        "errors": errors
    }

    print(json.dumps(result))
    sys.exit(0 if result["pass"] else 1)

if __name__ == "__main__":
    main()
