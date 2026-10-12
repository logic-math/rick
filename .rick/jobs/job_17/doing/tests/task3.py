#!/usr/bin/env python3
# Description: task3 验收测试 - 重构 easy.go 消除内部重复，复用 callClaudeCodeCLI
import json
import os
import subprocess
import sys

# 6 dirname calls from this file to reach project root
_FILE = os.path.abspath(__file__)
PROJECT_ROOT = os.path.dirname(os.path.dirname(os.path.dirname(
    os.path.dirname(os.path.dirname(os.path.dirname(_FILE))))))

EASY_GO = os.path.join(PROJECT_ROOT, "internal", "cmd", "easy.go")
PLAN_GO = os.path.join(PROJECT_ROOT, "internal", "cmd", "plan.go")
INTERNAL_CMD_DIR = os.path.join(PROJECT_ROOT, "internal", "cmd")


def _grep(pattern, filepath):
    """Return list of matching lines (empty list if none)."""
    try:
        result = subprocess.run(
            ["grep", "-n", pattern, filepath],
            capture_output=True, text=True
        )
        lines = [l for l in result.stdout.splitlines() if l.strip()]
        return lines
    except Exception as e:
        print(f"grep error: {e}", file=sys.stderr)
        return []


def _grep_dir(pattern, dirpath, file_pattern="*.go"):
    """grep -rn pattern in dirpath matching file_pattern."""
    try:
        result = subprocess.run(
            ["grep", "-rn", "--include", file_pattern, pattern, dirpath],
            capture_output=True, text=True
        )
        lines = [l for l in result.stdout.splitlines() if l.strip()]
        return lines
    except Exception as e:
        print(f"grep_dir error: {e}", file=sys.stderr)
        return []


