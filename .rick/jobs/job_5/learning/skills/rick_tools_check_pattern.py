# Description: Pattern for implementing rick tools check commands (argparse + JSON output standard)
#!/usr/bin/env python3
"""
rick_tools_check_pattern - Pattern for implementing rick tools check commands

This skill demonstrates the standard pattern for building check/validation
scripts that follow the rick tools check command conventions:
- argparse for argument parsing
- JSON output: {"pass": bool, "errors": [...]}
- Exit code 0=pass, 1=fail

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


def check_plan(job_id, project_root):
    """Example: check plan directory structure."""
    errors = []
    plan_dir = os.path.join(project_root, ".rick", "jobs", job_id, "plan")

    if not os.path.isdir(plan_dir):
        errors.append(f"plan directory does not exist: {plan_dir}")
        return errors

    task_files = [f for f in os.listdir(plan_dir)
                  if f.startswith("task") and f.endswith(".md")]
    if not task_files:
        errors.append(f"no task*.md files found in {plan_dir}")

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
    """Example: check doing directory structure."""
    errors = []
    doing_dir = os.path.join(project_root, ".rick", "jobs", job_id, "doing")

    # Check tasks.json
    tasks_json = os.path.join(doing_dir, "tasks.json")
    if not os.path.isfile(tasks_json):
        errors.append(f"tasks.json does not exist: {tasks_json}")
    else:
        try:
            import json as _json
            with open(tasks_json) as f:
                tasks = _json.load(f)
            # Check for zombie tasks
            for task in tasks:
                state = task.get("state_info", {})
                if state.get("status") == "running":
                    errors.append(f"zombie task detected: {task.get('task_id')}")
        except Exception as e:
            errors.append(f"failed to parse tasks.json: {e}")

    # Check debug.md (mandatory work log)
    debug_md = os.path.join(doing_dir, "debug.md")
    if not os.path.isfile(debug_md):
        errors.append(f"debug.md does not exist (mandatory work log): {debug_md}")

    return errors


def check_learning(job_id, project_root):
    """Example: check learning directory structure."""
    errors = []
    learning_dir = os.path.join(project_root, ".rick", "jobs", job_id, "learning")

    # Check SUMMARY.md
    summary_md = os.path.join(learning_dir, "SUMMARY.md")
    if not os.path.isfile(summary_md):
        errors.append(f"SUMMARY.md does not exist: {summary_md}")

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
