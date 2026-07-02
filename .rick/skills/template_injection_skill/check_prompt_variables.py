# Description: 验证 rick dry-run 输出的 prompt 中是否包含指定的变量或关键词

import argparse
import json
import subprocess
import sys
import os


def check_plan_prompt(project_root, keywords):
    """Run rick plan --dry-run and check for keywords."""
    rick_bin = os.path.join(project_root, "bin", "rick")
    if not os.path.exists(rick_bin):
        return False, f"rick binary not found at {rick_bin}"

    try:
        result = subprocess.run(
            [rick_bin, "plan", "test requirement", "--dry-run"],
            capture_output=True,
            text=True,
            timeout=30,
            cwd=project_root,
        )
        output = result.stdout + result.stderr
        missing = [kw for kw in keywords if kw not in output]
        if missing:
            return False, f"plan prompt missing keywords: {missing}"
        return True, "all keywords found in plan prompt"
    except subprocess.TimeoutExpired:
        return False, "rick plan --dry-run timed out"
    except Exception as e:
        return False, f"error running rick plan --dry-run: {e}"


def check_doing_prompt(project_root, job_id, keywords):
    """Run rick doing <job_id> --dry-run and check for keywords."""
    rick_bin = os.path.join(project_root, "bin", "rick")
    if not os.path.exists(rick_bin):
        return False, f"rick binary not found at {rick_bin}"

    try:
        result = subprocess.run(
            [rick_bin, "doing", job_id, "--dry-run"],
            capture_output=True,
            text=True,
            timeout=30,
            cwd=project_root,
        )
        output = result.stdout + result.stderr
        missing = [kw for kw in keywords if kw not in output]
        if missing:
            return False, f"关键词未找到: doing prompt does not contain: {missing}"
        return True, "all keywords found in doing prompt"
    except subprocess.TimeoutExpired:
        return False, "rick doing --dry-run timed out"
    except Exception as e:
        return False, f"error running rick doing --dry-run: {e}"


def check_testing_prompt(project_root, keywords):
    """Read test_python.md template and check for keywords."""
    template_path = os.path.join(
        project_root, "internal", "prompt", "templates", "test_python.md"
    )
    if not os.path.exists(template_path):
        return False, f"test_python.md template not found at {template_path}"
    try:
        with open(template_path, "r", encoding="utf-8") as f:
            content = f.read()
        missing = [kw for kw in keywords if kw not in content]
        if missing:
            return False, f"关键词未找到: testing prompt does not contain: {missing}"
        return True, "all keywords found in testing prompt"
    except Exception as e:
        return False, f"error reading test_python.md: {e}"


def check_learning_prompt(project_root, job_id, keywords):
    """Run rick learning <job_id> --dry-run and check for keywords."""
    rick_bin = os.path.join(project_root, "bin", "rick")
    if not os.path.exists(rick_bin):
        return False, f"rick binary not found at {rick_bin}"

    try:
        result = subprocess.run(
            [rick_bin, "learning", job_id, "--dry-run"],
            capture_output=True,
            text=True,
            timeout=30,
            cwd=project_root,
        )
        output = result.stdout + result.stderr
        missing = [kw for kw in keywords if kw not in output]
        if missing:
            return False, f"关键词未找到: learning prompt does not contain: {missing}"
        return True, "all keywords found in learning prompt"
    except subprocess.TimeoutExpired:
        return False, "rick learning --dry-run timed out"
    except Exception as e:
        return False, f"error running rick learning --dry-run: {e}"


def check_human_loop_prompt(project_root, topic, keywords):
    """Run rick human-loop <topic> --dry-run and check for keywords."""
    rick_bin = os.path.join(project_root, "bin", "rick")
    if not os.path.exists(rick_bin):
        return False, f"rick binary not found at {rick_bin}"

    try:
        result = subprocess.run(
            [rick_bin, "human-loop", topic, "--dry-run"],
            capture_output=True,
            text=True,
            timeout=30,
            cwd=project_root,
        )
        output = result.stdout + result.stderr
        missing = [kw for kw in keywords if kw not in output]
        if missing:
            return False, f"human-loop prompt missing keywords: {missing}"
        return True, "all keywords found in human-loop prompt"
    except subprocess.TimeoutExpired:
        return False, "rick human-loop --dry-run timed out"
    except Exception as e:
        return False, f"error running rick human-loop --dry-run: {e}"


