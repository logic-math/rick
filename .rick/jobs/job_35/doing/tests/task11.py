#!/usr/bin/env python3
r"""task11 测试脚本：验证「自然语言 subagent 触发词 → pi 显式触发语法」迁移的验收条件。

用法：
  python3 task11.py                  # 运行完整验收测试，stdout 输出恰好一行 JSON
  python3 task11.py --dump-baseline  # （迁移前）以同一正则捕获自然语言触发词基线并落盘

验收正则（基线捕获与验收共用同一正则，依据 RFC §2.1 / research-report-N1.md F1
所列自然语言 subagent 触发术语）：
  派发 subagent / SPAWN Sub Agent / SPAWN Step / 子 Agent / 父 Agent /
  Main Agent / Sub Agent / 每个 subagent 独立启动

等价 grep 对照（迁移前先落盘基线，迁移后据此断言计数下降）：
  grep -roiE '派发\s*subagent|SPAWN\s+Sub\s+Agent|SPAWN\s+Step|子\s*Agent|父\s*Agent|Main\s+Agent|Sub\s+Agent|每个\s+subagent\s+独立启动' internal/prompt/templates/ | wc -l
"""
import json
import os
import re
import subprocess
import sys

# ---------------------------------------------------------------------------
# 路径（绝对路径）
# ---------------------------------------------------------------------------
SCRIPT = os.path.abspath(__file__)


def find_project_root(start):
    """从脚本目录向上逐级查找含 go.mod 的仓库根目录。"""
    d = os.path.dirname(os.path.abspath(start))
    while True:
        if os.path.exists(os.path.join(d, 'go.mod')):
            return d
        parent = os.path.dirname(d)
        if parent == d:
            return None
        d = parent


PROJECT_ROOT = find_project_root(SCRIPT)
TEMPLATES_DIR = os.path.join(PROJECT_ROOT, 'internal', 'prompt', 'templates') if PROJECT_ROOT else None
BASELINE_FILE = os.path.join(PROJECT_ROOT, '.rick', 'jobs', 'job_35', 'doing', 'trigger-baseline.txt') if PROJECT_ROOT else None
SENSE_LOOP = os.path.join(TEMPLATES_DIR, 'sense_loop.md') if TEMPLATES_DIR else None
RICK_BIN = os.path.join(PROJECT_ROOT, 'bin', 'rick') if PROJECT_ROOT else None

# ---------------------------------------------------------------------------
# 验收正则
# ---------------------------------------------------------------------------
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
PI_SYNTAX_RE = re.compile(r'workflowScript|runs\.run|runs\.all')
AGENT_NAME_RE = re.compile(r"agent:'worker'|agent:'reviewer'|agent:'think'|agent:'research'|agent:'exporter'")
SENSE_RE = re.compile(r'批判门禁|反向回流|judgment\.md')


def md_files():
    """递归产出 templates 目录下所有 .md 文件路径。"""
    for dirpath, _dirs, files in os.walk(TEMPLATES_DIR):
        for fn in sorted(files):
            if fn.endswith('.md'):
                yield os.path.join(dirpath, fn)


def total_matches(regex):
    """统计所有模板文件中正则的匹配总次数（non-overlapping）。"""
    total = 0
    for path in md_files():
        try:
            with open(path, encoding='utf-8') as f:
                total += len(regex.findall(f.read()))
        except Exception:
            continue
    return total


def files_with_match(regex):
    """统计至少命中一次正则的模板文件数（等价 grep -c | grep -v ':0' | wc -l）。"""
    count = 0
    for path in md_files():
        try:
            with open(path, encoding='utf-8') as f:
                if regex.search(f.read()):
                    count += 1
        except Exception:
            continue
    return count


def read_baseline():
    """读取基线文件中的自然语言触发词计数（迁移前由 --dump-baseline 落盘）。"""
    if not os.path.exists(BASELINE_FILE):
        return None
    try:
        with open(BASELINE_FILE, encoding='utf-8') as f:
            text = f.read().strip()
        m = re.search(r'\d+', text)
        return int(m.group()) if m else None
    except Exception:
        return None


