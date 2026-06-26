#!/usr/bin/env python3
# Description: 验收 task4 — build + dry-run 输出检查 + 回归测试 + plan_check
import json
import sys
import os
import subprocess

def get_project_root():
    script_dir = os.path.dirname(os.path.abspath(__file__))
    # tests/ → doing/ → job_18/ → jobs/ → .rick/ → project_root
    return os.path.dirname(os.path.dirname(os.path.dirname(
        os.path.dirname(os.path.dirname(script_dir)))))

def main():
    errors = []
    project_root = get_project_root()
    print(f"project_root: {project_root}", file=sys.stderr)

    build_tool = os.path.join(project_root, ".rick", "tools", "build_and_get_rick_bin.py")

    # ── Test 1: 构建成功 ───────────────────────────────────────────────────────
    rick_bin = None
    try:
        result = subprocess.run(
            ["python3", build_tool],
            capture_output=True, text=True, cwd=project_root
        )
        build_output = json.loads(result.stdout)
        if not build_output.get("pass"):
            errors.append(f"构建失败: {build_output.get('errors', [])}")
        else:
            rick_bin = build_output.get("bin_path", os.path.join(project_root, "bin", "rick"))
            print(f"构建成功: {rick_bin}", file=sys.stderr)
    except Exception as e:
        errors.append(f"调用 build_and_get_rick_bin.py 失败: {e}")

    # ── Test 2: plan --dry-run 输出检查 ────────────────────────────────────────
    if rick_bin:
        try:
            result = subprocess.run(
                [rick_bin, "plan", "--dry-run"],
                capture_output=True, text=True, cwd=project_root
            )
            output = result.stdout + result.stderr
            print(f"plan --dry-run exit={result.returncode}", file=sys.stderr)

            if "skill_grilling" not in output:
                errors.append("plan --dry-run 输出不含 skill_grilling")
            if "sense_skill_path" in output:
                errors.append("plan --dry-run 输出仍含 sense_skill_path（应已删除）")
            if "{{grilling_skill_path}}" in output:
                errors.append("plan --dry-run 输出含未替换的 {{grilling_skill_path}} 占位符")
        except Exception as e:
            errors.append(f"执行 plan --dry-run 失败: {e}")

    # ── Test 3: go test ./internal/prompt/... ──────────────────────────────────
    try:
        result = subprocess.run(
            ["go", "test", "./internal/prompt/...", "-run", "."],
            capture_output=True, text=True, cwd=project_root, timeout=120
        )
        combined = result.stdout + result.stderr
        print(f"go test ./internal/prompt/... exit={result.returncode}", file=sys.stderr)
        print(combined[:500], file=sys.stderr)
        if result.returncode != 0 or "FAIL" in combined:
            # Extract FAIL lines for clarity
            fail_lines = [l for l in combined.splitlines() if "FAIL" in l or "--- FAIL" in l]
            errors.append(f"go test ./internal/prompt/... 有失败: {fail_lines[:5]}")
    except subprocess.TimeoutExpired:
        errors.append("go test ./internal/prompt/... 超时")
    except Exception as e:
        errors.append(f"执行 go test ./internal/prompt/... 失败: {e}")

    # ── Test 4: go test ./internal/executor/... ───────────────────────────────
    # task1-3 只改动了 internal/prompt/；internal/cmd/ 有与 grilling 无关的预存失败测试
    # 按 SPEC "go test 范围精确性" 只验证实际改动包 + executor（间接依赖）
    try:
        result = subprocess.run(
            ["go", "test", "./internal/executor/...", "-run", "."],
            capture_output=True, text=True, cwd=project_root, timeout=120
        )
        combined = result.stdout + result.stderr
        print(f"go test ./internal/executor/... exit={result.returncode}", file=sys.stderr)
        print(combined[:500], file=sys.stderr)
        if result.returncode != 0 or "FAIL" in combined:
            fail_lines = [l for l in combined.splitlines() if "FAIL" in l or "--- FAIL" in l]
            errors.append(f"go test ./internal/executor/... 有失败: {fail_lines[:5]}")
    except subprocess.TimeoutExpired:
        errors.append("go test ./internal/executor/... 超时")
    except Exception as e:
        errors.append(f"执行 go test ./internal/executor/... 失败: {e}")

    # ── Test 5: rick tools plan_check job_18 ─────────────────────────────────
    if rick_bin:
        try:
            result = subprocess.run(
                [rick_bin, "tools", "plan_check", "job_18"],
                capture_output=True, text=True, cwd=project_root, timeout=30
            )
            combined = result.stdout + result.stderr
            print(f"plan_check job_18 exit={result.returncode}", file=sys.stderr)
            print(combined[:300], file=sys.stderr)
            if result.returncode != 0:
                errors.append(f"plan_check job_18 失败（exit={result.returncode}）: {combined[:200]}")
        except subprocess.TimeoutExpired:
            errors.append("plan_check job_18 超时")
        except Exception as e:
            errors.append(f"执行 plan_check job_18 失败: {e}")

    result = {
        "pass": len(errors) == 0,
        "errors": errors
    }
    print(json.dumps(result, ensure_ascii=False))
    sys.exit(0 if result["pass"] else 1)

if __name__ == "__main__":
    main()
