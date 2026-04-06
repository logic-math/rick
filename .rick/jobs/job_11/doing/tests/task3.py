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

    # Derive project root: .rick/jobs/job_11/doing/tests/task3.py -> 5 levels up
    project_root = os.path.abspath(
        os.path.join(os.path.dirname(os.path.abspath(__file__)), '..', '..', '..', '..', '..')
    )
    print(f"Project root: {project_root}", file=sys.stderr)

    # Test 1: go build ./... compiles successfully and produces ./bin/rick
    rc, stdout, stderr = run_cmd("go build -o bin/rick ./cmd/rick", cwd=project_root)
    if rc != 0:
        errors.append(f"go build failed: {stderr.strip()}")
        print(json.dumps({"pass": False, "errors": errors}))
        sys.exit(1)

    rick_bin = os.path.join(project_root, "bin", "rick")

    # Test 2: TestPlanCheck passes
    rc, stdout, stderr = run_cmd(
        "go test ./internal/cmd/... -v -run TestPlanCheck",
        cwd=project_root
    )
    if rc != 0:
        errors.append(f"TestPlanCheck failed: {stderr.strip() or stdout.strip()}")
    else:
        print(f"TestPlanCheck passed", file=sys.stderr)

    # Test 3: TestDoingCheck passes
    rc, stdout, stderr = run_cmd(
        "go test ./internal/cmd/... -v -run TestDoingCheck",
        cwd=project_root
    )
    if rc != 0:
        errors.append(f"TestDoingCheck failed: {stderr.strip() or stdout.strip()}")
    else:
        print(f"TestDoingCheck passed", file=sys.stderr)

    # Test 4: TestLearningCheck passes
    rc, stdout, stderr = run_cmd(
        "go test ./internal/cmd/... -v -run TestLearningCheck",
        cwd=project_root
    )
    if rc != 0:
        errors.append(f"TestLearningCheck failed: {stderr.strip() or stdout.strip()}")
    else:
        print(f"TestLearningCheck passed", file=sys.stderr)

    # Test 5: Verify OKR.md check is present in source code (static check)
    # The unit tests (TestRunPlanCheck_MissingOKR) already verify the behavior.
    # Here we just confirm the source contains the OKR.md check logic.
    plan_check_src = os.path.join(project_root, "internal", "cmd", "tools_plan_check.go")
    with open(plan_check_src) as f:
        src = f.read()
    if "OKR.md" not in src:
        errors.append("tools_plan_check.go does not contain OKR.md check")
    elif "OKR.md not found in plan directory" not in src:
        errors.append("tools_plan_check.go missing expected OKR.md error message")
    else:
        print("plan_check OKR.md check found in source", file=sys.stderr)

    # Test 6: ./bin/rick tools doing_check job_9 — expect pass
    rc, stdout, stderr = run_cmd(f"{rick_bin} tools doing_check job_9", cwd=project_root)
    if rc != 0:
        errors.append(f"rick tools doing_check job_9 failed: {(stdout + stderr).strip()[:300]}")

    # Test 7: ./bin/rick tools learning_check job_9 — expect pass
    rc, stdout, stderr = run_cmd(f"{rick_bin} tools learning_check job_9", cwd=project_root)
    if rc != 0:
        errors.append(f"rick tools learning_check job_9 failed: {(stdout + stderr).strip()[:300]}")

    # Test 8: full test suite passes
    rc, stdout, stderr = run_cmd("go test ./... -count=1", cwd=project_root)
    if rc != 0:
        errors.append(f"go test ./... -count=1 failed: {(stderr + stdout).strip()[:500]}")

    result = {
        "pass": len(errors) == 0,
        "errors": errors
    }

    print(json.dumps(result))
    sys.exit(0 if result["pass"] else 1)

if __name__ == "__main__":
    main()
