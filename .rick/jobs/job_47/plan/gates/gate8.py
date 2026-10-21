#!/usr/bin/env python3
"""gate8.py — job_47 迭代4 门禁（task22-25：输入完整性/饱和信号/认输/预算/模板可移植）

A. py 全量单测全绿
B. agents 契约：run(max_turns) + truncated；config 四角色回合上限默认值
C. loop 契约：builder ctx 含 env_dir/perf_profile_cmd/saturation；_saturation_signal；terminate_recommend；max_turns 传参
D. 模板契约：super/prompts/*.md 无绝对路径硬编码（/Users/）；builder/adjust 含 saturation 与 terminate_recommend
E. 文档契约：gen_env skill 含可移植性/占位符/terminate_recommend；super-loop Step7 含饱和
F. run_all 全量（单测+e2e+双回归）
G. 生产零触碰（忽略 .rick/dream 用户资产）
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
        ERRORS.append(f"[A] py 全量失败：\n{(r.stdout + r.stderr)[-2000:]}")

    a = read("super/superopt/agents.py")
    for kw in ["max_turns", "truncated"]:
        if kw not in a: ERRORS.append(f"[B] agents.py 缺 {kw}")
    c = read("super/superopt/config.py")
    for kw in ["builder_max_turns", "coding_max_turns", "adjust_max_turns"]:
        if kw not in c: ERRORS.append(f"[B] config.py 缺 {kw}")

    l = read("super/superopt/loop.py")
    for kw in ["env_dir", "perf_profile_cmd", "saturation", "_saturation_signal", "terminate_recommend", "max_turns"]:
        if kw not in l: ERRORS.append(f"[C] loop.py 缺 {kw}")

    for f in ["builder", "adjust", "debug"]:
        s = read(f"super/prompts/{f}.md")
        if "/Users/" in s:
            ERRORS.append(f"[D] super/prompts/{f}.md 含硬编码绝对路径")
    for f, kws in [("builder", ["saturation"]), ("adjust", ["saturation", "terminate_recommend"])]:
        s = read(f"super/prompts/{f}.md")
        for kw in kws:
            if kw not in s: ERRORS.append(f"[D] {f}.md 缺 {kw}")

    g = read(".rick/skills/gen_env_skill/skill.md")
    for kw in ["可移植性", "占位符", "terminate_recommend"]:
        if kw not in g: ERRORS.append(f"[E] gen_env skill 缺 {kw}")
    lp = read(".rick/loops/super-loop.md")
    if "饱和" not in lp: ERRORS.append("[E] super-loop 缺饱和指引")

    r = run(["bash", "super/tests/run_all.sh"])
    if r.returncode != 0:
        ERRORS.append(f"[F] run_all 失败：\n{(r.stdout + r.stderr)[-2000:]}")

    wt = run(["git", "worktree", "list", "--porcelain"])
    prod = next((x.split(" ",1)[1].strip() for x in wt.stdout.splitlines() if x.startswith("worktree ") and "rick-dev" not in x), None)
    if prod and os.path.isdir(prod):
        r = run(["git", "status", "--porcelain", "-uno"], cwd=prod)
        changed = [x for x in r.stdout.splitlines() if ".rick/dream/" not in x]
        if changed: ERRORS.append("[G][RSI] 生产被改动：" + chr(10).join(changed)[:300])

    print(json.dumps({"pass": not ERRORS, "errors": ERRORS}, ensure_ascii=False, indent=2))
    sys.exit(0 if not ERRORS else 1)

if __name__ == "__main__": main()
