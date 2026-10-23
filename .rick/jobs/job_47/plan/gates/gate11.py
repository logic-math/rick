#!/usr/bin/env python3
"""gate11.py — 迭代6 L1（task27 gates 三要素 + task30 gen_env/super-loop 三要素追问）

A. test_gates 全绿（新布局+legacy 双布局）
B. gates.py 契约：gate_check 探测 + stage1 四件检查 + legacy 回退
C. gen_env skill：三要素追问（human 亲自）/ benchmark.py / verify.py / profile.py / gate_check / profile_history / designs
D. super-loop：三要素追问 / 不得停在 / designs
E. 生产零触碰
"""
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
    r = run([sys.executable, "-m", "unittest", "super.tests.test_gates", "-v"])
    if r.returncode != 0:
        ERRORS.append("[A] test_gates 失败：\n" + (r.stdout + r.stderr)[-1500:])
    g = read("super/superopt/gates.py")
    for kw in ["gate_check", "skills", "data/test_correctness.py"]:
        if kw not in g: ERRORS.append("[B] gates.py 缺 " + kw)
    s = read(".rick/skills/gen_env_skill/skill.md")
    for kw in ["三要素", "benchmark.py", "verify.py", "profile.py", "gate_check",
               "human 亲自", "profile_history", "designs"]:
        if kw not in s: ERRORS.append("[C] gen_env skill 缺 " + kw)
    lp = read(".rick/loops/super-loop.md")
    for kw in ["三要素追问", "不得停在", "designs"]:
        if kw not in lp: ERRORS.append("[D] super-loop 缺 " + kw)
    wt = run(["git", "worktree", "list", "--porcelain"])
    prod = next((x.split(" ",1)[1].strip() for x in wt.stdout.splitlines() if x.startswith("worktree ") and "rick-dev" not in x), None)
    if prod and os.path.isdir(prod):
        r = run(["git", "status", "--porcelain", "-uno"], cwd=prod)
        changed = [x for x in r.stdout.splitlines() if ".rick/dream/" not in x]
        if changed: ERRORS.append("[E][RSI] 生产被改动：" + chr(10).join(changed)[:300])
    print(json.dumps({"pass": not ERRORS, "errors": ERRORS}, ensure_ascii=False, indent=2))
    sys.exit(0 if not ERRORS else 1)

if __name__ == "__main__": main()
