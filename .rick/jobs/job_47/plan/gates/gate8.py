#!/usr/bin/env python3
"""gate8.py — 迭代5 L1（task22 agents 回合预算 + task26 ledger）

A. test_agents + test_ledger 全绿
B. agents：max_turns/truncated；config：四角色回合上限
C. ledger：模块存在 + API（append/read_all/brief/perf_timeline/own_decisions）
D. 生产零触碰（忽略 .rick/dream）
"""
import json, os, subprocess, sys

def root():
    p = os.path.abspath(__file__)
    for _ in range(6): p = os.path.dirname(p)
    return p

ROOT = root(); ERRORS = []
def run(cmd, cwd=ROOT, timeout=900):
    return subprocess.run(cmd, cwd=cwd, capture_output=True, text=True, timeout=timeout)

def read(p):
    with open(os.path.join(ROOT, p), encoding="utf-8") as f: return f.read()

def main():
    r = run([sys.executable, "-m", "unittest", "super.tests.test_agents", "super.tests.test_ledger", "-v"])
    if r.returncode != 0:
        ERRORS.append("[A] 单测失败：\n" + (r.stdout + r.stderr)[-1500:])
    a = read("super/superopt/agents.py")
    for kw in ["max_turns", "truncated"]:
        if kw not in a: ERRORS.append("[B] agents.py 缺 " + kw)
    c = read("super/superopt/config.py")
    for kw in ["builder_max_turns", "coding_max_turns", "adjust_max_turns", "debug_max_turns"]:
        if kw not in c: ERRORS.append("[B] config.py 缺 " + kw)
    lp = os.path.join(ROOT, "super/superopt/ledger.py")
    if not os.path.exists(lp):
        ERRORS.append("[C] 缺 ledger.py")
    else:
        s = open(lp, encoding="utf-8").read()
        for kw in ["class Ledger", "def append", "def read_all", "def brief", "def perf_timeline", "def own_decisions"]:
            if kw not in s: ERRORS.append("[C] ledger.py 缺 " + kw)
    wt = run(["git", "worktree", "list", "--porcelain"])
    prod = next((x.split(" ",1)[1].strip() for x in wt.stdout.splitlines() if x.startswith("worktree ") and "rick-dev" not in x), None)
    if prod and os.path.isdir(prod):
        r = run(["git", "status", "--porcelain", "-uno"], cwd=prod)
        changed = [x for x in r.stdout.splitlines() if ".rick/dream/" not in x]
        if changed: ERRORS.append("[D][RSI] 生产被改动：" + chr(10).join(changed)[:300])
    print(json.dumps({"pass": not ERRORS, "errors": ERRORS}, ensure_ascii=False, indent=2))
    sys.exit(0 if not ERRORS else 1)

if __name__ == "__main__": main()
