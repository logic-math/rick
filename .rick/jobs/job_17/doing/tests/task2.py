#!/usr/bin/env python3
# Description: task2 验收测试 - 验证 internal/parser/frontmatter.go 提取公共 frontmatter 解析函数
import json
import sys
import os
import subprocess

def get_project_root():
    # tests/ -> doing/ -> job_17/ -> jobs/ -> .rick/ -> rick/
    return os.path.dirname(os.path.dirname(os.path.dirname(os.path.dirname(
        os.path.dirname(os.path.dirname(os.path.abspath(__file__)))))))

def main():
    errors = []
    project_root = get_project_root()
    print(f"[DEBUG] project_root: {project_root}", file=sys.stderr)

    # Test 1: internal/parser/frontmatter.go 存在
    frontmatter_go = os.path.join(project_root, "internal", "parser", "frontmatter.go")
    if not os.path.exists(frontmatter_go):
        errors.append(f"internal/parser/frontmatter.go does not exist: {frontmatter_go}")
    else:
        # Test 2: ExtractBugFrontmatter 函数已导出（大写）
        try:
            with open(frontmatter_go, "r", encoding="utf-8") as f:
                content = f.read()
            if "func ExtractBugFrontmatter" not in content:
                errors.append("internal/parser/frontmatter.go missing exported func ExtractBugFrontmatter")
        except Exception as e:
            errors.append(f"Failed to read frontmatter.go: {e}")

    # Test 3: internal/parser/frontmatter_test.go 存在
    frontmatter_test_go = os.path.join(project_root, "internal", "parser", "frontmatter_test.go")
    if not os.path.exists(frontmatter_test_go):
        errors.append(f"internal/parser/frontmatter_test.go does not exist: {frontmatter_test_go}")

    # Test 4: easy_prompt.go 内联解析已替换（不再含 strings.HasPrefix.*summary/status）
    easy_prompt_go = os.path.join(project_root, "internal", "prompt", "easy_prompt.go")
    if not os.path.exists(easy_prompt_go):
        errors.append(f"internal/prompt/easy_prompt.go not found: {easy_prompt_go}")
    else:
        try:
            with open(easy_prompt_go, "r", encoding="utf-8") as f:
                ep_content = f.read()
            # 验证内联 frontmatter 解析已移除
            if 'strings.HasPrefix(t, "summary:")' in ep_content or \
               'strings.HasPrefix(t, "status:")' in ep_content:
                errors.append('easy_prompt.go still contains inline frontmatter parsing (HasPrefix summary/status)')
            # 验证已 import parser 包
            if 'github.com/sunquan/rick/internal/parser' not in ep_content:
                errors.append('easy_prompt.go missing import of internal/parser package')
        except Exception as e:
            errors.append(f"Failed to read easy_prompt.go: {e}")

    # Test 5: go test ./internal/executor/... ./internal/parser/...
    try:
        build_tool = os.path.join(project_root, ".rick", "tools", "build_and_get_rick_bin.py")
        if os.path.exists(build_tool):
            result = subprocess.run(
                ["python3", build_tool],
                capture_output=True, text=True, cwd=project_root
            )
            if result.returncode != 0:
                errors.append(f"build_and_get_rick_bin.py failed: {result.stderr.strip()[:300]}")
        else:
            print(f"[DEBUG] build tool not found, skipping build step", file=sys.stderr)
    except Exception as e:
        errors.append(f"Build step failed: {e}")

    try:
        result = subprocess.run(
            ["go", "test", "./internal/executor/...", "./internal/parser/..."],
            capture_output=True, text=True, cwd=project_root
        )
        print(f"[DEBUG] go test stdout: {result.stdout[:500]}", file=sys.stderr)
        print(f"[DEBUG] go test stderr: {result.stderr[:500]}", file=sys.stderr)
        if result.returncode != 0:
            errors.append(f"go test ./internal/executor/... ./internal/parser/... failed:\n{result.stdout[-400:]}\n{result.stderr[-400:]}")
    except Exception as e:
        errors.append(f"go test execution failed: {e}")

    # Test 6: go build ./... 无循环导入
    try:
        result = subprocess.run(
            ["go", "build", "./..."],
            capture_output=True, text=True, cwd=project_root
        )
        if result.returncode != 0:
            errors.append(f"go build ./... failed (possible import cycle):\n{result.stderr[-400:]}")
    except Exception as e:
        errors.append(f"go build ./... execution failed: {e}")

    result_json = {
        "pass": len(errors) == 0,
        "errors": errors
    }
    print(json.dumps(result_json, ensure_ascii=False))
    sys.exit(0 if result_json["pass"] else 1)

if __name__ == "__main__":
    main()
