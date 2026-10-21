#!/usr/bin/env python3
"""gate6.py — job_47 迭代3 L2 门禁（task19 loop 块）

A. py 全量单测全绿
B. loop.py 契约：expand_cell 应用链/热重载/快照 iter_N/adjust_decision 事件/weight_share 字段
C. adjudicator：expand_cell 裁决规则
D. 生产零触碰
"""
import json, os, subprocess, sys

def root():
    p = os.path.abspath(__file__)
    for _ in range(6): p = os.path.dirname(p)
    return p

ROOT = root(); ERRORS = []
def run(cmd, cwd=ROOT, timeout=1200):
    return subprocess.run(cmd, cwd=cwd, capture_output=True, text=True, timeout=timeout)

def main():
    r = run([sys.executable, "-m", "unittest", "discover", "-s", "super/tests", "-p", "test_*.py"])
    if r.returncode != 0:
        ERRORS.append(f"[A] py 全量失败：\n{(r.stdout + r.stderr)[-2000:]}")

    s = open(os.path.join(ROOT, "super/superopt/loop.py"), encoding="utf-8").read()
    for kw in ["expand_cell", "adjust_decision", "weight_share", "iter_{", "feature_space"]:
        if kw not in s: ERRORS.append(f"[B] loop.py 缺 {kw}")

    a = open(os.path.join(ROOT, "super/superopt/adjudicator.py"), encoding="utf-8").read()
    if "expand_cell" not in a: ERRORS.append("[C] adjudicator.py 缺 expand_cell 裁决")

    wt = run(["git", "worktree", "list", "--porcelain"])
    prod = next((l.split(" ",1)[1].strip() for l in wt.stdout.splitlines() if l.startswith("worktree ") and "rick-dev" not in l), None)
    if prod and os.path.isdir(prod):
        r = run(["git", "status", "--porcelain", "-uno"], cwd=prod)
        if r.stdout.strip(): ERRORS.append(f"[D][RSI] 生产被改动：{r.stdout.strip()[:300]}")

    print(json.dumps({"pass": not ERRORS, "errors": ERRORS}, ensure_ascii=False, indent=2))
    sys.exit(0 if not ERRORS else 1)

if __name__ == "__main__": main()
