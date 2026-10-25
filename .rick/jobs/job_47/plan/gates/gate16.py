#!/usr/bin/env python3
"""gate16.py — 迭代8 L1（task36 仓级 super_* skills + task37 gen_env 修复）

A. 五件 super_* skill 存在且三段式（触发场景/预期效果/核心内容）+ 关键语义
B. gen_env：流程重排（0.5 领域研究/前置输入）/职责划分/键一致性/preverify 自动化/优化幅度追问 debug_policy
C. 生产零触碰
"""
import json, os, subprocess, sys

def root():
    p = os.path.abspath(__file__)
    for _ in range(6): p = os.path.dirname(p)
    return p

ROOT = root(); ERRORS = []
def run(cmd, cwd=ROOT, timeout=600):
    return subprocess.run(cmd, cwd=cwd, capture_output=True, text=True, timeout=timeout)

def read(p):
    with open(os.path.join(ROOT, p), encoding="utf-8") as f: return f.read()

def main():
    skills = {
        "super_experiment_analysis": ["触发场景", "预期效果", "核心内容", "experiment_record"],
        "super_sense": ["触发场景", "预期效果", "核心内容", "主要矛盾"],
        "super_cell_management": ["触发场景", "预期效果", "核心内容", "正交"],
        "super_experiment_design": ["触发场景", "预期效果", "核心内容", "实验假设"],
        "super_debug_protocol": ["触发场景", "预期效果", "核心内容", "三个最可能"],
    }
    for name, kws in skills.items():
        p = ".rick/skills/%s/skill.md" % name
        if not os.path.isfile(os.path.join(ROOT, p)):
            ERRORS.append("[A] 缺 " + p); continue
        s = read(p)
        for kw in kws:
            if kw not in s: ERRORS.append("[A] %s 缺 %s" % (name, kw))
    g = read(".rick/skills/gen_env_skill/skill.md")
    for kw in ["0.5", "优化幅度", "debug_policy", "键一致性", "前置输入"]:
        if kw not in g: ERRORS.append("[B] gen_env 缺 " + kw)
    wt = run(["git", "worktree", "list", "--porcelain"])
    prod = next((x.split(" ",1)[1].strip() for x in wt.stdout.splitlines() if x.startswith("worktree ") and "rick-dev" not in x), None)
    if prod and os.path.isdir(prod):
        r = run(["git", "status", "--porcelain", "-uno"], cwd=prod)
        changed = [x for x in r.stdout.splitlines() if ".rick/dream/" not in x]
        if changed: ERRORS.append("[C][RSI] 生产被改动：" + chr(10).join(changed)[:300])
    print(json.dumps({"pass": not ERRORS, "errors": ERRORS}, ensure_ascii=False, indent=2))
    sys.exit(0 if not ERRORS else 1)

if __name__ == "__main__": main()
