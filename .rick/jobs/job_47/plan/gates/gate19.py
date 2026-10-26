#!/usr/bin/env python3
"""gate19.py — 迭代9 L1（task44 五要素+线程安全）"""
import json, os, subprocess, sys
def root():
    p = os.path.abspath(__file__)
    for _ in range(6): p = os.path.dirname(p)
    return p
ROOT = root(); ERRORS = []
def run(cmd, cwd=ROOT, timeout=1200):
    return subprocess.run(cmd, cwd=cwd, capture_output=True, text=True, timeout=timeout)
def read(p):
    with open(os.path.join(ROOT, p), encoding="utf-8") as f: return f.read()
def main():
    r = run([sys.executable, "-m", "unittest", "super.tests.test_archive", "super.tests.test_loop", "-v"])
    if r.returncode != 0: ERRORS.append("[A] 单测失败：\n" + (r.stdout + r.stderr)[-1500:])
    a = read("super/superopt/archive.py")
    for kw in ["elite_code_path", "elite_record_path", "RLock"]:
        if kw not in a: ERRORS.append("[B] archive.py 缺 " + kw)
    if "record_path" not in read("super/superopt/loop.py"): ERRORS.append("[C] loop.py 未传 record_path")
    wt = run(["git", "worktree", "list", "--porcelain"])
    prod = next((x.split(" ",1)[1].strip() for x in wt.stdout.splitlines() if x.startswith("worktree ") and "rick-dev" not in x), None)
    if prod and os.path.isdir(prod):
        r = run(["git", "status", "--porcelain", "-uno"], cwd=prod)
        ch = [x for x in r.stdout.splitlines() if ".rick/dream/" not in x]
        if ch: ERRORS.append("[D][RSI] 生产被改动：" + chr(10).join(ch)[:300])
    print(json.dumps({"pass": not ERRORS, "errors": ERRORS}, ensure_ascii=False, indent=2))
    sys.exit(0 if not ERRORS else 1)
if __name__ == "__main__": main()
