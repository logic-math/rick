#!/usr/bin/env python3
"""gate20.py — 迭代9 L2（task46 成本 skill/工具）"""
import json, os, subprocess, sys, tempfile
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
    p = ".rick/skills/super_cost_analysis/skill.md"
    if not os.path.isfile(os.path.join(ROOT, p)): ERRORS.append("[A] 缺 " + p)
    else:
        s = read(p)
        for kw in ["触发场景", "预期效果", "核心内容", "session", "浪费"]:
            if kw not in s: ERRORS.append("[A] cost skill 缺 " + kw)
    tp = os.path.join(ROOT, "super/tools/cost_report.py")
    if not os.path.isfile(tp): ERRORS.append("[B] 缺 cost_report.py")
    else:
        with tempfile.TemporaryDirectory() as td:
            evs = [{"ts":"2026-10-26T10:00:00+08:00","seq":1,"type":"agent_ended","source":"scheduler",
                    "payload":{"role":"coding","iteration_id":1,"usage":{"total_tokens":1000,"cost_usd":0.1},"wall_ms":1000}},
                   {"ts":"2026-10-26T10:01:00+08:00","seq":2,"type":"iteration_ended","source":"scheduler",
                    "payload":{"iteration_id":1,"tokens":1000,"by_role":{"coding":{"tokens":1000}}}}]
            with open(os.path.join(td,"events.jsonl"),"w") as f:
                for e in evs: f.write(json.dumps(e)+"\n")
            r = run([sys.executable, tp, "--workdir", td])
            if r.returncode != 0 or "|" not in r.stdout:
                ERRORS.append("[B] cost_report 冒烟失败：" + (r.stdout+r.stderr)[-300:])
    if "by_role" not in read("super/superopt/loop.py"): ERRORS.append("[C] loop.py iteration_ended 缺 by_role")
    r = run([sys.executable, "-m", "unittest", "super.tests.test_loop"])
    if r.returncode != 0: ERRORS.append("[D] test_loop 失败：" + (r.stdout+r.stderr)[-1200:])
    print(json.dumps({"pass": not ERRORS, "errors": ERRORS}, ensure_ascii=False, indent=2))
    sys.exit(0 if not ERRORS else 1)
if __name__ == "__main__": main()
