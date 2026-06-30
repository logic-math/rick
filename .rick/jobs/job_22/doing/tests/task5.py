#!/usr/bin/env python3
import json
import sys
import os
import subprocess
import re

def get_project_root():
    # 6 dirnames from this file to project root
    path = os.path.abspath(__file__)
    for _ in range(6):
        path = os.path.dirname(path)
    return path

def build_rick(project_root):
    build_script = os.path.join(project_root, '.rick', 'tools', 'build_and_get_rick_bin.py')
    result = subprocess.run(
        ['python3', build_script],
        capture_output=True, text=True, cwd=project_root
    )
    if result.returncode != 0:
        return None, f'build_and_get_rick_bin.py failed: {result.stderr.strip()}'
    try:
        data = json.loads(result.stdout.strip())
        return data.get('bin_path'), None
    except Exception as e:
        return None, f'Failed to parse build script output: {str(e)}'

def main():
    errors = []
    project_root = get_project_root()
    print(f'project_root: {project_root}', file=sys.stderr)

    # Build rick binary
    bin_path, build_err = build_rick(project_root)
    if build_err:
        errors.append(build_err)
        print(json.dumps({'pass': False, 'errors': errors}, ensure_ascii=False))
        sys.exit(1)

    # Test 1: dry-run 验证变量替换
    try:
        result = subprocess.run(
            [bin_path, 'plan', '--dry-run'],
            capture_output=True, text=True, cwd=project_root
        )
        output = result.stdout + result.stderr

        if '可用的项目 Loops' not in output:
            errors.append('plan --dry-run 输出不包含 "可用的项目 Loops"')

        if '{{okr_path}}' in output:
            errors.append('plan --dry-run 输出仍包含字面量 "{{okr_path}}"，变量未被替换或已被移除')

        if '{{spec_path}}' in output:
            errors.append('plan --dry-run 输出仍包含字面量 "{{spec_path}}"，变量未被替换或已被移除')

        if re.search(r'必须生成.*OKR\.md', output):
            errors.append('plan --dry-run 输出仍包含 "必须生成.*OKR.md" 指令，模板未更新')

    except Exception as e:
        errors.append(f'运行 plan --dry-run 失败: {str(e)}')

    # Test 2: 单元测试 - plan_prompt 相关测试通过
    try:
        result = subprocess.run(
            ['go', 'test', './internal/prompt/...', '-run', 'TestPlanPrompt', '-v'],
            capture_output=True, text=True, cwd=project_root
        )
        if result.returncode != 0:
            errors.append(f'go test TestPlanPrompt 失败:\n{result.stdout}\n{result.stderr}')
    except Exception as e:
        errors.append(f'运行 go test 失败: {str(e)}')

    # Test 3: plan_check 通过
    try:
        result = subprocess.run(
            [bin_path, 'tools', 'plan_check', 'job_22'],
            capture_output=True, text=True, cwd=project_root
        )
        combined = result.stdout + result.stderr
        if '✅ plan check passed' not in combined:
            errors.append(f'rick tools plan_check job_22 未输出 "✅ plan check passed"，实际输出：{combined[:300]}')
    except Exception as e:
        errors.append(f'运行 plan_check 失败: {str(e)}')

    # Test 4: go build 编译无错误
    try:
        result = subprocess.run(
            ['go', 'build', './...'],
            capture_output=True, text=True, cwd=project_root
        )
        if result.returncode != 0:
            errors.append(f'go build ./... 失败:\n{result.stderr.strip()}')
    except Exception as e:
        errors.append(f'运行 go build 失败: {str(e)}')

    # Test 5: plan.md 模板不包含 OKR.md 生成指令
    try:
        plan_md = os.path.join(project_root, 'internal', 'prompt', 'templates', 'plan.md')
        with open(plan_md, 'r', encoding='utf-8') as f:
            content = f.read()
        count = content.count('OKR.md')
        if count != 0:
            errors.append(f'internal/prompt/templates/plan.md 仍包含 {count} 处 "OKR.md"，应为 0')
    except Exception as e:
        errors.append(f'读取 plan.md 失败: {str(e)}')

    result = {
        'pass': len(errors) == 0,
        'errors': errors
    }

    print(json.dumps(result, ensure_ascii=False))
    sys.exit(0 if result['pass'] else 1)

if __name__ == '__main__':
    main()
