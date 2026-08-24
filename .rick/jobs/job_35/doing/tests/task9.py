#!/usr/bin/env python3
"""task9 验收测试：注册 think/research/exporter 为 pi 自定义 agent（经 env 职责 3 落盘）。

按 skill:tdd 四要素（前置条件/输入/操作/预期）实现测试方法三类断言，断言真实落盘文件、
不 mock：用一个临时 Go 程序直接调用真实的 `env.DeployRickCustomizations()`（隔离
RICK_PI_AGENT_DIR 指向 temp、程序内 chdir 到仓库根使 `.rick/skills` 源可见）。

1. 正常路径：前置=task3/5 完成 + RICK_PI_AGENT_DIR 指向 temp；输入=运行 DeployRickCustomizations；
   操作=运行 + `test -f agents/{think,research,exporter}.md` + 解析 agents/ 下 frontmatter 的
   `name` 字段（等价于 pi 真实发现入口 `{action:"list"}` 所列 agent 名）；预期=3 文件存在、
   `head -5 think.md` 含 `name: think`、发现清单 = {think,research,exporter}，且 frontmatter 含
   name/description/tools/defaultContext + rick-managed: true 标记、tools 对齐 pi 工具名。

2. 边界（幂等 + 覆盖语义）：前置=已注册一次；输入=再次运行；操作=再次运行 + sha256 对比前后；
   预期=内容不变（幂等）；另预置无 rick-managed: true 的同名文件 → 运行后不被覆盖（仅覆盖有
   标记的文件）；反向预置含 rick-managed: true 的陈旧文件 → 运行后被覆盖为最新 wiki 正文。

3. 异常（system prompt 非空）：前置=3 文件已写；输入=无；操作=解析 frontmatter 闭合后的正文，
   等价 `awk '/^---$/{n++} n>=2 && /[^[:space:]]/{print}'` 统计非空行；预期=每文件 ≥1 且含
   对应 skill wiki 标记（skill:think / skill:research / skill:exporter）。

脚本只读代码 + 跑 go run 临时程序 + go test ./internal/env/...，产物落临时目录，幂等；
仅向 stdout 输出一行 JSON。
"""
import hashlib
import json
import os
import shutil
import subprocess
import sys
import tempfile


# 3 个 agent 的期望规格：key = agent 名（也是文件名 {key}.md）。
AGENTS = {
    'think': {
        'tools': ['read', 'grep', 'find', 'ls'],
        'skill_marker': 'skill:think',
    },
    'research': {
        'tools': ['read', 'grep', 'find', 'ls', 'bash', 'web_search', 'fetch_content'],
        'skill_marker': 'skill:research',
    },
    'exporter': {
        'tools': ['read', 'write', 'bash'],
        'skill_marker': 'skill:exporter',
    },
}


# 临时 Go 程序：真实调用 env.DeployRickCustomizations()。第一个参数为运行期 cwd
# （chdir 后 workspace.GetRickDir() 才能定位到仓库的 .rick/skills 源目录，与 init-pi
# 从仓库根运行一致）。
_DEPLOY_GO = '''package main

import (
	"fmt"
	"os"

	"github.com/sunquan/rick/internal/env"
)

func main() {
	target := "."
	if len(os.Args) > 1 {
		target = os.Args[1]
	}
	if err := os.Chdir(target); err != nil {
		fmt.Fprintf(os.Stderr, "chdir %s failed: %v\\n", target, err)
		os.Exit(2)
	}
	if err := env.DeployRickCustomizations(); err != nil {
		fmt.Fprintf(os.Stderr, "deploy failed: %v\\n", err)
		os.Exit(1)
	}
	fmt.Println("deploy-ok")
}
'''


