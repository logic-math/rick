#!/usr/bin/env python3
import json
import sys
import os
import subprocess

def main():
    errors = []

    project_root = os.path.dirname(os.path.dirname(os.path.dirname(os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__)))))))
    templates_dir = os.path.join(project_root, 'internal', 'prompt', 'templates')

    plan_md = os.path.join(templates_dir, 'plan.md')
    doing_md = os.path.join(templates_dir, 'doing.md')
    test_python_md = os.path.join(templates_dir, 'test_python.md')

    # Test 1: plan.md SOP a-j 全覆盖 (10步)
    if not os.path.exists(plan_md):
        errors.append(f'plan.md does not exist at {plan_md}')
    else:
        try:
            with open(plan_md, 'r', encoding='utf-8') as f:
                content = f.read()

            # Check for a-j 10-step SOP steps
            sop_steps = ['a.', 'b.', 'c.', 'd.', 'e.', 'f.', 'g.', 'h.', 'i.', 'j.']
            missing_steps = [s for s in sop_steps if s not in content]
            if missing_steps:
                errors.append(f'plan.md missing SOP steps: {missing_steps}')

            # Check for sense skill reference
            if 'sense' not in content.lower():
                errors.append('plan.md does not reference sense skill')

            # Check for tc skill reference
            if '`tc`' not in content and 'tc skill' not in content.lower() and '`tc`' not in content:
                # More lenient check
                if 'tc' not in content:
                    errors.append('plan.md does not reference tc skill')

        except Exception as e:
            errors.append(f'Failed to read plan.md: {str(e)}')

    # Test 2: doing.md Cialdini 三原则
    if not os.path.exists(doing_md):
        errors.append(f'doing.md does not exist at {doing_md}')
    else:
        try:
            with open(doing_md, 'r', encoding='utf-8') as f:
                content = f.read()

            cialdini_principles = ['权威', '承诺', '稀缺']
            missing = [p for p in cialdini_principles if p not in content]
            if missing:
                errors.append(f'doing.md missing Cialdini principles: {missing}')

        except Exception as e:
            errors.append(f'Failed to read doing.md: {str(e)}')

    # Test 3: test_python.md Cialdini
    if not os.path.exists(test_python_md):
        errors.append(f'test_python.md does not exist at {test_python_md}')
    else:
        try:
            with open(test_python_md, 'r', encoding='utf-8') as f:
                content = f.read()

            cialdini_principles = ['权威', '承诺', '稀缺']
            missing = [p for p in cialdini_principles if p not in content]
            if missing:
                errors.append(f'test_python.md missing Cialdini principles: {missing}')

        except Exception as e:
            errors.append(f'Failed to read test_python.md: {str(e)}')

    # Test 4: Build check
    build_script = os.path.join(project_root, 'tools', 'build_and_get_rick_bin.py')
    if not os.path.exists(build_script):
        errors.append(f'build script not found at {build_script}')
    else:
        try:
            result = subprocess.run(
                ['python3', build_script],
                cwd=project_root,
                capture_output=True,
                text=True,
                timeout=120
            )
            if result.returncode != 0:
                errors.append(f'Build failed: {result.stderr[:500]}')
        except subprocess.TimeoutExpired:
            errors.append('Build timed out after 120 seconds')
        except Exception as e:
            errors.append(f'Build error: {str(e)}')

    # Test 5: go test ./...
    try:
        result = subprocess.run(
            ['go', 'test', './...'],
            cwd=project_root,
            capture_output=True,
            text=True,
            timeout=120
        )
        if result.returncode != 0:
            errors.append(f'go test ./... failed: {result.stdout[-500:]} {result.stderr[-500:]}')
    except subprocess.TimeoutExpired:
        errors.append('go test ./... timed out after 120 seconds')
    except Exception as e:
        errors.append(f'go test error: {str(e)}')

    result = {
        'pass': len(errors) == 0,
        'errors': errors
    }

    print(json.dumps(result))
    sys.exit(0 if result['pass'] else 1)

if __name__ == '__main__':
    main()
