#!/usr/bin/env python3
"""
e2e_v2_test.py - End-to-end v2 tests for Rick workflows using mock agent.

Tests 4 phases: plan / doing (stream-json) / learning (v2) / dream.
Each phase uses os.chdir(tmpdir) for CWD isolation.
"""

import json
import os
import shutil
import subprocess
import sys
import tempfile

PROJECT_ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
MOCK_AGENT = os.path.join(PROJECT_ROOT, "tests", "mock_agent", "mock_agent.py")


def run_mock(scenario, extra_args=None, env_extra=None, cwd=None):
    """Run mock_agent.py with the given scenario and return (stdout, stderr, returncode)."""
    env = os.environ.copy()
    env["MOCK_SCENARIO"] = scenario
    if env_extra:
        env.update(env_extra)

    cmd = ["python3", MOCK_AGENT]
    if extra_args:
        cmd.extend(extra_args)
    cmd.append("/dev/null")

    result = subprocess.run(
        cmd,
        capture_output=True,
        text=True,
        env=env,
        timeout=30,
        cwd=cwd or PROJECT_ROOT,
    )
    return result.stdout, result.stderr, result.returncode


def phase1_plan(errors):
    """Phase 1: plan_success - validate task1.md and OKR.md created."""
    tmpdir = tempfile.mkdtemp(prefix="e2e_v2_phase1_")
    try:
        plan_dir = os.path.join(tmpdir, ".rick", "jobs", "job_test", "plan")
        os.makedirs(plan_dir, exist_ok=True)

        _, _, rc = run_mock(
            "plan_success",
            env_extra={"RICK_PLAN_DIR": plan_dir},
            cwd=tmpdir,
        )
        if rc != 0:
            errors.append(f"Phase 1 Plan: mock exited with code {rc}")
            return

        task1_path = os.path.join(plan_dir, "task1.md")
        okr_path = os.path.join(plan_dir, "OKR.md")

        if not os.path.exists(task1_path):
            errors.append("Phase 1 Plan: task1.md not created")
        else:
            with open(task1_path) as f:
                content = f.read()
            for section in ["# 依赖关系", "# 任务名称", "# 任务目标", "# 关键结果", "# 测试方法"]:
                if section not in content:
                    errors.append(f"Phase 1 Plan: task1.md missing section {section}")

        if not os.path.exists(okr_path):
            errors.append("Phase 1 Plan: OKR.md not created")
        else:
            with open(okr_path) as f:
                content = f.read()
            if "KR" not in content:
                errors.append("Phase 1 Plan: OKR.md missing KR entries")

    finally:
        shutil.rmtree(tmpdir, ignore_errors=True)


def phase2_doing(errors):
    """Phase 2: doing_v2_success - validate act-path.md, raw_session.log."""
    tmpdir = tempfile.mkdtemp(prefix="e2e_v2_phase2_")
    try:
        doing_dir = os.path.join(tmpdir, ".rick", "jobs", "job_test", "doing")
        os.makedirs(doing_dir, exist_ok=True)

        stdout, _, rc = run_mock(
            "doing_v2_success",
            extra_args=["--output-format", "stream-json", "--verbose"],
            env_extra={"RICK_DOING_DIR": doing_dir},
            cwd=tmpdir,
        )

        # Validate NDJSON stream output (>=4 lines)
        ndjson_lines = [l for l in stdout.strip().splitlines() if l.strip()]
        if len(ndjson_lines) < 4:
            errors.append(
                f"Phase 2 Doing: expected >=4 NDJSON lines, got {len(ndjson_lines)}: {stdout[:200]}"
            )
        else:
            for i, line in enumerate(ndjson_lines):
                try:
                    json.loads(line)
                except json.JSONDecodeError as je:
                    errors.append(f"Phase 2 Doing: NDJSON line {i+1} invalid JSON: {je}")

            last = json.loads(ndjson_lines[-1])
            if "session_id" not in last:
                errors.append(f"Phase 2 Doing: last NDJSON line missing session_id: {ndjson_lines[-1][:80]}")

            tool_use_found = False
            for line in ndjson_lines:
                obj = json.loads(line)
                if obj.get("type") == "tool_use":
                    errors.append("Phase 2 Doing: tool_use found at top level (must be in message.content[])")
                if obj.get("type") == "assistant":
                    for item in obj.get("message", {}).get("content", []):
                        if item.get("type") == "tool_use":
                            tool_use_found = True
            if not tool_use_found:
                errors.append("Phase 2 Doing: no tool_use nested in message.content[]")

        # Validate act-path.md
        act_path = os.path.join(doing_dir, "tasks", "task1", "act-path.md")
        if not os.path.exists(act_path):
            errors.append(f"Phase 2 Doing: act-path.md not found at {act_path}")
        else:
            with open(act_path) as f:
                content = f.read()
            if "## 执行摘要" not in content:
                errors.append("Phase 2 Doing: act-path.md missing '## 执行摘要'")
            if "报错次数: 0" not in content:
                errors.append("Phase 2 Doing: act-path.md missing '报错次数: 0'")
            if "raw_session.log" not in content:
                errors.append("Phase 2 Doing: act-path.md missing raw_session.log reference link")

        # Validate raw_session.log
        raw_log = os.path.join(doing_dir, "tasks", "task1", "raw_session.log")
        if not os.path.exists(raw_log):
            errors.append(f"Phase 2 Doing: raw_session.log not found at {raw_log}")
        else:
            with open(raw_log) as f:
                log_lines = [l for l in f.read().strip().splitlines() if l.strip()]
            if not log_lines:
                errors.append("Phase 2 Doing: raw_session.log is empty")
            else:
                for i, line in enumerate(log_lines):
                    try:
                        json.loads(line)
                    except json.JSONDecodeError as je:
                        errors.append(f"Phase 2 Doing: raw_session.log line {i+1} not valid JSON: {je}")

    finally:
        shutil.rmtree(tmpdir, ignore_errors=True)


