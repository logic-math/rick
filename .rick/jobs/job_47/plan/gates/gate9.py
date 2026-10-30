#!/usr/bin/env python3
"""gate9.py — 迭代5 L2（task23 loop ctx 全面增强 + gate→debug + ledger 接线）

A. py 全量单测全绿（含 e2e 适配）
B. loop 契约：新 ctx 方法/键/recent_events 真值/gate→debug/ledger/saturation/terminate/max_turns
C. 生产零触碰
"""
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
    r = run([sys.executable, "-m", "unittest", "discover", "-s", "super/tests", "-p", "test_*.py"])
    if r.returncode != 0:
        ERRORS.append("[A] py 全量失败：\n" + (r.stdout + r.stderr)[-2000:])
    l = read("super/superopt/loop.py")
    for kw in ["_perf_timeline", "_budget_status", "_gate_defs", "_domain_brief",
               "_workdir_files", "_recent_interventions", "ledger_brief", "own_decisions",
               "gate_failure", "saturation", "_saturation_signal", "terminate_recommend",
               "max_turns", "self.ledger", "failure_stage"]:
        if kw not in l: ERRORS.append("[B] loop.py 缺 " + kw)
    # recent_events 不再只给 keys（排除注释/文档串：只查非注释代码行）
    # 只看「代码里构造字典键」的形态："payload_keys": ——注释/文档串中是反引号包裹
    code_lines = [x for x in l.splitlines()
                  if not x.strip().startswith("#")]
    if any('"payload_keys":' in x or "'payload_keys':" in x for x in code_lines):
        ERRORS.append("[B] loop.py 代码中仍用 payload_keys 摘要（应给真值）")
    # gate→debug：_verify_and_admit 内应有 gate_failure + attempt=100 结构
    if "_run_debug" in l and "gate_failure" in l and "attempt=100" in l:
        pass
    else:
        ERRORS.append("[B] 未见 gate 失败→debug 通道（_run_debug + gate_failure + attempt=100）")
    wt = run(["git", "worktree", "list", "--porcelain"])
    prod = next((x.split(" ",1)[1].strip() for x in wt.stdout.splitlines() if x.startswith("worktree ") and "rick-dev" not in x), None)
    if prod and os.path.isdir(prod):
        r = run(["git", "status", "--porcelain", "-uno"], cwd=prod)
        changed = [x for x in r.stdout.splitlines() if ".rick/dream/" not in x]
        if changed: ERRORS.append("[C][RSI] 生产被改动：" + chr(10).join(changed)[:300])
    print(json.dumps({"pass": not ERRORS, "errors": ERRORS}, ensure_ascii=False, indent=2))
    sys.exit(0 if not ERRORS else 1)

if __name__ == "__main__": main()
