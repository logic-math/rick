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
    if "五要素" not in read(".rick/skills/gen_env_skill/skill.md"):
        ERRORS.append("[C] gen_env 缺五要素")
    if "cost_report" not in read(".rick/loops/super-loop.md"):
        ERRORS.append("[C] super-loop 缺成本复盘")
    r = run(["bash", "super/tests/run_all.sh"])
    if r.returncode != 0: ERRORS.append("[D] run_all 失败：\n" + (r.stdout + r.stderr)[-2500:])
    print(json.dumps({"pass": not ERRORS, "errors": ERRORS}, ensure_ascii=False, indent=2))
    sys.exit(0 if not ERRORS else 1)
if __name__ == "__main__": main()
