#!/usr/bin/env python3
"""task10 验收测试：让 pibuilder 产出单文件内聚的 pi 定制化规范产物。

按 skill:tdd 四要素（前置条件/输入/操作/预期）实现测试方法三类断言，断言真实产物、
不 mock：用 `go build -o <tmp>/rick ./cmd/rick`（等价测试方法中的 `./bin/rick`，落临时
目录避免污染 bin/rick）跑真实 CLI，另用一个临时 Go 程序直接调用真实的
`PIBuilder.BuildPlan` 覆盖异常路径（删除内联源 grilling.md 后重建，验证返回 error 而非
panic/静默空产物）。

1. 正常路径：前置=task5 完成（build 成功）；输入=`plan --dry-run`；操作=运行后统计行数
   + 统计 `{{` 计数；预期=单个连贯 prompt（行数 > 0）且 `{{` 计数 == 0。

2. 边界（单文件内聚）：前置=build 成功；输入=`plan --dry-run`；操作=运行后统计
   `grep -cE 'Grilling|结构化追问'`；预期=≥1（grilling skill 内容内联进主产物，而非仅
   `skill_grilling.md` 路径引用）。

3. 异常（缺 skill 内联源）：前置=删除 `internal/prompt/templates/skills/grilling.md`
   后重新 `go build`（templates 由 go:embed 编译期内嵌，运行时改名不影响二进制，必须重建）；
   输入=运行 `PIBuilder.BuildPlan`；操作=调用检查返回；预期=返回 error 且消息含 grilling
   （非 panic、非静默产出空内容）。

脚本只读代码 + 跑 go build/临时 Go 程序，产物落临时目录，幂等；仅向 stdout 输出一行 JSON。
"""
import json
import os
import re
import shutil
import subprocess
import sys
import tempfile


# 临时 Go 程序：真实调用 builder.NewPIBuilder().BuildPlan(...)。
# 参数1 = rick_dir，参数2 = job_plan_dir；返回 error 时打印 `ERR:<msg>` 并正常退出（0），
# 成功时打印 `NO_ERROR`。这样 Python 侧可区分「正常返回 error」与「panic/崩溃（非 0 退出）」。
_PROBE_GO = '''package main

import (
	"fmt"
	"os"

	"github.com/sunquan/rick/internal/builder"
)

func main() {
	rickDir := "."
	jobPlanDir := ""
	if len(os.Args) > 1 {
		rickDir = os.Args[1]
	}
	if len(os.Args) > 2 {
		jobPlanDir = os.Args[2]
	}
	_, _, err := builder.NewPIBuilder().BuildPlan("dry-run requirement", map[string]string{
		"rick_dir":     rickDir,
		"job_plan_dir": jobPlanDir,
	})
	if err != nil {
		fmt.Printf("ERR:%v\\n", err)
		return
	}
	fmt.Println("NO_ERROR")
}
'''


