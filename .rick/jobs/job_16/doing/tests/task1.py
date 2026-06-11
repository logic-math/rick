#!/usr/bin/env python3
import json
import sys
import os

def main():
    errors = []

    script_dir = os.path.dirname(os.path.abspath(__file__))
    project_root = os.path.dirname(os.path.dirname(os.path.dirname(os.path.dirname(os.path.dirname(script_dir)))))

    debug_skill_path = os.path.join(project_root, "internal", "prompt", "templates", "skills", "debug_skill.md")

    # Test 1: file exists
    if not os.path.exists(debug_skill_path):
        errors.append(f"debug_skill.md does not exist at {debug_skill_path}")
        result = {"pass": False, "errors": errors}
        print(json.dumps(result, ensure_ascii=False))
        sys.exit(1)

    try:
        with open(debug_skill_path, "r", encoding="utf-8") as f:
            content = f.read()
    except Exception as e:
        errors.append(f"Failed to read debug_skill.md: {str(e)}")
        result = {"pass": False, "errors": errors}
        print(json.dumps(result, ensure_ascii=False))
        sys.exit(1)

    lines = content.splitlines()

    # Test 2: frontmatter contains name: debug-skill
    first5 = "\n".join(lines[:5])
    if "name: debug-skill" not in first5:
        errors.append(f"debug_skill.md head-5 does not contain 'name: debug-skill', got: {first5!r}")

    # Test 3: contains >= 4 stage keywords (阶段一建立假设, 阶段二简化复现, 阶段三简化复现, 阶段三建立传播链)
    stage_keywords = ["阶段一", "阶段二", "阶段三"]
    sub_keywords = ["建立假设", "简化复现", "建立传播链"]
    matched_stages = sum(1 for kw in stage_keywords if kw in content)
    matched_sub = sum(1 for kw in sub_keywords if kw in content)
    total_stage_matches = matched_stages + matched_sub
    if total_stage_matches < 4:
        errors.append(f"Expected >= 4 stage keyword matches (阶段一/二/三 + 建立假设/简化复现/建立传播链), got {total_stage_matches}")

    # Test 4: contains review debug agent protocol with two trigger points (两个触发点)
    if "review debug agent" not in content.lower() and "review-debug" not in content.lower() and "review debug" not in content:
        errors.append("debug_skill.md missing 'review debug agent' protocol")
    trigger_count = content.count("触发点") + content.count("trigger")
    if trigger_count < 2:
        errors.append(f"Expected >= 2 trigger point references, got {trigger_count}")

    # Test 5: contains runtime observation tools guidance
    runtime_keywords = ["运行时", "观察", "工具"]
    if not all(kw in content for kw in runtime_keywords):
        missing = [kw for kw in runtime_keywords if kw not in content]
        errors.append(f"debug_skill.md missing runtime observation tool keywords: {missing}")

    # Test 6: contains debug/ directory file format convention
    if "debug/" not in content and "debug目录" not in content and "debug 目录" not in content:
        errors.append("debug_skill.md missing debug/ directory convention")

    # Test 7: contains rollback constraint and loop limit (回滚约束, 循环上限)
    if "回滚" not in content:
        errors.append("debug_skill.md missing rollback constraint (回滚约束)")
    if "循环上限" not in content and "上限" not in content:
        errors.append("debug_skill.md missing loop limit (循环上限)")

    result = {
        "pass": len(errors) == 0,
        "errors": errors
    }
    print(json.dumps(result, ensure_ascii=False))
    sys.exit(0 if result["pass"] else 1)

if __name__ == "__main__":
    main()
