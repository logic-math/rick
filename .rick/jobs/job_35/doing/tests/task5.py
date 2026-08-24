#!/usr/bin/env python3
"""task5 验收测试：重构 builder 三件（templates + pibuilder + xxxxbuilder），注入路径而非内容。

覆盖测试方法中的三类断言（前置条件/输入/操作/预期四要素已核对）：

1. 正常路径：`go build -o <tmp>/rick ./cmd/rick` + `go test ./internal/builder/... ./internal/prompt/... -v`
   → build 成功、builder/prompt 测试全绿。

2. 边界（模板零改动 + 注入路径）：
   - `git diff --stat internal/prompt/templates/`（含 --cached）无 diff；
   - `<bin> plan --dry-run` 输出命中 `plan/task|doing/debug|/jobs/|/domain`（≥1 行），
     且无未替换 `{{...}}` 变量残留（注入的是真实路径，非 plan_dir 等变量名字面量）。

3. 异常（builder 缺参数）：`PIBuilder.BuildPlan("", nil)` 返回 error 含
   `requirement cannot be empty`（经临时 Go 程序调用真实方法验证）。

附加结构检查（对齐 task5.md 关键结果）：
- internal/builder 包存在 + xxxxbuilder.go 定义 RuntimeBuilder 接口（Name/BuildAgents/BuildPrompt）
  及 Method/AgentDef 类型；
- PIBuilder 类型 + BuildPlan/BuildDoing/BuildEasy/BuildHumanLoop/BuildCtrl/BuildDream/BuildLearning 方法；
- BuildPlan 签名 = (requirement string, params map[string]string) (method string, instance string, err error)；
- internal/prompt 保留 PromptBuilder/PromptManager/模板 embed 底层能力。

本脚本只读代码 + 跑 go build/go test + 临时 Go 程序（不修改源码，产物落临时目录），幂等；
仅向 stdout 输出一行 JSON。
"""
import json
import os
import re
import shutil
import subprocess
import sys
import tempfile


def find_repo_root(start_file):
    """定位仓库根目录（绝对路径）。

    优先向上查找 .git 标记；若不存在则回退到脚本相对路径：
    脚本位于 <root>/.rick/jobs/job_35/doing/tests/task5.py，向上 5 层即仓库根。
    """
    d = os.path.dirname(os.path.abspath(start_file))

    probe = d
    while True:
        if os.path.isdir(os.path.join(probe, '.git')):
            return probe
        parent = os.path.dirname(probe)
        if parent == probe:
            break
        probe = parent

    for _ in range(5):
        d = os.path.dirname(d)
    return d


def read_text(path):
    """读取文本文件内容，失败返回 None。"""
    try:
        with open(path, 'r', encoding='utf-8') as f:
            return f.read()
    except Exception:
        return None


def list_go_files(dirpath):
    """列出目录下（不含子目录）的 .go 文件绝对路径，目录不存在返回 []。"""
    if not os.path.isdir(dirpath):
        return []
    return sorted(
        os.path.join(dirpath, name)
        for name in os.listdir(dirpath)
        if name.endswith('.go')
    )


def concat_files(paths):
    """拼接多个文件内容为一个字符串，读取失败的文件记为 ''。"""
    parts = []
    for p in paths:
        txt = read_text(p)
        if txt is not None:
            parts.append(txt)
    return '\n'.join(parts)


def run(cmd, cwd, timeout=300, env=None):
    """运行子进程，返回 (returncode, stdout, stderr)。"""
    try:
        p = subprocess.run(
            cmd, cwd=cwd, capture_output=True, text=True, timeout=timeout, env=env
        )
        return p.returncode, p.stdout, p.stderr
    except FileNotFoundError as e:
        return 127, '', str(e)
    except subprocess.TimeoutExpired as e:
        out = e.stdout or ''
        err = (e.stderr or '') + '\n[timeout after %ds]' % timeout
        return 124, out, err


def tail(text, n=1600):
    """截取文本尾部，用于错误信息（避免刷屏）。"""
    if not text:
        return ''
    return text[-n:]


# PIBuilder.BuildPlan("") 异常路径的临时 Go 程序源码。
_PIBUILDER_EMPTY_GO = '''package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/sunquan/rick/internal/builder"
)

func main() {
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintf(os.Stderr, "PANIC: %v\\n", r)
			os.Exit(2)
		}
	}()

	var pb builder.PIBuilder
	_, _, err := pb.BuildPlan("", nil)
	if err == nil {
		fmt.Fprintln(os.Stderr, "FAIL: BuildPlan(\\"\\") returned nil error, want error")
		os.Exit(1)
	}
	if !strings.Contains(err.Error(), "requirement cannot be empty") {
		fmt.Fprintf(os.Stderr, "FAIL: error = %q, want contains %q\\n", err.Error(), "requirement cannot be empty")
		os.Exit(1)
	}
	fmt.Println("OK")
}
'''


