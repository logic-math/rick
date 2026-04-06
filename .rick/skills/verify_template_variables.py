# Description: 验证 prompt 模板变量是否被正确替换（不含未替换的 {{placeholder}} 占位符）

import argparse
import json
import sys
import re
import os


def check_file_for_placeholders(file_path):
    """Check a file for unreplaced template placeholders like {{variable_name}}."""
    if not os.path.exists(file_path):
        return False, f"File not found: {file_path}"

    with open(file_path, 'r', encoding='utf-8') as f:
        content = f.read()

    # Find all unreplaced placeholders
    placeholders = re.findall(r'\{\{[^}]+\}\}', content)
    if placeholders:
        return False, f"Found unreplaced placeholders: {placeholders}"

    return True, "No unreplaced placeholders found"


def check_string_for_placeholders(content, label="content"):
    """Check a string for unreplaced template placeholders."""
    placeholders = re.findall(r'\{\{[^}]+\}\}', content)
    if placeholders:
        return False, f"{label} contains unreplaced placeholders: {placeholders}"
    return True, f"{label} has no unreplaced placeholders"


def run_tests():
    """Built-in tests for the skill."""
    errors = []

    # Test 1: Clean content passes
    ok, msg = check_string_for_placeholders("/usr/local/bin/rick tools plan_check job_1", "clean content")
    if not ok:
        errors.append(f"Test 1 failed: {msg}")

    # Test 2: Content with placeholder fails
    ok, msg = check_string_for_placeholders("{{rick_bin_path}} tools plan_check {{job_id}}", "template content")
    if ok:
        errors.append("Test 2 failed: should have detected placeholders")

    # Test 3: Partial replacement fails
    ok, msg = check_string_for_placeholders("/usr/local/bin/rick tools plan_check {{job_id}}", "partial replacement")
    if ok:
        errors.append("Test 3 failed: should have detected remaining placeholder")

    # Test 4: Empty string passes
    ok, msg = check_string_for_placeholders("", "empty string")
    if not ok:
        errors.append(f"Test 4 failed: {msg}")

    if errors:
        return {"pass": False, "errors": errors}
    return {"pass": True, "errors": []}


def main():
    parser = argparse.ArgumentParser(
        description='Verify that prompt template variables have been correctly replaced'
    )
    parser.add_argument('--test', action='store_true', help='Run built-in tests')
    parser.add_argument('--file', type=str, help='Path to file to check for unreplaced placeholders')
    parser.add_argument('--string', type=str, help='String content to check for unreplaced placeholders')
    parser.add_argument('--label', type=str, default='content', help='Label for the content being checked')
    args = parser.parse_args()

    if args.test:
        result = run_tests()
        print(json.dumps(result))
        sys.exit(0 if result["pass"] else 1)

    if args.file:
        ok, msg = check_file_for_placeholders(args.file)
        result = {"pass": ok, "errors": [] if ok else [msg], "message": msg}
        print(json.dumps(result))
        sys.exit(0 if ok else 1)

    if args.string:
        ok, msg = check_string_for_placeholders(args.string, args.label)
        result = {"pass": ok, "errors": [] if ok else [msg], "message": msg}
        print(json.dumps(result))
        sys.exit(0 if ok else 1)

    # If no arguments, show usage
    parser.print_help()
    sys.exit(1)


if __name__ == "__main__":
    main()
