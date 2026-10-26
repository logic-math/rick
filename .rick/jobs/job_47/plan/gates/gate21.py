#!/usr/bin/env python3
"""gate21.py — 迭代9 L3（task45 并发/格锁 + task47 文档 + task48 e2e）"""
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
    for kw in ["max_parallel_experiments", "cell_lock", "acquire", "release"]:
        if kw not in l: ERRORS.append("[A] loop.py 缺 " + kw)
    c = read("super/superopt/config.py")
    for kw in ["max_parallel_experiments", "cell_lock_ttl_s"]:
        if kw not in c: ERRORS.append("[B] config 缺 " + kw)
    r = run([sys.executable, "-m", "unittest", "discover", "-s", "super/tests", "-p", "test_*.py"])
    if r.returncode != 0: ERRORS.append("[C] py 全量失败：\n" + (r.stdout + r.stderr)[-2500:])
    print(json.dumps({"pass": not ERRORS, "errors": ERRORS}, ensure_ascii=False, indent=2))
    sys.exit(0 if not ERRORS else 1)
if __name__ == "__main__": main()
