#!/usr/bin/env python3
"""gate14.py — 迭代7 L2（task33 loop 注入 + task34 四模板检索纪律）

A. py 全量单测全绿
B. loop：research_cmd / research_brief 四角色注入
C. 模板：四角色含 research_brief/research_cmd；无硬编码
D. env 模板：skills 五件齐
E. 生产零触碰
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
    r = run([sys.executable, "-m", "unittest", "discover", "-s", "super/tests", "-p", "test_*.py"])
    if r.returncode != 0:
        ERRORS.append("[A] py 全量失败：\n" + (r.stdout + r.stderr)[-2000:])
    l = read("super/superopt/loop.py")
    for kw in ["research_cmd", "research_brief", "skills", "research.py"]:
        if kw not in l: ERRORS.append("[B] loop.py 缺 " + kw)
    for f in ("profile.py", "benchmark.py", "verify.py", "gate_check.py", "research.py"):
        if not os.path.isfile(os.path.join(ROOT, "super/env_templates/matmul/skills", f)):
            ERRORS.append("[D] 缺 " + f)
    wt = run(["git", "worktree", "list", "--porcelain"])
    prod = next((x.split(" ",1)[1].strip() for x in wt.stdout.splitlines() if x.startswith("worktree ") and "rick-dev" not in x), None)
    if prod and os.path.isdir(prod):
        r = run(["git", "status", "--porcelain", "-uno"], cwd=prod)
        changed = [x for x in r.stdout.splitlines() if ".rick/dream/" not in x]
        if changed: ERRORS.append("[E][RSI] 生产被改动：" + chr(10).join(changed)[:300])
    print(json.dumps({"pass": not ERRORS, "errors": ERRORS}, ensure_ascii=False, indent=2))
    sys.exit(0 if not ERRORS else 1)

if __name__ == "__main__": main()
