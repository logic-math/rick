#!/usr/bin/env python3
import json
import sys
import os
import subprocess
import glob
import re


def run_cmd(cmd, cwd=None):
    """Run a shell command and return (returncode, stdout, stderr)."""
    result = subprocess.run(
        cmd, shell=True, capture_output=True, text=True, cwd=cwd
    )
    return result.returncode, result.stdout, result.stderr


def main():
    errors = []

    # Project root: this file is at .rick/jobs/job_12/doing/tests/task4.py
    # So project root is 5 levels up
    project_root = os.path.abspath(
        os.path.join(os.path.dirname(os.path.abspath(__file__)), '../../../../..')
    )
    print(f"[DEBUG] project_root: {project_root}", file=sys.stderr)

    rick_bin = os.path.join(project_root, 'bin', 'rick')
    skills_dir = os.path.join(project_root, '.rick', 'skills')
    index_md = os.path.join(skills_dir, 'index.md')
    learning_md = os.path.join(project_root, 'internal', 'prompt', 'templates', 'learning.md')
    tools_dir = os.path.join(project_root, 'tools')

    # Cache dry-run output to avoid running multiple times
    _dry_run_combined = None

    def get_dry_run_output():
        nonlocal _dry_run_combined
        if _dry_run_combined is None:
            if os.path.exists(rick_bin):
                rc, stdout, stderr = run_cmd(f'{rick_bin} doing job_12 --dry-run', cwd=project_root)
                _dry_run_combined = stdout + stderr
                print(f"[DEBUG] dry-run rc={rc}, output[:400]={_dry_run_combined[:400]}", file=sys.stderr)
            else:
                _dry_run_combined = ''
        return _dry_run_combined

    # Run rfc002_e2e_test.sh - exit 0 and no FAIL lines
    try:
        e2e_script = os.path.join(project_root, '.rick', 'jobs', 'job_12', 'doing', 'tests', 'rfc002_e2e_test.sh')
        if not os.path.exists(e2e_script):
            errors.append(f'rfc002_e2e_test.sh 不存在: {e2e_script}')
        else:
            rc, stdout, stderr = run_cmd(f'bash {e2e_script}', cwd=project_root)
            combined = stdout + stderr
            print(f"[DEBUG] rfc002_e2e_test.sh rc={rc}", file=sys.stderr)
            if rc != 0:
                errors.append(f'rfc002_e2e_test.sh 退出码 {rc}，stderr: {stderr[:300]}')
            fail_lines = [l for l in combined.split('\n') if 'FAIL' in l and not re.match(r'.*FAIL.*0.*', l)]
            if fail_lines:
                errors.append(f'rfc002_e2e_test.sh 含 FAIL 行: {fail_lines[:3]}')
    except Exception as e:
        errors.append(f'rfc002_e2e_test.sh 异常: {str(e)}')

    # Run tools_integration_test.sh - exit 0 and no FAIL lines
    try:
        integration_script = os.path.join(project_root, 'tests', 'tools_integration_test.sh')
        if not os.path.exists(integration_script):
            errors.append(f'tools_integration_test.sh 不存在: {integration_script}')
        else:
            rc, stdout, stderr = run_cmd(f'bash {integration_script}', cwd=project_root)
            combined = stdout + stderr
            print(f"[DEBUG] tools_integration_test.sh rc={rc}", file=sys.stderr)
            if rc != 0:
                errors.append(f'tools_integration_test.sh 退出码 {rc}，stderr: {stderr[:300]}')
            fail_lines = [l for l in combined.split('\n') if 'FAIL' in l and not re.match(r'.*FAIL.*0.*', l)]
            if fail_lines:
                errors.append(f'tools_integration_test.sh 含 FAIL 行: {fail_lines[:3]}')
    except Exception as e:
        errors.append(f'tools_integration_test.sh 异常: {str(e)}')

    # 断言1: ls tools/*.py | wc -l == 5
    try:
        py_files = glob.glob(os.path.join(tools_dir, '*.py'))
        count = len(py_files)
        print(f"[DEBUG] tools/*.py count: {count}, files: {[os.path.basename(f) for f in py_files]}", file=sys.stderr)
        if count != 5:
            errors.append(f'断言1失败: tools/*.py 文件数量为 {count}，期望 5，文件: {[os.path.basename(f) for f in py_files]}')
    except Exception as e:
        errors.append(f'断言1异常: {str(e)}')

    # 断言2: python3 tools/build_and_get_rick_bin.py 返回 JSON 且包含 rick_bin 字段
    try:
        script = os.path.join(tools_dir, 'build_and_get_rick_bin.py')
        if not os.path.exists(script):
            errors.append('断言2失败: tools/build_and_get_rick_bin.py 不存在')
        else:
            rc, stdout, stderr = run_cmd(f'python3 {script}', cwd=project_root)
            print(f"[DEBUG] build_and_get_rick_bin.py rc={rc}, stdout={stdout[:200]}", file=sys.stderr)
            if rc != 0:
                errors.append(f'断言2失败: build_and_get_rick_bin.py 退出码 {rc}, stderr: {stderr[:200]}')
            else:
                try:
                    data = json.loads(stdout.strip())
                    if 'bin_path' not in data and 'rick_bin' not in data:
                        errors.append(f'断言2失败: 输出 JSON 不含 bin_path/rick_bin 字段，got: {stdout[:200]}')
                except json.JSONDecodeError as je:
                    errors.append(f'断言2失败: 输出不是有效 JSON: {stdout[:200]}, error: {je}')
    except Exception as e:
        errors.append(f'断言2异常: {str(e)}')

    # 断言3: ls .rick/skills/*.py 为空（无 .py 文件）
    try:
        py_skill_files = glob.glob(os.path.join(skills_dir, '*.py'))
        print(f"[DEBUG] .rick/skills/*.py: {py_skill_files}", file=sys.stderr)
        if py_skill_files:
            errors.append(f'断言3失败: .rick/skills/ 中存在 .py 文件: {[os.path.basename(f) for f in py_skill_files]}')
    except Exception as e:
        errors.append(f'断言3异常: {str(e)}')

    # 断言4: .rick/skills/index.md 包含三列表格头
    try:
        if not os.path.exists(index_md):
            errors.append('断言4失败: .rick/skills/index.md 不存在')
        else:
            with open(index_md, 'r') as f:
                index_content = f.read()
            if '| Skill | 描述 | 触发场景 |' not in index_content:
                errors.append('断言4失败: index.md 不含三列表格头 "| Skill | 描述 | 触发场景 |"')
    except Exception as e:
        errors.append(f'断言4异常: {str(e)}')

    # 断言5: index.md 不含空触发场景列
    try:
        if os.path.exists(index_md):
            with open(index_md, 'r') as f:
                lines = f.readlines()
            empty_trigger_lines = []
            for i, line in enumerate(lines, 1):
                stripped = line.strip()
                if not stripped.startswith('|') or not stripped.endswith('|'):
                    continue
                parts = [c.strip() for c in stripped.split('|')]
                # parts[0] and parts[-1] are empty (outside the outer pipes)
                inner = parts[1:-1]
                if len(inner) < 3:
                    continue
                # Skip separator rows like |---|---|---|
                if all(re.match(r'^[-:]+$', c) for c in inner if c):
                    continue
                # Check if last column (trigger) is empty
                if inner[-1] == '':
                    empty_trigger_lines.append(f'行{i}: {line.rstrip()}')
            if empty_trigger_lines:
                errors.append(f'断言5失败: index.md 含空触发场景列，共 {len(empty_trigger_lines)} 行: {empty_trigger_lines[:3]}')
    except Exception as e:
        errors.append(f'断言5异常: {str(e)}')

    # 断言6: bin/rick doing job_12 --dry-run 输出包含 tools/ 字样
    try:
        if not os.path.exists(rick_bin):
            errors.append('断言6失败: bin/rick 不存在')
        else:
            combined = get_dry_run_output()
            if 'tools/' not in combined:
                errors.append('断言6失败: --dry-run 输出不含 "tools/" 字样（tools section 非空）')
    except Exception as e:
        errors.append(f'断言6异常: {str(e)}')

    # 断言7: bin/rick doing job_12 --dry-run 输出包含 .md skill 名称
    try:
        if os.path.exists(rick_bin):
            combined = get_dry_run_output()
            md_skills = glob.glob(os.path.join(skills_dir, '*.md'))
            md_skill_names = [
                os.path.splitext(os.path.basename(f))[0]
                for f in md_skills if os.path.basename(f) != 'index.md'
            ]
            print(f"[DEBUG] md_skill_names: {md_skill_names}", file=sys.stderr)
            found_skill = any(name in combined for name in md_skill_names)
            if not found_skill and md_skill_names:
                errors.append(f'断言7失败: --dry-run 输出不含任何 .md skill 名称，期望含: {md_skill_names[:3]}')
    except Exception as e:
        errors.append(f'断言7异常: {str(e)}')

    # 断言8: bin/rick doing job_12 --dry-run 输出 skills section 不含 .py 条目
    try:
        if os.path.exists(rick_bin):
            combined = get_dry_run_output()
            # Extract skills section
            skills_section_match = re.search(
                r'(?:## 可用的项目 Skills|## Skills)(.*?)(?=\n## |\Z)',
                combined, re.DOTALL
            )
            section_to_check = skills_section_match.group(1) if skills_section_match else combined
            py_in_skills = [
                line.strip() for line in section_to_check.split('\n')
                if '.py' in line and 'tools/' not in line
            ]
            if py_in_skills:
                errors.append(f'断言8失败: --dry-run skills section 含 .py 条目: {py_in_skills[:3]}')
    except Exception as e:
        errors.append(f'断言8异常: {str(e)}')

    # 断言9: learning.md 不含旧格式 skills/*.py
    try:
        if not os.path.exists(learning_md):
            errors.append('断言9失败: internal/prompt/templates/learning.md 不存在')
        else:
            with open(learning_md, 'r') as f:
                lm_content = f.read()
            if 'skills/*.py' in lm_content:
                errors.append('断言9失败: learning.md 含旧格式 "skills/*.py"')
    except Exception as e:
        errors.append(f'断言9异常: {str(e)}')

    # 断言10: learning.md 含 tools/*.py 且含 skills/*.md
    try:
        if os.path.exists(learning_md):
            with open(learning_md, 'r') as f:
                lm_content = f.read()
            if 'tools/*.py' not in lm_content:
                errors.append('断言10失败: learning.md 不含 "tools/*.py"')
            if 'skills/*.md' not in lm_content:
                errors.append('断言10失败: learning.md 不含 "skills/*.md"')
    except Exception as e:
        errors.append(f'断言10异常: {str(e)}')

    result = {
        'pass': len(errors) == 0,
        'errors': errors
    }

    print(json.dumps(result))
    sys.exit(0 if result['pass'] else 1)


if __name__ == '__main__':
    main()
