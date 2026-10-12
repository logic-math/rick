#!/usr/bin/env python3
import json
import sys
import os
import subprocess

def main():
    errors = []

    # project root: tests/ -> doing/ -> job_17/ -> jobs/ -> .rick/ -> rick/
    script_dir = os.path.dirname(os.path.abspath(__file__))
    project_root = os.path.dirname(os.path.dirname(os.path.dirname(os.path.dirname(os.path.dirname(script_dir)))))
    rick_dir = os.path.join(project_root, ".rick")

    spec_path = os.path.join(rick_dir, "SPEC.md")
    wiki_dir = os.path.join(rick_dir, "wiki")
    okr_path = os.path.join(rick_dir, "OKR.md")
    learning_workflow_path = os.path.join(wiki_dir, "learning_phase_workflow.md")

    # Test 1: SPEC.md 中无 tools merge / RFC-005 引用
    try:
        with open(spec_path, "r", encoding="utf-8") as f:
            spec_content = f.read()
        bad_patterns = ["tools merge", "rick merge", "RFC-005"]
        for pat in bad_patterns:
            if pat in spec_content:
                errors.append(f"SPEC.md 仍含 '{pat}' 引用，需清理")
    except Exception as e:
        errors.append(f"读取 SPEC.md 失败: {e}")

    # Test 2: wiki/ 下所有 .md 文件无 tools merge / RFC-005 残留
    try:
        result = subprocess.run(
            ["grep", "-rn", "tools merge", wiki_dir, "--include=*.md"],
            capture_output=True, text=True
        )
        if result.stdout.strip():
            errors.append(f"wiki/ 中仍含 'tools merge' 引用:\n{result.stdout.strip()}")
    except Exception as e:
        errors.append(f"grep wiki/ tools merge 失败: {e}")

    try:
        result = subprocess.run(
            ["grep", "-rn", "RFC-005", wiki_dir, "--include=*.md"],
            capture_output=True, text=True
        )
        if result.stdout.strip():
            errors.append(f"wiki/ 中仍含 'RFC-005' 引用:\n{result.stdout.strip()}")
    except Exception as e:
        errors.append(f"grep wiki/ RFC-005 失败: {e}")

    # Test 3: OKR.md KR2.2 不含 rick tools merge，且含 手动 或 人工
    try:
        with open(okr_path, "r", encoding="utf-8") as f:
            okr_content = f.read()
        # 找到 KR2.2 行
        kr22_lines = [l for l in okr_content.splitlines() if "KR2.2" in l]
        if not kr22_lines:
            errors.append("OKR.md 中找不到 KR2.2")
        else:
            kr22_text = "\n".join(kr22_lines)
            if "rick tools merge" in kr22_text:
                errors.append(f"OKR KR2.2 仍含 'rick tools merge': {kr22_text.strip()}")
            if "手动" not in kr22_text and "人工" not in kr22_text:
                errors.append(f"OKR KR2.2 缺少 '手动'/'人工' 关键词: {kr22_text.strip()}")
    except Exception as e:
        errors.append(f"读取 OKR.md 失败: {e}")

    # Test 4: learning_phase_workflow.md 含人工合并说明
    try:
        with open(learning_workflow_path, "r", encoding="utf-8") as f:
            lw_content = f.read()
        has_manual = any(kw in lw_content for kw in ["手动", "人工.*合并", "git merge"])
        # simple keyword check (no regex)
        if not ("手动" in lw_content or "人工" in lw_content or "git merge" in lw_content):
            errors.append("learning_phase_workflow.md 缺少人工合并说明（手动/人工/git merge）")
        # should NOT still say "tools merge 尚未实现" with RFC-005
        if "RFC-005" in lw_content:
            errors.append("learning_phase_workflow.md 仍含 RFC-005 引用，需清理")
    except Exception as e:
        errors.append(f"读取 learning_phase_workflow.md 失败: {e}")

    result = {"pass": len(errors) == 0, "errors": errors}
    print(json.dumps(result, ensure_ascii=False))
    sys.exit(0 if result["pass"] else 1)

if __name__ == "__main__":
    main()
