#!/usr/bin/env python3
import json
import sys
import os
import subprocess
import shutil

def run_cmd(cmd, cwd=None):
    result = subprocess.run(cmd, shell=True, capture_output=True, text=True, cwd=cwd)
    return result.returncode, result.stdout, result.stderr

def main():
    errors = []

    project_root = "/Users/sunquan/ai_coding/CODING/rick"
    binary = os.path.join(project_root, "bin", "rick")

    # Test 1: Build the binary
    print("Building binary...", file=sys.stderr)
    rc, out, err = run_cmd("go build -o bin/rick ./cmd/rick/", cwd=project_root)
    if rc != 0:
        errors.append(f"go build failed: {err}")
        print(json.dumps({"pass": False, "errors": errors}))
        sys.exit(1)

    # Test 2: run plan_check on existing job_1
    print("Testing plan_check on job_1...", file=sys.stderr)
    rc, out, err = run_cmd(f"{binary} tools plan_check job_1", cwd=project_root)
    combined = out + err
    if "plan check passed" not in combined and "✅" not in combined:
        errors.append(f"plan_check job_1 did not output success message. stdout={out!r} stderr={err!r}")

    # Test 3: missing section detection
    print("Testing missing section detection...", file=sys.stderr)
    test_job_dir = "/tmp/test_job_task3"
    plan_dir = os.path.join(test_job_dir, "plan", "tasks")
    os.makedirs(plan_dir, exist_ok=True)
    # Write a task.md missing '# 关键结果'
    bad_task_content = """# 依赖关系


# 任务名称
测试任务

# 任务目标
这是一个测试任务

# 测试方法
1. 运行测试
"""
    with open(os.path.join(plan_dir, "task1.md"), "w") as f:
        f.write(bad_task_content)

    rc, out, err = run_cmd(f"{binary} tools plan_check", cwd=test_job_dir)
    combined = out + err
    if "关键结果" not in combined and "error" not in combined.lower() and rc == 0:
        errors.append(f"plan_check did not detect missing '# 关键结果' section. stdout={out!r} stderr={err!r}")

    # Test 4: circular dependency detection
    print("Testing circular dependency detection...", file=sys.stderr)
    circ_job_dir = "/tmp/test_job_circ"
    circ_plan_dir = os.path.join(circ_job_dir, "plan", "tasks")
    os.makedirs(circ_plan_dir, exist_ok=True)

    task1_content = """# 依赖关系
task2

# 任务名称
任务1

# 任务目标
目标1

# 关键结果
1. 结果1

# 测试方法
1. 测试1
"""
    task2_content = """# 依赖关系
task1

# 任务名称
任务2

# 任务目标
目标2

# 关键结果
1. 结果2

# 测试方法
1. 测试2
"""
    with open(os.path.join(circ_plan_dir, "task1.md"), "w") as f:
        f.write(task1_content)
    with open(os.path.join(circ_plan_dir, "task2.md"), "w") as f:
        f.write(task2_content)

    rc, out, err = run_cmd(f"{binary} tools plan_check", cwd=circ_job_dir)
    combined = out + err
    if "cycl" not in combined.lower() and "circular" not in combined.lower() and "循环" not in combined and rc == 0:
        errors.append(f"plan_check did not detect circular dependency. stdout={out!r} stderr={err!r}")

    # Test 5: dangling dependency detection
    print("Testing dangling dependency detection...", file=sys.stderr)
    dangle_job_dir = "/tmp/test_job_dangle"
    dangle_plan_dir = os.path.join(dangle_job_dir, "plan", "tasks")
    os.makedirs(dangle_plan_dir, exist_ok=True)

    dangle_task_content = """# 依赖关系
task99

# 任务名称
任务1

# 任务目标
目标1

# 关键结果
1. 结果1

# 测试方法
1. 测试1
"""
    with open(os.path.join(dangle_plan_dir, "task1.md"), "w") as f:
        f.write(dangle_task_content)

    rc, out, err = run_cmd(f"{binary} tools plan_check", cwd=dangle_job_dir)
    combined = out + err
    if "task99" not in combined and "dangling" not in combined.lower() and "不存在" not in combined and "missing" not in combined.lower() and rc == 0:
        errors.append(f"plan_check did not detect dangling dependency on task99. stdout={out!r} stderr={err!r}")

    # Test 6: rick tools --help shows plan_check
    print("Testing rick tools --help...", file=sys.stderr)
    rc, out, err = run_cmd(f"{binary} tools --help", cwd=project_root)
    combined = out + err
    if "plan_check" not in combined and "plan-check" not in combined:
        errors.append(f"'rick tools --help' does not show plan_check subcommand. stdout={out!r} stderr={err!r}")

    # Test 7: go test
    print("Running go tests...", file=sys.stderr)
    rc, out, err = run_cmd("go test ./internal/cmd/... ./internal/parser/...", cwd=project_root)
    if rc != 0:
        errors.append(f"go test failed: {err}\n{out}")

    # Cleanup
    for d in [test_job_dir, circ_job_dir, dangle_job_dir]:
        if os.path.exists(d):
            shutil.rmtree(d)

    result = {
        "pass": len(errors) == 0,
        "errors": errors
    }

    print(json.dumps(result))
    sys.exit(0 if result["pass"] else 1)

if __name__ == "__main__":
    main()
