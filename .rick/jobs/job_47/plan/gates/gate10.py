#!/usr/bin/env python3
"""gate10.py — 迭代5 L3（task24 四模板 + task25 文档）

A. 模板契约：无绝对路径硬编码；四模板各含其新 ctx 键
B. 文档契约：gen_env skill 含可移植性/占位符/terminate_recommend/ledger；super-loop 含饱和
C. run_all 全量（单测+e2e+双回归）
D. 生产零触碰
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
    need = {"builder": ["saturation", "domain_brief", "perf_timeline", "budget_status", "ledger_brief", "gate_defs", "{{env_dir}}"],
            "coding": ["gate_defs", "perf_baseline", "workdir_files", "ledger_brief"],
            "debug": ["gate_failure", "gate_defs", "ledger_brief"],
            "adjust": ["saturation", "perf_timeline", "budget_status", "domain_brief", "ledger_brief", "own_decisions", "terminate_recommend"]}
    for f, kws in need.items():
        s = read("super/prompts/%s.md" % f)
        if "/Users/" in s:
            ERRORS.append("[A] %s.md 含硬编码绝对路径" % f)
        for kw in kws:
            if kw not in s: ERRORS.append("[A] %s.md 缺 %s" % (f, kw))
    g = read(".rick/skills/gen_env_skill/skill.md")
    for kw in ["可移植性", "占位符", "terminate_recommend", "ledger"]:
        if kw not in g: ERRORS.append("[B] gen_env skill 缺 " + kw)
    lp = read(".rick/loops/super-loop.md")
    for kw in ["饱和", "框架门禁"]:
        if kw not in lp: ERRORS.append("[B] super-loop 缺 " + kw)
    r = run(["bash", "super/tests/run_all.sh"])
    if r.returncode != 0:
        ERRORS.append("[C] run_all 失败：\n" + (r.stdout + r.stderr)[-2000:])
    wt = run(["git", "worktree", "list", "--porcelain"])
    prod = next((x.split(" ",1)[1].strip() for x in wt.stdout.splitlines() if x.startswith("worktree ") and "rick-dev" not in x), None)
    if prod and os.path.isdir(prod):
        r = run(["git", "status", "--porcelain", "-uno"], cwd=prod)
        changed = [x for x in r.stdout.splitlines() if ".rick/dream/" not in x]
        if changed: ERRORS.append("[D][RSI] 生产被改动：" + chr(10).join(changed)[:300])
    print(json.dumps({"pass": not ERRORS, "errors": ERRORS}, ensure_ascii=False, indent=2))
    sys.exit(0 if not ERRORS else 1)

if __name__ == "__main__": main()
