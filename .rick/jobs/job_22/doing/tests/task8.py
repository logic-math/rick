#!/usr/bin/env python3
import json
import sys
import os
import subprocess
import shutil

def main():
    errors = []

    # Resolve project root: tests/ -> doing/ -> job_22/ -> jobs/ -> .rick/ -> root
    script_dir = os.path.dirname(os.path.abspath(__file__))
    project_root = os.path.abspath(os.path.join(script_dir, '..', '..', '..', '..', '..'))
    rick_dir = os.path.join(project_root, '.rick')
    loops_dir = os.path.join(rick_dir, 'loops')
    skills_dir = os.path.join(rick_dir, 'skills')

    print(f'project_root={project_root}', file=sys.stderr)

    # Build binary
    build_tool = os.path.join(rick_dir, 'tools', 'build_and_get_rick_bin.py')
    try:
        r = subprocess.run(['python3', build_tool], capture_output=True, text=True, cwd=project_root)
        build_data = json.loads(r.stdout.strip())
        if not build_data.get('pass'):
            errors.append(f'Build failed: {build_data.get("errors")}')
            print(json.dumps({'pass': False, 'errors': errors}, ensure_ascii=False))
            sys.exit(1)
        rick_bin = build_data['bin_path']
    except Exception as e:
        errors.append(f'Failed to build rick binary: {e}')
        print(json.dumps({'pass': False, 'errors': errors}, ensure_ascii=False))
        sys.exit(1)

    def run_rick(*args):
        return subprocess.run([rick_bin] + list(args), capture_output=True, text=True, cwd=project_root)

    # Ensure learning/SUMMARY.md exists (prerequisite for learning_check)
    learning_dir = os.path.join(rick_dir, 'jobs', 'job_22', 'learning')
    os.makedirs(learning_dir, exist_ok=True)
    summary_path = os.path.join(learning_dir, 'SUMMARY.md')
    if not os.path.exists(summary_path):
        with open(summary_path, 'w') as f:
            f.write('# Job job_22 SUMMARY\n')

    # Track created test fixture files for cleanup
    created_files = []

    try:
        # --- Test 1: Unit tests for runLoopsAndSkillsCheck ---
        print('Test 1: unit tests TestLoopsSkillsCheck', file=sys.stderr)
        unit = subprocess.run(
            ['go', 'test', './internal/cmd/...', '-run', 'TestLoopsSkillsCheck', '-v'],
            capture_output=True, text=True, cwd=project_root
        )
        unit_out = unit.stdout + unit.stderr
        if unit.returncode != 0:
            errors.append(
                f'Unit tests (TestLoopsSkillsCheck) failed:\n{unit_out}'
            )
        elif 'no tests to run' in unit_out or '--- RUN' not in unit_out:
            errors.append(
                'Unit tests (TestLoopsSkillsCheck): no test cases found — TestLoopsSkillsCheck must be implemented'
            )

        # --- Test 2: Edge case - loops/skills dirs don't exist → learning_check passes ---
        print('Test 2: dirs not exist edge case', file=sys.stderr)
        loops_backup = loops_dir + '__backup_task8'
        skills_backup = skills_dir + '__backup_task8'
        moved_loops = False
        moved_skills = False
        try:
            if os.path.exists(loops_dir):
                shutil.move(loops_dir, loops_backup)
                moved_loops = True
            if os.path.exists(skills_dir):
                shutil.move(skills_dir, skills_backup)
                moved_skills = True

            r = run_rick('tools', 'learning_check', 'job_22')
            combined = r.stdout + r.stderr
            if r.returncode != 0:
                errors.append(
                    f'Edge (dirs absent): learning_check should pass, got exit={r.returncode}, output={combined}'
                )
            elif '✅ learning check passed' not in combined:
                errors.append(
                    f'Edge (dirs absent): expected "✅ learning check passed" but got: {combined}'
                )
        finally:
            if moved_loops:
                shutil.move(loops_backup, loops_dir)
            if moved_skills:
                shutil.move(skills_backup, skills_dir)

        # --- Set up valid fixture files ---
        os.makedirs(loops_dir, exist_ok=True)
        os.makedirs(skills_dir, exist_ok=True)

        good_loop = os.path.join(loops_dir, 'candidate_loop_1.md')
        created_files.append(good_loop)
        with open(good_loop, 'w') as f:
            f.write(
                '---\nname: test-loop\ntrigger: debugging\n---\n\n'
                '## 目标\n\nFind the bug.\n\n'
                '## 上下文管理\n\nLoad context.\n\n'
                '## 可调用工具\n\nUse bash.\n\n'
                '## 产出评估\n\nVerify fix.\n\n'
                '## 停止标准\n\nWhen done.\n'
            )

        good_skill = os.path.join(skills_dir, 'candidate_skill_1.md')
        created_files.append(good_skill)
        with open(good_skill, 'w') as f:
            f.write(
                '---\nname: test-skill\ndescription: A test skill\n---\n\n'
                '## When to Use\n\nUse it.\n\n'
                '## Procedure\n\n1. Do it.\n\n'
                '## Pitfalls\n\nWatch out.\n\n'
                '## Verification\n\nCheck output.\n'
            )

        # --- Test 3: Normal path - valid loops/skills → learning_check passes ---
        print('Test 3: normal path valid files', file=sys.stderr)
        r = run_rick('tools', 'learning_check', 'job_22')
        combined = r.stdout + r.stderr
        if r.returncode != 0:
            errors.append(
                f'Normal path: learning_check should pass with valid files, got exit={r.returncode}, output={combined}'
            )
        elif '✅ learning check passed' not in combined:
            errors.append(f'Normal path: expected pass but got: {combined}')

        # --- Test 4: README.md is skipped ---
        print('Test 4: README.md skipped', file=sys.stderr)
        readme = os.path.join(loops_dir, 'README.md')
        created_files.append(readme)
        with open(readme, 'w') as f:
            f.write('# Loops Directory\n\nFormat spec. No frontmatter, no required sections.\n')

        r = run_rick('tools', 'learning_check', 'job_22')
        combined = r.stdout + r.stderr
        if r.returncode != 0:
            errors.append(
                f'README.md skip: learning_check should ignore README.md, got exit={r.returncode}, output={combined}'
            )
        elif '✅ learning check passed' not in combined:
            errors.append(f'README.md skip: expected pass but got: {combined}')

        # --- Test 5: Abnormal - loop missing trigger → learning_check reports error ---
        print('Test 5: bad loop missing trigger', file=sys.stderr)
        bad_loop = os.path.join(loops_dir, 'bad_loop.md')
        with open(bad_loop, 'w') as f:
            f.write(
                '---\nname: bad-loop\n---\n\n'
                '## 目标\n\nSomething.\n\n'
                '## 上下文管理\n\nContext.\n\n'
                '## 可调用工具\n\nTools.\n\n'
                '## 产出评估\n\nOutput.\n\n'
                '## 停止标准\n\nStop.\n'
            )
        try:
            r = run_rick('tools', 'learning_check', 'job_22')
            combined = r.stdout + r.stderr
            if r.returncode == 0:
                errors.append('Bad loop (missing trigger): learning_check should fail but passed')
            else:
                if 'bad_loop.md' not in combined:
                    errors.append(f'Bad loop: error should mention "bad_loop.md", got: {combined}')
                if 'trigger' not in combined:
                    errors.append(f'Bad loop: error should mention "trigger", got: {combined}')
        finally:
            os.remove(bad_loop)

        # --- Test 6: Abnormal - skill missing Procedure → dream_check reports error ---
        print('Test 6: bad skill missing Procedure section', file=sys.stderr)
        bad_skill = os.path.join(skills_dir, 'bad_skill.md')
        with open(bad_skill, 'w') as f:
            f.write(
                '---\nname: bad-skill\ndescription: A bad skill\n---\n\n'
                '## When to Use\n\nUse it.\n\n'
                '## Pitfalls\n\nWatch out.\n\n'
                '## Verification\n\nCheck.\n'
            )
        try:
            r = run_rick('tools', 'dream_check')
            combined = r.stdout + r.stderr
            if r.returncode == 0:
                errors.append('Bad skill (missing Procedure): dream_check should fail but passed')
            else:
                if 'bad_skill.md' not in combined:
                    errors.append(f'Bad skill: error should mention "bad_skill.md", got: {combined}')
                if 'Procedure' not in combined:
                    errors.append(f'Bad skill: error should mention "Procedure", got: {combined}')
        finally:
            os.remove(bad_skill)

    finally:
        for f in created_files:
            if os.path.exists(f):
                os.remove(f)

    result = {'pass': len(errors) == 0, 'errors': errors}
    print(json.dumps(result, ensure_ascii=False))
    sys.exit(0 if result['pass'] else 1)

if __name__ == '__main__':
    main()
