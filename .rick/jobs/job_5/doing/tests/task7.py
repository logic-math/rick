#!/usr/bin/env python3
import json
import sys
import os
import subprocess

def run_cmd(cmd, cwd=None):
    """Run a command and return (returncode, stdout, stderr)."""
    result = subprocess.run(
        cmd, shell=True, capture_output=True, text=True, cwd=cwd
    )
    return result.returncode, result.stdout, result.stderr

def main():
    errors = []

    # Project root: tests/ -> doing/ -> job_5/ -> jobs/ -> .rick/ -> project root
    tests_dir = os.path.dirname(os.path.abspath(__file__))
    doing_dir = os.path.dirname(tests_dir)
    job_dir = os.path.dirname(doing_dir)
    jobs_dir = os.path.dirname(job_dir)
    rick_dir = os.path.dirname(jobs_dir)
    project_root = os.path.dirname(rick_dir)

    print(f"Project root: {project_root}", file=sys.stderr)

    # Test 1: go build
    print("Test 1: go build -o bin/rick ./cmd/rick/", file=sys.stderr)
    rc, stdout, stderr = run_cmd("go build -o bin/rick ./cmd/rick/", cwd=project_root)
    if rc != 0:
        errors.append(f"go build failed (exit {rc}): {stderr.strip()}")

    # Test 2: go test ./...
    print("Test 2: go test ./...", file=sys.stderr)
    rc, stdout, stderr = run_cmd("go test ./...", cwd=project_root)
    if rc != 0:
        errors.append(f"go test ./... failed (exit {rc}): {stderr.strip()}")

    # Test 3: mock_agent --self-test
    mock_agent = os.path.join(project_root, "tests", "mock_agent", "mock_agent.py")
    print(f"Test 3: python3 {mock_agent} --self-test", file=sys.stderr)
    if not os.path.exists(mock_agent):
        errors.append(f"mock_agent.py does not exist: {mock_agent}")
    else:
        rc, stdout, stderr = run_cmd(f"python3 {mock_agent} --self-test", cwd=project_root)
        if rc != 0:
            errors.append(f"mock_agent --self-test failed (exit {rc}): {stderr.strip() or stdout.strip()}")

    # Test 4: bash tests/tools_integration_test.sh
    integration_test = os.path.join(project_root, "tests", "tools_integration_test.sh")
    print(f"Test 4: bash {integration_test}", file=sys.stderr)
    if not os.path.exists(integration_test):
        errors.append(f"tools_integration_test.sh does not exist: {integration_test}")
    else:
        rc, stdout, stderr = run_cmd(f"bash {integration_test}", cwd=project_root)
        if rc != 0:
            errors.append(f"tools_integration_test.sh failed (exit {rc}): {stderr.strip() or stdout.strip()}")

    # Test 5: go test -cover with coverage >= 70%
    print("Test 5: go test -cover ./internal/cmd/... ./internal/executor/... ./internal/prompt/...", file=sys.stderr)
    rc, stdout, stderr = run_cmd(
        "go test -cover ./internal/cmd/... ./internal/executor/... ./internal/prompt/...",
        cwd=project_root
    )
    if rc != 0:
        errors.append(f"go test -cover failed (exit {rc}): {stderr.strip()}")
    else:
        # Check coverage lines for any package below 70%
        low_coverage = []
        for line in stdout.splitlines():
            if "coverage:" in line:
                # e.g. "ok  	github.com/xxx	0.123s	coverage: 45.2% of statements"
                try:
                    pct_str = line.split("coverage:")[1].strip().split("%")[0].strip()
                    pct = float(pct_str)
                    if pct < 70.0:
                        low_coverage.append(f"{line.strip()} (< 70%)")
                except Exception:
                    pass
        if low_coverage:
            errors.append("Coverage below 70%: " + "; ".join(low_coverage))

    # Test 6: ./bin/rick tools --help
    rick_bin = os.path.join(project_root, "bin", "rick")
    print(f"Test 6: {rick_bin} tools --help", file=sys.stderr)
    if not os.path.exists(rick_bin):
        errors.append(f"bin/rick does not exist: {rick_bin}")
    else:
        rc, stdout, stderr = run_cmd(f"{rick_bin} tools --help", cwd=project_root)
        if rc != 0:
            errors.append(f"rick tools --help failed (exit {rc}): {stderr.strip()}")
        else:
            combined = stdout + stderr
            # Should show subcommand list
            if "Available Commands" not in combined and "available commands" not in combined.lower():
                errors.append("rick tools --help does not show subcommand list")

    # Test 7: ./bin/rick --help shows tools command
    print(f"Test 7: {rick_bin} --help", file=sys.stderr)
    if os.path.exists(rick_bin):
        rc, stdout, stderr = run_cmd(f"{rick_bin} --help", cwd=project_root)
        combined = stdout + stderr
        if "tools" not in combined:
            errors.append("rick --help does not show 'tools' command")

    result = {
        "pass": len(errors) == 0,
        "errors": errors
    }

    print(json.dumps(result))
    sys.exit(0 if result["pass"] else 1)

if __name__ == "__main__":
    main()
