#!/usr/bin/env python3
"""gate3.py — job_47 第 3 层门禁（Kahn L3：task8 loop 四状态循环 + task9 adjudicator/intervention/__main__）

模块级集成测试（human 确认，agent 不得修改）：
  A. 全部 py 单测全绿（含 test_loop / test_adjudicator / test_intervention）
  B. 调度器循环契约（静态 + 单测断言已在 task8 测试内，此处做关键红线复检）：
     - loop.py 含 increment_visit 唯一递增、事件先于档案、--preverify-red 分支
     - adjudicator.py 独立进程入口（--proposal/--workdir）+ 决策文件协议 + 不写 events
     - __main__.py 存在且 --help exit 0
  C. mock 迭代 e2e 复跑：python3 -m unittest super.tests.test_loop -v（全 mock ≥3 迭代链路
     + 崩溃恢复游标一致 + visit 真实递增回归断言）
  D. 生产零触碰

输出 JSON {"pass": bool, "errors": [...]}，exit 0=绿 / 1=红。
"""
import json
import os
import subprocess
import sys


def get_project_root():
    p = os.path.abspath(__file__)
    for _ in range(6):
        p = os.path.dirname(p)
    return p


ROOT = get_project_root()
ERRORS = []


def run(cmd, cwd=ROOT, timeout=900, env=None):
    return subprocess.run(cmd, cwd=cwd, capture_output=True, text=True, timeout=timeout,
                          env=env)


def read(path):
    with open(os.path.join(ROOT, path), encoding="utf-8") as as_f:
        return as_f.read()


def exists(path):
    return os.path.exists(os.path.join(ROOT, path))


def main():
    # ---- A. 全部单测 ----
    r = run([sys.executable, "-m", "unittest", "discover", "-s", "super/tests",
             "-p", "test_*.py", "-v"])
    if r.returncode != 0:
        ERRORS.append(f"[A] py 全量单测失败：\n{(r.stdout + r.stderr)[-2500:]}")

    # ---- B. 循环/裁决/入口契约 ----
    if not exists("super/superopt/loop.py"):
        ERRORS.append("[B] 缺 loop.py")
    else:
        s = read("super/superopt/loop.py")
        for kw in ["increment_visit", "_recover", "_run_adjust", "_verify_and_admit",
                   "preverify-red", "iteration_started", "cell_sampled"]:
            if kw not in s:
                ERRORS.append(f"[B] loop.py 缺关键词 {kw}")

    if not exists("super/superopt/adjudicator.py"):
        ERRORS.append("[B] 缺 adjudicator.py")
    else:
        s = read("super/superopt/adjudicator.py")
        for kw in ["--proposal", "--workdir", "approve", "reject", "decision"]:
            if kw not in s:
                ERRORS.append(f"[B] adjudicator.py 缺关键词 {kw}")
        # [B][单写者] 精确检查：adjudicator 可读 events.jsonl（裁决证据），但不得写入。
        # 违规信号：import EventWriter / 以写模式打开 events.jsonl / append 调用。
        import re as _re
        if _re.search(r"from\s+\S*events\s+import\s+.*EventWriter|import\s+.*EventWriter", s):
            ERRORS.append("[B][单写者] adjudicator import 了 EventWriter（写入能力）")
        if _re.search(r'open\([^)]*events\.jsonl[^)]*["\'](w|a)', s):
            ERRORS.append("[B][单写者] adjudicator 以写模式打开 events.jsonl")

    if not exists("super/superopt/intervention.py"):
        ERRORS.append("[B] 缺 intervention.py")
    else:
        s = read("super/superopt/intervention.py")
        for kw in ["drain", "push", "interventions"]:
            if kw not in s:
                ERRORS.append(f"[B] intervention.py 缺关键词 {kw}")

    # ---- C. __main__ 冒烟（包在 super/ 下，PYTHONPATH=super 与 Go supervisor spawn 契约一致） ----
    import os as _os
    env = dict(_os.environ, PYTHONPATH="super")
    r = run([sys.executable, "-m", "superopt", "--help"], env=env)
    if r.returncode != 0:
        ERRORS.append(f"[C] python3 -m superopt --help 失败：\n{(r.stdout + r.stderr)[-500:]}")
    else:
        for flag in ["--workdir", "--max-iterations", "--preverify-red"]:
            if flag not in r.stdout:
                ERRORS.append(f"[C] -m superopt --help 缺 flag {flag}")

    # ---- D. 生产零触碰 ----
    wt = run(["git", "worktree", "list", "--porcelain"])
    prod = None
    for line in wt.stdout.splitlines():
        if line.startswith("worktree ") and "rick-dev" not in line:
            prod = line.split(" ", 1)[1].strip()
            break
    if prod and os.path.isdir(prod):
        r = run(["git", "status", "--porcelain", "-uno"], cwd=prod)
        if r.stdout.strip():
            ERRORS.append(f"[D][RSI] 生产仓库 tracked 文件被改动：\n{r.stdout.strip()[:500]}")
    else:
        ERRORS.append("[D][RSI] 未能定位生产仓库")

    print(json.dumps({"pass": len(ERRORS) == 0, "errors": ERRORS}, ensure_ascii=False, indent=2))
    sys.exit(0 if not ERRORS else 1)


if __name__ == "__main__":
    main()
