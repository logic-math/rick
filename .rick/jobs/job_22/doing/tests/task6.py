#!/usr/bin/env python3
import json
import sys
import os
import subprocess
import re


def get_project_root():
    # .rick/jobs/job_22/doing/tests/task6.py → 6 dirname calls
    path = os.path.abspath(__file__)
    for _ in range(6):
        path = os.path.dirname(path)
    return path


def read_learning_md(project_root):
    path = os.path.join(project_root, "internal", "prompt", "templates", "learning.md")
    with open(path, "r", encoding="utf-8") as f:
        return f.read()


def build_and_get_bin(project_root):
    build_tool = os.path.join(project_root, ".rick", "tools", "build_and_get_rick_bin.py")
    result = subprocess.run(
        [sys.executable, build_tool],
        capture_output=True, text=True, cwd=project_root, timeout=120
    )
    if result.returncode != 0:
        raise RuntimeError(f"build_and_get_rick_bin.py failed: {result.stderr[:300]}")
    data = json.loads(result.stdout.strip())
    return data["bin_path"]


def main():
    errors = []
    project_root = get_project_root()

    # Test 1: Old variables removed from learning.md template
    try:
        content = read_learning_md(project_root)
        for var in ["wiki_dir", "tools_dir", "spec_path"]:
            if re.search(r"\{\{" + re.escape(var) + r"\}\}", content):
                errors.append("learning.md still contains {{" + var + "}} (old variable not removed)")
    except Exception as e:
        errors.append(f"Failed to check old variables in learning.md: {str(e)}")

    # Test 2: New variables present in learning.md template (need >= 3)
    try:
        content = read_learning_md(project_root)
        new_vars = ["loops_dir", "skills_dir", "loops_context"]
        missing = [v for v in new_vars if v not in content]
        if missing:
            errors.append(f"learning.md missing new variables: {missing}")
    except Exception as e:
        errors.append(f"Failed to verify new variables in learning.md: {str(e)}")

    # Test 3: Candidate file naming present in learning.md (need >= 2 occurrences)
    try:
        content = read_learning_md(project_root)
        count = len(re.findall(r"candidate_loop|candidate_skill", content))
        if count < 2:
            errors.append(f"learning.md has only {count} candidate_loop/candidate_skill references (need >= 2)")
    except Exception as e:
        errors.append(f"Failed to verify candidate naming in learning.md: {str(e)}")

    # Test 4 + 5: Build binary and verify dry-run output
    try:
        rick_bin = build_and_get_bin(project_root)

        dr = subprocess.run(
            [rick_bin, "learning", "--job", "job_22", "--dry-run"],
            capture_output=True, text=True, cwd=project_root, timeout=30
        )
        output = dr.stdout + dr.stderr

        # Must contain new variable values (not literals)
        if "loops_dir" not in output:
            errors.append("dry-run output missing resolved 'loops_dir' path")
        if "skills_dir" not in output:
            errors.append("dry-run output missing resolved 'skills_dir' path")
        if "可用的项目 Loops" not in output:
            errors.append("dry-run output missing '可用的项目 Loops' section")

        # Must NOT contain unreplaced placeholders
        if "{{wiki_dir}}" in output:
            errors.append("dry-run output contains unreplaced {{wiki_dir}} literal")
        if "{{spec_path}}" in output:
            errors.append("dry-run output contains unreplaced {{spec_path}} literal")
    except Exception as e:
        errors.append(f"Build or dry-run test failed: {str(e)}")

    result = {"pass": len(errors) == 0, "errors": errors}
    print(json.dumps(result, ensure_ascii=False))
    sys.exit(0 if result["pass"] else 1)


if __name__ == "__main__":
    main()
