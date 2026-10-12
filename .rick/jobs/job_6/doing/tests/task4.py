#!/usr/bin/env python3
import json
import sys
import os
import subprocess

PROJECT_ROOT = "/opt/meituan/dolphinfs_sunquan20/ai_coding/Coding/rick"
RICK_BINARY = os.path.join(PROJECT_ROOT, "rick")


def build_binary(errors):
    """Build the rick binary and return True on success."""
    try:
        result = subprocess.run(
            ["go", "build", "-o", RICK_BINARY, "./cmd/rick/"],
            cwd=PROJECT_ROOT,
            capture_output=True,
            text=True,
            timeout=120,
        )
        if result.returncode != 0:
            errors.append(f"go build failed: {result.stderr.strip()}")
            return False
        return True
    except Exception as e:
        errors.append(f"go build exception: {str(e)}")
        return False


def main():
    errors = []

    # Test 1: go build ./... — confirm no compile errors and no flag redefined panic
    try:
        result = subprocess.run(
            ["go", "build", "./..."],
            cwd=PROJECT_ROOT,
            capture_output=True,
            text=True,
            timeout=120,
        )
        if result.returncode != 0:
            errors.append(f"go build ./... failed: {result.stderr.strip()}")
        elif "flag redefined" in (result.stderr + result.stdout):
            errors.append("go build ./... output contains 'flag redefined'")
    except Exception as e:
        errors.append(f"go build ./... exception: {str(e)}")

    # Test 2: go test ./internal/cmd/... (run only plan-related tests to avoid hanging doing tests)
    try:
        result = subprocess.run(
            ["go", "test", "-timeout", "30s", "-run", "TestPlanCmd|TestReEnterPlan|TestGenerateJobID", "./internal/cmd/..."],
            cwd=PROJECT_ROOT,
            capture_output=True,
            text=True,
            timeout=60,
        )
        if result.returncode != 0:
            errors.append(f"go test plan tests failed: {result.stderr.strip() or result.stdout.strip()}")
    except Exception as e:
        errors.append(f"go test exception: {str(e)}")

    # Build binary for CLI tests
    binary_ok = build_binary(errors)

    if binary_ok:
        try:
            # Test 3: dry-run with --job job_1 should output "[DRY-RUN] Would re-enter plan for job: job_1" and exit 0
            result = subprocess.run(
                [RICK_BINARY, "--job", "job_1", "plan", "--dry-run", "需求"],
                cwd=PROJECT_ROOT,
                capture_output=True,
                text=True,
                timeout=30,
            )
            combined = result.stdout + result.stderr
            expected = "[DRY-RUN] Would re-enter plan for job: job_1"
            if result.returncode != 0:
                errors.append(
                    f"rick --job job_1 plan --dry-run exited {result.returncode} (expected 0). "
                    f"stdout={result.stdout!r} stderr={result.stderr!r}"
                )
            elif expected not in combined:
                errors.append(
                    f"dry-run with --job job_1 did not output expected message {expected!r}. "
                    f"stdout={result.stdout!r} stderr={result.stderr!r}"
                )
        except Exception as e:
            errors.append(f"dry-run --job job_1 exception: {str(e)}")

        try:
            # Test 4: non-existent job should return non-zero exit code with error about plan directory not existing
            result = subprocess.run(
                [RICK_BINARY, "--job", "job_999", "plan", "需求"],
                cwd=PROJECT_ROOT,
                capture_output=True,
                text=True,
                timeout=30,
            )
            combined = result.stdout + result.stderr
            expected_substr = "job job_999 plan directory does not exist"
            if result.returncode == 0:
                errors.append(
                    "rick --job job_999 plan should have returned non-zero exit code but returned 0"
                )
            elif expected_substr not in combined:
                errors.append(
                    f"rick --job job_999 plan error message mismatch. "
                    f"Expected substring: {expected_substr!r}. "
                    f"stdout={result.stdout!r} stderr={result.stderr!r}"
                )
        except Exception as e:
            errors.append(f"--job job_999 exception: {str(e)}")

        try:
            # Test 5: without --job, dry-run should still create a new job (not re-enter), behavior unchanged
            result = subprocess.run(
                [RICK_BINARY, "plan", "--dry-run", "需求"],
                cwd=PROJECT_ROOT,
                capture_output=True,
                text=True,
                timeout=30,
            )
            combined = result.stdout + result.stderr
            if result.returncode != 0:
                errors.append(
                    f"rick plan --dry-run (no --job) exited {result.returncode} (expected 0). "
                    f"stdout={result.stdout!r} stderr={result.stderr!r}"
                )
            elif "re-enter plan for job" in combined:
                errors.append(
                    f"rick plan --dry-run (no --job) unexpectedly output re-enter message. "
                    f"stdout={result.stdout!r}"
                )
            elif "[DRY-RUN] Would create a plan" not in combined:
                errors.append(
                    f"rick plan --dry-run (no --job) did not output '[DRY-RUN] Would create a plan'. "
                    f"stdout={result.stdout!r} stderr={result.stderr!r}"
                )
        except Exception as e:
            errors.append(f"plan --dry-run (no --job) exception: {str(e)}")

        # Cleanup binary
        try:
            os.remove(RICK_BINARY)
        except Exception:
            pass

    result_obj = {
        "pass": len(errors) == 0,
        "errors": errors,
    }

    print(json.dumps(result_obj))
    sys.exit(0 if result_obj["pass"] else 1)


if __name__ == "__main__":
    main()
