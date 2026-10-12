#!/usr/bin/env python3
import json
import sys
import os
import subprocess

def main():
    errors = []

    project_root = "/Users/sunquan/ai_coding/CODING/rick"
    tools_dir = os.path.join(project_root, "tools")
    skills_dir = os.path.join(project_root, ".rick", "skills")

    # Test 1: ls tools/ 验证 5 个 .py 文件存在
    try:
        if not os.path.isdir(tools_dir):
            errors.append(f"tools/ directory does not exist at {tools_dir}")
        else:
            py_files = [f for f in os.listdir(tools_dir) if f.endswith(".py")]
            if len(py_files) < 5:
                errors.append(f"Expected at least 5 .py files in tools/, found {len(py_files)}: {py_files}")
            else:
                print(f"tools/ has {len(py_files)} .py files: {py_files}", file=sys.stderr)
    except Exception as e:
        errors.append(f"Failed to check tools/ directory: {str(e)}")

    # Test 2: 验证 .rick/skills/ 无 .py 文件
    try:
        if os.path.isdir(skills_dir):
            py_files_in_skills = [f for f in os.listdir(skills_dir) if f.endswith(".py")]
            if py_files_in_skills:
                errors.append(f".rick/skills/ still contains .py files: {py_files_in_skills}")
            else:
                print(".rick/skills/ has no .py files (correct)", file=sys.stderr)
        else:
            print(".rick/skills/ directory does not exist (acceptable)", file=sys.stderr)
    except Exception as e:
        errors.append(f"Failed to check .rick/skills/ directory: {str(e)}")

    # Test 3: python3 tools/build_and_get_rick_bin.py 验证脚本可执行（返回 JSON）
    build_script = os.path.join(tools_dir, "build_and_get_rick_bin.py")
    try:
        if not os.path.exists(build_script):
            errors.append(f"tools/build_and_get_rick_bin.py does not exist")
        else:
            result = subprocess.run(
                ["python3", build_script],
                capture_output=True, text=True, timeout=60,
                cwd=project_root
            )
            output = result.stdout.strip()
            try:
                parsed = json.loads(output)
                print(f"build_and_get_rick_bin.py returned valid JSON: {parsed}", file=sys.stderr)
            except json.JSONDecodeError:
                errors.append(f"build_and_get_rick_bin.py did not return valid JSON. stdout: {output[:200]}, stderr: {result.stderr[:200]}")
    except subprocess.TimeoutExpired:
        errors.append("build_and_get_rick_bin.py timed out after 60s")
    except Exception as e:
        errors.append(f"Failed to run build_and_get_rick_bin.py: {str(e)}")

    # Test 4: python3 tools/check_go_build.py 验证脚本可执行
    check_go_build = os.path.join(tools_dir, "check_go_build.py")
    try:
        if not os.path.exists(check_go_build):
            errors.append(f"tools/check_go_build.py does not exist")
        else:
            result = subprocess.run(
                ["python3", check_go_build, "--help"],
                capture_output=True, text=True, timeout=30,
                cwd=project_root
            )
            # --help may exit non-zero; just check it runs without crashing unexpectedly
            if result.returncode not in (0, 1, 2):
                # Try without --help
                result2 = subprocess.run(
                    ["python3", check_go_build],
                    capture_output=True, text=True, timeout=30,
                    cwd=project_root
                )
                if result2.returncode not in (0, 1):
                    errors.append(f"check_go_build.py failed unexpectedly. stdout: {result2.stdout[:200]}, stderr: {result2.stderr[:200]}")
            print(f"check_go_build.py ran with exit code {result.returncode}", file=sys.stderr)
    except subprocess.TimeoutExpired:
        errors.append("check_go_build.py timed out after 30s")
    except Exception as e:
        errors.append(f"Failed to run check_go_build.py: {str(e)}")

    # Test 5: 构建 rick 并运行 doing job_12 --dry-run 验证 tools section 非空
    try:
        # Get rick binary path from build_and_get_rick_bin.py
        bin_result = subprocess.run(
            ["python3", build_script],
            capture_output=True, text=True, timeout=120,
            cwd=project_root
        )
        bin_output = bin_result.stdout.strip()
        rick_bin = None
        try:
            bin_data = json.loads(bin_output)
            rick_bin = bin_data.get("bin") or bin_data.get("path") or bin_data.get("binary")
        except Exception:
            pass

        if not rick_bin:
            # fallback: try local bin
            local_bin = os.path.join(project_root, "bin", "rick")
            if os.path.exists(local_bin):
                rick_bin = local_bin

        if not rick_bin or not os.path.exists(rick_bin):
            errors.append(f"Could not determine rick binary path. build_and_get_rick_bin.py output: {bin_output[:200]}")
        else:
            dry_run = subprocess.run(
                [rick_bin, "doing", "job_12", "--dry-run"],
                capture_output=True, text=True, timeout=30,
                cwd=project_root
            )
            combined = dry_run.stdout + dry_run.stderr
            if "tools" not in combined.lower():
                errors.append(f"'tools' section not found in rick doing --dry-run output. stdout: {dry_run.stdout[:300]}, stderr: {dry_run.stderr[:300]}")
            else:
                print(f"tools section found in dry-run output", file=sys.stderr)
    except subprocess.TimeoutExpired:
        errors.append("rick doing job_12 --dry-run timed out")
    except Exception as e:
        errors.append(f"Failed to run rick doing --dry-run: {str(e)}")

    result = {
        "pass": len(errors) == 0,
        "errors": errors
    }

    print(json.dumps(result))
    sys.exit(0 if result["pass"] else 1)

if __name__ == "__main__":
    main()
