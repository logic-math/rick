#!/usr/bin/env python3
# I will use skill:tdd and skill:testing-anti-patterns for test generation.
import json
import sys
import os
import subprocess

def main():
    errors = []

    # 项目根目录（从 tests/ 向上 6 层）
    script_dir = os.path.dirname(os.path.abspath(__file__))
    project_root = script_dir
    for _ in range(5):
        project_root = os.path.dirname(project_root)

    skills_dir = os.path.join(project_root, "internal", "prompt", "templates", "skills")
    tdd_zh = os.path.join(skills_dir, "tdd-zh.md")
    manager_test = os.path.join(project_root, "internal", "prompt", "manager_test.go")

    # Test 1: 合并内容验证 — 六个关键词在 tdd-zh.md 中至少 5 行匹配
    keywords = ["测试用例四要素", "前置条件", "输入参数", "操作序列", "预期输出", "INSUFFICIENT_FUNDS"]
    try:
        with open(tdd_zh, "r", encoding="utf-8") as f:
            content = f.read()
        matched = sum(1 for kw in keywords if kw in content)
        if matched < 5:
            errors.append(
                f"tdd-zh.md 合并内容不足：关键词命中 {matched}/6，期望 >=5（缺失：{[kw for kw in keywords if kw not in content]}）"
            )
    except Exception as e:
        errors.append(f"读取 tdd-zh.md 失败: {e}")

    # Test 2: tdd-zh.md 包含四要素的四个 ### 子节标题
    four_elements = ["### 前置条件", "### 输入参数", "### 操作序列", "### 预期输出"]
    try:
        with open(tdd_zh, "r", encoding="utf-8") as f:
            content = f.read()
        missing = [h for h in four_elements if h not in content]
        if missing:
            errors.append(f"tdd-zh.md 缺少四要素子节标题: {missing}")
    except Exception as e:
        errors.append(f"读取 tdd-zh.md（四要素检查）失败: {e}")

    # Test 3: 死代码文件已删除
    deleted_files = [
        os.path.join(skills_dir, "tc.md"),
        os.path.join(skills_dir, "tdd.md"),
        os.path.join(skills_dir, "tdd", "testing-anti-patterns.md"),
    ]
    for f in deleted_files:
        if os.path.exists(f):
            errors.append(f"死代码文件未删除: {os.path.relpath(f, project_root)}")

    # Test 4: 保留文件仍存在
    preserved_files = [
        tdd_zh,
        os.path.join(skills_dir, "testing-anti-patterns-zh.md"),
    ]
    for f in preserved_files:
        if not os.path.exists(f):
            errors.append(f"保留文件丢失: {os.path.relpath(f, project_root)}")

    # Test 5: go test ./internal/prompt/... 通过
    try:
        result = subprocess.run(
            ["go", "test", "./internal/prompt/..."],
            cwd=project_root,
            capture_output=True,
            text=True,
            timeout=120,
        )
        if result.returncode != 0:
            errors.append(f"go test ./internal/prompt/... 失败:\n{result.stdout}\n{result.stderr}")
    except Exception as e:
        errors.append(f"运行 go test 失败: {e}")

    # Test 6: manager_test.go 无残留引用
    dead_patterns = ['"tc"', '"tdd"', 'tdd/testing-anti-patterns"']
    try:
        with open(manager_test, "r", encoding="utf-8") as f:
            mt_content = f.read()
        for pat in dead_patterns:
            if pat in mt_content:
                errors.append(f"manager_test.go 仍包含已删除 skill 引用: {pat}")
    except Exception as e:
        errors.append(f"读取 manager_test.go 失败: {e}")

    result = {"pass": len(errors) == 0, "errors": errors}
    print(json.dumps(result, ensure_ascii=False))
    sys.exit(0 if result["pass"] else 1)

if __name__ == "__main__":
    main()
