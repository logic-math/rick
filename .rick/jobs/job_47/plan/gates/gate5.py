#!/usr/bin/env python3
"""gate5.py — job_47 迭代3 L1 门禁（task15 sampler / task16 budget / task17 格手术 / task18 adjust.md）

A. test_sampler + test_budget 全绿（UCB 数学清除断言）
B. matmul 模板：6 格（2轴×3值）+ venv 三验
C. adjust.md 契约：采样份额语义 + expand_cell + 纪律
D. 生产零触碰
"""
import json, os, subprocess, sys

def root():
    p = os.path.abspath(__file__)
    for _ in range(6): p = os.path.dirname(p)
    return p

ROOT = root(); ERRORS = []
def run(cmd, cwd=ROOT, timeout=900, env=None):
    return subprocess.run(cmd, cwd=cwd, capture_output=True, text=True, timeout=timeout, env=env)

def main():
    r = run([sys.executable, "-m", "unittest", "super.tests.test_sampler", "super.tests.test_budget", "-v"])
    if r.returncode != 0:
        ERRORS.append(f"[A] sampler/budget 单测失败：\n{(r.stdout + r.stderr)[-1500:]}")
    s = open(os.path.join(ROOT, "super/superopt/sampler.py"), encoding="utf-8").read()
    if "math.log" in s or "sqrt" in s:
        ERRORS.append("[A] sampler.py 仍有 UCB 数学残留（sqrt/log）")
    for kw in ["choices", "weight"]:
        if kw not in s: ERRORS.append(f"[A] sampler.py 缺加权随机关键词 {kw}")

    fs = json.load(open(os.path.join(ROOT, "super/env_templates/matmul/feature_space.json")))
    axes = fs.get("axes", [])
    n = 1
    for a in axes: n *= len(a.get("values", []))
    if not (4 <= n <= 8):
        ERRORS.append(f"[B] matmul 格数 = {n}（期望 4-8 精选格）")
    names = [a.get("name") for a in axes]
    if len(names) != len(set(names)):
        ERRORS.append("[B] 特征轴重名")

    import tempfile
    with tempfile.TemporaryDirectory() as td:
        vpy = os.path.join(td, "v", "bin", "python")
        if run([sys.executable, "-m", "venv", os.path.join(td, "v")]).returncode != 0:
            ERRORS.append("[B] venv 失败")
        elif run([vpy, "-m", "pip", "install", "-q", "numpy"], timeout=600).returncode != 0:
            ERRORS.append("[B] numpy 安装失败")
        else:
            tpl = "super/env_templates/matmul"
            r = run([vpy, f"{tpl}/data/test_correctness.py", "--candidate", f"{tpl}/code/reference.py"], timeout=120)
            try:
                out = json.loads(r.stdout.strip().splitlines()[-1])
                if out.get("passed") is not True: ERRORS.append("[B] 参考实现未过门")
            except Exception: ERRORS.append("[B] correctness 输出异常")

    adj = open(os.path.join(ROOT, "super/prompts/adjust.md"), encoding="utf-8").read()
    for kw in ["采样份额", "expand_cell"]:
        if kw not in adj: ERRORS.append(f"[C] adjust.md 缺 {kw}")
    if "ucb" in adj.lower().replace("ucb_score", ""):
        ERRORS.append("[C] adjust.md 有 ucb 残留表述")

    wt = run(["git", "worktree", "list", "--porcelain"])
    prod = next((l.split(" ",1)[1].strip() for l in wt.stdout.splitlines() if l.startswith("worktree ") and "rick-dev" not in l), None)
    if prod and os.path.isdir(prod):
        r = run(["git", "status", "--porcelain", "-uno"], cwd=prod)
        # .rick/dream 等知识资产是用户侧自有改动（非本 job 产物），不计违规
        changed = [l for l in r.stdout.splitlines() if ".rick/dream/" not in l]
        if changed: ERRORS.append("[D][RSI] 生产被改动：" + chr(10).join(changed)[:300])

    print(json.dumps({"pass": not ERRORS, "errors": ERRORS}, ensure_ascii=False, indent=2))
    sys.exit(0 if not ERRORS else 1)

if __name__ == "__main__": main()
