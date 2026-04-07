#!/usr/bin/env python3
import json
import sys
import os
import subprocess

def main():
    errors = []

    project_root = "/Users/sunquan/ai_coding/CODING/rick"
    learning_md = os.path.join(project_root, "internal/prompt/templates/learning.md")

    # Test 1: 检查文件存在
    if not os.path.exists(learning_md):
        errors.append(f"learning.md does not exist: {learning_md}")
        print(json.dumps({"pass": False, "errors": errors}))
        sys.exit(1)

    try:
        with open(learning_md, "r") as f:
            content = f.read()
    except Exception as e:
        errors.append(f"Failed to read learning.md: {str(e)}")
        print(json.dumps({"pass": False, "errors": errors}))
        sys.exit(1)

    # Test 2: 不包含 skills/*.py
    if "skills/*.py" in content:
        errors.append("learning.md still contains 'skills/*.py' (should be removed)")

    # Test 3: 包含 tools/*.py
    if "tools/*.py" not in content:
        errors.append("learning.md does not contain 'tools/*.py'")

    # Test 4: 包含 skills/*.md
    if "skills/*.md" not in content:
        errors.append("learning.md does not contain 'skills/*.md'")

    # Test 5: go build
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
    except Exception as e:
        errors.append(f"go build error: {str(e)}")

    result = {
        "pass": len(errors) == 0,
        "errors": errors
    }
    print(json.dumps(result))
    sys.exit(0 if result["pass"] else 1)

if __name__ == "__main__":
    main()
