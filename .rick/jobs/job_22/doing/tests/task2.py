#!/usr/bin/env python3
import json
import sys
import os
import subprocess
import re

def get_project_root():
    # 6 dirnames up from this file: tests/ → doing/ → job_22/ → jobs/ → .rick/ → project root
    path = os.path.abspath(__file__)
    for _ in range(6):
        path = os.path.dirname(path)
    return path

def main():
    errors = []
    project_root = get_project_root()
    debug_skill = os.path.join(project_root, "internal", "prompt", "templates", "skills", "debug_skill.md")

    # Test 1: debug_skill.md 存在
    if not os.path.exists(debug_skill):
        errors.append(f"debug_skill.md 不存在: {debug_skill}")
        print(json.dumps({"pass": False, "errors": errors}, ensure_ascii=False))
        sys.exit(1)

    try:
        with open(debug_skill, "r", encoding="utf-8") as f:
            content = f.read()
    except Exception as e:
        errors.append(f"读取 debug_skill.md 失败: {str(e)}")
        print(json.dumps({"pass": False, "errors": errors}, ensure_ascii=False))
        sys.exit(1)

    # Test 2: Phase 1-6 全部存在（正文 "Phase [1-6]"，至少 6 处）
    phase_matches = re.findall(r"Phase [1-6]", content)
    if len(phase_matches) < 6:
        errors.append(f"Phase 1-6 出现次数不足 6，实际: {len(phase_matches)}")

    # Test 3: 旧阶段名已消失
    old_names = ["源码推理法", "增量调试法", "科学实验法"]
    for name in old_names:
        if name in content:
            errors.append(f"旧阶段名仍存在: {name}")

    # Test 4: 保留内容验证（review debug agent 协议、sense 路径、frontmatter 字段）
    preserved_patterns = [
        (r"review debug agent", "review debug agent 协议"),
        (r"sense_skill_path", "sense_skill_path 变量"),
        (r"summary|status", "frontmatter summary/status 字段"),
    ]
    for pattern, desc in preserved_patterns:
        if not re.search(pattern, content):
            errors.append(f"保留内容缺失: {desc}")

    # Test 5: ## Phase [1-6] 章节标题恰好 6 个
    heading_matches = re.findall(r"^## Phase [1-6]", content, re.MULTILINE)
    if len(heading_matches) != 6:
        errors.append(f"'## Phase [1-6]' 章节标题数量应为 6，实际: {len(heading_matches)}")

    # Test 6: 二进制构建验证
    build_script = os.path.join(project_root, "scripts", "build.sh")
    rick_bin = os.path.join(project_root, "bin", "rick")
    try:
        result = subprocess.run(
            ["bash", build_script],
            cwd=project_root,
            capture_output=True,
            text=True,
            timeout=120,
        )
        if result.returncode != 0:
            errors.append(f"build.sh 失败 (exit {result.returncode}): {result.stderr[:300]}")
        else:
            ver_result = subprocess.run(
                [rick_bin, "--version"],
                cwd=project_root,
                capture_output=True,
                text=True,
                timeout=10,
            )
            if ver_result.returncode != 0:
                errors.append(f"rick --version 失败: {ver_result.stderr[:200]}")
    except subprocess.TimeoutExpired:
        errors.append("build.sh 超时")
    except Exception as e:
        errors.append(f"构建过程异常: {str(e)}")

    # Test 7: 单元测试 TestEmbedded
    try:
        test_result = subprocess.run(
            ["go", "test", "./internal/prompt/...", "-run", "TestEmbedded", "-v"],
            cwd=project_root,
            capture_output=True,
            text=True,
            timeout=60,
        )
        if test_result.returncode != 0:
            errors.append(f"TestEmbedded 单元测试失败: {test_result.stdout[-300:]}{test_result.stderr[-200:]}")
    except subprocess.TimeoutExpired:
        errors.append("go test TestEmbedded 超时")
    except Exception as e:
        errors.append(f"go test 异常: {str(e)}")

    result = {
        "pass": len(errors) == 0,
        "errors": errors,
    }
    print(json.dumps(result, ensure_ascii=False))
    sys.exit(0 if result["pass"] else 1)

if __name__ == "__main__":
    main()