def check_dream_prompt(project_root, keywords):
    """Run rick dream --dry-run and check for keywords."""
    rick_bin = os.path.join(project_root, "bin", "rick")
    if not os.path.exists(rick_bin):
        return False, f"rick binary not found at {rick_bin}"

    try:
        result = subprocess.run(
            [rick_bin, "dream", "--dry-run"],
            capture_output=True,
            text=True,
            timeout=30,
            cwd=project_root,
        )
        output = result.stdout + result.stderr
        missing = [kw for kw in keywords if kw not in output]
        if missing:
            return False, f"dream prompt missing keywords: {missing}"
        return True, "all keywords found in dream prompt"
    except subprocess.TimeoutExpired:
        return False, "rick dream --dry-run timed out"
    except Exception as e:
        return False, f"error running rick dream --dry-run: {e}"


def run_tests(project_root):
    """Built-in self-tests."""
    errors = []

    # Test: rick binary exists
    rick_bin = os.path.join(project_root, "bin", "rick")
    if not os.path.exists(rick_bin):
        errors.append(f"rick binary not found: {rick_bin}")
        return errors

    # Test: plan --dry-run produces output
    result = subprocess.run(
        [rick_bin, "plan", "test", "--dry-run"],
        capture_output=True,
        text=True,
        timeout=30,
        cwd=project_root,
    )
    if not result.stdout and not result.stderr:
        errors.append("plan --dry-run produced no output")

    return errors


def main():
    parser = argparse.ArgumentParser(
        description="Verify rick dry-run prompt output contains expected keywords"
    )
    parser.add_argument("--test", action="store_true", help="Run built-in tests")
    parser.add_argument(
        "--phase",
        choices=["plan", "doing", "learning", "human-loop", "dream", "testing"],
        default="plan",
        help="Which phase to check (default: plan)",
    )
    parser.add_argument(
        "--job", default="job_1", help="Job ID for doing/learning phases (default: job_1)"
    )
    parser.add_argument(
        "--topic", default="测试主题", help="Topic for human-loop phase (default: 测试主题)"
    )
    parser.add_argument(
        "--keywords",
        nargs="+",
        default=[],
        help="Keywords to check for in the prompt output",
    )
    parser.add_argument(
        "--project-root",
        default=os.getcwd(),
        help="Project root directory (default: current directory)",
    )
    args = parser.parse_args()

    if args.test:
        errors = run_tests(args.project_root)
        result = {"pass": len(errors) == 0, "errors": errors}
        print(json.dumps(result, ensure_ascii=False))
        sys.exit(0 if result["pass"] else 1)

    if not args.keywords:
        print(json.dumps({"pass": False, "errors": ["--keywords is required"]}))
        sys.exit(1)

    if args.phase == "plan":
        ok, msg = check_plan_prompt(args.project_root, args.keywords)
    elif args.phase == "doing":
        ok, msg = check_doing_prompt(args.project_root, args.job, args.keywords)
    elif args.phase == "learning":
        ok, msg = check_learning_prompt(args.project_root, args.job, args.keywords)
    elif args.phase == "human-loop":
        ok, msg = check_human_loop_prompt(args.project_root, args.topic, args.keywords)
    elif args.phase == "dream":
        ok, msg = check_dream_prompt(args.project_root, args.keywords)
    elif args.phase == "testing":
        ok, msg = check_testing_prompt(args.project_root, args.keywords)

    result = {"pass": ok, "errors": [] if ok else [msg]}
    print(json.dumps(result, ensure_ascii=False))
    sys.exit(0 if ok else 1)


if __name__ == "__main__":
    main()
