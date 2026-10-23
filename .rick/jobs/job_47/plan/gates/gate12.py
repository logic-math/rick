#!/usr/bin/env python3
"""gate12.py — 迭代6 L2（task28 loop 三要素+profile历史+designs / task29 四模板 / task31 env 模板）

A. py 全量单测全绿
B. loop 契约：三要素命令注入（b/c/d）/ profile_history 采集与注入 / designs 落盘
C. 模板契约：三要素命令 + builder（profile 历史/arXiv/落盘）/ coding（gate_check）/ debug
D. env 模板：skills 四件齐备 + 无 self_check + gate_check 可跑（venv）
E. 生产零触碰
"""
import json, os, subprocess, sys, tempfile

def root():
    p = os.path.abspath(__file__)
    for _ in range(6): p = os.path.dirname(p)
    return p

ROOT = root(); ERRORS = []
def run(cmd, cwd=ROOT, timeout=1800, env=None):
    return subprocess.run(cmd, cwd=cwd, capture_output=True, text=True, timeout=timeout, env=env)

def read(p):
    with open(os.path.join(ROOT, p), encoding="utf-8") as f: return f.read()

def main():
    r = run([sys.executable, "-m", "unittest", "discover", "-s", "super/tests", "-p", "test_*.py"])
    if r.returncode != 0:
        ERRORS.append("[A] py 全量失败：\n" + (r.stdout + r.stderr)[-2000:])
    l = read("super/superopt/loop.py")
    for kw in ["profile_cmd", "benchmark_cmd", "verify_cmd", "gate_check_cmd",
               "profile_history", "designs", "papers", "inspirations"]:
        if kw not in l: ERRORS.append("[B] loop.py 缺 " + kw)
    need = {"builder": ["{{profile_cmd}}", "{{benchmark_cmd}}", "{{verify_cmd}}", "{{gate_check_cmd}}",
                        "target_profile_history", "{{designs_dir}}", "arxiv"],
            "coding": ["{{gate_check_cmd}}", "{{verify_cmd}}", "{{benchmark_cmd}}"],
            "debug": ["{{gate_check_cmd}}", "{{profile_cmd}}"]}
    for f, kws in need.items():
        s = read("super/prompts/%s.md" % f)
        for kw in kws:
            if kw not in s: ERRORS.append("[C] %s.md 缺 %s" % (f, kw))
        if "/Users/" in s: ERRORS.append("[C] %s.md 含硬编码路径" % f)
    tpl = "super/env_templates/matmul"
    for f in ["profile.py", "benchmark.py", "verify.py", "gate_check.py"]:
        if not os.path.isfile(os.path.join(ROOT, tpl, "skills", f)):
            ERRORS.append("[D] 缺 skills/" + f)
    if os.path.isfile(os.path.join(ROOT, tpl, "skills", "self_check.py")):
        ERRORS.append("[D] self_check.py 未删除")
    with tempfile.TemporaryDirectory() as td:
        v = os.path.join(td, "v")
        if run([sys.executable, "-m", "venv", v]).returncode == 0:
            vpy = os.path.join(v, "bin", "python")
            run([vpy, "-m", "pip", "install", "-q", "numpy"], timeout=600)
            r = run([vpy, os.path.join(ROOT, tpl, "skills", "gate_check.py"),
                     "--candidate", os.path.join(ROOT, tpl, "code", "reference.py")], timeout=300)
            try:
                out = json.loads(r.stdout.strip().splitlines()[-1])
                if out.get("passed") is not True:
                    ERRORS.append("[D] gate_check 对 reference 未通过：" + str(out)[:200])
            except Exception:
                ERRORS.append("[D] gate_check 输出非法：" + (r.stdout + r.stderr)[-300:])
    wt = run(["git", "worktree", "list", "--porcelain"])
    prod = next((x.split(" ",1)[1].strip() for x in wt.stdout.splitlines() if x.startswith("worktree ") and "rick-dev" not in x), None)
    if prod and os.path.isdir(prod):
        r = run(["git", "status", "--porcelain", "-uno"], cwd=prod)
        changed = [x for x in r.stdout.splitlines() if ".rick/dream/" not in x]
        if changed: ERRORS.append("[E][RSI] 生产被改动：" + chr(10).join(changed)[:300])
    print(json.dumps({"pass": not ERRORS, "errors": ERRORS}, ensure_ascii=False, indent=2))
    sys.exit(0 if not ERRORS else 1)

if __name__ == "__main__": main()
