#!/usr/bin/env python3
# I will use skill:tdd and skill:testing-anti-patterns for test generation.
import json
import sys
import os
import ast


def main():
    errors = []

    # 前置条件：rick doing 已调用 pi 执行本任务
    # 输入参数：目标文件（端到端验证的产物，由 pi 通过 rick doing 调用创建）
    # 预期输出：目标文件存在、非空、且为符合格式要求的 Python 测试脚本
    target_file = '/workdir/sunquan20/AI_CODING/rick/.rick/jobs/job_31/doing/tests/task1.py'

    # Test 1: 目标文件存在（验收标准：检查目标文件存在）
    if not os.path.exists(target_file):
        errors.append(f'目标文件不存在: {target_file}')
    elif not os.path.isfile(target_file):
        errors.append(f'目标路径不是普通文件: {target_file}')
    else:
        # Test 2: 目标文件可读且非空
        try:
            with open(target_file, 'r', encoding='utf-8') as f:
                content = f.read()
            if len(content.strip()) == 0:
                errors.append(f'目标文件为空: {target_file}')
        except Exception as e:
            errors.append(f'读取目标文件失败: {str(e)}')
            content = ''

        # Test 3: 目标文件是合法 Python 脚本（语法可解析，验证 pi 产物可执行）
        try:
            ast.parse(content)
        except SyntaxError as e:
            errors.append(f'目标文件存在 Python 语法错误: {e}')

        # Test 4: 目标文件包含测试脚本必需的 JSON 输出结构
        if 'json.dumps' not in content:
            errors.append('目标文件缺少 json.dumps 输出（测试脚本必须输出一行 JSON 结果）')
        if 'sys.exit' not in content:
            errors.append('目标文件缺少 sys.exit 退出码逻辑（pass=true 退出码 0，pass=false 退出码 1）')

    # 构建结果 JSON
    result = {
        'pass': len(errors) == 0,
        'errors': errors
    }

    # 输出 JSON (CRITICAL: 只有这一行输出到 stdout)
    print(json.dumps(result))

    # 使用合适的退出码
    sys.exit(0 if result['pass'] else 1)


if __name__ == '__main__':
    main()
