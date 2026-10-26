#!/usr/bin/env python3
"""gate26.py — 迭代10 L4（task52 测试/e2e+成本对比 + task53 文档）"""
import json, os, subprocess, sys
def root():
    p = os.path.abspath(__file__)
    for _ in range(6): p = os.path.dirname(p)
    return p
ROOT = root(); ERRORS = []
def run(cmd, cwd=ROOT, timeout=2700):
    return subprocess.run(cmd, cwd=cwd, capture_output=True, text=True, timeout=timeout)
def read(p):
    with open(os.path.join(ROOT, p), encoding="utf-8") as f: return f.read()
def main():
    e = read("super/tests/e2e_test.py")
    if "read_experiment" not in e: ERRORS.append("[A] e2e 未用 read_experiment 还原断言")
    lp = read(".rick/loops/super-loop.md")
    for kw in ["experiment_id", "read_experiment"]:
        if kw not in lp: ERRORS.append("[B] super-loop 缺 " + kw)
    dp = read(".rick/skills/super_debug_protocol/skill.md")
    if "一次性" not in dp: ERRORS.append("[C] debug skill 缺一次性根因")
    if "成本" not in read(".rick/jobs/job_47/doing/rsi/dev-iterations.md")[-4000:]:
        ERRORS.append("[D] rsi 未记录成本对比")
    r = run(["bash", "super/tests/run_all.sh"])
    if r.returncode != 0: ERRORS.append("[E] run_all 失败：\n" + (r.stdout + r.stderr)[-2500:])
    print(json.dumps({"pass": not ERRORS, "errors": ERRORS}, ensure_ascii=False, indent=2))
    sys.exit(0 if not ERRORS else 1)
if __name__ == "__main__": main()
