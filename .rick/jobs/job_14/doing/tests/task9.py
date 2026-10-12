#!/usr/bin/env python3
import json
import os
import subprocess
import sys

PROJECT_ROOT = os.path.dirname(os.path.dirname(os.path.dirname(os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__)))))))
TOOLS_DIR = os.path.join(PROJECT_ROOT, "tools")
TESTS_DIR = os.path.join(PROJECT_ROOT, "tests")
MOCK_AGENT = os.path.join(TESTS_DIR, "mock_agent", "mock_agent.py")


def main():
    errors = []

    # Step 1: Build rick binary
    try:
        sys.path.insert(0, TOOLS_DIR)
        from build_and_get_rick_bin import build_and_get_rick_bin
        bin_path, err = build_and_get_rick_bin(PROJECT_ROOT)
        if err:
            errors.append(f"Build failed: {err}")
        else:
            print(f"[task9] rick binary: {bin_path}", file=sys.stderr)
    except Exception as e:
        errors.append(f"Failed to build rick: {str(e)}")

    # Step 2: doing_v2_success scenario outputs valid stream-json NDJSON (>=4 lines)
    try:
        env = os.environ.copy()
        env["MOCK_SCENARIO"] = "doing_v2_success"
        result = subprocess.run(
            ["python3", MOCK_AGENT, "--output-format", "stream-json", "/dev/null"],
            capture_output=True, text=True, env=env, timeout=30
        )
        lines = [l for l in result.stdout.strip().splitlines() if l.strip()]
        if len(lines) < 4:
            errors.append(f"doing_v2_success: expected >=4 NDJSON lines, got {len(lines)}: {result.stdout[:200]}")
        else:
            # Validate each line is valid JSON
            for i, line in enumerate(lines):
                try:
                    json.loads(line)
                except json.JSONDecodeError as je:
                    errors.append(f"doing_v2_success: line {i+1} is not valid JSON: {line[:80]}: {je}")

            # Last line must contain session_id
            last = json.loads(lines[-1])
            if "session_id" not in last:
                errors.append(f"doing_v2_success: last line missing session_id: {lines[-1][:80]}")

            # tool_use must be nested in message.content[], not top-level
            tool_use_found = False
            for line in lines:
                obj = json.loads(line)
                if obj.get("type") == "assistant":
                    msg = obj.get("message", {})
                    content = msg.get("content", [])
                    for item in content:
                        if item.get("type") == "tool_use":
                            tool_use_found = True
                            break
                # Reject top-level tool_use
                if obj.get("type") == "tool_use":
                    errors.append("doing_v2_success: tool_use found at top level, must be in message.content[]")
            if not tool_use_found:
                errors.append("doing_v2_success: no tool_use found nested in message.content[]")

    except subprocess.TimeoutExpired:
        errors.append("doing_v2_success: timed out after 30s")
    except Exception as e:
        errors.append(f"doing_v2_success scenario check failed: {str(e)}")

    # Step 3: mock --self-test passes (no regression in existing scenarios)
    try:
        result = subprocess.run(
            ["python3", MOCK_AGENT, "--self-test"],
            capture_output=True, text=True, timeout=60
        )
        if result.returncode != 0:
            errors.append(f"mock --self-test failed (exit {result.returncode}): {result.stderr[-300:]}")
    except subprocess.TimeoutExpired:
        errors.append("mock --self-test timed out after 60s")
    except Exception as e:
        errors.append(f"mock --self-test check failed: {str(e)}")

    # Step 4: e2e_v2_test.py exists and all 4 phases pass
    e2e_script = os.path.join(TESTS_DIR, "e2e_v2_test.py")
    if not os.path.exists(e2e_script):
        errors.append(f"e2e_v2_test.py not found at {e2e_script}")
    else:
        try:
            result = subprocess.run(
                ["python3", e2e_script],
                capture_output=True, text=True, timeout=120, cwd=PROJECT_ROOT
            )
            combined = result.stdout + result.stderr
            if "E2E v2 all phases passed" not in combined:
                errors.append(
                    f"e2e_v2_test.py did not output '✅ E2E v2 all phases passed' "
                    f"(exit {result.returncode}): {combined[-400:]}"
                )
        except subprocess.TimeoutExpired:
            errors.append("e2e_v2_test.py timed out after 120s")
        except Exception as e:
            errors.append(f"e2e_v2_test.py execution failed: {str(e)}")

    # Step 5: go test ./... passes with no new failures
    try:
        result = subprocess.run(
            ["go", "test", "./..."],
            capture_output=True, text=True, timeout=120, cwd=PROJECT_ROOT
        )
        if result.returncode != 0:
            errors.append(f"go test ./... failed (exit {result.returncode}): {result.stdout[-300:]}{result.stderr[-300:]}")
    except subprocess.TimeoutExpired:
        errors.append("go test ./... timed out after 120s")
    except Exception as e:
        errors.append(f"go test ./... check failed: {str(e)}")

    # Step 6: mock_agent_testing.py passes
    mock_testing = os.path.join(TOOLS_DIR, "mock_agent_testing.py")
    if not os.path.exists(mock_testing):
        errors.append(f"mock_agent_testing.py not found at {mock_testing}")
    else:
        try:
            result = subprocess.run(
                ["python3", mock_testing],
                capture_output=True, text=True, timeout=60, cwd=PROJECT_ROOT
            )
            if result.returncode != 0:
                errors.append(
                    f"mock_agent_testing.py failed (exit {result.returncode}): "
                    f"{result.stdout[-300:]}{result.stderr[-300:]}"
                )
        except subprocess.TimeoutExpired:
            errors.append("mock_agent_testing.py timed out after 60s")
        except Exception as e:
            errors.append(f"mock_agent_testing.py check failed: {str(e)}")

    result_obj = {"pass": len(errors) == 0, "errors": errors}
    print(json.dumps(result_obj))
    sys.exit(0 if result_obj["pass"] else 1)


if __name__ == "__main__":
    main()
