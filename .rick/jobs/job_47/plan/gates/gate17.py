#!/usr/bin/env python3
"""gate17.py — 迭代8 L2（task38 模型/模块 + task39 loop 接线 + task40 四模板）

A. py 全量单测全绿（含新 test_experiment_record）
B. archive description / experiment_record API / neighbor sampling
C. loop：adjust 四步 ctx、决策记录、new_cell/modify_cell、builder 相邻格、coding TDD ctx、debug 三类触发、实验记录注入
D. 四模板：新 ctx 键 + skill 声明（super_* 路径）+ 无硬编码
E. 生产零触碰
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
    if not os.path.isfile(os.path.join(ROOT, "super/superopt/experiment_record.py")):
        ERRORS.append("[B] 缺 experiment_record.py")
    else:
        s = read("super/superopt/experiment_record.py")
        for kw in ["def create", "def append_section", "def read", "def digest", "def recent"]:
            if kw not in s: ERRORS.append("[B] experiment_record 缺 " + kw)
    a = read("super/superopt/archive.py")
    if "set_description" not in a: ERRORS.append("[B] archive 缺 set_description")
    fs = read("super/superopt/feature_space.py")
    for kw in ["neighbor_pool", "sample_neighbors"]:
        if kw not in fs: ERRORS.append("[B] feature_space 缺 " + kw)
    l = read("super/superopt/loop.py")
    for kw in ["experiment_records", "decision_history", "cell_descriptions", "decisions",
               "new_cell", "modify_cell", "neighbor_samples", "target_record",
               "research_report", "coding_plan", "tdd", "perf_gain_pct",
               "tolerance_band_recheck", "suspicious_gain", "experiment_record"]:
        if kw not in l: ERRORS.append("[C] loop.py 缺 " + kw)
    need = {"adjust": ["experiment_records", "decision_history", "super_experiment_analysis", "super_sense", "super_cell_management", "new_cell", "modify_cell"],
            "builder": ["neighbor_samples", "target_record", "research_report", "super_experiment_design"],
            "coding": ["research_report", "coding_plan", "test_solution", "TDD"],
            "debug": ["perf_gain_pct", "super_debug_protocol", "三个最可能"]}
    for f, kws in need.items():
        s = read("super/prompts/%s.md" % f)
        for kw in kws:
            if kw not in s: ERRORS.append("[D] %s.md 缺 %s" % (f, kw))
        if "/Users/" in s: ERRORS.append("[D] %s.md 含硬编码" % f)
    wt = run(["git", "worktree", "list", "--porcelain"])
    prod = next((x.split(" ",1)[1].strip() for x in wt.stdout.splitlines() if x.startswith("worktree ") and "rick-dev" not in x), None)
    if prod and os.path.isdir(prod):
        r = run(["git", "status", "--porcelain", "-uno"], cwd=prod)
        changed = [x for x in r.stdout.splitlines() if ".rick/dream/" not in x]
        if changed: ERRORS.append("[E][RSI] 生产被改动：" + chr(10).join(changed)[:300])
    print(json.dumps({"pass": not ERRORS, "errors": ERRORS}, ensure_ascii=False, indent=2))
    sys.exit(0 if not ERRORS else 1)

if __name__ == "__main__": main()
