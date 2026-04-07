#!/usr/bin/env python3
"""
mock_agent.py - Mock AI agent for testing Rick workflows without Claude CLI.

Usage:
    MOCK_SCENARIO=<scenario> python3 mock_agent.py <prompt_file>
    python3 mock_agent.py --self-test

Scenarios:
    plan_success            - Generates valid plan tasks
    plan_missing_section    - Generates plan task missing '# 关键结果' section
    plan_circular_dep       - Generates plan tasks with circular dependency
    doing_success           - Simulates successful task execution
    doing_no_debug          - Simulates doing without creating debug.md
    doing_zombie_task       - Simulates doing leaving a task in 'running' state
    learning_success        - Generates valid learning outputs
    learning_bad_skill      - Generates learning skill with Python syntax error
    learning_no_summary     - Generates learning outputs without SUMMARY.md
    claude_exit_nonzero     - Exits with non-zero exit code
    claude_bad_output       - Outputs garbled/invalid content
    claude_timeout          - Sleeps for a long time (simulate timeout)
"""

import json
import os
import sys
import time
import re
import subprocess
import tempfile
import shutil

# ─── Helpers ────────────────────────────────────────────────────────────────

def read_prompt_file(path):
    """Read and return the content of the prompt file."""
    with open(path, "r", encoding="utf-8") as f:
        return f.read()

def extract_job_dir_from_prompt(content):
    """Try to extract the job doing/learning/plan directory from prompt content."""
    # Look for paths like /path/to/.rick/jobs/job_N/...
    patterns = [
        r'(/[^\s]+/\.rick/jobs/job_\w+/doing)',
        r'(/[^\s]+/\.rick/jobs/job_\w+/learning)',
        r'(/[^\s]+/\.rick/jobs/job_\w+/plan)',
        r'(/[^\s]+/\.rick/jobs/job_\w+)',
    ]
    for pat in patterns:
        m = re.search(pat, content)
        if m:
            return m.group(1)
    return None

def find_doing_dir_from_env():
    """Find doing dir from RICK_DOING_DIR env or fallback to prompt scanning."""
    return os.environ.get("RICK_DOING_DIR", "")

def find_learning_dir_from_env():
    return os.environ.get("RICK_LEARNING_DIR", "")

def find_plan_dir_from_env():
    return os.environ.get("RICK_PLAN_DIR", "")

# ─── Scenario Handlers ───────────────────────────────────────────────────────

def scenario_plan_success(prompt_content, plan_dir):
    """Create a valid plan with 2 tasks."""
    if not plan_dir:
        print("[mock_agent] ERROR: RICK_PLAN_DIR not set", file=sys.stderr)
        sys.exit(1)
    os.makedirs(plan_dir, exist_ok=True)

    task1 = """# 依赖关系
（无）

# 任务名称
初始化项目结构

# 任务目标
创建基础目录结构和配置文件

# 关键结果
1. 创建 src/ 目录
2. 创建 config.json

# 测试方法
1. 检查 src/ 目录存在
2. 检查 config.json 存在
"""
    task2 = """# 依赖关系
task1

# 任务名称
实现核心功能

# 任务目标
实现主要业务逻辑

# 关键结果
1. 实现 main.go
2. 通过单元测试

# 测试方法
1. 运行 go test ./...
2. 验证编译成功
"""
    with open(os.path.join(plan_dir, "task1.md"), "w") as f:
        f.write(task1)
    with open(os.path.join(plan_dir, "task2.md"), "w") as f:
        f.write(task2)
    print("[mock_agent] plan_success: created task1.md and task2.md", file=sys.stderr)


def scenario_plan_missing_section(prompt_content, plan_dir):
    """Create a plan task missing '# 关键结果' section."""
    if not plan_dir:
        print("[mock_agent] ERROR: RICK_PLAN_DIR not set", file=sys.stderr)
        sys.exit(1)
    os.makedirs(plan_dir, exist_ok=True)

    task1 = """# 依赖关系
（无）

# 任务名称
初始化项目结构

# 任务目标
创建基础目录结构

# 测试方法
1. 检查目录存在
"""
    # Note: missing '# 关键结果' section
    with open(os.path.join(plan_dir, "task1.md"), "w") as f:
        f.write(task1)
    print("[mock_agent] plan_missing_section: created task1.md without 关键结果", file=sys.stderr)


