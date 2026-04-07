# Description: Mock AI agent for integration testing - simulates 11 claude scenarios without real API calls
#!/usr/bin/env python3
"""
mock_agent_testing - Pattern for testing AI agent integrations without real Claude

This skill demonstrates how to build a mock AI agent for integration testing.
The mock agent simulates different scenarios (success/failure/timeout) based
on a scenario name passed via environment variable or argument.

Usage:
    # As a drop-in replacement for claude CLI in tests:
    MOCK_SCENARIO=plan_success python3 mock_agent_testing.py --dangerously-skip-permissions prompt.md

    # List available scenarios:
    python3 mock_agent_testing.py --list-scenarios

Output:
    Simulates the behavior of a real AI agent for the given scenario.
    Exit code matches what a real agent would return.

Supported scenarios:
    plan_success        - Creates valid task files
    plan_missing_section - Creates task files missing required sections
    plan_circular_dep   - Creates tasks with circular dependencies
    doing_success       - Creates valid tasks.json and debug.md
    doing_no_debug      - Creates tasks.json but no debug.md
    doing_zombie        - Creates tasks.json with zombie task
    learning_success    - Creates valid SUMMARY.md and skills
    learning_bad_skill  - Creates skills with Python syntax errors
    learning_no_summary - Creates learning dir without SUMMARY.md
    claude_exit_nonzero - Exits with code 1 (simulates claude failure)
    claude_timeout      - Sleeps for 60s (simulates timeout)
"""

import argparse
import json
import os
import sys
import time


SCENARIOS = {
    "plan_success": "Creates valid task files in plan directory",
    "plan_missing_section": "Creates task files missing required sections",
    "plan_circular_dep": "Creates tasks with circular dependencies",
    "doing_success": "Creates valid tasks.json and debug.md",
    "doing_no_debug": "Creates tasks.json but no debug.md",
    "doing_zombie": "Creates tasks.json with zombie task",
    "learning_success": "Creates valid SUMMARY.md and skills",
    "learning_bad_skill": "Creates skills with Python syntax errors",
    "learning_no_summary": "Creates learning dir without SUMMARY.md",
    "claude_exit_nonzero": "Exits with code 1",
    "claude_timeout": "Sleeps 60s to simulate timeout",
}


def find_project_root():
    """Walk up from current directory to find project root."""
    current = os.getcwd()
    while current != os.path.dirname(current):
        if os.path.isdir(os.path.join(current, ".rick")):
            return current
        current = os.path.dirname(current)
    return os.getcwd()


def run_scenario(scenario, prompt_file):
    """Execute the given scenario."""
    project_root = find_project_root()

    if scenario == "claude_exit_nonzero":
        print("Mock agent: simulating failure", file=sys.stderr)
        sys.exit(1)

    if scenario == "claude_timeout":
        print("Mock agent: simulating timeout (sleeping 60s)...", file=sys.stderr)
        time.sleep(60)
        sys.exit(0)

    if scenario == "plan_success":
        # Read prompt to find job_id and plan dir
        plan_dir = os.path.join(project_root, ".rick", "jobs", "job_test", "plan")
        os.makedirs(plan_dir, exist_ok=True)
        for i in range(1, 3):
            task_content = f"""# 依赖关系
{"" if i == 1 else "task1"}

# 任务名称
Test Task {i}

# 任务目标
Test objective for task {i}

# 关键结果
1. Result {i}

# 测试方法
1. Run test {i}
"""
            with open(os.path.join(plan_dir, f"task{i}.md"), "w") as f:
                f.write(task_content)
        print("Mock agent: created valid plan files")
        sys.exit(0)

    if scenario == "doing_success":
        doing_dir = os.path.join(project_root, ".rick", "jobs", "job_test", "doing")
        os.makedirs(doing_dir, exist_ok=True)
        tasks = [
            {"task_id": "task1", "task_name": "Task 1", "dep": [],
             "state_info": {"status": "success", "commit_hash": "abc123"}}
        ]
        with open(os.path.join(doing_dir, "tasks.json"), "w") as f:
            json.dump(tasks, f)
        with open(os.path.join(doing_dir, "debug.md"), "w") as f:
            f.write("# debug1: Test entry\n\n**分析过程**: Normal execution.\n")
        print("Mock agent: created valid doing files")
        sys.exit(0)

    if scenario == "learning_success":
        learning_dir = os.path.join(project_root, ".rick", "jobs", "job_test", "learning")
        os.makedirs(os.path.join(learning_dir, "skills"), exist_ok=True)
        with open(os.path.join(learning_dir, "SUMMARY.md"), "w") as f:
            f.write("<!-- APPROVED: false -->\n# Job job_test 执行总结\n\n## 执行概述\n\nTest summary.\n")
        skill_content = '''#!/usr/bin/env python3
"""Test skill."""
import json
print(json.dumps({"pass": True, "errors": []}))
'''
        with open(os.path.join(learning_dir, "skills", "test_skill.py"), "w") as f:
            f.write(skill_content)
        print("Mock agent: created valid learning files")
        sys.exit(0)

    # Default: print scenario info and exit
    print(f"Mock agent: scenario '{scenario}' - {SCENARIOS.get(scenario, 'unknown')}")
    sys.exit(0)


def main():
    parser = argparse.ArgumentParser(
        description="Mock AI agent for rick integration testing"
    )
    parser.add_argument(
        "--list-scenarios", action="store_true",
        help="List all available scenarios"
    )
    parser.add_argument(
        "--dangerously-skip-permissions", action="store_true",
        help="Compatibility flag (ignored)"
    )
    parser.add_argument(
        "prompt_file", nargs="?",
        help="Prompt file path (ignored, for CLI compatibility)"
    )
    args = parser.parse_args()

    if args.list_scenarios:
        for name, desc in SCENARIOS.items():
            print(f"  {name:<30} {desc}")
        sys.exit(0)

    scenario = os.environ.get("MOCK_SCENARIO", "doing_success")
    if scenario not in SCENARIOS:
        print(f"Unknown scenario: {scenario}", file=sys.stderr)
        print(f"Available: {', '.join(SCENARIOS.keys())}", file=sys.stderr)
        sys.exit(1)

    run_scenario(scenario, args.prompt_file)


if __name__ == "__main__":
    main()
