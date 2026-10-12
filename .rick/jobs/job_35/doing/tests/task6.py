#!/usr/bin/env python3
"""task6 验收测试：落地 handler 调度聚合层并让 cli 变薄。

覆盖测试方法的三类断言（前置条件/输入/操作/预期四要素已核对）：

1. 正常路径：`go build -o bin/rick ./cmd/rick` + `go test ./internal/cmd/... ./internal/handler/... -timeout 60s -v`
   → build 成功、cmd/handler 测试全绿。

2. 边界（dry-run 变量替换 + 空 requirement）：
   - `rick plan --dry-run` 输出中无未替换的 `{{` 模板变量（grep -c '{{' 预期 0）；
   - `rick easy --dry-run -r ""` 报 `requirement cannot be empty`（非 0 退出）。

3. 异常（无效 job + 重入不存在 plan + --ctx 冲突）：
   - `rick doing job_nonexistent` 报 `job directory not found`，非 0 退出；
   - `rick plan --job job_nonexistent` 报 `plan directory does not exist`，非 0 退出；
   - `rick easy --ctx <已有 loops 的 .rick> -r "test requirement"` 报 `local context already exists`，非 0 退出。

附加结构检查（对齐 task6 关键结果）：
- internal/handler 包存在且含非 test 的 .go 文件；
- handler 不得 import internal/cmd（避免跨包循环依赖）；
- handler 定义 9 个显式迁移函数：Plan/ReEnterPlan/PlanDryRun/Doing/DoingDryRun/
  Easy/ResumeEasy/StartEasySession/EasyDryRun。

本脚本只读源码 + 跑 go build/go test + 跑 CLI（产物落临时目录，不污染仓库 bin/rick），幂等；
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

    优先向上查找 go.mod（Go 项目根标记）；找不到则回退为脚本相对位置向上 5 层：
    脚本位于 <root>/.rick/jobs/job_35/doing/tests/task6.py，向上 5 层即仓库根。
    """
    d = os.path.dirname(os.path.abspath(start_file))
    probe = d
    while True:
        if os.path.isfile(os.path.join(probe, 'go.mod')):
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


def run(cmd, cwd, timeout=300, env=None):
    """运行子进程，返回 (returncode, stdout, stderr)。"""
    try:
        p = subprocess.run(
            cmd, cwd=cwd, capture_output=True, text=True, timeout=timeout, env=env
        )
        return p.returncode, p.stdout or '', p.stderr or ''
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


