#!/usr/bin/env python3
import json
import sys
import os
import subprocess
import shutil
import tempfile

def run_cmd(cmd, cwd=None):
    result = subprocess.run(cmd, shell=True, capture_output=True, text=True, cwd=cwd)
    return result.returncode, result.stdout, result.stderr

def main():
    errors = []

    # Project root is 5 levels up from tests/ dir (.rick/jobs/job_5/doing/tests/task5.py)
    project_root = os.path.dirname(os.path.dirname(os.path.dirname(os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__)))))))
    print(f"project_root: {project_root}", file=sys.stderr)

    binary = os.path.join(project_root, 'bin', 'rick')

    # Test 1: Build the binary
    rc, out, err = run_cmd('go build -o bin/rick ./cmd/rick/', cwd=project_root)
    if rc != 0:
        errors.append(f'go build failed: {err}')
        print(json.dumps({'pass': False, 'errors': errors}))
        sys.exit(1)

    # Test 2: doing_check on existing job_1 should pass
    rc, out, err = run_cmd(f'{binary} tools doing_check job_1', cwd=project_root)
    if rc != 0:
        errors.append(f'doing_check job_1 should pass but failed: stdout={out.strip()} stderr={err.strip()}')

    # Test 3: doing_check detects missing debug.md
    job1_doing = os.path.join(project_root, '.rick', 'jobs', 'job_1', 'doing')
    debug_md = os.path.join(job1_doing, 'debug.md')
    debug_md_backup = debug_md + '.bak'
    debug_md_existed = os.path.exists(debug_md)

    if debug_md_existed:
        shutil.copy2(debug_md, debug_md_backup)
        os.remove(debug_md)

    try:
        rc, out, err = run_cmd(f'{binary} tools doing_check job_1', cwd=project_root)
        combined = (out + err).lower()
        if rc == 0:
            errors.append('doing_check job_1 should fail when debug.md is missing, but it passed')
        elif 'debug.md' not in combined and 'debug' not in combined:
            errors.append(f'doing_check missing debug.md error message should mention "debug.md", got: {out.strip()} {err.strip()}')
    finally:
        if debug_md_existed:
            shutil.copy2(debug_md_backup, debug_md)
            os.remove(debug_md_backup)

    # Test 4: learning_check detects syntax error in skills .py file
    job1_learning = os.path.join(project_root, '.rick', 'jobs', 'job_1', 'learning')
    skills_dir = os.path.join(job1_learning, 'skills')
    os.makedirs(skills_dir, exist_ok=True)
    bad_py = os.path.join(skills_dir, 'test_syntax_error_tmp.py')
    try:
        with open(bad_py, 'w') as f:
            f.write('def foo(\n    pass\n')  # syntax error: unclosed parenthesis

        rc, out, err = run_cmd(f'{binary} tools learning_check job_1', cwd=project_root)
        combined = out + err
        if rc == 0:
            errors.append('learning_check should fail when a .py file has syntax errors, but it passed')
        else:
            if 'test_syntax_error_tmp.py' not in combined and 'syntax' not in combined.lower():
                errors.append(f'learning_check error should mention filename or syntax error, got: {combined.strip()}')
    finally:
        if os.path.exists(bad_py):
            os.remove(bad_py)

    # Test 5: doing_check --help
    rc, out, err = run_cmd(f'{binary} tools doing_check --help', cwd=project_root)
    if rc != 0:
        errors.append(f'doing_check --help failed with rc={rc}: {err.strip()}')
    combined = (out + err).lower()
    if 'doing' not in combined and 'check' not in combined:
        errors.append(f'doing_check --help output missing expected content, got: {combined.strip()}')

    # Test 6: learning_check --help
    rc, out, err = run_cmd(f'{binary} tools learning_check --help', cwd=project_root)
    if rc != 0:
        errors.append(f'learning_check --help failed with rc={rc}: {err.strip()}')
    combined = (out + err).lower()
    if 'learning' not in combined and 'check' not in combined:
        errors.append(f'learning_check --help output missing expected content, got: {combined.strip()}')

    # Test 7: go test ./internal/cmd/...
    rc, out, err = run_cmd('go test ./internal/cmd/...', cwd=project_root)
    if rc != 0:
        errors.append(f'go test ./internal/cmd/... failed: {err.strip()}')

    result = {
        'pass': len(errors) == 0,
        'errors': errors
    }

    print(json.dumps(result))
    sys.exit(0 if result['pass'] else 1)

if __name__ == '__main__':
    main()
