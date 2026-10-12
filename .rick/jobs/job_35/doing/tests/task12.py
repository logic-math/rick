#!/usr/bin/env python3
r"""task12 测试脚本：三个 O 端到端验收 + 全命令可用 + 回滚兜底。

按 skill:tdd 四要素（前置条件 / 输入 / 操作 / 预期）实现测试方法三类路径，
断言真实文件与真实命令行为、不 mock：

1. 正常路径（三 O 验收）：
   - O1（spec 信息内核）：`.rick/domain/spec.md` + `.rick/domain/rick-spec.md` 均存在，
     且都含四要素关键词（模块边界/职责/接口契约/验收标准）；rick-spec.md 含「功能等价」；
     spec.md 枚举可操作判据（dry-run / go test / 集成测试）。
   - O2（三层金字塔 + 做薄）：internal/{cmd,handler,env,builder,runtime} 5 目录存在且含
     Go 源文件（职责落地）；6 冗余包 internal/{executor,parser,actpath,logging,git,agent}
     已删除（test ! -d）。
   - O3（pibuilder pi 对齐）：模板中 workflowScript/runs.run 出现 >0；think/research/exporter
     经 env.DeployRickCustomizations 真实落盘为 pi agent（RICK_PI_AGENT_DIR 指向 temp，
     验证 3 个 agent 文件 + extensions/rick-gates/ 部署）；`.rick/skills/rick-gates/helper.py`
     源存在；自然语言 subagent 触发词计数相对基线下降（同一正则口径）。
2. 边界（命令全可用）：`./bin/rick --help` + 8 个子命令 `--help` 均 exit 0、无 panic，
   root help 含 plan/doing/easy/learning/dream/tools/human-loop/ctrl。
3. 异常（回滚兜底）：记录当前 HEAD 后，在副本 worktree 中 `./scripts/build.sh` + `./bin/rick
   --help`（exit 0），验证当前 release commit 可编译运行；结束后不污染主工作区
   （`.rick/domain/` git status 前后一致）。

脚本只读代码 + 跑 go run 临时程序 + 命令 smoke，产物落临时目录，幂等；
仅向 stdout 输出一行 JSON。
"""
import json
import os
import re
import shutil
import subprocess
import sys
import tempfile

# ---------------------------------------------------------------------------
# 路径（绝对路径）
# ---------------------------------------------------------------------------
SCRIPT = os.path.abspath(__file__)

DOMAIN_DIR = '.rick/domain'
SPEC_MD = os.path.join(DOMAIN_DIR, 'spec.md')
RICK_SPEC_MD = os.path.join(DOMAIN_DIR, 'rick-spec.md')
TEMPLATES_DIR = os.path.join('internal', 'prompt', 'templates')
RICK_GATES_HELPER = os.path.join('.rick', 'skills', 'rick-gates', 'helper.py')
BASELINE_FILE = os.path.join('.rick', 'jobs', 'job_35', 'doing', 'trigger-baseline.txt')

# 四要素关键词（模块边界 / 职责 / 接口契约 / 验收标准）
FOUR_ELEMENTS = ['模块边界', '职责', '接口契约', '验收标准']

# O2：三层金字塔 5 目录 + 6 冗余包
LAYER_DIRS = ['cmd', 'handler', 'env', 'builder', 'runtime']
REDUNDANT_PKGS = ['executor', 'parser', 'actpath', 'logging', 'git', 'agent']

# 8 个业务子命令（排除 cobra 自动的 completion/help）
SUBCOMMANDS = ['plan', 'doing', 'easy', 'learning', 'dream', 'tools', 'human-loop', 'ctrl']

# 自然语言 subagent 触发词正则（口径与 task11 基线落盘时一致）
TRIGGER_RE = re.compile(
    r'派发\s*subagent'
    r'|SPAWN\s+Sub\s+Agent'
    r'|SPAWN\s+Step'
    r'|子\s*Agent'
    r'|父\s*Agent'
    r'|Main\s+Agent'
    r'|Sub\s+Agent'
    r'|每个\s+subagent\s+独立启动',
    re.IGNORECASE,
)
WORKFLOW_RE = re.compile(r'workflowScript|runs\.run|runs\.all')

