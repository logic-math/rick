#!/usr/bin/env python3
"""gate15.py — 迭代7 L3（task34 四模板检索纪律）

A. 四模板含 research_brief/research_cmd + 无硬编码
B. 模板检索纪律语义（builder 关注点对齐/ coding 不查论文/ adjust 查创新点）
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
    for f in ("builder", "coding", "debug", "adjust"):
        s = read("super/prompts/%s.md" % f)
        for kw in ("research_brief", "research_cmd"):
            if kw not in s: ERRORS.append("[A] %s.md 缺 %s" % (f, kw))
        if "/Users/" in s: ERRORS.append("[A] %s.md 含硬编码" % f)
    b = read("super/prompts/builder.md")
    if "关注点" not in b: ERRORS.append("[B] builder 缺关注点对齐纪律")
    c = read("super/prompts/coding.md")
    if "文档" not in c: ERRORS.append("[B] coding 缺文档检索纪律")
    a = read("super/prompts/adjust.md")
    if "创新" not in a and "趋势" not in a: ERRORS.append("[B] adjust 缺创新点/趋势检索纪律")
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
