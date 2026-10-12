#!/usr/bin/env python3
import json
import sys
import os
import subprocess
import re

def main():
    errors = []

    # Navigate to project root: tests/ -> doing/ -> job_16/ -> jobs/ -> .rick/ -> project root
    script_dir = os.path.dirname(os.path.abspath(__file__))
    project_root = os.path.dirname(os.path.dirname(os.path.dirname(os.path.dirname(os.path.dirname(script_dir)))))

    print(f"project_root: {project_root}", file=sys.stderr)

    # Test 1: LoadDebugContext exists in executor package
    debug_context_files = []
    executor_dir = os.path.join(project_root, 'internal', 'executor')
    for fname in os.listdir(executor_dir):
        if fname.endswith('.go'):
            fpath = os.path.join(executor_dir, fname)
            try:
                with open(fpath, 'r') as f:
                    content = f.read()
                    if 'func LoadDebugContext' in content:
                        debug_context_files.append(fname)
            except Exception as e:
                errors.append(f'Failed to read {fpath}: {e}')

    if not debug_context_files:
        errors.append('LoadDebugContext function not found in internal/executor/*.go')

    # Test 2: LoadDebugContext is called ≥ 5 times across all changed files
    files_to_check = [
        os.path.join(project_root, 'internal', 'executor', 'retry.go'),
        os.path.join(project_root, 'internal', 'executor', 'runner.go'),
        os.path.join(project_root, 'internal', 'cmd', 'learning.go'),
        os.path.join(project_root, 'internal', 'prompt', 'easy_prompt.go'),
    ]

    total_calls = 0
    for fpath in files_to_check:
        if not os.path.exists(fpath):
            errors.append(f'File not found: {fpath}')
            continue
        try:
            with open(fpath, 'r') as f:
                content = f.read()
                count = content.count('LoadDebugContext(')
                print(f"{os.path.basename(fpath)}: LoadDebugContext calls = {count}", file=sys.stderr)
                total_calls += count
        except Exception as e:
            errors.append(f'Failed to read {fpath}: {e}')

    if total_calls < 5:
        errors.append(f'LoadDebugContext called only {total_calls} times across changed files, expected ≥ 5')

    # Test 3: No residual old-style debug.md direct reads in the 4 changed files
    # retry.go should NOT have os.ReadFile(debugFile) for debug loading
    retry_go = os.path.join(project_root, 'internal', 'executor', 'retry.go')
    if os.path.exists(retry_go):
        try:
            with open(retry_go, 'r') as f:
                content = f.read()
            # old pattern: os.ReadFile(debugFile) inside loadDebugContext
            # After refactor, loadDebugContext should delegate to LoadDebugContext
            if 'os.ReadFile(debugFile)' in content:
                errors.append('retry.go still has residual os.ReadFile(debugFile); should use LoadDebugContext')
        except Exception as e:
            errors.append(f'Failed to read retry.go: {e}')

    # runner.go should NOT have contextMgr.LoadDebugFromFile for debug context
    runner_go = os.path.join(project_root, 'internal', 'executor', 'runner.go')
    if os.path.exists(runner_go):
        try:
            with open(runner_go, 'r') as f:
                content = f.read()
            if 'contextMgr.LoadDebugFromFile' in content:
                errors.append('runner.go still has contextMgr.LoadDebugFromFile; should use LoadDebugContext')
        except Exception as e:
            errors.append(f'Failed to read runner.go: {e}')

    # learning.go should NOT have os.ReadFile for debug.md reads
    learning_go = os.path.join(project_root, 'internal', 'cmd', 'learning.go')
    if os.path.exists(learning_go):
        try:
            with open(learning_go, 'r') as f:
                content = f.read()
            # old pattern from the task spec lines 102-103 and 164-171
            if re.search(r'os\.ReadFile\(.*debug.*\)', content):
                errors.append('learning.go still has os.ReadFile for debug path; should use LoadDebugContext')
        except Exception as e:
            errors.append(f'Failed to read learning.go: {e}')

    # Test 4: TODO fallback comment exists in LoadDebugContext implementation
    todo_found = False
    for fname in os.listdir(executor_dir):
        if fname.endswith('.go'):
            fpath = os.path.join(executor_dir, fname)
            try:
                with open(fpath, 'r') as f:
                    content = f.read()
                    if 'TODO' in content and 'debug' in content.lower() and ('2026-08' in content or 'fallback' in content.lower() or '回退' in content or 'debug.md' in content):
                        todo_found = True
                        break
            except Exception as e:
                errors.append(f'Failed to read {fpath}: {e}')

    if not todo_found:
        errors.append('TODO fallback comment not found in executor package (should mark debug.md fallback for 2026-08 removal)')

    # Test 5: go test ./internal/executor/... passes (no FAIL lines)
    try:
        result = subprocess.run(
            ['go', 'test', './internal/executor/...'],
            cwd=project_root,
            capture_output=True,
            text=True,
            timeout=120,
        )
        output = result.stdout + result.stderr
        print(f"go test executor output:\n{output}", file=sys.stderr)
        if 'FAIL' in output:
            errors.append(f'go test ./internal/executor/... has FAIL: {output[:500]}')
    except subprocess.TimeoutExpired:
        errors.append('go test ./internal/executor/... timed out')
    except Exception as e:
        errors.append(f'Failed to run go test ./internal/executor/...: {e}')

    # Test 6: go test on task4-affected packages (cmd + prompt)
    for pkg in ['./internal/cmd/...', './internal/prompt/...']:
        try:
            result = subprocess.run(
                ['go', 'test', pkg],
                cwd=project_root,
                capture_output=True,
                text=True,
                timeout=120,
            )
            output = result.stdout + result.stderr
            print(f"go test {pkg} output:\n{output}", file=sys.stderr)
            if 'FAIL' in output:
                errors.append(f'go test {pkg} has FAIL: {output[:500]}')
        except subprocess.TimeoutExpired:
            errors.append(f'go test {pkg} timed out')
        except Exception as e:
            errors.append(f'Failed to run go test {pkg}: {e}')

    result = {
        'pass': len(errors) == 0,
        'errors': errors,
    }

    print(json.dumps(result, ensure_ascii=False))
    sys.exit(0 if result['pass'] else 1)

if __name__ == '__main__':
    main()