# O3 期望落盘的 3 个 pi agent
AGENTS = ['think', 'research', 'exporter']

# 临时 Go 程序：真实调用 env.DeployRickCustomizations()（隔离 RICK_PI_AGENT_DIR）。
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


# ---------------------------------------------------------------------------
# 工具函数
# ---------------------------------------------------------------------------
def find_repo_root(start_file):
    """从脚本目录向上逐级查找含 go.mod 的仓库根目录（绝对路径）。"""
    d = os.path.dirname(os.path.abspath(start_file))
    while True:
        if os.path.exists(os.path.join(d, 'go.mod')):
            return d
        parent = os.path.dirname(d)
        if parent == d:
            return None
        d = parent


def run(cmd, cwd, timeout=300, env=None):
    """运行子进程，返回 (returncode, stdout, stderr)。"""
    try:
        p = subprocess.run(cmd, cwd=cwd, capture_output=True, text=True, timeout=timeout, env=env)
        return p.returncode, p.stdout, p.stderr
    except FileNotFoundError as e:
        return 127, '', str(e)
    except subprocess.TimeoutExpired as e:
        out = e.stdout or ''
        err = (e.stderr or '') + '\n[timeout after %ds]' % timeout
        return 124, out, err


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


def md_files(templates_dir):
    """递归产出 templates 目录下所有 .md 文件路径。"""
    if not os.path.isdir(templates_dir):
        return
    for dirpath, _dirs, files in os.walk(templates_dir):
        for fn in sorted(files):
            if fn.endswith('.md'):
                yield os.path.join(dirpath, fn)


def total_matches(regex, templates_dir):
    """统计所有模板文件中正则的匹配总次数。"""
    total = 0
    for path in md_files(templates_dir):
        content = read_text(path)
        if content is not None:
            total += len(regex.findall(content))
    return total


def files_with_match(regex, templates_dir):
    """统计至少命中一次正则的模板文件数。"""
    count = 0
    for path in md_files(templates_dir):
        content = read_text(path)
        if content is not None and regex.search(content):
            count += 1
    return count


def read_baseline():
    """读取 trigger-baseline.txt 中的自然语言触发词计数（迁移前基线）。"""
    if not os.path.exists(BASELINE_FILE):
        return None
    content = read_text(BASELINE_FILE)
    if content is None:
        return None
    m = re.search(r'\d+', content)
    return int(m.group()) if m else None


def tail(text, n=1600):
    """截取文本尾部，用于错误信息。"""
    if not text:
        return ''
    return text[-n:]


def git_domain_status():
    """返回 `git status --porcelain .rick/domain/` 的输出（快照）。"""
    rc, out, err = run(['git', 'status', '--porcelain', DOMAIN_DIR], cwd=REPO_ROOT)
    return out if rc == 0 else None


# ---------------------------------------------------------------------------
# 测试步骤
# ---------------------------------------------------------------------------
def test_o1(errors):
    """O1：spec 信息内核（两个 spec 文件 + 四要素关键词 + 功能等价 + 可操作判据）。"""
    for path in (SPEC_MD, RICK_SPEC_MD):
        full = os.path.join(REPO_ROOT, path)
        if not os.path.isfile(full):
            errors.append('%s 不存在' % path)
            continue
        content = read_text(full)
        if content is None:
            errors.append('%s 无法读取' % path)
            continue
        for kw in FOUR_ELEMENTS:
            if kw not in content:
                errors.append('%s 缺失四要素关键词「%s」' % (path, kw))

    rick_spec = read_text(os.path.join(REPO_ROOT, RICK_SPEC_MD))
    if rick_spec is not None and '功能等价' not in rick_spec:
        errors.append('%s 缺失「功能等价」判据' % RICK_SPEC_MD)

    spec = read_text(os.path.join(REPO_ROOT, SPEC_MD))
    if spec is not None and not re.search(r'dry-run|go test|集成测试', spec):
        errors.append('%s 未枚举可操作判据（dry-run/go test/集成测试）' % SPEC_MD)