def find_repo_root(start_file):
    """定位仓库根目录（绝对路径）。

    优先向上查找 .git 标记；若不存在则回退到脚本相对路径：
    脚本位于 <root>/.rick/jobs/job_35/doing/tests/task9.py，向上 5 层即仓库根。
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


def read_text(path):
    """读取文本文件内容，失败返回 None。"""
    try:
        with open(path, 'r', encoding='utf-8') as f:
            return f.read()
    except Exception:
        return None


def write_file(path, content):
    """写文件并自动创建父目录。"""
    os.makedirs(os.path.dirname(path), exist_ok=True)
    with open(path, 'w', encoding='utf-8') as f:
        f.write(content)


def sha256_file(path):
    """计算文件 sha256（十六进制），失败返回 None。"""
    h = hashlib.sha256()
    try:
        with open(path, 'rb') as f:
            for chunk in iter(lambda: f.read(65536), b''):
                h.update(chunk)
        return h.hexdigest()
    except Exception:
        return None


def parse_frontmatter(content):
    """解析 agent 文件，返回 (frontmatter dict, body 字符串, error)。

    frontmatter 由首行 `---` 与随后的第一个 `---` 行闭合；body 为闭合之后全部内容。
    （skill wiki 正文内部出现的 `---` 水平线属于 body，不影响解析。）
    """
    if content is None:
        return None, None, 'file not readable'
    lines = content.split('\n')
    if not lines or lines[0].strip() != '---':
        return None, None, 'missing opening ---'
    end = None
    for i in range(1, len(lines)):
        if lines[i].strip() == '---':
            end = i
            break
    if end is None:
        return None, None, 'missing closing ---'
    fm = {}
    for ln in lines[1:end]:
        ln = ln.strip()
        if not ln or ln.startswith('#'):
            continue
        if ':' in ln:
            k, v = ln.split(':', 1)
            fm[k.strip()] = v.strip()
    body = '\n'.join(lines[end + 1:])
    return fm, body, None


def count_body_nonblank(body):
    """等价 awk '/^---$/{n++} n>=2 && /[^[:space:]]/{print}' 的非空正文行数。"""
    if not body:
        return 0
    return sum(1 for ln in body.split('\n') if ln.strip())


def discover_agent_names(agents_dir):
    """从 agents 目录解析各 .md 的 frontmatter name 字段（等价于 pi {action:"list"} 列名）。"""
    names = set()
    if not os.path.isdir(agents_dir):
        return names
    for fn in sorted(os.listdir(agents_dir)):
        if not fn.endswith('.md'):
            continue
        content = read_text(os.path.join(agents_dir, fn))
        fm, _, err = parse_frontmatter(content)
        if err is None and fm.get('name'):
            names.add(fm['name'])
    return names


def write_deploy_program(repo_root):
    """在仓库根下建一个 `.tmp_task9_*` 目录写入临时 Go 程序，返回该目录路径。

    `.` 前缀目录不会被 `go build ./...` / `go test ./...` 拾取；位于模块内方可导入
    internal 包。运行期 cwd 由程序参数控制（见 _DEPLOY_GO）。
    """
    tmpdir = tempfile.mkdtemp(prefix='.tmp_task9_deploy_', dir=repo_root)
    write_file(os.path.join(tmpdir, 'main.go'), _DEPLOY_GO)
    return tmpdir


def run_deploy(go, prog_dir, target_cwd, agent_dir, home):
    """运行临时 Go 程序调用 DeployRickCustomizations，返回 (rc, out, err)。"""
    env = os.environ.copy()
    env['RICK_PI_AGENT_DIR'] = agent_dir
    env['HOME'] = home
    return run([go, 'run', '.', target_cwd], cwd=prog_dir, timeout=240, env=env)


def main():
    errors = []

    repo_root = find_repo_root(__file__)
    go = shutil.which('go')
    if not go:
        result = {'pass': False, 'errors': ['go toolchain not found in PATH']}
        print(json.dumps(result))
        sys.exit(1)

    prog_dir = write_deploy_program(repo_root)

    # ================= 正常路径 =================
    agent_dir = tempfile.mkdtemp(prefix='rick_task9_agent_')
    home = tempfile.mkdtemp(prefix='rick_task9_home_')
    rc, out, err = run_deploy(go, prog_dir, repo_root, agent_dir, home)
    if rc != 0:
        errors.append('DeployRickCustomizations (正常路径) exit=%d:\n%s' % (rc, tail(err or out)))
    else:
        # 3 文件存在 + frontmatter 字段校验。
        for name in AGENTS:
            p = os.path.join(agent_dir, 'agents', name + '.md')
            if not os.path.isfile(p):
                errors.append('%s.md 未落盘: %s' % (name, p))
                continue
            content = read_text(p)
            fm, body, ferr = parse_frontmatter(content)
            if ferr:
                errors.append('%s.md frontmatter 解析失败: %s' % (name, ferr))
                continue
            if fm.get('name') != name:
                errors.append('%s.md frontmatter name=%r, want %r' % (name, fm.get('name'), name))
            got_tools = sorted(t.strip() for t in fm.get('tools', '').split(',') if t.strip())
            want_tools = sorted(AGENTS[name]['tools'])
            if got_tools != want_tools:
                errors.append('%s.md frontmatter tools=%r, want %r' % (name, got_tools, want_tools))
            if not fm.get('description'):
                errors.append('%s.md frontmatter 缺非空 description' % name)
            if not fm.get('defaultContext'):
                errors.append('%s.md frontmatter 缺非空 defaultContext' % name)
            if fm.get('rick-managed') != 'true':
                errors.append('%s.md frontmatter 缺 rick-managed: true 标记' % name)

        # `head -5 think.md` 含 `name: think`。
        think_path = os.path.join(agent_dir, 'agents', 'think.md')
        think_content = read_text(think_path)
        if think_content is None:
            errors.append('think.md 无法读取（head -5 检查）')
        else:
            head5 = '\n'.join(think_content.split('\n')[:5])
            if 'name: think' not in head5:
                errors.append('head -5 think.md 不含 `name: think`')

        # 真实发现入口（等价：agents/ 下 frontmatter name 字段集合）。
        discovered = discover_agent_names(os.path.join(agent_dir, 'agents'))
        if discovered != set(AGENTS.keys()):
            errors.append('agent 发现清单 = %s, want %s'
                          % (sorted(discovered), sorted(AGENTS.keys())))

        # ================= 边界：幂等 =================
        before = {}
        for name in AGENTS:
            p = os.path.join(agent_dir, 'agents', name + '.md')
            if os.path.isfile(p):
                before[name] = sha256_file(p)
        rc2, out2, err2 = run_deploy(go, prog_dir, repo_root, agent_dir, home)
        if rc2 != 0:
            errors.append('DeployRickCustomizations (幂等重跑) exit=%d:\n%s' % (rc2, tail(err2 or out2)))
        else:
            changed = []
            for name, digest in before.items():
                p = os.path.join(agent_dir, 'agents', name + '.md')
                if not os.path.isfile(p):
                    changed.append('%s(file missing)' % name)
                elif sha256_file(p) != digest:
                    changed.append(name)
            if changed:
                errors.append('幂等重跑后文件内容变化（不幂等）: %s' % ', '.join(changed))

    # ================= 边界：覆盖语义（无标记不覆盖） =================
    ow_dir = tempfile.mkdtemp(prefix='rick_task9_ow_')
    ow_home = tempfile.mkdtemp(prefix='rick_task9_owhome_')
    think_ow = os.path.join(ow_dir, 'agents', 'think.md')
    user_content = '---\nname: think\ndescription: user custom agent\ntools: read\n---\nUSER CUSTOM BODY\n'
    write_file(think_ow, user_content)
    rc, out, err = run_deploy(go, prog_dir, repo_root, ow_dir, ow_home)
    if rc != 0:
        errors.append('DeployRickCustomizations (覆盖语义-无标记) exit=%d:\n%s' % (rc, tail(err or out)))
    else:
        after_content = read_text(think_ow)
        if after_content != user_content:
            errors.append('无 rick-managed: true 标记的 think.md 被覆盖（应跳过不覆盖）')
        for name in ('research', 'exporter'):
            p = os.path.join(ow_dir, 'agents', name + '.md')
            if not os.path.isfile(p):
                errors.append('覆盖语义（无标记）场景下 %s.md 未落盘' % name)

    # ================= 边界：覆盖语义（有标记才覆盖） =================
    mk_dir = tempfile.mkdtemp(prefix='rick_task9_mk_')
    mk_home = tempfile.mkdtemp(prefix='rick_task9_mkhome_')
    research_mk = os.path.join(mk_dir, 'agents', 'research.md')
    stale = ('---\nname: research\ndescription: stale rick agent\ntools: read\n'
             'defaultContext: fresh\nrick-managed: true\n---\nSTALE BODY\n')
    write_file(research_mk, stale)
    rc, out, err = run_deploy(go, prog_dir, repo_root, mk_dir, mk_home)
    if rc != 0:
        errors.append('DeployRickCustomizations (覆盖语义-有标记) exit=%d:\n%s' % (rc, tail(err or out)))
    else:
        after_content = read_text(research_mk)
        if after_content == stale:
            errors.append('含 rick-managed: true 的 research.md 未被覆盖（应覆盖）')
        elif 'STALE BODY' in (after_content or ''):
            errors.append('含 rick-managed: true 的 research.md 覆盖后仍含 STALE BODY')
        elif 'skill:research' not in (after_content or ''):
            errors.append('覆盖后 research.md 缺 skill:research 正文（覆盖内容不对）')

    # ================= 异常：system prompt 非空 =================
    ex_dir = tempfile.mkdtemp(prefix='rick_task9_ex_')
    ex_home = tempfile.mkdtemp(prefix='rick_task9_exhome_')
    rc, out, err = run_deploy(go, prog_dir, repo_root, ex_dir, ex_home)
    if rc != 0:
        errors.append('DeployRickCustomizations (异常阶段) exit=%d:\n%s' % (rc, tail(err or out)))
    else:
        for name in AGENTS:
            p = os.path.join(ex_dir, 'agents', name + '.md')
            if not os.path.isfile(p):
                errors.append('异常阶段 %s.md 未落盘' % name)
                continue
            content = read_text(p)
            fm, body, ferr = parse_frontmatter(content)
            if ferr:
                errors.append('异常阶段 %s.md frontmatter 解析失败: %s' % (name, ferr))
                continue
            if count_body_nonblank(body) < 1:
                errors.append('%s.md frontmatter 闭合后无非空正文（system prompt 为空）' % name)
            if AGENTS[name]['skill_marker'] not in (body or ''):
                errors.append('%s.md system prompt 缺 %s（wiki 内容未注入）'
                              % (name, AGENTS[name]['skill_marker']))

    # ================= 结构/编译：go test ./internal/env/... =================
    rc, out, err = run([go, 'test', '-timeout', '120s', './internal/env/...'],
                       cwd=repo_root, timeout=180)
    if rc != 0:
        errors.append('go test ./internal/env/... failed:\n' + tail(err or out))

    # 清理临时 Go 程序目录（agent_dir/home 等系统临时目录由 tempfile 管理）。
    shutil.rmtree(prog_dir, ignore_errors=True)

    result = {
        'pass': len(errors) == 0,
        'errors': errors,
    }

    # CRITICAL: 仅此一行输出到 stdout
    print(json.dumps(result))

    sys.exit(0 if result['pass'] else 1)


if __name__ == '__main__':
    main()