def scenario_plan_circular_dep(prompt_content, plan_dir):
    """Create plan tasks with circular dependency: task1 -> task2 -> task1."""
    if not plan_dir:
        print("[mock_agent] ERROR: RICK_PLAN_DIR not set", file=sys.stderr)
        sys.exit(1)
    os.makedirs(plan_dir, exist_ok=True)

    task1 = """# 依赖关系
task2

# 任务名称
任务一

# 任务目标
第一个任务

# 关键结果
1. 完成任务一

# 测试方法
1. 验证任务一完成
"""
    task2 = """# 依赖关系
task1

# 任务名称
任务二

# 任务目标
第二个任务

# 关键结果
1. 完成任务二

# 测试方法
1. 验证任务二完成
"""
    with open(os.path.join(plan_dir, "task1.md"), "w") as f:
        f.write(task1)
    with open(os.path.join(plan_dir, "task2.md"), "w") as f:
        f.write(task2)
    print("[mock_agent] plan_circular_dep: created circular task1 <-> task2", file=sys.stderr)


def scenario_doing_success(prompt_content, doing_dir):
    """Simulate successful task execution: create tasks.json and debug.md."""
    if not doing_dir:
        print("[mock_agent] ERROR: RICK_DOING_DIR not set", file=sys.stderr)
        sys.exit(1)
    os.makedirs(doing_dir, exist_ok=True)

    now = "2026-01-01T00:00:00Z"
    tasks_json = {
        "version": "1.0",
        "created_at": now,
        "updated_at": now,
        "tasks": [
            {
                "task_id": "task1",
                "task_name": "初始化项目结构",
                "status": "success",
                "dependencies": [],
                "attempts": 1,
                "commit_hash": "abc1234",
                "created_at": now,
                "updated_at": now
            },
            {
                "task_id": "task2",
                "task_name": "实现核心功能",
                "status": "success",
                "dependencies": ["task1"],
                "attempts": 1,
                "commit_hash": "def5678",
                "created_at": now,
                "updated_at": now
            }
        ]
    }
    with open(os.path.join(doing_dir, "tasks.json"), "w") as f:
        json.dump(tasks_json, f, indent=2)

    debug_md = """## task1: 初始化项目结构

**分析过程 (Analysis)**:
- 分析了项目结构需求

**实现步骤 (Implementation)**:
1. 创建目录结构
2. 初始化配置文件

**遇到的问题 (Issues)**:
- 无

**验证结果 (Verification)**:
- 测试命令: `ls src/`
- 测试输出:
  ```
  src/
  ```
- 结论：✅ 通过

## task2: 实现核心功能

**分析过程 (Analysis)**:
- 分析了核心功能需求

**实现步骤 (Implementation)**:
1. 实现主要逻辑

**遇到的问题 (Issues)**:
- 无

**验证结果 (Verification)**:
- 测试命令: `go test ./...`
- 测试输出:
  ```
  PASS
  ```
- 结论：✅ 通过
"""
    with open(os.path.join(doing_dir, "debug.md"), "w") as f:
        f.write(debug_md)
    print("[mock_agent] doing_success: created tasks.json and debug.md", file=sys.stderr)


