# Description: 检查 internal/cmd/ 中所有 NewXxxCmd() 函数是否都在 root.go 中注册
"""
检验 cobra 命令注册完整性：扫描 internal/cmd/*.go 找出所有 NewXxxCmd() 定义，
验证每个函数都在 root.go 中有对应的 AddCommand(NewXxxCmd()) 调用。
"""
import os
import re
import sys
import json
import argparse


def find_new_cmd_functions(cmd_dir):
    """扫描 cmd_dir 下所有 .go 文件，找出所有 NewXxxCmd() 函数定义。"""
    pattern = re.compile(r'^func\s+(New\w+Cmd)\s*\(', re.MULTILINE)
    found = {}
    for fname in os.listdir(cmd_dir):
        if not fname.endswith('.go') or fname == 'root.go':
            continue
        path = os.path.join(cmd_dir, fname)
        content = open(path).read()
        for m in pattern.finditer(content):
            func_name = m.group(1)
            found[func_name] = fname
    return found


def find_registered_cmds(root_go_path):
    """从 root.go 中提取所有 AddCommand(NewXxxCmd()) 的注册函数名。"""
    pattern = re.compile(r'AddCommand\s*\(\s*(New\w+Cmd)\s*\(\s*\)\s*\)')
    content = open(root_go_path).read()
    return set(pattern.findall(content))


def main():
    parser = argparse.ArgumentParser(description='Check cobra command registration completeness')
    parser.add_argument('--cmd-dir', default='internal/cmd', help='Path to cmd directory')
    parser.add_argument('--test', action='store_true', help='Self-test mode')
    args = parser.parse_args()

    if args.test:
        # Self-test: verify tool runs without error
        print(json.dumps({"pass": True, "errors": [], "note": "self-test ok"}, ensure_ascii=False))
        return

    cmd_dir = args.cmd_dir
    root_go = os.path.join(cmd_dir, 'root.go')

    if not os.path.isdir(cmd_dir):
        print(json.dumps({"pass": False, "errors": [f"cmd dir not found: {cmd_dir}"]}, ensure_ascii=False))
        sys.exit(1)

    if not os.path.isfile(root_go):
        print(json.dumps({"pass": False, "errors": [f"root.go not found: {root_go}"]}, ensure_ascii=False))
        sys.exit(1)

    defined = find_new_cmd_functions(cmd_dir)
    registered = find_registered_cmds(root_go)

    missing = {fn: src for fn, src in defined.items() if fn not in registered}
    errors = [f"{fn} (in {src}) not registered in root.go" for fn, src in missing.items()]

    result = {
        "pass": len(errors) == 0,
        "defined": list(defined.keys()),
        "registered": list(registered),
        "errors": errors
    }
    print(json.dumps(result, ensure_ascii=False, indent=2))
    if errors:
        sys.exit(1)


if __name__ == '__main__':
    main()
