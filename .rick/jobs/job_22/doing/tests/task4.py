#!/usr/bin/env python3
# Description: Test task4 - migrate doing prompt builder: remove SPEC/OKR, inject loops_context
import json
import os
import shutil
import subprocess
import sys


def main():
    errors = []

    # Project root: 6 dirnames up from this script
    script_path = os.path.abspath(__file__)
    project_root = script_path
    for _ in range(6):
        project_root = os.path.dirname(project_root)

    print(f"project_root: {project_root}", file=sys.stderr)

    # Step 1: Build binary via build_and_get_rick_bin.py
    print("Step 1: Building binary...", file=sys.stderr)
    build_script = os.path.join(project_root, ".rick", "tools", "build_and_get_rick_bin.py")
    bin_path = None
    try:
        proc = subprocess.run(
            ["python3", build_script],
            capture_output=True, text=True, cwd=project_root, timeout=120
        )
        build_result = json.loads(proc.stdout.strip())
        if not build_result.get("pass"):
            errors.append(f"Build failed: {build_result.get('errors', [])}")
        else:
            bin_path = build_result.get("bin_path")
            print(f"bin_path: {bin_path}", file=sys.stderr)
    except Exception as e:
        errors.append(f"Build step exception: {str(e)}")

    if not bin_path:
        print(json.dumps({"pass": False, "errors": errors}, ensure_ascii=False))
        sys.exit(1)

    # Step 2: dry-run normal path — loops_context present, SPEC/OKR removed
    print("Step 2: dry-run normal path...", file=sys.stderr)
    try:
        proc = subprocess.run(
            [bin_path, "doing", "--job", "job_22", "--dry-run"],
            capture_output=True, text=True, cwd=project_root, timeout=30
        )
        output = proc.stdout + proc.stderr

        # loops_context injected: check for specific loop entry from .rick/loops/candidate_loop_1.md
        if "candidate-loop-1" not in output and "when implementing new features" not in output:
            errors.append(
                "dry-run output missing loop entry from .rick/loops/candidate_loop_1.md — "
                "loops_context not injected (expected 'candidate-loop-1' or 'when implementing new features')"
            )

        if "### 项目 SPEC" in output:
            errors.append("dry-run output still contains '### 项目 SPEC' section — spec_content not removed")

        if "### Job OKR" in output:
            errors.append("dry-run output still contains '### Job OKR' section — job_okr_content not removed")

    except Exception as e:
        errors.append(f"dry-run step exception: {str(e)}")

    # Step 3: Go unit tests for doing prompt (precise package scope)
    print("Step 3: Go unit tests ./internal/prompt/...", file=sys.stderr)
    try:
        proc = subprocess.run(
            ["go", "test", "./internal/prompt/...", "-run", "TestDoingPrompt", "-v"],
            capture_output=True, text=True, cwd=project_root, timeout=60
        )
        if proc.returncode != 0:
            errors.append(
                f"Go unit tests failed (exit {proc.returncode}):\n"
                f"{(proc.stdout + proc.stderr)[-1500:]}"
            )
    except subprocess.TimeoutExpired:
        errors.append("Go unit tests timed out after 60s")
    except Exception as e:
        errors.append(f"Go unit test exception: {str(e)}")

    # Step 4: Edge case — empty loops dir should show fallback text
    print("Step 4: Edge case — empty loops dir...", file=sys.stderr)
    loops_dir = os.path.join(project_root, ".rick", "loops")
    loops_backup = loops_dir + "_task4_bak"
    try:
        if os.path.isdir(loops_dir):
            shutil.copytree(loops_dir, loops_backup)
            shutil.rmtree(loops_dir)
        os.makedirs(loops_dir, exist_ok=True)

        proc = subprocess.run(
            [bin_path, "doing", "--job", "job_22", "--dry-run"],
            capture_output=True, text=True, cwd=project_root, timeout=30
        )
        output = proc.stdout + proc.stderr

        if proc.returncode != 0:
            errors.append(f"Empty loops dir: command exited {proc.returncode}, expected 0")

        if "暂无項目 Loop" not in output and "暂无项目 Loop" not in output:
            errors.append("Empty loops dir: output missing fallback text '暂无项目 Loop'")

    except Exception as e:
        errors.append(f"Edge case empty loops exception: {str(e)}")
    finally:
        if os.path.isdir(loops_dir):
            shutil.rmtree(loops_dir)
        if os.path.isdir(loops_backup):
            shutil.copytree(loops_backup, loops_dir)
            shutil.rmtree(loops_backup)

    # Step 5: Edge case — loops dir missing entirely should not panic
    print("Step 5: Edge case — missing loops dir...", file=sys.stderr)
    loops_backup2 = loops_dir + "_task4_bak2"
    try:
        if os.path.isdir(loops_dir):
            shutil.copytree(loops_dir, loops_backup2)
            shutil.rmtree(loops_dir)

        proc = subprocess.run(
            [bin_path, "doing", "--job", "job_22", "--dry-run"],
            capture_output=True, text=True, cwd=project_root, timeout=30
        )
        output = proc.stdout + proc.stderr

        if proc.returncode != 0:
            errors.append(f"Missing loops dir: command exited {proc.returncode}, expected 0 (no panic)")

        if "暂无項目 Loop" not in output and "暂无项目 Loop" not in output:
            errors.append("Missing loops dir: output missing fallback text '暂无项目 Loop'")

    except Exception as e:
        errors.append(f"Edge case missing loops exception: {str(e)}")
    finally:
        if not os.path.isdir(loops_dir) and os.path.isdir(loops_backup2):
            shutil.copytree(loops_backup2, loops_dir)
        if os.path.isdir(loops_backup2):
            shutil.rmtree(loops_backup2)

    result = {
        "pass": len(errors) == 0,
        "errors": errors,
    }
    print(json.dumps(result, ensure_ascii=False))
    sys.exit(0 if result["pass"] else 1)


if __name__ == "__main__":
    main()
