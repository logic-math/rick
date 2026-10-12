#!/usr/bin/env python3
import json
import sys
import os
import re

def get_project_root():
    # 6 dirnames: tests/ -> doing/ -> job_22/ -> jobs/ -> .rick/ -> project root
    path = os.path.abspath(__file__)
    for _ in range(6):
        path = os.path.dirname(path)
    return path

def main():
    errors = []
    project_root = get_project_root()

    loops_readme = os.path.join(project_root, '.rick', 'loops', 'README.md')
    example_loop = os.path.join(project_root, '.rick', 'loops', 'example_loop.md')
    skills_readme = os.path.join(project_root, '.rick', 'skills', 'README.md')

    # Test 1: 文件存在性验证
    for f in [loops_readme, example_loop, skills_readme]:
        if not os.path.exists(f):
            errors.append(f'File does not exist: {f}')

    # Test 2: loop frontmatter 校验（name + trigger 字段）
    if os.path.exists(example_loop):
        try:
            content = open(example_loop, 'r', encoding='utf-8').read()
            fm_match = re.search(r'^---\n(.*?)\n---', content, re.DOTALL)
            if not fm_match:
                errors.append('example_loop.md: no frontmatter found')
            else:
                fields = {line.split(':')[0].strip() for line in fm_match.group(1).splitlines() if ':' in line}
                missing = {'name', 'trigger'} - fields
                if missing:
                    errors.append(f'example_loop.md frontmatter missing fields: {missing}')
        except Exception as e:
            errors.append(f'Failed to read example_loop.md: {e}')

    # Test 3: loop 五要素章节验证
    if os.path.exists(example_loop):
        try:
            content = open(example_loop, 'r', encoding='utf-8').read()
            required_sections = ['## 目标', '## 上下文管理', '## 可调用工具', '## 产出评估', '## 停止标准']
            missing_sections = [s for s in required_sections if s not in content]
            if missing_sections:
                errors.append(f'example_loop.md missing sections: {missing_sections}')
        except Exception as e:
            errors.append(f'Failed to check example_loop.md sections: {e}')

    # Test 4: skill README 四要素验证
    if os.path.exists(skills_readme):
        try:
            content = open(skills_readme, 'r', encoding='utf-8').read()
            required_elements = ['When to Use', 'Procedure', 'Pitfalls', 'Verification']
            missing_elements = [e for e in required_elements if e not in content]
            if missing_elements:
                errors.append(f'skills/README.md missing elements: {missing_elements}')
        except Exception as e:
            errors.append(f'Failed to check skills/README.md: {e}')

    result = {'pass': len(errors) == 0, 'errors': errors}
    print(json.dumps(result, ensure_ascii=False))
    sys.exit(0 if result['pass'] else 1)

if __name__ == '__main__':
    main()