def check_pibuilder_empty_requirement(repo_root, go):
    """通过临时 Go 程序调用真实的 PIBuilder.BuildPlan("", nil)，验证空 requirement 返回预期 error。

    临时目录放在仓库根下、以 '.' 前缀命名，既满足 internal 包导入的模块内约束，
    又不被 `go build ./...` / `go test ./...` 拾取。返回 (ok, message)。
    """
    tmpdir = tempfile.mkdtemp(prefix='.tmp_pibuilder_check_', dir=repo_root)
    try:
        main_go = os.path.join(tmpdir, 'main.go')
        with open(main_go, 'w', encoding='utf-8') as f:
            f.write(_PIBUILDER_EMPTY_GO)

        rc, out, err = run([go, 'run', '.'], cwd=tmpdir, timeout=240)
        if rc == 0 and 'OK' in out:
            return True, ''
        if rc == 2:
            return False, 'PIBuilder.BuildPlan("") panicked:\n' + tail(out or err)
        if rc == 1:
            return False, 'PIBuilder.BuildPlan("") did not return expected error:\n' + tail(err or out)
        # 编译失败（例如 internal/builder 尚不存在）等其它非零退出码
        return False, 'PIBuilder.BuildPlan("") check failed (exit %d):\n%s' % (rc, tail(err or out))
    finally:
        shutil.rmtree(tmpdir, ignore_errors=True)


