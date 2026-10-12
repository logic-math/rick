#!/usr/bin/env python3
import json
import sys
import os
import subprocess

PROJECT_ROOT = "/Users/sunquan/ai_coding/CODING/rick"

EXPECTED_SKILL_FILES = [
    "sense.md",
    "tc.md",
    "tdd.md",
    "testing.md",
    "debug.md",
    "gen-skill.md",
    "evolve-skills.md",
    os.path.join("tdd", "testing-anti-patterns.md"),
]

def main():
    errors = []

    # Test 1: 验证 8 个 core-skill 文件存在且非空
    skills_dir = os.path.join(PROJECT_ROOT, "internal", "prompt", "templates", "skills")
    for skill_file in EXPECTED_SKILL_FILES:
        full_path = os.path.join(skills_dir, skill_file)
        if not os.path.exists(full_path):
            errors.append(f"skill file missing: internal/prompt/templates/skills/{skill_file}")
        else:
            try:
                with open(full_path, "r") as f:
                    content = f.read().strip()
                if not content:
                    errors.append(f"skill file is empty: internal/prompt/templates/skills/{skill_file}")
            except Exception as e:
                errors.append(f"failed to read skill file {skill_file}: {e}")

    # Test 2: 编译项目
    build_script = os.path.join(PROJECT_ROOT, "tools", "build_and_get_rick_bin.py")
    if not os.path.exists(build_script):
        errors.append(f"build script not found: {build_script}")
    else:
        try:
            result = subprocess.run(
                ["python3", build_script],
                cwd=PROJECT_ROOT,
                capture_output=True,
                text=True,
                timeout=120,
            )
            if result.returncode != 0:
                errors.append(f"build failed (exit {result.returncode}): {result.stderr.strip()[:500]}")
        except subprocess.TimeoutExpired:
            errors.append("build timed out after 120s")
        except Exception as e:
            errors.append(f"build script error: {e}")

    # Test 3: 运行 TestCoreSkillsEmbed 单元测试
    try:
        result = subprocess.run(
            ["go", "test", "./internal/prompt/...", "-run", "TestCoreSkillsEmbed", "-v"],
            cwd=PROJECT_ROOT,
            capture_output=True,
            text=True,
            timeout=60,
        )
        if result.returncode != 0:
            errors.append(f"TestCoreSkillsEmbed failed: {result.stdout.strip()[-500:]}\n{result.stderr.strip()[-200:]}")
        else:
            output = result.stdout
            # 验证测试确实运行了（不是 no test files）
            if "TestCoreSkillsEmbed" not in output and "no test files" not in output:
                errors.append("TestCoreSkillsEmbed did not appear to run")
            elif "FAIL" in output:
                errors.append(f"TestCoreSkillsEmbed reported FAIL: {output.strip()[-500:]}")
    except subprocess.TimeoutExpired:
        errors.append("TestCoreSkillsEmbed timed out after 60s")
    except Exception as e:
        errors.append(f"go test error: {e}")

    # Test 4: 全量测试无新增失败
    try:
        result = subprocess.run(
            ["go", "test", "./..."],
            cwd=PROJECT_ROOT,
            capture_output=True,
            text=True,
            timeout=120,
        )
        if result.returncode != 0:
            errors.append(f"go test ./... failed: {result.stdout.strip()[-500:]}\n{result.stderr.strip()[-200:]}")
    except subprocess.TimeoutExpired:
        errors.append("go test ./... timed out after 120s")
    except Exception as e:
        errors.append(f"go test ./... error: {e}")

    result = {
        "pass": len(errors) == 0,
        "errors": errors,
    }

    print(json.dumps(result))
    sys.exit(0 if result["pass"] else 1)


if __name__ == "__main__":
    main()
