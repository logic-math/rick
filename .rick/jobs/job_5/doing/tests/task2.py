#!/usr/bin/env python3
import json
import sys
import os
import subprocess

def main():
    errors = []

    project_root = "/Users/sunquan/ai_coding/CODING/rick"
    template_file = os.path.join(project_root, "internal/prompt/templates/doing.md")

    # Test 1: 文件存在
    if not os.path.exists(template_file):
        errors.append(f"doing.md template file does not exist: {template_file}")
        print(json.dumps({"pass": False, "errors": errors}))
        sys.exit(1)

    try:
        with open(template_file, "r", encoding="utf-8") as f:
            content = f.read()
    except Exception as e:
        errors.append(f"Failed to read doing.md: {str(e)}")
        print(json.dumps({"pass": False, "errors": errors}))
        sys.exit(1)

    # Test 2: 不再包含"遇到问题才记录"等软性表述
    soft_phrases = ["遇到问题时才记录", "遇到问题才记录", "只有遇到问题", "出现问题时记录"]
    for phrase in soft_phrases:
        if phrase in content:
            errors.append(f'doing.md still contains soft phrasing: "{phrase}"')

    # Test 3: 包含强制约束关键词（强制、必须）
    hard_keywords = ["强制", "必须"]
    for kw in hard_keywords:
        if kw not in content:
            errors.append(f'doing.md missing hard constraint keyword: "{kw}"')

    # Test 4: 包含四个必填部分
    required_sections = ["分析过程", "实现步骤", "遇到的问题", "验证结果"]
    for section in required_sections:
        if section not in content:
            errors.append(f'doing.md missing required section: "{section}"')

    # Test 5: 包含"在 git commit 之前必须先更新 debug.md"的约束
    git_commit_constraint = "在 git commit 之前必须先更新 debug.md"
    if git_commit_constraint not in content:
        errors.append(f'doing.md missing constraint: "{git_commit_constraint}"')

    # Test 6: go build 验证编译正常（模板是 embedded，编译时包含）
    try:
        result = subprocess.run(
            ["go", "build", "./..."],
            cwd=project_root,
            capture_output=True,
            text=True,
            timeout=60
        )
        if result.returncode != 0:
            errors.append(f"go build failed: {result.stderr.strip()}")
    except subprocess.TimeoutExpired:
        errors.append("go build timed out after 60 seconds")
    except Exception as e:
        errors.append(f"Failed to run go build: {str(e)}")

    result = {
        "pass": len(errors) == 0,
        "errors": errors
    }

    print(json.dumps(result))
    sys.exit(0 if result["pass"] else 1)

if __name__ == "__main__":
    main()
