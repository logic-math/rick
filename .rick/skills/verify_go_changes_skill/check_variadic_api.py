# Description: 检查 Go 函数签名是否已改为 variadic（可变参数）形式，用于验证向后兼容性改造

import argparse
import json
import re
import sys
import os


def check_variadic(file_path: str, func_name: str) -> dict:
    """
    检查指定 Go 文件中的函数是否已改为 variadic 形式。
    例如: func NewPromptManager(templateDir ...string) 而非 func NewPromptManager(templateDir string)
    """
    if not os.path.exists(file_path):
        return {"pass": False, "result": None, "errors": [f"File not found: {file_path}"]}

    with open(file_path, "r") as f:
        content = f.read()

    # 匹配 func <name>(...) 形式，查找 variadic 参数（...type）
    # 同时也要找到函数签名行
    pattern = rf"func\s+{re.escape(func_name)}\s*\(([^)]*)\)"
    match = re.search(pattern, content)

    if not match:
        return {
            "pass": False,
            "result": None,
            "errors": [f"Function '{func_name}' not found in {file_path}"],
        }

    params = match.group(1).strip()
    is_variadic = "..." in params

    return {
        "pass": is_variadic,
        "result": {
            "function": func_name,
            "signature": f"func {func_name}({params})",
            "is_variadic": is_variadic,
        },
        "errors": [] if is_variadic else [
            f"Function '{func_name}' is NOT variadic. Current params: ({params})"
        ],
    }


def run_tests():
    """内置测试：创建临时 Go 文件验证检测逻辑"""
    import tempfile

    # 测试1: variadic 函数应通过
    variadic_code = """package prompt

func NewPromptManager(templateDir ...string) *PromptManager {
    return &PromptManager{}
}
"""
    # 测试2: 非 variadic 函数应失败
    non_variadic_code = """package prompt

func NewPromptManager(templateDir string) *PromptManager {
    return &PromptManager{}
}
"""

    errors = []

    with tempfile.NamedTemporaryFile(mode="w", suffix=".go", delete=False) as f:
        f.write(variadic_code)
        tmp_variadic = f.name

    with tempfile.NamedTemporaryFile(mode="w", suffix=".go", delete=False) as f:
        f.write(non_variadic_code)
        tmp_non_variadic = f.name

    try:
        result1 = check_variadic(tmp_variadic, "NewPromptManager")
        if not result1["pass"]:
            errors.append(f"Test1 FAILED: variadic function should pass. Got: {result1}")

        result2 = check_variadic(tmp_non_variadic, "NewPromptManager")
        if result2["pass"]:
            errors.append(f"Test2 FAILED: non-variadic function should fail. Got: {result2}")

        result3 = check_variadic(tmp_variadic, "NonExistentFunc")
        if result3["pass"]:
            errors.append(f"Test3 FAILED: non-existent function should fail. Got: {result3}")
    finally:
        os.unlink(tmp_variadic)
        os.unlink(tmp_non_variadic)

    if errors:
        print(json.dumps({"pass": False, "errors": errors}))
        sys.exit(1)
    else:
        print(json.dumps({"pass": True, "result": "All 3 tests passed", "errors": []}))
        sys.exit(0)


def main():
    parser = argparse.ArgumentParser(
        description="Check if a Go function has been converted to variadic form"
    )
    parser.add_argument("--test", action="store_true", help="Run built-in tests")
    parser.add_argument("--file", help="Path to the Go source file")
    parser.add_argument("--func", dest="func_name", help="Function name to check")
    args = parser.parse_args()

    if args.test:
        run_tests()
        return

    if not args.file or not args.func_name:
        print(json.dumps({"pass": False, "errors": ["--file and --func are required"]}))
        sys.exit(1)

    result = check_variadic(args.file, args.func_name)
    print(json.dumps(result))
    sys.exit(0 if result["pass"] else 1)


if __name__ == "__main__":
    main()
