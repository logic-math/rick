#!/usr/bin/env python3
import json
import sys
import os
import subprocess

def run_cmd(cmd, cwd=None):
    result = subprocess.run(cmd, shell=True, capture_output=True, text=True, cwd=cwd)
    return result.returncode, result.stdout, result.stderr

def main():
    errors = []
    project_root = "/Users/sunquan/ai_coding/CODING/rick"

    # Test 1: go build ./...
    code, stdout, stderr = run_cmd("go build ./...", cwd=project_root)
    if code != 0:
        errors.append(f"go build failed: {stderr}")

    # Test 2: go test ./internal/cmd/ -run TestLearning -v
    code, stdout, stderr = run_cmd("go test ./internal/cmd/ -run TestLearning -v", cwd=project_root)
    if code != 0:
        errors.append(f"go test ./internal/cmd/ -run TestLearning failed: {stderr}\n{stdout}")

    # Test 3: go test ./internal/prompt/ -run TestLearning -v
    code, stdout, stderr = run_cmd("go test ./internal/prompt/ -run TestLearning -v", cwd=project_root)
    if code != 0:
        errors.append(f"go test ./internal/prompt/ -run TestLearning failed: {stderr}\n{stdout}")

    # Test 4: go test ./...
    code, stdout, stderr = run_cmd("go test ./...", cwd=project_root)
    if code != 0:
        errors.append(f"go test ./... failed: {stderr}\n{stdout}")

    # Test 5: grep for hardcoded fake data string
    code, stdout, stderr = run_cmd('grep -r "本周期内新增" .', cwd=project_root)
    if code == 0 and stdout.strip():
        errors.append(f'Found hardcoded fake data string "本周期内新增" in: {stdout.strip()}')

    result = {
        'pass': len(errors) == 0,
        'errors': errors
    }

    print(json.dumps(result))
    sys.exit(0 if result['pass'] else 1)

if __name__ == '__main__':
    main()
