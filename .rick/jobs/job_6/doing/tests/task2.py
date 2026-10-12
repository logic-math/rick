#!/usr/bin/env python3
import json
import sys
import os
import subprocess

def main():
    errors = []

    # File is at: .rick/jobs/job_6/doing/tests/task2.py
    # project_root is 5 levels up: tests/ -> doing/ -> job_6/ -> jobs/ -> .rick/ -> project
    project_root = os.path.abspath(
        os.path.join(os.path.dirname(os.path.abspath(__file__)), '..', '..', '..', '..', '..')
    )
    print(f"project_root: {project_root}", file=sys.stderr)

    # Test 1: go build ./... - confirm no compile errors
    try:
        result = subprocess.run(
            ['go', 'build', './...'],
            cwd=project_root,
            capture_output=True,
            text=True,
            timeout=120
        )
        if result.returncode != 0:
            errors.append(f'go build ./... failed: {result.stderr.strip()}')
        else:
            print("Test 1 passed: go build ./...", file=sys.stderr)
    except Exception as e:
        errors.append(f'go build ./... error: {str(e)}')

    # Test 2: go test ./internal/prompt/... - all existing tests pass
    try:
        result = subprocess.run(
            ['go', 'test', './internal/prompt/...'],
            cwd=project_root,
            capture_output=True,
            text=True,
            timeout=120
        )
        if result.returncode != 0:
            errors.append(f'go test ./internal/prompt/... failed: {result.stderr.strip() or result.stdout.strip()}')
        else:
            print("Test 2 passed: go test ./internal/prompt/...", file=sys.stderr)
    except Exception as e:
        errors.append(f'go test ./internal/prompt/... error: {str(e)}')

    # Test 3: go test ./internal/workspace/... - GetRFCDir returns correct path containing ".rick" and "RFC"
    try:
        result = subprocess.run(
            ['go', 'test', './internal/workspace/...', '-run', 'TestGetRFCDir', '-v'],
            cwd=project_root,
            capture_output=True,
            text=True,
            timeout=120
        )
        if result.returncode != 0:
            errors.append(f'go test ./internal/workspace/... (TestGetRFCDir) failed: {result.stderr.strip() or result.stdout.strip()}')
        else:
            combined = result.stdout + result.stderr
            if 'PASS' not in combined and '--- PASS' not in combined:
                errors.append(f'TestGetRFCDir did not report PASS: {combined.strip()}')
            else:
                print("Test 3 passed: TestGetRFCDir", file=sys.stderr)
    except Exception as e:
        errors.append(f'go test ./internal/workspace/... error: {str(e)}')

    # Test 4: Check that internal/workspace/paths.go contains GetRFCDir with .rick/RFC
    paths_go = os.path.join(project_root, 'internal', 'workspace', 'paths.go')
    if not os.path.exists(paths_go):
        errors.append('internal/workspace/paths.go does not exist')
    else:
        try:
            with open(paths_go, 'r') as f:
                content = f.read()
            if 'GetRFCDir' not in content:
                errors.append('GetRFCDir function not found in internal/workspace/paths.go')
            else:
                print("Test 4a passed: GetRFCDir exists in paths.go", file=sys.stderr)
            if 'RFC' not in content:
                errors.append('paths.go does not reference RFC directory')
            else:
                print("Test 4b passed: RFC referenced in paths.go", file=sys.stderr)
        except Exception as e:
            errors.append(f'Failed to read paths.go: {str(e)}')

    # Test 5: Check that internal/prompt/human_loop_prompt.go exists and contains GenerateHumanLoopPromptFile
    human_loop_file = os.path.join(project_root, 'internal', 'prompt', 'human_loop_prompt.go')
    if not os.path.exists(human_loop_file):
        errors.append('internal/prompt/human_loop_prompt.go does not exist')
    else:
        try:
            with open(human_loop_file, 'r') as f:
                content = f.read()
            if 'GenerateHumanLoopPromptFile' not in content:
                errors.append('GenerateHumanLoopPromptFile not found in human_loop_prompt.go')
            else:
                print("Test 5 passed: GenerateHumanLoopPromptFile exists", file=sys.stderr)
        except Exception as e:
            errors.append(f'Failed to read human_loop_prompt.go: {str(e)}')

    # Test 6: Call GenerateHumanLoopPromptFile via a temporary Go test to verify runtime behavior
    # NewPromptManager is variadic: NewPromptManager() uses embedded templates
    test_code = '''package prompt_test

import (
\t"os"
\t"strings"
\t"testing"

\t"github.com/sunquan/rick/internal/prompt"
)

func TestGenerateHumanLoopPromptFileIntegration(t *testing.T) {
\tpm := prompt.NewPromptManager()
\ttmpDir, err := os.MkdirTemp("", "rfc_test_*")
\tif err != nil {
\t\tt.Fatalf("failed to create temp dir: %v", err)
\t}
\tdefer os.RemoveAll(tmpDir)

\tfilePath, err := prompt.GenerateHumanLoopPromptFile("如何重构?", tmpDir, pm)
\tif err != nil {
\t\tt.Fatalf("GenerateHumanLoopPromptFile failed: %v", err)
\t}
\tif filePath == "" {
\t\tt.Fatal("returned file path is empty")
\t}
\tif _, statErr := os.Stat(filePath); os.IsNotExist(statErr) {
\t\tt.Fatalf("returned file path does not exist: %s", filePath)
\t}
\tcontent, err := os.ReadFile(filePath)
\tif err != nil {
\t\tt.Fatalf("failed to read prompt file: %v", err)
\t}
\tif !strings.Contains(string(content), "如何重构?") {
\t\tt.Fatalf("prompt file does not contain topic '如何重构?'")
\t}
\tos.Remove(filePath)
}
'''
    test_file = os.path.join(project_root, 'internal', 'prompt', 'human_loop_integration_test.go')
    try:
        with open(test_file, 'w') as f:
            f.write(test_code)

        result = subprocess.run(
            ['go', 'test', './internal/prompt/...', '-v', '-run', 'TestGenerateHumanLoopPromptFileIntegration'],
            cwd=project_root,
            capture_output=True,
            text=True,
            timeout=120
        )
        if result.returncode != 0:
            errors.append(
                f'GenerateHumanLoopPromptFile integration test failed: '
                f'{result.stderr.strip() or result.stdout.strip()}'
            )
        else:
            print("Test 6 passed: GenerateHumanLoopPromptFile integration test", file=sys.stderr)
    except Exception as e:
        errors.append(f'GenerateHumanLoopPromptFile integration test error: {str(e)}')
    finally:
        if os.path.exists(test_file):
            os.remove(test_file)

    result = {
        'pass': len(errors) == 0,
        'errors': errors
    }

    print(json.dumps(result))
    sys.exit(0 if result['pass'] else 1)


if __name__ == '__main__':
    main()
