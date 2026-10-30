#!/usr/bin/env python3
"""gate25.py — 迭代10 L3（task50 并发槽池）"""
import json, os, subprocess, sys
def root():
    p = os.path.abspath(__file__)
    for _ in range(6): p = os.path.dirname(p)
    return p
ROOT = root(); ERRORS = []
def run(cmd, cwd=ROOT, timeout=2400):
    return subprocess.run(cmd, cwd=cwd, capture_output=True, text=True, timeout=timeout)
def read(p):
    with open(os.path.join(ROOT, p), encoding="utf-8") as f: return f.read()
def main():
    l = read("super/superopt/loop.py")
    for kw in ["max_parallel_experiments", "experiment_started", "experiment_finished", "slot"]:
        if kw not in l: ERRORS.append("[A] loop.py 缺 " + kw)
    c = read("super/superopt/config.py")
    if '"max_parallel_experiments": 2' not in c: ERRORS.append("[B] config 并发默认未设 2")
    r = run([sys.executable, "-m", "unittest", "super.tests.test_loop", "-v"])
    if r.returncode != 0: ERRORS.append("[C] test_loop 失败：\n" + (r.stdout + r.stderr)[-2000:])
    print(json.dumps({"pass": not ERRORS, "errors": ERRORS}, ensure_ascii=False, indent=2))
    sys.exit(0 if not ERRORS else 1)
if __name__ == "__main__": main()