def find_repo_root(start_file):
    """定位仓库根目录（绝对路径）。

    优先向上查找 .git 标记；若不存在则回退到脚本相对路径：
    脚本位于 <root>/.rick/jobs/job_35/doing/tests/task10.py，向上 5 层即仓库根。
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


def write_file(path, content):
    """写文本文件并自动创建父目录。"""
    os.makedirs(os.path.dirname(path), exist_ok=True)
    with open(path, 'w', encoding='utf-8') as f:
        f.write(content)


def read_bytes(path):
    """读取文件原始字节，失败返回 None。"""
    try:
        with open(path, 'rb') as f:
            return f.read()
    except Exception:
        return None


def write_bytes(path, content):
    """写原始字节并自动创建父目录。"""
    os.makedirs(os.path.dirname(path), exist_ok=True)
    with open(path, 'wb') as f:
        f.write(content)


def main():
    errors = []

    repo_root = find_repo_root(__file__)
    go = shutil.which('go')
    if not go:
        print(json.dumps({'pass': False, 'errors': ['go toolchain not found in PATH']}))
        sys.exit(1)

    # ================= 前置：build 成功（产物落临时目录，不污染 bin/rick） =================
    bin_dir = tempfile.mkdtemp(prefix='rick_task10_bin_')
    rick_bin = os.path.join(bin_dir, 'rick')
    rc, out, err = run([go, 'build', '-o', rick_bin, './cmd/rick'], cwd=repo_root, timeout=600)
    if rc != 0:
        errors.append('go build ./cmd/rick failed:\n' + tail(err or out))
        print(json.dumps({'pass': False, 'errors': errors}))
        sys.exit(1)

    # ================= 正常路径：单个连贯 prompt，行数 > 0，`{{` 计数 0 =================
    rc, out, err = run([rick_bin, 'plan', '--dry-run'], cwd=repo_root, timeout=120)
    if rc != 0:
        errors.append('rick plan --dry-run (正常路径) exit=%d:\n%s' % (rc, tail(err or out)))
    else:
        lines = len(out.splitlines())
        if lines <= 0:
            errors.append('plan --dry-run 输出行数 = %d, 应为 > 0（非单个连贯 prompt）' % lines)
        brace_count = out.count('{{')
        if brace_count != 0:
            errors.append('plan --dry-run 输出含 %d 个未替换 `{{` 占位符（应全部内联/替换）' % brace_count)

    # ================= 边界（单文件内聚）：grilling 内容内联进主产物 =================
    rc, out, err = run([rick_bin, 'plan', '--dry-run'], cwd=repo_root, timeout=120)
    if rc != 0:
        errors.append('rick plan --dry-run (边界) exit=%d:\n%s' % (rc, tail(err or out)))
    else:
        grilling_hits = len(re.findall(r'Grilling|结构化追问', out))
        if grilling_hits < 1:
            errors.append(
                'plan --dry-run 输出未内联 grilling 内容'
                '（grep "Grilling|结构化追问" 计数 = %d, 应 ≥ 1，而非仅路径引用）' % grilling_hits
            )

    # ================= 异常（缺 skill 内联源）：删 grilling.md + 重建 =================
    probe_dir = tempfile.mkdtemp(prefix='.tmp_task10_probe_', dir=repo_root)
    write_file(os.path.join(probe_dir, 'main.go'), _PROBE_GO)
    probe_bin = os.path.join(probe_dir, 'probe')
    grilling_path = os.path.join(
        repo_root, 'internal', 'prompt', 'templates', 'skills', 'grilling.md'
    )
    rick_dir = os.path.join(repo_root, '.rick')
    job_plan_dir = os.path.join(rick_dir, 'jobs', 'job_N', 'plan')

    # 控制运行：grilling 存在时 BuildPlan 应成功（验证 probe 与编译链路可用）。
    rc, out, err = run([go, 'build', '-o', probe_bin, '.'], cwd=probe_dir, timeout=600)
    if rc != 0:
        errors.append('probe go build (控制) failed:\n' + tail(err or out))
    else:
        rc, out, err = run([probe_bin, rick_dir, job_plan_dir], cwd=repo_root, timeout=120)
        if rc != 0:
            errors.append(
                'probe 控制运行 exit=%d（BuildPlan 在 grilling 存在时应成功，不 panic）:\n%s'
                % (rc, tail(err or out))
            )
        elif 'NO_ERROR' not in out:
            errors.append(
                'probe 控制运行未得到 NO_ERROR（BuildPlan 在 grilling 存在时应成功）:\n%s'
                % tail(out)
            )

    # 删除 grilling.md → 重新 go build → 运行 BuildPlan → 校验 error 含 grilling。
    backup = read_bytes(grilling_path)
    try:
        if backup is not None:
            os.remove(grilling_path)
        rc, out, err = run([go, 'build', '-o', probe_bin, '.'], cwd=probe_dir, timeout=600)
        if rc != 0:
            errors.append('probe go build (缺 grilling) failed:\n' + tail(err or out))
        else:
            rc, out, err = run([probe_bin, rick_dir, job_plan_dir], cwd=repo_root, timeout=120)
            if rc != 0:
                errors.append(
                    'BuildPlan 在缺 grilling 时崩溃（exit=%d），应返回 error 而非 panic:\n%s'
                    % (rc, tail(err or out))
                )
            elif 'ERR:' not in out:
                errors.append(
                    'BuildPlan 在缺 grilling 时未返回 error（静默产出/成功），'
                    '应返回含 grilling 的 error:\n%s' % tail(out)
                )
            elif 'grilling' not in out.lower():
                errors.append('BuildPlan 返回的 error 消息未含 grilling:\n%s' % tail(out))
    finally:
        if backup is not None:
            write_bytes(grilling_path, backup)

    # 清理临时目录（幂等：不污染仓库）。
    shutil.rmtree(probe_dir, ignore_errors=True)
    shutil.rmtree(bin_dir, ignore_errors=True)

    result = {
        'pass': len(errors) == 0,
        'errors': errors,
    }

    # CRITICAL: 仅此一行输出到 stdout
    print(json.dumps(result))

    sys.exit(0 if result['pass'] else 1)


if __name__ == '__main__':
    main()
