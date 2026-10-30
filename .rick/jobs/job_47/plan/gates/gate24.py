#!/usr/bin/env python3
"""gate24.py — 迭代10 L2（task51 成本优化）"""
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
    c = read("super/superopt/config.py")
    if '"debug_max_turns": 0' not in c and "'debug_max_turns': 0" not in c:
        ERRORS.append("[A] config debug_max_turns 未设为 0（不设回合上限）")
    if "event_fsync" not in c: ERRORS.append("[A] config 缺 event_fsync")
    d = read("super/prompts/debug.md")
    for kw in ["一次性", "三假设", "禁止"]:
        if kw not in d: ERRORS.append("[B] debug.md 缺 " + kw)
    b = read("super/prompts/builder.md")
    if "摘要" not in b: ERRORS.append("[C] builder.md 相邻格未改摘要形态")
    r = run([sys.executable, "-m", "unittest", "super.tests.test_loop", "super.tests.test_prompts"])
    if r.returncode != 0: ERRORS.append("[D] 单测失败：\n" + (r.stdout + r.stderr)[-1500:])
    print(json.dumps({"pass": not ERRORS, "errors": ERRORS}, ensure_ascii=False, indent=2))
    sys.exit(0 if not ERRORS else 1)
if __name__ == "__main__": main()
