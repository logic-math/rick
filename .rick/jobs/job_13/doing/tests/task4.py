#!/usr/bin/env python3
import json
import sys
import os
import subprocess

def main():
    errors = []
    project_root = os.path.abspath(os.path.join(os.path.dirname(__file__), '..', '..', '..', '..', '..'))

    # Test 1: skills 目录已删除
    skills_dir = os.path.join(project_root, 'skills')
    if os.path.isdir(skills_dir):
        errors.append(f'skills/ directory still exists at {skills_dir}')

    # Test 2: install.sh 不含 skills 逻辑
    install_sh = os.path.join(project_root, 'scripts', 'install.sh')
    if not os.path.exists(install_sh):
        errors.append(f'scripts/install.sh not found at {install_sh}')
    else:
        try:
            with open(install_sh, 'r') as f:
                content = f.read()
            for pattern in ['install_skills', 'verify_skills', 'claude/skills']:
                if pattern in content:
                    errors.append(f'scripts/install.sh still contains: {pattern}')
        except Exception as e:
            errors.append(f'Failed to read install.sh: {str(e)}')

    # Test 3: install.sh 语法检查
    if os.path.exists(install_sh):
        try:
            result = subprocess.run(['bash', '-n', install_sh], capture_output=True, text=True)
            if result.returncode != 0:
                errors.append(f'install.sh syntax error: {result.stderr.strip()}')
        except Exception as e:
            errors.append(f'Failed to syntax-check install.sh: {str(e)}')

    # Test 4: uninstall.sh 语法检查
    uninstall_sh = os.path.join(project_root, 'scripts', 'uninstall.sh')
    if not os.path.exists(uninstall_sh):
        errors.append(f'scripts/uninstall.sh not found at {uninstall_sh}')
    else:
        try:
            result = subprocess.run(['bash', '-n', uninstall_sh], capture_output=True, text=True)
            if result.returncode != 0:
                errors.append(f'uninstall.sh syntax error: {result.stderr.strip()}')
        except Exception as e:
            errors.append(f'Failed to syntax-check uninstall.sh: {str(e)}')

    result = {
        'pass': len(errors) == 0,
        'errors': errors
    }

    print(json.dumps(result))
    sys.exit(0 if result['pass'] else 1)

if __name__ == '__main__':
    main()
