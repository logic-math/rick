#!/usr/bin/env python3
import json
import sys
import os
import subprocess

def run_go_command(cmd, cwd):
    """Run a go command and return (success, output)"""
    try:
        result = subprocess.run(
            cmd,
            cwd=cwd,
            capture_output=True,
            text=True,
            timeout=120
        )
        return result.returncode == 0, result.stdout + result.stderr
    except subprocess.TimeoutExpired:
        return False, f"Command timed out: {' '.join(cmd)}"
    except Exception as e:
        return False, f"Command failed: {str(e)}"

def main():
    errors = []

    # Project root
    project_root = "/Users/sunquan/ai_coding/CODING/rick"

    # Test 1: go build ./...
    print("Running go build ./...", file=sys.stderr)
    ok, output = run_go_command(["go", "build", "./..."], project_root)
    if not ok:
        errors.append(f"go build ./... failed: {output}")

    # Test 2: go test ./internal/workspace/ -v
    print("Running go test ./internal/workspace/ -v", file=sys.stderr)
    ok, output = run_go_command(["go", "test", "./internal/workspace/", "-v"], project_root)
    if not ok:
        errors.append(f"go test ./internal/workspace/ failed: {output}")

    # Test 3: go test ./internal/prompt/ -v
    print("Running go test ./internal/prompt/ -v", file=sys.stderr)
    ok, output = run_go_command(["go", "test", "./internal/prompt/", "-v"], project_root)
    if not ok:
        errors.append(f"go test ./internal/prompt/ failed: {output}")

    # Test 4: Check that skills/index.md exists and has expected format
    index_md = os.path.join(project_root, ".rick", "skills", "index.md")
    if not os.path.exists(index_md):
        errors.append(f".rick/skills/index.md does not exist")
    else:
        try:
            with open(index_md, "r") as f:
                content = f.read()
            # index.md should have skill entries
            if len(content.strip()) == 0:
                errors.append(".rick/skills/index.md is empty")
        except Exception as e:
            errors.append(f"Failed to read .rick/skills/index.md: {str(e)}")

    # Test 5: Check LoadSkillsIndex exists in workspace/skills.go
    skills_go = os.path.join(project_root, "internal", "workspace", "skills.go")
    if not os.path.exists(skills_go):
        errors.append("internal/workspace/skills.go does not exist")
    else:
        try:
            with open(skills_go, "r") as f:
                content = f.read()
            if "LoadSkillsIndex" not in content:
                errors.append("LoadSkillsIndex not found in internal/workspace/skills.go")
        except Exception as e:
            errors.append(f"Failed to read skills.go: {str(e)}")

    # Test 6: Check doing_prompt.go uses index.md (reads index or falls back)
    doing_prompt_files = []
    for fname in ["doing_prompt.go", "prompt_doing.go"]:
        p = os.path.join(project_root, "internal", "prompt", fname)
        if os.path.exists(p):
            doing_prompt_files.append(p)
    # Also check internal/cmd/doing.go or similar
    for root, dirs, files in os.walk(os.path.join(project_root, "internal")):
        for f in files:
            if f.endswith(".go"):
                full = os.path.join(root, f)
                if full not in doing_prompt_files:
                    try:
                        with open(full, "r") as fh:
                            c = fh.read()
                        if "formatSkillsSection" in c or "LoadSkillsIndex" in c:
                            doing_prompt_files.append(full)
                    except Exception:
                        pass

    found_index_usage = False
    for fp in doing_prompt_files:
        try:
            with open(fp, "r") as fh:
                c = fh.read()
            if "LoadSkillsIndex" in c or "index.md" in c:
                found_index_usage = True
                break
        except Exception:
            pass
    if not found_index_usage:
        errors.append("No Go file found that uses LoadSkillsIndex or references index.md in skills section")

    # Test 7: Check plan prompt has skills_index template variable
    found_skills_index_in_plan = False
    for root, dirs, files in os.walk(os.path.join(project_root, "internal")):
        for f in files:
            if f.endswith(".go") or f.endswith(".md"):
                full = os.path.join(root, f)
                try:
                    with open(full, "r") as fh:
                        c = fh.read()
                    if "skills_index" in c and ("plan" in f.lower() or "plan" in root.lower()):
                        found_skills_index_in_plan = True
                        break
                except Exception:
                    pass
        if found_skills_index_in_plan:
            break
    if not found_skills_index_in_plan:
        errors.append("skills_index template variable not found in plan-related prompt files")

    # Test 8: go test ./... (full test suite)
    print("Running go test ./...", file=sys.stderr)
    ok, output = run_go_command(["go", "test", "./..."], project_root)
    if not ok:
        errors.append(f"go test ./... failed: {output}")

    result = {
        "pass": len(errors) == 0,
        "errors": errors
    }

    print(json.dumps(result))
    sys.exit(0 if result["pass"] else 1)

if __name__ == "__main__":
    main()