def scenario_doing_no_debug(prompt_content, doing_dir):
    """Simulate doing without debug.md (only tasks.json)."""
    if not doing_dir:
        print("[mock_agent] ERROR: RICK_DOING_DIR not set", file=sys.stderr)
        sys.exit(1)
    os.makedirs(doing_dir, exist_ok=True)

    now = "2026-01-01T00:00:00Z"
    tasks_json = {
        "version": "1.0",
        "created_at": now,
        "updated_at": now,
        "tasks": [
            {
                "task_id": "task1",
                "task_name": "初始化项目结构",
                "status": "success",
                "dependencies": [],
                "attempts": 1,
                "commit_hash": "abc1234",
                "created_at": now,
                "updated_at": now
            }
        ]
    }
    with open(os.path.join(doing_dir, "tasks.json"), "w") as f:
        json.dump(tasks_json, f, indent=2)
    # Intentionally NOT creating debug.md
    print("[mock_agent] doing_no_debug: created tasks.json but NO debug.md", file=sys.stderr)


def scenario_doing_zombie_task(prompt_content, doing_dir):
    """Simulate doing leaving a task in 'running' zombie state."""
    if not doing_dir:
        print("[mock_agent] ERROR: RICK_DOING_DIR not set", file=sys.stderr)
        sys.exit(1)
    os.makedirs(doing_dir, exist_ok=True)

    now = "2026-01-01T00:00:00Z"
    tasks_json = {
        "version": "1.0",
        "created_at": now,
        "updated_at": now,
        "tasks": [
            {
                "task_id": "task1",
                "task_name": "初始化项目结构",
                "status": "running",  # zombie state
                "dependencies": [],
                "attempts": 1,
                "created_at": now,
                "updated_at": now
            }
        ]
    }
    with open(os.path.join(doing_dir, "tasks.json"), "w") as f:
        json.dump(tasks_json, f, indent=2)

    with open(os.path.join(doing_dir, "debug.md"), "w") as f:
        f.write("## task1: 初始化项目结构（zombie）\n\n**分析过程 (Analysis)**:\n- 任务执行中断\n\n**实现步骤 (Implementation)**:\n1. 开始执行\n\n**遇到的问题 (Issues)**:\n- 任务未完成\n\n**验证结果 (Verification)**:\n- 测试命令: `echo check`\n- 测试输出:\n  ```\n  check\n  ```\n- 结论：❌ 失败\n")
    print("[mock_agent] doing_zombie_task: task1 left in 'running' state", file=sys.stderr)


def scenario_learning_success(prompt_content, learning_dir):
    """Generate valid learning outputs."""
    if not learning_dir:
        print("[mock_agent] ERROR: RICK_LEARNING_DIR not set", file=sys.stderr)
        sys.exit(1)
    os.makedirs(learning_dir, exist_ok=True)
    os.makedirs(os.path.join(learning_dir, "skills"), exist_ok=True)
    os.makedirs(os.path.join(learning_dir, "wiki"), exist_ok=True)

    summary = """APPROVED: true
# Job job_test 执行总结

本次任务执行成功，所有关键结果均已达成。

## 主要成果
1. 完成了项目初始化
2. 实现了核心功能
"""
    with open(os.path.join(learning_dir, "SUMMARY.md"), "w") as f:
        f.write(summary)

    okr = """## O1: 完成项目基础建设

### 关键结果
1. KR1: 创建项目结构 ✅
2. KR2: 实现核心功能 ✅
"""
    with open(os.path.join(learning_dir, "OKR.md"), "w") as f:
        f.write(okr)

    spec = """## 技术栈
- Go 1.21
- Cobra CLI

## 架构设计
模块化架构，清晰分层

## 开发规范
遵循 Go 标准规范

## 工程实践
使用 DAG 任务调度
"""
    with open(os.path.join(learning_dir, "SPEC.md"), "w") as f:
        f.write(spec)

    skill_py = """def check_go_build(project_root):
    \"\"\"Check if Go project builds successfully.\"\"\"
    import subprocess
    result = subprocess.run(
        ["go", "build", "./..."],
        cwd=project_root,
        capture_output=True,
        text=True
    )
    return result.returncode == 0
"""
    with open(os.path.join(learning_dir, "skills", "check_go_build.py"), "w") as f:
        f.write(skill_py)

    wiki_content = """# 项目架构

本项目采用模块化架构设计。
"""
    with open(os.path.join(learning_dir, "wiki", "architecture.md"), "w") as f:
        f.write(wiki_content)

    print("[mock_agent] learning_success: created all learning outputs", file=sys.stderr)


