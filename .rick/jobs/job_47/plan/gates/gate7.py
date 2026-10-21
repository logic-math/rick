#!/usr/bin/env python3
"""gate7.py — job_47 迭代3 L3 门禁（task20 工具+e2e / task21 自改进文档）

A. run_all.sh 全量（含 e2e 适配后全绿 + 双回归）
B. adjust_trace.py 存在 + CLI 契约（--workdir）+ 冒烟（fixture 事件 → markdown 输出）
C. gen_env skill 含轨迹→prompt 进化流程（PROMPTS_LOG/_history/四角色）；super-loop 含自改进 SOP
D. 生产零触碰
"""
import json, os, subprocess, sys, tempfile

def root():
    p = os.path.abspath(__file__)
    for _ in range(6): p = os.path.dirname(p)
    return p

ROOT = root(); ERRORS = []
def run(cmd, cwd=ROOT, timeout=1800):
    return subprocess.run(cmd, cwd=cwd, capture_output=True, text=True, timeout=timeout)

def main():
    r = run(["bash", "super/tests/run_all.sh"])
    if r.returncode != 0:
        ERRORS.append(f"[A] run_all.sh 失败：\n{(r.stdout + r.stderr)[-2500:]}")

    tp = os.path.join(ROOT, "super/tools/adjust_trace.py")
    if not os.path.exists(tp):
        ERRORS.append("[B] 缺 super/tools/adjust_trace.py")
    else:
        s = open(tp, encoding="utf-8").read()
        if "--workdir" not in s: ERRORS.append("[B] adjust_trace.py 缺 --workdir")
        # 冒烟：手工造最小事件流
        with tempfile.TemporaryDirectory() as td:
            os.makedirs(td, exist_ok=True)
            evs = [
                {"ts":"2026-10-20T10:00:00+08:00","seq":1,"type":"adjust_decision","source":"scheduler","payload":{"iteration":2,"proposals":[{"type":"soft_weight","target_cell":"a=x","new_weight":0.5}]}},
                {"ts":"2026-10-20T10:01:00+08:00","seq":2,"type":"cell_sampled","source":"scheduler","payload":{"iteration_id":3,"cell_id":"a=x","visit":2,"cold_start":False,"weight_share":0.33}},
                {"ts":"2026-10-20T10:02:00+08:00","seq":3,"type":"gate_result","source":"gate","payload":{"passed":True,"cell_id":"a=x","metrics":{"perf":{"value":100.0}}}},
            ]
            with open(os.path.join(td, "events.jsonl"), "w") as f:
                for e in evs: f.write(json.dumps(e, ensure_ascii=False) + "\n")
            r2 = run([sys.executable, tp, "--workdir", td])
            if r2.returncode != 0 or "|" not in r2.stdout:
                ERRORS.append(f"[B] adjust_trace 冒烟失败：{(r2.stdout + r2.stderr)[-400:]}")

    g = open(os.path.join(ROOT, ".rick/skills/gen_env_skill/skill.md"), encoding="utf-8").read()
    for kw in ["轨迹", "PROMPTS_LOG", "_history"]:
        if kw not in g: ERRORS.append(f"[C] gen_env skill 缺 {kw}")
    l = open(os.path.join(ROOT, ".rick/loops/super-loop.md"), encoding="utf-8").read()
    if "自改进" not in l: ERRORS.append("[C] super-loop 缺自改进 SOP")

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