def phase3_learning(errors):
    """Phase 3: learning_v2_success - validate SUMMARY.md and run_log_1.md."""
    tmpdir = tempfile.mkdtemp(prefix="e2e_v2_phase3_")
    try:
        learning_dir = os.path.join(tmpdir, ".rick", "jobs", "job_test", "learning")
        rick_dir = os.path.join(tmpdir, ".rick")
        os.makedirs(learning_dir, exist_ok=True)
        os.makedirs(rick_dir, exist_ok=True)

        _, _, rc = run_mock(
            "learning_v2_success",
            env_extra={
                "RICK_LEARNING_DIR": learning_dir,
                "RICK_DIR": rick_dir,
            },
            cwd=tmpdir,
        )
        if rc != 0:
            errors.append(f"Phase 3 Learning: mock exited with code {rc}")
            return

        summary_path = os.path.join(learning_dir, "SUMMARY.md")
        if not os.path.exists(summary_path):
            errors.append("Phase 3 Learning: SUMMARY.md not created")
        else:
            with open(summary_path) as f:
                content = f.read()
            if "APPROVED: true" not in content:
                errors.append("Phase 3 Learning: SUMMARY.md missing 'APPROVED: true'")

        run_log_path = os.path.join(rick_dir, "dream", "run_log_1.md")
        if not os.path.exists(run_log_path):
            errors.append(f"Phase 3 Learning: run_log_1.md not created at {run_log_path}")
        else:
            with open(run_log_path) as f:
                content = f.read()
            if "job_test" not in content:
                errors.append("Phase 3 Learning: run_log_1.md missing job_test reference")

    finally:
        shutil.rmtree(tmpdir, ignore_errors=True)


def phase4_dream(errors):
    """Phase 4: dream_success - validate readme.md contains processed job record."""
    tmpdir = tempfile.mkdtemp(prefix="e2e_v2_phase4_")
    try:
        rick_dir = os.path.join(tmpdir, ".rick")
        dream_dir = os.path.join(rick_dir, "dream")
        os.makedirs(dream_dir, exist_ok=True)

        _, _, rc = run_mock(
            "dream_success",
            env_extra={"RICK_DIR": rick_dir},
            cwd=tmpdir,
        )
        if rc != 0:
            errors.append(f"Phase 4 Dream: mock exited with code {rc}")
            return

        readme_path = os.path.join(dream_dir, "readme.md")
        if not os.path.exists(readme_path):
            errors.append(f"Phase 4 Dream: readme.md not created at {readme_path}")
        else:
            with open(readme_path) as f:
                content = f.read()
            if "job_test" not in content:
                errors.append("Phase 4 Dream: readme.md missing job_test record")
            if "已处理" not in content and "Processed" not in content:
                errors.append("Phase 4 Dream: readme.md missing processed status marker")

    finally:
        shutil.rmtree(tmpdir, ignore_errors=True)


def main():
    errors = []

    print("[e2e_v2] Phase 1: Plan", file=sys.stderr)
    phase1_plan(errors)

    print("[e2e_v2] Phase 2: Doing (stream-json)", file=sys.stderr)
    phase2_doing(errors)

    print("[e2e_v2] Phase 3: Learning v2", file=sys.stderr)
    phase3_learning(errors)

    print("[e2e_v2] Phase 4: Dream", file=sys.stderr)
    phase4_dream(errors)

    if errors:
        print(f"\n❌ E2E v2 FAILED: {len(errors)} error(s):", file=sys.stderr)
        for e in errors:
            print(f"  - {e}", file=sys.stderr)
        sys.exit(1)
    else:
        print("\n✅ E2E v2 all phases passed", file=sys.stderr)
        sys.exit(0)


if __name__ == "__main__":
    main()
