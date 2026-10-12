#!/usr/bin/env python3
# Description: Verify grilling.md skill file creation and LoadCoreSkills integration
import json
import sys
import os
import subprocess
import re


def main():
    errors = []

    # Project root: 6 dirname calls from .rick/jobs/job_18/doing/tests/task1.py
    this_file = os.path.abspath(__file__)
    project_root = this_file
    for _ in range(6):
        project_root = os.path.dirname(project_root)

    print(f"project_root: {project_root}", file=sys.stderr)

    # Test 1: grilling.md must exist
    grilling_path = os.path.join(
        project_root, 'internal', 'prompt', 'templates', 'skills', 'grilling.md'
    )
    if not os.path.exists(grilling_path):
        errors.append(
            f'grilling.md does not exist at internal/prompt/templates/skills/grilling.md'
        )
    else:
        try:
            with open(grilling_path, 'r', encoding='utf-8') as f:
                content = f.read()

            # Test 2: must contain the required grilling protocol phrase
            if 'Interview me relentlessly' not in content:
                errors.append(
                    'grilling.md missing required phrase: "Interview me relentlessly"'
                )

            # Test 3: content length must be > 100 chars (non-trivial file)
            if len(content) <= 100:
                errors.append(
                    f'grilling.md too short: {len(content)} chars, expected > 100'
                )

            # Test 4 (boundary): no unresolved {{...}} template variables
            if re.search(r'\{\{', content):
                errors.append(
                    'grilling.md contains unresolved {{...}} template variables'
                )

        except Exception as e:
            errors.append(f'Failed to read grilling.md: {str(e)}')

    # Test 5: go test TestLoadCoreSkills_Grilling must exist and pass
    try:
        result = subprocess.run(
            ['go', 'test', './internal/prompt/...', '-run', 'TestLoadCoreSkills_Grilling', '-v'],
            capture_output=True, text=True, cwd=project_root, timeout=120
        )
        output = result.stdout + result.stderr
        print(f"go test output:\n{output[:600]}", file=sys.stderr)

        if result.returncode != 0:
            errors.append(
                f'go test failed (exit {result.returncode}): {output[:400]}'
            )
        elif 'no tests to run' in output:
            errors.append(
                'TestLoadCoreSkills_Grilling not found — test function must be added to manager_test.go'
            )
        elif 'PASS' not in output:
            errors.append(
                f'TestLoadCoreSkills_Grilling did not PASS: {output[:400]}'
            )

    except subprocess.TimeoutExpired:
        errors.append('go test timed out after 120s')
    except Exception as e:
        errors.append(f'Failed to run go test: {str(e)}')

    out = {'pass': len(errors) == 0, 'errors': errors}
    print(json.dumps(out, ensure_ascii=False))
    sys.exit(0 if out['pass'] else 1)


if __name__ == '__main__':
    main()