def test_o2(errors):
    """O2：三层金字塔 5 目录存在且含 Go 源文件 + 6 冗余包已删除。"""
    for d in LAYER_DIRS:
        full = os.path.join(REPO_ROOT, 'internal', d)
        if not os.path.isdir(full):
            errors.append('internal/%s 目录不存在' % d)
            continue
        gos = [f for f in os.listdir(full) if f.endswith('.go') and os.path.isfile(os.path.join(full, f))]
        if not gos:
            errors.append('internal/%s 目录存在但无 Go 源文件（职责未落地）' % d)

    for d in REDUNDANT_PKGS:
        full = os.path.join(REPO_ROOT, 'internal', d)
        if os.path.isdir(full):
            errors.append('冗余包 internal/%s 仍存在（应已删除）' % d)


def test_o3(errors, go, prog_dir):
    """O3：pibuilder pi 对齐（workflowScript 计数 + 3 agent 落盘 + rick-gates 脚本 + 触发词下降）。"""
    templates = os.path.join(REPO_ROOT, TEMPLATES_DIR)
    if not os.path.isdir(templates):
        errors.append('templates 目录不存在: %s' % templates)
        return

    # workflowScript / runs.run 出现 >0
    workflow_files = files_with_match(WORKFLOW_RE, templates)
    if workflow_files < 1:
        errors.append('模板中 workflowScript/runs.run 命中文件数 %d < 1' % workflow_files)

    # 自然语言触发词计数下降（同一正则口径对比基线）
    baseline = read_baseline()
    if baseline is None:
        errors.append('trigger-baseline.txt 不存在或非法；迁移前必须先落盘基线')
    else:
        current = total_matches(TRIGGER_RE, templates)
        if current >= baseline:
            errors.append('自然语言触发词计数未下降：当前 %d >= 基线 %d' % (current, baseline))

    # think/research/exporter 落盘为 pi agent（真实 deploy，隔离 RICK_PI_AGENT_DIR）。
    # 只覆盖 RICK_PI_AGENT_DIR，不覆盖 HOME（避免 go run 把工具链重新下载到临时
    # 目录形成只读 go/pkg/mod，导致清理失败）；AgentDir() 优先读 RICK_PI_AGENT_DIR。
    agent_dir = tempfile.mkdtemp(prefix='rick_task12_agent_')
    env = os.environ.copy()
    env['RICK_PI_AGENT_DIR'] = agent_dir
    rc, out, err = run([go, 'run', '.', REPO_ROOT], cwd=prog_dir, timeout=240, env=env)
    if rc != 0:
        errors.append('DeployRickCustomizations exit=%d: %s' % (rc, tail(err or out)))
    else:
        for name in AGENTS:
            p = os.path.join(agent_dir, 'agents', name + '.md')
            if not os.path.isfile(p):
                errors.append('pi agent %s.md 未落盘: %s' % (name, p))
        gate_deployed = os.path.join(agent_dir, 'extensions', 'rick-gates', 'helper.py')
        if not os.path.isfile(gate_deployed):
            errors.append('rick-gates 扩展未部署: %s' % gate_deployed)
    shutil.rmtree(agent_dir, ignore_errors=True)

    # rick-gates 门禁源脚本存在
    if not os.path.isfile(os.path.join(REPO_ROOT, RICK_GATES_HELPER)):
        errors.append('门禁源脚本不存在: %s' % RICK_GATES_HELPER)


def test_commands(errors):
    """边界：rick 全命令可用（root --help + 8 子命令 --help 均 exit 0、无 panic）。"""
    rick_bin = os.path.join(REPO_ROOT, 'bin', 'rick')
    if not os.path.isfile(rick_bin):
        errors.append('rick 二进制不存在: %s' % rick_bin)
        return

    # root --help
    rc, out, err = run([rick_bin, '--help'], cwd=REPO_ROOT, timeout=60)
    if rc != 0:
        errors.append('./bin/rick --help exit=%d: %s' % (rc, tail(err or out)))
    else:
        if 'panic' in (out + err):
            errors.append('./bin/rick --help 输出含 panic')
        for cmd in SUBCOMMANDS:
            if cmd not in out:
                errors.append('./bin/rick --help 缺少子命令「%s」' % cmd)

    # 8 个子命令 --help
    for cmd in SUBCOMMANDS:
        rc, out, err = run([rick_bin, cmd, '--help'], cwd=REPO_ROOT, timeout=60)
        if rc != 0:
            errors.append('./bin/rick %s --help exit=%d: %s' % (cmd, rc, tail(err or out)))
        elif 'panic' in (out + err):
            errors.append('./bin/rick %s --help 输出含 panic' % cmd)


