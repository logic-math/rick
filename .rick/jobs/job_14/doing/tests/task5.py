#!/usr/bin/env python3
import json
import sys
import os
import subprocess

PROJECT_ROOT = "/Users/sunquan/ai_coding/CODING/rick"

def run(cmd, **kwargs):
    return subprocess.run(cmd, cwd=PROJECT_ROOT, capture_output=True, text=True, **kwargs)

def main():
    errors = []

    # Test 1: build and verify dream command exists
    rick_bin = None
    try:
        result = run(["python3", "tools/build_and_get_rick_bin.py"])
        if result.returncode != 0:
            errors.append(f"build failed: {result.stderr.strip()}")
        else:
            try:
                parsed = json.loads(result.stdout.strip())
                rick_bin = parsed.get("bin_path", "").strip()
            except (json.JSONDecodeError, AttributeError):
                rick_bin = result.stdout.strip()
            if not rick_bin or not os.path.exists(rick_bin):
                errors.append(f"build_and_get_rick_bin.py returned invalid path: {rick_bin!r}")
                rick_bin = None
    except Exception as e:
        errors.append(f"build error: {str(e)}")

    # Test 2: dream --help shows --dry-run flag
    if rick_bin:
        try:
            result = run([rick_bin, "dream", "--help"])
            output = result.stdout + result.stderr
            if "--dry-run" not in output:
                errors.append(f"'rick dream --help' missing --dry-run flag; got: {output[:300]}")
        except Exception as e:
            errors.append(f"dream --help error: {str(e)}")

    # Test 3: dry-run output contains required SOP keywords
    if rick_bin:
        try:
            result = run([rick_bin, "dream", "--dry-run"])
            output = result.stdout + result.stderr
            for keyword in ["skill:sense", "skill:evolve-skills", "readme.md", "I will use skill"]:
                if keyword not in output:
                    errors.append(f"dry-run output missing '{keyword}'")
        except Exception as e:
            errors.append(f"dream --dry-run error: {str(e)}")

    # Test 4: dream.md exists and contains 500-line constraint
    dream_template = os.path.join(PROJECT_ROOT, "internal", "prompt", "templates", "dream.md")
    if not os.path.exists(dream_template):
        errors.append("internal/prompt/templates/dream.md does not exist")
    else:
        try:
            with open(dream_template, "r") as f:
                content = f.read()
            if "500" not in content:
                errors.append("dream.md missing SPEC.md 500-line constraint mention")
        except Exception as e:
            errors.append(f"Failed to read dream.md: {str(e)}")

    # Test 5: check_prompt_variables sense skill present
    try:
        result = run(["python3", "tools/check_prompt_variables.py", "--phase", "dream", "--keywords", "skill:sense"])
        output = result.stdout.strip()
        try:
            parsed = json.loads(output)
            if not parsed.get("pass"):
                errors.append(f"check_prompt_variables sense failed: {parsed.get('errors', output)}")
        except json.JSONDecodeError:
            if "关键词未找到" in output or result.returncode != 0:
                errors.append(f"dream prompt missing skill:sense; output: {output}")
    except Exception as e:
        errors.append(f"check_prompt_variables sense error: {str(e)}")

    # Test 6: check_prompt_variables evolve-skills present
    try:
        result = run(["python3", "tools/check_prompt_variables.py", "--phase", "dream", "--keywords", "skill:evolve-skills"])
        output = result.stdout.strip()
        try:
            parsed = json.loads(output)
            if not parsed.get("pass"):
                errors.append(f"check_prompt_variables evolve-skills failed: {parsed.get('errors', output)}")
        except json.JSONDecodeError:
            if "关键词未找到" in output or result.returncode != 0:
                errors.append(f"dream prompt missing skill:evolve-skills; output: {output}")
    except Exception as e:
        errors.append(f"check_prompt_variables evolve-skills error: {str(e)}")

    # Test 7: no tdd pollution - skill:tdd must NOT be in dream prompt
    try:
        result = run(["python3", "tools/check_prompt_variables.py", "--phase", "dream", "--keywords", "skill:tdd"])
        output = result.stdout.strip()
        try:
            parsed = json.loads(output)
            if parsed.get("pass"):
                errors.append("dream prompt contains skill:tdd (tdd pollution)")
        except json.JSONDecodeError:
            # "关键词未找到" means not found, which is what we want
            if "关键词未找到" not in output and result.returncode == 0:
                errors.append(f"unexpected output from tdd check: {output}")
    except Exception as e:
        errors.append(f"check_prompt_variables tdd error: {str(e)}")

    # Test 8: internal/cmd/dream.go exists
    dream_go = os.path.join(PROJECT_ROOT, "internal", "cmd", "dream.go")
    if not os.path.exists(dream_go):
        errors.append("internal/cmd/dream.go does not exist")

    # Test 9: internal/prompt/dream_prompt.go exists
    dream_prompt_go = os.path.join(PROJECT_ROOT, "internal", "prompt", "dream_prompt.go")
    if not os.path.exists(dream_prompt_go):
        errors.append("internal/prompt/dream_prompt.go does not exist")

    # Test 10: workspace TestDreamDir
    try:
        result = run(["go", "test", "./internal/workspace/...", "-v", "-run", "TestDreamDir"])
        if result.returncode != 0:
            errors.append(f"TestDreamDir failed:\n{result.stdout.strip()}\n{result.stderr.strip()}")
    except Exception as e:
        errors.append(f"go test workspace error: {str(e)}")

    # Test 11: go test ./...
    try:
        result = run(["go", "test", "./..."])
        if result.returncode != 0:
            errors.append(f"go test ./... failed:\n{result.stderr.strip()}")
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
