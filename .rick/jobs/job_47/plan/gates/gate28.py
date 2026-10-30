#!/usr/bin/env python3
"""gate28.py — 迭代12（task56: JSON 修复 resume + 失败恢复 + 软监控 + API 断连）"""
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
    r = run([sys.executable, "-m", "unittest", "super.tests.test_agents", "super.tests.test_loop", "-v"])
    if r.returncode != 0: ERRORS.append("[A] 单测失败：\n" + (r.stdout + r.stderr)[-2000:])
    a = read("super/superopt/agents.py")
    for kw in ["session_id", "json_repair", "resume", "turn_warn"]:
        if kw not in a: ERRORS.append("[B] agents.py 缺 " + kw)
    l = read("super/superopt/loop.py")
    for kw in ["api_outage", "turn_budget_alert", "json_repair", "new_cell_rejected", "max_cells"]:
        if kw not in l: ERRORS.append("[C] loop.py 缺 " + kw)
    c = read("super/superopt/config.py")
    for kw in ["turn_soft_warn", "api_outage_threshold", "max_cells"]:
        if kw not in c: ERRORS.append("[D] config 缺 " + kw)
    r = run([sys.executable, "-m", "unittest", "discover", "-s", "super/tests", "-p", "test_*.py"])
    if r.returncode != 0: ERRORS.append("[E] 全量失败：\n" + (r.stdout + r.stderr)[-2000:])
    print(json.dumps({"pass": not ERRORS, "errors": ERRORS}, ensure_ascii=False, indent=2))
    sys.exit(0 if not ERRORS else 1)
if __name__ == "__main__": main()
