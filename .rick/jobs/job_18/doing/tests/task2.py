#!/usr/bin/env python3
import json
import sys
import os
import subprocess

def main():
    errors = []

    # 获取项目根目录（6 次 dirname）
    project_root = os.path.dirname(os.path.dirname(os.path.dirname(
        os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__)))))))

    plan_md = os.path.join(project_root, "internal", "prompt", "templates", "plan.md")
    plan_prompt_go = os.path.join(project_root, "internal", "prompt", "plan_prompt.go")

    # Test 1: plan.md 不含 sense_skill_path
    try:
        with open(plan_md, "r", encoding="utf-8") as f:
            content = f.read()
        if "sense_skill_path" in content:
            errors.append("plan.md 仍含 sense_skill_path 引用，应已删除")
    except Exception as e:
        errors.append(f"读取 plan.md 失败: {e}")

    # Test 2: plan.md 含 grilling_skill_path
    try:
        with open(plan_md, "r", encoding="utf-8") as f:
            content = f.read()
        if "grilling_skill_path" not in content:
            errors.append("plan.md 未含 grilling_skill_path，应已添加 grilling 步骤")
    except Exception as e:
        errors.append(f"读取 plan.md 失败: {e}")

    # Test 3: plan_prompt.go 不含 sense_skill_path
    try:
        with open(plan_prompt_go, "r", encoding="utf-8") as f:
            content = f.read()
        if "sense_skill_path" in content:
            errors.append("plan_prompt.go 仍含 sense_skill_path，应已替换为 grilling_skill_path")
    except Exception as e:
        errors.append(f"读取 plan_prompt.go 失败: {e}")

    # Test 4: plan_prompt.go 含 grilling_skill_path
    try:
        with open(plan_prompt_go, "r", encoding="utf-8") as f:
            content = f.read()
        if "grilling_skill_path" not in content:
            errors.append("plan_prompt.go 未含 grilling_skill_path")
    except Exception as e:
        errors.append(f"读取 plan_prompt.go 失败: {e}")

    # Test 5: 构建并验证 dry-run 输出
    build_tool = os.path.join(project_root, ".rick", "tools", "build_and_get_rick_bin.py")
    try:
        result = subprocess.run(
            ["python3", build_tool],
            capture_output=True, text=True, cwd=project_root
        )
        build_output = json.loads(result.stdout)
        if not build_output.get("pass"):
            errors.append(f"构建失败: {build_output.get('errors', [])}")
            # 构建失败则跳过后续 binary 测试
        else:
            rick_bin = build_output.get("bin_path", os.path.join(project_root, "bin", "rick"))
            _run_dryrun_tests(rick_bin, project_root, errors)
    except Exception as e:
        errors.append(f"调用 build_and_get_rick_bin.py 失败: {e}")

    result = {
        "pass": len(errors) == 0,
        "errors": errors
    }
    print(json.dumps(result, ensure_ascii=False))
    sys.exit(0 if result["pass"] else 1)


def _run_dryrun_tests(rick_bin, project_root, errors):
    # Test 5a: dry-run 输出含 skill_grilling
    try:
        result = subprocess.run(
            [rick_bin, "plan", "--dry-run"],
            capture_output=True, text=True, cwd=project_root
        )
        output = result.stdout
        if "skill_grilling" not in output:
            errors.append("plan --dry-run 输出不含 skill_grilling")
        if "sense_skill_path" in output:
            errors.append("plan --dry-run 输出仍含 sense_skill_path")
        if "{{grilling_skill_path}}" in output:
            errors.append("plan --dry-run 输出含未替换的 {{grilling_skill_path}} 占位符")
    except Exception as e:
        errors.append(f"执行 plan --dry-run 失败: {e}")


if __name__ == "__main__":
    main()
