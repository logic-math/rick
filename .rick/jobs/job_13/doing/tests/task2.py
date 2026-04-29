#!/usr/bin/env python3
import json
import sys
import os
import subprocess

def main():
    errors = []

    template_path = "/Users/sunquan/ai_coding/CODING/rick/internal/prompt/templates/human_loop.md"

    # Test 1: 检查路径占位符存在
    for placeholder in ["{{think_agent_path}}", "{{learn_agent_path}}", "{{express_agent_path}}"]:
        result = subprocess.run(
            ["grep", "-q", placeholder, template_path],
            capture_output=True
        )
        if result.returncode != 0:
            errors.append(f"Missing placeholder: {placeholder}")

    # Test 2: 检查不含斜杠命令
    result = subprocess.run(
        ["grep", "-qE", "/sense-human-loop|/human-loop", template_path],
        capture_output=True
    )
    if result.returncode == 0:
        errors.append("Template contains forbidden slash commands (/sense-human-loop or /human-loop)")

    # Test 3: 检查包含复杂度判断
    result = subprocess.run(
        ["grep", "-q", "Level 1", template_path],
        capture_output=True
    )
    if result.returncode != 0:
        errors.append("Template missing complexity judgment (Level 1)")

    result = {
        "pass": len(errors) == 0,
        "errors": errors
    }

    print(json.dumps(result))
    sys.exit(0 if result["pass"] else 1)

if __name__ == "__main__":
    main()
