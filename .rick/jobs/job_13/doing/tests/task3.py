#!/usr/bin/env python3
import json
import sys
import os
import subprocess


def run_command(cmd, shell=True):
    result = subprocess.run(cmd, shell=shell, capture_output=True, text=True, timeout=120)
    return result.returncode, result.stdout, result.stderr


def main():
    errors = []

    project_root = "/Users/sunquan/ai_coding/CODING/rick"
    os.chdir(project_root)

    check_prompt_script = f"{project_root}/tools/check_prompt_variables.py"

    # Test 1: Go build check
    rc, out, err = run_command(f"python3 {project_root}/tools/check_go_build.py")
    if rc != 0:
        errors.append(f"Go build failed: {out}{err}")

    # Test 2: dry-run output contains human_loop_think keyword
    rc, out, err = run_command(
        f"python3 {check_prompt_script} --phase human-loop --topic '测试主题' --keywords human_loop_think"
    )
    if rc != 0:
        errors.append(f"dry-run missing think_agent_path: {out}{err}")

    # Test 3: dry-run output contains human_loop_learn keyword
    rc, out, err = run_command(
        f"python3 {check_prompt_script} --phase human-loop --topic '测试主题' --keywords human_loop_learn"
    )
    if rc != 0:
        errors.append(f"dry-run missing learn_agent_path: {out}{err}")

    # Test 4: dry-run output contains human_loop_express keyword
    rc, out, err = run_command(
        f"python3 {check_prompt_script} --phase human-loop --topic '测试主题' --keywords human_loop_express"
    )
    if rc != 0:
        errors.append(f"dry-run missing express_agent_path: {out}{err}")

    # Test 5: dry-run outputs full prompt content (not a placeholder one-liner)
    # Build rick binary first
    rc, rick_bin_out, bin_err = run_command(f"python3 {project_root}/tools/build_and_get_rick_bin.py")
    if rc != 0:
        errors.append(f"build_and_get_rick_bin.py failed: {bin_err}")
    else:
        try:
            build_result = json.loads(rick_bin_out.strip())
            rick_bin = build_result["bin_path"]
        except (json.JSONDecodeError, KeyError):
            rick_bin = rick_bin_out.strip()
        rc, out, err = run_command(f"{rick_bin} human-loop --dry-run '测试主题'")
        if rc != 0:
            errors.append(f"human-loop --dry-run failed (exit {rc}): {err}")
        else:
            if len(out.strip()) < 100:
                errors.append(
                    f"dry-run output too short ({len(out.strip())} chars); expected full prompt, got: {repr(out[:200])}"
                )

    result = {
        "pass": len(errors) == 0,
        "errors": errors
    }

    print(json.dumps(result))
    sys.exit(0 if result["pass"] else 1)


if __name__ == "__main__":
    main()