def scenario_learning_bad_skill(prompt_content, learning_dir):
    """Generate learning with a Python skill that has syntax errors."""
    if not learning_dir:
        print("[mock_agent] ERROR: RICK_LEARNING_DIR not set", file=sys.stderr)
        sys.exit(1)
    os.makedirs(learning_dir, exist_ok=True)
    os.makedirs(os.path.join(learning_dir, "skills"), exist_ok=True)

    summary = """APPROVED: true
# Job job_test 执行总结

本次任务执行成功。
"""
    with open(os.path.join(learning_dir, "SUMMARY.md"), "w") as f:
        f.write(summary)

    # Intentionally bad Python syntax
    bad_skill = """def broken_function(
    # Missing closing paren and body
    x = 1 +
"""
    with open(os.path.join(learning_dir, "skills", "bad_skill.py"), "w") as f:
        f.write(bad_skill)
    print("[mock_agent] learning_bad_skill: created skill with syntax error", file=sys.stderr)


def scenario_learning_no_summary(prompt_content, learning_dir):
    """Generate learning outputs without SUMMARY.md."""
    if not learning_dir:
        print("[mock_agent] ERROR: RICK_LEARNING_DIR not set", file=sys.stderr)
        sys.exit(1)
    os.makedirs(learning_dir, exist_ok=True)

    okr = """## O1: 完成项目

### 关键结果
1. KR1: 完成 ✅
"""
    with open(os.path.join(learning_dir, "OKR.md"), "w") as f:
        f.write(okr)
    # Intentionally NOT creating SUMMARY.md
    print("[mock_agent] learning_no_summary: created OKR.md but NO SUMMARY.md", file=sys.stderr)


def scenario_claude_exit_nonzero(prompt_content):
    """Exit with non-zero exit code to simulate Claude failure."""
    print("[mock_agent] claude_exit_nonzero: exiting with code 1", file=sys.stderr)
    sys.exit(1)


def scenario_claude_bad_output(prompt_content):
    """Output garbled/invalid content."""
    print("[mock_agent] claude_bad_output: outputting invalid content", file=sys.stderr)
    # Output binary-like garbage
    sys.stdout.buffer.write(b'\x00\x01\x02\xff\xfe\xfd invalid output \x00\x00')
    sys.stdout.flush()
    sys.exit(0)


def scenario_claude_timeout(prompt_content):
    """Sleep for a long time to simulate timeout."""
    print("[mock_agent] claude_timeout: sleeping for 300 seconds", file=sys.stderr)
    time.sleep(300)


# ─── Self-test ───────────────────────────────────────────────────────────────

