#!/usr/bin/env python3
import json
import sys
import os
import subprocess

PROJECT_ROOT = os.path.abspath(os.path.join(os.path.dirname(os.path.abspath(__file__)), '..', '..', '..', '..', '..'))


def run_go_test(package, run_pattern):
    """Run a specific go test and return (passed, output)."""
    result = subprocess.run(
        ['go', 'test', package, '-run', run_pattern, '-v', '-count=1'],
        capture_output=True, text=True, cwd=PROJECT_ROOT, timeout=120
    )
    return result.returncode == 0, result.stdout + result.stderr


def main():
    errors = []

    # ── Step 1: go test ./internal/prompt/ -run TestIntegration_RFC001 ───────
    print('[step 1] TestIntegration_RFC001 ...', file=sys.stderr)
    passed, output = run_go_test('./internal/prompt/', 'TestIntegration_RFC001')
    if not passed:
        errors.append(f'TestIntegration_RFC001 FAILED:\n{output[:1000]}')
    else:
        print('  PASS', file=sys.stderr)

    # ── Step 2: go test ./internal/workspace/ -run TestLoadToolsList ─────────
    print('[step 2] TestLoadToolsList ...', file=sys.stderr)
    passed, output = run_go_test('./internal/workspace/', 'TestLoadToolsList')
    if not passed:
        errors.append(f'TestLoadToolsList FAILED:\n{output[:1000]}')
    else:
        print('  PASS', file=sys.stderr)

    # ── Step 3: go test ./internal/workspace/ -run TestLoadSkillsIndex ───────
    print('[step 3] TestLoadSkillsIndex ...', file=sys.stderr)
    passed, output = run_go_test('./internal/workspace/', 'TestLoadSkillsIndex')
    if not passed:
        errors.append(f'TestLoadSkillsIndex FAILED:\n{output[:1000]}')
    else:
        print('  PASS', file=sys.stderr)

    # ── Step 4: go test ./internal/cmd/ -run TestDryRun ─────────────────────
    print('[step 4] TestDryRun ...', file=sys.stderr)
    passed, output = run_go_test('./internal/cmd/', 'TestDryRun')
    if not passed:
        errors.append(f'TestDryRun FAILED:\n{output[:1000]}')
    else:
        print('  PASS', file=sys.stderr)

    # ── Step 5: go test ./... (full suite, no new failures) ──────────────────
    print('[step 5] go test ./... ...', file=sys.stderr)
    try:
        result = subprocess.run(
            ['go', 'test', './...'],
            capture_output=True, text=True, cwd=PROJECT_ROOT, timeout=300
        )
        if result.returncode != 0:
            fail_lines = [l for l in (result.stdout + result.stderr).splitlines() if 'FAIL' in l]
            errors.append(f'go test ./... has failures: {"; ".join(fail_lines[:5])}')
        else:
            print('  PASS', file=sys.stderr)
    except Exception as e:
        errors.append(f'go test ./... exception: {e}')

    # ── Step 6: bin/rick plan --dry-run prints full prompt ───────────────────
    print('[step 6] rick plan --dry-run output ...', file=sys.stderr)
    rick_bin = os.path.join(PROJECT_ROOT, 'bin', 'rick')
    if not os.path.exists(rick_bin):
        errors.append(f'bin/rick not found at {rick_bin}; run go build ./cmd/rick/ first')
    else:
        try:
            result = subprocess.run(
                [rick_bin, 'plan', '--dry-run', '测试需求'],
                capture_output=True, text=True, cwd=PROJECT_ROOT, timeout=30
            )
            out = result.stdout
            if '[DRY-RUN]' not in out:
                errors.append("rick plan --dry-run: output missing '[DRY-RUN]' header")
            elif len(out.strip().splitlines()) < 5:
                errors.append(f'rick plan --dry-run: output too short ({len(out.strip().splitlines())} lines), expected full prompt')
            else:
                print('  PASS', file=sys.stderr)
        except subprocess.TimeoutExpired:
            errors.append('rick plan --dry-run: timed out after 30s')
        except Exception as e:
            errors.append(f'rick plan --dry-run: {e}')

    # ── Step 7: bin/rick learning --dry-run job_N prints full prompt ─────────
    print('[step 7] rick learning --dry-run output ...', file=sys.stderr)
    if os.path.exists(rick_bin):
        try:
            result = subprocess.run(
                [rick_bin, 'learning', '--dry-run', 'job_9'],
                capture_output=True, text=True, cwd=PROJECT_ROOT, timeout=30
            )
            out = result.stdout
            if '[DRY-RUN]' not in out:
                errors.append("rick learning --dry-run: output missing '[DRY-RUN]' header")
            elif len(out.strip().splitlines()) < 5:
                errors.append(f'rick learning --dry-run: output too short ({len(out.strip().splitlines())} lines), expected full prompt')
            else:
                print('  PASS', file=sys.stderr)
        except subprocess.TimeoutExpired:
            errors.append('rick learning --dry-run: timed out after 30s')
        except Exception as e:
            errors.append(f'rick learning --dry-run: {e}')

    # ── Output ────────────────────────────────────────────────────────────────
    res = {'pass': len(errors) == 0, 'errors': errors}
    print(json.dumps(res))
    sys.exit(0 if res['pass'] else 1)


if __name__ == '__main__':
    main()
