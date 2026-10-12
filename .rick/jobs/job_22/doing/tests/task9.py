#!/usr/bin/env python3
import json
import sys
import os
import subprocess

def get_project_root():
    # 6 levels up from this file: tests/ → doing/ → job_22/ → jobs/ → .rick/ → project root
    path = os.path.abspath(__file__)
    for _ in range(6):
        path = os.path.dirname(path)
    return path

def main():
    errors = []
    project_root = get_project_root()
    print(f"project_root: {project_root}", file=sys.stderr)

    # Test 1: loop_protocol.md 文件存在
    loop_protocol_file = os.path.join(project_root, "internal", "prompt", "templates", "skills", "loop_protocol.md")
    if not os.path.exists(loop_protocol_file):
        errors.append(f"loop_protocol.md 不存在: {loop_protocol_file}")
    else:
        # Test 1b: 文件包含协议正文关键字
        try:
            with open(loop_protocol_file, "r", encoding="utf-8") as f:
                content = f.read()
            if "Step 1" not in content and "loop-protocol" not in content:
                errors.append("loop_protocol.md 缺少预期内容（name: loop-protocol 或 Step 1）")
        except Exception as e:
            errors.append(f"读取 loop_protocol.md 失败: {e}")

    # Test 2: doing.md 正向验证 — 包含 {{loop_protocol_path}}
    doing_md = os.path.join(project_root, "internal", "prompt", "templates", "doing.md")
    try:
        with open(doing_md, "r", encoding="utf-8") as f:
            doing_content = f.read()
        count = doing_content.count("{{loop_protocol_path}}")
        if count < 1:
            errors.append(f"doing.md 未包含 {{{{loop_protocol_path}}}}（count={count}）")
    except Exception as e:
        errors.append(f"读取 doing.md 失败: {e}")
        doing_content = ""

    # Test 3: doing.md 负向验证 — 不含协议正文
    if doing_content:
        for keyword in ["Step 1：加载 Loop", "Step 2：执行一次迭代"]:
            if keyword in doing_content:
                errors.append(f"doing.md 内联了协议正文（含 '{keyword}'），应只引用路径")

    # Test 4: doing.md 回归验证 — {{loops_context}} 未被删除
    if doing_content:
        if "{{loops_context}}" not in doing_content:
            errors.append("doing.md 缺少 {{loops_context}}（task4 产出被覆盖）")

    # Test 5: easy.md 正向验证 — 包含 {{loop_protocol_path}}
    easy_md = os.path.join(project_root, "internal", "prompt", "templates", "easy.md")
    try:
        with open(easy_md, "r", encoding="utf-8") as f:
            easy_content = f.read()
        count = easy_content.count("{{loop_protocol_path}}")
        if count < 1:
            errors.append(f"easy.md 未包含 {{{{loop_protocol_path}}}}（count={count}）")
    except Exception as e:
        errors.append(f"读取 easy.md 失败: {e}")
        easy_content = ""

    # Test 6: easy.md 负向验证 — 不含协议正文
    if easy_content:
        for keyword in ["Step 1：加载 Loop", "Step 2：执行一次迭代"]:
            if keyword in easy_content:
                errors.append(f"easy.md 内联了协议正文（含 '{keyword}'），应只引用路径")

    # Test 7: easy.md 回归验证 — {{loops_context}} 未被删除
    if easy_content:
        if "{{loops_context}}" not in easy_content:
            errors.append("easy.md 缺少 {{loops_context}}（task7 产出被覆盖）")

    # Test 8: 单一变更点验证 — "Step 1：加载 Loop" 只在 loop_protocol.md 中存在
    try:
        result = subprocess.run(
            ["grep", "-r", "Step 1：加载 Loop", os.path.join(project_root, "internal", "prompt", "templates")],
            capture_output=True, text=True
        )
        if result.returncode == 0 and result.stdout.strip():
            lines = [l for l in result.stdout.strip().splitlines() if l]
            non_protocol = [l for l in lines if "loop_protocol.md" not in l]
            if non_protocol:
                errors.append(f"协议正文出现在 loop_protocol.md 以外的文件: {non_protocol}")
            # loop_protocol.md 本身应该包含它
            if not any("loop_protocol.md" in l for l in lines):
                errors.append("loop_protocol.md 中未找到 'Step 1：加载 Loop'（协议正文缺失）")
        elif result.returncode != 0:
            # grep 找不到说明 loop_protocol.md 本身也没有该内容
            errors.append("'Step 1：加载 Loop' 不存在于任何模板文件（loop_protocol.md 正文缺失）")
    except Exception as e:
        errors.append(f"单一变更点验证失败: {e}")

    # Test 9: embed.FS 可加载验证 — go test TestEmbedded 包含 loop_protocol
    try:
        result = subprocess.run(
            ["go", "test", "./internal/prompt/...", "-run", "TestEmbedded", "-v"],
            capture_output=True, text=True, cwd=project_root, timeout=60
        )
        combined = result.stdout + result.stderr
        if "loop_protocol" not in combined.lower():
            errors.append("go test TestEmbedded 输出未提及 loop_protocol（embed.FS 未包含该文件）")
        if "FAIL" in combined and "PASS" not in combined:
            errors.append(f"go test TestEmbedded 失败: {combined[:300]}")
    except subprocess.TimeoutExpired:
        errors.append("go test TestEmbedded 超时")
    except Exception as e:
        errors.append(f"go test TestEmbedded 执行失败: {e}")

    # Test 10: doing dry-run 路径注入验证 — 真实路径而非字面量占位符
    try:
        build_tool = os.path.join(project_root, ".rick", "tools", "build_and_get_rick_bin.py")
        build_result = subprocess.run(
            ["python3", build_tool],
            capture_output=True, text=True, cwd=project_root, timeout=120
        )
        bin_info = json.loads(build_result.stdout.strip())
        rick_bin = bin_info.get("bin_path", "")
        if not rick_bin or not os.path.exists(rick_bin):
            errors.append(f"build_and_get_rick_bin.py 未返回有效 bin_path: {build_result.stdout[:200]}")
        else:
            dry_result = subprocess.run(
                [rick_bin, "doing", "--job", "job_22", "--dry-run"],
                capture_output=True, text=True, cwd=project_root, timeout=30
            )
            combined = dry_result.stdout + dry_result.stderr
            # 检查是否包含真实路径（含斜杠的 loop_protocol.md 路径）
            import re
            real_path_pattern = re.compile(r'/[^\s]*loop_protocol\.md')
            if not real_path_pattern.search(combined):
                errors.append("rick doing --dry-run 输出未包含 loop_protocol 真实路径（如 /path/to/loop_protocol.md）")
    except subprocess.TimeoutExpired:
        errors.append("doing dry-run 构建或执行超时")
    except Exception as e:
        errors.append(f"doing dry-run 验证失败: {e}")

    # Test 11: easy prompt 单元测试 — TestGenerateEasyPromptFile_LoopProtocolInjected
    try:
        result = subprocess.run(
            ["go", "test", "./internal/prompt/...",
             "-run", "TestGenerateEasyPromptFile_LoopProtocolInjected", "-v"],
            capture_output=True, text=True, cwd=project_root, timeout=60
        )
        combined = result.stdout + result.stderr
        # 必须确认具名测试函数实际运行并通过（不能只看顶层 PASS）
        if "--- PASS: TestGenerateEasyPromptFile_LoopProtocolInjected" not in combined:
            errors.append(
                "TestGenerateEasyPromptFile_LoopProtocolInjected 未运行或未通过（测试函数可能尚未定义）"
                + (f": {combined[:200]}" if combined.strip() else "")
            )
        if "FAIL" in combined:
            errors.append(f"TestGenerateEasyPromptFile_LoopProtocolInjected 失败: {combined[:300]}")
    except subprocess.TimeoutExpired:
        errors.append("go test LoopProtocolInjected 超时")
    except Exception as e:
        errors.append(f"go test LoopProtocolInjected 执行失败: {e}")

    result = {
        "pass": len(errors) == 0,
        "errors": errors
    }

    print(json.dumps(result, ensure_ascii=False))
    sys.exit(0 if result["pass"] else 1)

if __name__ == "__main__":
    main()