def test_rollback(errors):
    """异常：回滚兜底 —— 副本 worktree 中 build + --help 可编译运行。"""
    rc, out, err = run(['git', 'rev-parse', 'HEAD'], cwd=REPO_ROOT, timeout=30)
    if rc != 0:
        errors.append('git rev-parse HEAD 失败: %s' % tail(err))
        return
    commit = out.strip()
    if not commit:
        errors.append('git rev-parse HEAD 返回空 commit')
        return

    wt_dir = tempfile.mkdtemp(prefix='rick_rollback_')
    shutil.rmtree(wt_dir, ignore_errors=True)  # git worktree add 需要目录不存在
    try:
        rc, out, err = run(['git', 'worktree', 'add', '--detach', wt_dir, commit],
                           cwd=REPO_ROOT, timeout=120)
        if rc != 0:
            errors.append('git worktree add 失败: %s' % tail(err or out))
            return

        # 副本 worktree 中构建
        rc, out, err = run(['./scripts/build.sh'], cwd=wt_dir, timeout=900)
        if rc != 0:
            errors.append('回滚副本 build 失败: %s' % tail(err or out))
            return

        # 副本 worktree 中运行 --help
        wt_rick = os.path.join(wt_dir, 'bin', 'rick')
        if not os.path.isfile(wt_rick):
            errors.append('回滚副本未产出 bin/rick: %s' % wt_rick)
            return
        rc, out, err = run([wt_rick, '--help'], cwd=wt_dir, timeout=60)
        if rc != 0:
            errors.append('回滚副本 ./bin/rick --help exit=%d: %s' % (rc, tail(err or out)))
    finally:
        run(['git', 'worktree', 'remove', '--force', wt_dir], cwd=REPO_ROOT, timeout=60)
        shutil.rmtree(wt_dir, ignore_errors=True)


# ---------------------------------------------------------------------------
# 主流程
# ---------------------------------------------------------------------------
REPO_ROOT = find_repo_root(SCRIPT)


def main():
    errors = []

    if REPO_ROOT is None:
        result = {'pass': False, 'errors': ['无法定位项目根目录（未找到 go.mod）']}
        print(json.dumps(result))
        sys.exit(1)

    # 快照 .rick/domain/ 状态（用于断言测试本身未引入变更）
    domain_before = git_domain_status()

    go = shutil.which('go')
    if not go:
        errors.append('go toolchain 未在 PATH 中找到')

    # 临时 Go 程序目录（位于模块内方可 import internal/env）
    prog_dir = None
    if go:
        prog_dir = tempfile.mkdtemp(prefix='.tmp_task12_deploy_', dir=REPO_ROOT)
        write_file(os.path.join(prog_dir, 'main.go'), _DEPLOY_GO)

    # ===== Test 1: 正常路径（三 O 验收）=====
    test_o1(errors)
    test_o2(errors)
    if go and prog_dir:
        test_o3(errors, go, prog_dir)

    # ===== Test 2: 边界（命令全可用）=====
    test_commands(errors)

    # ===== Test 3: 异常（回滚兜底）=====
    test_rollback(errors)

    # 清理临时 Go 程序目录
    if prog_dir:
        shutil.rmtree(prog_dir, ignore_errors=True)

    # 断言 .rick/domain/ 未被本测试污染
    domain_after = git_domain_status()
    if domain_before is not None and domain_after is not None and domain_before != domain_after:
        errors.append('.rick/domain/ 状态被测试改变：\nbefore=%r\nafter=%r' % (domain_before, domain_after))

    result = {'pass': len(errors) == 0, 'errors': errors}

    # CRITICAL: 仅此一行输出到 stdout
    print(json.dumps(result))

    sys.exit(0 if result['pass'] else 1)


if __name__ == '__main__':
    main()
