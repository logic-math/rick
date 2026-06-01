#!/usr/bin/env python3
import json
import sys
import os
import subprocess
import tempfile
import shutil

PROJECT_ROOT = "/Users/sunquan/ai_coding/CODING/rick"

def main():
    errors = []

    # Test 1: Build
    try:
        result = subprocess.run(
            ["python3", "tools/build_and_get_rick_bin.py"],
            cwd=PROJECT_ROOT,
            capture_output=True, text=True, timeout=120
        )
        if result.returncode != 0:
            errors.append(f"Build failed: {result.stderr[:500]}")
    except Exception as e:
        errors.append(f"Build error: {str(e)}")

    # Test 2: collectActPathContent unit test
    try:
        result = subprocess.run(
            ["go", "test", "./...", "-v", "-run", "TestCollectActPathContent"],
            cwd=PROJECT_ROOT,
            capture_output=True, text=True, timeout=120
        )
        if result.returncode != 0:
            errors.append(f"TestCollectActPathContent failed: {result.stdout[-500:]} {result.stderr[-300:]}")
    except Exception as e:
        errors.append(f"TestCollectActPathContent error: {str(e)}")

    # Test 3: learning prompt must NOT contain "skill:tdd"
    try:
        result = subprocess.run(
            ["python3", "tools/check_prompt_variables.py", "--phase", "learning", "--keywords", "skill:tdd"],
            cwd=PROJECT_ROOT,
            capture_output=True, text=True, timeout=30
        )
        combined = result.stdout + result.stderr
        if "关键词未找到" not in combined and "not found" not in combined.lower():
            errors.append(f"Learning prompt pollution check failed: {combined[:300]}")
    except Exception as e:
        errors.append(f"check_prompt_variables error: {str(e)}")

    # Test 4: TestDreamDir workspace test
    try:
        result = subprocess.run(
            ["go", "test", "./internal/workspace/...", "-v", "-run", "TestDreamDir"],
            cwd=PROJECT_ROOT,
            capture_output=True, text=True, timeout=60
        )
        if result.returncode != 0:
            errors.append(f"TestDreamDir failed: {result.stdout[-500:]} {result.stderr[-300:]}")
    except Exception as e:
        errors.append(f"TestDreamDir error: {str(e)}")

    # Test 5: learning.md contains gen-skill reference
    learning_tpl = os.path.join(PROJECT_ROOT, "internal/prompt/templates/learning.md")
    try:
        with open(learning_tpl, "r") as f:
            content = f.read()
        if "gen-skill" not in content:
            errors.append("learning.md missing gen-skill reference")
        if "act-path" not in content:
            errors.append("learning.md missing act-path reference")
        if "run_log" not in content:
            errors.append("learning.md missing run_log reference")
    except Exception as e:
        errors.append(f"Failed to read learning.md: {str(e)}")

    # Test 6: Full test suite - no new failures
    try:
        result = subprocess.run(
            ["go", "test", "./..."],
            cwd=PROJECT_ROOT,
            capture_output=True, text=True, timeout=180
        )
        if result.returncode != 0:
            errors.append(f"go test ./... failed: {result.stdout[-500:]} {result.stderr[-300:]}")
    except Exception as e:
        errors.append(f"go test ./... error: {str(e)}")

    result = {
        "pass": len(errors) == 0,
        "errors": errors
    }

    print(json.dumps(result))
    sys.exit(0 if result["pass"] else 1)

if __name__ == "__main__":
    main()