def main():
    errors = []

    print(f"[DEBUG] PROJECT_ROOT={PROJECT_ROOT}", file=sys.stderr)

    # ──────────────────────────────────────────────
    # Test 1: 重复函数已删除
    # ──────────────────────────────────────────────
    matches = _grep(
        r"func callClaudeCodeCLIEasy\|func callClaudeCodeCLIResume",
        EASY_GO
    )
    if matches:
        errors.append(
            f"Test1 FAIL: duplicate functions still exist in easy.go: {matches}"
        )
    else:
        print("[DEBUG] Test1 PASS: no duplicate functions found", file=sys.stderr)

    # ──────────────────────────────────────────────
    # Test 2: 参数顺序校验（promptFile 在 cfg 后，sessionID 为 extraArgs）
    # callClaudeCodeCLI(cfg, <promptFile>, ...) 而非 callClaudeCodeCLI(cfg, sessionID, ...)
    # ──────────────────────────────────────────────
    call_lines = _grep(r"callClaudeCodeCLI(cfg", EASY_GO)
    bad_order_lines = []
    for line in call_lines:
        # 检测 callClaudeCodeCLI(cfg, sessionID, ...) 形式 — sessionID 紧跟 cfg 作为第 2 参数
        # 合法形式: callClaudeCodeCLI(cfg, mainFile, ...) / callClaudeCodeCLI(cfg, "", ...)
        # 非法形式: callClaudeCodeCLI(cfg, sessionID, ...)
        # sessionID 变量名包含 "session" 但不含 File/file/Path/path/"" 形式
        import re
        m = re.search(r'callClaudeCodeCLI\(cfg,\s*([^,)]+)', line)
        if m:
            second_arg = m.group(1).strip()
            # second_arg 不应是 sessionID（含 session 但不是 mainFile/promptFile/"" 等）
            if "session" in second_arg.lower() and "file" not in second_arg.lower() and second_arg != '""':
                bad_order_lines.append(line)
    if bad_order_lines:
        errors.append(
            f"Test2 FAIL: callClaudeCodeCLI in easy.go has sessionID as 2nd arg: {bad_order_lines}"
        )
    else:
        print("[DEBUG] Test2 PASS: parameter order looks correct", file=sys.stderr)

    # ──────────────────────────────────────────────
    # Test 3: callClaudeCodeCLI 已支持 extraArgs（plan.go 中至少 2 行匹配）
    # ──────────────────────────────────────────────
    extra_lines = _grep("extraArgs", PLAN_GO)
    if len(extra_lines) < 2:
        errors.append(
            f"Test3 FAIL: 'extraArgs' found only {len(extra_lines)} time(s) in plan.go (need >= 2): {extra_lines}"
        )
    else:
        print(f"[DEBUG] Test3 PASS: extraArgs found {len(extra_lines)} times in plan.go", file=sys.stderr)

    # ──────────────────────────────────────────────
    # Test 4A: go build ./... 通过
    # ──────────────────────────────────────────────
    try:
        result = subprocess.run(
            ["go", "build", "./..."],
            capture_output=True, text=True, cwd=PROJECT_ROOT
        )
        if result.returncode != 0:
            errors.append(
                f"Test4A FAIL: go build ./... failed (exit {result.returncode}):\n{result.stderr}"
            )
        else:
            print("[DEBUG] Test4A PASS: go build ./... succeeded", file=sys.stderr)
    except Exception as e:
        errors.append(f"Test4A ERROR: go build failed with exception: {e}")

    # ──────────────────────────────────────────────
    # Test 4B: TestCallClaudeCodeCLI_MockBinary 存在
    # ──────────────────────────────────────────────
    mock_test_lines = _grep_dir("TestCallClaudeCodeCLI_MockBinary", INTERNAL_CMD_DIR)
    if not mock_test_lines:
        errors.append(
            "Test4B FAIL: TestCallClaudeCodeCLI_MockBinary not found in internal/cmd/"
        )
    else:
        print(f"[DEBUG] Test4B PASS: TestCallClaudeCodeCLI_MockBinary found: {mock_test_lines}", file=sys.stderr)

    # ──────────────────────────────────────────────
    # Test 4C: go test ./internal/cmd/... -run TestCallClaudeCodeCLI_MockBinary -v 两个子用例均 PASS
    # ──────────────────────────────────────────────
    try:
        result = subprocess.run(
            ["go", "test", "./internal/cmd/...", "-run", "TestCallClaudeCodeCLI_MockBinary", "-v"],
            capture_output=True, text=True, cwd=PROJECT_ROOT,
            timeout=60
        )
        output = result.stdout + result.stderr
        if result.returncode != 0:
            errors.append(
                f"Test4C FAIL: TestCallClaudeCodeCLI_MockBinary tests failed (exit {result.returncode}):\n{output}"
            )
        else:
            # 验证两个子用例都 PASS
            has_prompt_nonempty = "promptFile_nonempty" in output and "PASS" in output
            has_prompt_empty = "promptFile_empty" in output and "PASS" in output
            if not has_prompt_nonempty:
                errors.append(
                    "Test4C FAIL: subtest 'promptFile_nonempty' not found or not PASS in output"
                )
            if not has_prompt_empty:
                errors.append(
                    "Test4C FAIL: subtest 'promptFile_empty' not found or not PASS in output"
                )
            if has_prompt_nonempty and has_prompt_empty:
                print("[DEBUG] Test4C PASS: both MockBinary subtests PASS", file=sys.stderr)
    except subprocess.TimeoutExpired:
        errors.append("Test4C ERROR: go test timed out after 60s")
    except Exception as e:
        errors.append(f"Test4C ERROR: {e}")

    # ──────────────────────────────────────────────
    # Tests 5 & 6: build rick binary then test CLI flags
    # ──────────────────────────────────────────────
    build_tool = os.path.join(PROJECT_ROOT, ".rick", "tools", "build_and_get_rick_bin.py")
    bin_path = None
    try:
        result = subprocess.run(
            ["python3", build_tool],
            capture_output=True, text=True, cwd=PROJECT_ROOT, timeout=120
        )
        if result.returncode != 0:
            errors.append(
                f"Test5+6 SETUP FAIL: build_and_get_rick_bin.py failed:\n{result.stderr}"
            )
        else:
            output = result.stdout.strip()
            # parse JSON output from build tool
            try:
                build_result = json.loads(output)
                bin_path = build_result.get("bin_path", "")
            except Exception:
                # fallback: last non-empty line
                bin_path = [l for l in output.splitlines() if l.strip()][-1] if output else ""
            if not bin_path or not os.path.exists(bin_path):
                errors.append(f"Test5+6 SETUP FAIL: invalid bin_path={bin_path!r}")
                bin_path = None
            else:
                print(f"[DEBUG] bin_path={bin_path}", file=sys.stderr)
    except subprocess.TimeoutExpired:
        errors.append("Test5+6 SETUP ERROR: build timed out")
    except Exception as e:
        errors.append(f"Test5+6 SETUP ERROR: {e}")

    if bin_path:
        # Test 5: rick doing --easy flag 保留
        try:
            result = subprocess.run(
                [bin_path, "doing", "--help"],
                capture_output=True, text=True, timeout=10
            )
            help_output = result.stdout + result.stderr
            missing_flags = []
            if "--easy" not in help_output:
                missing_flags.append("--easy")
            if "--ctx" not in help_output:
                missing_flags.append("--ctx")
            if missing_flags:
                errors.append(
                    f"Test5 FAIL: 'rick doing --help' missing flags: {missing_flags}"
                )
            else:
                print("[DEBUG] Test5 PASS: --easy and --ctx flags present", file=sys.stderr)
        except Exception as e:
            errors.append(f"Test5 ERROR: {e}")

        # Test 6: rick doing --dry-run 正常路径不受影响
        job1_plan = os.path.join(PROJECT_ROOT, ".rick", "jobs", "job_1", "plan")
        if not os.path.isdir(job1_plan):
            print("[DEBUG] Test6 SKIP: job_1/plan dir not found, skipping dry-run test", file=sys.stderr)
        else:
            try:
                result = subprocess.run(
                    [bin_path, "doing", "--dry-run", "--job", "job_1"],
                    capture_output=True, text=True, timeout=15
                )
                combined = result.stdout + result.stderr
                if result.returncode != 0:
                    errors.append(
                        f"Test6 FAIL: rick doing --dry-run --job job_1 exited {result.returncode}:\n{combined}"
                    )
                elif "[DRY-RUN]" not in combined:
                    errors.append(
                        "Test6 FAIL: dry-run output missing '[DRY-RUN]' marker"
                    )
                elif "panic" in combined.lower():
                    errors.append(
                        "Test6 FAIL: dry-run output contains 'panic'"
                    )
                else:
                    print("[DEBUG] Test6 PASS: dry-run exits 0 with [DRY-RUN] marker", file=sys.stderr)
            except Exception as e:
                errors.append(f"Test6 ERROR: {e}")

    result = {
        "pass": len(errors) == 0,
        "errors": errors
    }
    print(json.dumps(result, ensure_ascii=False))
    sys.exit(0 if result["pass"] else 1)


if __name__ == "__main__":
    main()
