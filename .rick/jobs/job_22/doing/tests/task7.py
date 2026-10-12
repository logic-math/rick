#!/usr/bin/env python3
import json
import sys
import os
import subprocess
import re


def get_project_root():
    # .rick/jobs/job_22/doing/tests/task7.py → 6 dirname calls
    path = os.path.abspath(__file__)
    for _ in range(6):
        path = os.path.dirname(path)
    return path


def read_template(project_root, name):
    path = os.path.join(project_root, "internal", "prompt", "templates", name)
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

    # Test 1: easy.md — okr_content and spec_content removed
    try:
        content = read_template(project_root, "easy.md")
        for var in ["okr_content", "spec_content"]:
            if re.search(r"\{\{" + re.escape(var) + r"\}\}", content):
                errors.append(f"easy.md still contains {{{{{var}}}}} (should be removed)")
    except Exception as e:
        errors.append(f"Failed to check easy.md for removed vars: {str(e)}")

    # Test 2: dream.md — wiki_dir and tools_dir removed
    try:
        content = read_template(project_root, "dream.md")
        for var in ["wiki_dir", "tools_dir"]:
            if re.search(r"\{\{" + re.escape(var) + r"\}\}", content):
                errors.append(f"dream.md still contains {{{{{var}}}}} (should be removed)")
    except Exception as e:
        errors.append(f"Failed to check dream.md for removed vars: {str(e)}")

    # Test 3: easy.md — debug_content/debug preserved (boundary check)
    try:
        content = read_template(project_root, "easy.md")
        if not re.search(r"debug_content|debug", content):
            errors.append("easy.md missing debug_content/debug reference (must be preserved)")
    except Exception as e:
        errors.append(f"Failed to check debug_content in easy.md: {str(e)}")

    # Test 4: easy.md — loops_context injected
    try:
        content = read_template(project_root, "easy.md")
        if "loops_context" not in content:
            errors.append("easy.md missing loops_context (should be added)")
    except Exception as e:
        errors.append(f"Failed to check loops_context in easy.md: {str(e)}")

    # Test 5: dream.md — loops_context, loops_dir, skills_dir injected
    try:
        content = read_template(project_root, "dream.md")
        for var in ["loops_context", "loops_dir", "skills_dir"]:
            if var not in content:
                errors.append(f"dream.md missing {var} (should be added)")
    except Exception as e:
        errors.append(f"Failed to check new vars in dream.md: {str(e)}")

    # Build binary (also validates compilation)
    rick_bin = None
    try:
        rick_bin = build_and_get_bin(project_root)
    except Exception as e:
        errors.append(f"Build failed: {str(e)}")

    # Test 6: go test ./internal/prompt/... -run TestEasy
    try:
        result = subprocess.run(
            ["go", "test", "./internal/prompt/...", "-run", "TestEasy", "-v"],
            capture_output=True, text=True, cwd=project_root, timeout=60
        )
        if result.returncode != 0:
            errors.append(f"go test TestEasy failed:\n{(result.stdout + result.stderr)[:500]}")
    except Exception as e:
        errors.append(f"Failed to run go test: {str(e)}")

    if rick_bin:
        # Test 7: easy --dry-run — contains loops section, no spec/okr literals
        try:
            dr = subprocess.run(
                [rick_bin, "easy", "--dry-run"],
                capture_output=True, text=True, cwd=project_root, timeout=30
            )
            output = dr.stdout + dr.stderr
            if "可用的项目 Loops" not in output:
                errors.append("easy --dry-run output missing '可用的项目 Loops' section")
            if "{{spec_content}}" in output:
                errors.append("easy --dry-run output contains unreplaced {{spec_content}} literal")
            if "{{okr_content}}" in output:
                errors.append("easy --dry-run output contains unreplaced {{okr_content}} literal")
        except Exception as e:
            errors.append(f"easy --dry-run test failed: {str(e)}")

        # Test 8: dream --dry-run — contains loops_dir path, no wiki_dir/spec_path literals
        try:
            dr = subprocess.run(
                [rick_bin, "dream", "--dry-run"],
                capture_output=True, text=True, cwd=project_root, timeout=30
            )
            output = dr.stdout + dr.stderr
            if "loops_dir" not in output:
                errors.append("dream --dry-run output missing 'loops_dir' path")
            if "{{wiki_dir}}" in output:
                errors.append("dream --dry-run output contains unreplaced {{wiki_dir}} literal")
            if "{{spec_path}}" in output:
                errors.append("dream --dry-run output contains unreplaced {{spec_path}} literal")
        except Exception as e:
            errors.append(f"dream --dry-run test failed: {str(e)}")

    result = {"pass": len(errors) == 0, "errors": errors}
    print(json.dumps(result, ensure_ascii=False))
    sys.exit(0 if result["pass"] else 1)


if __name__ == "__main__":
    main()
