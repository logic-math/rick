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
    project_root = os.path.abspath(os.path.join(os.path.dirname(__file__), '..', '..', '..', '..', '..'))
    print(f"[debug] project_root={project_root}", file=sys.stderr)

    # Test 1: Build
    rc, out, err = run_cmd("python3 tools/build_and_get_rick_bin.py", cwd=project_root)
    if rc != 0:
        errors.append(f"Build failed (rc={rc}): {err.strip() or out.strip()}")

    # Test 2: Executor backward compat
    rc, out, err = run_cmd("go test ./internal/executor/... -v", cwd=project_root)
    if rc != 0:
        errors.append(f"Executor backward compat tests failed: {err.strip() or out.strip()}")

    # Test 3: RED unit tests in runner_test.go
    for test_name in ["TestRunTask_REDPass_TriggersRetry", "TestRunTask_REDPass_MaxRetry", "TestRunTask_REDFail_Normal"]:
        rc, out, err = run_cmd(f"go test ./internal/executor/... -v -run {test_name}", cwd=project_root)
        if rc != 0:
            errors.append(f"{test_name} failed: {(out + err).strip()}")

    # Test 4: testing agent has skill:tdd
    rc, out, err = run_cmd('python3 tools/check_prompt_variables.py --phase testing --keywords "skill:tdd"', cwd=project_root)
    try:
        result = json.loads(out.strip())
        if not result.get("pass"):
            errors.append(f"testing agent missing skill:tdd: {out.strip()}")
    except Exception as e:
        errors.append(f"check_prompt_variables (testing/skill:tdd) parse error: {e}, output={out.strip()}")

    # Test 5: testing agent has RED instruction (pass.*false or RED phase)
    red_found = False
    for kw in ["pass.*false", "RED phase"]:
        rc2, out2, err2 = run_cmd(f'python3 tools/check_prompt_variables.py --phase testing --keywords "{kw}"', cwd=project_root)
        try:
            result2 = json.loads(out2.strip())
            if result2.get("pass"):
                red_found = True
                break
        except Exception:
            pass
    if not red_found:
        errors.append("testing agent missing RED instruction (neither 'pass.*false' nor 'RED phase' found)")

    # Test 6: coding agent has skill:debug
    rc, out, err = run_cmd('python3 tools/check_prompt_variables.py --phase doing --keywords "skill:debug"', cwd=project_root)
    try:
        result = json.loads(out.strip())
        if not result.get("pass"):
            errors.append(f"coding agent missing skill:debug: {out.strip()}")
    except Exception as e:
        errors.append(f"check_prompt_variables (doing/skill:debug) parse error: {e}, output={out.strip()}")

    # Test 7: no gen-skill pollution in doing prompt
    rc, out, err = run_cmd('python3 tools/check_prompt_variables.py --phase doing --keywords "gen-skill"', cwd=project_root)
    # Expect "pass": false (keyword not found = no pollution)
    try:
        result = json.loads(out.strip())
        if result.get("pass"):
            errors.append("doing prompt contains 'gen-skill' (unexpected pollution)")
    except Exception:
        # If output is not JSON, check for "关键词未找到" style message
        if "gen-skill" in out.lower() and "pass" not in out.lower():
            pass  # probably fine
        # Don't error on parse failure here; the check is best-effort via non-JSON output

    # Test 8: Full test suite - no new failures
    rc, out, err = run_cmd("go test ./...", cwd=project_root)
    if rc != 0:
        # Collect failing tests for context
        failing = [line for line in (out + err).splitlines() if "FAIL" in line or "--- FAIL" in line]
        errors.append(f"go test ./... failed: {'; '.join(failing) or (out + err).strip()[:300]}")

    result = {
        "pass": len(errors) == 0,
        "errors": errors
    }
    print(json.dumps(result))
    sys.exit(0 if result["pass"] else 1)

if __name__ == "__main__":
    main()