def main():
    errors = []

    repo_root = find_repo_root(__file__)
    go = shutil.which('go')
    if not go:
        result = {'pass': False, 'errors': ['go toolchain not found in PATH']}
        print(json.dumps(result))
        sys.exit(1)

    # ---------- 正常路径 1：go build -o <tmp>/rick ./cmd/rick ----------
    # 构建产物落临时目录，避免污染仓库中被 git 跟踪的 bin/rick。
    build_dir = tempfile.mkdtemp(prefix='rick_task6_build_')
    bin_path = os.path.join(build_dir, 'rick')
    rc, out, err = run([go, 'build', '-o', bin_path, './cmd/rick'], cwd=repo_root, timeout=300)
    if rc != 0:
        errors.append('go build ./cmd/rick failed:\n' + tail(err or out))
    elif not os.path.isfile(bin_path):
        errors.append('go build ./cmd/rick succeeded but binary not produced at %s' % bin_path)

    # ---------- 正常路径 2：go test ./internal/cmd/... ./internal/handler/... -timeout 60s -v ----------
    rc, out, err = run(
        [go, 'test', './internal/cmd/...', './internal/handler/...', '-timeout', '60s', '-v'],
        cwd=repo_root, timeout=300,
    )
    if rc != 0:
        errors.append('go test ./internal/cmd/... ./internal/handler/... failed:\n' + tail(err or out))

    # ---------- 边界 1：plan --dry-run 无未替换 '{{' 变量 ----------
    if os.path.isfile(bin_path):
        rc, out, err = run([bin_path, 'plan', '--dry-run'], cwd=repo_root, timeout=120)
        if rc != 0:
            errors.append('rick plan --dry-run failed (exit %d):\n%s' % (rc, tail(err or out)))
        else:
            brace_count = out.count('{{')
            if brace_count != 0:
                errors.append(
                    "rick plan --dry-run output contains %d unsubstituted '{{' placeholder(s)"
                    % brace_count
                )
    else:
        errors.append('rick binary missing, skipping plan --dry-run check')

    # ---------- 边界 2：easy --dry-run -r "" 报 requirement cannot be empty ----------
    if os.path.isfile(bin_path):
        rc, out, err = run([bin_path, 'easy', '--dry-run', '-r', ''], cwd=repo_root, timeout=120)
        combined = out + err
        if 'requirement cannot be empty' not in combined:
            errors.append(
                "rick easy --dry-run -r \"\" did not report 'requirement cannot be empty' "
                '(exit %d)' % rc
            )
        elif rc == 0:
            errors.append(
                "rick easy --dry-run -r \"\" reported empty requirement but exited 0 "
                '(expected non-zero)'
            )
    else:
        errors.append('rick binary missing, skipping easy --dry-run -r "" check')

    # ---------- 异常 1：doing job_nonexistent 报 job directory not found ----------
    if os.path.isfile(bin_path):
        rc, out, err = run([bin_path, 'doing', 'job_nonexistent'], cwd=repo_root, timeout=120)
        combined = out + err
        if 'job directory not found' not in combined:
            errors.append(
                "rick doing job_nonexistent did not report 'job directory not found' (exit %d)" % rc
            )
        elif rc == 0:
            errors.append('rick doing job_nonexistent exited 0 (expected non-zero)')
    else:
        errors.append('rick binary missing, skipping doing job_nonexistent check')

    # ---------- 异常 2：plan --job job_nonexistent 报 plan directory does not exist ----------
    if os.path.isfile(bin_path):
        rc, out, err = run([bin_path, 'plan', '--job', 'job_nonexistent'], cwd=repo_root, timeout=120)
        combined = out + err
        if 'plan directory does not exist' not in combined:
            errors.append(
                "rick plan --job job_nonexistent did not report 'plan directory does not exist' "
                '(exit %d)' % rc
            )
        elif rc == 0:
            errors.append('rick plan --job job_nonexistent exited 0 (expected non-zero)')
    else:
        errors.append('rick binary missing, skipping plan --job job_nonexistent check')

    # ---------- 异常 3：easy --ctx <已有 loops 的 .rick> 报 local context already exists ----------
    # 附带 -r 以满足 requirement（否则会先进入交互式 prompt 而拿不到 ctx 冲突错误）。
    ctx_path = os.path.join(repo_root, '.rick')
    if os.path.isfile(bin_path):
        rc, out, err = run(
            [bin_path, 'easy', '--ctx', ctx_path, '-r', 'test requirement'],
            cwd=repo_root, timeout=120,
        )
        combined = out + err
        if 'local context already exists' not in combined:
            errors.append(
                "rick easy --ctx did not report 'local context already exists' (exit %d)" % rc
            )
        elif rc == 0:
            errors.append('rick easy --ctx exited 0 (expected non-zero)')
    else:
        errors.append('rick binary missing, skipping easy --ctx check')

    # ---------- 结构：internal/handler 包存在 + 非 test .go 文件 ----------
    handler_dir = os.path.join(repo_root, 'internal', 'handler')
    handler_src_txt = ''
    if not os.path.isdir(handler_dir):
        errors.append('internal/handler directory does not exist')
    else:
        handler_files = list_go_files(handler_dir)
        src_files = [f for f in handler_files if not f.endswith('_test.go')]
        handler_src_txt = '\n'.join(
            txt for txt in (read_text(f) for f in src_files) if txt is not None
        )
        if not src_files:
            errors.append('internal/handler has no non-test .go files')

    # ---------- 结构：handler 不得 import internal/cmd（跨包循环依赖） ----------
    if handler_src_txt:
        if '"github.com/sunquan/rick/internal/cmd"' in handler_src_txt:
            errors.append(
                'internal/handler must not import internal/cmd (cross-package cycle); '
                'flag values must be passed as parameters (Options{...})'
            )

    # ---------- 结构：handler 定义 9 个显式迁移函数 ----------
    if handler_src_txt:
        missing_funcs = []
        for fn in ('Plan', 'ReEnterPlan', 'PlanDryRun', 'Doing', 'DoingDryRun',
                   'Easy', 'ResumeEasy', 'StartEasySession', 'EasyDryRun'):
            if not re.search(r'\bfunc\s+' + fn + r'\s*\(', handler_src_txt):
                missing_funcs.append(fn)
        if missing_funcs:
            errors.append(
                'internal/handler missing migrated functions: %s' % ', '.join(missing_funcs)
            )

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
