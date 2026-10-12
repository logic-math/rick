# Description: Pattern for implementing rick tools check commands (argparse + JSON output standard)
#!/usr/bin/env python3
"""
rick_tools_check_pattern - Pattern for implementing rick tools check commands

This skill demonstrates the standard pattern for building check/validation
scripts that follow the rick tools check command conventions:
- argparse for argument parsing
- JSON output: {"pass": bool, "errors": [...]}
- Exit code 0=pass, 1=fail

Two levels of file validation (as of job_11):
  Level 1 (existence): os.path.isfile(path)
  Level 2 (content):   read + check non-empty + check required structure

Usage:
    python3 rick_tools_check_pattern.py --job-id job_1
    python3 rick_tools_check_pattern.py --job-id job_1 --check-type doing

Output:
    {"pass": true, "errors": []}
    {"pass": false, "errors": ["error description 1", "error description 2"]}
"""

import argparse
import json
import os
import sys


def find_project_root():
    """Walk up from script location to find project root (contains .rick/)."""
    current = os.path.dirname(os.path.abspath(__file__))
    while current != os.path.dirname(current):
        if os.path.isdir(os.path.join(current, ".rick")):
            return current
        current = os.path.dirname(current)
    return None


def check_file_exists_and_has_content(path, required_substring=None):
    """
    Two-level file check (job_11 pattern):
      1. File must exist
      2. File must be non-empty
      3. (Optional) File must contain required_substring

    Returns list of error strings (empty = passed).
    """
    errors = []
    if not os.path.isfile(path):
        errors.append(f"file not found: {path}")
        return errors  # can't check content if missing

    content = open(path, encoding="utf-8").read()
    if not content.strip():
        errors.append(f"file exists but is empty: {path}")
        return errors

    if required_substring and required_substring not in content:
        errors.append(
            f"file exists but missing required content '{required_substring}': {path}"
        )
    return errors


def check_plan(job_id, project_root):
    """Check plan directory structure."""
    errors = []
    plan_dir = os.path.join(project_root, ".rick", "jobs", job_id, "plan")

    if not os.path.isdir(plan_dir):
        errors.append(f"plan directory does not exist: {plan_dir}")
        return errors

    # Check OKR.md exists (job_11: mandatory)
    errors.extend(check_file_exists_and_has_content(
        os.path.join(plan_dir, "OKR.md"),
        required_substring="# "
    ))

    # Check task*.md files
    task_files = [f for f in os.listdir(plan_dir)
                  if f.startswith("task") and f.endswith(".md")]
    if not task_files:
        errors.append(f"no task*.md files found in {plan_dir}")
        return errors

    for task_file in task_files:
        task_path = os.path.join(plan_dir, task_file)
        with open(task_path, "r", encoding="utf-8") as f:
            content = f.read()
        required_sections = [
            "# 依赖关系", "# 任务名称", "# 任务目标", "# 关键结果", "# 测试方法"
        ]
        for section in required_sections:
            if section not in content:
                errors.append(f"{task_file} missing section: {section}")

    return errors


def check_doing(job_id, project_root):
    """Check doing directory structure (job_11: content validation included)."""
    errors = []
    doing_dir = os.path.join(project_root, ".rick", "jobs", job_id, "doing")

    # Check tasks.json
    tasks_json_path = os.path.join(doing_dir, "tasks.json")
    if not os.path.isfile(tasks_json_path):
        errors.append(f"tasks.json does not exist: {tasks_json_path}")
    else:
        try:
            with open(tasks_json_path) as f:
                tasks = json.load(f)
            for task in tasks:
                state = task.get("state_info", {})
                if state.get("status") == "running":
                    errors.append(f"zombie task detected: {task.get('task_id')}")
        except Exception as e:
            errors.append(f"failed to parse tasks.json: {e}")

    # Check debug.md: existence + non-empty + contains ## task records (job_11 pattern)
    debug_md_path = os.path.join(doing_dir, "debug.md")
    errors.extend(check_file_exists_and_has_content(
        debug_md_path,
        required_substring="## task"
    ))

    return errors


def check_learning(job_id, project_root):
    """Check learning directory structure (job_11: content validation included)."""
    errors = []
    learning_dir = os.path.join(project_root, ".rick", "jobs", job_id, "learning")

    # Check SUMMARY.md: existence + non-empty + contains # Job heading (job_11 pattern)
    summary_md_path = os.path.join(learning_dir, "SUMMARY.md")
    errors.extend(check_file_exists_and_has_content(
        summary_md_path,
        required_substring="# Job"
    ))

    # Check skills/*.py syntax
    skills_dir = os.path.join(learning_dir, "skills")
    if os.path.isdir(skills_dir):
        for skill_file in os.listdir(skills_dir):
            if skill_file.endswith(".py"):
                skill_path = os.path.join(skills_dir, skill_file)
                result = os.system(f"python3 -m py_compile {skill_path} 2>/dev/null")
                if result != 0:
                    errors.append(f"Python syntax error in skills/{skill_file}")

    return errors


CHECK_FUNCTIONS = {
    "plan": check_plan,
    "doing": check_doing,
    "learning": check_learning,
}


def main():
    parser = argparse.ArgumentParser(
        description="Rick tools check pattern example"
    )
    parser.add_argument("--job-id", required=True, help="Job ID (e.g. job_1)")
    parser.add_argument(
        "--check-type",
        choices=list(CHECK_FUNCTIONS.keys()),
        default="doing",
        help="Type of check to run"
    )
    args = parser.parse_args()

    project_root = find_project_root()
    if project_root is None:
        result = {"pass": False, "errors": ["could not find project root (.rick/ directory)"]}
        print(json.dumps(result))
        sys.exit(1)

    check_fn = CHECK_FUNCTIONS[args.check_type]
    errors = check_fn(args.job_id, project_root)

    result = {"pass": len(errors) == 0, "errors": errors}
    print(json.dumps(result))
    sys.exit(0 if result["pass"] else 1)


if __name__ == "__main__":
    main()
