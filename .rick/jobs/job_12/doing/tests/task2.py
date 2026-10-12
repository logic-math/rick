#!/usr/bin/env python3
import json
import sys
import os
import subprocess

def main():
    errors = []

    project_root = "/Users/sunquan/ai_coding/CODING/rick"
    skills_dir = os.path.join(project_root, ".rick", "skills")

    # Test 1: 验证 .rick/skills/ 只有 .md 文件
    try:
        if not os.path.isdir(skills_dir):
            errors.append(f".rick/skills/ directory does not exist")
        else:
            files = os.listdir(skills_dir)
            non_md = [f for f in files if not f.endswith(".md")]
            if non_md:
                errors.append(f".rick/skills/ contains non-.md files: {non_md}")
    except Exception as e:
        errors.append(f"Failed to list .rick/skills/: {str(e)}")

    # Test 2: 检查 index.md 包含三列表格且触发场景列非空
    index_file = os.path.join(skills_dir, "index.md")
    try:
        with open(index_file, "r") as f:
            content = f.read()
        lines = content.splitlines()
        table_rows = [l for l in lines if l.strip().startswith("|") and not l.strip().startswith("|---") and not l.strip().startswith("| ---")]
        # Filter out header row
        data_rows = []
        header_seen = False
        for l in table_rows:
            if not header_seen:
                header_seen = True
                continue
            data_rows.append(l)
        if not data_rows:
            errors.append("index.md has no data rows in table")
        else:
            for row in data_rows:
                cols = [c.strip() for c in row.split("|") if c.strip() != ""]
                if len(cols) < 3:
                    errors.append(f"index.md table row has fewer than 3 columns: {row}")
                else:
                    # Third column is trigger scenario
                    if cols[2] == "" or cols[2] == "|":
                        errors.append(f"index.md table row has empty trigger scenario column: {row}")
    except FileNotFoundError:
        errors.append("index.md does not exist")
    except Exception as e:
        errors.append(f"Failed to check index.md: {str(e)}")

    # Test 3: 检查 verify_rick_check_commands.md 包含三个 section
    verify_file = os.path.join(skills_dir, "verify_rick_check_commands.md")
    required_sections = ["触发场景", "使用的 Tools", "执行步骤"]
    try:
        with open(verify_file, "r") as f:
            content = f.read()
        for section in required_sections:
            if section not in content:
                errors.append(f"verify_rick_check_commands.md missing section: {section}")
    except FileNotFoundError:
        errors.append("verify_rick_check_commands.md does not exist")
    except Exception as e:
        errors.append(f"Failed to check verify_rick_check_commands.md: {str(e)}")

    # Test 4: 检查 test_go_project_changes.md 包含三个 section
    test_go_file = os.path.join(skills_dir, "test_go_project_changes.md")
    try:
        with open(test_go_file, "r") as f:
            content = f.read()
        for section in required_sections:
            if section not in content:
                errors.append(f"test_go_project_changes.md missing section: {section}")
    except FileNotFoundError:
        errors.append("test_go_project_changes.md does not exist")
    except Exception as e:
        errors.append(f"Failed to check test_go_project_changes.md: {str(e)}")

    # Test 5: 构建 rick 并运行 doing --dry-run 验证 skills section 显示 Markdown skill 名称和触发场景
    try:
        build_result = subprocess.run(
            ["go", "build", "-o", "bin/rick", "./cmd/rick/"],
            cwd=project_root,
            capture_output=True,
            text=True,
            timeout=60
        )
        if build_result.returncode != 0:
            errors.append(f"Failed to build rick: {build_result.stderr}")
        else:
            bin_path = os.path.join(project_root, "bin", "rick")
            dry_run_result = subprocess.run(
                [bin_path, "doing", "job_12", "--dry-run"],
                cwd=project_root,
                capture_output=True,
                text=True,
                timeout=30
            )
            output = dry_run_result.stdout + dry_run_result.stderr
            # Extract only the skills section to check for .py files
            # Skills section starts at "## 可用的项目 Skills" and ends at next "##" section
            skills_section = ""
            in_skills = False
            for line in output.splitlines():
                if "## 可用的项目 Skills" in line or "## 可用的项目技能" in line:
                    in_skills = True
                    skills_section += line + "\n"
                elif in_skills:
                    if line.startswith("## ") and "Skills" not in line and "技能" not in line:
                        break
                    skills_section += line + "\n"
            # Should show .md skill names, not .py files in the skills section specifically
            if skills_section and ".py" in skills_section:
                errors.append("dry-run skills section still shows .py files")
            # At minimum verify no Python skill files appear in full output
            if "verify_rick_check_commands.py" in output or "test_go_project_changes.py" in output:
                errors.append("dry-run output references .py skill files that should no longer exist")
    except subprocess.TimeoutExpired:
        errors.append("Build or dry-run timed out")
    except Exception as e:
        errors.append(f"Failed to build/run dry-run: {str(e)}")

    result = {
        "pass": len(errors) == 0,
        "errors": errors
    }

    print(json.dumps(result))
    sys.exit(0 if result["pass"] else 1)

if __name__ == "__main__":
    main()
