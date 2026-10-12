#!/usr/bin/env python3
import json
import sys
import os
import subprocess
import tempfile

def main():
    errors = []

    # The project root is at the known absolute path
    project_root = '/opt/meituan/dolphinfs_sunquan20/ai_coding/Coding/rick'

    # Test 1: go build ./...
    try:
        result = subprocess.run(
            ['go', 'build', './...'],
            cwd=project_root,
            capture_output=True,
            text=True
        )
        if result.returncode != 0:
            errors.append(f'go build ./... failed: {result.stderr.strip()}')
    except Exception as e:
        errors.append(f'Failed to run go build: {str(e)}')

    # Test 2: go test ./internal/prompt/...
    try:
        result = subprocess.run(
            ['go', 'test', './internal/prompt/...'],
            cwd=project_root,
            capture_output=True,
            text=True
        )
        if result.returncode != 0:
            errors.append(f'go test ./internal/prompt/... failed: {result.stdout.strip()}\n{result.stderr.strip()}')
    except Exception as e:
        errors.append(f'Failed to run go test: {str(e)}')

    # Test 3: Check human_loop.md template file exists
    template_path = os.path.join(project_root, 'internal', 'prompt', 'templates', 'human_loop.md')
    if not os.path.exists(template_path):
        errors.append(f'Template file does not exist: {template_path}')
    else:
        # Test 4: Check template contains {{topic}} and {{rfc_dir}} placeholders
        try:
            with open(template_path, 'r', encoding='utf-8') as f:
                content = f.read()
            if '{{topic}}' not in content:
                errors.append('human_loop.md missing {{topic}} placeholder')
            if '{{rfc_dir}}' not in content:
                errors.append('human_loop.md missing {{rfc_dir}} placeholder')
        except Exception as e:
            errors.append(f'Failed to read human_loop.md: {str(e)}')

    # Test 5: Write a Go test file inside the module to verify LoadTemplate and variables
    go_test_code = '''package prompt_test

import (
	"sort"
	"testing"

	"github.com/sunquan/rick/internal/prompt"
)

func TestHumanLoopTemplateLoadAndVariables(t *testing.T) {
	pm := prompt.NewPromptManager()
	tmpl, err := pm.LoadTemplate("human_loop")
	if err != nil {
		t.Fatalf("LoadTemplate failed: %v", err)
	}

	// Check template content contains the two placeholders
	if len(tmpl.Content) == 0 {
		t.Fatal("template content is empty")
	}

	// Check variables via GetMissingVariables (no variables set => all are missing)
	builder := prompt.NewPromptBuilder(tmpl)
	missing := builder.GetMissingVariables()
	sort.Strings(missing)

	expected := []string{"rfc_dir", "topic"}
	if len(missing) != len(expected) {
		t.Fatalf("GetMissingVariables() = %v (len=%d), want %v (len=%d)", missing, len(missing), expected, len(expected))
	}
	for i, v := range missing {
		if v != expected[i] {
			t.Errorf("variable mismatch at index %d: expected %s, got %s", i, expected[i], v)
		}
	}
}
'''

    test_file = os.path.join(project_root, 'internal', 'prompt', 'human_loop_vars_test.go')
    try:
        with open(test_file, 'w', encoding='utf-8') as f:
            f.write(go_test_code)

        result = subprocess.run(
            ['go', 'test', '-run', 'TestHumanLoopTemplateLoadAndVariables', './internal/prompt/...'],
            cwd=project_root,
            capture_output=True,
            text=True
        )
        print(f'go test stdout: {result.stdout.strip()}', file=sys.stderr)
        print(f'go test stderr: {result.stderr.strip()}', file=sys.stderr)

        if result.returncode != 0:
            errors.append(f'LoadTemplate/variable check failed: {result.stdout.strip()}\n{result.stderr.strip()}')
    except Exception as e:
        errors.append(f'Failed to run Go variable check: {str(e)}')
    finally:
        # Clean up temp test file
        if os.path.exists(test_file):
            os.remove(test_file)

    # Build result
    result = {
        'pass': len(errors) == 0,
        'errors': errors
    }

    print(json.dumps(result))
    sys.exit(0 if result['pass'] else 1)

if __name__ == '__main__':
    main()
