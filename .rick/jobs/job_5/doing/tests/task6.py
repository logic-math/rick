#!/usr/bin/env python3
import json
import sys
import os
import subprocess
import shutil
import tempfile

def run_cmd(cmd, cwd=None):
    result = subprocess.run(cmd, shell=True, capture_output=True, text=True, cwd=cwd)
    return result.returncode, result.stdout, result.stderr

def main():
    errors = []

    # Project root
    project_root = "/Users/sunquan/ai_coding/CODING/rick"
    rick_dir = os.path.join(project_root, ".rick")
    binary = os.path.join(project_root, "bin", "rick")

    # Step 1: Build binary
    print("Building binary...", file=sys.stderr)
    rc, out, err = run_cmd("go build -o bin/rick ./cmd/rick/", cwd=project_root)
    if rc != 0:
        errors.append(f"go build failed: {err}")
        print(json.dumps({"pass": False, "errors": errors}))
        sys.exit(1)

    # --- Setup: prepare job_1 learning test files ---
    job1_learning = os.path.join(rick_dir, "jobs", "job_1", "learning")
    os.makedirs(os.path.join(job1_learning, "wiki"), exist_ok=True)
    os.makedirs(os.path.join(job1_learning, "skills"), exist_ok=True)

    # Create test wiki file
    with open(os.path.join(job1_learning, "wiki", "test_wiki.md"), "w") as f:
        f.write("# 测试 Wiki\n\nThis is a test wiki page.\n")

    # Create test skill file
    with open(os.path.join(job1_learning, "skills", "test_skill.py"), "w") as f:
        f.write("# Description: 测试技能\n\nprint('test skill')\n")

    # Create OKR.md
    with open(os.path.join(job1_learning, "OKR.md"), "w") as f:
        f.write("<!-- Updated in learning phase -->\n# OKR\n\nTest OKR content.\n")

    # Create SPEC.md
    with open(os.path.join(job1_learning, "SPEC.md"), "w") as f:
        f.write("<!-- Updated in learning phase -->\n# SPEC\n\nTest SPEC content.\n")

    # Step 2: Test rejection without APPROVED: true
    print("Testing rejection without APPROVED...", file=sys.stderr)
    # Ensure SUMMARY.md does NOT have APPROVED: true
    summary_path = os.path.join(job1_learning, "SUMMARY.md")
    with open(summary_path, "w") as f:
        f.write("# Summary\n\nNo approval here.\n")

    rc, out, err = run_cmd(f"{binary} tools merge job_1", cwd=project_root)
    if rc == 0:
        errors.append("rick tools merge job_1 should fail without APPROVED: true in SUMMARY.md, but it succeeded")
    else:
        combined = out + err
        if "APPROVED" not in combined and "approved" not in combined.lower() and "approve" not in combined.lower():
            errors.append(f"rejection message should mention APPROVED, got: {combined[:200]}")

    # Step 3: Add APPROVED: true to SUMMARY.md and run merge
    print("Testing merge with APPROVED: true...", file=sys.stderr)
    with open(summary_path, "w") as f:
        f.write("APPROVED: true\n\n# Summary\n\nLearning summary content.\n")

    # Get current branch before merge
    rc_branch, current_branch, _ = run_cmd("git rev-parse --abbrev-ref HEAD", cwd=project_root)
    current_branch = current_branch.strip()

    rc, out, err = run_cmd(f"{binary} tools merge job_1", cwd=project_root)
    if rc != 0:
        errors.append(f"rick tools merge job_1 failed: {err}\nstdout: {out}")
    else:
        # Check wiki file
        wiki_file = os.path.join(rick_dir, "wiki", "test_wiki.md")
        if not os.path.exists(wiki_file):
            errors.append(f".rick/wiki/test_wiki.md does not exist after merge")

        # Check wiki README
        wiki_readme = os.path.join(rick_dir, "wiki", "README.md")
        if os.path.exists(wiki_readme):
            with open(wiki_readme, "r") as f:
                content = f.read()
            if "test_wiki" not in content:
                errors.append(".rick/wiki/README.md does not contain test_wiki entry")
        else:
            errors.append(".rick/wiki/README.md does not exist after merge")

        # Check skills file
        skill_file = os.path.join(rick_dir, "skills", "test_skill.py")
        if not os.path.exists(skill_file):
            errors.append(f".rick/skills/test_skill.py does not exist after merge")

        # Check skills README
        skills_readme = os.path.join(rick_dir, "skills", "README.md")
        if os.path.exists(skills_readme):
            with open(skills_readme, "r") as f:
                content = f.read()
            if "test_skill" not in content and "测试技能" not in content:
                errors.append(".rick/skills/README.md does not contain test_skill description")
        else:
            errors.append(".rick/skills/README.md does not exist after merge")

        # Check OKR.md overwritten
        okr_file = os.path.join(rick_dir, "OKR.md")
        if os.path.exists(okr_file):
            with open(okr_file, "r") as f:
                content = f.read()
            if "Test OKR content" not in content:
                errors.append(".rick/OKR.md was not overwritten with learning/OKR.md content")
        else:
            errors.append(".rick/OKR.md does not exist after merge")

        # Check SPEC.md overwritten
        spec_file = os.path.join(rick_dir, "SPEC.md")
        if os.path.exists(spec_file):
            with open(spec_file, "r") as f:
                content = f.read()
            if "Test SPEC content" not in content:
                errors.append(".rick/SPEC.md was not overwritten with learning/SPEC.md content")
        else:
            errors.append(".rick/SPEC.md does not exist after merge")

        # Check git log shows new commit
        rc_log, log_out, _ = run_cmd("git log --oneline -5", cwd=project_root)
        if "learning" not in log_out and "merge" not in log_out.lower() and "job_1" not in log_out:
            errors.append(f"git log does not show expected commit after merge: {log_out[:200]}")

        # Check output contains summary info
        if "wiki" not in out.lower() and "skill" not in out.lower():
            errors.append(f"merge output should contain summary of changes (wiki/skills), got: {out[:300]}")

        # Check current branch switched back to original
        rc_b2, branch_now, _ = run_cmd("git rev-parse --abbrev-ref HEAD", cwd=project_root)
        branch_now = branch_now.strip()
        if branch_now != current_branch:
            errors.append(f"branch should be restored to '{current_branch}' after merge, but is '{branch_now}'")

    # Step 4: Test skills injection into doing prompt
    print("Testing skills injection in doing prompt...", file=sys.stderr)
    skills_dir = os.path.join(rick_dir, "skills")
    os.makedirs(skills_dir, exist_ok=True)

    # Place a skill file with Description
    check_go_skill = os.path.join(skills_dir, "check_go_build.py")
    with open(check_go_skill, "w") as f:
        f.write("# Description: 检查 Go 项目编译\n\nimport subprocess\nsubprocess.run(['go', 'build', './...'])\n")

    # Run doing with --dry-run and check for skills section
    rc, out, err = run_cmd(f"{binary} doing job_1 --dry-run", cwd=project_root)
    combined_output = out + err
    if "可用的项目 Skills" not in combined_output and "Skills" not in combined_output:
        # Also check for temp prompt file
        # Try to find generated prompt file
        found_skills_section = False
        tmp_dir = tempfile.gettempdir()
        for fname in os.listdir(tmp_dir):
            if fname.startswith("rick") and fname.endswith(".md"):
                fpath = os.path.join(tmp_dir, fname)
                try:
                    with open(fpath, "r") as f:
                        content = f.read()
                    if "可用的项目 Skills" in content or ("Skills" in content and "check_go_build" in content):
                        found_skills_section = True
                        break
                except Exception:
                    pass
        if not found_skills_section:
            errors.append("doing --dry-run output/prompt does not contain skills section when skills exist")

    # Step 5: Test that empty skills dir does NOT produce skills section
    print("Testing empty skills dir...", file=sys.stderr)
    # Remove the skill file temporarily
    os.remove(check_go_skill)
    # Remove any other .py files in skills dir
    for fname in os.listdir(skills_dir):
        if fname.endswith(".py"):
            os.remove(os.path.join(skills_dir, fname))

    rc, out, err = run_cmd(f"{binary} doing job_1 --dry-run", cwd=project_root)
    if rc != 0:
        # Only error if it's not a "no tasks" type error
        if "no task" not in (out + err).lower() and "task" not in (out + err).lower():
            errors.append(f"doing --dry-run with empty skills dir returned error: {err[:200]}")
    # Check output doesn't contain skills section
    combined_output = out + err
    if "可用的项目 Skills" in combined_output:
        errors.append("doing --dry-run should not contain skills section when skills dir is empty")

    # Restore skill file for cleanup
    with open(check_go_skill, "w") as f:
        f.write("# Description: 检查 Go 项目编译\n\nimport subprocess\nsubprocess.run(['go', 'build', './...'])\n")

    # Step 6: Run unit tests
    print("Running unit tests...", file=sys.stderr)
    rc, out, err = run_cmd(
        "go test ./internal/workspace/... ./internal/prompt/... ./internal/cmd/...",
        cwd=project_root
    )
    if rc != 0:
        errors.append(f"go test failed:\n{err}\n{out}")

    result = {
        "pass": len(errors) == 0,
        "errors": errors
    }

    print(json.dumps(result))
    sys.exit(0 if result["pass"] else 1)

if __name__ == "__main__":
    main()