def run_self_test():
    """Run self-test: verify each scenario produces the expected artifacts."""
    errors = []
    tmpdir = tempfile.mkdtemp(prefix="mock_agent_test_")

    try:
        # Test plan_success
        plan_dir = os.path.join(tmpdir, "plan_success", "plan")
        os.environ["RICK_PLAN_DIR"] = plan_dir
        scenario_plan_success("", plan_dir)
        if not os.path.exists(os.path.join(plan_dir, "task1.md")):
            errors.append("plan_success: task1.md not created")
        if not os.path.exists(os.path.join(plan_dir, "task2.md")):
            errors.append("plan_success: task2.md not created")
        # Verify task1.md has all required sections
        with open(os.path.join(plan_dir, "task1.md")) as f:
            content = f.read()
        for section in ["# 依赖关系", "# 任务名称", "# 任务目标", "# 关键结果", "# 测试方法"]:
            if section not in content:
                errors.append(f"plan_success: task1.md missing section {section}")

        # Test plan_missing_section
        plan_dir2 = os.path.join(tmpdir, "plan_missing", "plan")
        os.environ["RICK_PLAN_DIR"] = plan_dir2
        scenario_plan_missing_section("", plan_dir2)
        with open(os.path.join(plan_dir2, "task1.md")) as f:
            content = f.read()
        if "# 关键结果" in content:
            errors.append("plan_missing_section: task1.md should NOT have 关键结果")

        # Test plan_circular_dep
        plan_dir3 = os.path.join(tmpdir, "plan_circular", "plan")
        os.environ["RICK_PLAN_DIR"] = plan_dir3
        scenario_plan_circular_dep("", plan_dir3)
        if not os.path.exists(os.path.join(plan_dir3, "task1.md")):
            errors.append("plan_circular_dep: task1.md not created")

        # Test doing_success
        doing_dir = os.path.join(tmpdir, "doing_success", "doing")
        os.environ["RICK_DOING_DIR"] = doing_dir
        scenario_doing_success("", doing_dir)
        if not os.path.exists(os.path.join(doing_dir, "tasks.json")):
            errors.append("doing_success: tasks.json not created")
        if not os.path.exists(os.path.join(doing_dir, "debug.md")):
            errors.append("doing_success: debug.md not created")
        with open(os.path.join(doing_dir, "tasks.json")) as f:
            data = json.load(f)
        for task in data["tasks"]:
            if task["status"] == "success" and not task.get("commit_hash"):
                errors.append(f"doing_success: task {task['task_id']} missing commit_hash")

        # Test doing_no_debug
        doing_dir2 = os.path.join(tmpdir, "doing_no_debug", "doing")
        os.environ["RICK_DOING_DIR"] = doing_dir2
        scenario_doing_no_debug("", doing_dir2)
        if os.path.exists(os.path.join(doing_dir2, "debug.md")):
            errors.append("doing_no_debug: debug.md should NOT exist")

        # Test doing_zombie_task
        doing_dir3 = os.path.join(tmpdir, "doing_zombie", "doing")
        os.environ["RICK_DOING_DIR"] = doing_dir3
        scenario_doing_zombie_task("", doing_dir3)
        with open(os.path.join(doing_dir3, "tasks.json")) as f:
            data = json.load(f)
        has_running = any(t["status"] == "running" for t in data["tasks"])
        if not has_running:
            errors.append("doing_zombie_task: no task in 'running' state")

        # Test learning_success
        learning_dir = os.path.join(tmpdir, "learning_success", "learning")
        os.environ["RICK_LEARNING_DIR"] = learning_dir
        scenario_learning_success("", learning_dir)
        if not os.path.exists(os.path.join(learning_dir, "SUMMARY.md")):
            errors.append("learning_success: SUMMARY.md not created")
        with open(os.path.join(learning_dir, "SUMMARY.md")) as f:
            content = f.read()
        if not content.startswith("APPROVED: true"):
            errors.append("learning_success: SUMMARY.md should start with 'APPROVED: true'")
        if "# Job" not in content:
            errors.append("learning_success: SUMMARY.md should contain '# Job' heading")
        # Check skill is valid Python
        skill_path = os.path.join(learning_dir, "skills", "check_go_build.py")
        if not os.path.exists(skill_path):
            errors.append("learning_success: check_go_build.py not created")
        else:
            result = subprocess.run(
                ["python3", "-c", f"import ast; ast.parse(open({repr(skill_path)}).read())"],
                capture_output=True, text=True
            )
            if result.returncode != 0:
                errors.append(f"learning_success: skill has syntax error: {result.stderr}")

        # Test learning_bad_skill
        learning_dir2 = os.path.join(tmpdir, "learning_bad", "learning")
        os.environ["RICK_LEARNING_DIR"] = learning_dir2
        scenario_learning_bad_skill("", learning_dir2)
        bad_skill_path = os.path.join(learning_dir2, "skills", "bad_skill.py")
        if not os.path.exists(bad_skill_path):
            errors.append("learning_bad_skill: bad_skill.py not created")
        else:
            result = subprocess.run(
                ["python3", "-c", f"import ast; ast.parse(open({repr(bad_skill_path)}).read())"],
                capture_output=True, text=True
            )
            if result.returncode == 0:
                errors.append("learning_bad_skill: bad_skill.py should have syntax error")

        # Test learning_no_summary
        learning_dir3 = os.path.join(tmpdir, "learning_no_summary", "learning")
        os.environ["RICK_LEARNING_DIR"] = learning_dir3
        scenario_learning_no_summary("", learning_dir3)
        if os.path.exists(os.path.join(learning_dir3, "SUMMARY.md")):
            errors.append("learning_no_summary: SUMMARY.md should NOT exist")

    finally:
        shutil.rmtree(tmpdir, ignore_errors=True)

    if errors:
        print(f"FAIL: {len(errors)} error(s):", file=sys.stderr)
        for e in errors:
            print(f"  - {e}", file=sys.stderr)
        sys.exit(1)
    else:
        print("OK: all self-tests passed", file=sys.stderr)
        sys.exit(0)


