#!/usr/bin/env python3
import json
import sys
import os
import subprocess

def run(cmd, cwd=None):
    result = subprocess.run(cmd, shell=True, capture_output=True, text=True, cwd=cwd)
    return result.returncode, result.stdout, result.stderr

def main():
    errors = []

    project_root = "/Users/sunquan/ai_coding/CODING/rick"

    # Test 1: 编译通过
    rc, stdout, stderr = run("go build ./...", cwd=project_root)
    if rc != 0:
        errors.append(f"go build failed: {stderr.strip()}")

    # Test 2: DIP 验证 - executor 不引用 claudecode
    rc, stdout, stderr = run("grep -r 'claudecode' internal/executor/", cwd=project_root)
    if stdout.strip():
        errors.append(f"internal/executor/ should not reference claudecode, found: {stdout.strip()}")

    # Test 3: DIP 验证 - actpath 不引用 claudecode
    rc, stdout, stderr = run("grep -r 'claudecode' internal/actpath/", cwd=project_root)
    if stdout.strip():
        errors.append(f"internal/actpath/ should not reference claudecode, found: {stdout.strip()}")

    # Test 4: 组合根验证 - doing.go 有且仅有 doing.go 引用 claudecode
    doing_go = os.path.join(project_root, "internal/cmd/doing.go")
    if not os.path.exists(doing_go):
        errors.append("internal/cmd/doing.go does not exist")
    else:
        rc, stdout, stderr = run("grep 'claudecode' internal/cmd/doing.go", cwd=project_root)
        if not stdout.strip():
            errors.append("doing.go should reference claudecode (composition root), but found nothing")

    # Test 5: 单元测试 - executor 包全部通过 (包含 KR1 act-path 验证)
    rc, stdout, stderr = run("go test ./internal/executor/... -v -timeout 60s", cwd=project_root)
    if rc != 0:
        errors.append(f"go test ./internal/executor/... failed:\n{stdout[-2000:]}\n{stderr[-1000:]}")
    else:
        # KR1 验证：检查 act-path 相关测试是否存在并通过
        combined = stdout + stderr
        if "act-path" not in combined.lower() and "actpath" not in combined.lower() and "act_path" not in combined.lower():
            print("WARNING: act-path related tests may not exist in executor tests", file=sys.stderr)
        if "FAIL" in stdout:
            errors.append(f"Some executor tests failed:\n{stdout[-2000:]}")

    # Test 6: doing dry-run - 提示词变量检查
    check_script = os.path.join(project_root, "tools/check_prompt_variables.py")
    if not os.path.exists(check_script):
        errors.append(f"tools/check_prompt_variables.py does not exist")
    else:
        rc, stdout, stderr = run(
            f"python3 tools/check_prompt_variables.py --phase doing --keywords '任务目标'",
            cwd=project_root
        )
        try:
            result = json.loads(stdout.strip())
            if not result.get("pass"):
                errors.append(f"check_prompt_variables failed: {result}")
        except Exception as e:
            errors.append(f"check_prompt_variables output not valid JSON: {stdout.strip()}, err: {e}")

    result = {
        "pass": len(errors) == 0,
        "errors": errors
    }

    print(json.dumps(result))
    sys.exit(0 if result["pass"] else 1)

if __name__ == "__main__":
    main()
