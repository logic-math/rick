#!/usr/bin/env python3
"""gate18.py — 迭代8 L3（task41 super-loop + task42 测试/e2e 适配）

A. super-loop：四 agent loop / skill 加载规范 / 实验记录 / 决策记录 / TDD
B. run_all 全量（单测+e2e+双回归）
C. 生产零触碰
"""
import json, os, subprocess, sys

def root():
    p = os.path.abspath(__file__)
    for _ in range(6): p = os.path.dirname(p)
    return p

ROOT = root(); ERRORS = []
def run(cmd, cwd=ROOT, timeout=1800):
    return subprocess.run(cmd, cwd=cwd, capture_output=True, text=True, timeout=timeout)

def read(p):
    with open(os.path.join(ROOT, p), encoding="utf-8") as f: return f.read()

def main():
    lp = read(".rick/loops/super-loop.md")
    for kw in ["skill 加载规范", "实验记录", "决策记录", "TDD"]:
        if kw not in lp: ERRORS.append("[A] super-loop 缺 " + kw)
    r = run(["bash", "super/tests/run_all.sh"])
    if r.returncode != 0:
        ERRORS.append("[B] run_all 失败：\n" + (r.stdout + r.stderr)[-2500:])
    wt = run(["git", "worktree", "list", "--porcelain"])
    prod = next((x.split(" ",1)[1].strip() for x in wt.stdout.splitlines() if x.startswith("worktree ") and "rick-dev" not in x), None)
    if prod and os.path.isdir(prod):
        r = run(["git", "status", "--porcelain", "-uno"], cwd=prod)
        changed = [x for x in r.stdout.splitlines() if ".rick/dream/" not in x]
        if changed: ERRORS.append("[C][RSI] 生产被改动：" + chr(10).join(changed)[:300])
    print(json.dumps({"pass": not ERRORS, "errors": ERRORS}, ensure_ascii=False, indent=2))
    sys.exit(0 if not ERRORS else 1)

if __name__ == "__main__": main()
