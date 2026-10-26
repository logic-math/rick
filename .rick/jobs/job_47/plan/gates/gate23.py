#!/usr/bin/env python3
"""gate23.py — 迭代10 L1（task49 事件并行化）"""
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
    r = run([sys.executable, "-m", "unittest", "super.tests.test_events", "super.tests.test_lineage", "super.tests.test_loop", "-v"])
    if r.returncode != 0: ERRORS.append("[A] 单测失败：\n" + (r.stdout + r.stderr)[-2000:])
    e = read("super/superopt/events.py")
    for kw in ["experiment_id", "experiment_seq", "read_experiment"]:
        if kw not in e: ERRORS.append("[B] events.py 缺 " + kw)
    l = read("super/superopt/lineage.py")
    if "interrupted_experiments" not in l: ERRORS.append("[C] lineage 缺 interrupted_experiments")
    lo = read("super/superopt/loop.py")
    if "experiment_id" not in lo: ERRORS.append("[D] loop.py 事件未带 experiment_id")
    r = run([sys.executable, "-m", "unittest", "discover", "-s", "super/tests", "-p", "test_*.py"])
    if r.returncode != 0: ERRORS.append("[E] 全量失败：\n" + (r.stdout + r.stderr)[-2000:])
    print(json.dumps({"pass": not ERRORS, "errors": ERRORS}, ensure_ascii=False, indent=2))
    sys.exit(0 if not ERRORS else 1)
if __name__ == "__main__": main()
