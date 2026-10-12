#!/usr/bin/env python3
import json
import sys
import os

def main():
    errors = []

    project_root = "/Users/sunquan/ai_coding/CODING/rick"

    # Test 1: 检查三个文件存在
    files = [
        "internal/prompt/templates/human_loop_think.md",
        "internal/prompt/templates/human_loop_learn.md",
        "internal/prompt/templates/human_loop_express.md",
    ]
    for f in files:
        path = os.path.join(project_root, f)
        if not os.path.exists(path):
            errors.append(f"{f} does not exist")

    # Test 2: 检查 think 包含假设追问格式
    think_path = os.path.join(project_root, "internal/prompt/templates/human_loop_think.md")
    if os.path.exists(think_path):
        try:
            with open(think_path, "r") as f:
                content = f.read()
                if "如果这个成立其实假设了" not in content:
                    errors.append("human_loop_think.md missing '如果这个成立其实假设了'")
        except Exception as e:
            errors.append(f"Failed to read human_loop_think.md: {str(e)}")

    # Test 3: 检查 learn 包含事实性断言触发
    learn_path = os.path.join(project_root, "internal/prompt/templates/human_loop_learn.md")
    if os.path.exists(learn_path):
        try:
            with open(learn_path, "r") as f:
                content = f.read()
                if "事实性的断言" not in content:
                    errors.append("human_loop_learn.md missing '事实性的断言'")
        except Exception as e:
            errors.append(f"Failed to read human_loop_learn.md: {str(e)}")

    # Test 4: 检查 express 包含固定文档结构
    express_path = os.path.join(project_root, "internal/prompt/templates/human_loop_express.md")
    if os.path.exists(express_path):
        try:
            with open(express_path, "r") as f:
                content = f.read()
                if "澄清问题（Subject）" not in content:
                    errors.append("human_loop_express.md missing '澄清问题（Subject）'")
        except Exception as e:
            errors.append(f"Failed to read human_loop_express.md: {str(e)}")

    result = {
        "pass": len(errors) == 0,
        "errors": errors,
    }

    print(json.dumps(result))
    sys.exit(0 if result["pass"] else 1)

if __name__ == "__main__":
    main()
