#!/usr/bin/env python3
import json
import sys
import os
import subprocess

def main():
    errors = []

    # Project root: 6 dirnames from this file
    # tests/ -> doing/ -> job_16/ -> jobs/ -> .rick/ -> rick/
    script_dir = os.path.dirname(os.path.abspath(__file__))
    project_root = script_dir
    for _ in range(5):
        project_root = os.path.dirname(project_root)

    templates_dir = os.path.join(project_root, "internal", "prompt", "templates")
    doing_md = os.path.join(templates_dir, "doing.md")
    plan_md = os.path.join(templates_dir, "plan.md")
    super_debugging_file = os.path.join(templates_dir, "skills", "super-debugging-zh.md")

    # Test 1: super-debugging-zh.md is deleted
    if os.path.exists(super_debugging_file):
        errors.append("internal/prompt/templates/skills/super-debugging-zh.md 仍存在，应已删除")

    # Test 2: no old references in templates/
    try:
        result = subprocess.run(
            ["git", "grep", "-l", r"super_debugging\|super-debugging", templates_dir],
            capture_output=True, text=True, cwd=project_root
        )
        if result.stdout.strip():
            errors.append(f"模板目录中仍有旧引用 (super_debugging/super-debugging): {result.stdout.strip()}")
    except Exception as e:
        errors.append(f"执行 git grep 失败: {str(e)}")

    # Test 3: doing.md contains debug_skill_path
    try:
        with open(doing_md, "r", encoding="utf-8") as f:
            doing_content = f.read()
        if "debug_skill_path" not in doing_content:
            errors.append("doing.md 缺少 {{debug_skill_path}} 占位符")
        if "debug-skill" not in doing_content:
            errors.append("doing.md 缺少 skill:debug-skill 声明")
    except Exception as e:
        errors.append(f"读取 doing.md 失败: {str(e)}")
        doing_content = ""

    # Test 4: plan.md contains debug_skill_path
    try:
        with open(plan_md, "r", encoding="utf-8") as f:
            plan_content = f.read()
        if "debug_skill_path" not in plan_content:
            errors.append("plan.md 缺少 {{debug_skill_path}} 占位符")
    except Exception as e:
        errors.append(f"读取 plan.md 失败: {str(e)}")

    # Test 5: doing.md preserves task execution log format
    if doing_content and "## task" not in doing_content:
        errors.append("doing.md 缺少 task 执行日志格式 (## task)")

    # Test 6 (boundary): doing.md no longer has debug{N} format
    if doing_content and "## debug{N}" in doing_content:
        errors.append("doing.md 中旧的 debug{N} 格式未清理")

    # Test 7: doing.md contains sense_skill_path
    if doing_content and "sense_skill_path" not in doing_content:
        errors.append("doing.md 缺少 {{sense_skill_path}} 占位符")

    result = {
        "pass": len(errors) == 0,
        "errors": errors
    }

    print(json.dumps(result, ensure_ascii=False))
    sys.exit(0 if result["pass"] else 1)

if __name__ == "__main__":
    main()
