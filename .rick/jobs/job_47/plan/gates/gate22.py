#!/usr/bin/env python3
"""gate22.py — 迭代9 L4（task47 文档 + task48 e2e 适配）"""
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
    g = read(".rick/skills/gen_env_skill/skill.md")
    for kw in ["五要素", "code_path", "并发", "cost_report"]:
        if kw not in g: ERRORS.append("[A] gen_env 缺 " + kw)
    l = read(".rick/loops/super-loop.md")
    for kw in ["成本复盘", "cost_report"]:
        if kw not in l: ERRORS.append("[B] super-loop 缺 " + kw)
    e = read("super/tests/e2e_test.py")
    for kw in ["elite_code_path", "by_role"]:
        if kw not in e: ERRORS.append("[C] e2e 缺 " + kw)
    r = run(["bash", "super/tests/run_all.sh"])
    if r.returncode != 0: ERRORS.append("[D] run_all 失败：\n" + (r.stdout + r.stderr)[-2500:])
    print(json.dumps({"pass": not ERRORS, "errors": ERRORS}, ensure_ascii=False, indent=2))
    sys.exit(0 if not ERRORS else 1)
if __name__ == "__main__": main()
