#!/usr/bin/env python3
import json
import sys
import os
import subprocess

def main():
    errors = []

    # Project root: 6 x dirname from __file__ (task3.py → tests → doing → job_16 → jobs → .rick → rick/)
    p = os.path.abspath(__file__)
    for _ in range(6):
        p = os.path.dirname(p)
    project_root = p

    prompt_dir = os.path.join(project_root, "internal", "prompt")
    files_to_check = [
        os.path.join(prompt_dir, "doing_prompt.go"),
        os.path.join(prompt_dir, "plan_prompt.go"),
        os.path.join(prompt_dir, "easy_prompt.go"),
    ]
    manager_test = os.path.join(prompt_dir, "manager_test.go")

    OLD_PATTERNS = ["super-debugging-zh", "super_debugging_zh", "super_debugging_path",
                    "super_debugging_skill_path", "super-debugging"]

    # Test 1: No residual super-debugging references in prompt Go files
    for fpath in files_to_check:
        if not os.path.exists(fpath):
            errors.append(f"File not found: {fpath}")
            continue
        try:
            with open(fpath, "r", encoding="utf-8") as f:
                content = f.read()
            for pat in OLD_PATTERNS:
                if pat in content:
                    errors.append(f"{os.path.basename(fpath)} still contains old pattern: '{pat}'")
        except Exception as e:
            errors.append(f"Failed to read {fpath}: {e}")

    # Test 2: debug_skill references exist in doing_prompt.go and easy_prompt.go
    for fname in ["doing_prompt.go", "easy_prompt.go"]:
        fpath = os.path.join(prompt_dir, fname)
        if not os.path.exists(fpath):
            errors.append(f"File not found: {fpath}")
            continue
        try:
            with open(fpath, "r", encoding="utf-8") as f:
                content = f.read()
            if "debug_skill" not in content and "debug-skill" not in content:
                errors.append(f"{fname} missing 'debug_skill'/'debug-skill' reference")
        except Exception as e:
            errors.append(f"Failed to read {fpath}: {e}")

    # Test 3: debug_skill_path reference exists in plan_prompt.go
    plan_path = os.path.join(prompt_dir, "plan_prompt.go")
    if os.path.exists(plan_path):
        try:
            with open(plan_path, "r", encoding="utf-8") as f:
                content = f.read()
            if "debug_skill_path" not in content:
                errors.append("plan_prompt.go missing 'debug_skill_path' reference")
        except Exception as e:
            errors.append(f"Failed to read plan_prompt.go: {e}")

    # Test 4: manager_test.go updated - no old super-debugging-zh, has debug_skill
    if os.path.exists(manager_test):
        try:
            with open(manager_test, "r", encoding="utf-8") as f:
                content = f.read()
            for pat in OLD_PATTERNS:
                if pat in content:
                    errors.append(f"manager_test.go still contains old pattern: '{pat}'")
            if "debug_skill" not in content and "debug-skill" not in content:
                errors.append("manager_test.go missing 'debug_skill'/'debug-skill' reference")
        except Exception as e:
            errors.append(f"Failed to read manager_test.go: {e}")
    else:
        errors.append(f"manager_test.go not found: {manager_test}")

    # Test 5: go test ./internal/prompt/... passes
    try:
        result = subprocess.run(
            ["go", "test", "./internal/prompt/..."],
            cwd=project_root,
            capture_output=True,
            text=True,
            timeout=120,
        )
        if result.returncode != 0:
            errors.append(f"go test ./internal/prompt/... FAILED:\n{result.stdout}\n{result.stderr}")
    except Exception as e:
        errors.append(f"Failed to run go test: {e}")

    result = {
        "pass": len(errors) == 0,
        "errors": errors,
    }

    print(json.dumps(result, ensure_ascii=False))
    sys.exit(0 if result["pass"] else 1)

if __name__ == "__main__":
    main()