def main():
    errors = []

    repo_root = find_repo_root(__file__)
    go = shutil.which('go')
    if not go:
        result = {'pass': False, 'errors': ['go toolchain not found in PATH']}
        print(json.dumps(result))
        sys.exit(1)

    internal_dir = os.path.join(repo_root, 'internal')
    builder_dir = os.path.join(internal_dir, 'builder')
    prompt_dir = os.path.join(internal_dir, 'prompt')
    templates_dir = os.path.join(prompt_dir, 'templates')
    prompt_builder_go = os.path.join(prompt_dir, 'builder.go')
    prompt_manager_go = os.path.join(prompt_dir, 'manager.go')

    # ---------- 正常路径 1：go build -o <tmp>/rick ./cmd/rick ----------
    build_dir = tempfile.mkdtemp(prefix='rick_task5_build_')
    build_out = os.path.join(build_dir, 'rick')
    rc, out, err = run([go, 'build', '-o', build_out, './cmd/rick'], cwd=repo_root, timeout=300)
    if rc != 0:
        errors.append('go build ./cmd/rick failed:\n' + tail(err or out))
    elif not os.path.isfile(build_out):
        errors.append('go build ./cmd/rick succeeded but binary not produced at %s' % build_out)

    # ---------- 正常路径 2：go test ./internal/builder/... ./internal/prompt/... -v ----------
    rc, out, err = run(
        [go, 'test', '-timeout', '180s', '-v', './internal/builder/...', './internal/prompt/...'],
        cwd=repo_root, timeout=240,
    )
    if rc != 0:
        errors.append('go test ./internal/builder/... ./internal/prompt/... failed:\n' + tail(err or out))

    # ---------- 边界 1：模板零改动（git diff 无 diff，含暂存区） ----------
    if not os.path.isdir(templates_dir):
        errors.append('internal/prompt/templates/ directory does not exist')
    else:
        rc, out, err = run(['git', 'diff', '--stat', '--', 'internal/prompt/templates/'], cwd=repo_root, timeout=60)
        if rc != 0:
            errors.append('git diff --stat internal/prompt/templates/ failed:\n' + tail(err or out))
        elif out.strip():
            errors.append('templates changed (unstaged):\n' + tail(out))
        rc2, out2, err2 = run(['git', 'diff', '--cached', '--stat', '--', 'internal/prompt/templates/'], cwd=repo_root, timeout=60)
        if rc2 != 0:
            errors.append('git diff --cached --stat internal/prompt/templates/ failed:\n' + tail(err2 or out2))
        elif out2.strip():
            errors.append('templates changed (staged):\n' + tail(out2))

    # ---------- 边界 2：注入路径（dry-run 输出命中真实路径片段 + 无未替换变量残留） ----------
    if os.path.isfile(build_out):
        rc, out, err = run([build_out, 'plan', '--dry-run'], cwd=repo_root, timeout=120)
        if rc != 0:
            errors.append('rick plan --dry-run failed (exit %d):\n%s' % (rc, tail(err or out)))
        else:
            pattern = re.compile(r'plan/task|doing/debug|/jobs/|/domain')
            matched_lines = sum(1 for line in out.splitlines() if pattern.search(line))
            if matched_lines < 1:
                errors.append(
                    'rick plan --dry-run output missing injected path fragments '
                    '(expected >=1 line matching plan/task|doing/debug|/jobs/|/domain)'
                )
            if '{{' in out:
                # 注入的是真实路径，而非未替换的 {{plan_dir}} 等变量名字面量
                errors.append('rick plan --dry-run output contains unsubstituted {{...}} placeholders')

    # ---------- 异常：PIBuilder.BuildPlan("") 返回 error 含 requirement cannot be empty ----------
    ok, msg = check_pibuilder_empty_requirement(repo_root, go)
    if not ok:
        errors.append(msg)

    # ---------- 结构：internal/builder 包存在 ----------
    builder_src_files = []
    builder_test_files = []
    builder_src_txt = ''
    if not os.path.isdir(builder_dir):
        errors.append('internal/builder directory does not exist')
    else:
        builder_files = list_go_files(builder_dir)
        builder_src_files = [f for f in builder_files if not f.endswith('_test.go')]
        builder_test_files = [f for f in builder_files if f.endswith('_test.go')]
        builder_src_txt = concat_files(builder_src_files)
        if not builder_src_files:
            errors.append('internal/builder has no non-test .go files')

    # ---------- 结构：xxxxbuilder.go 定义 RuntimeBuilder 接口 + Method/AgentDef ----------
    xxxxbuilder_go = os.path.join(builder_dir, 'xxxxbuilder.go')
    xxxx_txt = read_text(xxxxbuilder_go)
    if xxxx_txt is None:
        errors.append('internal/builder/xxxxbuilder.go does not exist (RuntimeBuilder 扩展位缺失)')
    else:
        for needle, label in (
            ('type RuntimeBuilder interface', 'type RuntimeBuilder interface'),
            ('Name() string', 'RuntimeBuilder.Name() string'),
            ('BuildAgents(method []Method) ([]AgentDef, error)', 'RuntimeBuilder.BuildAgents(method []Method) ([]AgentDef, error)'),
            ('BuildPrompt(cmd string, params map[string]string) (string, error)', 'RuntimeBuilder.BuildPrompt(cmd string, params map[string]string) (string, error)'),
        ):
            if needle not in xxxx_txt:
                errors.append('internal/builder/xxxxbuilder.go missing %s' % label)

    if builder_src_txt:
        if not re.search(r'\btype Method\b', builder_src_txt):
            errors.append('internal/builder missing type Method (BuildAgents 入参类型)')
        if not re.search(r'\btype AgentDef\b', builder_src_txt):
            errors.append('internal/builder missing type AgentDef (BuildAgents 返回类型)')

    # ---------- 结构：PIBuilder 类型 + 全部 Build* 方法 ----------
    if builder_src_txt:
        if not re.search(r'\btype PIBuilder\b', builder_src_txt):
            errors.append('internal/builder missing type PIBuilder')
        for meth in ('BuildPlan', 'BuildDoing', 'BuildEasy', 'BuildHumanLoop', 'BuildCtrl', 'BuildDream', 'BuildLearning'):
            if not re.search(r'\)\s*' + meth + r'\s*\(', builder_src_txt):
                errors.append('internal/builder PIBuilder missing method %s' % meth)

    # ---------- 结构：BuildPlan 签名 = (requirement string, params map[string]string) (method, instance, err) ----------
    if builder_src_txt:
        if 'BuildPlan(requirement string' not in builder_src_txt:
            errors.append('internal/builder BuildPlan missing "requirement string" first param')
        if '(method string, instance string, err error)' not in builder_src_txt:
            errors.append('internal/builder BuildPlan missing return signature (method string, instance string, err error)')

    # ---------- 结构：internal/prompt 保留 PromptBuilder/PromptManager/模板 embed 底层能力 ----------
    pb_txt = read_text(prompt_builder_go)
    if pb_txt is None:
        errors.append('internal/prompt/builder.go does not exist (PromptBuilder 底层能力应保留)')
    elif 'type PromptBuilder struct' not in pb_txt:
        errors.append('internal/prompt/builder.go missing type PromptBuilder')

    pm_txt = read_text(prompt_manager_go)
    if pm_txt is None:
        errors.append('internal/prompt/manager.go does not exist (PromptManager 底层能力应保留)')
    else:
        if 'type PromptManager struct' not in pm_txt:
            errors.append('internal/prompt/manager.go missing type PromptManager')
        if '//go:embed' not in pm_txt:
            errors.append('internal/prompt/manager.go missing //go:embed (模板 embed 应保留)')

    # 清理构建产物临时目录
    shutil.rmtree(build_dir, ignore_errors=True)

    result = {
        'pass': len(errors) == 0,
        'errors': errors,
    }

    # CRITICAL: 仅此一行输出到 stdout
    print(json.dumps(result))

    sys.exit(0 if result['pass'] else 1)


if __name__ == '__main__':
    main()
