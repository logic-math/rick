#!/usr/bin/env python3
import json
import sys
import os
import subprocess
import re

def run_cmd(cmd, cwd=None):
    """Run a shell command and return (returncode, stdout, stderr)."""
    result = subprocess.run(cmd, shell=True, capture_output=True, text=True, cwd=cwd)
    return result.returncode, result.stdout, result.stderr

def main():
    errors = []

    project_root = "/Users/sunquan/ai_coding/CODING/rick"

    # Test 1: go build ./... compiles successfully
    print("Running go build...", file=sys.stderr)
    rc, stdout, stderr = run_cmd("go build ./...", cwd=project_root)
    if rc != 0:
        errors.append(f"go build ./... failed: {stderr.strip()}")

    # Test 2: go test ./internal/prompt/... -v passes
    print("Running prompt tests...", file=sys.stderr)
    rc, stdout, stderr = run_cmd("go test ./internal/prompt/... -v -count=1", cwd=project_root)
    if rc != 0:
        errors.append(f"go test ./internal/prompt/... failed:\n{stdout[-2000:] if len(stdout) > 2000 else stdout}\n{stderr[-500:] if len(stderr) > 500 else stderr}")

    # Test 3: plan.md template contains a mandatory plan_check section
    plan_md = os.path.join(project_root, "internal", "prompt", "templates", "plan.md")
    try:
        with open(plan_md, "r") as f:
            plan_content = f.read()
        # Should contain a forced validation step referencing plan_check
        if "plan_check" not in plan_content:
            errors.append("plan.md template does not contain 'plan_check' — mandatory validation step is missing")
        # Should reference the rick_bin_path and job_id template variables
        if "{{rick_bin_path}}" not in plan_content:
            errors.append("plan.md template does not contain '{{rick_bin_path}}' variable")
        if "{{job_id}}" not in plan_content:
            errors.append("plan.md template does not contain '{{job_id}}' variable")
    except Exception as e:
        errors.append(f"Failed to read plan.md: {e}")

    # Test 4: doing.md template contains a mandatory doing_check constraint
    doing_md = os.path.join(project_root, "internal", "prompt", "templates", "doing.md")
    try:
        with open(doing_md, "r") as f:
            doing_content = f.read()
        # Should contain doing_check reference
        if "doing_check" not in doing_content:
            errors.append("doing.md template does not contain 'doing_check' — mandatory constraint is missing")
        # Should reference the rick_bin_path and job_id template variables
        if "{{rick_bin_path}}" not in doing_content:
            errors.append("doing.md template does not contain '{{rick_bin_path}}' variable")
        if "{{job_id}}" not in doing_content:
            errors.append("doing.md template does not contain '{{job_id}}' variable")
    except Exception as e:
        errors.append(f"Failed to read doing.md: {e}")

    # Test 5: learning.md Step 3 has strengthened mandatory language
    learning_md = os.path.join(project_root, "internal", "prompt", "templates", "learning.md")
    try:
        with open(learning_md, "r") as f:
            learning_content = f.read()
        # Step 3 should contain strong mandatory language
        mandatory_phrases = ["必须通过", "才能进入 Step 4", "才能继续"]
        found_mandatory = any(phrase in learning_content for phrase in mandatory_phrases)
        if not found_mandatory:
            errors.append(
                "learning.md Step 3 does not contain mandatory language "
                "('必须通过', '才能进入 Step 4', or '才能继续') — "
                "the learning_check requirement must be strengthened"
            )
    except Exception as e:
        errors.append(f"Failed to read learning.md: {e}")

    # Test 6: plan_prompt.go injects rick_bin_path and job_id variables
    plan_prompt_go = os.path.join(project_root, "internal", "prompt", "plan_prompt.go")
    try:
        with open(plan_prompt_go, "r") as f:
            plan_prompt_content = f.read()
        if "rick_bin_path" not in plan_prompt_content:
            errors.append("plan_prompt.go does not inject 'rick_bin_path' variable")
        if '"job_id"' not in plan_prompt_content and "'job_id'" not in plan_prompt_content:
            # Check for SetVariable("job_id", ...)
            if 'SetVariable("job_id"' not in plan_prompt_content:
                errors.append("plan_prompt.go does not inject 'job_id' variable via SetVariable")
    except Exception as e:
        errors.append(f"Failed to read plan_prompt.go: {e}")

    # Test 7: doing_prompt.go injects rick_bin_path and job_id variables
    doing_prompt_go = os.path.join(project_root, "internal", "prompt", "doing_prompt.go")
    try:
        with open(doing_prompt_go, "r") as f:
            doing_prompt_content = f.read()
        if "rick_bin_path" not in doing_prompt_content:
            errors.append("doing_prompt.go does not inject 'rick_bin_path' variable")
        if 'SetVariable("job_id"' not in doing_prompt_content:
            errors.append("doing_prompt.go does not inject 'job_id' variable via SetVariable")
    except Exception as e:
        errors.append(f"Failed to read doing_prompt.go: {e}")

    # Test 8: full test suite passes
    print("Running full test suite...", file=sys.stderr)
    rc, stdout, stderr = run_cmd("go test ./... -count=1", cwd=project_root)
    if rc != 0:
        errors.append(f"go test ./... failed:\n{stdout[-2000:] if len(stdout) > 2000 else stdout}\n{stderr[-500:] if len(stderr) > 500 else stderr}")

    result = {
        "pass": len(errors) == 0,
        "errors": errors
    }

    print(json.dumps(result))
    sys.exit(0 if result["pass"] else 1)

if __name__ == "__main__":
    main()