# ─── Main ────────────────────────────────────────────────────────────────────

def main():
    if len(sys.argv) >= 2 and sys.argv[1] == "--self-test":
        run_self_test()
        return

    if len(sys.argv) < 2:
        print("Usage: mock_agent.py <prompt_file>", file=sys.stderr)
        print("       mock_agent.py --self-test", file=sys.stderr)
        sys.exit(1)

    prompt_file = sys.argv[1]
    if not os.path.exists(prompt_file):
        print(f"Error: prompt file not found: {prompt_file}", file=sys.stderr)
        sys.exit(1)

    prompt_content = read_prompt_file(prompt_file)
    scenario = os.environ.get("MOCK_SCENARIO", "")

    if not scenario:
        print("Error: MOCK_SCENARIO environment variable not set", file=sys.stderr)
        sys.exit(1)

    # Route to scenario handler
    if scenario == "plan_success":
        plan_dir = find_plan_dir_from_env() or extract_job_dir_from_prompt(prompt_content)
        scenario_plan_success(prompt_content, plan_dir)

    elif scenario == "plan_missing_section":
        plan_dir = find_plan_dir_from_env() or extract_job_dir_from_prompt(prompt_content)
        scenario_plan_missing_section(prompt_content, plan_dir)

    elif scenario == "plan_circular_dep":
        plan_dir = find_plan_dir_from_env() or extract_job_dir_from_prompt(prompt_content)
        scenario_plan_circular_dep(prompt_content, plan_dir)

    elif scenario == "doing_success":
        doing_dir = find_doing_dir_from_env() or extract_job_dir_from_prompt(prompt_content)
        scenario_doing_success(prompt_content, doing_dir)

    elif scenario == "doing_no_debug":
        doing_dir = find_doing_dir_from_env() or extract_job_dir_from_prompt(prompt_content)
        scenario_doing_no_debug(prompt_content, doing_dir)

    elif scenario == "doing_zombie_task":
        doing_dir = find_doing_dir_from_env() or extract_job_dir_from_prompt(prompt_content)
        scenario_doing_zombie_task(prompt_content, doing_dir)

    elif scenario == "learning_success":
        learning_dir = find_learning_dir_from_env() or extract_job_dir_from_prompt(prompt_content)
        scenario_learning_success(prompt_content, learning_dir)

    elif scenario == "learning_bad_skill":
        learning_dir = find_learning_dir_from_env() or extract_job_dir_from_prompt(prompt_content)
        scenario_learning_bad_skill(prompt_content, learning_dir)

    elif scenario == "learning_no_summary":
        learning_dir = find_learning_dir_from_env() or extract_job_dir_from_prompt(prompt_content)
        scenario_learning_no_summary(prompt_content, learning_dir)

    elif scenario == "claude_exit_nonzero":
        scenario_claude_exit_nonzero(prompt_content)

    elif scenario == "claude_bad_output":
        scenario_claude_bad_output(prompt_content)

    elif scenario == "claude_timeout":
        scenario_claude_timeout(prompt_content)

    else:
        print(f"Error: unknown MOCK_SCENARIO: {scenario}", file=sys.stderr)
        sys.exit(1)


if __name__ == "__main__":
    main()
