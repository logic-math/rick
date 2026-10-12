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
    project_root = "/Users/sunquan/ai_coding/CODING/rick"

    # Test 1: go build ./...
    print("Test 1: go build ./...", file=sys.stderr)
    rc, stdout, stderr = run_cmd("go build ./...", cwd=project_root)
    if rc != 0:
        errors.append(f"go build ./... failed: {stderr}")

    # Test 2: workspace.LoadToolsList - run the unit test
    print("Test 2: TestLoadToolsList", file=sys.stderr)
    rc, stdout, stderr = run_cmd(
        "go test ./internal/workspace/ -run TestLoadToolsList -v",
        cwd=project_root
    )
    if rc != 0:
        errors.append(f"TestLoadToolsList failed: {stderr}\n{stdout}")

    # Test 3: go test ./internal/prompt/ -v
    print("Test 3: go test ./internal/prompt/ -v", file=sys.stderr)
    rc, stdout, stderr = run_cmd("go test ./internal/prompt/ -v", cwd=project_root)
    if rc != 0:
        errors.append(f"go test ./internal/prompt/ failed: {stderr}\n{stdout}")

    # Test 4: go test ./...
    print("Test 4: go test ./...", file=sys.stderr)
    rc, stdout, stderr = run_cmd("go test ./...", cwd=project_root)
    if rc != 0:
        errors.append(f"go test ./... failed: {stderr}\n{stdout}")

    # Test 5: Check key source files exist
    print("Test 5: Check source files", file=sys.stderr)
    # Check LoadToolsList exists in workspace package
    workspace_files = []
    workspace_dir = os.path.join(project_root, "internal", "workspace")
    if os.path.isdir(workspace_dir):
        for f in os.listdir(workspace_dir):
            if f.endswith(".go"):
                workspace_files.append(os.path.join(workspace_dir, f))

    found_load_tools = False
    for fpath in workspace_files:
        try:
            with open(fpath, "r") as f:
                content = f.read()
                if "LoadToolsList" in content:
                    found_load_tools = True
                    break
        except Exception as e:
            errors.append(f"Failed to read {fpath}: {str(e)}")

    if not found_load_tools:
        errors.append("LoadToolsList function not found in internal/workspace/ package")

    # Check formatToolsSection exists in doing_prompt.go
    doing_prompt = os.path.join(project_root, "internal", "prompt", "doing_prompt.go")
    if not os.path.exists(doing_prompt):
        errors.append(f"doing_prompt.go does not exist at {doing_prompt}")
    else:
        try:
            with open(doing_prompt, "r") as f:
                content = f.read()
                if "formatToolsSection" not in content and "ToolsSection" not in content and "tools" not in content.lower():
                    errors.append("formatToolsSection (or tools injection) not found in doing_prompt.go")
        except Exception as e:
            errors.append(f"Failed to read doing_prompt.go: {str(e)}")

    # Check plan_prompt.go has tools injection
    plan_prompt = os.path.join(project_root, "internal", "prompt", "plan_prompt.go")
    if not os.path.exists(plan_prompt):
        errors.append(f"plan_prompt.go does not exist at {plan_prompt}")
    else:
        try:
            with open(plan_prompt, "r") as f:
                content = f.read()
                if "tools" not in content.lower():
                    errors.append("tools injection not found in plan_prompt.go")
        except Exception as e:
            errors.append(f"Failed to read plan_prompt.go: {str(e)}")

    # Check templates have tools section
    for tmpl_name in ["plan.md", "doing.md"]:
        tmpl_path = os.path.join(project_root, "internal", "prompt", "templates", tmpl_name)
        if not os.path.exists(tmpl_path):
            errors.append(f"Template {tmpl_name} does not exist at {tmpl_path}")
        else:
            try:
                with open(tmpl_path, "r") as f:
                    content = f.read()
                    if "tools" not in content.lower():
                        errors.append(f"tools section not found in template {tmpl_name}")
            except Exception as e:
                errors.append(f"Failed to read template {tmpl_name}: {str(e)}")

    result = {
        "pass": len(errors) == 0,
        "errors": errors
    }

    print(json.dumps(result))
    sys.exit(0 if result["pass"] else 1)

if __name__ == "__main__":
    main()