def run(cmd, timeout=300):
    """在项目根目录执行命令，返回 (returncode, stdout, stderr)。"""
    try:
        p = subprocess.run(cmd, cwd=PROJECT_ROOT, capture_output=True, text=True, timeout=timeout)
        return p.returncode, p.stdout, p.stderr
    except Exception as e:
        return -1, '', str(e)


def main():
    # 迁移前捕获基线（与验收共用同一正则，保证前后口径一致）
    if '--dump-baseline' in sys.argv[1:]:
        if PROJECT_ROOT is None:
            print(json.dumps({'pass': False, 'errors': ['无法定位项目根目录（未找到 go.mod）']}))
            sys.exit(1)
        count = total_matches(TRIGGER_RE)
        os.makedirs(os.path.dirname(BASELINE_FILE), exist_ok=True)
        with open(BASELINE_FILE, 'w', encoding='utf-8') as f:
            f.write(str(count) + '\n')
        print(count)
        sys.exit(0)

    errors = []

    # 前置检查：项目根目录与 templates 目录必须存在
    if PROJECT_ROOT is None or not os.path.isdir(TEMPLATES_DIR):
        errors.append('无法定位项目根目录或 templates 目录不存在')
        print(json.dumps({'pass': False, 'errors': errors}))
        sys.exit(1)

    # ------------------------------------------------------------------
    # Test 1 正常路径：迁移落地 + 自然语言触发词计数下降
    # ------------------------------------------------------------------
    baseline = read_baseline()
    if baseline is None:
        errors.append(
            'trigger-baseline.txt 不存在或格式非法；迁移前必须先执行 '
            '`python3 task11.py --dump-baseline` 捕获自然语言触发词基线'
        )

    pi_files = files_with_match(PI_SYNTAX_RE)
    if pi_files < 1:
        errors.append(
            'pi 触发语法(workflowScript/runs.run/runs.all)命中文件数 %d < 1，迁移未落地' % pi_files
        )

    current = total_matches(TRIGGER_RE)
    if baseline is not None and current >= baseline:
        errors.append(
            '自然语言触发词计数未下降：当前 %d >= 基线 %d' % (current, baseline)
        )

    # ------------------------------------------------------------------
    # Test 2 边界：真实内置/自定义 agent 名被显式引用
    # ------------------------------------------------------------------
    agent_files = files_with_match(AGENT_NAME_RE)
    if agent_files < 1:
        errors.append(
            "真实 agent 名(agent:'worker'/'reviewer'/'think'/'research'/'exporter')命中文件数 %d < 1" % agent_files
        )

    # ------------------------------------------------------------------
    # Test 3 异常：SENSE 特有语义不丢 + 无变量泄漏 + go 全绿
    # ------------------------------------------------------------------
    sense_hits = 0
    if os.path.exists(SENSE_LOOP):
        with open(SENSE_LOOP, encoding='utf-8') as f:
            sense_hits = len(SENSE_RE.findall(f.read()))
        if sense_hits < 1:
            errors.append('sense_loop.md 批判门禁/反向回流/judgment.md 语义命中 %d < 1' % sense_hits)
    else:
        errors.append('sense_loop.md 不存在: ' + SENSE_LOOP)

    # go build
    rc, out, err = run(['go', 'build', './...'])
    if rc != 0:
        errors.append('go build ./... 失败: ' + (err or out)[:500])

    # plan --dry-run 无 {{ 泄漏（PromptBuilder 未替换变量会泄漏 "{{"）
    if os.path.exists(RICK_BIN):
        rc, out, err = run([RICK_BIN, 'plan', '--dry-run'], timeout=120)
        if rc != 0:
            errors.append('rick plan --dry-run 失败: ' + (err or out)[:500])
        else:
            brace = out.count('{{')
            if brace != 0:
                errors.append('rick plan --dry-run 输出含未替换变量 "{{" 共 ' + str(brace) + ' 处')
    else:
        errors.append('rick 二进制不存在: ' + RICK_BIN)

    # go test ./internal/prompt/... -v 全绿
    rc, out, err = run(['go', 'test', './internal/prompt/...', '-v'], timeout=600)
    if rc != 0:
        errors.append('go test ./internal/prompt/... -v 未全绿: ' + (err or out)[-800:])

    result = {'pass': len(errors) == 0, 'errors': errors}
    print(json.dumps(result))
    sys.exit(0 if result['pass'] else 1)


if __name__ == '__main__':
    main()
