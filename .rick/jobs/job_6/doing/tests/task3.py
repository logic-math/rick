#!/usr/bin/env python3
import json
import sys
import os
import subprocess
import tempfile
import stat

def main():
    errors = []

    # Derive project root from this script's location:
    # .rick/jobs/job_6/doing/tests/task3.py -> 5 levels up
    project_root = os.path.abspath(
        os.path.join(os.path.dirname(os.path.abspath(__file__)), '..', '..', '..', '..', '..')
    )
    # Fallback to known path if derivation fails
    if not os.path.exists(os.path.join(project_root, 'internal', 'cmd')):
        project_root = "/opt/meituan/dolphinfs_sunquan20/ai_coding/Coding/rick"
    print(f"[DEBUG] project_root: {project_root}", file=sys.stderr)

    human_loop_file = os.path.join(project_root, 'internal', 'cmd', 'human_loop.go')
    plan_file = os.path.join(project_root, 'internal', 'cmd', 'plan.go')
    root_file = os.path.join(project_root, 'internal', 'cmd', 'root.go')

    # Test 1: human_loop.go exists with required content
    if not os.path.exists(human_loop_file):
        errors.append(f'human_loop.go does not exist at {human_loop_file}')
    else:
        try:
            with open(human_loop_file, 'r') as f:
                content = f.read()
            if 'NewHumanLoopCmd' not in content:
                errors.append('human_loop.go does not define NewHumanLoopCmd()')
            if 'func callClaudeCodeCLI' in content:
                errors.append('human_loop.go must NOT redefine callClaudeCodeCLI (already defined in plan.go)')
            if 'callClaudeCodeCLI' not in content:
                errors.append('human_loop.go does not call callClaudeCodeCLI')
            if 'GenerateHumanLoopPromptFile' not in content:
                errors.append('human_loop.go does not call GenerateHumanLoopPromptFile')
            if 'GetRFCDir' not in content:
                errors.append('human_loop.go does not call GetRFCDir()')
            if 'MkdirAll' not in content and 'Mkdir' not in content:
                errors.append('human_loop.go does not call os.MkdirAll/os.Mkdir to auto-create RFC directory')
            if 'topic is required' not in content:
                errors.append('human_loop.go missing "topic is required" error message')
            if 'DRY-RUN' not in content:
                errors.append('human_loop.go missing dry-run handling')
            if '思考记录已保存到 .rick/RFC/' not in content:
                errors.append('human_loop.go missing completion message about .rick/RFC/ directory')
        except Exception as e:
            errors.append(f'Failed to read human_loop.go: {str(e)}')

    # Test 2: callClaudeCodeCLI defined in plan.go (not in human_loop.go)
    if not os.path.exists(plan_file):
        errors.append(f'plan.go does not exist at {plan_file}')
    else:
        try:
            with open(plan_file, 'r') as f:
                plan_content = f.read()
            if 'func callClaudeCodeCLI' not in plan_content:
                errors.append('plan.go missing callClaudeCodeCLI function definition')
        except Exception as e:
            errors.append(f'Failed to read plan.go: {str(e)}')

    # Test 3: root.go registers NewHumanLoopCmd
    if not os.path.exists(root_file):
        errors.append(f'root.go does not exist at {root_file}')
    else:
        try:
            with open(root_file, 'r') as f:
                root_content = f.read()
            if 'NewHumanLoopCmd' not in root_content:
                errors.append('root.go does not register NewHumanLoopCmd (missing AddCommand call)')
        except Exception as e:
            errors.append(f'Failed to read root.go: {str(e)}')

    # Test 4: go build ./... succeeds (no redeclared errors)
    # Use -C flag because subprocess cwd= is unreliable in this environment
    try:
        result = subprocess.run(
            ['go', 'build', '-C', project_root, './...'],
            capture_output=True,
            text=True,
            timeout=120
        )
        if result.returncode != 0:
            stderr = result.stderr.strip()
            if 'redeclared' in stderr:
                errors.append(f'go build failed with redeclared error: {stderr}')
            else:
                errors.append(f'go build ./... failed: {stderr}')
        else:
            print("[DEBUG] go build ./... passed", file=sys.stderr)
    except subprocess.TimeoutExpired:
        errors.append('go build ./... timed out after 120s')
    except Exception as e:
        errors.append(f'Failed to run go build: {str(e)}')

    # Test 5: go test for HumanLoop tests passes (run -run TestHumanLoop to avoid unrelated hanging tests)
    try:
        result = subprocess.run(
            ['go', 'test', '-C', project_root, '-run', 'TestHumanLoop', '-timeout', '60s', '-v', './internal/cmd/...'],
            capture_output=True,
            text=True,
            timeout=90
        )
        if result.returncode != 0:
            errors.append(f'go test -run TestHumanLoop failed:\n{result.stdout.strip()}\n{result.stderr.strip()}')
        else:
            print("[DEBUG] go test -run TestHumanLoop passed", file=sys.stderr)
    except subprocess.TimeoutExpired:
        errors.append('go test -run TestHumanLoop timed out after 90s')
    except Exception as e:
        errors.append(f'Failed to run go test: {str(e)}')

    # Build rick binary for CLI tests (always build fresh to /tmp to avoid using wrong binary)
    rick_binary = '/tmp/rick_task3_test'
    try:
        build_result = subprocess.run(
            ['go', 'build', '-C', project_root, '-o', rick_binary, './cmd/rick/'],
            capture_output=True,
            text=True,
            timeout=120
        )
        if build_result.returncode != 0:
            errors.append(f'Failed to build rick binary: {build_result.stderr.strip()}')
            rick_binary = None
    except Exception as e:
        errors.append(f'Failed to build rick binary: {str(e)}')
        rick_binary = None

    if rick_binary and os.path.exists(rick_binary):
        # Test 6: dry-run mode outputs correct message and exits 0
        try:
            result = subprocess.run(
                [rick_binary, 'human-loop', '--dry-run', '如何重构?'],
                capture_output=True,
                text=True,
                timeout=30
            )
            output = (result.stdout + result.stderr).strip()
            if result.returncode != 0:
                errors.append(f'rick human-loop --dry-run exited with code {result.returncode}: {output}')
            elif '[DRY-RUN] Would start human-loop session for topic: 如何重构?' not in output:
                errors.append(f'rick human-loop --dry-run wrong output. Got: "{output}"')
            else:
                print(f"[DEBUG] dry-run passed: {output}", file=sys.stderr)
        except subprocess.TimeoutExpired:
            errors.append('rick human-loop --dry-run timed out')
        except Exception as e:
            errors.append(f'Failed to run rick human-loop --dry-run: {str(e)}')

        # Test 7: no args returns "topic is required" error
        try:
            result = subprocess.run(
                [rick_binary, 'human-loop'],
                capture_output=True,
                text=True,
                timeout=30
            )
            combined = result.stdout + result.stderr
            if result.returncode == 0:
                errors.append('rick human-loop with no args should fail but exited 0')
            elif 'topic is required' not in combined:
                errors.append(f'Expected "topic is required" error, got: {combined.strip()}')
            else:
                print(f"[DEBUG] no-args test passed", file=sys.stderr)
        except subprocess.TimeoutExpired:
            errors.append('rick human-loop (no args) timed out')
        except Exception as e:
            errors.append(f'Failed to run rick human-loop (no args): {str(e)}')

        # Test 8: mock binary test - RFC dir created and flow completes
        # Config is loaded from $HOME/.rick/config.json, so we set HOME to tmpdir
        try:
            with tempfile.TemporaryDirectory() as tmpdir:
                # Create mock claude script (exit 0)
                mock_claude = os.path.join(tmpdir, 'mock_claude')
                with open(mock_claude, 'w') as f:
                    f.write('#!/bin/sh\nexit 0\n')
                os.chmod(mock_claude, stat.S_IRWXU | stat.S_IRGRP | stat.S_IXGRP | stat.S_IROTH | stat.S_IXOTH)

                # Create fake HOME with .rick/config.json pointing to mock claude
                fake_home = os.path.join(tmpdir, 'home')
                rick_dir = os.path.join(fake_home, '.rick')
                os.makedirs(rick_dir, exist_ok=True)
                cfg_content = json.dumps({'claude_code_path': mock_claude})
                with open(os.path.join(rick_dir, 'config.json'), 'w') as f:
                    f.write(cfg_content)

                # Use fake_home as work dir so .rick/RFC is created under it
                work_dir = fake_home

                env = os.environ.copy()
                env['HOME'] = fake_home

                result = subprocess.run(
                    [rick_binary, 'human-loop', '如何重构?'],
                    cwd=work_dir,
                    capture_output=True,
                    text=True,
                    timeout=30,
                    stdin=subprocess.DEVNULL,
                    env=env
                )
                combined = result.stdout + result.stderr

                # Check RFC dir was auto-created under the work dir
                rfc_dir = os.path.join(rick_dir, 'RFC')
                if not os.path.isdir(rfc_dir):
                    errors.append(f'.rick/RFC/ directory was not auto-created during human-loop execution (checked {rfc_dir})')

                if result.returncode != 0:
                    errors.append(f'human-loop with mock claude failed (exit {result.returncode}): {combined.strip()}')
                elif '思考记录已保存到 .rick/RFC/' not in combined:
                    errors.append(f'human-loop completion message missing, got: {combined.strip()}')
                else:
                    print(f"[DEBUG] mock binary test passed", file=sys.stderr)
        except subprocess.TimeoutExpired:
            errors.append('rick human-loop mock binary test timed out')
        except Exception as e:
            errors.append(f'Failed to run mock binary test: {str(e)}')

    result_obj = {
        'pass': len(errors) == 0,
        'errors': errors
    }

    print(json.dumps(result_obj))
    sys.exit(0 if result_obj['pass'] else 1)

if __name__ == '__main__':
    main()
