#!/usr/bin/env python3
"""gate13.py — 迭代7 L1（task32 env 模板 research.py + task35 gen_env/super-loop 检索三问）

A. research.py 四角色输出合法 JSON（三元组+suggested_queries 非空）
B. gen_env skill：检索三问 / research.py / 五件
C. super-loop：检索定位
D. 生产零触碰
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
    rp = os.path.join(ROOT, "super/env_templates/matmul/skills/research.py")
    if not os.path.isfile(rp):
        ERRORS.append("[A] 缺 skills/research.py")
    else:
        for role in ("adjust", "builder", "coding", "debug"):
            r = run([sys.executable, rp, "--role", role])
            try:
                o = json.loads(r.stdout.strip().splitlines()[-1])
                for k in ("method", "sources", "focus", "suggested_queries"):
                    if not o.get(k):
                        ERRORS.append("[A] %s 缺 %s" % (role, k))
            except Exception:
                ERRORS.append("[A] %s 输出非法: %r" % (role, (r.stdout + r.stderr)[-200:]))
    g = read(".rick/skills/gen_env_skill/skill.md")
    for kw in ["检索三问", "research.py", "五件"]:
        if kw not in g: ERRORS.append("[B] gen_env skill 缺 " + kw)
    lp = read(".rick/loops/super-loop.md")
    if "检索" not in lp: ERRORS.append("[C] super-loop 缺检索")
    wt = run(["git", "worktree", "list", "--porcelain"])
    prod = next((x.split(" ",1)[1].strip() for x in wt.stdout.splitlines() if x.startswith("worktree ") and "rick-dev" not in x), None)
    if prod and os.path.isdir(prod):
        r = run(["git", "status", "--porcelain", "-uno"], cwd=prod)
        changed = [x for x in r.stdout.splitlines() if ".rick/dream/" not in x]
        if changed: ERRORS.append("[D][RSI] 生产被改动：" + chr(10).join(changed)[:300])
    print(json.dumps({"pass": not ERRORS, "errors": ERRORS}, ensure_ascii=False, indent=2))
    sys.exit(0 if not ERRORS else 1)

if __name__ == "__main__": main()
