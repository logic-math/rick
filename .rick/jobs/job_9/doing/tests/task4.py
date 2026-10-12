#!/usr/bin/env python3
import json
import sys
import os
import subprocess

def main():
    errors = []
    project_root = "/Users/sunquan/ai_coding/CODING/rick"

    # Test 1: go build ./...
    try:
        result = subprocess.run(
            ["go", "build", "./..."],
            cwd=project_root,
            capture_output=True,
            text=True
        )
        if result.returncode != 0:
            errors.append(f"go build ./... failed: {result.stderr}")
    except Exception as e:
        errors.append(f"go build ./... error: {str(e)}")

    # Test 2: go test ./internal/cmd/ -v
    try:
        result = subprocess.run(
            ["go", "test", "./internal/cmd/", "-v"],
            cwd=project_root,
            capture_output=True,
            text=True,
            timeout=120
        )
        if result.returncode != 0:
            errors.append(f"go test ./internal/cmd/ failed: {result.stderr}")
    except Exception as e:
        errors.append(f"go test ./internal/cmd/ error: {str(e)}")

    # Test 3: go test ./internal/prompt/ -v
    try:
        result = subprocess.run(
            ["go", "test", "./internal/prompt/", "-v"],
            cwd=project_root,
            capture_output=True,
            text=True,
            timeout=120
        )
        if result.returncode != 0:
            errors.append(f"go test ./internal/prompt/ failed: {result.stderr}")
    except Exception as e:
        errors.append(f"go test ./internal/prompt/ error: {str(e)}")

    # Test 4: go test ./...
    try:
        result = subprocess.run(
            ["go", "test", "./..."],
            cwd=project_root,
            capture_output=True,
            text=True,
            timeout=180
        )
        if result.returncode != 0:
            errors.append(f"go test ./... failed: {result.stderr}")
    except Exception as e:
        errors.append(f"go test ./... error: {str(e)}")

    # Test 5: Verify plan.md template does NOT have {{okr_content}}, but mentions job OKR.md generation
    plan_md = os.path.join(project_root, "internal", "prompt", "templates", "plan.md")
    try:
        with open(plan_md, "r") as f:
            plan_content = f.read()
        if "{{okr_content}}" in plan_content:
            errors.append("plan.md template still contains {{okr_content}} (global OKR variable should be removed)")
        if "OKR.md" not in plan_content:
            errors.append("plan.md template does not instruct Claude to generate job-level OKR.md")
    except Exception as e:
        errors.append(f"Failed to read plan.md template: {str(e)}")

    # Test 6: Verify doing.md template has {{job_okr_content}}
    doing_md = os.path.join(project_root, "internal", "prompt", "templates", "doing.md")
    try:
        with open(doing_md, "r") as f:
            doing_content = f.read()
        if "{{job_okr_content}}" not in doing_content:
            errors.append("doing.md template missing {{job_okr_content}} variable")
    except Exception as e:
        errors.append(f"Failed to read doing.md template: {str(e)}")

    # Test 7: Verify plan.go does not load global OKR (LoadOKRFromFile with global path)
    plan_go = os.path.join(project_root, "internal", "cmd", "plan.go")
    try:
        with open(plan_go, "r") as f:
            plan_go_content = f.read()
        # The global OKR loading pattern should be removed
        if "LoadOKRFromFile" in plan_go_content and "OKR.md" in plan_go_content:
            # Check if it references the global .rick/OKR.md path (not job-level)
            if '".rick/OKR.md"' in plan_go_content or '"OKR.md"' in plan_go_content:
                errors.append("plan.go still loads global OKR.md via LoadOKRFromFile")
    except Exception as e:
        errors.append(f"Failed to read plan.go: {str(e)}")

    # Test 8: Verify doing.go reads job-level OKR (job_N/plan/OKR.md)
    doing_go = os.path.join(project_root, "internal", "cmd", "doing.go")
    try:
        with open(doing_go, "r") as f:
            doing_go_content = f.read()
        # doing.go should reference job-level OKR path
        if "OKR.md" not in doing_go_content and "job_okr" not in doing_go_content.lower():
            errors.append("doing.go does not read job-level OKR.md (expected reference to plan/OKR.md or job_okr)")
    except Exception as e:
        errors.append(f"Failed to read doing.go: {str(e)}")

    # Test 9: Verify plan_prompt.go does not set okr_content variable
    plan_prompt_go = os.path.join(project_root, "internal", "cmd", "plan_prompt.go")
    if os.path.exists(plan_prompt_go):
        try:
            with open(plan_prompt_go, "r") as f:
                pp_content = f.read()
            if '"okr_content"' in pp_content:
                errors.append("plan_prompt.go still sets okr_content variable (should be removed)")
        except Exception as e:
            errors.append(f"Failed to read plan_prompt.go: {str(e)}")

    # Test 10: Verify doing_prompt.go sets job_okr_content variable
    doing_prompt_go = os.path.join(project_root, "internal", "cmd", "doing_prompt.go")
    if os.path.exists(doing_prompt_go):
        try:
            with open(doing_prompt_go, "r") as f:
                dp_content = f.read()
            if '"job_okr_content"' not in dp_content:
                errors.append("doing_prompt.go does not set job_okr_content variable")
        except Exception as e:
            errors.append(f"Failed to read doing_prompt.go: {str(e)}")

    # Test 11: rick plan --dry-run - prompt should not contain global OKR, should mention job OKR.md
    try:
        result = subprocess.run(
            ["go", "run", "./cmd/rick", "plan", "--dry-run"],
            cwd=project_root,
            capture_output=True,
            text=True,
            timeout=30
        )
        output = result.stdout + result.stderr
        if "{{okr_content}}" in output:
            errors.append("plan --dry-run prompt still contains unresolved {{okr_content}} variable")
        if "OKR.md" not in output:
            errors.append("plan --dry-run prompt does not mention generating job-level OKR.md")
    except Exception as e:
        errors.append(f"rick plan --dry-run error: {str(e)}")

    # Test 12: rick doing --dry-run job_9 - prompt should include job OKR section if OKR.md exists
    job9_okr = os.path.join(project_root, ".rick", "jobs", "job_9", "plan", "OKR.md")
    try:
        result = subprocess.run(
            ["go", "run", "./cmd/rick", "doing", "--dry-run", "job_9"],
            cwd=project_root,
            capture_output=True,
            text=True,
            timeout=30
        )
        output = result.stdout + result.stderr
        if os.path.exists(job9_okr):
            if "OKR" not in output:
                errors.append("doing --dry-run job_9: OKR.md exists but doing prompt missing job OKR section")
    except Exception as e:
        errors.append(f"rick doing --dry-run job_9 error: {str(e)}")

    result = {
        "pass": len(errors) == 0,
        "errors": errors
    }

    print(json.dumps(result))
    sys.exit(0 if result["pass"] else 1)

if __name__ == "__main__":
    main()
