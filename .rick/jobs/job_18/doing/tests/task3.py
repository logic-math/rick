#!/usr/bin/env python3
# Description: 验收 task3 — easy.md + easy_prompt.go 注入 grilling skill + requirement.md 追加指令
import json
import sys
import os
import subprocess

def main():
    errors = []

    script_dir = os.path.dirname(os.path.abspath(__file__))
    # tests/ → doing/ → job_18/ → jobs/ → .rick/ → rick/
    project_root = os.path.dirname(
        os.path.dirname(
            os.path.dirname(
                os.path.dirname(
                    os.path.dirname(script_dir)
                )
            )
        )
    )
    print(f"project_root: {project_root}", file=sys.stderr)

    easy_md = os.path.join(project_root, "internal", "prompt", "templates", "easy.md")
    easy_prompt_go = os.path.join(project_root, "internal", "prompt", "easy_prompt.go")

    # ── Test 1: easy.md 包含 {{grilling_skill_path}} 变量 ──────────────────────
    try:
        with open(easy_md, "r", encoding="utf-8") as f:
            easy_content = f.read()
        if "{{grilling_skill_path}}" not in easy_content:
            errors.append("easy.md 缺少 {{grilling_skill_path}} 变量")
    except Exception as e:
        errors.append(f"读取 easy.md 失败: {str(e)}")
        easy_content = ""

    # ── Test 2: easy.md 包含"追加"指令（requirement.md 追加，非覆写） ──────────
    if easy_content:
        if "追加" not in easy_content:
            errors.append("easy.md 缺少追加指令（未找到\"追加\"字样），应指引 agent 将 grilling 澄清结论追加到 requirement.md")
        if "覆写" in easy_content or "重写" in easy_content:
            errors.append("easy.md 含有覆写/重写字样，违反追加原则（不得覆盖原始用户输入）")

    # ── Test 3: easy_prompt.go 写出 skill_grilling.md ──────────────────────────
    try:
        with open(easy_prompt_go, "r", encoding="utf-8") as f:
            go_content = f.read()

        if 'skill_grilling.md' not in go_content:
            errors.append('easy_prompt.go 缺少 WriteSkillFile 写出 skill_grilling.md 的调用')

        if 'grilling_skill_path' not in go_content:
            errors.append('easy_prompt.go 缺少 SetVariable("grilling_skill_path", ...)')

        if 'grillingFile' not in go_content:
            errors.append('easy_prompt.go 缺少 grillingFile 变量')
    except Exception as e:
        errors.append(f"读取 easy_prompt.go 失败: {str(e)}")
        go_content = ""

    # ── Test 4: grillingFile 被加入 skillFiles 切片 ────────────────────────────
    if go_content:
        # skillFiles = []string{tddFile, debugSkillFile, senseFile, grillingFile} 或类似
        if "grillingFile" in go_content:
            # 找到 skillFiles 赋值行，确认包含 grillingFile
            import re
            skill_files_match = re.search(
                r'skillFiles\s*:?=\s*\[\]string\{([^}]+)\}',
                go_content
            )
            if skill_files_match:
                slice_content = skill_files_match.group(1)
                if "grillingFile" not in slice_content:
                    errors.append(
                        "easy_prompt.go 的 skillFiles 切片未包含 grillingFile"
                    )
            else:
                errors.append(
                    "easy_prompt.go 未找到 skillFiles := []string{...} 赋值（无法确认 grillingFile 已加入）"
                )

    # ── Test 5: build 编译通过 ──────────────────────────────────────────────────
    build_tool = os.path.join(project_root, ".rick", "tools", "build_and_get_rick_bin.py")
    bin_path = ""
    try:
        build_result = subprocess.run(
            ["python3", build_tool],
            capture_output=True, text=True, cwd=project_root, timeout=120
        )
        if build_result.returncode != 0:
            errors.append(f"build 失败:\n{build_result.stderr.strip()}")
        else:
            bin_path = build_result.stdout.strip()
            print(f"bin_path: {bin_path}", file=sys.stderr)
    except Exception as e:
        errors.append(f"运行 build_and_get_rick_bin.py 失败: {str(e)}")

    # ── Test 6: go test ./internal/prompt/... 全部通过 ─────────────────────────
    try:
        test_result = subprocess.run(
            ["go", "test", "-timeout", "60s", "-v", "./internal/prompt/..."],
            capture_output=True, text=True, cwd=project_root, timeout=120
        )
        if test_result.returncode != 0:
            errors.append(
                f"go test ./internal/prompt/... 失败:\n"
                f"{test_result.stdout[-2000:]}\n{test_result.stderr[-1000:]}"
            )
        else:
            # 确保有 grilling 相关测试被执行
            combined = test_result.stdout + test_result.stderr
            if "Grilling" not in combined and "grilling" not in combined:
                print("warn: go test 输出中未见 grilling 相关测试，请确认已添加验收测试", file=sys.stderr)
    except Exception as e:
        errors.append(f"运行 go test 失败: {str(e)}")

    # ── Test 7: GenerateEasyPromptFile 运行时生成 skill_grilling.md ────────────
    # 通过写一个临时 Go test 文件验证运行时行为
    import tempfile
    try:
        # 构造一个最小化测试文件，放到 internal/prompt 包下
        test_src = '''\
package prompt

import (
\t"os"
\t"path/filepath"
\t"testing"
)

func TestTask3_EasyPromptGeneratesGrillingSkill(t *testing.T) {
\tdir := t.TempDir()
\trick_dir := filepath.Join(dir, ".rick")
\tos.MkdirAll(filepath.Join(rick_dir, "jobs", "t3job", "doing", "prompts"), 0755)

\tmainFile, skillFiles, err := GenerateEasyPromptFile("t3job", "需求文本", rick_dir, "")
\tif err != nil {
\t\tt.Fatalf("GenerateEasyPromptFile error: %v", err)
\t}

\tpromptsDir := filepath.Dir(mainFile)

\t// skill_grilling.md 必须写出到 prompts dir
\tgrillingPath := filepath.Join(promptsDir, "skill_grilling.md")
\tif _, err := os.Stat(grillingPath); os.IsNotExist(err) {
\t\tt.Error("skill_grilling.md 未写出到 prompts dir")
\t}

\t// skillFiles 必须包含 grillingFile
\tfound := false
\tfor _, f := range skillFiles {
\t\tif filepath.Base(f) == "skill_grilling.md" {
\t\t\tfound = true
\t\t\tbreak
\t\t}
\t}
\tif !found {
\t\tt.Errorf("skillFiles 未包含 skill_grilling.md，实际值: %v", skillFiles)
\t}

\t// easy_prompt.md 不得含未替换的占位符
\tdata, _ := os.ReadFile(mainFile)
\tcontent := string(data)
\tif contains(content, "{{grilling_skill_path}}") {
\t\tt.Error("easy_prompt.md 仍含未替换的 {{grilling_skill_path}}")
\t}
\tif !contains(content, "skill_grilling.md") {
\t\tt.Error("easy_prompt.md 未引用 skill_grilling.md")
\t}
}

func contains(s, sub string) bool {
\treturn len(s) >= len(sub) && (s == sub || len(s) > 0 && containsStr(s, sub))
}

func containsStr(s, sub string) bool {
\tfor i := 0; i <= len(s)-len(sub); i++ {
\t\tif s[i:i+len(sub)] == sub {
\t\t\treturn true
\t\t}
\t}
\treturn false
}
'''
        tmp_test_file = os.path.join(
            project_root, "internal", "prompt", "task3_acceptance_test.go"
        )
        with open(tmp_test_file, "w", encoding="utf-8") as f:
            f.write(test_src)

        run_test = subprocess.run(
            ["go", "test", "-timeout", "30s", "-run",
             "TestTask3_EasyPromptGeneratesGrillingSkill",
             "./internal/prompt/..."],
            capture_output=True, text=True, cwd=project_root, timeout=60
        )
        if run_test.returncode != 0:
            errors.append(
                f"TestTask3_EasyPromptGeneratesGrillingSkill 失败:\n"
                f"{run_test.stdout}\n{run_test.stderr}"
            )
    except Exception as e:
        errors.append(f"运行时 grilling 生成验证失败: {str(e)}")
    finally:
        # 删除临时测试文件
        try:
            os.remove(tmp_test_file)
        except Exception:
            pass

    result = {
        "pass": len(errors) == 0,
        "errors": errors
    }
    print(json.dumps(result, ensure_ascii=False))
    sys.exit(0 if result["pass"] else 1)


if __name__ == "__main__":
    main()
